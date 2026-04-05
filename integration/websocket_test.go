package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/0xfurai/gonest"
	"github.com/0xfurai/gonest/websocket"
)

// ---------------------------------------------------------------------------
// WebSocket Integration Tests
// Mirror: original/integration/websockets/
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Mock WebSocket connection for testing
// ---------------------------------------------------------------------------

type mockWSConn struct {
	mu       sync.Mutex
	inbox    chan []byte
	outbox   [][]byte
	closed   bool
}

func newMockWSConn() *mockWSConn {
	return &mockWSConn{
		inbox: make(chan []byte, 100),
	}
}

func (c *mockWSConn) ReadMessage() (int, []byte, error) {
	msg, ok := <-c.inbox
	if !ok {
		return 0, nil, &connClosedErr{}
	}
	return 1, msg, nil
}

func (c *mockWSConn) WriteMessage(msgType int, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.outbox = append(c.outbox, append([]byte{}, data...))
	return nil
}

func (c *mockWSConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		c.closed = true
		close(c.inbox)
	}
	return nil
}

func (c *mockWSConn) getOutbox() [][]byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([][]byte, len(c.outbox))
	copy(out, c.outbox)
	return out
}

type connClosedErr struct{}

func (e *connClosedErr) Error() string { return "connection closed" }

// ---------------------------------------------------------------------------
// Mock upgrader that captures connections
// ---------------------------------------------------------------------------

type mockUpgrader struct {
	mu    sync.Mutex
	conns []*mockWSConn
}

func newMockUpgrader() *mockUpgrader {
	return &mockUpgrader{}
}

func (u *mockUpgrader) Upgrade(w http.ResponseWriter, r *http.Request) (websocket.WebSocketConn, error) {
	conn := newMockWSConn()
	u.mu.Lock()
	u.conns = append(u.conns, conn)
	u.mu.Unlock()
	return conn, nil
}

func (u *mockUpgrader) lastConn() *mockWSConn {
	u.mu.Lock()
	defer u.mu.Unlock()
	if len(u.conns) == 0 {
		return nil
	}
	return u.conns[len(u.conns)-1]
}

// ---------------------------------------------------------------------------
// Chat gateway
// ---------------------------------------------------------------------------

type chatGateway struct {
	mu          sync.Mutex
	connections int
}

func (g *chatGateway) Handlers() map[string]websocket.MessageHandler {
	return map[string]websocket.MessageHandler{
		"message": g.handleMessage,
		"ping":    g.handlePing,
	}
}

func (g *chatGateway) handleMessage(client *websocket.Client, data json.RawMessage) (any, error) {
	var msg string
	json.Unmarshal(data, &msg)
	return map[string]string{"echo": msg}, nil
}

func (g *chatGateway) handlePing(client *websocket.Client, data json.RawMessage) (any, error) {
	return "pong", nil
}

func (g *chatGateway) OnConnection(client *websocket.Client) {
	g.mu.Lock()
	g.connections++
	g.mu.Unlock()
}

func (g *chatGateway) OnDisconnect(client *websocket.Client) {
	g.mu.Lock()
	g.connections--
	g.mu.Unlock()
}

// ---------------------------------------------------------------------------
// Tests: Gateway via Server.ServeHTTP
// ---------------------------------------------------------------------------

func TestWebSocket_ConnectionViaServeHTTP(t *testing.T) {
	gateway := &chatGateway{}
	upgrader := newMockUpgrader()
	server := websocket.NewServer(gateway, upgrader)

	// Simulate HTTP upgrade request
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ws", nil)
	server.ServeHTTP(w, req)

	time.Sleep(50 * time.Millisecond)

	if server.ClientCount() != 1 {
		t.Fatalf("expected 1 client, got %d", server.ClientCount())
	}

	gateway.mu.Lock()
	conns := gateway.connections
	gateway.mu.Unlock()
	if conns != 1 {
		t.Errorf("expected OnConnection called once, got %d", conns)
	}

	// Send message through the mock connection
	conn := upgrader.lastConn()
	if conn == nil {
		t.Fatal("no connection captured by upgrader")
	}

	msg, _ := json.Marshal(websocket.Message{
		Event: "message",
		Data:  json.RawMessage(`"hello"`),
	})
	conn.inbox <- msg
	time.Sleep(50 * time.Millisecond)

	outbox := conn.getOutbox()
	if len(outbox) == 0 {
		t.Fatal("expected response message")
	}

	var response websocket.OutgoingMessage
	json.Unmarshal(outbox[0], &response)
	if response.Event != "message" {
		t.Errorf("expected event=message, got %q", response.Event)
	}

	// Clean up
	conn.Close()
	time.Sleep(100 * time.Millisecond)
}

func TestWebSocket_PingPong(t *testing.T) {
	gateway := &chatGateway{}
	upgrader := newMockUpgrader()
	server := websocket.NewServer(gateway, upgrader)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ws", nil)
	server.ServeHTTP(w, req)
	time.Sleep(50 * time.Millisecond)

	conn := upgrader.lastConn()
	msg, _ := json.Marshal(websocket.Message{
		Event: "ping",
		Data:  json.RawMessage(`null`),
	})
	conn.inbox <- msg
	time.Sleep(50 * time.Millisecond)

	outbox := conn.getOutbox()
	if len(outbox) == 0 {
		t.Fatal("expected pong response")
	}

	var response websocket.OutgoingMessage
	json.Unmarshal(outbox[0], &response)
	if response.Event != "ping" {
		t.Errorf("expected event=ping, got %q", response.Event)
	}

	conn.Close()
	time.Sleep(50 * time.Millisecond)
}

func TestWebSocket_UnknownEvent(t *testing.T) {
	gateway := &chatGateway{}
	upgrader := newMockUpgrader()
	server := websocket.NewServer(gateway, upgrader)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ws", nil)
	server.ServeHTTP(w, req)
	time.Sleep(50 * time.Millisecond)

	conn := upgrader.lastConn()
	msg, _ := json.Marshal(websocket.Message{
		Event: "unknown_event",
		Data:  json.RawMessage(`null`),
	})
	conn.inbox <- msg
	time.Sleep(50 * time.Millisecond)

	outbox := conn.getOutbox()
	if len(outbox) == 0 {
		t.Fatal("expected error response for unknown event")
	}

	var response websocket.OutgoingMessage
	json.Unmarshal(outbox[0], &response)
	if response.Event != "error" {
		t.Errorf("expected event=error, got %q", response.Event)
	}

	conn.Close()
	time.Sleep(50 * time.Millisecond)
}

func TestWebSocket_MultipleClients(t *testing.T) {
	gateway := &chatGateway{}
	upgrader := newMockUpgrader()
	server := websocket.NewServer(gateway, upgrader)

	if server.ClientCount() != 0 {
		t.Fatalf("expected 0 clients, got %d", server.ClientCount())
	}

	// Connect client 1
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("GET", "/ws", nil)
	server.ServeHTTP(w1, req1)
	time.Sleep(50 * time.Millisecond)

	if server.ClientCount() != 1 {
		t.Fatalf("expected 1 client, got %d", server.ClientCount())
	}

	// Connect client 2
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/ws", nil)
	server.ServeHTTP(w2, req2)
	time.Sleep(50 * time.Millisecond)

	if server.ClientCount() != 2 {
		t.Fatalf("expected 2 clients, got %d", server.ClientCount())
	}

	// Disconnect client 1
	upgrader.mu.Lock()
	conn1 := upgrader.conns[0]
	upgrader.mu.Unlock()
	conn1.Close()
	time.Sleep(100 * time.Millisecond)

	if server.ClientCount() != 1 {
		t.Fatalf("expected 1 client after disconnect, got %d", server.ClientCount())
	}

	// Clean up
	upgrader.mu.Lock()
	conn2 := upgrader.conns[1]
	upgrader.mu.Unlock()
	conn2.Close()
	time.Sleep(100 * time.Millisecond)
}

func TestWebSocket_Broadcast(t *testing.T) {
	gateway := &chatGateway{}
	upgrader := newMockUpgrader()
	server := websocket.NewServer(gateway, upgrader)

	// Connect two clients
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/ws", nil)
		server.ServeHTTP(w, req)
	}
	time.Sleep(50 * time.Millisecond)

	// Broadcast to all
	server.Broadcast("announcement", "hello all")
	time.Sleep(50 * time.Millisecond)

	upgrader.mu.Lock()
	conns := make([]*mockWSConn, len(upgrader.conns))
	copy(conns, upgrader.conns)
	upgrader.mu.Unlock()

	for i, conn := range conns {
		outbox := conn.getOutbox()
		if len(outbox) == 0 {
			t.Errorf("client %d: expected broadcast message", i)
			continue
		}
		var msg websocket.OutgoingMessage
		json.Unmarshal(outbox[len(outbox)-1], &msg)
		if msg.Event != "announcement" {
			t.Errorf("client %d: expected event=announcement, got %q", i, msg.Event)
		}
	}

	// Cleanup
	for _, conn := range conns {
		conn.Close()
	}
	time.Sleep(50 * time.Millisecond)
}

func TestWebSocket_MultipleMessages(t *testing.T) {
	gateway := &chatGateway{}
	upgrader := newMockUpgrader()
	server := websocket.NewServer(gateway, upgrader)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ws", nil)
	server.ServeHTTP(w, req)
	time.Sleep(50 * time.Millisecond)

	conn := upgrader.lastConn()
	messages := []string{"hello", "world", "test"}
	for _, m := range messages {
		msg, _ := json.Marshal(websocket.Message{
			Event: "message",
			Data:  json.RawMessage(`"` + m + `"`),
		})
		conn.inbox <- msg
	}
	time.Sleep(100 * time.Millisecond)

	outbox := conn.getOutbox()
	if len(outbox) != 3 {
		t.Fatalf("expected 3 responses, got %d", len(outbox))
	}

	conn.Close()
	time.Sleep(50 * time.Millisecond)
}

// ---------------------------------------------------------------------------
// Tests: WebSocket Module integration with HTTP app
// ---------------------------------------------------------------------------

func TestWebSocket_Module_WithApp(t *testing.T) {
	gateway := &chatGateway{}
	upgrader := newMockUpgrader()

	wsModule := websocket.NewModule(websocket.GatewayOptions{
		Path:     "/ws",
		Gateway:  gateway,
		Upgrader: upgrader,
	})

	appMod := gonest.NewModule(gonest.ModuleOptions{
		Imports: []*gonest.Module{wsModule},
	})

	app := gonest.Create(appMod, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	// WebSocket upgrade via HTTP route
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ws", nil)
	app.Handler().ServeHTTP(w, req)
	time.Sleep(50 * time.Millisecond)

	conn := upgrader.lastConn()
	if conn == nil {
		t.Fatal("expected mock connection from upgrader")
	}

	// Send a message
	msg, _ := json.Marshal(websocket.Message{
		Event: "message",
		Data:  json.RawMessage(`"from-app"`),
	})
	conn.inbox <- msg
	time.Sleep(50 * time.Millisecond)

	outbox := conn.getOutbox()
	if len(outbox) == 0 {
		t.Fatal("expected response")
	}

	var response websocket.OutgoingMessage
	json.Unmarshal(outbox[0], &response)
	if response.Event != "message" {
		t.Errorf("expected event=message, got %q", response.Event)
	}

	conn.Close()
	time.Sleep(50 * time.Millisecond)
}

// ---------------------------------------------------------------------------
// Tests: BroadcastInterceptor
// ---------------------------------------------------------------------------

func TestWebSocket_BroadcastInterceptor(t *testing.T) {
	gateway := &chatGateway{}
	upgrader := newMockUpgrader()
	server := websocket.NewServer(gateway, upgrader)

	// Connect a client
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ws", nil)
	server.ServeHTTP(w, req)
	time.Sleep(50 * time.Millisecond)

	conn := upgrader.lastConn()

	interceptor := &websocket.BroadcastInterceptor{
		Server: server,
		Event:  "update",
	}

	// Simulate interceptor intercepting a handler result
	called := false
	handler := gonest.NewCallHandler(func() (any, error) {
		called = true
		return map[string]string{"data": "broadcast"}, nil
	})

	_, err := interceptor.Intercept(nil, handler)
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("handler not called")
	}

	time.Sleep(50 * time.Millisecond)

	outbox := conn.getOutbox()
	if len(outbox) == 0 {
		t.Fatal("expected broadcast from interceptor")
	}

	var msg websocket.OutgoingMessage
	json.Unmarshal(outbox[len(outbox)-1], &msg)
	if msg.Event != "update" {
		t.Errorf("expected event=update, got %q", msg.Event)
	}

	conn.Close()
	time.Sleep(50 * time.Millisecond)
}
