package kafka

// KafkaRecord represents a Kafka message with data, headers, key, and partition.
type KafkaRecord struct {
	Data      any
	Headers   map[string]string
	Key       string
	Partition int
}

// KafkaRecordBuilder provides a fluent API for building KafkaRecord messages.
type KafkaRecordBuilder struct {
	data      any
	headers   map[string]string
	key       string
	partition int
}

// NewKafkaRecordBuilder creates a new KafkaRecordBuilder.
func NewKafkaRecordBuilder() *KafkaRecordBuilder {
	return &KafkaRecordBuilder{
		headers: make(map[string]string),
	}
}

// SetData sets the message data.
func (b *KafkaRecordBuilder) SetData(data any) *KafkaRecordBuilder {
	b.data = data
	return b
}

// SetHeaders replaces all headers.
func (b *KafkaRecordBuilder) SetHeaders(headers map[string]string) *KafkaRecordBuilder {
	b.headers = headers
	return b
}

// SetHeader sets a single header key-value pair.
func (b *KafkaRecordBuilder) SetHeader(key, value string) *KafkaRecordBuilder {
	b.headers[key] = value
	return b
}

// SetKey sets the Kafka message key (used for partition assignment).
func (b *KafkaRecordBuilder) SetKey(key string) *KafkaRecordBuilder {
	b.key = key
	return b
}

// SetPartition sets the target partition.
func (b *KafkaRecordBuilder) SetPartition(partition int) *KafkaRecordBuilder {
	b.partition = partition
	return b
}

// Build creates the KafkaRecord.
func (b *KafkaRecordBuilder) Build() *KafkaRecord {
	return &KafkaRecord{
		Data:      b.data,
		Headers:   b.headers,
		Key:       b.key,
		Partition: b.partition,
	}
}
