package websocket

import "net/http"

// WebSocketAdapter defines the pluggable transport interface for WebSocket
// implementations. Equivalent to NestJS WebSocketAdapter.
//
// Implementations:
//   - IoAdapter (websocket/socketio) — Socket.IO–style adapter with rooms, namespaces, ack
//   - WsAdapter (websocket default)  — Raw WebSocket adapter
//
// Usage:
//
//	app.UseWebSocketAdapter(socketio.NewIoAdapter(socketio.IoAdapterOptions{...}))
type WebSocketAdapter interface {
	// Create initializes the server on the given HTTP handler or address.
	Create(handler http.Handler) error
	// BindClientConnect registers a callback for new client connections.
	BindClientConnect(callback func(client *Client))
	// BindClientDisconnect registers a callback for client disconnections.
	BindClientDisconnect(client *Client, callback func())
	// BindMessageHandlers registers event handlers on a specific client.
	BindMessageHandlers(client *Client, handlers []WsMessageHandler)
	// Close shuts down the server.
	Close() error
	// ServeHTTP makes the adapter usable as an http.Handler.
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

// WsMessageHandler binds an event name to a handler function.
type WsMessageHandler struct {
	// Message is the event name.
	Message string
	// Callback processes the message and returns a response.
	Callback MessageHandler
}
