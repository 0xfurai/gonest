package socketio

import (
	"net/http"

	"github.com/gonest"
	"github.com/gonest/websocket"
)

// ModuleOptions configures the Socket.IO module.
type ModuleOptions struct {
	// Adapter is the IoAdapter instance. If nil, one will be created using Upgrader.
	Adapter *IoAdapter
	// Gateway is the gateway implementation handling messages.
	Gateway websocket.Gateway
	// Namespace is the namespace for this gateway (default: "/").
	Namespace string
	// Path is the HTTP path for WebSocket connections (default: "/socket.io").
	Path string
	// Upgrader handles HTTP→WS upgrade. Required if Adapter is nil.
	Upgrader websocket.Upgrader
}

// NewModule creates a Socket.IO module that registers a gateway with rooms,
// namespaces, and acknowledgement support.
//
// Usage:
//
//	module := socketio.NewModule(socketio.ModuleOptions{
//	    Gateway:   &ChatGateway{},
//	    Namespace: "/chat",
//	    Path:      "/socket.io",
//	    Upgrader:  myUpgrader,
//	})
func NewModule(opts ModuleOptions) *gonest.Module {
	if opts.Path == "" {
		opts.Path = "/socket.io"
	}
	if opts.Namespace == "" {
		opts.Namespace = "/"
	}

	adapter := opts.Adapter
	if adapter == nil {
		upgrader := opts.Upgrader
		if upgrader == nil {
			upgrader = &websocket.DefaultUpgrader{}
		}
		adapter = NewIoAdapter(IoAdapterOptions{
			Upgrader: upgrader,
		})
	}

	// Set up the namespace with gateway handlers
	ns := adapter.GetNamespace(opts.Namespace)

	if opts.Gateway != nil {
		// Register gateway event handlers on the namespace
		for event, handler := range opts.Gateway.Handlers() {
			ns.On(event, handler)
		}

		// Register connection/disconnection callbacks
		ns.OnConnection(func(sock *Socket) {
			opts.Gateway.OnConnection(sock.Client)
		})
		ns.OnDisconnection(func(sock *Socket) {
			opts.Gateway.OnDisconnect(sock.Client)
		})
	}

	ctrl := &ioController{path: opts.Path, adapter: adapter}

	// Build the underlying websocket.Server for backward compatibility
	// with existing code that injects *websocket.Server
	wsServer := websocket.NewServer(opts.Gateway, opts.Upgrader)

	return gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{func() *ioController { return ctrl }},
		Providers: []any{
			gonest.ProvideValue[*IoAdapter](adapter),
			gonest.ProvideValue[*Namespace](ns),
			gonest.ProvideValue[*websocket.Server](wsServer),
		},
		Exports: []any{
			(*IoAdapter)(nil),
			(*Namespace)(nil),
			(*websocket.Server)(nil),
		},
	})
}

// ioController handles the HTTP endpoint for Socket.IO connections.
type ioController struct {
	path    string
	adapter *IoAdapter
}

func (c *ioController) Register(r gonest.Router) {
	r.Get(c.path, c.upgrade)
	// Socket.IO also uses POST for polling transport
	r.Post(c.path, c.upgrade)
}

func (c *ioController) upgrade(ctx gonest.Context) error {
	c.adapter.ServeHTTP(ctx.ResponseWriter(), ctx.Request())
	return nil
}

// IoGateway extends the base websocket.Gateway with Socket.IO–specific methods.
// Implement this for richer Socket.IO gateway functionality.
type IoGateway interface {
	websocket.Gateway
	// AfterInit is called after the gateway has been initialized.
	AfterInit(server *IoAdapter)
}

// NewModuleWithIoGateway creates a module using an IoGateway, calling AfterInit
// once the adapter is ready.
func NewModuleWithIoGateway(opts ModuleOptions, gateway IoGateway) *gonest.Module {
	opts.Gateway = gateway
	mod := NewModule(opts)

	// Call AfterInit with the adapter
	adapter := opts.Adapter
	if adapter == nil {
		upgrader := opts.Upgrader
		if upgrader == nil {
			upgrader = &websocket.DefaultUpgrader{}
		}
		adapter = NewIoAdapter(IoAdapterOptions{Upgrader: upgrader})
	}
	gateway.AfterInit(adapter)

	return mod
}

// GetSocket retrieves a Socket from a gonest.Context if the request was upgraded
// through the IoAdapter. Returns nil if not a WebSocket context.
func GetSocket(ctx gonest.Context) *Socket {
	val, ok := ctx.Get("__socketio_socket")
	if !ok {
		return nil
	}
	sock, _ := val.(*Socket)
	return sock
}

// handshakeInfo extracts handshake information from the upgrade request.
type HandshakeInfo struct {
	Headers http.Header
	Query   map[string]string
	Address string
	Secure  bool
}

// GetHandshake returns handshake info for a socket by parsing the original
// upgrade request headers. This is a simplified version — the real handshake
// data should be captured during upgrade.
func GetHandshake(sock *Socket) *HandshakeInfo {
	if sock == nil || sock.Data == nil {
		return &HandshakeInfo{}
	}
	info := &HandshakeInfo{}
	if headers, ok := sock.Data["headers"].(http.Header); ok {
		info.Headers = headers
	}
	if addr, ok := sock.Data["address"].(string); ok {
		info.Address = addr
	}
	return info
}
