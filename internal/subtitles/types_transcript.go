package subtitles

import (
	"github.com/patrickprogramme/subscribe/pkg/model"
)

// Phrase représente une phrase extraite des sous-titres,
// avec son timestamp (début de la phrase) en millisecondes.
type Phrase struct {
	TimestampMs int64  // début de la phrase (ms depuis le début de la vidéo)
	Text        string // texte normalisé de la phrase
	RuneCount   int    // nombre de runes unicode (len([]rune(Text)))
	WordCount   int    // nombre de mots (strings.Fields)
}

// Transcript représente le transcript résultant d'un traitement
// (parse raw json3 -> transformation en phrases -> post-traitement).
type Transcript struct {
	Title    string              // titre (hérité de SubtitleDownload)
	Track    model.SubtitleTrack // métadonnée sur la piste (hérité de SubtitleDownload)
	Phrases  []Phrase            // phrases extraites et post-traitées
	Chapters []model.Chapter
}

// NewTranscript construit un Transcript à partir de données déjà prêtes.
// - pure function, pas d'I/O ni de parsing.
func NewTranscript(title string, track model.SubtitleTrack, phrases []Phrase, chapters []model.Chapter) Transcript {
	return Transcript{
		Title:    title,
		Track:    track,
		Phrases:  phrases,
		Chapters: chapters,
	}
}
