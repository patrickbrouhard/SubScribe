package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"time"

	"github.com/patrickprogramme/subscribe/internal/fsutil"
	"github.com/patrickprogramme/subscribe/internal/ia"
	"github.com/patrickprogramme/subscribe/internal/subtitles"
	"github.com/patrickprogramme/subscribe/internal/ui"
	"github.com/patrickprogramme/subscribe/internal/updater"
	"github.com/patrickprogramme/subscribe/pkg/model"
)

var ErrPromptTooLong = errors.New("prompt dépasse le seuil autorisé")

// FetchSubtitleDownload télécharge la piste de sous-titres pour la meta `m` et la source `ss`.
func FetchSubtitleDownload(
	ctx context.Context, m *model.Meta, ss model.SubSource) (subtitles.SubtitleDownload, error) {
	var empty subtitles.SubtitleDownload
	maxBytes := int64(10_000_000)
	timeout := 15 * time.Second

	sd, err := subtitles.DownloadSubtitleFromMeta(ctx, m, ss, timeout, maxBytes)
	if err != nil {
		// s'il n'y a pas de sous-titres, ce n'est pas une erreur fatale...
		if errors.Is(err, subtitles.ErrNoSubtitle) {
			fmt.Printf("No %s available\n", string(ss))
			return empty, nil // ...on retourne la valeur vide + nil.
		}
		return empty, fmt.Errorf("download %s subtitles: %w", string(ss), err)
	}
	return sd, nil
}

// SaveSubtitleDownload sauvegarde le contenu de sd sur disque dans outDir.
// - utilise PrettyJSON() si disponible, sinon sd.Data brut
// - écrit de façon atomique via fsutil.WriteFileAtomic
// Retourne une erreur si l'écriture échoue ou si les données à écrire sont vides.
func SaveSubtitleDownload(sd subtitles.SubtitleDownload, outDir string) error {
	if len(sd.Data) == 0 {
		return fmt.Errorf("SaveSubtitleDownload: pas de données dans SubtitleDownload")
	}

	filename := sd.Filename()
	path := filepath.Join(outDir, filename)

	// préférence pour pretty JSON si disponible
	var dataToSave []byte
	if pretty, perr := sd.PrettyJSON(); perr == nil && len(pretty) > 0 {
		dataToSave = []byte(pretty)
	} else {
		dataToSave = sd.Data
	}

	if len(dataToSave) == 0 {
		return fmt.Errorf("SaveSubtitleDownload: pas de données à sauvegarder pour %s", filename)
	}

	if werr := fsutil.WriteFileAtomic(path, dataToSave, 0o644); werr != nil {
		return fmt.Errorf("write subtitle %s: %w", path, werr)
	}
	return nil
}

func BuildTranscriptFromSubtitleDownload(
	sd *subtitles.SubtitleDownload, m *model.Meta) (subtitles.Transcript, error) {
	var empty subtitles.Transcript

	if sd == nil {
		return empty, fmt.Errorf("BuildTranscript: SubtitleDownload est nil")
	}

	// parse JSON3
	raw, err := sd.ParseRawJSON3()
	if err != nil {
		return empty, fmt.Errorf("parse raw json3: %w", err)
	}

	// phrases, err := subtitles.TransformAutoRawToPhrases(raw)
	phrases, err := subtitles.TransformRawToPhrases(raw, sd.Track.Source)
	if err != nil {
		return empty, fmt.Errorf("transform subs: %w", err)
	}

	tr := subtitles.NewTranscript(sd.Title, sd.Track, phrases, m.Chapters)
	return tr, err
}

// SaveTranscript sauvegarde le transcript avec fsutil.WriteFileAtomic
func SaveTranscript(tr subtitles.Transcript, format model.Format, outDir string) error {
	if len(tr.Phrases) == 0 {
		return fmt.Errorf("SaveTranscript: pas de données Phrases dans SaveTranscript")
	}

	filename, err := tr.Filename(format)
	if err != nil {
		return fmt.Errorf("SaveTranscript: %w", err)
	}
	path := filepath.Join(outDir, filename)
	data := []byte(tr.Plain())
	if werr := fsutil.WriteFileAtomic(path, data, 0o644); werr != nil {
		return fmt.Errorf("write subtitle %s: %w", path, werr)
	}
	return nil
}

// BuildFullChatPrompt construit le prompt complet : prompt + texte.
// retourne [][]byte car on prévoit d'ajouter le prompt splitting
func BuildFullChatPrompt(t subtitles.Transcript, promptSplitThreshold int) ([][]byte, error) {
	var out [][]byte
	p, err := ia.GetChatPrompt()
	if err != nil {
		return nil, fmt.Errorf("erreur de construction du prompt: %w", err)
	}
	tc := t.Collapsed()
	fullLen := len(p) + len(tc)

	var ptc bytes.Buffer
	ptc.Write(p)
	ptc.WriteString("\n\n")
	ptc.WriteString(tc)

	out = append(out, ptc.Bytes())

	if fullLen > promptSplitThreshold {
		return out, fmt.Errorf("%w: taille totale %d > %d", ErrPromptTooLong, fullLen, promptSplitThreshold)
	}
	return out, nil
}

func (a *App) WaitForAIResponse(ctx context.Context, initialPrompt string) (string, bool, error) {
	// initialPrompt : le contenu copié initialement dans le clipboard (le prompt)
	if a.cfg.AutoMode {
		// mode auto -> polling
		interval := 500 * time.Millisecond
		timeout := time.Duration(300) * time.Second // remplacer 300 par TimeoutSec dans la config
		a.ui.PrintInfo(ctx, "Mode auto activé: surveillance du presse-papier en cours.")
		resp, err := a.ui.WaitForClipboardChange(ctx, initialPrompt, interval, timeout)
		if err != nil {
			return "", false, fmt.Errorf("en attente d'un changement du presse-papier: %w", err)
		}
		// on accepte automatiquement la première valeur différente
		return resp, true, nil
	}

	// mode interactif : demander à l'utilisateur d'indiquer qu'il a copié la réponse
	skip, err := a.ui.WaitForUserToCopyResponse(ctx)
	if err != nil {
		return "", false, fmt.Errorf("en attente que l'utilisateur copie la réponse: %w", err)
	}
	if skip {
		return "", false, nil
	}

	// ensuite, afficher preview et choix (GetClipboardChoice existant)
	for {
		content, choice, err := a.ui.GetClipboardChoice(ctx)
		if err != nil {
			return "", false, fmt.Errorf("get clipboard choice: %w", err)
		}
		switch choice {
		case ui.ChoiceUse: // approuvé
			return content, true, nil
		case ui.ChoiceSkip: // ignore et passe
			return "", false, nil
		case ui.ChoiceRetry:
			time.Sleep(250 * time.Millisecond)
			continue
		default:
			time.Sleep(250 * time.Millisecond)
			continue
		}
	}
}

func (a App) WaitForClipboardChoice(ctx context.Context) (string, bool, error) {
	for {
		content, choice, err := a.ui.GetClipboardChoice(ctx)
		if err != nil {
			return "", false, fmt.Errorf("a.ui.GetClipboardChoice: %w", err)
		}

		switch choice {
		case "use": // approuvé
			return content, true, nil
		case "skip": // ignore et passe
			return "", false, nil
		case "retry":
			time.Sleep(250 * time.Millisecond)
			continue
		default:
			time.Sleep(250 * time.Millisecond)
			continue
		}
	}
}

func (a App) YtDlpUpdateCheck(ctx context.Context, timeout time.Duration, version string) error {
	uc, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	check, err := updater.CheckYtDlpUpdate(uc, version)
	if err != nil {
		return fmt.Errorf("vérification de mise à jour a échoué : %v", err)
	}

	if check.IsUpToDate {
		a.ui.PrintInfo(ctx, fmt.Sprintf("✅ yt-dlp est à jour (%s)", check.CurrentVersion))
		return nil
	}

	a.ui.PrintInfo(ctx, "⚠️ Nouvelle version de Yt-dlp disponible :")
	a.ui.PrintInfo(ctx, fmt.Sprintf("  Installée : %s", check.CurrentVersion))
	a.ui.PrintInfo(ctx, fmt.Sprintf("  Dernière  : %s", check.LatestRelease.TagName))
	a.ui.PrintInfo(ctx, "Téléchargez-la ici:")
	a.ui.PrintInfo(ctx, check.GetUpdateLink(runtime.GOOS))

	return nil
}
