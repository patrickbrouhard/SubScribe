package clipboard

import (
	"errors"

	"github.com/atotto/clipboard"
)

// ReadAll lit le contenu texte du presse-papier.
// Retourne une chaîne de caractères et une erreur éventuelle.
func ReadAll() (string, error) {
	text, err := clipboard.ReadAll()
	if err != nil {
		return "", err
	}
	return text, nil
}

// WriteAll écrit une chaîne de caractères dans le presse-papier.
// Retourne une erreur si l'opération échoue.
func WriteAll(text string) error {
	if text == "" {
		return errors.New("le texte à copier ne peut pas être vide")
	}
	return clipboard.WriteAll(text)
}

// ClipboardEquals vérifie si le contenu actuel du presse-papier
// est strictement égal à la chaîne passée en paramètre.
// Retourne true si les deux sont identiques, false sinon.
// En cas d'erreur de lecture, retourne false et ignore l'erreur silencieusement.
// A utiliser pour tester si le presse-papier est vide, et aussi s'il a changé
func ClipboardEquals(text string) bool {
	current, err := clipboard.ReadAll()
	if err != nil {
		return false
	}
	return current == text
}
