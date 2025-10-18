package subtitles

import "strings"

// rawJSON3 représente la structure "brute" telle qu'on la récupère depuis yt-dlp / YouTube json3.
type rawJSON3 struct {
	WireMagic string     `json:"wireMagic,omitempty"`
	Events    []rawEvent `json:"events"`
}

type rawEvent struct {
	TStartMs    *int64   `json:"tStartMs,omitempty"`
	DDurationMs *int64   `json:"dDurationMs,omitempty"`
	AAppend     *int     `json:"aAppend,omitempty"` // utilité ?
	Segs        []rawSeg `json:"segs,omitempty"`
	// On ignore volontairement d'autres champs (wpWinPosId, wWinId, etc.)
}

type rawSeg struct {
	Utf8      string `json:"utf8"`
	TOffsetMs *int64 `json:"tOffsetMs,omitempty"`
	// acAsrConf est censé donner le nv de confiance, mais a l'air inopérant côté Youtube, à voir.
}

// IsNewlineOnly indique si l'event est uniquement un retour à la ligne.
// Il retourne true pour des segs qui ne contiennent que "\n", "\\n" ou des espaces.
func (e rawEvent) IsNewlineOnly() bool {
	if len(e.Segs) == 0 {
		return false
	}
	for _, s := range e.Segs {
		t := strings.TrimSpace(s.Utf8)
		if t == "" {
			continue
		}
		if t == "\n" || t == "\\n" {
			continue
		}
		// si un seg contient du contenu non-newline, il n'est pas "NewlineOnly"
		return false
	}
	return true
}
