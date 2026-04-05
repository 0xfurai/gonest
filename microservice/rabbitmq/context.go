package rabbitmq

import "github.com/0xfurai/gonest/microservice"

// RmqContext carries RabbitMQ-specific metadata for a received message.
type RmqContext struct {
	*microservice.MessageContext
	queue      string
	exchange   string
	routingKey string
	headers    map[string]string
}

// NewRmqContext creates a new RmqContext.
func NewRmqContext(ctx *microservice.MessageContext, queue, exchange, routingKey string, headers map[string]string) *RmqContext {
	if headers == nil {
		headers = make(map[string]string)
	}
	return &RmqContext{
		MessageContext: ctx,
		queue:          queue,
		exchange:       exchange,
		routingKey:     routingKey,
		headers:        headers,
	}
}

// Queue returns the RabbitMQ queue name.
func (c *RmqContext) Queue() string { return c.queue }

// Exchange returns the RabbitMQ exchange name.
func (c *RmqContext) Exchange() string { return c.exchange }

// RoutingKey returns the RabbitMQ routing key.
func (c *RmqContext) RoutingKey() string { return c.routingKey }

// Headers returns the RabbitMQ message headers.
func (c *RmqContext) Headers() map[string]string { return c.headers }
