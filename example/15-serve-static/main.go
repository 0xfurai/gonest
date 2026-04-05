package main

import (
	"log"
	"net/http"

	"github.com/gonest"
)

// --- Controller ---

type AppController struct{}

func NewAppController() *AppController { return &AppController{} }

func (c *AppController) Register(r gonest.Router) {
	// API endpoint
	r.Get("/api", c.hello)
}

func (c *AppController) hello(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, map[string]string{
		"message": "Hello from GoNest API!",
	})
}

// --- Static File Controller ---

type StaticController struct{}

func NewStaticController() *StaticController { return &StaticController{} }

func (c *StaticController) Register(r gonest.Router) {
	// Serve static files from ./public directory
	r.Get("/static/*", gonest.StaticFiles("/static/", "./public"))
}

// --- Module ---

var AppModule = gonest.NewModule(gonest.ModuleOptions{
	Controllers: []any{NewAppController, NewStaticController},
})

func main() {
	app := gonest.Create(AppModule)
	app.EnableCors()
	log.Fatal(app.Listen(":3000"))
}
