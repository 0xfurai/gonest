package main

import (
	"log"
	"net/http"

	"github.com/0xfurai/gonest"
	"github.com/0xfurai/gonest/health"
)

type AppController struct{}

func NewAppController() *AppController { return &AppController{} }

func (c *AppController) Register(r gonest.Router) {
	// Root endpoint
	r.Get("/", c.root)
}

func (c *AppController) root(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, map[string]string{"message": "App running"})
}

func main() {
	healthMod := health.NewModule(health.Options{
		Indicators: []health.HealthIndicator{
			&health.PingIndicator{},
			&health.CustomIndicator{
				IndicatorName: "app",
				CheckFn: func() health.HealthResult {
					return health.HealthResult{
						Status:  health.StatusUp,
						Details: map[string]any{"version": "1.0.0"},
					}
				},
			},
		},
	})

	appModule := gonest.NewModule(gonest.ModuleOptions{
		Imports:     []*gonest.Module{healthMod},
		Controllers: []any{NewAppController},
	})

	app := gonest.Create(appModule)
	log.Fatal(app.Listen(":3000"))
}
