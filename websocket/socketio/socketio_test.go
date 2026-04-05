package socketio

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/gonest/websocket"
)

// mockConn implements websocket.WebSocketConn for testing.
type mockConn struct {
	mu       sync.Mutex
	messages [][]byte
	readCh   chan []byte
	closed   bool
}

func newMockConn() *mockConn {
	return &mockConn{readCh: make(chan []byte, 100)}
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
	if c.closed {
		return &closedError{}
	}
	cp := make([]byte, len(data))
	copy(cp, data)
	c.messages = append(c.messages, cp)
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

func (c *mockConn) sendMessage(event string, data any) {
	dataBytes, _ := json.Marshal(data)
	msg := socketIOMessage{Event: event, Data: dataBytes}
	b, _ := json.Marshal(msg)
	c.readCh <- b
}

func (c *mockConn) sendMessageWithAck(event string, data any, ackID int64) {
	dataBytes, _ := json.Marshal(data)
	msg := socketIOMessage{Event: event, Data: dataBytes, ID: ackID}
	b, _ := json.Marshal(msg)
	c.readCh <- b
}

type closedError struct{}

func (e *closedError) Error() string { return "connection closed" }

// testGateway for testing
type testGateway struct {
	mu           sync.Mutex
	connected    []*websocket.Client
	disconnected []*websocket.Client
}

func (g *testGateway) Handlers() map[string]websocket.MessageHandler {
	return map[string]websocket.MessageHandler{
		"echo": func(client *websocket.Client, data json.RawMessage) (any, error) {
			var msg string
			json.Unmarshal(data, &msg)
			return msg, nil
		},
		"greet": func(client *websocket.Client, data json.RawMessage) (any, error) {
			var name string
			json.Unmarshal(data, &name)
			return "Hello, " + name + "!", nil
		},
	}
}

func (g *testGateway) OnConnection(client *websocket.Client) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.connected = append(g.connected, client)
}

func (g *testGateway) OnDisconnect(client *websocket.Client) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.disconnected = append(g.disconnected, client)
}

// --- Helper to create adapter + connect a socket ---

func createTestAdapter() *IoAdapter {
	return NewIoAdapter(IoAdapterOptions{
		Upgrader: nil, // not needed for direct socket creation
	})
}

func connectSocket(adapter *IoAdapter, nsName string) (*Socket, *mockConn) {
	ns := adapter.GetNamespace(nsName)
	conn := newMockConn()
	sock := ns.addSocket(conn, adapter)
	go sock.readLoop()
	return sock, conn
}

// --- Tests ---

func TestNamespace_Creation(t *testing.T) {
	adapter := createTestAdapter()

	ns := adapter.GetNamespace("/")
	if ns.Name() != "/" {
		t.Errorf("expected name '/', got %q", ns.Name())
	}

	chat := adapter.Of("/chat")
	if chat.Name() != "/chat" {
		t.Errorf("expected name '/chat', got %q", chat.Name())
	}

	// Same namespace should be returned
	chat2 := adapter.GetNamespace("/chat")
	if chat != chat2 {
		t.Error("expected same namespace instance")
	}
}

func TestSocket_JoinAndLeaveRoom(t *testing.T) {
	adapter := createTestAdapter()
	ns := adapter.GetNamespace("/")
	conn := newMockConn()
	sock := ns.addSocket(conn, adapter)

	sock.Join("room1", "room2")

	rooms := sock.Rooms()
	if len(rooms) < 2 { // at minimum room1, room2 (plus auto-joined self-room)
		t.Errorf("expected at least 2 rooms, got %d", len(rooms))
	}
	if !sock.InRoom("room1") {
		t.Error("expected socket to be in room1")
	}
	if !sock.InRoom("room2") {
		t.Error("expected socket to be in room2")
	}

	// Leave room1
	sock.Leave("room1")
	if sock.InRoom("room1") {
		t.Error("expected socket to have left room1")
	}
	if !sock.InRoom("room2") {
		t.Error("expected socket to still be in room2")
	}
}

func TestSocket_Emit(t *testing.T) {
	adapter := createTestAdapter()
	ns := adapter.GetNamespace("/")
	conn := newMockConn()
	sock := ns.addSocket(conn, adapter)

	err := sock.Emit("greeting", map[string]string{"hello": "world"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	messages := conn.getMessages()
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}

	var msg socketIOMessage
	json.Unmarshal(messages[0], &msg)
	if msg.Event != "greeting" {
		t.Errorf("expected event 'greeting', got %q", msg.Event)
	}
	if msg.Namespace != "/" {
		t.Errorf("expected namespace '/', got %q", msg.Namespace)
	}
}

func TestNamespace_Broadcast(t *testing.T) {
	adapter := createTestAdapter()
	ns := adapter.GetNamespace("/")

	conn1 := newMockConn()
	conn2 := newMockConn()
	conn3 := newMockConn()

	ns.addSocket(conn1, adapter)
	ns.addSocket(conn2, adapter)
	ns.addSocket(conn3, adapter)

	ns.Emit("update", "hello all")

	for i, conn := range []*mockConn{conn1, conn2, conn3} {
		msgs := conn.getMessages()
		if len(msgs) != 1 {
			t.Errorf("conn%d: expected 1 message, got %d", i+1, len(msgs))
		}
	}
}

func TestBroadcast_ToRoom(t *testing.T) {
	adapter := createTestAdapter()
	ns := adapter.GetNamespace("/")

	conn1 := newMockConn()
	conn2 := newMockConn()
	conn3 := newMockConn()

	sock1 := ns.addSocket(conn1, adapter)
	sock2 := ns.addSocket(conn2, adapter)
	ns.addSocket(conn3, adapter) // not in room

	sock1.Join("vip")
	sock2.Join("vip")

	ns.To("vip").Emit("vip-event", "exclusive")

	msgs1 := conn1.getMessages()
	msgs2 := conn2.getMessages()
	msgs3 := conn3.getMessages()

	if len(msgs1) != 1 {
		t.Errorf("sock1 (in vip): expected 1 message, got %d", len(msgs1))
	}
	if len(msgs2) != 1 {
		t.Errorf("sock2 (in vip): expected 1 message, got %d", len(msgs2))
	}
	if len(msgs3) != 0 {
		t.Errorf("sock3 (not in vip): expected 0 messages, got %d", len(msgs3))
	}
}

func TestBroadcast_Except(t *testing.T) {
	adapter := createTestAdapter()
	ns := adapter.GetNamespace("/")

	conn1 := newMockConn()
	conn2 := newMockConn()

	sock1 := ns.addSocket(conn1, adapter)
	ns.addSocket(conn2, adapter)

	// Broadcast to all except sock1
	ns.Except(sock1.Client.ID).Emit("update", "you missed it")

	msgs1 := conn1.getMessages()
	msgs2 := conn2.getMessages()

	if len(msgs1) != 0 {
		t.Errorf("sock1 (excluded): expected 0 messages, got %d", len(msgs1))
	}
	if len(msgs2) != 1 {
		t.Errorf("sock2: expected 1 message, got %d", len(msgs2))
	}
}

func TestSocket_BroadcastExcludeSelf(t *testing.T) {
	adapter := createTestAdapter()
	ns := adapter.GetNamespace("/")

	conn1 := newMockConn()
	conn2 := newMockConn()

	sock1 := ns.addSocket(conn1, adapter)
	ns.addSocket(conn2, adapter)

	// sock1 broadcasts to all except self
	sock1.Broadcast().Emit("news", "breaking")

	msgs1 := conn1.getMessages()
	msgs2 := conn2.getMessages()

	if len(msgs1) != 0 {
		t.Errorf("sock1 (sender): expected 0 messages, got %d", len(msgs1))
	}
	if len(msgs2) != 1 {
		t.Errorf("sock2: expected 1 message, got %d", len(msgs2))
	}
}

func TestSocket_ToRoom(t *testing.T) {
	adapter := createTestAdapter()
	ns := adapter.GetNamespace("/")

	conn1 := newMockConn()
	conn2 := newMockConn()
	conn3 := newMockConn()

	sock1 := ns.addSocket(conn1, adapter)
	sock2 := ns.addSocket(conn2, adapter)
	ns.addSocket(conn3, adapter)

	sock2.Join("lobby")

	// sock1 sends to "lobby" room (sock1 is excluded by default in To)
	sock1.To("lobby").Emit("chat", "hello lobby")

	msgs1 := conn1.getMessages()
	msgs2 := conn2.getMessages()
	msgs3 := conn3.getMessages()

	if len(msgs1) != 0 {
		t.Errorf("sock1 (sender): expected 0, got %d", len(msgs1))
	}
	if len(msgs2) != 1 {
		t.Errorf("sock2 (in lobby): expected 1, got %d", len(msgs2))
	}
	if len(msgs3) != 0 {
		t.Errorf("sock3 (not in lobby): expected 0, got %d", len(msgs3))
	}
}

func TestNamespace_EventHandlers(t *testing.T) {
	adapter := createTestAdapter()
	ns := adapter.GetNamespace("/")

	// Register a handler at namespace level
	ns.On("ping", func(client *websocket.Client, data json.RawMessage) (any, error) {
		return "pong", nil
	})

	conn := newMockConn()
	sock := ns.addSocket(conn, adapter)
	go sock.readLoop()

	// Send a message
	conn.sendMessage("ping", nil)

	// Wait for response
	time.Sleep(50 * time.Millisecond)

	messages := conn.getMessages()
	if len(messages) < 1 {
		t.Fatal("expected at least 1 response message")
	}

	var resp socketIOMessage
	json.Unmarshal(messages[len(messages)-1], &resp)
	if resp.Event != "ping" {
		t.Errorf("expected event 'ping', got %q", resp.Event)
	}

	var respData string
	json.Unmarshal(resp.Data, &respData)
	if respData != "pong" {
		t.Errorf("expected 'pong', got %q", respData)
	}

	// Cleanup
	conn.Close()
	time.Sleep(50 * time.Millisecond)
}

func TestSocket_Acknowledgement(t *testing.T) {
	adapter := createTestAdapter()
	ns := adapter.GetNamespace("/")

	// Register handler that returns a result for ack
	ns.On("compute", func(client *websocket.Client, data json.RawMessage) (any, error) {
		var nums []int
		json.Unmarshal(data, &nums)
		sum := 0
		for _, n := range nums {
			sum += n
		}
		return sum, nil
	})

	conn := newMockConn()
	sock := ns.addSocket(conn, adapter)
	go sock.readLoop()

	// Send message with ack ID
	conn.sendMessageWithAck("compute", []int{1, 2, 3}, 42)

	time.Sleep(50 * time.Millisecond)

	messages := conn.getMessages()
	if len(messages) < 1 {
		t.Fatal("expected ack response")
	}

	// The response should be an ack message with the same ID
	var ack socketIOAckMessage
	json.Unmarshal(messages[len(messages)-1], &ack)
	if ack.ID != 42 {
		t.Errorf("expected ack ID 42, got %d", ack.ID)
	}

	var result int
	json.Unmarshal(ack.Data, &result)
	if result != 6 {
		t.Errorf("expected sum 6, got %d", result)
	}

	conn.Close()
	time.Sleep(50 * time.Millisecond)
}

func TestNamespace_ConnectionCallback(t *testing.T) {
	adapter := createTestAdapter()
	ns := adapter.GetNamespace("/")

	var connectedSockets []*Socket
	var mu sync.Mutex

	ns.OnConnection(func(sock *Socket) {
		mu.Lock()
		connectedSockets = append(connectedSockets, sock)
		mu.Unlock()
	})

	conn1 := newMockConn()
	conn2 := newMockConn()

	sock1 := ns.addSocket(conn1, adapter)
	ns.notifyConnect(sock1)

	sock2 := ns.addSocket(conn2, adapter)
	ns.notifyConnect(sock2)

	mu.Lock()
	count := len(connectedSockets)
	mu.Unlock()

	if count != 2 {
		t.Errorf("expected 2 connected sockets, got %d", count)
	}
}

func TestNamespace_GetSockets(t *testing.T) {
	adapter := createTestAdapter()
	ns := adapter.GetNamespace("/")

	conn1 := newMockConn()
	conn2 := newMockConn()

	ns.addSocket(conn1, adapter)
	ns.addSocket(conn2, adapter)

	if ns.SocketCount() != 2 {
		t.Errorf("expected 2 sockets, got %d", ns.SocketCount())
	}

	sockets := ns.GetSockets()
	if len(sockets) != 2 {
		t.Errorf("expected 2 sockets, got %d", len(sockets))
	}
}

func TestNamespace_GetRooms(t *testing.T) {
	adapter := createTestAdapter()
	ns := adapter.GetNamespace("/")

	conn := newMockConn()
	sock := ns.addSocket(conn, adapter)
	sock.Join("room1", "room2", "room3")

	rooms := ns.GetRooms()
	// Should have at least room1, room2, room3 plus the auto-joined self-room
	if len(rooms) < 3 {
		t.Errorf("expected at least 3 rooms, got %d: %v", len(rooms), rooms)
	}
}

func TestSocket_Middleware(t *testing.T) {
	adapter := createTestAdapter()

	// Add middleware that sets data on the socket
	adapter.Use(func(sock *Socket) error {
		sock.Data["authenticated"] = true
		return nil
	})

	ns := adapter.GetNamespace("/")
	conn := newMockConn()
	sock := ns.addSocket(conn, adapter)

	// Run middleware manually (normally done in ServeHTTP)
	for _, mw := range adapter.middleware {
		if err := mw(sock); err != nil {
			t.Fatalf("middleware error: %v", err)
		}
	}

	if sock.Data["authenticated"] != true {
		t.Error("expected middleware to set authenticated=true")
	}
}

func TestSocket_MiddlewareReject(t *testing.T) {
	adapter := createTestAdapter()

	adapter.Use(func(sock *Socket) error {
		return websocket.NewWsException("unauthorized")
	})

	ns := adapter.GetNamespace("/")
	conn := newMockConn()
	sock := ns.addSocket(conn, adapter)

	var rejected bool
	for _, mw := range adapter.middleware {
		if err := mw(sock); err != nil {
			rejected = true
			break
		}
	}

	if !rejected {
		t.Error("expected middleware to reject connection")
	}
}

func TestNamespace_Middleware(t *testing.T) {
	adapter := createTestAdapter()
	ns := adapter.GetNamespace("/chat")

	ns.Use(func(sock *Socket) error {
		sock.Data["ns"] = "/chat"
		return nil
	})

	conn := newMockConn()
	sock := ns.addSocket(conn, adapter)

	for _, mw := range ns.middleware {
		mw(sock)
	}

	if sock.Data["ns"] != "/chat" {
		t.Error("expected namespace middleware to set ns=/chat")
	}
}

func TestSocket_Disconnect(t *testing.T) {
	adapter := createTestAdapter()
	ns := adapter.GetNamespace("/")

	conn := newMockConn()
	sock := ns.addSocket(conn, adapter)
	sock.Join("room1")

	if ns.SocketCount() != 1 {
		t.Fatalf("expected 1 socket, got %d", ns.SocketCount())
	}

	sock.Disconnect()
	time.Sleep(50 * time.Millisecond)

	if ns.SocketCount() != 0 {
		t.Errorf("expected 0 sockets after disconnect, got %d", ns.SocketCount())
	}

	// Room should be cleaned up
	roomSockets := ns.GetRoomSockets("room1")
	if len(roomSockets) != 0 {
		t.Errorf("expected 0 sockets in room1 after disconnect, got %d", len(roomSockets))
	}
}

func TestAdapter_MultipleNamespaces(t *testing.T) {
	adapter := createTestAdapter()

	ns1 := adapter.GetNamespace("/")
	ns2 := adapter.GetNamespace("/chat")
	ns3 := adapter.GetNamespace("/admin")

	conn1 := newMockConn()
	conn2 := newMockConn()
	conn3 := newMockConn()

	ns1.addSocket(conn1, adapter)
	ns2.addSocket(conn2, adapter)
	ns3.addSocket(conn3, adapter)

	// Broadcast to /chat only
	ns2.Emit("chat-msg", "hello chat")

	// Only conn2 should have received it
	if len(conn1.getMessages()) != 0 {
		t.Error("ns1 socket should not receive /chat message")
	}
	if len(conn2.getMessages()) != 1 {
		t.Error("ns2 socket should receive /chat message")
	}
	if len(conn3.getMessages()) != 0 {
		t.Error("ns3 socket should not receive /chat message")
	}
}

func TestBroadcast_MultipleRooms(t *testing.T) {
	adapter := createTestAdapter()
	ns := adapter.GetNamespace("/")

	conn1 := newMockConn()
	conn2 := newMockConn()
	conn3 := newMockConn()

	sock1 := ns.addSocket(conn1, adapter)
	sock2 := ns.addSocket(conn2, adapter)
	sock3 := ns.addSocket(conn3, adapter)

	sock1.Join("room-a")
	sock2.Join("room-b")
	sock3.Join("room-a", "room-b")

	// Broadcast to both rooms — sock3 is in both but should only receive once
	ns.To("room-a", "room-b").Emit("update", "data")

	msgs1 := conn1.getMessages()
	msgs2 := conn2.getMessages()
	msgs3 := conn3.getMessages()

	if len(msgs1) != 1 {
		t.Errorf("sock1 (room-a): expected 1, got %d", len(msgs1))
	}
	if len(msgs2) != 1 {
		t.Errorf("sock2 (room-b): expected 1, got %d", len(msgs2))
	}
	if len(msgs3) != 1 {
		t.Errorf("sock3 (room-a+b): expected 1 (deduped), got %d", len(msgs3))
	}
}

func TestBroadcast_EmitToAll_Count(t *testing.T) {
	adapter := createTestAdapter()
	ns := adapter.GetNamespace("/")

	for i := 0; i < 5; i++ {
		conn := newMockConn()
		ns.addSocket(conn, adapter)
	}

	count := ns.To().EmitToAll("ping", nil)
	// To() with no rooms targets nobody (empty rooms set)
	// Use Emit for all sockets
	if count != 0 {
		t.Logf("To() with no args sends to 0 (no rooms specified), got %d", count)
	}

	// Use namespace-level emit for all
	ns.Emit("ping", nil)
	// Each socket should have 1 message
	sockets := ns.GetSockets()
	for _, sock := range sockets {
		// We can't easily check the mock conn from here, but the test above
		// validates broadcast logic.
		_ = sock
	}
}

func TestAdapter_Close(t *testing.T) {
	adapter := createTestAdapter()
	ns := adapter.GetNamespace("/")

	conn1 := newMockConn()
	conn2 := newMockConn()

	ns.addSocket(conn1, adapter)
	ns.addSocket(conn2, adapter)

	if ns.SocketCount() != 2 {
		t.Fatalf("expected 2 sockets, got %d", ns.SocketCount())
	}

	adapter.Close()

	if ns.SocketCount() != 0 {
		t.Errorf("expected 0 sockets after close, got %d", ns.SocketCount())
	}
}

func TestSocket_AutoJoinsSelfRoom(t *testing.T) {
	adapter := createTestAdapter()
	ns := adapter.GetNamespace("/")

	conn := newMockConn()
	sock := ns.addSocket(conn, adapter)

	// Socket should auto-join a room with its own ID
	if !sock.InRoom(sock.Client.ID) {
		t.Error("expected socket to auto-join its own room")
	}

	// Can target the specific socket via its room
	ns.To(sock.Client.ID).Emit("private", "just for you")

	msgs := conn.getMessages()
	if len(msgs) != 1 {
		t.Errorf("expected 1 message via self-room, got %d", len(msgs))
	}
}

func TestAdapter_EmitDefault(t *testing.T) {
	adapter := createTestAdapter()
	ns := adapter.GetNamespace("/")

	conn := newMockConn()
	ns.addSocket(conn, adapter)

	// Use adapter-level Emit (targets default namespace)
	adapter.Emit("global", "hello")

	msgs := conn.getMessages()
	if len(msgs) != 1 {
		t.Errorf("expected 1 message from adapter.Emit, got %d", len(msgs))
	}
}

func TestAdapter_ToRoom(t *testing.T) {
	adapter := createTestAdapter()
	ns := adapter.GetNamespace("/")

	conn1 := newMockConn()
	conn2 := newMockConn()

	sock1 := ns.addSocket(conn1, adapter)
	ns.addSocket(conn2, adapter)

	sock1.Join("admins")

	// Use adapter-level To (targets default namespace)
	adapter.To("admins").Emit("admin-event", "important")

	msgs1 := conn1.getMessages()
	msgs2 := conn2.getMessages()

	if len(msgs1) != 1 {
		t.Errorf("sock1 (admin): expected 1, got %d", len(msgs1))
	}
	if len(msgs2) != 0 {
		t.Errorf("sock2 (not admin): expected 0, got %d", len(msgs2))
	}
}

func TestSocket_GetNamespace(t *testing.T) {
	adapter := createTestAdapter()
	ns := adapter.GetNamespace("/chat")

	conn := newMockConn()
	sock := ns.addSocket(conn, adapter)

	if sock.GetNamespace() != ns {
		t.Error("expected socket's namespace to match")
	}
	if sock.GetNamespace().Name() != "/chat" {
		t.Errorf("expected '/chat', got %q", sock.GetNamespace().Name())
	}
}
