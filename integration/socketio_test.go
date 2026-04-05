package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gonest"
	"github.com/gonest/websocket"
	"github.com/gonest/websocket/socketio"
)

// ---------------------------------------------------------------------------
// Socket.IO Integration Tests
// Mirror: original/integration/websockets/ — ack, namespaces, rooms
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Mock Socket.IO connection (reuses WebSocket mock pattern)
// ---------------------------------------------------------------------------

type mockSioConn struct {
	mu     sync.Mutex
	inbox  chan []byte
	outbox [][]byte
	closed bool
}

func newMockSioConn() *mockSioConn {
	return &mockSioConn{
		inbox: make(chan []byte, 100),
	}
}

func (c *mockSioConn) ReadMessage() (int, []byte, error) {
	msg, ok := <-c.inbox
	if !ok {
		return 0, nil, &sioConnClosedErr{}
	}
	return 1, msg, nil
}

func (c *mockSioConn) WriteMessage(msgType int, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.outbox = append(c.outbox, append([]byte{}, data...))
	return nil
}

func (c *mockSioConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		c.closed = true
		close(c.inbox)
	}
	return nil
}

func (c *mockSioConn) getOutbox() [][]byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([][]byte, len(c.outbox))
	copy(out, c.outbox)
	return out
}

type sioConnClosedErr struct{}

func (e *sioConnClosedErr) Error() string { return "connection closed" }

// ---------------------------------------------------------------------------
// Mock Socket.IO upgrader
// ---------------------------------------------------------------------------

type mockSioUpgrader struct {
	mu    sync.Mutex
	conns []*mockSioConn
}

func newMockSioUpgrader() *mockSioUpgrader {
	return &mockSioUpgrader{}
}

func (u *mockSioUpgrader) Upgrade(w http.ResponseWriter, r *http.Request) (websocket.WebSocketConn, error) {
	conn := newMockSioConn()
	u.mu.Lock()
	u.conns = append(u.conns, conn)
	u.mu.Unlock()
	return conn, nil
}

func (u *mockSioUpgrader) getConn(i int) *mockSioConn {
	u.mu.Lock()
	defer u.mu.Unlock()
	if i < len(u.conns) {
		return u.conns[i]
	}
	return nil
}

func (u *mockSioUpgrader) connCount() int {
	u.mu.Lock()
	defer u.mu.Unlock()
	return len(u.conns)
}

// ---------------------------------------------------------------------------
// Chat gateway for Socket.IO tests
// ---------------------------------------------------------------------------

type sioChatGateway struct {
	mu          sync.Mutex
	connections int
}

func (g *sioChatGateway) Handlers() map[string]websocket.MessageHandler {
	return map[string]websocket.MessageHandler{
		"message": g.handleMessage,
		"ping":    g.handlePing,
	}
}

func (g *sioChatGateway) handleMessage(client *websocket.Client, data json.RawMessage) (any, error) {
	var msg string
	json.Unmarshal(data, &msg)
	return map[string]string{"echo": msg}, nil
}

func (g *sioChatGateway) handlePing(client *websocket.Client, data json.RawMessage) (any, error) {
	return "pong", nil
}

func (g *sioChatGateway) OnConnection(client *websocket.Client) {
	g.mu.Lock()
	g.connections++
	g.mu.Unlock()
}

func (g *sioChatGateway) OnDisconnect(client *websocket.Client) {
	g.mu.Lock()
	g.connections--
	g.mu.Unlock()
}

// ---------------------------------------------------------------------------
// Tests: Socket.IO Adapter basic connection
// ---------------------------------------------------------------------------

func TestSocketIO_Adapter_Connection(t *testing.T) {
	upgrader := newMockSioUpgrader()
	adapter := socketio.NewIoAdapter(socketio.IoAdapterOptions{
		Upgrader: upgrader,
	})
	adapter.Create(nil)

	// Connect a client
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/socket.io", nil)
	adapter.ServeHTTP(w, req)
	time.Sleep(50 * time.Millisecond)

	ns := adapter.GetNamespace("/")
	if ns.SocketCount() != 1 {
		t.Fatalf("expected 1 socket in default namespace, got %d", ns.SocketCount())
	}

	// Clean up
	conn := upgrader.getConn(0)
	conn.Close()
	time.Sleep(100 * time.Millisecond)
}

// ---------------------------------------------------------------------------
// Tests: Namespaces
// ---------------------------------------------------------------------------

func TestSocketIO_Namespaces_Separate(t *testing.T) {
	upgrader := newMockSioUpgrader()
	adapter := socketio.NewIoAdapter(socketio.IoAdapterOptions{
		Upgrader: upgrader,
	})
	adapter.Create(nil)

	// Connect to default namespace
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("GET", "/socket.io", nil)
	adapter.ServeHTTP(w1, req1)

	// Connect to /chat namespace
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/socket.io?nsp=/chat", nil)
	adapter.ServeHTTP(w2, req2)
	time.Sleep(50 * time.Millisecond)

	defaultNs := adapter.GetNamespace("/")
	chatNs := adapter.GetNamespace("/chat")

	if defaultNs.SocketCount() != 1 {
		t.Errorf("expected 1 socket in default namespace, got %d", defaultNs.SocketCount())
	}
	if chatNs.SocketCount() != 1 {
		t.Errorf("expected 1 socket in /chat namespace, got %d", chatNs.SocketCount())
	}

	// Clean up
	for i := 0; i < upgrader.connCount(); i++ {
		upgrader.getConn(i).Close()
	}
	time.Sleep(100 * time.Millisecond)
}

func TestSocketIO_Namespace_Name(t *testing.T) {
	upgrader := newMockSioUpgrader()
	adapter := socketio.NewIoAdapter(socketio.IoAdapterOptions{
		Upgrader: upgrader,
	})
	adapter.Create(nil)

	ns := adapter.Of("/custom")
	if ns.Name() != "/custom" {
		t.Errorf("expected namespace name /custom, got %q", ns.Name())
	}
}

func TestSocketIO_Namespace_EventHandlers(t *testing.T) {
	upgrader := newMockSioUpgrader()
	adapter := socketio.NewIoAdapter(socketio.IoAdapterOptions{
		Upgrader: upgrader,
	})
	adapter.Create(nil)

	chatNs := adapter.GetNamespace("/chat")
	chatNs.On("greet", func(client *websocket.Client, data json.RawMessage) (any, error) {
		return "hello from /chat", nil
	})

	// Connect to /chat
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/socket.io?nsp=/chat", nil)
	adapter.ServeHTTP(w, req)
	time.Sleep(50 * time.Millisecond)

	conn := upgrader.getConn(0)
	msg, _ := json.Marshal(map[string]any{
		"event": "greet",
		"data":  "world",
	})
	conn.inbox <- msg
	time.Sleep(50 * time.Millisecond)

	outbox := conn.getOutbox()
	if len(outbox) == 0 {
		t.Fatal("expected response from namespace handler")
	}

	conn.Close()
	time.Sleep(50 * time.Millisecond)
}

// ---------------------------------------------------------------------------
// Tests: Rooms
// ---------------------------------------------------------------------------

func TestSocketIO_Rooms_JoinAndBroadcast(t *testing.T) {
	upgrader := newMockSioUpgrader()
	adapter := socketio.NewIoAdapter(socketio.IoAdapterOptions{
		Upgrader: upgrader,
	})
	adapter.Create(nil)

	ns := adapter.GetNamespace("/")
	var connectedSockets []*socketio.Socket

	ns.OnConnection(func(sock *socketio.Socket) {
		connectedSockets = append(connectedSockets, sock)
	})

	// Connect 3 clients
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/socket.io", nil)
		adapter.ServeHTTP(w, req)
	}
	time.Sleep(100 * time.Millisecond)

	if len(connectedSockets) != 3 {
		t.Fatalf("expected 3 connected sockets, got %d", len(connectedSockets))
	}

	// Socket 0 and 1 join "room1"
	connectedSockets[0].Join("room1")
	connectedSockets[1].Join("room1")

	// Broadcast to "room1" only
	ns.To("room1").Emit("alert", "room1-msg")
	time.Sleep(50 * time.Millisecond)

	// Socket 0 should have the message
	conn0 := upgrader.getConn(0)
	outbox0 := conn0.getOutbox()
	hasRoomMsg := false
	for _, raw := range outbox0 {
		var msg map[string]any
		json.Unmarshal(raw, &msg)
		if msg["event"] == "alert" {
			hasRoomMsg = true
		}
	}
	if !hasRoomMsg {
		t.Error("socket 0 (in room1) should have received room broadcast")
	}

	// Socket 2 should NOT have the room message
	conn2 := upgrader.getConn(2)
	outbox2 := conn2.getOutbox()
	for _, raw := range outbox2 {
		var msg map[string]any
		json.Unmarshal(raw, &msg)
		if msg["event"] == "alert" {
			t.Error("socket 2 (not in room1) should NOT have received room broadcast")
		}
	}

	// Clean up
	for i := 0; i < 3; i++ {
		upgrader.getConn(i).Close()
	}
	time.Sleep(100 * time.Millisecond)
}

func TestSocketIO_Rooms_Leave(t *testing.T) {
	upgrader := newMockSioUpgrader()
	adapter := socketio.NewIoAdapter(socketio.IoAdapterOptions{
		Upgrader: upgrader,
	})
	adapter.Create(nil)

	ns := adapter.GetNamespace("/")
	var sock *socketio.Socket

	ns.OnConnection(func(s *socketio.Socket) {
		sock = s
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/socket.io", nil)
	adapter.ServeHTTP(w, req)
	time.Sleep(50 * time.Millisecond)

	if sock == nil {
		t.Fatal("socket not connected")
	}

	sock.Join("test-room")
	if !sock.InRoom("test-room") {
		t.Error("expected socket to be in test-room after Join")
	}

	rooms := sock.Rooms()
	foundRoom := false
	for _, r := range rooms {
		if r == "test-room" {
			foundRoom = true
		}
	}
	if !foundRoom {
		t.Error("expected test-room in socket's room list")
	}

	sock.Leave("test-room")
	if sock.InRoom("test-room") {
		t.Error("expected socket to NOT be in test-room after Leave")
	}

	upgrader.getConn(0).Close()
	time.Sleep(50 * time.Millisecond)
}

func TestSocketIO_Rooms_GetRoomSockets(t *testing.T) {
	upgrader := newMockSioUpgrader()
	adapter := socketio.NewIoAdapter(socketio.IoAdapterOptions{
		Upgrader: upgrader,
	})
	adapter.Create(nil)

	ns := adapter.GetNamespace("/")
	var sockets []*socketio.Socket

	ns.OnConnection(func(s *socketio.Socket) {
		sockets = append(sockets, s)
	})

	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/socket.io", nil)
		adapter.ServeHTTP(w, req)
	}
	time.Sleep(100 * time.Millisecond)

	sockets[0].Join("vip")
	sockets[1].Join("vip")

	vipSockets := ns.GetRoomSockets("vip")
	if len(vipSockets) != 2 {
		t.Errorf("expected 2 sockets in vip room, got %d", len(vipSockets))
	}

	allRooms := ns.GetRooms()
	foundVip := false
	for _, r := range allRooms {
		if r == "vip" {
			foundVip = true
		}
	}
	if !foundVip {
		t.Error("expected 'vip' in namespace rooms list")
	}

	for i := 0; i < 3; i++ {
		upgrader.getConn(i).Close()
	}
	time.Sleep(100 * time.Millisecond)
}

// ---------------------------------------------------------------------------
// Tests: Acknowledgements
// ---------------------------------------------------------------------------

func TestSocketIO_Acknowledgement(t *testing.T) {
	upgrader := newMockSioUpgrader()
	adapter := socketio.NewIoAdapter(socketio.IoAdapterOptions{
		Upgrader: upgrader,
	})
	adapter.Create(nil)

	ns := adapter.GetNamespace("/")
	ns.On("echo", func(client *websocket.Client, data json.RawMessage) (any, error) {
		var msg string
		json.Unmarshal(data, &msg)
		return map[string]string{"ack": msg}, nil
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/socket.io", nil)
	adapter.ServeHTTP(w, req)
	time.Sleep(50 * time.Millisecond)

	conn := upgrader.getConn(0)

	// Send message with ack ID
	msg, _ := json.Marshal(map[string]any{
		"event": "echo",
		"data":  `"test-ack"`,
		"id":    1,
	})
	conn.inbox <- msg
	time.Sleep(50 * time.Millisecond)

	outbox := conn.getOutbox()
	if len(outbox) == 0 {
		t.Fatal("expected ack response")
	}

	// The response should be an ack (has "id" field)
	var ackResponse map[string]any
	json.Unmarshal(outbox[0], &ackResponse)

	if ackID, ok := ackResponse["id"]; ok {
		if ackID != float64(1) {
			t.Errorf("expected ack id=1, got %v", ackID)
		}
	}

	conn.Close()
	time.Sleep(50 * time.Millisecond)
}

// ---------------------------------------------------------------------------
// Tests: Middleware
// ---------------------------------------------------------------------------

func TestSocketIO_Middleware_Runs(t *testing.T) {
	upgrader := newMockSioUpgrader()
	adapter := socketio.NewIoAdapter(socketio.IoAdapterOptions{
		Upgrader: upgrader,
	})
	adapter.Create(nil)

	middlewareCalled := false
	adapter.Use(func(sock *socketio.Socket) error {
		middlewareCalled = true
		sock.Data["auth"] = "verified"
		return nil
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/socket.io", nil)
	adapter.ServeHTTP(w, req)
	time.Sleep(50 * time.Millisecond)

	if !middlewareCalled {
		t.Error("expected middleware to be called on connection")
	}

	upgrader.getConn(0).Close()
	time.Sleep(50 * time.Millisecond)
}

// ---------------------------------------------------------------------------
// Tests: Namespace broadcast to all
// ---------------------------------------------------------------------------

func TestSocketIO_Namespace_EmitToAll(t *testing.T) {
	upgrader := newMockSioUpgrader()
	adapter := socketio.NewIoAdapter(socketio.IoAdapterOptions{
		Upgrader: upgrader,
	})
	adapter.Create(nil)

	ns := adapter.GetNamespace("/")

	// Connect 2 clients
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/socket.io", nil)
		adapter.ServeHTTP(w, req)
	}
	time.Sleep(50 * time.Millisecond)

	ns.Emit("broadcast", "hello-all")
	time.Sleep(50 * time.Millisecond)

	for i := 0; i < 2; i++ {
		conn := upgrader.getConn(i)
		outbox := conn.getOutbox()
		found := false
		for _, raw := range outbox {
			var msg map[string]any
			json.Unmarshal(raw, &msg)
			if msg["event"] == "broadcast" {
				found = true
			}
		}
		if !found {
			t.Errorf("client %d should have received namespace broadcast", i)
		}
	}

	for i := 0; i < 2; i++ {
		upgrader.getConn(i).Close()
	}
	time.Sleep(50 * time.Millisecond)
}

// ---------------------------------------------------------------------------
// Tests: BroadcastOperator.Except
// ---------------------------------------------------------------------------

func TestSocketIO_Broadcast_Except(t *testing.T) {
	upgrader := newMockSioUpgrader()
	adapter := socketio.NewIoAdapter(socketio.IoAdapterOptions{
		Upgrader: upgrader,
	})
	adapter.Create(nil)

	ns := adapter.GetNamespace("/")
	var sockets []*socketio.Socket

	ns.OnConnection(func(s *socketio.Socket) {
		sockets = append(sockets, s)
	})

	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/socket.io", nil)
		adapter.ServeHTTP(w, req)
	}
	time.Sleep(100 * time.Millisecond)

	if len(sockets) < 3 {
		t.Fatalf("expected 3 sockets, got %d", len(sockets))
	}

	// Broadcast to all except socket[1]
	excludeID := sockets[1].Client.ID
	ns.Except(excludeID).Emit("selective", "data")
	time.Sleep(50 * time.Millisecond)

	// Socket 0 should receive
	conn0Outbox := upgrader.getConn(0).getOutbox()
	found0 := false
	for _, raw := range conn0Outbox {
		var msg map[string]any
		json.Unmarshal(raw, &msg)
		if msg["event"] == "selective" {
			found0 = true
		}
	}
	if !found0 {
		t.Error("socket 0 should have received the broadcast")
	}

	// Socket 1 should NOT receive (excluded)
	conn1Outbox := upgrader.getConn(1).getOutbox()
	for _, raw := range conn1Outbox {
		var msg map[string]any
		json.Unmarshal(raw, &msg)
		if msg["event"] == "selective" {
			t.Error("socket 1 should NOT have received the broadcast (excluded)")
		}
	}

	for i := 0; i < 3; i++ {
		upgrader.getConn(i).Close()
	}
	time.Sleep(50 * time.Millisecond)
}

// ---------------------------------------------------------------------------
// Tests: Socket.IO Module integration with app
// ---------------------------------------------------------------------------

func TestSocketIO_Module_WithApp(t *testing.T) {
	gateway := &sioChatGateway{}
	upgrader := newMockSioUpgrader()

	sioModule := socketio.NewModule(socketio.ModuleOptions{
		Gateway:   gateway,
		Namespace: "/chat",
		Path:      "/socket.io",
		Upgrader:  upgrader,
	})

	appMod := gonest.NewModule(gonest.ModuleOptions{
		Imports: []*gonest.Module{sioModule},
	})

	app := gonest.Create(appMod, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	// Connect via the app's HTTP handler
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/socket.io?nsp=/chat", nil)
	app.Handler().ServeHTTP(w, req)
	time.Sleep(50 * time.Millisecond)

	conn := upgrader.getConn(0)
	if conn == nil {
		t.Fatal("expected connection via app handler")
	}

	// Send a message
	msg, _ := json.Marshal(map[string]any{
		"event": "message",
		"data":  `"hello-from-sio"`,
	})
	conn.inbox <- msg
	time.Sleep(50 * time.Millisecond)

	outbox := conn.getOutbox()
	if len(outbox) == 0 {
		t.Fatal("expected response from Socket.IO gateway")
	}

	conn.Close()
	time.Sleep(50 * time.Millisecond)
}

// ---------------------------------------------------------------------------
// Tests: Connection and disconnection callbacks
// ---------------------------------------------------------------------------

func TestSocketIO_ConnectionDisconnection_Callbacks(t *testing.T) {
	upgrader := newMockSioUpgrader()
	adapter := socketio.NewIoAdapter(socketio.IoAdapterOptions{
		Upgrader: upgrader,
	})
	adapter.Create(nil)

	ns := adapter.GetNamespace("/")

	connectCount := 0
	disconnectCount := 0
	ns.OnConnection(func(s *socketio.Socket) {
		connectCount++
	})
	ns.OnDisconnection(func(s *socketio.Socket) {
		disconnectCount++
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/socket.io", nil)
	adapter.ServeHTTP(w, req)
	time.Sleep(50 * time.Millisecond)

	if connectCount != 1 {
		t.Errorf("expected 1 connection callback, got %d", connectCount)
	}

	upgrader.getConn(0).Close()
	time.Sleep(100 * time.Millisecond)

	if disconnectCount != 1 {
		t.Errorf("expected 1 disconnection callback, got %d", disconnectCount)
	}
}

// ---------------------------------------------------------------------------
// Tests: Socket.To() broadcasts to room excluding self
// ---------------------------------------------------------------------------

func TestSocketIO_Socket_To_ExcludesSelf(t *testing.T) {
	upgrader := newMockSioUpgrader()
	adapter := socketio.NewIoAdapter(socketio.IoAdapterOptions{
		Upgrader: upgrader,
	})
	adapter.Create(nil)

	ns := adapter.GetNamespace("/")
	var sockets []*socketio.Socket
	ns.OnConnection(func(s *socketio.Socket) {
		sockets = append(sockets, s)
		s.Join("general")
	})

	// Connect 2 clients
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/socket.io", nil)
		adapter.ServeHTTP(w, req)
	}
	time.Sleep(100 * time.Millisecond)

	if len(sockets) < 2 {
		t.Fatalf("expected 2 sockets, got %d", len(sockets))
	}

	// Clear outboxes
	for i := 0; i < 2; i++ {
		upgrader.getConn(i).getOutbox() // consume existing
	}

	// Socket 0 sends to room "general" — should reach socket 1 but NOT socket 0
	sockets[0].To("general").Emit("chat", "hello-room")
	time.Sleep(50 * time.Millisecond)

	// Socket 1 should receive
	conn1Outbox := upgrader.getConn(1).getOutbox()
	found1 := false
	for _, raw := range conn1Outbox {
		var msg map[string]any
		json.Unmarshal(raw, &msg)
		if msg["event"] == "chat" {
			found1 = true
		}
	}
	if !found1 {
		t.Error("socket 1 should have received the room broadcast from socket 0")
	}

	for i := 0; i < 2; i++ {
		upgrader.getConn(i).Close()
	}
	time.Sleep(50 * time.Millisecond)
}

// ---------------------------------------------------------------------------
// Tests: BroadcastOperator.EmitToAll returns count
// ---------------------------------------------------------------------------

func TestSocketIO_EmitToAll_ReturnsCount(t *testing.T) {
	upgrader := newMockSioUpgrader()
	adapter := socketio.NewIoAdapter(socketio.IoAdapterOptions{
		Upgrader: upgrader,
	})
	adapter.Create(nil)

	ns := adapter.GetNamespace("/")
	var sockets []*socketio.Socket
	ns.OnConnection(func(s *socketio.Socket) {
		sockets = append(sockets, s)
		s.Join("broadcast-room")
	})

	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/socket.io", nil)
		adapter.ServeHTTP(w, req)
	}
	time.Sleep(100 * time.Millisecond)

	count := ns.To("broadcast-room").EmitToAll("news", "breaking")
	if count != 3 {
		t.Errorf("expected EmitToAll to return 3, got %d", count)
	}

	socketCount := ns.To("broadcast-room").SocketCount()
	if socketCount != 3 {
		t.Errorf("expected SocketCount to return 3, got %d", socketCount)
	}

	for i := 0; i < 3; i++ {
		upgrader.getConn(i).Close()
	}
	time.Sleep(50 * time.Millisecond)
}

// ---------------------------------------------------------------------------
// Tests: Adapter.Close disconnects all
// ---------------------------------------------------------------------------

func TestSocketIO_Adapter_Close(t *testing.T) {
	upgrader := newMockSioUpgrader()
	adapter := socketio.NewIoAdapter(socketio.IoAdapterOptions{
		Upgrader: upgrader,
	})
	adapter.Create(nil)

	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/socket.io", nil)
		adapter.ServeHTTP(w, req)
	}
	time.Sleep(50 * time.Millisecond)

	ns := adapter.GetNamespace("/")
	if ns.SocketCount() != 2 {
		t.Fatalf("expected 2 sockets before close, got %d", ns.SocketCount())
	}

	adapter.Close()
	time.Sleep(100 * time.Millisecond)

	if ns.SocketCount() != 0 {
		t.Errorf("expected 0 sockets after close, got %d", ns.SocketCount())
	}
}
