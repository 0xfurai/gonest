package nats

import "github.com/0xfurai/gonest/microservice"

// NatsContext carries NATS-specific metadata for a received message.
type NatsContext struct {
	*microservice.MessageContext
	subject string
	headers map[string]string
}

// NewNatsContext creates a new NatsContext.
func NewNatsContext(ctx *microservice.MessageContext, subject string, headers map[string]string) *NatsContext {
	if headers == nil {
		headers = make(map[string]string)
	}
	return &NatsContext{
		MessageContext: ctx,
		subject:        subject,
		headers:        headers,
	}
}

// Subject returns the NATS subject.
func (c *NatsContext) Subject() string { return c.subject }

// Headers returns the NATS message headers.
func (c *NatsContext) Headers() map[string]string { return c.headers }
