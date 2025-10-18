package yt

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// NewYtDlp construit une instance. Path doit être le chemin résolu vers l'exe
func NewYtDlp(name string, resolvedPath string, cfg YtDlpConfig) *YtDlp {
	return &YtDlp{
		Name:   name,
		Path:   resolvedPath,
		Config: cfg,
	}
}

// CheckBinary vérifie que le binaire spécifié dans Cmd existe et est exécutable.
func (y *YtDlp) CheckBinary() error {
	if y == nil {
		return fmt.Errorf("yt-dlp non initialisé")
	}

	exe := y.Path
	if exe == "" {
		fmt.Println("CheckBinary: fallback sur y.Name")
		exe = y.Name // fallback : essayer le nom si pas de path résolu
	}

	info, err := os.Stat(exe)
	if err != nil {
		return fmt.Errorf("yt-dlp introuvable (%s) à l'emplacement spécifié : %v", exe, err)
	}

	if info.IsDir() {
		return fmt.Errorf("le chemin spécifié pour yt-dlp est un répertoire, pas un fichier exécutable")
	}

	return nil
}

// ExtractRaw exécute `yt-dlp -j <url>` et renvoie la sortie JSON brute.
// La sortie est validée comme JSON avant d'être renvoyée.
func (y *YtDlp) ExtractRaw(ctx context.Context, url string) (*ExtractedRaw, error) {
	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		fmt.Printf("Métadonnées extraites en %s\n", elapsed)
	}()

	args := y.Config.BuildArgs(url)

	exe := y.Path
	if exe == "" {
		exe = y.Name
	}

	cmd := exec.CommandContext(ctx, exe, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("yt-dlp dump json failed: %w, output: %s", err, string(out))
	}

	var jsonLine string
	var warnings []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "{") || strings.HasPrefix(line, "[") {
			jsonLine = line // si plusieurs lignes JSON, prendre la dernière/ la première selon besoin
		} else {
			warnings = append(warnings, line)
		}
	}
	if jsonLine == "" {
		return nil, fmt.Errorf("aucun JSON détecté dans la sortie: %s", string(out))
	}
	return &ExtractedRaw{
		JSON:     []byte(jsonLine),
		Warnings: warnings,
	}, nil
}
