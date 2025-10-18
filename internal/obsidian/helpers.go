package obsidian

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/patrickprogramme/subscribe/pkg/model"
)

// // yamlListPure: représentation inline simple : [a, b]
// func yamlListPure(xs []string) string {
// 	if len(xs) == 0 {
// 		return "[]"
// 	}
// 	return "[" + strings.Join(xs, ", ") + "]"
// }

// yamlListInline transforme: {"apple", "banana", "cherry"} -> ["apple", "banana", "cherry"]
func yamlListInline(xs []string) string {
	if len(xs) == 0 {
		return "[]"
	}
	quoted := make([]string, 0, len(xs))
	for _, s := range xs {
		quoted = append(quoted, strconv.Quote(s))
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

// yamlListBlock retourne une liste
func yamlListBlock(xs []string) string {
	if len(xs) == 0 {
		return " []" // note l'espace: on l'utilise après 'tags:'
	}
	var b strings.Builder
	for _, s := range xs {
		// on quote pour sécurité (c'est valide YAML): - "mon tag"
		quoted := strconv.Quote(s)
		b.WriteString("\n  - ")
		b.WriteString(quoted)
	}
	return b.String()
}

// joinHashtagsPure : ajoute '#' quand il manque et join par espace.
func joinHashtagsPure(xs []string) string {
	if len(xs) == 0 {
		return ""
	}
	out := make([]string, 0, len(xs))
	for _, h := range xs {
		h = strings.TrimSpace(h)
		if h == "" {
			continue
		}
		if strings.HasPrefix(h, "#") {
			out = append(out, h)
		} else {
			out = append(out, "#"+h)
		}
	}
	return strings.Join(out, " ")
}

// quoteBlockPure : préfixe chaque ligne par "> " pour un blockquote Markdown.
func quoteBlockPure(s string) string {
	if s == "" {
		return ""
	}
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i := range lines {
		lines[i] = "> " + lines[i]
	}
	return strings.Join(lines, "\n")
}

// markdownListPure génère des lignes "- item" (avec saut final).
// Usage dans template : {{ markdownList .Categories }}
func markdownListPure(xs []string) string {
	if len(xs) == 0 {
		return ""
	}
	var b strings.Builder
	for _, s := range xs {
		trim := strings.TrimSpace(s)
		if trim == "" {
			continue
		}
		b.WriteString("- ")
		b.WriteString(trim)
		b.WriteString("\n")
	}
	return b.String()
}

// formatChaptersPure : génère les lignes Markdown cliquables.
// Si baseURL est vide, on produit des lignes sans lien.
func formatChaptersPure(chs []model.Chapter, baseURL string) string {
	if len(chs) == 0 {
		return ""
	}
	sep := "?"
	if strings.Contains(baseURL, "?") {
		sep = "&"
	}
	var b strings.Builder
	for _, c := range chs {
		secs := int64(c.Start)
		ts := c.Start.TimestampHHMMSS()
		title := strings.TrimSpace(strings.ReplaceAll(c.Title, "\n", " "))

		if baseURL == "" {
			b.WriteString(fmt.Sprintf("- %s - %s\n", ts, title))
		} else {
			link := fmt.Sprintf("%s%st=%ds", baseURL, sep, secs)
			b.WriteString(fmt.Sprintf("- [%s](%s) - %s\n", ts, link, title))
		}
	}
	return b.String()
}
