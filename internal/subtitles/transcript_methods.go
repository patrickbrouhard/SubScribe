package subtitles

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/patrickprogramme/subscribe/internal/fsutil"
	"github.com/patrickprogramme/subscribe/pkg/model"
)

// Plain retourne le transcript au format lisible (une phrase par ligne).
// Si des chapitres existent, ils sont insérés via TranscriptWithChaptersSplit(0, asPlain).
func (t Transcript) Plain() string {
	if len(t.Phrases) == 0 {
		return ""
	}
	if len(t.Chapters) == 0 {
		// pas de chapitres
		return t.PlainNoChapters()
	}
	return t.transcriptWithChaptersSplit(0, asPlain)
}

// PlainNoChapters retourne le transcript sous forme lisible : une phrase par ligne.
// Utile pour sauvegarde .txt simple.
func (t Transcript) PlainNoChapters() string {
	var b strings.Builder
	for i, p := range t.Phrases {
		// on écrit la phrase telle quelle (déjà normalisée)
		b.WriteString(p.Text)
		// fin de ligne sauf si c'est la dernière phrase (on ajoute tout de même un newline final)
		if i < len(t.Phrases)-1 {
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")
	return b.String()
}

// Collapsed retourne le transcript en un seul paragraphe (phrases collées).
// Si des chapitres existent, ils sont insérés via TranscriptWithChaptersSplit(0, asCollapsed).
func (t Transcript) Collapsed() string {
	if len(t.Phrases) == 0 {
		return ""
	}
	if len(t.Chapters) == 0 {
		// pas de chapitres
		return t.CollapsedNoChapters()
	}
	return t.transcriptWithChaptersSplit(0, asCollapsed)
}

// CollapsedNoChapters retourne le transcript en un seul paragraphe (tout sur une ligne).
// Utile pour envoyer à un LLM ?
func (t Transcript) CollapsedNoChapters() string {
	parts := make([]string, 0, len(t.Phrases))
	for _, p := range t.Phrases {
		parts = append(parts, strings.TrimSpace(p.Text))
	}
	// join par un espace unique et trim final
	out := strings.TrimSpace(strings.Join(parts, " ")) + "\n"
	return out
}

// SaveAs écrit le transcript dans le fichier `path` selon le format donné.
// Utilise les constantes model.Format (txt, md, srt, vtt, json3).
// Pour json3, SaveAs retourne une erreur : le raw JSON n'est pas stocké dans Transcript.
func (t Transcript) SaveAs(path string, format model.Format) error {
	var data []byte
	var err error

	switch format {
	case model.FormatTXT:
		data = []byte(t.PlainNoChapters())
	case model.FormatJSON3:
		// Transcript ne conserve pas le raw json3 original ; renvoyer une erreur
		return errors.New("SaveAs format json3 non supporté depuis Transcript (utiliser SubtitleDownload.Data)")
	default:
		return fmt.Errorf("format inconnu dans SaveAs: %s", format)
	}

	// écrire le fichier (écrase si existe)
	if err = os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("échec écriture fichier %s : %w", path, err)
	}
	return nil
}

func (t Transcript) Filename(format model.Format) (string, error) {
	base := strings.TrimSpace(t.Title)
	base = fsutil.SanitizeFilename(base)
	if format.IsTextual() {
		return base + format.Extension(), nil
	}
	return "", fmt.Errorf("format inconnu dans Filename: %q", format)
}

// isSentenceTerminatorRune déjà définie ailleurs ; redéfinie ici si nécessaire.
// func isSentenceTerminatorRune(r rune) bool { return r == '.' || r == '!' || r == '?' }
