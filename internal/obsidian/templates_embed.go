package obsidian

import "embed"

//go:embed assets/templates/*.tmpl
var embeddedTemplates embed.FS

func GetTemplates() embed.FS {
	return embeddedTemplates
}
