package main

import (
	"log"
	"net/http"

	"github.com/gonest"
)

type CatsControllerV1 struct{}

func NewCatsControllerV1() *CatsControllerV1 { return &CatsControllerV1{} }

func (c *CatsControllerV1) Register(r gonest.Router) {
	r.Prefix("/v1/cats")

	// V1: returns simple string array
	r.Get("/", c.findAll)
}

func (c *CatsControllerV1) findAll(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, map[string]any{
		"version": "1",
		"cats":    []string{"Pixel"},
	})
}

type CatsControllerV2 struct{}

func NewCatsControllerV2() *CatsControllerV2 { return &CatsControllerV2{} }

func (c *CatsControllerV2) Register(r gonest.Router) {
	r.Prefix("/v2/cats")

	// V2: returns objects with id, name, breed
	r.Get("/", c.findAll)
}

func (c *CatsControllerV2) findAll(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, map[string]any{
		"version": "2",
		"data": []map[string]any{
			{"id": 1, "name": "Pixel", "breed": "Bombay"},
		},
	})
}

var AppModule = gonest.NewModule(gonest.ModuleOptions{
	Controllers: []any{NewCatsControllerV1, NewCatsControllerV2},
})

func main() {
	app := gonest.Create(AppModule)
	app.UseGlobalMiddleware(gonest.NewVersioningMiddleware(gonest.VersioningOptions{
		Type:           gonest.VersioningURI,
		DefaultVersion: "1",
	}))
	log.Fatal(app.Listen(":3000"))
}
