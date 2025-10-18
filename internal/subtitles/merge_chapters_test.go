package subtitles

import (
	"strings"
	"testing"

	"github.com/patrickprogramme/subscribe/pkg/model"
)

// helper : renvoie true si substr apparaît avant substr2 dans s (index >= 0)
func appearsBefore(s, substr, substr2 string) bool {
	i := strings.Index(s, substr)
	j := strings.Index(s, substr2)
	return i >= 0 && j >= 0 && i < j
}

func TestNearestTieChoosesNextPhrase(t *testing.T) {
	// phrases à 0 ms et 100 000 ms (0s et 100s)
	tr := Transcript{
		Phrases: []Phrase{
			{TimestampMs: 0, Text: "phrase1"},
			{TimestampMs: 100000, Text: "phrase2"},
		},
		Chapters: []model.Chapter{
			{Start: model.Seconds(50), Title: "Chap"}, // 50s -> midpoint exact
		},
	}

	out := tr.transcriptWithChaptersSplit(0, asPlain) // threshold 0 => toujours nudge

	// comportement actuel : midpoint -> voisin de droite (phrase2), le chapitre est attaché à phrase2
	if !appearsBefore(out, "## Chap", "phrase2") {
		t.Fatalf("expected chapter before phrase2 (tie chooses next), got:\n%s", out)
	}
}

func TestThresholdPreventsNudge(t *testing.T) {
	// phrases à 0s et 100s
	tr := Transcript{
		Phrases: []Phrase{
			{TimestampMs: 0, Text: "A"},
			{TimestampMs: 100000, Text: "B"},
		},
		Chapters: []model.Chapter{
			{Start: model.Seconds(51), Title: "C"}, // 51s -> nearest = B (dist = 49s = 49000ms)
		},
	}
	// threshold = 20000 ms (20s) -> dist (49000) > 20000 -> pas de nudge
	out := tr.transcriptWithChaptersSplit(20000, asPlain)

	// on veut la séquence A C B (chapitre entre les deux)
	if !appearsBefore(out, "A", "C") || !appearsBefore(out, "C", "B") {
		t.Fatalf("expected A C B order with threshold preventing nudge, got:\n%s", out)
	}
}

func TestCollapsedModeProducesChapterThenCollapsedContent(t *testing.T) {
	tr := Transcript{
		Phrases: []Phrase{
			{TimestampMs: 0, Text: "phrase1"},
			{TimestampMs: 100000, Text: "phrase2"},
		},
		Chapters: []model.Chapter{
			{Start: model.Seconds(51), Title: "ChapB"}, // nearest -> phrase2 and nudge with threshold 0
		},
	}

	out := tr.transcriptWithChaptersSplit(0, asCollapsed)

	if !strings.Contains(out, "## ChapB") {
		t.Fatalf("expected chapter header present, got:\n%s", out)
	}
	// ensure chapter appears before phrase2
	if !appearsBefore(out, "## ChapB", "phrase2") {
		t.Fatalf("expected chapter before phrase2 in collapsed, got:\n%s", out)
	}
}

func TestMultipleChaptersStableOrder(t *testing.T) {
	tr := Transcript{
		Phrases: []Phrase{
			{TimestampMs: 0, Text: "X"},
			{TimestampMs: 100000, Text: "Y"},
		},
		// deux chapitres avant la première phrase (chaps_before)
		Chapters: []model.Chapter{
			{Start: model.Seconds(0), Title: "C1"},
			{Start: model.Seconds(0), Title: "C2"},
		},
	}

	out := tr.transcriptWithChaptersSplit(0, asPlain)

	// on s'attend à conserver l'ordre relatif des chapitres (C1 avant C2)
	if !appearsBefore(out, "## C1", "## C2") {
		t.Fatalf("expected C1 before C2 for stability, got:\n%s", out)
	}
}

func TestNoPhrasesWithChaptersReturnsConcatenatedTitles(t *testing.T) {
	tr := Transcript{
		Phrases:  []Phrase{},
		Chapters: []model.Chapter{{Start: model.Seconds(0), Title: "OnlyChap1"}, {Start: model.Seconds(10), Title: "OnlyChap2"}},
	}

	out := tr.transcriptWithChaptersSplit(0, asPlain)
	// d'après ton code actuel, si len(phrases)==0 on renvoie la concaténation des Title tels quels
	if !strings.Contains(out, "OnlyChap1OnlyChap2") {
		t.Fatalf("expected concatenated chapter titles (raw), got:\n%s", out)
	}
}
