package subtitles

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/patrickprogramme/subscribe/internal/fsutil"
	"github.com/patrickprogramme/subscribe/pkg/model"
)

var ErrNoSubtitle = errors.New("no subtitle available for given source")

// SubtitleDownload contient la piste + contexte utile (titre) + payload.
type SubtitleDownload struct {
	Title string
	Track model.SubtitleTrack
	Data  []byte // nil tant que non téléchargé
}

type Options struct {
	BreakOnAbbreviations bool
	Abbreviations        map[string]struct{} // pour recherche rapide
	MergeShortPhrases    bool
	MinPhraseWordsToKeep int
}

// Filename compose le nom de fichier pour ce SubtitleDownload à partir de s.Title
// (qui doit être renseigné par NewSubtitleDownloadFromMeta). Exemple :
// "The simplest tech stack (en).json"
func (s SubtitleDownload) Filename() string {
	base := strings.TrimSpace(s.Title)

	// sanitize le titre pour obtenir un nom de fichier sûr
	base = fsutil.SanitizeFilename(base)
	if strings.TrimSpace(base) == "" {
		// fallback de sécurité si sanitize rend la chaîne vide
		base = "subtitle"
	}

	// langue (fallback "und")
	lang := strings.TrimSpace(s.Track.Lang)
	if lang == "" {
		lang = "und"
	}

	// extension à partir du format
	ext := strings.ToLower(strings.TrimSpace(string(s.Track.Format)))
	ext = strings.TrimPrefix(ext, ".")
	if ext == "json3" {
		ext = "json"
	}
	if ext == "" {
		ext = "json"
	}

	filename := fmt.Sprintf("%s (%s).%s", base, lang, ext)
	return filepath.Base(filename)
}

// PrettyJSON retourne une version indentée du JSON contenu dans Data.
// Retourne une erreur si Data est vide ou si le contenu n'est pas un JSON valide.
// Note : staticcheck recommande d'éviter le test `s.Data == nil` car len(nil) == 0.
func (s SubtitleDownload) PrettyJSON() (string, error) {
	if len(s.Data) == 0 {
		return "", fmt.Errorf("no data to pretty-print")
	}

	// Vérifier que le format est JSON-like si possible
	ext := strings.ToLower(strings.TrimSpace(string(s.Track.Format)))
	if ext != "" && ext != "json" && ext != "json3" {
		return "", fmt.Errorf("pretty-print not supported for format %q", s.Track.Format)
	}

	var v interface{}
	if err := json.Unmarshal(s.Data, &v); err != nil {
		return "", fmt.Errorf("pretty json: decode error: %w", err)
	}

	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", fmt.Errorf("pretty json: marshal indent: %w", err)
	}
	return string(out), nil
}

// String implémente fmt.Stringer pour SubtitleDownload.
// Affiche Title, les infos essentielles du Track et si Data est présent.
func (s SubtitleDownload) String() string {
	dataPresent := len(s.Data) > 0
	dataLen := len(s.Data)

	// Préparer un aperçu de l'URL sans afficher des centaines de caractères
	urlPreview := s.Track.URL
	if urlPreview == "" {
		urlPreview = "<no url>"
	} else if len(urlPreview) > 80 {
		urlPreview = urlPreview[:77] + "..."
	}

	return fmt.Sprintf(
		"SubtitleDownload{Title:%q\n, Lang:%q, Format:%q, Source:%q\n, URL:%q\n, DataPresent:%t, DataLen:%d}",
		s.Title,
		s.Track.Lang,
		string(s.Track.Format),
		string(s.Track.Source),
		urlPreview,
		dataPresent,
		dataLen,
	)
}

// ParseRawJSON3 essaie de parser sd.Data et retourne la structure rawJSON3.
// Erreur si Data == nil ou parse fail.
func (sd *SubtitleDownload) ParseRawJSON3() (rawJSON3, error) {
	var empty rawJSON3
	if sd == nil {
		return empty, fmt.Errorf("ParseRawJSON3: SubtitleDownload data est nil")
	}
	if len(sd.Data) == 0 {
		return empty, fmt.Errorf("ParseRawJSON3: pas de données dans SubtitleDownload (nil/empty)")
	}
	return ParseJSON3Bytes(sd.Data)
}
