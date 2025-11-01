package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/patrickprogramme/subscribe/internal/assets"
	"github.com/patrickprogramme/subscribe/internal/fsutil"
	"gopkg.in/yaml.v3"
)

const CurrentConfigVersion = 1

// struct pour les paramètres de configuration
type Config struct {
	// Chemins
	OutputDir        string `yaml:"output_dir"`
	ObsidianVaultDir string `yaml:"obsidian_output_dir"`

	// Organisation
	SaveInSubdir bool `yaml:"save_in_subdir"`

	// Métadonnées
	SaveRawJSON bool `yaml:"save_raw_json"`

	// Sous-titres
	PreferManualSubs bool `yaml:"prefer_manual_subs"`
	SaveRawSubs      bool `yaml:"save_raw_subs"`

	// Transcription
	SaveTranscript   bool   `yaml:"save_transcript"`
	TranscriptFormat string `yaml:"transcript_format"`

	// Mode automatique
	AutoMode bool `yaml:"auto_mode"`

	// AI features
	GenerateAIPrompt     bool `yaml:"generate_ai_prompt"`
	PromptSplitThreshold int  `yaml:"prompt_split_threshold"`

	// yt-dlp
	YtDlp struct {
		Name            string `yaml:"name"`
		Path            string `yaml:"path"`
		ShowWarnings    bool   `yaml:"show_warnings"`
		AutoUpdateCheck bool   `yaml:"auto_update_check"`

		// ResolvedPath contient le chemin effectif vers l'exécutable
		ResolvedPath string `yaml:"-"`
	} `yaml:"yt_dlp"`

	ConfigVersion int `yaml:"config_version"`

	configFilePath string
}

// configuration par défaut
// Configuration par défaut (fallback si l'asset embarqué est manquant)
func defaultConfig() *Config {
	c := &Config{}

	// Chemins
	c.OutputDir = "."
	c.ObsidianVaultDir = ""

	// Organisation
	c.SaveInSubdir = true

	// Métadonnées
	c.SaveRawJSON = false

	// Sous-titres
	c.PreferManualSubs = true
	c.SaveRawSubs = false

	// Transcription
	c.SaveTranscript = true
	c.TranscriptFormat = "txt"

	// Mode automatique
	c.AutoMode = false

	// AI features
	c.GenerateAIPrompt = true
	c.PromptSplitThreshold = 32000

	// yt-dlp
	c.YtDlp.Name = "yt-dlp"
	c.YtDlp.Path = ""
	c.YtDlp.ShowWarnings = false
	c.YtDlp.AutoUpdateCheck = false

	c.ConfigVersion = CurrentConfigVersion

	return c
}

// Load lit la config; si le fichier n'existe pas, on copie l'exemple embarqué depuis internal/assets
func Load(path string) (*Config, error) {
	if path == "" {
		path = "subscribe.yaml"
	}

	// si le fichier n'existe pas -> essayer de créer à partir de l'asset embarqué
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := createDefaultConfigFromEmbedded(path); err != nil {
			return nil, fmt.Errorf("échec de création du fichier de configuration par défaut : %w", err)
		}
	}

	cfg := defaultConfig()

	// lire le YAML brut et déserialiser dans cfg (les champs présents écraseront les defaults)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("lecture du fichier de configuration %s impossible : %w", path, err)
	}

	// corriger les chemins Windows avec des backslashes
	data = bytes.ReplaceAll(data, []byte(`\`), []byte(`/`))

	// On déserialise dans cfg initialisé : les champs absents conservent les valeurs par défaut.
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("analyse du fichier de configuration %s impossible : %w", path, err)
	}
	cfg.configFilePath = path

	cfg.normalizeConfig()

	// gestion de version : si le fichier est plus ancien -> orchestrer la mise à jour
	if cfg.ConfigVersion < CurrentConfigVersion {
		// orchestrateConfigUpgrade doit faire la sauvegarde, migrer et écrire la config
		if err := orchestrateConfigUpgrade(cfg, cfg.ConfigVersion); err != nil {
			return nil, fmt.Errorf("échec de mise à niveau de la configuration : %w", err)
		}
		// re-normaliser au cas où la migration a modifié des valeurs
		cfg.normalizeConfig()
	}

	return cfg, nil
}

func createDefaultConfigFromEmbedded(dstPath string) error {
	// lire l'asset embarqué via assets.Embedded et DefaultConfigAsset
	b, err := assets.Embedded.ReadFile(assets.DefaultConfigAsset)
	if err != nil {
		return fmt.Errorf("lecture du modèle de configuration embarqué impossible : %w", err)
	}

	// s'assurer que le dossier parent existe
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return fmt.Errorf("échec mkdir pour la configuration %s : %w", filepath.Dir(dstPath), err)
	}

	// écrire atomiquement sur disque (évite les fichiers partiels)
	if err := fsutil.WriteFileAtomic(dstPath, b, 0o644); err != nil {
		return fmt.Errorf("échec d'écriture du fichier de configuration %s : %w", dstPath, err)
	}

	// log utile pour le debugging
	fmt.Printf("info : fichier de configuration par défaut créé : %s\n", dstPath)
	return nil
}

func (c *Config) normalizeConfig() {
	// Nettoyage des chemins
	c.OutputDir = filepath.Clean(c.OutputDir)
	c.ObsidianVaultDir = filepath.Clean(c.ObsidianVaultDir)

	// Trim and normalize strings
	c.TranscriptFormat = strings.TrimSpace(strings.ToLower(c.TranscriptFormat))
	if c.TranscriptFormat == "" {
		c.TranscriptFormat = "txt"
	}

	if c.PromptSplitThreshold <= 0 {
		c.PromptSplitThreshold = 32000
	}

	// centraliser la résolution/normalisation de yt-dlp
	c.ResolveYtDlpPath()
}

// ResolveYtDlpPath normalise le nom et résout le chemin complet vers l'exécutable.
// Appeler après avoir modifié cfg.YtDlp.Name ou cfg.YtDlp.Path.
func (c *Config) ResolveYtDlpPath() {
	if c == nil {
		return
	}

	// Normaliser le nom et ajouter .exe sur Windows si nécessaire
	c.YtDlp.Name = strings.TrimSpace(c.YtDlp.Name)
	if c.YtDlp.Name == "" {
		c.YtDlp.Name = "yt-dlp"
	}

	// ajoute .exe si nécessaire
	if runtime.GOOS == "windows" && !strings.HasSuffix(strings.ToLower(c.YtDlp.Name), ".exe") {
		c.YtDlp.Name = c.YtDlp.Name + ".exe"
	}

	// Résolution du chemin
	// si cfg.Path est vide -> "./<exe>"
	exeName := c.YtDlp.Name
	cfgPath := strings.TrimSpace(c.YtDlp.Path)
	if cfgPath == "" {
		relativePath := "./" + exeName
		c.YtDlp.ResolvedPath = relativePath
		return
	}
	cleanPath := filepath.Clean(cfgPath)

	// si le chemin fourni finit déjà par l'exécutable -> on l'utilise
	if filepath.Base(cleanPath) == exeName {
		c.YtDlp.ResolvedPath = cleanPath
	} else {
		// sinon on considère cfgPath comme un répertoire et on y joint l'exe
		c.YtDlp.ResolvedPath = filepath.Join(cleanPath, exeName)
	}
}
