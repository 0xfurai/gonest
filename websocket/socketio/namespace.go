package socketio

import (
	"sync"

	"github.com/gonest/websocket"
)

// Namespace represents a Socket.IO namespace. Each namespace has its own
// set of rooms, sockets, event handlers, and middleware.
// Equivalent to NestJS Socket.IO namespace (io.of("/nsp")).
type Namespace struct {
	name    string
	adapter *IoAdapter

	mu      sync.RWMutex
	sockets map[string]*Socket // socketID -> socket
	rooms   map[string]map[string]*Socket // roomName -> socketID -> socket

	handlers   map[string]websocket.MessageHandler
	handlersMu sync.RWMutex

	middleware []SocketMiddleware

	onConnect    []func(*Socket)
	onDisconnect []func(*Socket)
	connectMu    sync.RWMutex
}

func newNamespace(name string, adapter *IoAdapter) *Namespace {
	return &Namespace{
		name:     name,
		adapter:  adapter,
		sockets:  make(map[string]*Socket),
		rooms:    make(map[string]map[string]*Socket),
		handlers: make(map[string]websocket.MessageHandler),
	}
}

// Name returns the namespace name (e.g., "/", "/chat").
func (ns *Namespace) Name() string {
	return ns.name
}

// Use adds middleware to this namespace.
func (ns *Namespace) Use(mw ...SocketMiddleware) {
	ns.middleware = append(ns.middleware, mw...)
}

// On registers an event handler at the namespace level.
// All sockets in this namespace will use this handler for the given event.
func (ns *Namespace) On(event string, handler websocket.MessageHandler) {
	ns.handlersMu.Lock()
	defer ns.handlersMu.Unlock()
	ns.handlers[event] = handler
}

// OnConnection registers a callback for new socket connections in this namespace.
func (ns *Namespace) OnConnection(callback func(*Socket)) {
	ns.connectMu.Lock()
	defer ns.connectMu.Unlock()
	ns.onConnect = append(ns.onConnect, callback)
}

// OnDisconnection registers a callback for socket disconnections in this namespace.
func (ns *Namespace) OnDisconnection(callback func(*Socket)) {
	ns.connectMu.Lock()
	defer ns.connectMu.Unlock()
	ns.onDisconnect = append(ns.onDisconnect, callback)
}

// Emit broadcasts an event to all sockets in this namespace.
func (ns *Namespace) Emit(event string, data any) {
	ns.mu.RLock()
	defer ns.mu.RUnlock()
	for _, sock := range ns.sockets {
		_ = sock.Emit(event, data)
	}
}

// To returns a BroadcastOperator targeting specific rooms in this namespace.
func (ns *Namespace) To(rooms ...string) *BroadcastOperator {
	roomSet := make(map[string]bool, len(rooms))
	for _, r := range rooms {
		roomSet[r] = true
	}
	return &BroadcastOperator{
		namespace: ns,
		rooms:     roomSet,
		except:    make(map[string]bool),
	}
}

// In is an alias for To, matching Socket.IO's namespace.in("room") API.
func (ns *Namespace) In(rooms ...string) *BroadcastOperator {
	return ns.To(rooms...)
}

// Except returns a BroadcastOperator that excludes specific socket IDs.
func (ns *Namespace) Except(ids ...string) *BroadcastOperator {
	exceptSet := make(map[string]bool, len(ids))
	for _, id := range ids {
		exceptSet[id] = true
	}
	return &BroadcastOperator{
		namespace: ns,
		rooms:     nil,
		except:    exceptSet,
	}
}

// GetSocket returns a socket by ID, or nil if not found.
func (ns *Namespace) GetSocket(id string) *Socket {
	ns.mu.RLock()
	defer ns.mu.RUnlock()
	return ns.sockets[id]
}

// GetSockets returns all sockets in this namespace.
func (ns *Namespace) GetSockets() []*Socket {
	ns.mu.RLock()
	defer ns.mu.RUnlock()
	sockets := make([]*Socket, 0, len(ns.sockets))
	for _, s := range ns.sockets {
		sockets = append(sockets, s)
	}
	return sockets
}

// SocketCount returns the number of connected sockets in this namespace.
func (ns *Namespace) SocketCount() int {
	ns.mu.RLock()
	defer ns.mu.RUnlock()
	return len(ns.sockets)
}

// GetRoomSockets returns all sockets in a specific room.
func (ns *Namespace) GetRoomSockets(room string) []*Socket {
	ns.mu.RLock()
	defer ns.mu.RUnlock()
	sockets, ok := ns.rooms[room]
	if !ok {
		return nil
	}
	result := make([]*Socket, 0, len(sockets))
	for _, s := range sockets {
		result = append(result, s)
	}
	return result
}

// GetRooms returns all room names in this namespace.
func (ns *Namespace) GetRooms() []string {
	ns.mu.RLock()
	defer ns.mu.RUnlock()
	rooms := make([]string, 0, len(ns.rooms))
	for r := range ns.rooms {
		rooms = append(rooms, r)
	}
	return rooms
}

// Close disconnects all sockets in this namespace.
func (ns *Namespace) Close() {
	ns.mu.RLock()
	sockets := make([]*Socket, 0, len(ns.sockets))
	for _, s := range ns.sockets {
		sockets = append(sockets, s)
	}
	ns.mu.RUnlock()

	for _, s := range sockets {
		s.Disconnect()
	}
}

// addSocket registers a new socket in this namespace.
func (ns *Namespace) addSocket(conn websocket.WebSocketConn, adapter *IoAdapter) *Socket {
	id := ns.adapter.nextID.Add(1)
	sock := newSocket(itoa(id), conn, ns, adapter)

	ns.mu.Lock()
	ns.sockets[sock.Client.ID] = sock
	ns.mu.Unlock()

	// Every socket automatically joins a room matching its own ID
	sock.Join(sock.Client.ID)

	// Register namespace-level handlers on the socket
	ns.handlersMu.RLock()
	for event, handler := range ns.handlers {
		sock.addHandler(event, handler)
	}
	ns.handlersMu.RUnlock()

	return sock
}

// removeSocket unregisters a socket from this namespace.
func (ns *Namespace) removeSocket(sock *Socket) {
	ns.mu.Lock()
	delete(ns.sockets, sock.Client.ID)
	ns.mu.Unlock()
}

// joinRoom adds a socket to a room.
func (ns *Namespace) joinRoom(room string, sock *Socket) {
	ns.mu.Lock()
	defer ns.mu.Unlock()
	if ns.rooms[room] == nil {
		ns.rooms[room] = make(map[string]*Socket)
	}
	ns.rooms[room][sock.Client.ID] = sock
}

// leaveRoom removes a socket from a room.
func (ns *Namespace) leaveRoom(room string, sock *Socket) {
	ns.mu.Lock()
	defer ns.mu.Unlock()
	if sockets, ok := ns.rooms[room]; ok {
		delete(sockets, sock.Client.ID)
		if len(sockets) == 0 {
			delete(ns.rooms, room)
		}
	}
}

// getHandler returns a namespace-level handler for an event.
func (ns *Namespace) getHandler(event string) (websocket.MessageHandler, bool) {
	ns.handlersMu.RLock()
	defer ns.handlersMu.RUnlock()
	h, ok := ns.handlers[event]
	return h, ok
}

// notifyConnect calls all connection callbacks.
func (ns *Namespace) notifyConnect(sock *Socket) {
	ns.connectMu.RLock()
	defer ns.connectMu.RUnlock()
	for _, cb := range ns.onConnect {
		cb(sock)
	}
}

// notifyDisconnect calls all disconnection callbacks.
func (ns *Namespace) notifyDisconnect(sock *Socket) {
	ns.connectMu.RLock()
	defer ns.connectMu.RUnlock()
	for _, cb := range ns.onDisconnect {
		cb(sock)
	}
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
