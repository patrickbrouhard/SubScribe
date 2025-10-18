package updater

import (
	"context"
	"fmt"
)

// UpdateCheck contient le résultat de la comparaison
type UpdateCheck struct {
	CurrentVersion string            // version récupérée localement
	LatestRelease  *YtDlpReleaseInfo // info complète de la release distante
	IsUpToDate     bool              // true si CurrentVersion == LatestRelease.TagName
}

// CheckYtDlpUpdate compare la version locale et la version GitHub.
func CheckYtDlpUpdate(ctx context.Context, localVer string) (*UpdateCheck, error) {

	// Récupérer la release distante
	latest, err := GetLatestYtDlpRelease(ctx)
	if err != nil {
		return nil, fmt.Errorf("impossible de récupérer la release GitHub : %w", err)
	}

	// 3. Comparer
	isUpToDate := localVer == latest.TagName

	return &UpdateCheck{
		CurrentVersion: localVer,
		LatestRelease:  latest,
		IsUpToDate:     isUpToDate,
	}, nil
}

func (u UpdateCheck) GetUpdateLink(system string) string {
	if system == "windows" {
		return u.LatestRelease.WindowsRelease.BrowserDownloadURL
	}
	return u.LatestRelease.LinuxRelease.BrowserDownloadURL
}
