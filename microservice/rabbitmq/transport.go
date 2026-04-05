package rabbitmq

import (
	"fmt"

	"github.com/gonest/microservice"
)

// Options configures a RabbitMQ microservice transport.
type Options struct {
	Host       string
	Port       int
	Queue      string // queue name for consuming
	Exchange   string // exchange name
	RoutingKey string // routing key for binding
	Durable    bool   // durable queue
	NoAck      bool   // auto-acknowledge messages
}

func (o Options) Address() string {
	host := o.Host
	if host == "" {
		host = "localhost"
	}
	port := o.Port
	if port == 0 {
		port = 5672
	}
	return fmt.Sprintf("%s:%d", host, port)
}

// ServerOptions creates microservice.ServerOptions for RabbitMQ.
func ServerOptions(opts Options) microservice.ServerOptions {
	return microservice.ServerOptions{
		Transport: microservice.TransportRabbitMQ,
		Host:      opts.Host,
		Port:      opts.Port,
	}
}

// ClientOptions creates microservice.ClientOptions for RabbitMQ.
func ClientOptions(opts Options) microservice.ClientOptions {
	return microservice.ClientOptions{
		Transport: microservice.TransportRabbitMQ,
		Host:      opts.Host,
		Port:      opts.Port,
	}
}
