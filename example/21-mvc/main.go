package main

import (
	"html/template"
	"log"
	"os"
	"path/filepath"

	"github.com/gonest"
)

// --- Data ---

// PageData carries view model data for template rendering.
type PageData struct {
	Title    string
	Heading  string
	Message  string
	Items    []string
	Year     int
}

// --- Service ---

// ContentService provides content for template rendering.
type ContentService struct{}

// NewContentService creates a new ContentService.
func NewContentService() *ContentService {
	return &ContentService{}
}

// GetHomePage returns data for the home page.
func (s *ContentService) GetHomePage() PageData {
	return PageData{
		Title:   "GoNest MVC",
		Heading: "Welcome to GoNest MVC",
		Message: "This example demonstrates server-side template rendering with the GoNest TemplateEngine.",
		Items:   []string{"Modules", "Controllers", "Services", "Templates"},
		Year:    2026,
	}
}

// GetAboutPage returns data for the about page.
func (s *ContentService) GetAboutPage() PageData {
	return PageData{
		Title:   "About - GoNest MVC",
		Heading: "About GoNest",
		Message: "GoNest is a progressive Go framework inspired by NestJS, bringing structure and modularity to your Go backend applications.",
		Year:    2026,
	}
}

// GetContactPage returns data for the contact page.
func (s *ContentService) GetContactPage() PageData {
	return PageData{
		Title:   "Contact - GoNest MVC",
		Heading: "Contact Us",
		Message: "Get in touch with the GoNest team.",
		Year:    2026,
	}
}

// --- Templates ---

// setupTemplates creates a temporary directory with HTML template files
// and returns a configured TemplateEngine.
func setupTemplates() (*gonest.TemplateEngine, string) {
	viewsDir, err := os.MkdirTemp("", "gonest-mvc-views-*")
	if err != nil {
		log.Fatalf("failed to create views directory: %v", err)
	}

	// Layout template that other templates extend
	layoutHTML := `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{.Title}}</title>
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; line-height: 1.6; color: #333; }
    nav { background: #2c3e50; padding: 1rem 2rem; }
    nav a { color: #ecf0f1; text-decoration: none; margin-right: 1.5rem; font-weight: 500; }
    nav a:hover { color: #3498db; }
    .container { max-width: 800px; margin: 2rem auto; padding: 0 1rem; }
    h1 { margin-bottom: 1rem; color: #2c3e50; }
    .content { background: #f8f9fa; border-radius: 8px; padding: 2rem; margin-bottom: 2rem; }
    ul { list-style: none; padding: 0; }
    ul li { padding: 0.5rem 0; border-bottom: 1px solid #dee2e6; }
    ul li:last-child { border-bottom: none; }
    footer { text-align: center; padding: 2rem; color: #6c757d; font-size: 0.9rem; }
  </style>
</head>
<body>
  <nav>
    <a href="/">Home</a>
    <a href="/about">About</a>
    <a href="/contact">Contact</a>
  </nav>
  <div class="container">
    <h1>{{.Heading}}</h1>
    <div class="content">
      <p>{{.Message}}</p>
      {{if .Items}}
      <h3 style="margin-top: 1.5rem; margin-bottom: 0.5rem;">Core Concepts:</h3>
      <ul>
        {{range .Items}}<li>{{.}}</li>{{end}}
      </ul>
      {{end}}
    </div>
  </div>
  <footer>&copy; {{.Year}} GoNest Framework</footer>
</body>
</html>`

	// Write template files for each page — they all use the same layout structure.
	pages := map[string]string{
		"home":    layoutHTML,
		"about":   layoutHTML,
		"contact": layoutHTML,
	}

	for name, content := range pages {
		filePath := filepath.Join(viewsDir, name+".html")
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			log.Fatalf("failed to write template %s: %v", name, err)
		}
	}

	engine := gonest.NewTemplateEngine(viewsDir)
	engine.AddFunc("upper", func(s string) string {
		result := make([]byte, len(s))
		for i := 0; i < len(s); i++ {
			c := s[i]
			if c >= 'a' && c <= 'z' {
				c -= 32
			}
			result[i] = c
		}
		return string(result)
	})

	return engine, viewsDir
}

// setupStaticDir creates a temporary directory for static CSS assets.
func setupStaticDir() string {
	staticDir, err := os.MkdirTemp("", "gonest-mvc-static-*")
	if err != nil {
		log.Fatalf("failed to create static directory: %v", err)
	}

	css := `/* GoNest MVC Static Styles */
body { background-color: #ffffff; }
.highlight { color: #e74c3c; font-weight: bold; }
`
	if err := os.WriteFile(filepath.Join(staticDir, "style.css"), []byte(css), 0644); err != nil {
		log.Fatalf("failed to write CSS file: %v", err)
	}

	return staticDir
}

// --- Controller ---

// ViewController renders HTML pages using the TemplateEngine.
type ViewController struct {
	engine  *gonest.TemplateEngine
	service *ContentService
}

// NewViewController creates a new ViewController.
func NewViewController(service *ContentService, engine *gonest.TemplateEngine) *ViewController {
	return &ViewController{engine: engine, service: service}
}

// Register defines the view routes.
func (c *ViewController) Register(r gonest.Router) {
	r.Get("/", c.home)
	r.Get("/about", c.about)
	r.Get("/contact", c.contact)
	r.Get("/inline", c.inlineTemplate)
}

func (c *ViewController) home(ctx gonest.Context) error {
	data := c.service.GetHomePage()
	return c.engine.Render(ctx.ResponseWriter(), "home", data)
}

func (c *ViewController) about(ctx gonest.Context) error {
	data := c.service.GetAboutPage()
	return c.engine.Render(ctx.ResponseWriter(), "about", data)
}

func (c *ViewController) contact(ctx gonest.Context) error {
	data := c.service.GetContactPage()
	return c.engine.Render(ctx.ResponseWriter(), "contact", data)
}

// inlineTemplate demonstrates rendering a template compiled at runtime
// rather than from the filesystem.
func (c *ViewController) inlineTemplate(ctx gonest.Context) error {
	tmpl := template.Must(template.New("inline").Parse(`<!DOCTYPE html>
<html>
<head><title>Inline Template</title></head>
<body>
  <h1>{{.Heading}}</h1>
  <p>This template was compiled inline at runtime.</p>
  <p>Rendered via: {{.Framework}}</p>
</body>
</html>`))

	data := struct {
		Heading   string
		Framework string
	}{
		Heading:   "Inline Template Example",
		Framework: "GoNest TemplateEngine",
	}

	w := ctx.ResponseWriter()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return tmpl.Execute(w, data)
}

// --- Static Files Controller ---

// StaticController serves static assets from a directory.
type StaticController struct {
	staticDir string
}

// NewStaticController creates a new StaticController.
func NewStaticController(staticDir string) *StaticController {
	return &StaticController{staticDir: staticDir}
}

// Register defines the static file route.
func (c *StaticController) Register(r gonest.Router) {
	r.Get("/static/*", gonest.StaticFiles("/static/", c.staticDir))
}

// --- Module ---

func main() {
	engine, viewsDir := setupTemplates()
	defer os.RemoveAll(viewsDir)

	staticDir := setupStaticDir()
	defer os.RemoveAll(staticDir)

	// Build the MVC module with the template engine and static controller.
	mvcModule := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{
			func(svc *ContentService) *ViewController {
				return NewViewController(svc, engine)
			},
			func() *StaticController {
				return NewStaticController(staticDir)
			},
		},
		Providers: []any{NewContentService},
	})

	appModule := gonest.NewModule(gonest.ModuleOptions{
		Imports: []*gonest.Module{mvcModule},
	})

	// --- Bootstrap ---
	app := gonest.Create(appModule)

	log.Println("MVC application running at http://localhost:3000")
	log.Println("Pages: /, /about, /contact, /inline")
	log.Println("Static: /static/style.css")
	log.Fatal(app.Listen(":3000"))
}
