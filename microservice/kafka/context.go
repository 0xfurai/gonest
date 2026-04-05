package kafka

import "github.com/0xfurai/gonest/microservice"

// KafkaContext carries Kafka-specific metadata for a received message.
type KafkaContext struct {
	*microservice.MessageContext
	topic     string
	partition int
	offset    int64
	groupID   string
	headers   map[string]string
}

// NewKafkaContext creates a new KafkaContext.
func NewKafkaContext(ctx *microservice.MessageContext, topic string, partition int, offset int64, groupID string, headers map[string]string) *KafkaContext {
	if headers == nil {
		headers = make(map[string]string)
	}
	return &KafkaContext{
		MessageContext: ctx,
		topic:          topic,
		partition:      partition,
		offset:         offset,
		groupID:        groupID,
		headers:        headers,
	}
}

// Topic returns the Kafka topic.
func (c *KafkaContext) Topic() string { return c.topic }

// Partition returns the Kafka partition number.
func (c *KafkaContext) Partition() int { return c.partition }

// Offset returns the Kafka message offset.
func (c *KafkaContext) Offset() int64 { return c.offset }

// GroupID returns the consumer group ID.
func (c *KafkaContext) GroupID() string { return c.groupID }

// Headers returns the Kafka message headers.
func (c *KafkaContext) Headers() map[string]string { return c.headers }
