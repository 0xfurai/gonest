package mqtt

// MqttRecord represents an MQTT message with data, topic, and QoS.
type MqttRecord struct {
	Data     any
	Topic    string
	QoS      QoS
	Retained bool
	Headers  map[string]string // MQTT 5.0 user properties
}

// MqttRecordBuilder provides a fluent API for building MqttRecord messages.
type MqttRecordBuilder struct {
	data     any
	topic    string
	qos      QoS
	retained bool
	headers  map[string]string
}

// NewMqttRecordBuilder creates a new MqttRecordBuilder.
func NewMqttRecordBuilder() *MqttRecordBuilder {
	return &MqttRecordBuilder{
		headers: make(map[string]string),
	}
}

// SetData sets the message data.
func (b *MqttRecordBuilder) SetData(data any) *MqttRecordBuilder {
	b.data = data
	return b
}

// SetTopic sets the MQTT topic.
func (b *MqttRecordBuilder) SetTopic(topic string) *MqttRecordBuilder {
	b.topic = topic
	return b
}

// SetQoS sets the quality of service level.
func (b *MqttRecordBuilder) SetQoS(qos QoS) *MqttRecordBuilder {
	b.qos = qos
	return b
}

// SetRetained sets the retained flag.
func (b *MqttRecordBuilder) SetRetained(retained bool) *MqttRecordBuilder {
	b.retained = retained
	return b
}

// SetHeaders replaces all headers (MQTT 5.0 user properties).
func (b *MqttRecordBuilder) SetHeaders(headers map[string]string) *MqttRecordBuilder {
	b.headers = headers
	return b
}

// SetHeader sets a single header key-value pair.
func (b *MqttRecordBuilder) SetHeader(key, value string) *MqttRecordBuilder {
	b.headers[key] = value
	return b
}

// Build creates the MqttRecord.
func (b *MqttRecordBuilder) Build() *MqttRecord {
	return &MqttRecord{
		Data:     b.data,
		Topic:    b.topic,
		QoS:      b.qos,
		Retained: b.retained,
		Headers:  b.headers,
	}
}
