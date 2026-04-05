package socketio

import (
	"encoding/json"
	"sync"
	"sync/atomic"

	"github.com/0xfurai/gonest/websocket"
)

// SocketMiddleware is a function that runs when a new socket connects.
// Return an error to reject the connection.
type SocketMiddleware func(socket *Socket) error

// Socket represents a single Socket.IO client connection.
// It extends the base websocket.Client with rooms, acknowledgements,
// and namespace-scoped event handling.
type Socket struct {
	// Client is the underlying WebSocket client.
	*websocket.Client

	namespace *Namespace
	adapter   *IoAdapter
	conn      websocket.WebSocketConn

	mu       sync.RWMutex
	rooms    map[string]bool
	handlers map[string]websocket.MessageHandler
	closed   bool

	// Handshake data attached during middleware/auth.
	Data map[string]any

	ackCounter atomic.Int64
	pendingAck sync.Map // map[int64]chan json.RawMessage
}

// socketIOMessage is the wire format for Socket.IO messages.
// It carries the event name, data payload, an optional ack ID,
// and the namespace.
type socketIOMessage struct {
	Event     string          `json:"event"`
	Data      json.RawMessage `json:"data"`
	ID        int64           `json:"id,omitempty"`       // ack ID
	Namespace string          `json:"nsp,omitempty"`
}

// socketIOAckMessage is sent as an acknowledgement.
type socketIOAckMessage struct {
	ID        int64           `json:"id"`
	Data      json.RawMessage `json:"data,omitempty"`
	Namespace string          `json:"nsp,omitempty"`
}

func newSocket(id string, conn websocket.WebSocketConn, ns *Namespace, adapter *IoAdapter) *Socket {
	client := &websocket.Client{ID: id}
	return &Socket{
		Client:    client,
		namespace: ns,
		adapter:   adapter,
		conn:      conn,
		rooms:     make(map[string]bool),
		handlers:  make(map[string]websocket.MessageHandler),
		Data:      make(map[string]any),
	}
}

// Join adds the socket to one or more rooms.
func (s *Socket) Join(rooms ...string) {
	s.mu.Lock()
	for _, room := range rooms {
		s.rooms[room] = true
	}
	s.mu.Unlock()

	// Also register with the namespace room manager
	for _, room := range rooms {
		s.namespace.joinRoom(room, s)
	}
}

// Leave removes the socket from a room.
func (s *Socket) Leave(room string) {
	s.mu.Lock()
	delete(s.rooms, room)
	s.mu.Unlock()

	s.namespace.leaveRoom(room, s)
}

// Rooms returns the list of rooms this socket is currently in.
func (s *Socket) Rooms() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rooms := make([]string, 0, len(s.rooms))
	for r := range s.rooms {
		rooms = append(rooms, r)
	}
	return rooms
}

// InRoom returns true if the socket is in the given room.
func (s *Socket) InRoom(room string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.rooms[room]
}

// Emit sends an event with data to this socket.
func (s *Socket) Emit(event string, data any) error {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return nil
	}
	s.mu.RUnlock()

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	msg := socketIOMessage{
		Event:     event,
		Data:      dataBytes,
		Namespace: s.namespace.name,
	}
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return s.conn.WriteMessage(1, b) // 1 = TextMessage
}

// EmitWithAck sends an event and waits for an acknowledgement response.
// The returned channel receives the ack data when the client responds.
func (s *Socket) EmitWithAck(event string, data any) (<-chan json.RawMessage, error) {
	ackID := s.ackCounter.Add(1)
	ch := make(chan json.RawMessage, 1)
	s.pendingAck.Store(ackID, ch)

	dataBytes, err := json.Marshal(data)
	if err != nil {
		s.pendingAck.Delete(ackID)
		close(ch)
		return ch, err
	}

	msg := socketIOMessage{
		Event:     event,
		Data:      dataBytes,
		ID:        ackID,
		Namespace: s.namespace.name,
	}
	b, err := json.Marshal(msg)
	if err != nil {
		s.pendingAck.Delete(ackID)
		close(ch)
		return ch, err
	}

	if err := s.conn.WriteMessage(1, b); err != nil {
		s.pendingAck.Delete(ackID)
		close(ch)
		return ch, err
	}

	return ch, nil
}

// To returns a BroadcastOperator targeting a room, excluding this socket.
func (s *Socket) To(room string) *BroadcastOperator {
	return &BroadcastOperator{
		namespace: s.namespace,
		rooms:     map[string]bool{room: true},
		except:    map[string]bool{s.Client.ID: true},
	}
}

// Broadcast returns a BroadcastOperator that sends to all sockets in the
// namespace except this one.
func (s *Socket) Broadcast() *BroadcastOperator {
	return &BroadcastOperator{
		namespace: s.namespace,
		rooms:     nil, // nil means all sockets
		except:    map[string]bool{s.Client.ID: true},
	}
}

// Disconnect closes the socket connection.
func (s *Socket) Disconnect() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	s.mu.Unlock()

	s.cleanup()
	_ = s.conn.Close()
}

// GetNamespace returns the namespace this socket belongs to.
func (s *Socket) GetNamespace() *Namespace {
	return s.namespace
}

// addHandler registers an event handler on this socket.
func (s *Socket) addHandler(event string, handler websocket.MessageHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[event] = handler
}

// readLoop reads messages from the WebSocket connection and dispatches to handlers.
func (s *Socket) readLoop() {
	defer func() {
		s.mu.Lock()
		if !s.closed {
			s.closed = true
		}
		s.mu.Unlock()
		s.cleanup()
		_ = s.conn.Close()
	}()

	for {
		_, data, err := s.conn.ReadMessage()
		if err != nil {
			break
		}

		// Try to parse as ack message first
		var ackMsg socketIOAckMessage
		if err := json.Unmarshal(data, &ackMsg); err == nil && ackMsg.ID > 0 && ackMsg.Data != nil {
			if ch, ok := s.pendingAck.LoadAndDelete(ackMsg.ID); ok {
				ackCh := ch.(chan json.RawMessage)
				ackCh <- ackMsg.Data
				close(ackCh)
				continue
			}
		}

		// Parse as regular message
		var msg socketIOMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}

		// Skip empty events
		if msg.Event == "" {
			continue
		}

		s.mu.RLock()
		handler, ok := s.handlers[msg.Event]
		s.mu.RUnlock()

		// Also check namespace-level handlers
		if !ok {
			handler, ok = s.namespace.getHandler(msg.Event)
		}

		if !ok {
			_ = s.Emit("error", map[string]string{"message": "unknown event: " + msg.Event})
			continue
		}

		result, err := handler(s.Client, msg.Data)
		if err != nil {
			_ = s.Emit("error", map[string]string{"message": err.Error()})
			continue
		}

		// If the message had an ack ID, send the response as an ack
		if msg.ID > 0 && result != nil {
			resultBytes, _ := json.Marshal(result)
			ack := socketIOAckMessage{
				ID:        msg.ID,
				Data:      resultBytes,
				Namespace: s.namespace.name,
			}
			ackBytes, _ := json.Marshal(ack)
			_ = s.conn.WriteMessage(1, ackBytes)
			continue
		}

		// Otherwise send back as a regular response
		if result != nil {
			_ = s.Emit(msg.Event, result)
		}
	}
}

// cleanup removes the socket from all rooms and the namespace.
func (s *Socket) cleanup() {
	// Leave all rooms
	s.mu.RLock()
	rooms := make([]string, 0, len(s.rooms))
	for r := range s.rooms {
		rooms = append(rooms, r)
	}
	s.mu.RUnlock()

	for _, room := range rooms {
		s.namespace.leaveRoom(room, s)
	}

	// Remove from namespace
	s.namespace.removeSocket(s)

	// Notify adapter
	s.adapter.notifyDisconnect(s.Client.ID)

	// Close pending acks
	s.pendingAck.Range(func(key, value any) bool {
		ch := value.(chan json.RawMessage)
		close(ch)
		s.pendingAck.Delete(key)
		return true
	})

	// Notify namespace disconnect handlers
	s.namespace.notifyDisconnect(s)
}
