package mqtt

import "github.com/gonest/microservice"

// MqttContext carries MQTT-specific metadata for a received message.
type MqttContext struct {
	*microservice.MessageContext
	topic    string
	qos      QoS
	retained bool
}

// NewMqttContext creates a new MqttContext.
func NewMqttContext(ctx *microservice.MessageContext, topic string, qos QoS, retained bool) *MqttContext {
	return &MqttContext{
		MessageContext: ctx,
		topic:          topic,
		qos:            qos,
		retained:       retained,
	}
}

// Topic returns the MQTT topic.
func (c *MqttContext) Topic() string { return c.topic }

// QoS returns the MQTT quality of service level.
func (c *MqttContext) QoS() QoS { return c.qos }

// Retained returns whether this was a retained message.
func (c *MqttContext) Retained() bool { return c.retained }
