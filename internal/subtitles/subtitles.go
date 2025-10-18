package subtitles

import (
	"context"
	"fmt"
	"time"

	"github.com/patrickprogramme/subscribe/internal/fetch"
	"github.com/patrickprogramme/subscribe/pkg/model"
)

// NewSubtitleDownloadFromMeta : constructeur pur.
// Retourne un SubtitleDownload avec Title et Track remplis, Data == nil.
// convention? : le préfixe New implique pas d’effets de bord.
func NewSubtitleDownloadFromMeta(m *model.Meta, ss model.SubSource) (SubtitleDownload, bool) {
	if m == nil {
		return SubtitleDownload{}, false
	}
	var tracks []model.SubtitleTrack
	switch ss {
	case model.SubSourceManual:
		tracks = m.ManualSubs
	case model.SubSourceAutomatic:
		tracks = m.AutoSubs
	default:
		tracks = m.AutoSubs // fallback
	}

	for _, t := range tracks {
		if t.URL == "" {
			continue
		}
		title := tTitleOrID(m)
		return SubtitleDownload{
			Title: title,
			Track: t,
			Data:  nil,
		}, true
	}
	return SubtitleDownload{}, false
}

// tTitleOrID retourne le titre, ou sinon l'ID de la vidéo
func tTitleOrID(m *model.Meta) string {
	if m == nil {
		return ""
	}
	if s := m.Title; s != "" {
		return s
	}
	return m.ID
}

// DownloadSubtitleFromMeta : wrapper qui télécharge la piste et retourne
// le SubtitleDownload avec Data rempli. Nom explicite => fait du réseau.
//
// - ctx : contexte (annulation/timeout). Peut être nil.
// - timeout, maxBytes : paramètres pour fetch.FetchBytesWithTimeout.
// Retourne ErrNoSubtitle si aucune piste trouvée pour la source demandée.
func DownloadSubtitleFromMeta(ctx context.Context, m *model.Meta, ss model.SubSource, timeout time.Duration, maxBytes int64) (SubtitleDownload, error) {
	// constructeur pur
	sd, ok := NewSubtitleDownloadFromMeta(m, ss)
	if !ok {
		return SubtitleDownload{}, ErrNoSubtitle
	}

	// downloader (utilise internal/fetch helper)
	data, err := fetch.FetchBytesWithTimeout(ctx, sd.Track.URL, timeout, maxBytes)
	if err != nil {
		return SubtitleDownload{}, fmt.Errorf("download subtitle: %w", err)
	}
	sd.Data = data
	return sd, nil
}

// TransformRawToPhrases choisit la stratégie selon la source.
func TransformRawToPhrases(raw rawJSON3, src model.SubSource) ([]Phrase, error) {
	switch src {
	case model.SubSourceManual:
		return TransformManualRawToPhrases(raw)
	case model.SubSourceAutomatic:
		return TransformAutoRawToPhrases(raw)
	default:
		return TransformAutoRawToPhrases(raw) // choix par défaut
	}
}
