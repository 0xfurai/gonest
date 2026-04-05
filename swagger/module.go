package swagger

import (
	"net/http"

	"github.com/0xfurai/gonest"
)

// Options configures the Swagger module.
type Options struct {
	Title       string
	Description string
	Version     string
	BasePath    string
	// Path is the URL where swagger UI is served (default: "/swagger").
	Path string
	// BearerAuth enables a Bearer/JWT "Authorize" button in Swagger UI.
	BearerAuth bool
}

// Module creates a Swagger module that serves OpenAPI documentation.
func Module(opts Options) *gonest.Module {
	if opts.Path == "" {
		opts.Path = "/swagger"
	}
	if opts.Version == "" {
		opts.Version = "1.0.0"
	}

	gen := NewGenerator(opts)
	ctrl := &swaggerController{generator: gen, opts: opts}

	return gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{func() *swaggerController { return ctrl }},
		Providers:   []any{gonest.ProvideValue[*Generator](gen)},
		Exports:     []any{(*Generator)(nil)},
	})
}

type swaggerController struct {
	generator *Generator
	opts      Options
}

func (c *swaggerController) Register(r gonest.Router) {
	r.Prefix(c.opts.Path)

	// Swagger endpoints are always public (no auth required)
	r.Get("/", c.serveUI).SetMetadata("public", true)
	r.Get("/json", c.serveSpec).SetMetadata("public", true)
}

func (c *swaggerController) serveSpec(ctx gonest.Context) error {
	spec := c.generator.Generate()
	return ctx.JSON(http.StatusOK, spec)
}

func (c *swaggerController) serveUI(ctx gonest.Context) error {
	specURL := c.opts.Path + "/json"
	html := generateSwaggerHTML(c.opts.Title, specURL)
	w := ctx.ResponseWriter()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte(html))
	return err
}

func generateSwaggerHTML(title, specURL string) string {
	return `<!DOCTYPE html>
<html>
<head>
  <title>` + title + ` - Swagger UI</title>
  <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist/swagger-ui-bundle.js"></script>
  <script>
    SwaggerUIBundle({
      url: "` + specURL + `",
      dom_id: '#swagger-ui',
      presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
      layout: "BaseLayout",
      persistAuthorization: true
    });
  </script>
</body>
</html>`
}
