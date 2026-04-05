package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
)

// Gateway is the interface for WebSocket gateways.
// Implement this to handle WebSocket connections and messages.
type Gateway interface {
	// Handlers returns message handlers keyed by event name.
	Handlers() map[string]MessageHandler
	// OnConnection is called when a new client connects.
	OnConnection(client *Client)
	// OnDisconnect is called when a client disconnects.
	OnDisconnect(client *Client)
}

// MessageHandler processes a WebSocket message and returns a response.
type MessageHandler func(client *Client, data json.RawMessage) (any, error)

// Message represents a WebSocket message with an event type.
type Message struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

// OutgoingMessage is sent to WebSocket clients.
type OutgoingMessage struct {
	Event string `json:"event"`
	Data  any    `json:"data"`
}

// Client represents a connected WebSocket client.
type Client struct {
	ID     string
	conn   WebSocketConn
	server *Server
	mu     sync.Mutex
	closed bool
}

// Send sends a message to this client.
func (c *Client) Send(event string, data any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	msg := OutgoingMessage{Event: event, Data: data}
	bytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return c.conn.WriteMessage(1, bytes) // 1 = TextMessage
}

// Close disconnects the client.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	return c.conn.Close()
}

// WebSocketConn abstracts a WebSocket connection (e.g., gorilla/websocket).
type WebSocketConn interface {
	ReadMessage() (messageType int, p []byte, err error)
	WriteMessage(messageType int, data []byte) error
	Close() error
}

// Upgrader abstracts HTTP to WebSocket upgrade.
type Upgrader interface {
	Upgrade(w http.ResponseWriter, r *http.Request) (WebSocketConn, error)
}

// Server manages WebSocket connections and message routing.
type Server struct {
	mu       sync.RWMutex
	clients  map[string]*Client
	gateway  Gateway
	upgrader Upgrader
	nextID   int
}

// NewServer creates a new WebSocket server.
func NewServer(gateway Gateway, upgrader Upgrader) *Server {
	return &Server{
		clients:  make(map[string]*Client),
		gateway:  gateway,
		upgrader: upgrader,
	}
}

// ServeHTTP handles WebSocket upgrade requests.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r)
	if err != nil {
		log.Printf("websocket upgrade error: %v", err)
		return
	}

	s.mu.Lock()
	s.nextID++
	client := &Client{
		ID:     itoa(s.nextID),
		conn:   conn,
		server: s,
	}
	s.clients[client.ID] = client
	s.mu.Unlock()

	s.gateway.OnConnection(client)

	go s.readLoop(client)
}

// Broadcast sends a message to all connected clients.
func (s *Server) Broadcast(event string, data any) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, client := range s.clients {
		_ = client.Send(event, data)
	}
}

// ClientCount returns the number of connected clients.
func (s *Server) ClientCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.clients)
}

func (s *Server) readLoop(client *Client) {
	defer func() {
		s.mu.Lock()
		delete(s.clients, client.ID)
		s.mu.Unlock()
		s.gateway.OnDisconnect(client)
		client.Close()
	}()

	handlers := s.gateway.Handlers()

	for {
		_, data, err := client.conn.ReadMessage()
		if err != nil {
			break
		}

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}

		handler, ok := handlers[msg.Event]
		if !ok {
			_ = client.Send("error", map[string]string{"message": "unknown event: " + msg.Event})
			continue
		}

		result, err := handler(client, msg.Data)
		if err != nil {
			_ = client.Send("error", map[string]string{"message": err.Error()})
			continue
		}

		if result != nil {
			_ = client.Send(msg.Event, result)
		}
	}
}

func itoa(n int) string {
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
