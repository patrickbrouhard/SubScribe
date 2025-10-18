package assets

import "embed"

//go:embed subscribe.example.yaml
//go:embed templates/*tmpl
var Embedded embed.FS

// Nom de l'asset de config par défaut (chemin DANS Embedded)
const DefaultConfigAsset = "subscribe.example.yaml"

// DefaultTemplatePaths : liste ordonnée des templates "par défaut" embarqués.
// Ce sont des chemins relatifs DANS Embedded (ex: "templates/obsidian_note.md.tmpl").
var DefaultTemplatePaths = []string{
	"templates/obsidian_note.md.tmpl",
	"templates/prompt_for_ai.txt.tmpl",
}

// TemplateByName donne un accès par clé (map).
var TemplateByName = map[string]string{
	"obsidian_note": "templates/obsidian_note.md.tmpl",
	"ai_prompt":     "templates/prompt_for_ai.txt.tmpl",
}
