package mqtt

import (
	"fmt"

	"github.com/gonest/microservice"
)

// QoS represents MQTT Quality of Service levels.
type QoS int

const (
	// QoSAtMostOnce delivers the message at most once (fire and forget).
	QoSAtMostOnce QoS = 0
	// QoSAtLeastOnce delivers the message at least once (acknowledged delivery).
	QoSAtLeastOnce QoS = 1
	// QoSExactlyOnce delivers the message exactly once (assured delivery).
	QoSExactlyOnce QoS = 2
)

// Options configures an MQTT microservice transport.
type Options struct {
	Host     string
	Port     int
	ClientID string // MQTT client identifier
	QoS      QoS    // quality of service level
	Topic    string // default topic prefix
}

func (o Options) Address() string {
	host := o.Host
	if host == "" {
		host = "localhost"
	}
	port := o.Port
	if port == 0 {
		port = 1883
	}
	return fmt.Sprintf("%s:%d", host, port)
}

// ServerOptions creates microservice.ServerOptions for MQTT.
func ServerOptions(opts Options) microservice.ServerOptions {
	return microservice.ServerOptions{
		Transport: microservice.TransportMQTT,
		Host:      opts.Host,
		Port:      opts.Port,
	}
}

// ClientOptions creates microservice.ClientOptions for MQTT.
func ClientOptions(opts Options) microservice.ClientOptions {
	return microservice.ClientOptions{
		Transport: microservice.TransportMQTT,
		Host:      opts.Host,
		Port:      opts.Port,
	}
}
