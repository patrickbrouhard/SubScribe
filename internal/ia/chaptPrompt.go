package ia

import (
	"fmt"

	"github.com/patrickprogramme/subscribe/internal/assets"
)

func GetChatPrompt() ([]byte, error) {
	// récupération du chemin dans l'embed
	tplPath := assets.TemplateByName["ai_prompt"]
	if tplPath == "" {
		return nil, fmt.Errorf("template ai_prompt introuvable dans assets.TemplateByName")
	}
	b, err := assets.Embedded.ReadFile(tplPath)
	if err != nil {
		return nil, fmt.Errorf("lecture template embarqué %s: %w", tplPath, err)
	}
	return b, nil
}

// func PromptSplit(t subtitles.Transcript)

// func BuildManifest(p string) {}
