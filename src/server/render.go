// Package server - Server-side template rendering with Go html/template.
// See AI.md PART 16: "Server-side rendering (Go templates)".
// All templates are embedded via Go embed.
package server

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"sync"
)

// TemplateRenderer renders embedded Go templates
type TemplateRenderer struct {
	templates map[string]*template.Template
	mu        sync.RWMutex
}

// NewTemplateRenderer parses all embedded templates
func NewTemplateRenderer() *TemplateRenderer {
	r := &TemplateRenderer{
		templates: make(map[string]*template.Template),
	}
	r.parseTemplates()
	return r
}

func (r *TemplateRenderer) parseTemplates() {
	templateFiles := []string{
		"template/index.html",
		"template/admin_login.html",
		"template/admin_setup.html",
		"template/admin_dashboard.html",
	}

	for _, name := range templateFiles {
		data, err := TemplateFS.ReadFile(name)
		if err != nil {
			log.Printf("Warning: template %s not found: %v", name, err)
			continue
		}

		tmpl, err := template.New(name).Parse(string(data))
		if err != nil {
			log.Printf("Warning: template %s parse error: %v", name, err)
			continue
		}

		r.templates[name] = tmpl
	}
}

// Render renders a template with the given data
func (r *TemplateRenderer) Render(w io.Writer, name string, data interface{}) error {
	r.mu.RLock()
	tmpl, ok := r.templates[name]
	r.mu.RUnlock()

	if !ok {
		return fmt.Errorf("template %q not found", name)
	}

	return tmpl.Execute(w, data)
}

// HasTemplate checks if a template exists
func (r *TemplateRenderer) HasTemplate(name string) bool {
	r.mu.RLock()
	_, ok := r.templates[name]
	r.mu.RUnlock()
	return ok
}
