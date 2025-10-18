package obsidian

import (
	"fmt"
	"html"
	"regexp"
	"strings"

	"github.com/patrickprogramme/subscribe/internal/fsutil"
	"github.com/patrickprogramme/subscribe/pkg/model"
)

const baseYtURL = "https://www.youtube.com/watch?v="

// rawTagRe : match un hashtag #suivi_dun_mot.
// - capture (grp[1]) le texte sans le `#`
// - autorise lettres Unicode (\p{L}), chiffres (\p{N}), underscore et tiret
var rawTagRe = regexp.MustCompile(`#([\p{L}\p{N}_-]+)`)
var baseTags = []string{"youtube", "source"}

// NoteData contient les données "brutes" pour la note.
type NoteData struct {
	URL         string
	Title       string
	Uploader    string
	DateStr     string // formaté YYYY-MM-DD
	Categories  []string
	Tags        []string
	Hashtags    []string
	YtTags      []string
	Description string
	Chapters    []model.Chapter
	Filename    string
	Summary     string
}

func (n NoteData) DisplayHashtags() {
	fmt.Println("Hashtags:", strings.Join(n.Hashtags, ", "))
}

func (n NoteData) DisplayYtTags() {
	fmt.Println("YtTags:", strings.Join(n.YtTags, ", "))
}

// NewNoteData construit NoteData à partir de model.Meta
func NewNoteData(m *model.Meta, summary string) NoteData {
	url := baseYtURL + m.ID

	var suffixe string
	dateStr := "unknown"
	if !m.UploadDate.IsZero() {
		dateStr = m.UploadDate.Format("2006-01-02")
		suffixe = dateStr
	} else {
		suffixe = m.ID
	}

	// tags par défaut, minimum obligatoire
	tags := baseTags

	// hashtags dérivés depuis catégories (simple transformation)
	hashtags := findRawTags(m.Description)

	base := fsutil.SanitizeFilename(m.Title)
	filename := fmt.Sprintf("%s %s", base, suffixe)
	t := fsutil.CapitalizeFirst(m.Title)

	return NoteData{
		URL:         url,
		Title:       t,
		Uploader:    m.Uploader,
		DateStr:     dateStr,
		Categories:  m.Categories,
		Tags:        tags,
		Hashtags:    hashtags,
		YtTags:      m.YtTags,
		Description: m.Description,
		Chapters:    m.Chapters,
		Filename:    filename,
		Summary:     summary,
	}
}

// findRawTags trouve tous les hastags mentionnés dans une chaine, et les retourne en []string
func findRawTags(text string) []string {
	if text == "" {
		return nil
	}
	// Décodage HTML entités. Exemple Caf&eacute -> Café
	text = html.UnescapeString(text)

	matches := rawTagRe.FindAllStringSubmatch(text, -1)

	tags := make([]string, 0, len(matches))
	for _, grp := range matches {
		// grp[1] = contenu sans le #
		word := strings.ToLower(grp[1])
		tags = append(tags, word)
		if len(tags) > 64 {
			break
		}
	}
	return tags
}
