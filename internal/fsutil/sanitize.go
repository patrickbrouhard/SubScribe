package fsutil

import (
	"regexp"
	"strings"
	"unicode"
)

// limite de longueur de la chaine
const max = 200

// invalidFileRunes définit les caractères interdits dans les noms de fichiers
// \x00-\x1F sont les caractères de contrôle
var invalidFileRunes = regexp.MustCompile(`[<>"/\\|?*\x00-\x1F]`)

// multiSpace détecte les séquences de plusieurs espaces pour les réduire à un seul.
var multiSpace = regexp.MustCompile(`\s+`)

// SanitizeFilename nettoie une chaîne de caractères pour en faire un nom de fichier valide.
// Étapes :
// - Remplace ":" par "-" explicitement
// - Remplace les autres caractères interdits par "_"
// - Supprime les espaces superflus
// - Limite la longueur du nom
// - Fournit un nom par défaut si la chaîne est vide
func SanitizeFilename(name string) string {
	if name == "" {
		return "untitled"
	}

	// Remplacement de ":" par "-"
	name = strings.ReplaceAll(name, ":", "-")

	// Remplacement des autres caractères interdits par " "
	clean := invalidFileRunes.ReplaceAllString(name, " ")

	// Suppression des espaces en début/fin
	clean = strings.TrimSpace(clean)

	// Réduction des espaces multiples à un seul espace
	clean = multiSpace.ReplaceAllString(clean, " ")

	// Suppression des points terminaux (un ou plusieurs)
	clean = strings.TrimRight(clean, ".")

	if clean == "" {
		return "untitled"
	}

	if len(clean) > max {
		clean = clean[:max]
	}

	return CapitalizeFirst(clean)
}

// CapitalizeFirst met en majuscule le premier caractère (rune) de s.
// Ne touche pas au reste de la chaîne. Vide -> retourne "".
func CapitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	rs := []rune(s)
	rs[0] = unicode.ToUpper(rs[0])
	return string(rs)
}
