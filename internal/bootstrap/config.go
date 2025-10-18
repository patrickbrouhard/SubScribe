package bootstrap

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/patrickprogramme/subscribe/internal/fsutil"
)

// EnsureConfigPresent copie un fichier embarqué (assetPath dans fsys) vers dstPath
// si dstPath n'existe pas encore.
// - dstPath : chemin complet sur disque (ex: binDir/subscribe.yaml)
// - fsys : embed.FS (ou autre fs.FS) contenant l'asset
// - assetPath : chemin dans fsys vers l'asset (ex: "subscribe.example.yaml")
// Comportement : idempotent, ne remplace jamais un fichier existant.
func EnsureConfigPresent(dstPath string, fsys fs.FS, assetPath string) error {
	// sécurité: vérifier parent
	parent := filepath.Dir(dstPath)
	if parent == "" {
		parent = "."
	}
	if st, err := os.Stat(parent); err != nil {
		if os.IsNotExist(err) {
			// créer le dossier parent si absent (on suppose qu'on peut écrire à cet emplacement)
			if err := os.MkdirAll(parent, 0o755); err != nil {
				return fmt.Errorf("échec création répertoire parent %s: %w", parent, err)
			}
		} else {
			return fmt.Errorf("échec test parent %s: %w", parent, err)
		}
	} else if !st.IsDir() {
		return fmt.Errorf("le parent existe mais n'est pas un répertoire : %s", parent)
	}

	// si le fichier existe déjà -> ne rien faire
	if _, err := os.Stat(dstPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("échec stat fichier cible %s: %w", dstPath, err)
	}

	// lire l'asset embarqué
	data, err := fs.ReadFile(fsys, filepath.ToSlash(assetPath))
	if err != nil {
		return fmt.Errorf("lecture asset embarqué %s: %w", assetPath, err)
	}

	// écrire atomiquement
	if err := fsutil.WriteFileAtomic(dstPath, data, 0o644); err != nil {
		return fmt.Errorf("échec écriture config %s: %w", dstPath, err)
	}

	// log info
	fmt.Printf("info: created default config at %s\n", dstPath)

	return nil
}
