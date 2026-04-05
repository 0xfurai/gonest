package websocket

import (
	"encoding/json"
	"sync"
	"testing"
)

// mockConn implements WebSocketConn for testing
type mockConn struct {
	mu       sync.Mutex
	messages [][]byte
	readCh   chan []byte
	closed   bool
}

func newMockConn() *mockConn {
	return &mockConn{readCh: make(chan []byte, 10)}
}

func (c *mockConn) ReadMessage() (int, []byte, error) {
	data, ok := <-c.readCh
	if !ok {
		return 0, nil, &closedError{}
	}
	return 1, data, nil
}

func (c *mockConn) WriteMessage(messageType int, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.messages = append(c.messages, data)
	return nil
}

func (c *mockConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		c.closed = true
		close(c.readCh)
	}
	return nil
}

func (c *mockConn) getMessages() [][]byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	cp := make([][]byte, len(c.messages))
	copy(cp, c.messages)
	return cp
}

type closedError struct{}

func (e *closedError) Error() string { return "connection closed" }

// testGateway for testing
type testGateway struct {
	connected    []*Client
	disconnected []*Client
	mu           sync.Mutex
}

func (g *testGateway) Handlers() map[string]MessageHandler {
	return map[string]MessageHandler{
		"echo": func(client *Client, data json.RawMessage) (any, error) {
			var msg string
			json.Unmarshal(data, &msg)
			return msg, nil
		},
		"sum": func(client *Client, data json.RawMessage) (any, error) {
			var nums []int
			json.Unmarshal(data, &nums)
			sum := 0
			for _, n := range nums {
				sum += n
			}
			return sum, nil
		},
	}
}

func (g *testGateway) OnConnection(client *Client) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.connected = append(g.connected, client)
}

func (g *testGateway) OnDisconnect(client *Client) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.disconnected = append(g.disconnected, client)
}

func TestClient_Send(t *testing.T) {
	conn := newMockConn()
	client := &Client{ID: "1", conn: conn}

	err := client.Send("greeting", map[string]string{"hello": "world"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	messages := conn.getMessages()
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}

	var msg OutgoingMessage
	json.Unmarshal(messages[0], &msg)
	if msg.Event != "greeting" {
		t.Errorf("expected event 'greeting', got %q", msg.Event)
	}
}

func TestClient_Close(t *testing.T) {
	conn := newMockConn()
	client := &Client{ID: "1", conn: conn}

	client.Close()
	if !client.closed {
		t.Error("expected client to be closed")
	}

	// Send after close should not panic
	err := client.Send("test", nil)
	if err != nil {
		t.Errorf("expected nil error after close, got %v", err)
	}
}

func TestServer_ClientCount(t *testing.T) {
	gw := &testGateway{}
	server := NewServer(gw, nil)
	if server.ClientCount() != 0 {
		t.Errorf("expected 0 clients, got %d", server.ClientCount())
	}
}

func TestServer_Broadcast(t *testing.T) {
	gw := &testGateway{}
	server := NewServer(gw, nil)

	conn1 := newMockConn()
	conn2 := newMockConn()

	server.mu.Lock()
	server.clients["1"] = &Client{ID: "1", conn: conn1, server: server}
	server.clients["2"] = &Client{ID: "2", conn: conn2, server: server}
	server.mu.Unlock()

	server.Broadcast("update", map[string]string{"status": "ok"})

	msgs1 := conn1.getMessages()
	msgs2 := conn2.getMessages()
	if len(msgs1) != 1 || len(msgs2) != 1 {
		t.Errorf("expected broadcast to reach both clients: %d, %d", len(msgs1), len(msgs2))
	}
}

func TestMessage_Serialization(t *testing.T) {
	msg := OutgoingMessage{Event: "test", Data: []int{1, 2, 3}}
	bytes, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}

	var decoded OutgoingMessage
	json.Unmarshal(bytes, &decoded)
	if decoded.Event != "test" {
		t.Errorf("expected event 'test', got %q", decoded.Event)
	}
}

func TestGateway_Handlers(t *testing.T) {
	gw := &testGateway{}
	handlers := gw.Handlers()

	if _, ok := handlers["echo"]; !ok {
		t.Error("expected echo handler")
	}
	if _, ok := handlers["sum"]; !ok {
		t.Error("expected sum handler")
	}

	// Test echo handler
	result, err := handlers["echo"](nil, json.RawMessage(`"hello"`))
	if err != nil {
		t.Fatal(err)
	}
	if result != "hello" {
		t.Errorf("expected 'hello', got %v", result)
	}

	// Test sum handler
	result, err = handlers["sum"](nil, json.RawMessage(`[1,2,3,4,5]`))
	if err != nil {
		t.Fatal(err)
	}
	if result != 15 {
		t.Errorf("expected 15, got %v", result)
	}
}
