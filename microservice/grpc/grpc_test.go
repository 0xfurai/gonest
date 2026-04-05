package grpc

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/0xfurai/gonest/microservice"
)

func TestGRPC_Integration(t *testing.T) {
	port := 29876

	server := NewServer(Options{Host: "127.0.0.1", Port: port, ServiceName: "TestService"})

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

	client := NewClient(Options{Host: "127.0.0.1", Port: port, ServiceName: "TestService"})
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
	resp, err = client.Send(ctx, microservice.Pattern{Cmd: "echo"}, "hello-grpc")
	if err != nil {
		t.Fatalf("send echo: %v", err)
	}
	var echo string
	json.Unmarshal(resp, &echo)
	if echo != "hello-grpc" {
		t.Errorf("expected 'hello-grpc', got %q", echo)
	}
}

func TestGRPC_UnknownPattern(t *testing.T) {
	port := 29877

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
		t.Fatal("expected error for unknown pattern")
	}
}

func TestGRPC_Emit(t *testing.T) {
	port := 29878
	received := make(chan string, 1)

	server := NewServer(Options{Host: "127.0.0.1", Port: port})
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

	client := NewClient(Options{Host: "127.0.0.1", Port: port})
	if err := client.Connect(); err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer client.Close()

	err := client.Emit(context.Background(), microservice.Pattern{Cmd: "log"}, "grpc-event")
	if err != nil {
		t.Fatalf("emit: %v", err)
	}

	select {
	case msg := <-received:
		if msg != "grpc-event" {
			t.Errorf("expected 'grpc-event', got %q", msg)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestGRPC_MultipleRequests(t *testing.T) {
	port := 29879

	server := NewServer(Options{Host: "127.0.0.1", Port: port, ServiceName: "MathService"})
	server.AddMessageHandler(microservice.Pattern{Cmd: "multiply"}, func(ctx *microservice.MessageContext) (any, error) {
		var nums []int
		json.Unmarshal(ctx.Data, &nums)
		result := 1
		for _, n := range nums {
			result *= n
		}
		return result, nil
	})

	if err := server.Listen(); err != nil {
		t.Fatalf("server listen: %v", err)
	}
	defer server.Close()
	time.Sleep(50 * time.Millisecond)

	client := NewClient(Options{Host: "127.0.0.1", Port: port, ServiceName: "MathService"})
	if err := client.Connect(); err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Send multiple requests in sequence to verify framing stays in sync
	for i := 0; i < 10; i++ {
		resp, err := client.Send(ctx, microservice.Pattern{Cmd: "multiply"}, []int{2, 3})
		if err != nil {
			t.Fatalf("request %d: %v", i, err)
		}
		var result int
		json.Unmarshal(resp, &result)
		if result != 6 {
			t.Errorf("request %d: expected 6, got %d", i, result)
		}
	}
}

func TestGRPC_Context(t *testing.T) {
	msgCtx := &microservice.MessageContext{
		Pattern:   microservice.Pattern{Cmd: "test"},
		Transport: microservice.TransportGRPC,
	}

	grpcCtx := NewGrpcContext(msgCtx, "UserService", "GetUser")
	if grpcCtx.ServiceName() != "UserService" {
		t.Errorf("expected 'UserService', got %q", grpcCtx.ServiceName())
	}
	if grpcCtx.MethodName() != "GetUser" {
		t.Errorf("expected 'GetUser', got %q", grpcCtx.MethodName())
	}
	if grpcCtx.Pattern.Cmd != "test" {
		t.Errorf("expected 'test', got %q", grpcCtx.Pattern.Cmd)
	}
}

func TestGRPC_Options(t *testing.T) {
	opts := Options{Host: "127.0.0.1", Port: 5000, ServiceName: "Test"}
	if opts.Address() != "127.0.0.1:5000" {
		t.Errorf("expected '127.0.0.1:5000', got %q", opts.Address())
	}

	opts2 := Options{Port: 5000}
	if opts2.Address() != "localhost:5000" {
		t.Errorf("expected 'localhost:5000', got %q", opts2.Address())
	}

	sOpts := ServerOptions(opts)
	if sOpts.Transport != microservice.TransportGRPC {
		t.Errorf("expected TransportGRPC, got %d", sOpts.Transport)
	}

	cOpts := ClientOptions(opts)
	if cOpts.Transport != microservice.TransportGRPC {
		t.Errorf("expected TransportGRPC, got %d", cOpts.Transport)
	}
}
