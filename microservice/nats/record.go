package nats

// NatsRecord represents a NATS message with data and headers.
type NatsRecord struct {
	Data    any
	Headers map[string]string
}

// NatsRecordBuilder provides a fluent API for building NatsRecord messages.
type NatsRecordBuilder struct {
	data    any
	headers map[string]string
}

// NewNatsRecordBuilder creates a new NatsRecordBuilder.
func NewNatsRecordBuilder() *NatsRecordBuilder {
	return &NatsRecordBuilder{
		headers: make(map[string]string),
	}
}

// SetData sets the message data.
func (b *NatsRecordBuilder) SetData(data any) *NatsRecordBuilder {
	b.data = data
	return b
}

// SetHeaders replaces all headers.
func (b *NatsRecordBuilder) SetHeaders(headers map[string]string) *NatsRecordBuilder {
	b.headers = headers
	return b
}

// SetHeader sets a single header key-value pair.
func (b *NatsRecordBuilder) SetHeader(key, value string) *NatsRecordBuilder {
	b.headers[key] = value
	return b
}

// Build creates the NatsRecord.
func (b *NatsRecordBuilder) Build() *NatsRecord {
	return &NatsRecord{
		Data:    b.data,
		Headers: b.headers,
	}
}
