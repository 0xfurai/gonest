package platform

import "net/http"

// HandlerFunc is the platform-agnostic handler type.
type HandlerFunc func(w http.ResponseWriter, r *http.Request, params map[string]string)

// HTTPAdapter abstracts the underlying HTTP server implementation.
type HTTPAdapter interface {
	// Handle registers a route handler for a specific method and path.
	Handle(method, path string, handler HandlerFunc)
	// Use registers global middleware that runs on every request.
	Use(middleware func(http.Handler) http.Handler)
	// Listen starts the HTTP server on the given address.
	Listen(addr string) error
	// Handler returns the underlying http.Handler for testing.
	Handler() http.Handler
	// SetNotFoundHandler configures the 404 handler.
	SetNotFoundHandler(handler http.HandlerFunc)
	// SetMethodNotAllowedHandler configures the 405 handler.
	SetMethodNotAllowedHandler(handler http.HandlerFunc)
	// Shutdown gracefully shuts down the server.
	Shutdown() error
}
