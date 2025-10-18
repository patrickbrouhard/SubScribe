package subtitles

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// transform_auto.go : traitement dédié aux captions (sous-titres automatiques).
// Le format ASR fournit des timestamps par mot (tOffsetMs + tStartMs). Ici on
// reconstruit des phrases lisibles en se basant sur ces timestamps, les
// ponctuations et les pauses.
//
// Vérifie : cette fonction suppose que les helpers suivants existent dans le
// package : calculateAbsTime, splitSegIntoSubSentences, trimTrailingClosers,
// lastNonSpaceRune, isSentenceTerminatorRune, normalizeWhitespace.

const (
	// seuil pour couper une phrase quand la pause entre deux mots est trop longue
	pauseThresholdMs = 2000
	// sécurité : nombre maximum de mots par phrase
	maxWordsPerPhrase = 100
)

// TransformAutoRawToPhrases transforme rawJSON3 (captions ASR) en []Phrase.
// Version simplifiée : on considère chaque seg comme une unité atomique.
// On n'essaie PAS de découper un seg en plusieurs sous-phrases même si
// celui-ci contient plusieurs terminators (cas extrêmement rare en ASR).
func TransformAutoRawToPhrases(raw rawJSON3) ([]Phrase, error) {
	var phrases []Phrase
	if len(raw.Events) == 0 {
		return phrases, nil
	}

	var (
		currentSb      strings.Builder      // accumulateur de la phrase courante
		currentStartMs int64           = -1 // timestamp du premier mot de la phrase en cours
		lastWordTs     int64           = -1 // timestamp du dernier mot vu (pour pause)
	)

	commit := func() {
		txt := strings.TrimSpace(currentSb.String())
		if txt == "" {
			currentSb.Reset()
			currentStartMs = -1
			return
		}
		// par défaut on met le timestamp du premier mot si disponible,
		// sinon on utilise lastWordTs (fallback)
		ts := lastWordTs
		if currentStartMs >= 0 {
			ts = currentStartMs
		}
		p := Phrase{
			TimestampMs: ts,
			Text:        txt,
		}
		p.RuneCount = utf8.RuneCountInString(strings.TrimSpace(p.Text))
		p.WordCount = len(strings.FieldsFunc(p.Text, unicode.IsSpace))

		phrases = append(phrases, p)

		// reset pour la phrase suivante
		currentSb.Reset()
		currentStartMs = -1
		// lastWordTs reste inchangé (utile comme fallback)
	}

	appendSegmentText := func(segText string) {
		segText = normalizeWhitespace(segText)
		if segText == "" {
			return
		}
		if currentSb.Len() == 0 {
			currentSb.WriteString(segText)
		} else {
			// garder un espace entre fragments
			currentSb.WriteByte(' ')
			currentSb.WriteString(strings.TrimSpace(segText))
		}
	}

	// boucle principale sur Events, voir raw_types.go
	for _, ev := range raw.Events {
		// pas de segs, pas de mots = skip
		if ev.IsNewlineOnly() {
			continue
		}

		// parcourir les segs (chaque seg est traité comme une unité)
		for _, seg := range ev.Segs {
			s := seg.Utf8
			s = strings.ReplaceAll(s, "\\n", "\n")
			// ignorer segs vides ou contenant uniquement des espaces / \n
			if strings.TrimSpace(s) == "" || s == "\n" {
				continue
			}

			// calcul du timestamp absolu pour ce seg (mot)
			ts := calculateAbsTime(ev, seg)
			if ts > 0 {
				// si pause plus longue que le seuil, on coupe la phrase en cours
				if lastWordTs >= 0 && (ts-lastWordTs) > pauseThresholdMs && currentSb.Len() > 0 {
					commit()
				}
				// mettre à jour lastWordTs et initialiser currentStartMs si premier mot de la phrase
				lastWordTs = ts
				if currentStartMs < 0 {
					currentStartMs = ts
				}
			}

			appendSegmentText(s)

			// sécurité : limiter la longueur en nombre de mots
			if len(strings.Fields(currentSb.String())) >= maxWordsPerPhrase {
				commit()
				continue
			}

			// décision de commit basée uniquement sur le dernier rune du seg
			trimmed := trimTrailingClosers(s)         // enlève quotes/closers en fin
			lastRune, ok := lastNonSpaceRune(trimmed) // rune utile finale
			if ok && isSentenceTerminatorRune(lastRune) {
				// si le seg se termine par un terminator, on commit la phrase
				commit()
				continue
			}
			// sinon on continue d'accumuler
		}
	}

	// flush final
	commit()
	return phrases, nil
}
