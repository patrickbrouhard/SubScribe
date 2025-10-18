package subtitles

import (
	"math"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

type SplitPiece struct {
	Text              string
	EndRune           int  // exclusive: nombre de runes consommées depuis le début du segment
	EndWithTerminator bool // true si la pièce se termine par . ! ou ?
}

var reMultiSpace = regexp.MustCompile(`\s+`)

// EventText : assemble et nettoie le texte d'un event (ré-usage de cleanSeg)
func EventText(ev rawEvent) string {
	var parts []string
	for _, seg := range ev.Segs {
		txt := cleanSeg(seg.Utf8)
		if txt == "" {
			continue
		}
		parts = append(parts, txt)
	}
	return strings.Join(parts, " ")
}

// cleanSeg normalise un seg : convertit les "\n" et "\\n" en espaces,
// remplace les séquences d'espaces par un seul espace, et trim.
func cleanSeg(s string) string {
	// remplacer les séquences d'échappement de newline
	s = strings.ReplaceAll(s, "\\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	// normaliser les espaces (tabs, multiples espaces, etc.)
	s = reMultiSpace.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

// splitSegOffsetsString : même rôle que splitSegOffsets([]rune) mais sans allocation de []rune.
// Retourne []SplitPiece avec EndRune compté en runes (pas d'octets).
func splitSegOffsetsString(s string) []SplitPiece {
	if s == "" {
		return nil
	}

	var out []SplitPiece
	var sb strings.Builder            // accumulateur principal pour le texte courant
	var terminatorBuf strings.Builder // buffer temporaire pour terminator + closers
	candidat := false

	idx := 0       // offset byte dans la string
	runeIndex := 0 // nombre de runes déjà consommées depuis le début de la string

	for idx < len(s) {
		r, size := utf8.DecodeRuneInString(s[idx:])
		if r == utf8.RuneError && size == 1 {
			// octet invalide : on l'ignore
			idx++
			continue
		}

		// si on est déjà en mode candidat (on a des terminators en attente),
		// on regarde le type de rune actuel pour décider de la suite.
		if candidat {
			// si la rune courante est encore un terminator ou un closer -> accumuler
			if isSentenceTerminatorRune(r) || isCloserRune(r) {
				terminatorBuf.WriteRune(r)
				idx += size
				runeIndex++
				continue
			}

			// si la rune est un espace -> la phrase est terminée ; consommer tous les espaces
			if unicode.IsSpace(r) {
				// consommer tous les espaces suivants et compter leur nombre en runes
				j := idx
				spaceCount := 0
				for j < len(s) {
					r2, size2 := utf8.DecodeRuneInString(s[j:])
					if r2 == utf8.RuneError && size2 == 1 {
						// ignorer l'octet invalide en avançant d'un
						j++
						continue
					}
					if !unicode.IsSpace(r2) {
						break
					}
					spaceCount++
					j += size2
				}
				// incrémente le compteur de runes consommées pour inclure les espaces
				runeIndex += spaceCount
				// construire la pièce commitée
				combined := sb.String() + terminatorBuf.String()
				text := normalizeWhitespace(combined)
				if strings.TrimSpace(text) != "" {
					out = append(out, SplitPiece{
						Text:              text,
						EndRune:           runeIndex,
						EndWithTerminator: terminatorBuf.Len() > 0,
					})
				}
				// reset et avancer après les espaces
				sb.Reset()
				terminatorBuf.Reset()
				candidat = false
				idx = j
				continue
			}

			// sinon : faux-positif (ex: 2.6) -> rattacher terminatorBuf au texte et re-traiter rune courante
			sb.WriteString(terminatorBuf.String())
			terminatorBuf.Reset()
			candidat = false
			// NB: on ne modifie pas idx/runeIndex ici pour re-traiter la rune courante dans la branche non-candidat
			continue
		}

		// cas normal (pas de candidat en attente)
		if isSentenceTerminatorRune(r) {
			terminatorBuf.WriteRune(r)
			idx += size
			runeIndex++
			candidat = true
			continue
		}

		// rune ordinaire
		sb.WriteRune(r)
		idx += size
		runeIndex++
	}

	// vidage final
	if candidat && terminatorBuf.Len() > 0 {
		combined := sb.String() + terminatorBuf.String()
		text := normalizeWhitespace(combined)
		if strings.TrimSpace(text) != "" {
			out = append(out, SplitPiece{
				Text:              text,
				EndRune:           runeIndex,
				EndWithTerminator: true,
			})
		}
	} else if sb.Len() > 0 {
		text := normalizeWhitespace(sb.String())
		if strings.TrimSpace(text) != "" {
			out = append(out, SplitPiece{
				Text:              text,
				EndRune:           runeIndex,
				EndWithTerminator: false,
			})
		}
	}

	return out
}

// TransformManualRawToPhrases : pipeline complet -> []Phrase
// - découpe chaque event en SplitPiece (via splitSegOffsets sur le texte complet de l'event)
// - calcule perRuneMs à partir de event.Duration / #runes
// - construit et finalise des phrases qui peuvent traverser plusieurs events
func TransformManualRawToPhrases(raw rawJSON3) ([]Phrase, error) {
	var out []Phrase

	var currBuilder strings.Builder
	var currStartMs int64 = -1 // timestamp de début de la phrase en construction; -1 = unset

	for _, ev := range raw.Events {
		// récupérer start/duration de l'event (sécurisé sur pointeurs)
		var evStart, evDur int64
		if ev.TStartMs != nil {
			evStart = *ev.TStartMs
		}
		if ev.DDurationMs != nil {
			evDur = *ev.DDurationMs
		}

		evText := EventText(ev)
		if strings.TrimSpace(evText) == "" {
			// rien à faire pour cet event
			continue
		}

		// découper le texte de l'event en pieces (avec EndRune)
		pieces := splitSegOffsetsString(evText)

		// fallback si splitSegOffsets renvoie nil (on garde l'event entier)
		if len(pieces) == 0 {
			// tout l'event comme une seule piece
			pieces = []SplitPiece{{
				Text:              normalizeWhitespace(evText),
				EndRune:           utf8.RuneCountInString(evText),
				EndWithTerminator: false,
			}}

		}

		// per-rune ms pour cet event
		runesInEvent := utf8.RuneCountInString(evText)
		var perRuneMs float64
		if runesInEvent > 0 && evDur > 0 {
			perRuneMs = float64(evDur) / float64(runesInEvent)
		} else {
			perRuneMs = 0
		}

		prevEnd := 0 // nombre de runes consommées avant la piece courante (dans l'event)
		for _, p := range pieces {
			startRuneInEvent := prevEnd
			// startMs estimé pour cette piece (si besoin)
			pieceStartMs := evStart + int64(math.Round(perRuneMs*float64(startRuneInEvent)))

			// si aucune phrase en cours, on initialise son timestamp au début de cette piece
			if currBuilder.Len() == 0 {
				currStartMs = pieceStartMs
			}

			// fusionner la piece au builder courant (avec un espace si nécessaire)
			if currBuilder.Len() > 0 {
				currBuilder.WriteString(" ")
			}
			currBuilder.WriteString(p.Text)

			if p.EndWithTerminator {
				// finaliser la phrase courante
				text := normalizeWhitespace(currBuilder.String())
				if strings.TrimSpace(text) != "" {
					rc := utf8.RuneCountInString(text)
					wc := len(strings.Fields(text))
					out = append(out, Phrase{
						TimestampMs: currStartMs,
						Text:        text,
						RuneCount:   rc,
						WordCount:   wc,
					})
				}
				// reset builder (la phrase suivante commencera après cette piece)
				currBuilder.Reset()
				currStartMs = -1
			}
			// avancer prevEnd
			prevEnd = p.EndRune
		}
	}

	// flush final si reste non terminé
	if strings.TrimSpace(currBuilder.String()) != "" {
		// si currStartMs n'a jamais été initialisé, on le met à 0 (ou la moindre start connue) ;
		// en pratique currStartMs devrait toujours avoir été set dès la 1ère piece, mais on protège.
		if currStartMs == -1 {
			currStartMs = 0
		}
		text := normalizeWhitespace(currBuilder.String())
		rc := utf8.RuneCountInString(text)
		wc := len(strings.Fields(text))
		out = append(out, Phrase{
			TimestampMs: currStartMs,
			Text:        text,
			RuneCount:   rc,
			WordCount:   wc,
		})
	}

	return out, nil
}
