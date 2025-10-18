package yt

import (
	"context"
	"fmt"
	"time"

	"github.com/patrickprogramme/subscribe/internal/config"
)

const defaultVersionTimeout = 5 * time.Second

// InitYtDlp initialise le client YtDlp, vérifie le binaire et récupère la version.
// Retourne le client (implémentant Interface) et la version.
func InitYtDlp(ctx context.Context, cfg *config.Config) (Interface, string, error) {
	ytDlpcfg := NewYtDlpConfig(cfg.YtDlp.ShowWarnings)
	dl := NewYtDlp(cfg.YtDlp.Name, cfg.YtDlp.ResolvedPath, *ytDlpcfg)
	dl.ShowPath()

	// vérifier la présence du binaire
	if err := dl.CheckBinary(); err != nil {
		return nil, "", fmt.Errorf("yt-dlp introuvable : %w", err)
	}

	// récupérer la version (avec timeout)
	vctx, cancel := context.WithTimeout(ctx, defaultVersionTimeout)
	defer cancel()
	version, err := dl.GetVersion(vctx)
	if err != nil {
		return dl, "", fmt.Errorf("échec récupération version yt-dlp : %w", err)
	}

	return dl, version, nil
}
