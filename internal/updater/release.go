package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/patrickprogramme/subscribe/pkg/github"
)

type rawRelease struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	PublishedAt time.Time `json:"published_at"`
	Body        string    `json:"body"`
	HTMLURL     string    `json:"html_url"`
	Assets      []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
		ContentType        string `json:"content_type"`
	} `json:"assets"`
}

// GetLatestYtDlpRelease découpe clairement les responsabilités
func GetLatestYtDlpRelease(ctx context.Context) (*YtDlpReleaseInfo, error) {
	data, err := github.FetchReleaseJSON(ctx, "yt-dlp", "yt-dlp")
	if err != nil {
		return nil, err
	}

	var raw rawRelease
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("décodage JSON: %w", err)
	}

	info := &YtDlpReleaseInfo{
		TagName:     raw.TagName,
		Name:        raw.Name,
		PublishedAt: raw.PublishedAt,
		Body:        raw.Body,
		HTMLURL:     raw.HTMLURL,
	}

	for _, a := range raw.Assets {
		switch a.Name {
		case "yt-dlp.exe":
			info.WindowsRelease = YtDlpAsset{a.Name, a.BrowserDownloadURL, a.ContentType}
		case "yt-dlp":
			info.LinuxRelease = YtDlpAsset{a.Name, a.BrowserDownloadURL, a.ContentType}
		}
	}

	if info.WindowsRelease.BrowserDownloadURL == "" {
		return nil, fmt.Errorf("asset Windows introuvable")
	}
	if info.LinuxRelease.BrowserDownloadURL == "" {
		return nil, fmt.Errorf("asset Linux introuvable")
	}

	return info, nil
}
