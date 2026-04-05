package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/0xfurai/gonest"
	"github.com/0xfurai/gonest/websocket"
)

// EventsGateway handles WebSocket connections.
type EventsGateway struct {
	server *websocket.Server
}

func (g *EventsGateway) Handlers() map[string]websocket.MessageHandler {
	return map[string]websocket.MessageHandler{
		"events": func(client *websocket.Client, data json.RawMessage) (any, error) {
			return []int{1, 2, 3}, nil
		},
		"identity": func(client *websocket.Client, data json.RawMessage) (any, error) {
			var num int
			json.Unmarshal(data, &num)
			return num, nil
		},
		"broadcast": func(client *websocket.Client, data json.RawMessage) (any, error) {
			g.server.Broadcast("notification", map[string]string{"message": "hello everyone"})
			return "broadcasted", nil
		},
	}
}

func (g *EventsGateway) OnConnection(client *websocket.Client) {
	fmt.Printf("Client connected: %s\n", client.ID)
}

func (g *EventsGateway) OnDisconnect(client *websocket.Client) {
	fmt.Printf("Client disconnected: %s\n", client.ID)
}

// WebSocket controller serves the upgrade endpoint
type WSController struct{}

func NewWSController() *WSController { return &WSController{} }

func (c *WSController) Register(r gonest.Router) {
	// Welcome message with WS connection instructions
	r.Get("/", c.home)
}

func (c *WSController) home(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, map[string]string{
		"message": "WebSocket server running. Connect to /ws",
	})
}

var AppModule = gonest.NewModule(gonest.ModuleOptions{
	Controllers: []any{NewWSController},
})

func main() {
	app := gonest.Create(AppModule)
	app.EnableCors()

	log.Fatal(app.Listen(":3000"))
}
