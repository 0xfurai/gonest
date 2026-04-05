package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gonest"
)

type ApiController struct{}

func NewApiController() *ApiController { return &ApiController{} }

func (c *ApiController) Register(r gonest.Router) {
	r.Prefix("/api")

	// Regular endpoint (uses global rate limit)
	r.Get("/data", c.getData)

	// Expensive operation (custom limit: 2 per minute)
	r.Post("/expensive", c.expensiveOp).
		SetMetadata("throttle_limit", 2).
		SetMetadata("throttle_window", time.Minute)
}

func (c *ApiController) getData(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, map[string]string{"data": "hello"})
}

func (c *ApiController) expensiveOp(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, map[string]string{"result": "done"})
}

var AppModule = gonest.NewModule(gonest.ModuleOptions{
	Controllers: []any{NewApiController},
})

func main() {
	app := gonest.Create(AppModule)
	// Global rate limit: 100 requests per minute
	app.UseGlobalGuards(gonest.NewThrottleGuard(100, time.Minute))
	log.Fatal(app.Listen(":3000"))
}
