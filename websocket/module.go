package websocket

import (
	"net/http"

	"github.com/gonest"
)

// GatewayOptions configures the WebSocket module.
type GatewayOptions struct {
	// Path is the HTTP path to upgrade to WebSocket (default: "/ws").
	Path string
	// Gateway is the gateway implementation handling messages.
	Gateway Gateway
	// Upgrader handles HTTP→WS upgrade. Provide your own (e.g. gorilla/websocket).
	Upgrader Upgrader
}

// NewModule creates a WebSocket module that registers a gateway on the given path.
func NewModule(opts GatewayOptions) *gonest.Module {
	if opts.Path == "" {
		opts.Path = "/ws"
	}

	server := NewServer(opts.Gateway, opts.Upgrader)
	ctrl := &wsController{path: opts.Path, server: server}

	return gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{func() *wsController { return ctrl }},
		Providers: []any{
			gonest.ProvideValue[*Server](server),
		},
		Exports: []any{(*Server)(nil)},
	})
}

type wsController struct {
	path   string
	server *Server
}

func (c *wsController) Register(r gonest.Router) {
	r.Get(c.path, c.upgrade)
}

func (c *wsController) upgrade(ctx gonest.Context) error {
	c.server.ServeHTTP(ctx.ResponseWriter(), ctx.Request())
	return nil
}

// BroadcastInterceptor is an interceptor that can broadcast events to
// all connected WebSocket clients after a handler completes.
type BroadcastInterceptor struct {
	Server *Server
	Event  string
}

func (i *BroadcastInterceptor) Intercept(ctx gonest.ExecutionContext, next gonest.CallHandler) (any, error) {
	result, err := next.Handle()
	if err != nil {
		return nil, err
	}
	if result != nil && i.Event != "" {
		i.Server.Broadcast(i.Event, result)
	}
	return result, nil
}

// DefaultUpgrader is a no-op placeholder. Replace with gorilla/websocket:
//
//	import "github.com/gorilla/websocket"
//	upgrader := &websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
type DefaultUpgrader struct{}

func (u *DefaultUpgrader) Upgrade(w http.ResponseWriter, r *http.Request) (WebSocketConn, error) {
	return nil, gonest.NewInternalServerError("no WebSocket upgrader configured; use gorilla/websocket")
}
