package updater

import (
	"time"
)

// YtDlpAsset représente un exécutable Windows ou Linux.
type YtDlpAsset struct {
	Name               string
	BrowserDownloadURL string
	ContentType        string
}

// YtDlpReleaseInfo contient les métadonnées de la release
// et les deux assets spécifiques à la mise à jour.
type YtDlpReleaseInfo struct {
	TagName        string
	Name           string
	PublishedAt    time.Time
	Body           string
	HTMLURL        string
	WindowsRelease YtDlpAsset
	LinuxRelease   YtDlpAsset
}
