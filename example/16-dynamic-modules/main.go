package main

import (
	"log"
	"net/http"

	"github.com/gonest"
)

// --- Dynamic Module Pattern ---

// GreetingOptions configures the greeting module.
type GreetingOptions struct {
	Message string
}

// GreetingService provides a configurable greeting.
type GreetingService struct {
	message string
}

func (s *GreetingService) GetMessage() string { return s.message }

// GreetingModuleForRoot creates a dynamically configured module.
// This mirrors NestJS's ConfigModule.register() pattern.
func GreetingModuleForRoot(opts GreetingOptions) *gonest.Module {
	svc := &GreetingService{message: opts.Message}
	return gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{gonest.ProvideValue[*GreetingService](svc)},
		Exports:   []any{(*GreetingService)(nil)},
	})
}

// --- Controller ---

type AppController struct {
	greeting *GreetingService
}

func NewAppController(greeting *GreetingService) *AppController {
	return &AppController{greeting: greeting}
}

func (c *AppController) Register(r gonest.Router) {
	r.Get("/", c.hello)
}

func (c *AppController) hello(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, map[string]string{
		"message": c.greeting.GetMessage(),
	})
}

// --- Module ---

var AppModule = gonest.NewModule(gonest.ModuleOptions{
	Imports:     []*gonest.Module{GreetingModuleForRoot(GreetingOptions{Message: "Hello from dynamic module!"})},
	Controllers: []any{NewAppController},
})

func main() {
	app := gonest.Create(AppModule)
	log.Fatal(app.Listen(":3000"))
}
