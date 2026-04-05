package kafka

import (
	"fmt"

	"github.com/0xfurai/gonest/microservice"
)

// Options configures a Kafka microservice transport.
type Options struct {
	Host    string
	Port    int
	GroupID string // consumer group ID
	Topic   string // default topic
}

func (o Options) Address() string {
	host := o.Host
	if host == "" {
		host = "localhost"
	}
	port := o.Port
	if port == 0 {
		port = 9092
	}
	return fmt.Sprintf("%s:%d", host, port)
}

// ServerOptions creates microservice.ServerOptions for Kafka.
func ServerOptions(opts Options) microservice.ServerOptions {
	return microservice.ServerOptions{
		Transport: microservice.TransportKafka,
		Host:      opts.Host,
		Port:      opts.Port,
	}
}

// ClientOptions creates microservice.ClientOptions for Kafka.
func ClientOptions(opts Options) microservice.ClientOptions {
	return microservice.ClientOptions{
		Transport: microservice.TransportKafka,
		Host:      opts.Host,
		Port:      opts.Port,
	}
}
