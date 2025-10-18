package fsutil

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// IsDirEmpty renvoie true si le répertoire spécifié par path est vide, false sinon.
func IsDirEmpty(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	if !info.IsDir() {
		return false, fmt.Errorf("%s is not a directory", path)
	}

	// Ouvre le répertoire
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	// Lit au plus un nom de fichier dans le répertoire
	_, err = f.Readdirnames(1)
	if err == io.EOF {
		// Pas d'entrée trouvée : dossier vide
		return true, nil
	}
	if err != nil {
		// Erreur d'accès au contenu
		return false, err
	}
	// Au moins une entrée existante → dossier non vide
	return false, nil
}

// DirHasMatchingFiles vérifie si le répertoire path contient au moins un fichier
// correspondant à l'un des motifs fournis dans patterns.
// - patterns utilise la syntaxe de filepath.Match/glob (ex: "*.md.tmpl").
// - La recherche n'est pas récursive ; elle cherche uniquement dans path.
// Renvoie (true, nil) si au moins un fichier correspond, (false, nil) s'il n'y en
// a pas, ou une erreur en cas de problème IO.
func DirHasMatchingFiles(path string, patterns []string) (bool, error) {
	// si le répertoire n'existe pas -> pas de fichiers correspondants
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	if !info.IsDir() {
		return false, errors.New("path exists but is not a directory")
	}

	for _, pat := range patterns {
		glob := filepath.Join(path, pat)
		matches, err := filepath.Glob(glob)
		if err != nil {
			// généralement filepath.Glob ne retourne pas d'erreur sauf motif invalide
			return false, err
		}
		if len(matches) > 0 {
			return true, nil
		}
	}
	return false, nil
}

// writeFileAtomic écrit data dans destPath de manière atomique : écriture dans
// un fichier temporaire du même répertoire puis os.Rename(tmp -> dest).
// Crée les répertoires parents si nécessaire.
// Quand téléchargera des fichiers plus gros (yt-dlp.exe) il faudra implémenter une
// écriture en streaming
//
// destPath : chemin complet vers le fichier cible.
// data : contenu à écrire.
// perm : permissions POSIX (ex: 0o644).
func WriteFileAtomic(destPath string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(destPath)
	if dir == "" {
		dir = "."
	}
	// repertoire parent existe ?
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}

	// creation fichier temp
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()

	// cleanup si échec
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}()

	// écriture
	if _, err := tmp.Write(data); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		// Garantit que les données sont physiquement stockées
		//  et pas juste en cache ("best-effort")
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	// set permission (best-effort)
	_ = os.Chmod(tmpName, perm)

	// rename
	if err := os.Rename(tmpName, destPath); err != nil {
		return fmt.Errorf("rename tmp -> dest: %w", err)
	}
	return nil
}

// SaveMarkdownAtomic écrit content dans outDir sous baseName+".md".
// - overwrite=false : si le fichier existe, on ajoute un suffixe _1, _2, ...
// - overwrite=true  : on écrase directement (écriture atomique via tmp+rename).
// Retourne le chemin final du fichier.
func SaveMarkdownAtomic(outDir, baseName string, content []byte, overwrite bool) (string, error) {
	if baseName == "" {
		return "", fmt.Errorf("baseName empty")
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir %s: %w", outDir, err)
	}

	// construction du nom final
	final := filepath.Join(outDir, baseName+".md")

	// gestion collision si on ne veut pas overwrite
	if !overwrite {
		if _, err := os.Stat(final); err == nil {
			// incrémenter suffixe _1, _2, ...
			const maxAttempts = 1000
			for i := 1; i <= maxAttempts; i++ {
				candidate := filepath.Join(outDir, fmt.Sprintf("%s_%d.md", baseName, i))
				if _, err := os.Stat(candidate); os.IsNotExist(err) {
					final = candidate
					break
				}
			}
			// si au bout des essais le fichier existe encore, fallback timestamp
			if _, err := os.Stat(final); err == nil {
				final = filepath.Join(outDir, fmt.Sprintf("%s_%d.md", baseName, time.Now().Unix()))
			}
		}
	}

	// écrire dans un tmp file local au même dossier pour rename atomique
	tmp := final + ".tmp"
	if err := os.WriteFile(tmp, content, 0o644); err != nil {
		return "", fmt.Errorf("write tmp file %s: %w", tmp, err)
	}
	// rename atomique
	if err := os.Rename(tmp, final); err != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("rename tmp->final: %w", err)
	}
	return final, nil
}
