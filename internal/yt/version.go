package yt

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// GetVersion exécute le binaire yt-dlp avec l'option --version et retourne sa sortie.
// Utilise CombinedOutput pour capturer à la fois stdout et stderr,
// ce qui facilite le diagnostic en cas d'échec.
func (y *YtDlp) GetVersion(ctx context.Context) (string, error) {
	exe := y.Path
	if exe == "" {
		exe = y.Name
	}
	out, err := exec.CommandContext(ctx, exe, "--version").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("échec exécution yt-dlp --version : %w, output: %s", err, string(out))
	}
	return strings.TrimSpace(string(out)), nil
}
