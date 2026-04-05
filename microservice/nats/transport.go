package nats

import (
	"fmt"

	"github.com/gonest/microservice"
)

// Options configures a NATS microservice transport.
type Options struct {
	Host  string
	Port  int
	Queue string // queue group for load balancing
}

func (o Options) Address() string {
	host := o.Host
	if host == "" {
		host = "localhost"
	}
	port := o.Port
	if port == 0 {
		port = 4222
	}
	return fmt.Sprintf("%s:%d", host, port)
}

// ServerOptions creates microservice.ServerOptions for NATS.
func ServerOptions(opts Options) microservice.ServerOptions {
	return microservice.ServerOptions{
		Transport: microservice.TransportNATS,
		Host:      opts.Host,
		Port:      opts.Port,
	}
}

// ClientOptions creates microservice.ClientOptions for NATS.
func ClientOptions(opts Options) microservice.ClientOptions {
	return microservice.ClientOptions{
		Transport: microservice.TransportNATS,
		Host:      opts.Host,
		Port:      opts.Port,
	}
}
