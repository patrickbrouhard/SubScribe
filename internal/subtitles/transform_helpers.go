package subtitles

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// calculateAbsTime calcule tStartMs (absolu) + tOffsetMs (offset) si disponible
// sinon retourne base (tStartMs) ou 0.
func calculateAbsTime(ev rawEvent, seg rawSeg) int64 {
	var base int64 = 0
	if ev.TStartMs != nil {
		base = *ev.TStartMs
	}
	if seg.TOffsetMs != nil {
		return base + *seg.TOffsetMs
	}
	return base
}

func isSentenceTerminatorRune(r rune) bool {
	return r == '.' || r == '!' || r == '?'
}

func isCloserRune(r rune) bool {
	switch r {
	case '"', '\'', '”', '’', ')', ']', '}', '»':
		return true
	}
	return false
}

// trimTrailingClosers enlève guillemets/parenthèses fermantes
// accolées à la fin qui masquent un terminator
func trimTrailingClosers(s string) string {
	for {
		s = strings.TrimRightFunc(s, unicode.IsSpace) // Supprime espaces à la fin de la chaîne
		if s == "" {
			return s
		}
		r, size := utf8.DecodeLastRuneInString(s)
		if r == utf8.RuneError && size == 1 { // Si dernier caractère est invalide...
			s = s[:len(s)-1] // ...on supprime le dernier byte et on continue
			continue
		}
		if isCloserRune(r) {
			// remove last rune and continue trimming
			s = s[:len(s)-size]
			continue
		}
		break
	}
	return s
}

// lastNonSpaceRune returns the last rune that is not a whitespace, and true if found.
func lastNonSpaceRune(s string) (rune, bool) {
	for len(s) > 0 {
		r, size := utf8.DecodeLastRuneInString(s)
		if r == utf8.RuneError && size == 1 {
			s = s[:len(s)-1] // c'est un octet invalide, drop + continue
			continue
		}
		if !unicode.IsSpace(r) {
			return r, true
		}
		s = s[:len(s)-size]
	}
	return 0, false
}

// normalizeWhitespace nettoie les espace : un seul espace entre mots, aucun en début/fin
func normalizeWhitespace(s string) string {
	return strings.TrimSpace(strings.Join(strings.Fields(s), " "))
}
