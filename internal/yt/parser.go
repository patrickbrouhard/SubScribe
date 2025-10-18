package yt

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/patrickprogramme/subscribe/pkg/model"
)

const suffix = "-orig"

// ParseYTDLP transforme le JSON brut en struct Meta
func ParseYTDLP(raw []byte) (*model.Meta, error) {
	var y ytdlpOutput
	if err := json.Unmarshal(raw, &y); err != nil {
		return nil, fmt.Errorf("unmarshal ytdlp output: %w", err)
	}

	meta := &model.Meta{
		ID:          y.ID,
		Title:       y.Title,
		Uploader:    y.Uploader,
		Categories:  y.Categories,
		YtTags:      y.YtTags,
		Description: y.Description,
	}

	// upload_date: try YYYYMMDD puis timestamp (fallback)
	if y.UploadDate != "" {
		if t, err := time.Parse("20060102", y.UploadDate); err == nil {
			meta.UploadDate = t
		}
	}
	if meta.UploadDate.IsZero() && y.Timestamp != 0 {
		meta.UploadDate = time.Unix(y.Timestamp, 0).UTC()
	}

	// chapters
	for _, c := range y.Chapters {
		start := c.StartTime // StartTime est prioritaire: implémentation moderne
		if start == 0 {
			start = c.Start
		}
		meta.Chapters = append(meta.Chapters, model.Chapter{
			Start: model.Seconds(int64(math.Round(start))),
			Title: c.Title,
		})
	}

	// sous-titres manuels : on garde tout ce qui est au bon format
	manual := selectManualSubs(y.Subtitles, model.FormatJSON3)
	if len(manual) > 0 {
		meta.ManualSubs = append(meta.ManualSubs, manual...)
	}

	// sous-titres automatiques : on garde uniquement : bon format + orig
	auto := selectCaptionOriginal(y.AutomaticCaptions, model.FormatJSON3)
	if len(auto) > 0 {
		meta.AutoSubs = append(meta.AutoSubs, auto...)
	}

	return meta, nil
}

// selectCaptionOriginal parcourt la map `auto` (automatic_captions) et renvoie
// toutes les pistes dont la clé langue se termine par "-orig" et dont le format
// correspond au paramètre `format`.
func selectCaptionOriginal(auto map[string][]subtitleItem, format model.Format) []model.SubtitleTrack {
	var out []model.SubtitleTrack
	for lang, tracks := range auto {
		// on ne veut que les langues originales : -orig
		if !strings.HasSuffix(lang, suffix) {
			continue
		}

		// langClean := strings.TrimSuffix(lang, suffix)
		langClean := lang // temporaire, décommente au dessus et supprime cette ligne
		for _, it := range tracks {
			// ne garde qu'un format
			if pf, err := model.ParseFormat(it.Ext); err == nil && pf == format {
				st := model.SubtitleTrack{
					Lang:   langClean,
					Format: pf,
					URL:    it.URL,
					Source: model.SubSourceAutomatic,
				}
				out = append(out, st)
			}
		}
	}
	return out
}

// selectManualSubs récupère tous les sous-titres manuels
func selectManualSubs(manual map[string][]subtitleItem, format model.Format) []model.SubtitleTrack {
	var out []model.SubtitleTrack
	for lang, tracks := range manual {
		for _, it := range tracks {
			if pf, err := model.ParseFormat(it.Ext); err == nil && pf == format {
				out = append(out, model.SubtitleTrack{
					Lang:   lang,
					Format: pf,
					URL:    it.URL,
					Source: model.SubSourceManual,
				})
			}
		}
	}

	return out
}
