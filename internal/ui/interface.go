package ui

import (
	"context"
	"time"
)

type Interface interface {
	// GetYtURL doit renvoyer une URL valide.
	// Implémentation terminale : priorité clipboard -> prompt
	GetYtURL(ctx context.Context) (string, error)

	// WaitForExit bloque jusqu'à ce qu'un signal d'annulation soit reçu via ctx (Ctrl+C).
	WaitForExit(ctx context.Context) error

	PrintInfo(ctx context.Context, s string)
	PrintError(ctx context.Context, s string)

	WaitForUserToCopyResponse(ctx context.Context) (bool, error)
	// GetClipboardChoice interagit avec l'utilisateur et retourne:
	// - content : texte (potentiellement vide si choice != "use")
	// - choice  : "use", "retry" ou "skip"
	// - err     : erreur éventuelle
	GetClipboardChoice(ctx context.Context) (content string, choice string, err error)
	WaitForClipboardChange(ctx context.Context, initial string, interval time.Duration, timeout time.Duration) (string, error)
}
