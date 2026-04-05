package gonest

import (
	"bytes"
	"html/template"
	"io/fs"
	"net/http"
	"path/filepath"
	"sync"
)

// ViewEngine is the interface for template rendering engines.
// Equivalent to NestJS view engine integration (Express/Handlebars/Pug/EJS).
type ViewEngine interface {
	// Render renders a template by name with the given data.
	Render(name string, data any) ([]byte, error)
}

// GoTemplateEngine is a ViewEngine backed by Go's html/template.
type GoTemplateEngine struct {
	mu        sync.RWMutex
	templates *template.Template
	dir       string
	ext       string
	funcMap   template.FuncMap
}

// GoTemplateEngineOptions configures the Go template engine.
type GoTemplateEngineOptions struct {
	// Dir is the directory containing template files.
	Dir string
	// Extension is the file extension for templates (default: ".html").
	Extension string
	// FuncMap provides additional template functions.
	FuncMap template.FuncMap
}

// NewGoTemplateEngine creates a template engine using Go's html/template.
func NewGoTemplateEngine(opts GoTemplateEngineOptions) (*GoTemplateEngine, error) {
	ext := opts.Extension
	if ext == "" {
		ext = ".html"
	}

	funcMap := template.FuncMap{}
	for k, v := range opts.FuncMap {
		funcMap[k] = v
	}

	pattern := filepath.Join(opts.Dir, "*"+ext)
	tmpl, err := template.New("").Funcs(funcMap).ParseGlob(pattern)
	if err != nil {
		return nil, err
	}

	return &GoTemplateEngine{
		templates: tmpl,
		dir:       opts.Dir,
		ext:       ext,
		funcMap:   funcMap,
	}, nil
}

// NewGoTemplateEngineFromFS creates a template engine from an fs.FS.
func NewGoTemplateEngineFromFS(fsys fs.FS, pattern string, funcMap ...template.FuncMap) (*GoTemplateEngine, error) {
	fm := template.FuncMap{}
	if len(funcMap) > 0 {
		for k, v := range funcMap[0] {
			fm[k] = v
		}
	}

	tmpl, err := template.New("").Funcs(fm).ParseFS(fsys, pattern)
	if err != nil {
		return nil, err
	}

	return &GoTemplateEngine{
		templates: tmpl,
		funcMap:   fm,
	}, nil
}

// Render renders a named template with data.
func (e *GoTemplateEngine) Render(name string, data any) ([]byte, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var buf bytes.Buffer
	if err := e.templates.ExecuteTemplate(&buf, name, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Render renders a template and writes it to the response.
// This is a handler-level function equivalent to NestJS @Render() decorator.
//
// Usage:
//
//	r.Get("/", func(ctx gonest.Context) error {
//	    return gonest.Render(ctx, "index.html", map[string]any{
//	        "title": "Home",
//	    })
//	}).SetMetadata("render", "index.html")
func Render(ctx Context, template string, data any) error {
	engine, ok := ctx.Get("__view_engine")
	if !ok {
		return NewInternalServerError("no view engine configured")
	}
	ve, ok := engine.(ViewEngine)
	if !ok {
		return NewInternalServerError("invalid view engine type")
	}

	rendered, err := ve.Render(template, data)
	if err != nil {
		return NewInternalServerError("template render error: " + err.Error())
	}

	w := ctx.ResponseWriter()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, writeErr := w.Write(rendered)
	return writeErr
}

// RenderInterceptor automatically renders templates for routes with "render" metadata.
// It injects the view engine into the context and renders the template if the handler
// returns data (via context store key "__render_data").
type RenderInterceptor struct {
	engine ViewEngine
}

// NewRenderInterceptor creates an interceptor that renders templates.
func NewRenderInterceptor(engine ViewEngine) *RenderInterceptor {
	return &RenderInterceptor{engine: engine}
}

func (ri *RenderInterceptor) Intercept(ctx ExecutionContext, next CallHandler) (any, error) {
	// Inject view engine into context
	ctx.Set("__view_engine", ri.engine)

	result, err := next.Handle()
	if err != nil {
		return nil, err
	}

	// Check if route has "render" metadata for auto-rendering
	templateName, ok := ctx.GetMetadata("render")
	if !ok || templateName == nil {
		return result, nil
	}

	name, ok := templateName.(string)
	if !ok || name == "" {
		return result, nil
	}

	// The handler should have set data; use the result as template data
	data := result
	if data == nil {
		data = map[string]any{}
	}

	rendered, renderErr := ri.engine.Render(name, data)
	if renderErr != nil {
		return nil, NewInternalServerError("template render error: " + renderErr.Error())
	}

	w := ctx.ResponseWriter()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, writeErr := w.Write(rendered)
	return nil, writeErr
}
