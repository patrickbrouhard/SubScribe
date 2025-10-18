package yt

// YtDlpConfig représente les flags ajoutables quand on utilise yt-dlp
type YtDlpConfig struct {
	SkipDownload bool
	NoWarnings   bool // true => ajouter --no-warnings
	NoProgress   bool
	NoUpdate     bool
	NoConfig     bool // true => ajouter --no-config pour ignorer les configs utilisateur
}

// NewYtDlpConfig initalise une configuration standard de yt-dlp, showWarning vient du yaml de config
func NewYtDlpConfig(showWarning bool) *YtDlpConfig {
	return &YtDlpConfig{
		SkipDownload: true,
		NoWarnings:   !showWarning,
		NoProgress:   true,
		NoUpdate:     true,
		NoConfig:     true, // valeur par défaut : ignorer les fichiers de config extérieurs (plus prévisible)
	}
}

// BuildArgs construit une slice des arguments à passer à yt-dlp.
func (c *YtDlpConfig) BuildArgs(url string) []string {
	args := make([]string, 0, 8)
	// mettre --no-config en tête pour éviter que des configs locales/modifient le comportement
	if c.NoConfig {
		args = append(args, "--no-config")
	}
	args = append(args, "-j")
	if c.SkipDownload {
		args = append(args, "--skip-download")
	}
	if c.NoWarnings {
		args = append(args, "--no-warnings")
	}
	if c.NoProgress {
		args = append(args, "--no-progress")
	}
	if c.NoUpdate {
		args = append(args, "--no-update")
	}
	args = append(args, url)
	return args
}
