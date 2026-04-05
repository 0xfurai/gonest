package main

import (
	"fmt"
	"log"
	"time"

	"github.com/gonest"
)

type SSEController struct{}

func NewSSEController() *SSEController { return &SSEController{} }

func (c *SSEController) Register(r gonest.Router) {
	// Stream events every second for 30 seconds
	r.Get("/sse", gonest.SSE(func(stream *gonest.SSEStream) {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		count := 0
		for range ticker.C {
			count++
			stream.Send(gonest.SSEEvent{
				ID:    fmt.Sprintf("%d", count),
				Event: "message",
				Data:  map[string]any{"count": count, "time": time.Now().Format(time.RFC3339)},
			})
			if count >= 30 {
				stream.Close()
				return
			}
		}
	}))
}

var AppModule = gonest.NewModule(gonest.ModuleOptions{
	Controllers: []any{NewSSEController},
})

func main() {
	app := gonest.Create(AppModule)
	app.EnableCors()
	log.Fatal(app.Listen(":3000"))
}
