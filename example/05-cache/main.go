package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gonest"
	"github.com/gonest/cache"
)

type ItemsController struct {
	store cache.Store
}

func NewItemsController() *ItemsController {
	return &ItemsController{store: cache.NewMemoryStore()}
}

func (c *ItemsController) Register(r gonest.Router) {
	r.Prefix("/items")
	r.UseInterceptors(cache.NewCacheInterceptor(c.store, 10*time.Second))

	// List all items (cached for 10s)
	r.Get("/", c.findAll)

	// Get item by ID (cached for 10s)
	r.Get("/:id", c.findOne)
}

func (c *ItemsController) findAll(ctx gonest.Context) error {
	// Simulate slow database query
	time.Sleep(100 * time.Millisecond)
	return ctx.JSON(http.StatusOK, []map[string]any{
		{"id": 1, "name": "Item 1"},
		{"id": 2, "name": "Item 2"},
	})
}

func (c *ItemsController) findOne(ctx gonest.Context) error {
	id := ctx.Param("id")
	return ctx.JSON(http.StatusOK, map[string]any{"id": id, "name": "Item"})
}

var AppModule = gonest.NewModule(gonest.ModuleOptions{
	Controllers: []any{NewItemsController},
})

func main() {
	app := gonest.Create(AppModule)
	log.Fatal(app.Listen(":3000"))
}
