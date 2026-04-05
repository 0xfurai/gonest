package grpc

import (
	"fmt"

	"github.com/gonest/microservice"
)

// Options configures a gRPC microservice transport.
type Options struct {
	Host        string
	Port        int
	ServiceName string // logical service name for routing
}

func (o Options) Address() string {
	host := o.Host
	if host == "" {
		host = "localhost"
	}
	return fmt.Sprintf("%s:%d", host, o.Port)
}

// ServerOptions creates microservice.ServerOptions for gRPC.
func ServerOptions(opts Options) microservice.ServerOptions {
	return microservice.ServerOptions{
		Transport: microservice.TransportGRPC,
		Host:      opts.Host,
		Port:      opts.Port,
	}
}

// ClientOptions creates microservice.ClientOptions for gRPC.
func ClientOptions(opts Options) microservice.ClientOptions {
	return microservice.ClientOptions{
		Transport: microservice.TransportGRPC,
		Host:      opts.Host,
		Port:      opts.Port,
	}
}
