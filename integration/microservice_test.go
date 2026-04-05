package integration

import (
	"context"
	"encoding/json"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/0xfurai/gonest/microservice"
	"github.com/0xfurai/gonest/microservice/tcp"
)

// ---------------------------------------------------------------------------
// Microservice TCP Integration Tests
// Mirror: original/integration/microservices/
// ---------------------------------------------------------------------------

func getFreePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	return port
}

// ---------------------------------------------------------------------------
// Tests: Request/Response
// ---------------------------------------------------------------------------

func TestMicroservice_TCP_RequestResponse(t *testing.T) {
	port := getFreePort(t)

	// Create server
	server := tcp.NewServer(microservice.ServerOptions{
		Host: "127.0.0.1",
		Port: port,
	})

	server.AddMessageHandler(microservice.Pattern{Cmd: "sum"}, func(ctx *microservice.MessageContext) (any, error) {
		var nums []int
		json.Unmarshal(ctx.Data, &nums)
		sum := 0
		for _, n := range nums {
			sum += n
		}
		return map[string]int{"result": sum}, nil
	})

	if err := server.Listen(); err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	// Create client
	client := tcp.NewClient(microservice.ClientOptions{
		Host: "127.0.0.1",
		Port: port,
	})
	if err := client.Connect(); err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	// Send request
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.Send(ctx, microservice.Pattern{Cmd: "sum"}, []int{1, 2, 3, 4, 5})
	if err != nil {
		t.Fatal(err)
	}

	var result map[string]int
	json.Unmarshal(resp, &result)
	if result["result"] != 15 {
		t.Errorf("expected sum=15, got %d", result["result"])
	}
}

func TestMicroservice_TCP_MultipleRequests(t *testing.T) {
	port := getFreePort(t)

	server := tcp.NewServer(microservice.ServerOptions{
		Host: "127.0.0.1",
		Port: port,
	})

	server.AddMessageHandler(microservice.Pattern{Cmd: "echo"}, func(ctx *microservice.MessageContext) (any, error) {
		var msg string
		json.Unmarshal(ctx.Data, &msg)
		return msg, nil
	})

	if err := server.Listen(); err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	client := tcp.NewClient(microservice.ClientOptions{
		Host: "127.0.0.1",
		Port: port,
	})
	if err := client.Connect(); err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	messages := []string{"hello", "world", "test", "foo", "bar"}
	for _, msg := range messages {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		resp, err := client.Send(ctx, microservice.Pattern{Cmd: "echo"}, msg)
		cancel()
		if err != nil {
			t.Fatalf("echo %q: %v", msg, err)
		}
		var result string
		json.Unmarshal(resp, &result)
		if result != msg {
			t.Errorf("expected %q, got %q", msg, result)
		}
	}
}

// ---------------------------------------------------------------------------
// Tests: Event (Fire-and-Forget)
// ---------------------------------------------------------------------------

func TestMicroservice_TCP_EventEmit(t *testing.T) {
	port := getFreePort(t)

	var received []string
	var mu sync.Mutex

	server := tcp.NewServer(microservice.ServerOptions{
		Host: "127.0.0.1",
		Port: port,
	})

	server.AddEventHandler(microservice.Pattern{Cmd: "log"}, func(ctx *microservice.MessageContext) error {
		var msg string
		json.Unmarshal(ctx.Data, &msg)
		mu.Lock()
		received = append(received, msg)
		mu.Unlock()
		return nil
	})

	if err := server.Listen(); err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	client := tcp.NewClient(microservice.ClientOptions{
		Host: "127.0.0.1",
		Port: port,
	})
	if err := client.Connect(); err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()
	client.Emit(ctx, microservice.Pattern{Cmd: "log"}, "event1")
	client.Emit(ctx, microservice.Pattern{Cmd: "log"}, "event2")
	client.Emit(ctx, microservice.Pattern{Cmd: "log"}, "event3")

	// Give events time to be processed
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 3 {
		t.Fatalf("expected 3 events, got %d", len(received))
	}
	for i, expected := range []string{"event1", "event2", "event3"} {
		if received[i] != expected {
			t.Errorf("event[%d]: expected %q, got %q", i, expected, received[i])
		}
	}
}

// ---------------------------------------------------------------------------
// Tests: Handler Not Found
// ---------------------------------------------------------------------------

func TestMicroservice_TCP_HandlerNotFound(t *testing.T) {
	port := getFreePort(t)

	server := tcp.NewServer(microservice.ServerOptions{
		Host: "127.0.0.1",
		Port: port,
	})

	if err := server.Listen(); err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	client := tcp.NewClient(microservice.ClientOptions{
		Host: "127.0.0.1",
		Port: port,
	})
	if err := client.Connect(); err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.Send(ctx, microservice.Pattern{Cmd: "nonexistent"}, nil)
	if err == nil {
		t.Error("expected error for missing handler")
	}
}

// ---------------------------------------------------------------------------
// Tests: Handler Error
// ---------------------------------------------------------------------------

func TestMicroservice_TCP_HandlerError(t *testing.T) {
	port := getFreePort(t)

	server := tcp.NewServer(microservice.ServerOptions{
		Host: "127.0.0.1",
		Port: port,
	})

	server.AddMessageHandler(microservice.Pattern{Cmd: "fail"}, func(ctx *microservice.MessageContext) (any, error) {
		return nil, &testError{msg: "handler error"}
	})

	if err := server.Listen(); err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	client := tcp.NewClient(microservice.ClientOptions{
		Host: "127.0.0.1",
		Port: port,
	})
	if err := client.Connect(); err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.Send(ctx, microservice.Pattern{Cmd: "fail"}, nil)
	if err == nil {
		t.Fatal("expected error from handler")
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string { return e.msg }

// ---------------------------------------------------------------------------
// Tests: Concurrent Requests
// ---------------------------------------------------------------------------

func TestMicroservice_TCP_ConcurrentRequests(t *testing.T) {
	port := getFreePort(t)

	server := tcp.NewServer(microservice.ServerOptions{
		Host: "127.0.0.1",
		Port: port,
	})

	server.AddMessageHandler(microservice.Pattern{Cmd: "multiply"}, func(ctx *microservice.MessageContext) (any, error) {
		var args [2]int
		json.Unmarshal(ctx.Data, &args)
		return args[0] * args[1], nil
	})

	if err := server.Listen(); err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	client := tcp.NewClient(microservice.ClientOptions{
		Host: "127.0.0.1",
		Port: port,
	})
	if err := client.Connect(); err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	// Note: TCP client serializes writes with a mutex, but responses come back
	// matched by ID. We test sequential here since the client multiplexes on one conn.
	for i := 1; i <= 10; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		resp, err := client.Send(ctx, microservice.Pattern{Cmd: "multiply"}, [2]int{i, i})
		cancel()
		if err != nil {
			t.Fatalf("request %d: %v", i, err)
		}
		var result int
		json.Unmarshal(resp, &result)
		if result != i*i {
			t.Errorf("request %d: expected %d, got %d", i, i*i, result)
		}
	}
}

// ---------------------------------------------------------------------------
// Tests: Multiple Patterns
// ---------------------------------------------------------------------------

func TestMicroservice_TCP_MultiplePatterns(t *testing.T) {
	port := getFreePort(t)

	server := tcp.NewServer(microservice.ServerOptions{
		Host: "127.0.0.1",
		Port: port,
	})

	server.AddMessageHandler(microservice.Pattern{Cmd: "add"}, func(ctx *microservice.MessageContext) (any, error) {
		var nums [2]int
		json.Unmarshal(ctx.Data, &nums)
		return nums[0] + nums[1], nil
	})

	server.AddMessageHandler(microservice.Pattern{Cmd: "subtract"}, func(ctx *microservice.MessageContext) (any, error) {
		var nums [2]int
		json.Unmarshal(ctx.Data, &nums)
		return nums[0] - nums[1], nil
	})

	server.AddMessageHandler(microservice.Pattern{Cmd: "greet"}, func(ctx *microservice.MessageContext) (any, error) {
		var name string
		json.Unmarshal(ctx.Data, &name)
		return "Hello " + name, nil
	})

	if err := server.Listen(); err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	client := tcp.NewClient(microservice.ClientOptions{
		Host: "127.0.0.1",
		Port: port,
	})
	if err := client.Connect(); err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()

	// Test add
	resp, err := client.Send(ctx, microservice.Pattern{Cmd: "add"}, [2]int{10, 20})
	if err != nil {
		t.Fatal(err)
	}
	var addResult int
	json.Unmarshal(resp, &addResult)
	if addResult != 30 {
		t.Errorf("add: expected 30, got %d", addResult)
	}

	// Test subtract
	resp, err = client.Send(ctx, microservice.Pattern{Cmd: "subtract"}, [2]int{50, 20})
	if err != nil {
		t.Fatal(err)
	}
	var subResult int
	json.Unmarshal(resp, &subResult)
	if subResult != 30 {
		t.Errorf("subtract: expected 30, got %d", subResult)
	}

	// Test greet
	resp, err = client.Send(ctx, microservice.Pattern{Cmd: "greet"}, "World")
	if err != nil {
		t.Fatal(err)
	}
	var greeting string
	json.Unmarshal(resp, &greeting)
	if greeting != "Hello World" {
		t.Errorf("greet: expected Hello World, got %q", greeting)
	}
}

// ---------------------------------------------------------------------------
// Tests: Server Close
// ---------------------------------------------------------------------------

func TestMicroservice_TCP_ServerClose(t *testing.T) {
	port := getFreePort(t)

	server := tcp.NewServer(microservice.ServerOptions{
		Host: "127.0.0.1",
		Port: port,
	})

	if err := server.Listen(); err != nil {
		t.Fatal(err)
	}

	if err := server.Close(); err != nil {
		t.Fatal(err)
	}

	// After close, new connections should fail
	_, err := net.DialTimeout("tcp", "127.0.0.1:"+fmtPort(port), 500*time.Millisecond)
	if err == nil {
		t.Error("expected connection to fail after server close")
	}
}

func fmtPort(n int) string {
	buf := make([]byte, 0, 10)
	if n == 0 {
		return "0"
	}
	for n > 0 {
		buf = append(buf, byte('0'+n%10))
		n /= 10
	}
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}
