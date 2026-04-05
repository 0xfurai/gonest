package tcp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/0xfurai/gonest/microservice"
)

func TestTCP_Integration(t *testing.T) {
	port := 19876

	server := NewServer(microservice.ServerOptions{Host: "127.0.0.1", Port: port})

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

	client := NewClient(microservice.ClientOptions{Host: "127.0.0.1", Port: port})
	if err := client.Connect(); err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test sum
	resp, err := client.Send(ctx, microservice.Pattern{Cmd: "sum"}, []int{1, 2, 3, 4, 5})
	if err != nil {
		t.Fatalf("send sum: %v", err)
	}
	var sum int
	json.Unmarshal(resp, &sum)
	if sum != 15 {
		t.Errorf("expected 15, got %d", sum)
	}

	// Test echo
	resp, err = client.Send(ctx, microservice.Pattern{Cmd: "echo"}, "hello")
	if err != nil {
		t.Fatalf("send echo: %v", err)
	}
	var echo string
	json.Unmarshal(resp, &echo)
	if echo != "hello" {
		t.Errorf("expected 'hello', got %q", echo)
	}
}

func TestTCP_UnknownPattern(t *testing.T) {
	port := 19877

	server := NewServer(microservice.ServerOptions{Host: "127.0.0.1", Port: port})
	if err := server.Listen(); err != nil {
		t.Fatalf("server listen: %v", err)
	}
	defer server.Close()
	time.Sleep(50 * time.Millisecond)

	client := NewClient(microservice.ClientOptions{Host: "127.0.0.1", Port: port})
	if err := client.Connect(); err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := client.Send(ctx, microservice.Pattern{Cmd: "nonexistent"}, nil)
	if err == nil {
		t.Fatal("expected error for unknown pattern")
	}
}

func TestTCP_Emit(t *testing.T) {
	port := 19878
	received := make(chan string, 1)

	server := NewServer(microservice.ServerOptions{Host: "127.0.0.1", Port: port})
	server.AddEventHandler(microservice.Pattern{Cmd: "log"}, func(ctx *microservice.MessageContext) error {
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

	client := NewClient(microservice.ClientOptions{Host: "127.0.0.1", Port: port})
	if err := client.Connect(); err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer client.Close()

	err := client.Emit(context.Background(), microservice.Pattern{Cmd: "log"}, "test-message")
	if err != nil {
		t.Fatalf("emit: %v", err)
	}

	select {
	case msg := <-received:
		if msg != "test-message" {
			t.Errorf("expected 'test-message', got %q", msg)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestMessageContext(t *testing.T) {
	ctx := &microservice.MessageContext{
		Pattern:   microservice.Pattern{Cmd: "test"},
		Transport: microservice.TransportTCP,
	}

	if ctx.Context() == nil {
		t.Error("expected non-nil context")
	}
	if ctx.Pattern.Cmd != "test" {
		t.Errorf("expected 'test', got %q", ctx.Pattern.Cmd)
	}
}

func TestServerOptions_Address(t *testing.T) {
	opts := microservice.ServerOptions{Host: "127.0.0.1", Port: 3000}
	if opts.Address() != "127.0.0.1:3000" {
		t.Errorf("expected '127.0.0.1:3000', got %q", opts.Address())
	}
}

func TestServerOptions_DefaultHost(t *testing.T) {
	opts := microservice.ServerOptions{Port: 3000}
	if opts.Address() != "localhost:3000" {
		t.Errorf("expected 'localhost:3000', got %q", opts.Address())
	}
}
