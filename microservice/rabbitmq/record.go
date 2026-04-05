package rabbitmq

// RmqRecord represents a RabbitMQ message with data and headers.
type RmqRecord struct {
	Data       any
	Headers    map[string]string
	RoutingKey string
	Exchange   string
}

// RmqRecordBuilder provides a fluent API for building RmqRecord messages.
type RmqRecordBuilder struct {
	data       any
	headers    map[string]string
	routingKey string
	exchange   string
}

// NewRmqRecordBuilder creates a new RmqRecordBuilder.
func NewRmqRecordBuilder() *RmqRecordBuilder {
	return &RmqRecordBuilder{
		headers: make(map[string]string),
	}
}

// SetData sets the message data.
func (b *RmqRecordBuilder) SetData(data any) *RmqRecordBuilder {
	b.data = data
	return b
}

// SetHeaders replaces all headers.
func (b *RmqRecordBuilder) SetHeaders(headers map[string]string) *RmqRecordBuilder {
	b.headers = headers
	return b
}

// SetHeader sets a single header key-value pair.
func (b *RmqRecordBuilder) SetHeader(key, value string) *RmqRecordBuilder {
	b.headers[key] = value
	return b
}

// SetRoutingKey sets the RabbitMQ routing key.
func (b *RmqRecordBuilder) SetRoutingKey(key string) *RmqRecordBuilder {
	b.routingKey = key
	return b
}

// SetExchange sets the RabbitMQ exchange name.
func (b *RmqRecordBuilder) SetExchange(exchange string) *RmqRecordBuilder {
	b.exchange = exchange
	return b
}

// Build creates the RmqRecord.
func (b *RmqRecordBuilder) Build() *RmqRecord {
	return &RmqRecord{
		Data:       b.data,
		Headers:    b.headers,
		RoutingKey: b.routingKey,
		Exchange:   b.exchange,
	}
}
