package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidateYtDlpPresence vérifie de manière statique que si un ResolvedPath est défini,
// le fichier existe et que le répertoire parent est accessible.
// Retourne warnings (non-fataux) et une erreur si c'est critique.
func (c *Config) ValidateYtDlpPresence() (warnings []string, err error) {
	if c == nil {
		return nil, fmt.Errorf("config nil")
	}

	// assure que le resolved path est calculé
	c.ResolveYtDlpPath()

	p := strings.TrimSpace(c.YtDlp.ResolvedPath)
	if p == "" {
		// pas de chemin résolu : on ne considère pas ça comme une erreur fatale ici,
		// la découverte dans PATH ou l'installation peut être tentée plus tard.
		warnings = append(warnings, "aucun chemin résolu pour yt-dlp; recherche dans PATH possible")
		return warnings, nil
	}

	parent := filepath.Dir(p)
	if st, serr := os.Stat(parent); serr != nil {
		if os.IsNotExist(serr) {
			warnings = append(warnings, fmt.Sprintf("le dossier parent du chemin yt-dlp n'existe pas : %s", parent))
		} else {
			return warnings, fmt.Errorf("impossible d'accéder au dossier parent %s : %w", parent, serr)
		}
	} else if !st.IsDir() {
		return warnings, fmt.Errorf("le parent du chemin yt-dlp n'est pas un répertoire : %s", parent)
	}

	// vérifier si le fichier existe (stat)
	if info, serr := os.Stat(p); serr != nil {
		if os.IsNotExist(serr) {
			warnings = append(warnings, fmt.Sprintf("yt-dlp introuvable à l'emplacement configuré : %s", p))
			return warnings, nil
		}
		return warnings, fmt.Errorf("erreur lors du test du fichier %s : %w", p, serr)
	} else {
		if info.IsDir() {
			return warnings, fmt.Errorf("le chemin configuré pour yt-dlp est un répertoire : %s", p)
		}
		// tout ok : on peut garder le resolved path tel quel
	}

	return warnings, nil
}
