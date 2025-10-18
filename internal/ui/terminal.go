package ui

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/patrickprogramme/subscribe/internal/clipboard"
	"github.com/patrickprogramme/subscribe/internal/yt"
)

type terminalUI struct {
	reader *bufio.Reader
}

func NewTerminal() Interface {
	return &terminalUI{reader: bufio.NewReader(os.Stdin)}
}

// Choix de l'utilisateur retourné par GetAIResponseFromClipboardChoice
const (
	ChoiceUse   = "use"   // utiliser le texte du clipboard
	ChoiceRetry = "retry" // ne pas utiliser et recommencer le processus
	ChoiceSkip  = "skip"  // continuer sans utiliser le texte (générer sans résumé)
)

func (t *terminalUI) GetYtURL(ctx context.Context) (string, error) {
	// 1) clipboard
	if clip, err := clipboard.ReadAll(); err == nil {
		if yt.IsYouTubeURL(clip) {
			t.PrintInfo(ctx, fmt.Sprintf("Utilisation de l'URL depuis le presse-papier: %s", clip))
			return clip, nil
		}
	}
	// 2) prompt
	for {
		fmt.Print("Entrez l'URL d'une vidéo Youtube: ")
		input, _ := t.reader.ReadString('\n')
		url := strings.TrimSpace(input)
		if yt.IsYouTubeURL(url) {
			return url, nil
		}
		fmt.Println("❌ URL invalide. Essayez à nouveau.")
	}
}

func (t *terminalUI) WaitForExit(ctx context.Context) error {
	fmt.Println("\n\nAppuyez sur Ctrl+C pour quitter.")

	// Prépare le canal pour les signaux d'interruption
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	select {
	case <-ctx.Done(): // Context annulé ailleurs
		return ctx.Err()
	case <-sigCh: // Reçu Ctrl+C (SIGINT ou SIGTERM)
		return nil
	}
}

func (t *terminalUI) PrintInfo(ctx context.Context, s string) {
	fmt.Println(s)
}

func (t *terminalUI) PrintError(ctx context.Context, s string) {
	fmt.Fprintln(os.Stderr, s)
}

// GetClipboardChoice propose d'utiliser le texte du presse-papier.
// Retourne (content, choice, err).
// - content : texte provenant du clipboard (vide si choice != ChoiceUse).
// - choice : one of "use", "retry", "skip".
func (t *terminalUI) GetClipboardChoice(ctx context.Context) (string, string, error) {
	// tentative de lecture du clipboard
	clip, err := clipboard.ReadAll()
	if err != nil || strings.TrimSpace(clip) == "" {
		fmt.Println("Le presse-papier est vide ou inaccessible.")
		fmt.Println("Appuyez sur Entrée pour réessayer, ou tapez 's' puis Entrée pour ignorer et continuer sans résumé.")
		input, _ := t.reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))
		if input == "s" {
			return "", ChoiceSkip, nil
		}
		return "", ChoiceRetry, nil
	}

	// affiche un aperçu
	lines := strings.SplitN(clip, "\n", 6)
	preview := strings.Join(lines[:min(len(lines), 5)], "\n")
	fmt.Println("Aperçu du presse-papier :")
	fmt.Println("────────────────────────")
	fmt.Println(preview)
	if len(strings.Split(clip, "\n")) > 5 {
		fmt.Println("...")
	}
	fmt.Println("────────────────────────")
	fmt.Print("(o) Utiliser ce texte  (n) Réessayer  (s) Ignorer et continuer sans résumé  ? [o/n/s] : ")

	// lecture choix utilisateur (bloquant)
	resp, _ := t.reader.ReadString('\n')
	resp = strings.TrimSpace(strings.ToLower(resp))

	switch resp {
	case "o", "oui", "y", "yes":
		// petite normalisation : retirer BOM éventuel et trim final
		clip = strings.TrimPrefix(clip, "\ufeff")
		clip = strings.ReplaceAll(clip, "\r\n", "\n")
		return clip, ChoiceUse, nil
	case "s":
		return "", ChoiceSkip, nil
	default:
		// par défaut on considère comme retry
		// petit délai avant return pour améliorer UX (optionnel)
		time.Sleep(100 * time.Millisecond)
		return "", ChoiceRetry, nil
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// WaitForUserToCopyResponse attend que l'utilisateur indique qu'il a copié la réponse IA.
// Retourne true si l'utilisateur a choisi d'ignorer/sauter (tape 's'), false sinon.
func (t *terminalUI) WaitForUserToCopyResponse(ctx context.Context) (bool, error) {
	fmt.Println("Ouvrez votre chat IA, collez le prompt, puis copiez la réponse.")
	fmt.Print("Quand vous êtes prêt, appuyez sur Entrée. Tapez 's' puis Entrée pour ignorer et continuer sans résumé : ")
	input, err := t.reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("lecture stdin: %w", err)
	}
	input = strings.TrimSpace(strings.ToLower(input))
	if input == "s" {
		return true, nil
	}
	return false, nil
}

// WaitForClipboardChange poll le presse-papier jusqu'à ce que son contenu
// diffère de `initial` et soit non vide, ou jusqu'au timeout/context done.
// interval : durée entre lectures (ex: 500*time.Millisecond).
// timeout : 0 => attendre indéfiniment (ou utiliser ctx pour annulation).
func (t *terminalUI) WaitForClipboardChange(ctx context.Context, initial string, interval time.Duration, timeout time.Duration) (string, error) {
	normalize := func(s string) string {
		s = strings.TrimPrefix(s, "\ufeff")
		s = strings.ReplaceAll(s, "\r\n", "\n")
		return strings.TrimSpace(s)
	}
	initial = normalize(initial)

	// laisse l'OS opérer le collage si on vient d'écrire le clipboard
	time.Sleep(150 * time.Millisecond)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var deadline <-chan time.Time
	if timeout > 0 {
		d := time.After(timeout)
		deadline = d
	}

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
			current, err := clipboard.ReadAll()
			if err != nil {
				continue
			}
			current = normalize(current)
			if current != "" && current != initial {
				return current, nil
			}
		case <-deadline:
			return "", fmt.Errorf("timeout waiting clipboard change after %v", timeout)
		}
	}
}
