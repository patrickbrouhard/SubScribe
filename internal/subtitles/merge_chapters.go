package subtitles

import (
	"slices"
	"sort"
	"strings"

	"github.com/patrickprogramme/subscribe/pkg/model"
)

// event représente un élément temporel dans la ligne du transcript.
// Il peut s'agir soit d'une phrase, soit d'un chapitre, identifiés par isChapter.
// Les événements sont ensuite fusionnés et triés par timestamp (ts).
// Le champ order sert de critère de tri stable en cas d'égalité de timestamps.
type event struct {
	ts        int64  // Timestamp en millisecondes
	isChapter bool   // Indique s'il s'agit d'un chapitre (true) ou d'une phrase (false)
	text      string // Contenu textuel (titre du chapitre ou phrase)
	order     int    // Critère de tri stable en cas d'égalité de ts
}

// absInt64 retourne la valeur absolue d'un entier 64 bits.
func absInt64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

// ensureSorted trie seulement si la slice n'est pas déjà triée
// vérif (O(n) + tri O(n log n).
func ensureSortedPhrases(phrases []Phrase) {
	if len(phrases) <= 1 {
		return
	}
	if sort.SliceIsSorted(phrases, func(i, j int) bool { return phrases[i].TimestampMs < phrases[j].TimestampMs }) {
		return
	}
	sort.Slice(phrases, func(i, j int) bool { return phrases[i].TimestampMs < phrases[j].TimestampMs })
}

// sortChapters trie seulement si la slice n'est pas déjà triée
func sortChapters(chaps []model.Chapter) {
	if len(chaps) <= 1 {
		return
	}
	sort.Slice(chaps, func(i, j int) bool {
		return chaps[i].Start.Milliseconds() < chaps[j].Start.Milliseconds()
	})
}

// splitChapters : sépare en before / middle / after (respecte l'ordre d'entrée)
func splitChapters(chaps []model.Chapter, firstPhraseTs, lastPhraseTs int64) (before, middle, after []model.Chapter) {
	for _, c := range chaps {
		ts := c.Start.Milliseconds()
		switch {
		case ts <= firstPhraseTs:
			before = append(before, c)
		case ts > lastPhraseTs:
			after = append(after, c)
		default:
			middle = append(middle, c)
		}
	}
	return
}

// nearestPhraseIndex : recherche binaire pour trouver l'index de la phrase la plus proche.
// Retourne index (0..len(phrases)-1) et la distance abs en ms.
func nearestPhraseIndex(phrases []Phrase, chTs int64) (int, int64) {
	// maxInt64 : valeur arbitrairement grande utilisée comme distance maximale initiale.
	const maxInt64 = int64(1<<62 - 1)

	// cas trivial
	n := len(phrases)
	if n == 0 {
		return -1, maxInt64
	}

	// Utilise slices.BinarySearchFunc (Go 1.21+). idx est le point d'insertion si found == false.
	idx, found := slices.BinarySearchFunc(phrases, chTs, func(p Phrase, key int64) int {
		if p.TimestampMs < key {
			return -1
		}
		if p.TimestampMs > key {
			return 1
		}
		return 0
	})

	if found {
		return idx, 0
	}

	nearest := -1
	minDist := maxInt64

	// voisin de droite = idx (s'il existe)
	if idx < n {
		d := absInt64(phrases[idx].TimestampMs - chTs)
		if d < minDist {
			minDist = d
			nearest = idx
		}
	}
	// voisin de gauche = idx-1 (s'il existe)
	if idx-1 >= 0 {
		d := absInt64(chTs - phrases[idx-1].TimestampMs)
		if d < minDist {
			minDist = d
			nearest = idx - 1
		}
	}
	return nearest, minDist
}

// adjustMiddleChapters : pour chaque chapter dans middle, trouve nearest phrase et éventuellement "nudge"
// si dist <= thresholdMs (threshold==0 => toujours nudge).
// Retourne une slice de "events" ready-to-merge (chapters with adjusted timestamps).
func adjustMiddleChapters(middle []model.Chapter, phrases []Phrase, thresholdMs int64, baseOrder int) []event {
	events := make([]event, 0, len(middle))
	for i, ch := range middle {
		chMs := ch.Start.Milliseconds()
		idx, dist := nearestPhraseIndex(phrases, chMs)
		adjusted := chMs
		if idx >= 0 && (thresholdMs == 0 || dist <= thresholdMs) {
			// nudge juste avant la phrase la plus proche
			target := phrases[idx].TimestampMs
			if target > 0 {
				adjusted = target - 1
			} else {
				adjusted = 0
			}
		}
		events = append(events, event{
			ts:        adjusted,
			isChapter: true,
			text:      ch.Title,
			order:     baseOrder + i,
		})
	}
	return events
}

// buildEventsFromPhrases : transforme les phrases en events (isChapter=false)
func buildEventsFromPhrases(phrases []Phrase, baseOrder int) []event {
	ev := make([]event, 0, len(phrases))
	for i, p := range phrases {
		ev = append(ev, event{
			ts:        p.TimestampMs,
			isChapter: false,
			text:      p.Text,
			order:     baseOrder + i,
		})
	}
	return ev
}

type OutputTextLayout int

const (
	asPlain OutputTextLayout = iota
	asCollapsed
)

// mergeAndRender : tri des events et rendu final en string.
// request : asCollapsed -> version "collapsed by chapter", asPlain -> classique
func mergeAndRender(ev []event, request OutputTextLayout) string {
	// tri stable
	sort.SliceStable(ev, func(i, j int) bool {
		if ev[i].ts != ev[j].ts {
			return ev[i].ts < ev[j].ts
		}
		if ev[i].isChapter != ev[j].isChapter {
			return ev[i].isChapter
		}
		return ev[i].order < ev[j].order
	})

	var b strings.Builder
	inContent := false // true si on écrit une ligne de contenu

	// réglages par mode
	var phraseSep byte    // séparateur entre phrases
	var chapterSep string // séparation avant ET après la ligne du chapitre

	if request == asCollapsed {
		phraseSep = ' '
		chapterSep = "\n" // quitte la ligne de contenu puis ligne du chapitre puis retour
	} else {
		phraseSep = '\n'
		chapterSep = "\n\n" // paragraphe autour du chapitre
	}

	for _, e := range ev {
		if e.isChapter {
			// normaliser le titre : enlever # et espaces initiaux
			title := strings.TrimSpace(strings.TrimLeft(e.text, "# "))

			// si on était en train d'écrire du contenu, écrire la séparation avant le chapitre
			if inContent {
				b.WriteString(chapterSep)
				inContent = false
			}
			// écrire la ligne chapitre
			b.WriteString("## ")
			b.WriteString(title)
			b.WriteString(chapterSep)

			// après le chapitre, on n'a pas encore commencé la ligne de contenu suivante
			inContent = false
			continue
		}

		// phrase
		text := strings.TrimSpace(e.text)
		if text == "" {
			continue
		}
		if !inContent {
			// début d'une ligne / fragment de contenu
			b.WriteString(text)
			inContent = true
		} else {
			// on colle la phrase (séparateur selon mode)
			b.WriteByte(phraseSep)
			b.WriteString(text)
		}
	}

	// normaliser la fin : s'assurer d'un seul newline final
	out := strings.TrimRight(b.String(), " \t\n\r")
	out = out + "\n"
	return out
}

// transcriptWithChaptersSplit : implémentation méthode plain + chapitres insérés
// thresholdMs: seuil de distance en millisecondes
// thresholdMs == 0 => on colle toujours un chapitre sur la phrase la plus proche.
func (t Transcript) transcriptWithChaptersSplit(thresholdMs int64, request OutputTextLayout) string {
	// copies
	phrases := make([]Phrase, len(t.Phrases))
	copy(phrases, t.Phrases)
	chaps := make([]model.Chapter, len(t.Chapters))
	copy(chaps, t.Chapters)

	ensureSortedPhrases(phrases)
	sortChapters(chaps)

	if len(phrases) == 0 {
		// simple : renvoyer chapitres (formatés)
		var b strings.Builder
		for _, c := range chaps {
			b.WriteString(c.Title)
		}
		return b.String()
	}

	firstTs := phrases[0].TimestampMs
	lastTs := phrases[len(phrases)-1].TimestampMs

	before, middle, after := splitChapters(chaps, firstTs, lastTs)

	// On construit events:
	events := make([]event, 0, len(phrases)+len(chaps))
	// chaps_before : insérer tels quels (on convertit en events)
	for i, c := range before {
		events = append(events, event{
			ts:        c.Start.Milliseconds(),
			isChapter: true,
			text:      c.Title,
			order:     i, // garde l'ordre relatif
		})
	}
	// phrases -> events en leur donnant un ordre de base
	// situé après les events déjà présents dans "before".
	baseOrderPhrases := len(before)
	newPhraseEvents := buildEventsFromPhrases(phrases, baseOrderPhrases)
	events = append(events, newPhraseEvents...)

	// ajuste les chapitres "middle" (leur baseOrder vient après les phrases)
	baseOrderMiddle := baseOrderPhrases + len(phrases)
	adjusted := adjustMiddleChapters(middle, phrases, thresholdMs, baseOrderMiddle)
	events = append(events, adjusted...)

	// chaps_after : conservés tels quels, ordonnés après tous les autres events
	baseOrderAfter := baseOrderMiddle + len(middle)
	for i, c := range after {
		events = append(events, event{
			ts:        c.Start.Milliseconds(),
			isChapter: true,
			text:      c.Title,
			order:     baseOrderAfter + i,
		})
	}

	// merge, sort and render
	return mergeAndRender(events, request)
}
