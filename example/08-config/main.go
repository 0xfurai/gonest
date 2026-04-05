package main

import (
	"log"
	"net/http"

	"github.com/0xfurai/gonest"
	"github.com/0xfurai/gonest/config"
)

type AppController struct {
	config *config.ConfigService
}

func NewAppController(cfg *config.ConfigService) *AppController {
	return &AppController{config: cfg}
}

func (c *AppController) Register(r gonest.Router) {
	// Show current configuration values
	r.Get("/config", c.showConfig)
}

func (c *AppController) showConfig(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, map[string]any{
		"port":     c.config.GetOrDefault("PORT", "3000"),
		"env":      c.config.GetOrDefault("NODE_ENV", "development"),
		"db_host":  c.config.GetOrDefault("DB_HOST", "localhost"),
		"debug":    c.config.GetBoolOrDefault("DEBUG", false),
	})
}

var ConfigModule = config.NewModule(config.ModuleOptions{
	IsGlobal: true,
})

var AppModule = gonest.NewModule(gonest.ModuleOptions{
	Imports:     []*gonest.Module{ConfigModule},
	Controllers: []any{NewAppController},
})

func main() {
	app := gonest.Create(AppModule)
	port := ":" + "3000" // Would use config service in production
	log.Fatal(app.Listen(port))
}
