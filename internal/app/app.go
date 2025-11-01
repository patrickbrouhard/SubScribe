package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/patrickprogramme/subscribe/internal/clipboard"
	"github.com/patrickprogramme/subscribe/internal/config"
	"github.com/patrickprogramme/subscribe/internal/fsutil"
	"github.com/patrickprogramme/subscribe/internal/obsidian"
	"github.com/patrickprogramme/subscribe/internal/ui"
	"github.com/patrickprogramme/subscribe/internal/yt"
	"github.com/patrickprogramme/subscribe/pkg/model"
)

const (
	defaultUpdateTimeout  = 15 * time.Second
	defaultExtractTimeout = 2 * time.Minute
	dirPerm               = 0o755
	filePerm              = 0o644
)

// CLIFlags contient les information venant des flags de l'app
type CLIFlags struct {
	ConfigPath string
	URL        string
	Auto       bool
	YtDlpPath  string
}

// App orchestre les différentes dépendances (UI, YtDlp, FS...)
type App struct {
	cfg      *config.Config
	ui       ui.Interface
	flags    *CLIFlags
	ytClient yt.Interface // **présent** : client yt-dlp initialisé dans Run
	renderer *obsidian.Renderer
}

// New construit l'application en initialisant les dépendances par défaut.
// Pour les tests, on préférera construire App en injectant des implémentations mock.
func New(cfg *config.Config, uiClient ui.Interface, flags *CLIFlags, renderer *obsidian.Renderer) *App {
	return &App{
		cfg:      cfg,
		ui:       uiClient,
		flags:    flags,
		renderer: renderer,
	}
}

// Run exécute le flux principal. Il initialise ytClient (via InitYtDlp) en utilisant le ctx.
// Ainsi l'initialisation respecte annulation/signaux.
func (a *App) Run(ctx context.Context) error {
	// Récupération de l'URL : priorité flag > clipboard > prompt
	url := a.flags.URL
	if url == "" {
		// ui.GetYtURL effectue clipboard + prompt si nécessaire
		u, err := a.ui.GetYtURL(ctx)
		if err != nil {
			return fmt.Errorf("get url: %w", err)
		}
		url = u
	}

	// si l'utilisateur a passé --yt-dlp-path, l'appliquer et re-resoudre
	if a.flags.YtDlpPath != "" {
		// on peut assigner dans cfg le champ Path (valeur brute), puis
		a.cfg.YtDlp.Path = a.flags.YtDlpPath
		// et appeler la méthode qui résout Name/ResolvedPath
		// a.cfg.EnsureYtDlpResolved()
	}

	// Init yt-dlp (CheckBinary + version)
	dl, version, err := yt.InitYtDlp(ctx, a.cfg)
	if err != nil {
		return fmt.Errorf("yt init: %w", err)
	}
	a.ytClient = dl

	// Update check (optionnel)
	if a.cfg.YtDlp.AutoUpdateCheck {
		a.YtDlpUpdateCheck(ctx, defaultUpdateTimeout, version)
	}

	// Extraction des métadonnées
	exCtx, exCancel := context.WithTimeout(ctx, defaultExtractTimeout)
	defer exCancel()

	raw, err := a.ytClient.ExtractRaw(exCtx, url)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return fmt.Errorf("opération annulée")
		}
		return fmt.Errorf("extract raw: %w", err)
	}
	raw.PrintWarnings()

	// parse métadonnées
	meta, err := yt.ParseYTDLP(raw.JSON)
	if err != nil {
		return fmt.Errorf("parse ytdlp: %w", err)
	}
	a.ui.PrintInfo(ctx, meta.Pretty())

	// préparation dossier de sortie + sauvegardes
	outDir := a.cfg.OutputDir
	if a.cfg.SaveInSubdir {
		outDir = filepath.Join(outDir, fsutil.SanitizeFilename(meta.Title))
	}
	if err := os.MkdirAll(outDir, dirPerm); err != nil {
		return fmt.Errorf("create out dir: %w", err)
	}

	if a.cfg.SaveRawJSON {
		pretty, err := raw.PrettyJSON()
		if err != nil {
			return err
		}
		jsonPath := filepath.Join(outDir, "metadata.json")
		if err := os.WriteFile(jsonPath, pretty, filePerm); err != nil {
			return fmt.Errorf("write metadata.json: %w", err)
		}
	}

	// téléchargement des sous-titre
	var subsSource model.SubSource
	if a.cfg.PreferManualSubs && meta.HasManualSubs() {
		subsSource = model.SubSourceManual
	} else {
		subsSource = model.SubSourceAutomatic
	}

	subsDownloaded, err := FetchSubtitleDownload(ctx, meta, subsSource)
	if err != nil {
		return err
	}

	if a.cfg.SaveRawSubs {
		if err := SaveSubtitleDownload(subsDownloaded, outDir); err != nil {
			return err
		}
	}
	// Création du transcript + sauvegarde
	transcript, err := BuildTranscriptFromSubtitleDownload(&subsDownloaded, meta)
	if err != nil {
		return err
	}
	if a.cfg.SaveTranscript {
		tFormat, err := model.ParseFormat(a.cfg.TranscriptFormat)
		if err != nil {
			return err
		}
		if err := SaveTranscript(transcript, tFormat, outDir); err != nil {
			return fmt.Errorf("échec de la sauvegarde du transcript: %w", err)
		}
	}

	var summary string
	if a.cfg.GenerateAIPrompt {
		// génération du prompt + copie dans le presse-papier.
		fullPrompt, err := BuildFullChatPrompt(transcript, a.cfg.PromptSplitThreshold)
		if err != nil {
			if errors.Is(err, ErrPromptTooLong) {
				fmt.Println("⚠️  Le prompt dépasse la limite, attention à la taille totale.")
			} else {
				return fmt.Errorf("app.go: %w", err)
			}
		}
		if err := clipboard.WriteAll(string(fullPrompt[0])); err != nil {
			return fmt.Errorf("app.go: %w", err)
		}

		initial, readErr := clipboard.ReadAll()
		if readErr != nil {
			a.ui.PrintError(ctx, fmt.Sprintf("warning: impossible de relire le presse-papier: %v", readErr))
		}
		a.ui.PrintInfo(ctx, "Prompt complet copié dans le presse-papier.")

		// interaction utilisateur
		resp, approved, err := a.WaitForAIResponse(ctx, initial)
		if err != nil {
			return fmt.Errorf("app.go: %w", err)
		}
		if approved {
			summary = resp
		} else {
			summary = ""
		}
	}

	// Création de la note
	noteData := obsidian.NewNoteData(meta, summary)
	var vaultDir string
	if a.cfg.ObsidianVaultDir == "" || a.cfg.ObsidianVaultDir == "." {
		vaultDir = outDir
	} else {
		vaultDir = a.cfg.ObsidianVaultDir
	}

	content, err := a.renderer.Render("obsidian_note.md.tmpl", noteData)
	if err != nil {
		return fmt.Errorf("render error: %v", err)
	}

	outPath, err := fsutil.SaveMarkdownAtomic(vaultDir, noteData.Filename, content, true)
	if err != nil {
		return fmt.Errorf("cannot save file to disk: %v", err)
	}
	fmt.Printf("Note écrite dans le répertoire:\n%s\n", outPath)

	// Attendre terminaison (Entrée OU Ctrl+C) via UI
	return a.ui.WaitForExit(ctx)
}
