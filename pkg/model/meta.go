package model

import (
	"fmt"
	"strings"
	"time"
)

// SubSource représente la provenance d'une piste de sous-titres.
// automatic = généré automatiquement par Youtube
// manual = fourni par l'auteur de la vidéo
type SubSource string

const (
	SubSourceUnknown   SubSource = "unknown"
	SubSourceAutomatic SubSource = "automatic"
	SubSourceManual    SubSource = "manual"
)

func (s SubSource) String() string {
	switch s {
	case SubSourceAutomatic:
		return "auto captions"
	case SubSourceManual:
		return "manual subtitles"
	default:
		return "unknown subtitles"
	}
}

// Chapter représente un chapitre d'une vidéo avec un timestamp et un titre.
type Chapter struct {
	Start Seconds `json:"start"`
	Title string  `json:"title"`
}

// SubtitleTrack décrit une piste de sous-titres associée à une vidéo.
type SubtitleTrack struct {
	Lang   string    `json:"lang"`
	Format Format    `json:"format,omitempty"`
	URL    string    `json:"url,omitempty"`
	Source SubSource `json:"source,omitempty"`
}

func (s SubtitleTrack) String() string {
	return fmt.Sprintf("SubtitleTrack(lang=%s, format=%s, source=%s)", s.Lang, s.Format, s.Source)
}

// Meta regroupe les métadonnées extraites d'une vidéo YouTube.
type Meta struct {
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	Uploader    string          `json:"uploader,omitempty"`
	UploadDate  time.Time       `json:"upload_date,omitempty"`
	Categories  []string        `json:"categories,omitempty"`
	YtTags      []string        `json:"yt_tags,omitempty"`
	Description string          `json:"description,omitempty"`
	Chapters    []Chapter       `json:"chapters,omitempty"`
	AutoSubs    []SubtitleTrack `json:"subtitles,omitempty"`
	ManualSubs  []SubtitleTrack `json:"manual_subtitles,omitempty"`
}

func (m Meta) HasManualSubs() bool {
	return len(m.ManualSubs) != 0
}

func (m Meta) HasAutoSubs() bool {
	return len(m.AutoSubs) != 0
}

func (m Meta) String() string {
	return fmt.Sprintf("Meta[ID=%s, Title=%q, Uploader=%s, Date=%s, Chapters=%d, Subtitles=%d]",
		m.ID, m.Title, m.Uploader, m.UploadDate.Format("2006-01-02"),
		len(m.Chapters), len(m.AutoSubs)+len(m.ManualSubs))
}

// Pretty retourne une fiche multi-lignes simple.
// Elle montre les langues présentes dans AutoSubs et ManualSubs
// en les listant telles qu'elles apparaissent dans les SubtitleTrack.
func (m Meta) Pretty() string {
	dateStr := "<unknown>"
	if !m.UploadDate.IsZero() {
		dateStr = m.UploadDate.Format("2006-01-02")
	}

	langsFrom := func(tracks []SubtitleTrack) []string {
		out := make([]string, 0, len(tracks))
		for _, t := range tracks {
			// on prend la valeur telle quelle ; vide -> on ignore
			if t.Lang != "" {
				out = append(out, t.Lang)
			}
		}
		return out
	}

	formatLangs := func(list []string) string {
		if len(list) == 0 {
			return "(aucun)"
		}
		return strings.Join(list, ", ")
	}

	return fmt.Sprintf(
		"Meta:\n"+
			"  ID         : %s\n"+
			"  Title      : %q\n"+
			"  Uploader   : %s\n"+
			"  Date       : %s\n"+
			"  Chapters   : %d\n"+
			"  AutoSubs   : %s\n"+
			"  ManualSubs : %s\n",
		m.ID,
		m.Title,
		m.Uploader,
		dateStr,
		len(m.Chapters),
		formatLangs(langsFrom(m.AutoSubs)),
		formatLangs(langsFrom(m.ManualSubs)),
	)
}
