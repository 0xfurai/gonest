package nats

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/0xfurai/gonest/microservice"
)

func TestNATS_Integration(t *testing.T) {
	port := 39876

	server := NewServer(Options{Host: "127.0.0.1", Port: port, Queue: "test-queue"})

	server.AddMessageHandler(microservice.Pattern{Cmd: "sum"}, func(ctx *microservice.MessageContext) (any, error) {
		var nums []int
		json.Unmarshal(ctx.Data, &nums)
		sum := 0
		for _, n := range nums {
			sum += n
		}
		return sum, nil
	})

	server.AddMessageHandler(microservice.Pattern{Cmd: "echo"}, func(ctx *microservice.MessageContext) (any, error) {
		var msg string
		json.Unmarshal(ctx.Data, &msg)
		return msg, nil
	})

	if err := server.Listen(); err != nil {
		t.Fatalf("server listen: %v", err)
	}
	defer server.Close()

	time.Sleep(50 * time.Millisecond)

	client := NewClient(Options{Host: "127.0.0.1", Port: port, Queue: "test-queue"})
	if err := client.Connect(); err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test sum (subject-based routing)
	resp, err := client.Send(ctx, microservice.Pattern{Cmd: "sum"}, []int{10, 20, 30})
	if err != nil {
		t.Fatalf("send sum: %v", err)
	}
	var sum int
	json.Unmarshal(resp, &sum)
	if sum != 60 {
		t.Errorf("expected 60, got %d", sum)
	}

	// Test echo
	resp, err = client.Send(ctx, microservice.Pattern{Cmd: "echo"}, "hello-nats")
	if err != nil {
		t.Fatalf("send echo: %v", err)
	}
	var echo string
	json.Unmarshal(resp, &echo)
	if echo != "hello-nats" {
		t.Errorf("expected 'hello-nats', got %q", echo)
	}
}

func TestNATS_UnknownSubject(t *testing.T) {
	port := 39877

	server := NewServer(Options{Host: "127.0.0.1", Port: port})
	if err := server.Listen(); err != nil {
		t.Fatalf("server listen: %v", err)
	}
	defer server.Close()
	time.Sleep(50 * time.Millisecond)

	client := NewClient(Options{Host: "127.0.0.1", Port: port})
	if err := client.Connect(); err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := client.Send(ctx, microservice.Pattern{Cmd: "nonexistent"}, nil)
	if err == nil {
		t.Fatal("expected error for unknown subject")
	}
}

func TestNATS_Emit(t *testing.T) {
	port := 39878
	received := make(chan string, 1)

	server := NewServer(Options{Host: "127.0.0.1", Port: port})
	server.AddEventHandler(microservice.Pattern{Cmd: "notifications"}, func(ctx *microservice.MessageContext) error {
		var msg string
		json.Unmarshal(ctx.Data, &msg)
		received <- msg
		return nil
	})

	if err := server.Listen(); err != nil {
		t.Fatalf("server listen: %v", err)
	}
	defer server.Close()
	time.Sleep(50 * time.Millisecond)

	client := NewClient(Options{Host: "127.0.0.1", Port: port})
	if err := client.Connect(); err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer client.Close()

	err := client.Emit(context.Background(), microservice.Pattern{Cmd: "notifications"}, "nats-event")
	if err != nil {
		t.Fatalf("emit: %v", err)
	}

	select {
	case msg := <-received:
		if msg != "nats-event" {
			t.Errorf("expected 'nats-event', got %q", msg)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestNATS_RecordBuilder(t *testing.T) {
	record := NewNatsRecordBuilder().
		SetData(map[string]string{"greeting": "hello"}).
		SetHeader("x-request-id", "abc-123").
		SetHeader("x-trace-id", "trace-456").
		Build()

	if record.Headers["x-request-id"] != "abc-123" {
		t.Errorf("expected header 'abc-123', got %q", record.Headers["x-request-id"])
	}
	if record.Headers["x-trace-id"] != "trace-456" {
		t.Errorf("expected header 'trace-456', got %q", record.Headers["x-trace-id"])
	}

	dataMap, ok := record.Data.(map[string]string)
	if !ok {
		t.Fatal("expected Data to be map[string]string")
	}
	if dataMap["greeting"] != "hello" {
		t.Errorf("expected 'hello', got %q", dataMap["greeting"])
	}
}

func TestNATS_RecordBuilderSetHeaders(t *testing.T) {
	headers := map[string]string{
		"content-type": "application/json",
		"x-version":    "2",
	}
	record := NewNatsRecordBuilder().
		SetData("payload").
		SetHeaders(headers).
		Build()

	if len(record.Headers) != 2 {
		t.Errorf("expected 2 headers, got %d", len(record.Headers))
	}
	if record.Headers["content-type"] != "application/json" {
		t.Errorf("expected 'application/json', got %q", record.Headers["content-type"])
	}
}

func TestNATS_SendWithRecord(t *testing.T) {
	port := 39879

	server := NewServer(Options{Host: "127.0.0.1", Port: port})

	server.AddMessageHandler(microservice.Pattern{Cmd: "greet"}, func(ctx *microservice.MessageContext) (any, error) {
		var name string
		json.Unmarshal(ctx.Data, &name)
		return "hello, " + name, nil
	})

	if err := server.Listen(); err != nil {
		t.Fatalf("server listen: %v", err)
	}
	defer server.Close()
	time.Sleep(50 * time.Millisecond)

	client := NewClient(Options{Host: "127.0.0.1", Port: port})
	if err := client.Connect(); err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	record := NewNatsRecordBuilder().
		SetData("world").
		SetHeader("x-request-id", "req-001").
		Build()

	resp, err := client.Send(ctx, microservice.Pattern{Cmd: "greet"}, record)
	if err != nil {
		t.Fatalf("send with record: %v", err)
	}
	var greeting string
	json.Unmarshal(resp, &greeting)
	if greeting != "hello, world" {
		t.Errorf("expected 'hello, world', got %q", greeting)
	}
}

func TestNATS_Context(t *testing.T) {
	msgCtx := &microservice.MessageContext{
		Pattern:   microservice.Pattern{Cmd: "test.subject"},
		Transport: microservice.TransportNATS,
	}

	natsCtx := NewNatsContext(msgCtx, "test.subject", map[string]string{"x-id": "123"})
	if natsCtx.Subject() != "test.subject" {
		t.Errorf("expected 'test.subject', got %q", natsCtx.Subject())
	}
	if natsCtx.Headers()["x-id"] != "123" {
		t.Errorf("expected header '123', got %q", natsCtx.Headers()["x-id"])
	}
	if natsCtx.Pattern.Cmd != "test.subject" {
		t.Errorf("expected 'test.subject', got %q", natsCtx.Pattern.Cmd)
	}
}

func TestNATS_ContextNilHeaders(t *testing.T) {
	msgCtx := &microservice.MessageContext{
		Pattern:   microservice.Pattern{Cmd: "test"},
		Transport: microservice.TransportNATS,
	}

	natsCtx := NewNatsContext(msgCtx, "test", nil)
	if natsCtx.Headers() == nil {
		t.Error("expected non-nil headers map")
	}
}

func TestNATS_Options(t *testing.T) {
	opts := Options{Host: "127.0.0.1", Port: 4222, Queue: "workers"}
	if opts.Address() != "127.0.0.1:4222" {
		t.Errorf("expected '127.0.0.1:4222', got %q", opts.Address())
	}

	opts2 := Options{}
	if opts2.Address() != "localhost:4222" {
		t.Errorf("expected 'localhost:4222', got %q", opts2.Address())
	}

	sOpts := ServerOptions(opts)
	if sOpts.Transport != microservice.TransportNATS {
		t.Errorf("expected TransportNATS, got %d", sOpts.Transport)
	}

	cOpts := ClientOptions(opts)
	if cOpts.Transport != microservice.TransportNATS {
		t.Errorf("expected TransportNATS, got %d", cOpts.Transport)
	}
}

func TestNATS_MultipleRequests(t *testing.T) {
	port := 39880

	server := NewServer(Options{Host: "127.0.0.1", Port: port})
	server.AddMessageHandler(microservice.Pattern{Cmd: "ping"}, func(ctx *microservice.MessageContext) (any, error) {
		return "pong", nil
	})

	if err := server.Listen(); err != nil {
		t.Fatalf("server listen: %v", err)
	}
	defer server.Close()
	time.Sleep(50 * time.Millisecond)

	client := NewClient(Options{Host: "127.0.0.1", Port: port})
	if err := client.Connect(); err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for i := 0; i < 20; i++ {
		resp, err := client.Send(ctx, microservice.Pattern{Cmd: "ping"}, nil)
		if err != nil {
			t.Fatalf("request %d: %v", i, err)
		}
		var result string
		json.Unmarshal(resp, &result)
		if result != "pong" {
			t.Errorf("request %d: expected 'pong', got %q", i, result)
		}
	}
}
