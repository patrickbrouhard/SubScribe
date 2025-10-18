package obsidian

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"text/template"

	"github.com/patrickprogramme/subscribe/pkg/model"
)

// TODO: ligne 54 : remplacer par un filtre,
// par exemple tous les fichiers qui commencent par "obsidian"

// Renderer gère parsing paresseux (lazy) des templates et fournit des méthodes de rendu.
type Renderer struct {
	templates *template.Template // templates parsés
	fsys      fs.FS              // source des templates (embed.FS ou os.DirFS)
	patterns  []string           // patterns relatifs au fsys, ex: "assets/templates/*.tmpl"
	once      sync.Once          // protège l'initialisation paresseuse
	err       error              // mémorise l'erreur d'initialisation (utile avec once)
}

// NewRendererFromFS construit un Renderer configuré pour parser ultérieurement les patterns
// fournis depuis le fsys (ne parse pas immédiatement).
func NewRendererFromFS(fsys fs.FS, patterns []string) (*Renderer, error) {
	if fsys == nil {
		return nil, fmt.Errorf("fsys est nil")
	}
	if len(patterns) == 0 {
		return nil, fmt.Errorf("aucun template fourni")
	}
	// copy patterns pour sécurité
	cp := append([]string(nil), patterns...)
	return &Renderer{
		fsys:     fsys,
		patterns: cp,
	}, nil
}

// DefaultRenderer construit un Renderer parse tout de suite.
func DefaultRenderer(exePath string) (*Renderer, error) {
	binDir := filepath.Dir(exePath)
	tplDir := filepath.Join(binDir, "templates")

	// Lire les templates depuis le dossier à côté du binaire
	fsys := os.DirFS(tplDir)

	r, err := NewRendererFromFS(fsys, []string{"obsidian_note.md.tmpl"})
	if err != nil {
		return nil, err
	}
	if err := r.ParseNow(); err != nil {
		return nil, err
	}
	return r, nil
}

// parseTemplates effectue le parsing des templates une seule fois (sync.Once).
func (r *Renderer) parseTemplates() error {
	r.once.Do(func() {
		t := template.New("root").Funcs(baseFuncMap())
		var lastErr error
		for _, p := range r.patterns {
			var parseErr error
			t, parseErr = t.ParseFS(r.fsys, p)
			if parseErr != nil {
				lastErr = fmt.Errorf("parse pattern %q: %w", p, parseErr)
				// stoppe ici : il est préférable de remonter l'erreur immédiatement
				break
			}
		}
		if lastErr != nil {
			r.err = lastErr
			return
		}
		r.templates = t
	})
	return r.err
}

// ParseNow force l'initialisation / parsing immédiat et retourne l'erreur si problème.
func (r *Renderer) ParseNow() error {
	if r == nil {
		return fmt.Errorf("nil renderer")
	}
	return r.parseTemplates()
}

// Render exécute le template nommé tmplName (basename du fichier .tmpl) avec data.
// Assure le parsing paresseux avant exécution.
func (r *Renderer) Render(tmplName string, data NoteData) ([]byte, error) {
	if r == nil {
		return nil, fmt.Errorf("renderer is nil")
	}
	if err := r.parseTemplates(); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := r.templates.ExecuteTemplate(&buf, tmplName, data); err != nil {
		return nil, fmt.Errorf("execute template %s: %w", tmplName, err)
	}
	return buf.Bytes(), nil
}

// TemplateNames retourne la liste des noms (basenames) des templates parsés.
// Si le parsing n'a pas encore été fait, renvoie une liste heuristique basée sur patterns.
func (r *Renderer) TemplateNames() []string {
	if r == nil {
		return nil
	}
	if r.templates == nil {
		// pas encore parsé -> retourner les basenames des patterns pour indice
		out := make([]string, 0, len(r.patterns))
		for _, p := range r.patterns {
			out = append(out, filepath.Base(p))
		}
		return out
	}
	names := make([]string, 0, len(r.templates.Templates()))
	for _, t := range r.templates.Templates() {
		if n := t.Name(); n != "" {
			names = append(names, n)
		}
	}
	return names
}

// baseFuncMap construit la liste des fonctions exposées aux templates.
// On expose yamlList (par défaut en block YAML), et aussi yamlListInline si on veut l'inline.
// On expose markdownList pour des listes visibles dans le corps, et warning/quote pour les callouts.
func baseFuncMap() template.FuncMap {
	return template.FuncMap{
		// YAML helpers
		"yamlList":       yamlListBlock,  // défaut : block YAML (lisible dans le frontmatter)
		"yamlListInline": yamlListInline, // explicit inline (["a","b"])

		// Markdown helpers
		"markdownList": markdownListPure, // pour afficher - item dans le corps

		// Hashtags / quotes
		"joinHashtags": joinHashtagsPure,
		"quoteBlock":   quoteBlockPure,

		// Callouts
		"warning": warningFunc,
		"quote":   quoteFunc,

		// Chapters formatter : usage {{ formatChapters .Chapters .URL }}
		"formatChapters": func(chs []model.Chapter, baseURL string) string {
			return formatChaptersPure(chs, baseURL)
		},
	}
}
