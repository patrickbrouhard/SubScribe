package config

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/patrickprogramme/subscribe/internal/fsutil"
	"gopkg.in/yaml.v3"
)

// CurrentConfigVersion définie dans migrate.go (ou config.go) — s'assurer que valeur identique
// const CurrentConfigVersion = 1  // si pas défini dans config.go

// orchestrateConfigUpgrade : sauvegarde, migration, écriture
func orchestrateConfigUpgrade(cfg *Config, fromVersion int) error {
	if cfg == nil {
		return fmt.Errorf("config nil lors de la migration")
	}
	if cfg.configFilePath == "" {
		return fmt.Errorf("chemin du fichier de configuration inconnu : impossible de faire une sauvegarde")
	}

	// 1) backup
	backupPath, err := backupConfig(cfg.configFilePath)
	if err != nil {
		return fmt.Errorf("échec de la sauvegarde du fichier de configuration avant migration : %w", err)
	}

	// 2) appliquer migrations successives
	if err := migrateConfig(cfg, fromVersion); err != nil {
		return fmt.Errorf("échec lors de la migration de la configuration (depuis %d) : %w", fromVersion, err)
	}

	// 2b) normaliser au cas où la migration aurait introduit des valeurs à nettoyer
	cfg.normalizeConfig()
	// ajoute .exe si nécessaire
	if runtime.GOOS == "windows" && !strings.HasSuffix(strings.ToLower(cfg.YtDlp.Name), ".exe") {
		cfg.YtDlp.Name = cfg.YtDlp.Name + ".exe"
	}

	// mettre à jour la version EN MÉMOIRE (et, si pertinent, remonter)
	cfg.ConfigVersion = CurrentConfigVersion

	// 3) sérialiser la config en YAML (indentation 2)
	b, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("échec d'encodage YAML de la configuration migrée : %w", err)
	}

	// 4) écrire atomiquement le YAML
	if err := fsutil.WriteFileAtomic(cfg.configFilePath, b, 0o644); err != nil {
		// tentative de restauration depuis la sauvegarde (meilleure résilience)
		_ = fsutil.WriteFileAtomic(cfg.configFilePath, mustReadFileOrEmpty(backupPath), 0o644)
		return fmt.Errorf("échec d'écriture du fichier de configuration migré %s : %w", cfg.configFilePath, err)
	}

	// log d'info
	fmt.Printf("info : configuration mise à jour de la version %d à %d (sauvegarde : %s)\n", fromVersion, CurrentConfigVersion, backupPath)
	return nil
}

// mustReadFileOrEmpty lit le contenu d'un fichier, et retourne un slice vide en cas d'erreur
func mustReadFileOrEmpty(path string) []byte {
	if path == "" {
		return []byte{}
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return []byte{}
	}
	return b
}

// backupConfig : sauvegarde le fichier de config et retourne le chemin de la sauvegarde
func backupConfig(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("lecture du fichier pour sauvegarde impossible : %w", err)
	}
	backup := path + ".bak." + time.Now().Format("20060102T150405")
	if err := fsutil.WriteFileAtomic(backup, data, 0o644); err != nil {
		return "", fmt.Errorf("écriture de la sauvegarde %s impossible : %w", backup, err)
	}
	return backup, nil
}

// migrateConfig : appliquer les transformations nécessaires entre versions
// ici tu implémentes les étapes concrètes ; pour l'instant c'est un squelette idempotent.
func migrateConfig(cfg *Config, from int) error {
	if cfg == nil {
		return fmt.Errorf("pas de fichier congif fourni")
	}
	// Exemple : si from == 0 -> migrer vers 1
	// on peut utiliser un switch et appliquer les étapes successives
	for v := from; v < CurrentConfigVersion; v++ {
		switch v {
		case 0:
			// migration 0 -> 1 : exemple (rien à faire pour l'instant)
			// ex: if cfg.TranscriptFormat == "" { cfg.TranscriptFormat = "txt" }
		// case 1:
		// migration 1 -> 2 : ...
		default:
			// pas de changement par défaut
		}
	}
	return nil
}
