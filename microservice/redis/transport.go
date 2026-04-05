package redis

import (
	"fmt"

	"github.com/0xfurai/gonest/microservice"
)

// Options configures a Redis microservice transport.
type Options struct {
	Host     string
	Port     int
	Password string
	DB       int
}

func (o Options) Address() string {
	host := o.Host
	if host == "" {
		host = "localhost"
	}
	port := o.Port
	if port == 0 {
		port = 6379
	}
	return fmt.Sprintf("%s:%d", host, port)
}

// ServerOptions creates microservice.ServerOptions for Redis.
func ServerOptions(opts Options) microservice.ServerOptions {
	return microservice.ServerOptions{
		Transport: microservice.TransportRedis,
		Host:      opts.Host,
		Port:      opts.Port,
	}
}

// ClientOptions creates microservice.ClientOptions for Redis.
func ClientOptions(opts Options) microservice.ClientOptions {
	return microservice.ClientOptions{
		Transport: microservice.TransportRedis,
		Host:      opts.Host,
		Port:      opts.Port,
	}
}
