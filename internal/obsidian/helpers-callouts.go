package obsidian

import (
	"fmt"
	"strings"
	"unicode"
)

// buildCalloutBase construit l'en-tête du callout.
// kind -> en majuscule dans [!KIND]
// title optionnel : si non-vide, ajouté sur la même ligne que l'en-tête.
func buildCalloutBase(kind, title string) string {
	k := strings.ToUpper(strings.TrimSpace(kind))
	if k == "" {
		k = "NOTE"
	}
	// clean kind : ne garder que lettres, - et _
	var cleanKind []rune
	for _, r := range k {
		if unicode.IsLetter(r) || r == '-' || r == '_' {
			cleanKind = append(cleanKind, r)
		}
	}
	header := fmt.Sprintf("> [!%s]", strings.ToUpper(string(cleanKind)))
	if t := strings.TrimSpace(title); t != "" {
		header = header + " " + t
	}
	return header + "\n"
}

// prefixLines ajoute "> " au début de chaque ligne (callout style).
func prefixLinesWithQuote(content string) string {
	content = strings.TrimRight(content, "\n")
	if content == "" {
		// garder un bloc vide avec une ligne >
		return "> \n"
	}
	lines := strings.Split(content, "\n")
	var b strings.Builder
	for _, L := range lines {
		// on préserve le contenu en trimant les espaces finaux
		L = strings.TrimRight(L, " \t")
		b.WriteString("> ")
		b.WriteString(L)
		b.WriteString("\n")
	}
	return b.String()
}

// ensureQuoted si le texte ne commence pas par guillemet (", « ou ’), on ajoute des " autour.
// utile pour quote callout.
func ensureQuoted(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return `""`
	}
	first := []rune(s)[0]
	if first == '"' || first == '\'' || first == '«' || first == '“' || first == '’' {
		return s
	}
	// on ajoute "..." (on pourrait utiliser « » pour fr, mais " est universel)
	return `"` + s + `"`
}

// warningFunc : usage dans template:
//   - {{ warning .Description }}
//   - {{ warning "Titre court" .Description }}
//
// Accept multiple args: 1 => content only; 2 => title, content.
func warningFunc(args ...interface{}) string {
	var title, content string
	if len(args) == 1 {
		content = fmt.Sprint(args[0])
	} else if len(args) >= 2 {
		title = fmt.Sprint(args[0])
		content = fmt.Sprint(args[1])
	}
	h := buildCalloutBase("warning", title)
	body := prefixLinesWithQuote(content)
	return h + body
}

// quoteFunc : usage:
//   - {{ quote .Quote }}                 -> pas d'auteur
//   - {{ quote .Author .Quote }}         -> auteur + citation
//
// Si le quote ne commence pas par un guillemet, on l'entoure automatiquement.
func quoteFunc(args ...interface{}) string {
	var author, quote string
	if len(args) == 1 {
		quote = fmt.Sprint(args[0])
	} else if len(args) >= 2 {
		author = fmt.Sprint(args[0])
		quote = fmt.Sprint(args[1])
	}
	quote = ensureQuoted(quote)
	h := buildCalloutBase("quote", author)
	body := prefixLinesWithQuote(quote)
	return h + body
}
