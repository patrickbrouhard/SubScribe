package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/patrickprogramme/subscribe/internal/app"
	"github.com/patrickprogramme/subscribe/internal/assets"
	"github.com/patrickprogramme/subscribe/internal/bootstrap"
	"github.com/patrickprogramme/subscribe/internal/config"
	"github.com/patrickprogramme/subscribe/internal/obsidian"
	"github.com/patrickprogramme/subscribe/internal/ui"
)

func main() {
	flags := parseFlags()

	// déterminer exePath/binDir
	binDir := "."
	exePath, err := os.Executable()
	if err != nil {
		log.Printf("impossible de déterminer le chemin de l'executable: %v", err)
	} else {
		binDir = filepath.Dir(exePath)
		fmt.Printf("Lancement depuis: %s\n", exePath)
	}

	// emplacement config par défaut
	if flags.ConfigPath == "subscribe.yaml" || flags.ConfigPath == "" {
		flags.ConfigPath = filepath.Join(binDir, "subscribe.yaml")
	}

	// s'assurer que le fichier config existe, si non on le crée
	if err := bootstrap.EnsureConfigPresent(
		flags.ConfigPath,
		assets.Embedded,
		assets.DefaultConfigAsset,
	); err != nil {
		log.Printf("erreur: EnsureConfigPresent: %v", err)
	}

	// s'assurer que les templates existent (dans binDir/templates)
	tplDir := filepath.Join(binDir, "templates")
	if err := bootstrap.EnsureTemplatesPresent(
		tplDir,
		assets.Embedded,
		assets.DefaultTemplatePaths,
	); err != nil {
		log.Printf("warning: ensure templates present: %v", err)
	}

	// charger la config depuis flags.ConfigPath (qui pointe vers binDir/subscribe.yaml si par défaut)
	cfg, err := config.Load(flags.ConfigPath)
	if err != nil {
		log.Fatalf("config load: %v", err)
	}

	// appliquer le flag -auto par-dessus la config
	if flags.Auto {
		cfg.AutoMode = true
	}

	// construction du renderer
	renderer, err := obsidian.DefaultRenderer(exePath)
	if err != nil {
		log.Fatalf("impossible de construire le renderer: %v", err)
	}

	// root context qui s'annule sur SIGINT / SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	tui := ui.NewTerminal()
	a := app.New(cfg, tui, flags, renderer)
	if err := a.Run(ctx); err != nil {
		log.Fatalf("app run: %v", err)
	}
}

func parseFlags() *app.CLIFlags {
	f := &app.CLIFlags{}
	flag.StringVar(&f.ConfigPath, "config", "subscribe.yaml", "path to config file")
	flag.StringVar(&f.URL, "url", "", "YouTube URL (optional)")
	flag.BoolVar(&f.Auto, "auto", false, "exécution automatique sans interaction")
	flag.StringVar(&f.YtDlpPath, "yt-dlp-path", "", "chemin absolu vers l'exécutable yt-dlp")
	flag.Parse()
	return f
}
