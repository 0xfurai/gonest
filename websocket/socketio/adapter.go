// Package socketio provides a Socket.IO–style WebSocket adapter for GoNest.
//
// It implements rooms, namespaces, acknowledgements, middleware, and broadcasting
// patterns matching the NestJS platform-socket.io adapter.
//
// Usage:
//
//	adapter := socketio.NewIoAdapter(socketio.IoAdapterOptions{
//	    Upgrader: myUpgrader,              // implements websocket.Upgrader
//	    PingInterval: 25 * time.Second,
//	    PingTimeout:  20 * time.Second,
//	})
//
//	module := socketio.NewModule(socketio.ModuleOptions{
//	    Adapter:   adapter,
//	    Gateway:   &ChatGateway{},
//	    Namespace: "/chat",
//	    Path:      "/socket.io",
//	})
package socketio

import (
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gonest/websocket"
)

// IoAdapter implements a Socket.IO–compatible WebSocket adapter with rooms,
// namespaces, acknowledgements, and middleware support.
// Equivalent to NestJS IoAdapter from @nestjs/platform-socket.io.
type IoAdapter struct {
	mu         sync.RWMutex
	opts       IoAdapterOptions
	namespaces map[string]*Namespace
	upgrader   websocket.Upgrader
	nextID     atomic.Int64

	onConnect    func(client *websocket.Client)
	onDisconnect map[string]func()
	disconnMu    sync.RWMutex

	middleware []SocketMiddleware
}

// IoAdapterOptions configures the Socket.IO adapter.
type IoAdapterOptions struct {
	// Upgrader handles HTTP→WebSocket upgrade. Required.
	Upgrader websocket.Upgrader
	// PingInterval is the interval between keep-alive pings (default: 25s).
	PingInterval time.Duration
	// PingTimeout is the timeout waiting for a pong response (default: 20s).
	PingTimeout time.Duration
	// MaxPayload is the maximum message size in bytes (default: 1MB).
	MaxPayload int64
	// CORS origin for upgrade requests (default: "*").
	CorsOrigin string
}

// NewIoAdapter creates a new Socket.IO adapter.
func NewIoAdapter(opts IoAdapterOptions) *IoAdapter {
	if opts.PingInterval == 0 {
		opts.PingInterval = 25 * time.Second
	}
	if opts.PingTimeout == 0 {
		opts.PingTimeout = 20 * time.Second
	}
	if opts.MaxPayload == 0 {
		opts.MaxPayload = 1 << 20 // 1MB
	}
	if opts.CorsOrigin == "" {
		opts.CorsOrigin = "*"
	}

	return &IoAdapter{
		opts:         opts,
		namespaces:   make(map[string]*Namespace),
		upgrader:     opts.Upgrader,
		onDisconnect: make(map[string]func()),
	}
}

// Create initializes the adapter. The handler parameter is available for
// attaching to an existing HTTP server but is not required for standalone use.
func (a *IoAdapter) Create(handler http.Handler) error {
	// Ensure the default namespace exists
	a.GetNamespace("/")
	return nil
}

// Use adds middleware that runs on every new socket connection.
func (a *IoAdapter) Use(mw ...SocketMiddleware) {
	a.middleware = append(a.middleware, mw...)
}

// BindClientConnect registers the connection callback.
func (a *IoAdapter) BindClientConnect(callback func(client *websocket.Client)) {
	a.onConnect = callback
}

// BindClientDisconnect registers a disconnect callback for a specific client.
func (a *IoAdapter) BindClientDisconnect(client *websocket.Client, callback func()) {
	a.disconnMu.Lock()
	a.onDisconnect[client.ID] = callback
	a.disconnMu.Unlock()
}

// BindMessageHandlers registers event handlers for a client.
func (a *IoAdapter) BindMessageHandlers(client *websocket.Client, handlers []websocket.WsMessageHandler) {
	// Find the socket associated with this client
	a.mu.RLock()
	for _, ns := range a.namespaces {
		ns.mu.RLock()
		if sock, ok := ns.sockets[client.ID]; ok {
			ns.mu.RUnlock()
			for _, h := range handlers {
				sock.addHandler(h.Message, h.Callback)
			}
			a.mu.RUnlock()
			return
		}
		ns.mu.RUnlock()
	}
	a.mu.RUnlock()
}

// Close shuts down all namespaces and disconnects all sockets.
func (a *IoAdapter) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, ns := range a.namespaces {
		ns.Close()
	}
	return nil
}

// ServeHTTP handles WebSocket upgrade requests.
func (a *IoAdapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	origin := a.opts.CorsOrigin
	if origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}

	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Determine namespace from query or path
	nsName := r.URL.Query().Get("nsp")
	if nsName == "" {
		nsName = "/"
	}

	conn, err := a.upgrader.Upgrade(w, r)
	if err != nil {
		log.Printf("socketio: upgrade error: %v", err)
		return
	}

	ns := a.GetNamespace(nsName)
	sock := ns.addSocket(conn, a)

	// Run middleware chain
	for _, mw := range a.middleware {
		if err := mw(sock); err != nil {
			_ = sock.Emit("error", map[string]string{"message": err.Error()})
			sock.Disconnect()
			return
		}
	}

	// Run namespace middleware
	for _, mw := range ns.middleware {
		if err := mw(sock); err != nil {
			_ = sock.Emit("error", map[string]string{"message": err.Error()})
			sock.Disconnect()
			return
		}
	}

	// Notify connection
	if a.onConnect != nil {
		a.onConnect(sock.Client)
	}
	ns.notifyConnect(sock)

	// Start the read loop
	go sock.readLoop()
}

// GetNamespace returns a namespace, creating it if it doesn't exist.
func (a *IoAdapter) GetNamespace(name string) *Namespace {
	a.mu.Lock()
	defer a.mu.Unlock()
	if ns, ok := a.namespaces[name]; ok {
		return ns
	}
	ns := newNamespace(name, a)
	a.namespaces[name] = ns
	return ns
}

// Of is an alias for GetNamespace, matching Socket.IO's io.of("/nsp") API.
func (a *IoAdapter) Of(name string) *Namespace {
	return a.GetNamespace(name)
}

// Emit broadcasts an event to all sockets in the default namespace.
func (a *IoAdapter) Emit(event string, data any) {
	ns := a.GetNamespace("/")
	ns.Emit(event, data)
}

// To returns a broadcast builder targeting a specific room in the default namespace.
func (a *IoAdapter) To(room string) *BroadcastOperator {
	ns := a.GetNamespace("/")
	return ns.To(room)
}

// notifyDisconnect is called when a socket disconnects.
func (a *IoAdapter) notifyDisconnect(clientID string) {
	a.disconnMu.RLock()
	cb, ok := a.onDisconnect[clientID]
	a.disconnMu.RUnlock()
	if ok {
		cb()
		a.disconnMu.Lock()
		delete(a.onDisconnect, clientID)
		a.disconnMu.Unlock()
	}
}
