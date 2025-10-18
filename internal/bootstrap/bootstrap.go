package bootstrap

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/patrickprogramme/subscribe/internal/fsutil"
)

// ExportDefaults copie récursivement tous les fichiers sous srcPrefix (dans fsys)
// vers destDir en préservant la hiérarchie relative.
// - fsys : embed.FS (ou tout fs.FS)
// - srcPrefix : chemin racine dans fsys à copier (ex: "templates")
// - destDir : dossier sur disque cible
// - force : si true, écrase les fichiers différents (avec backup)
//
// Retourne une map[embeddedPath]status et une erreur globale si Walk échoue.
func ExportDefaults(fsys fs.FS, srcPrefix, destDir string, force bool) (map[string]string, error) {
	status := make(map[string]string)

	err := fs.WalkDir(fsys, srcPrefix, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		// chemin relatif par rapport à srcPrefix
		rel, err := filepath.Rel(srcPrefix, path)
		if err != nil {
			return err
		}

		// skip root dir entry
		if rel == "." {
			if d.IsDir() {
				return nil
			}
		}

		// si c'est un répertoire : créer sur disque
		if d.IsDir() {
			if rel == "." {
				return nil
			}
			destPath := filepath.Join(destDir, rel)
			_ = os.MkdirAll(destPath, 0o755)
			return nil
		}

		// fichier : lire depuis embed
		data, err := fs.ReadFile(fsys, filepath.ToSlash(path))
		if err != nil {
			status[path] = "error: read embedded failed"
			return err
		}

		destPath := filepath.Join(destDir, rel)

		// si le fichier existe déjà : comparer
		if existing, err := os.ReadFile(destPath); err == nil {
			if bytes.Equal(existing, data) {
				status[path] = "unchanged"
				return nil
			}
			if !force {
				status[path] = "skipped (different)"
				return nil
			}
			// force == true -> backup + overwrite
			backup := destPath + ".bak." + time.Now().Format("20060102T150405")
			if err := os.WriteFile(backup, existing, 0o644); err != nil {
				status[path] = "error: backup failed"
				return fmt.Errorf("backup failed for %s: %w", destPath, err)
			}
			if err := fsutil.WriteFileAtomic(destPath, data, 0o644); err != nil {
				status[path] = "error: overwrite failed"
				return err
			}
			status[path] = "overwritten"
			return nil
		}

		// dest n'existe pas -> écrire atomiquement
		if err := fsutil.WriteFileAtomic(destPath, data, 0o644); err != nil {
			status[path] = "error: write failed"
			return err
		}
		status[path] = "written"
		return nil
	})

	return status, err
}

// EnsureTemplatesPresent s'assure que les templates listés existent sur disque.
//
// - tplDir  : dossier destination sur disque (ex: "./templates")
// - fsys    : embed.FS (ou autre fs.FS) contenant les ressources embarquées
// - srcFiles: liste explicite de chemins DANS fsys (ex: "templates/obsidian_note.md.tmpl")
//
// Comportement :
//  1. Si tplDir n'existe pas -> crée tplDir et copie TOUS les fichiers listés dans srcFiles.
//  2. Si tplDir existe et est vide -> même comportement (copie tous).
//  3. Si tplDir existe et n'est pas vide -> pour chaque fichier listé, si le fichier
//     correspondant est absent sur disque -> le copie depuis fsys.
//  4. NE REMPLACE JAMAIS les fichiers existants.
//
// Remarque : les chemins dans srcFiles doivent être utilisables avec fs.ReadFile(fsys, path).
func EnsureTemplatesPresent(tplDir string, fsys fs.FS, srcFiles []string) error {
	// 1) vérifier que le répertoire parent existe (sécurité)
	parent := filepath.Dir(tplDir)
	if parent == "" || parent == "." {
		parent = "."
	}
	if st, err := os.Stat(parent); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("le répertoire parent n'existe pas : %s", parent)
		}
		return fmt.Errorf("échec lors du test du répertoire parent %s : %w", parent, err)
	} else if !st.IsDir() {
		return fmt.Errorf("le parent existe mais n'est pas un répertoire : %s", parent)
	}

	// 2) si tplDir n'existe pas -> créer et copier tous les fichiers listés
	if _, err := os.Stat(tplDir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(tplDir, 0o755); err != nil {
				return fmt.Errorf("échec de création du répertoire de templates %s : %w", tplDir, err)
			}
			for _, src := range srcFiles {
				base := filepath.Base(src)
				dest := filepath.Join(tplDir, base)
				// lire le fichier embarqué (utiliser des slashs)
				data, rerr := fs.ReadFile(fsys, filepath.ToSlash(src))
				if rerr != nil {
					return fmt.Errorf("échec de lecture de la ressource embarquée %s : %w", src, rerr)
				}
				// écrire atomiquement sur disque
				if err := fsutil.WriteFileAtomic(dest, data, 0o644); err != nil {
					return fmt.Errorf("échec d'écriture du fichier %s : %w", dest, err)
				}
			}
			return nil
		}
		return fmt.Errorf("échec lors du test du répertoire de templates %s : %w", tplDir, err)
	}

	// 3) tplDir existe -> vérifier s'il est vide
	empty, err := fsutil.IsDirEmpty(tplDir)
	if err != nil {
		return fmt.Errorf("échec lors de la vérification du répertoire %s : %w", tplDir, err)
	}
	if empty {
		// comportement identique à tplDir manquant : copier tous les fichiers listés
		for _, src := range srcFiles {
			base := filepath.Base(src)
			dest := filepath.Join(tplDir, base)
			data, rerr := fs.ReadFile(fsys, filepath.ToSlash(src))
			if rerr != nil {
				return fmt.Errorf("échec de lecture de la ressource embarquée %s : %w", src, rerr)
			}
			if err := fsutil.WriteFileAtomic(dest, data, 0o644); err != nil {
				return fmt.Errorf("échec d'écriture du fichier %s : %w", dest, err)
			}
		}
		return nil
	}

	// 4) tplDir non vide -> n'ajouter que les fichiers manquants (ne pas écraser)
	for _, src := range srcFiles {
		base := filepath.Base(src)
		dest := filepath.Join(tplDir, base)
		if _, err := os.Stat(dest); err == nil {
			// le fichier existe déjà -> on saute
			continue
		} else if !os.IsNotExist(err) {
			// erreur inattendue lors du stat
			return fmt.Errorf("échec lors du test du fichier %s : %w", dest, err)
		}
		// fichier manquant -> lire depuis l'embed et écrire atomiquement
		data, rerr := fs.ReadFile(fsys, filepath.ToSlash(src))
		if rerr != nil {
			return fmt.Errorf("fichier embarqué introuvable %s : %w", src, rerr)
		}
		if err := fsutil.WriteFileAtomic(dest, data, 0o644); err != nil {
			return fmt.Errorf("échec d'écriture du template %s : %w", dest, err)
		}
	}
	return nil
}
