package gonest

import (
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

// TemplateEngine renders HTML templates for MVC-style responses.
type TemplateEngine struct {
	mu        sync.RWMutex
	templates map[string]*template.Template
	viewsDir  string
	extension string
	funcMap   template.FuncMap
}

// NewTemplateEngine creates a template engine that loads templates from viewsDir.
func NewTemplateEngine(viewsDir string) *TemplateEngine {
	return &TemplateEngine{
		templates: make(map[string]*template.Template),
		viewsDir:  viewsDir,
		extension: ".html",
		funcMap:   template.FuncMap{},
	}
}

// SetExtension sets the template file extension (default: ".html").
func (e *TemplateEngine) SetExtension(ext string) {
	e.extension = ext
}

// AddFunc adds a template function.
func (e *TemplateEngine) AddFunc(name string, fn any) {
	e.funcMap[name] = fn
}

// Render renders a template with the given data and writes to the response.
func (e *TemplateEngine) Render(w http.ResponseWriter, name string, data any) error {
	tmpl, err := e.getTemplate(name)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return tmpl.Execute(w, data)
}

func (e *TemplateEngine) getTemplate(name string) (*template.Template, error) {
	e.mu.RLock()
	tmpl, ok := e.templates[name]
	e.mu.RUnlock()
	if ok {
		return tmpl, nil
	}

	path := filepath.Join(e.viewsDir, name+e.extension)
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, NewInternalServerError("template error: " + err.Error())
	}
	tmpl, err = template.New(name).Funcs(e.funcMap).Parse(string(content))
	if err != nil {
		return nil, NewInternalServerError("template error: " + err.Error())
	}

	e.mu.Lock()
	e.templates[name] = tmpl
	e.mu.Unlock()

	return tmpl, nil
}

// RenderHandler creates a handler that renders a template.
func RenderHandler(engine *TemplateEngine, name string, dataFn func(ctx Context) any) HandlerFunc {
	return func(ctx Context) error {
		data := dataFn(ctx)
		return engine.Render(ctx.ResponseWriter(), name, data)
	}
}

// StaticFiles serves static files from a directory.
func StaticFiles(prefix, dir string) HandlerFunc {
	fs := http.StripPrefix(prefix, http.FileServer(http.Dir(dir)))
	return func(ctx Context) error {
		fs.ServeHTTP(ctx.ResponseWriter(), ctx.Request())
		return nil
	}
}
