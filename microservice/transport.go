package microservice

import (
	"context"
	"encoding/json"
	"fmt"
)

// Transport identifies the microservice transport layer.
type Transport int

const (
	TransportTCP Transport = iota
	TransportGRPC
	TransportNATS
	TransportRedis
	TransportKafka
	TransportRabbitMQ
	TransportMQTT
	TransportCustom
)

// String returns the string representation of a Transport.
func (t Transport) String() string {
	switch t {
	case TransportTCP:
		return "TCP"
	case TransportGRPC:
		return "GRPC"
	case TransportNATS:
		return "NATS"
	case TransportRedis:
		return "REDIS"
	case TransportKafka:
		return "KAFKA"
	case TransportRabbitMQ:
		return "RABBITMQ"
	case TransportMQTT:
		return "MQTT"
	case TransportCustom:
		return "CUSTOM"
	default:
		return "UNKNOWN"
	}
}

// Pattern is a message pattern used for request/response communication.
type Pattern struct {
	Cmd string `json:"cmd"`
}

// MessageContext carries metadata about a received message.
type MessageContext struct {
	Pattern   Pattern
	Transport Transport
	Data      json.RawMessage
	ctx       context.Context
}

// Context returns the context associated with this message.
func (mc *MessageContext) Context() context.Context {
	if mc.ctx == nil {
		return context.Background()
	}
	return mc.ctx
}

// MessageHandler processes incoming microservice messages.
type MessageHandler func(ctx *MessageContext) (any, error)

// EventHandler processes incoming events (fire-and-forget).
type EventHandler func(ctx *MessageContext) error

// Server defines the interface for a microservice server.
type Server interface {
	// AddMessageHandler registers a request/response handler for a pattern.
	AddMessageHandler(pattern Pattern, handler MessageHandler)
	// AddEventHandler registers a fire-and-forget handler for a pattern.
	AddEventHandler(pattern Pattern, handler EventHandler)
	// Listen starts the microservice server.
	Listen() error
	// Close shuts down the server.
	Close() error
}

// ClientProxy sends messages to a remote microservice.
type ClientProxy interface {
	// Send sends a request and waits for a response.
	Send(ctx context.Context, pattern Pattern, data any) (json.RawMessage, error)
	// Emit sends a fire-and-forget event.
	Emit(ctx context.Context, pattern Pattern, data any) error
	// Connect establishes the connection.
	Connect() error
	// Close closes the connection.
	Close() error
}

// ServerOptions configures a microservice server.
type ServerOptions struct {
	Transport Transport
	Host      string
	Port      int
}

func (o ServerOptions) Address() string {
	host := o.Host
	if host == "" {
		host = "localhost"
	}
	return fmt.Sprintf("%s:%d", host, o.Port)
}

// ClientOptions configures a microservice client.
type ClientOptions struct {
	Transport Transport
	Host      string
	Port      int
}

func (o ClientOptions) Address() string {
	host := o.Host
	if host == "" {
		host = "localhost"
	}
	return fmt.Sprintf("%s:%d", host, o.Port)
}

// CustomTransportStrategy is the interface for user-defined transport strategies.
// Implement this to create custom microservice transports beyond the built-in
// TCP, gRPC, NATS, Redis, Kafka, RabbitMQ, and MQTT transports.
//
// Usage:
//
//	type MyTransportServer struct {
//	    handlers map[string]microservice.MessageHandler
//	    events   map[string]microservice.EventHandler
//	}
//
//	func (s *MyTransportServer) AddMessageHandler(p Pattern, h MessageHandler) {
//	    s.handlers[p.Cmd] = h
//	}
//	func (s *MyTransportServer) AddEventHandler(p Pattern, h EventHandler) {
//	    s.events[p.Cmd] = h
//	}
//	func (s *MyTransportServer) Listen() error { /* start custom server */ }
//	func (s *MyTransportServer) Close() error  { /* stop custom server */ }
//	func (s *MyTransportServer) GetTransportId() Transport { return TransportCustom }
//
// Then register it:
//
//	app := gonest.CreateMicroservice(rootModule, microservice.MicroserviceOptions{
//	    Strategy: &MyTransportServer{},
//	})
type CustomTransportStrategy interface {
	Server
	// GetTransportId returns the transport identifier for this strategy.
	GetTransportId() Transport
}

// MicroserviceOptions configures a microservice with either built-in or custom transport.
type MicroserviceOptions struct {
	// Strategy is a custom transport strategy. If set, Server is ignored.
	Strategy CustomTransportStrategy
	// Server is a built-in transport server. Used when Strategy is nil.
	Server Server
}

// GetServer returns the effective server, preferring the custom strategy.
func (o MicroserviceOptions) GetServer() Server {
	if o.Strategy != nil {
		return o.Strategy
	}
	return o.Server
}

