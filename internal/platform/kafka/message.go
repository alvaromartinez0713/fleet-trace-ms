package kafka

import (
	"context"
	"time"

	kafkago "github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel/propagation"
)

// Header is a key/value pair attached to a message.
type Header struct {
	Key   string
	Value []byte
}

// Message is a message published or consumed through this package.
type Message struct {
	Topic     string
	Partition int
	Offset    int64
	Key       []byte
	Value     []byte
	Headers   []Header
	Time      time.Time
}

// Handler processes one consumed message. Returning nil commits the offset.
// Returning an error stops Consume without committing, so the message is redelivered.
type Handler func(ctx context.Context, msg Message) error

// headerCarrier adapts message headers to the OpenTelemetry propagation carrier interface.
type headerCarrier struct {
	headers *[]Header
}

var _ propagation.TextMapCarrier = headerCarrier{}

// Get returns the value of the first header with the given key, or an empty string.
func (c headerCarrier) Get(key string) string {
	for _, h := range *c.headers {
		if h.Key == key {
			return string(h.Value)
		}
	}
	return ""
}

// Set replaces the value of the first header with the given key, or appends a new header.
func (c headerCarrier) Set(key, value string) {
	for i := range *c.headers {
		if (*c.headers)[i].Key == key {
			(*c.headers)[i].Value = []byte(value)
			return
		}
	}
	*c.headers = append(*c.headers, Header{Key: key, Value: []byte(value)})
}

// Keys returns every header key.
func (c headerCarrier) Keys() []string {
	keys := make([]string, 0, len(*c.headers))
	for _, h := range *c.headers {
		keys = append(keys, h.Key)
	}
	return keys
}

// toKafkaHeaders converts headers to the kafka-go type.
func toKafkaHeaders(headers []Header) []kafkago.Header {
	out := make([]kafkago.Header, 0, len(headers))
	for _, h := range headers {
		out = append(out, kafkago.Header{Key: h.Key, Value: h.Value})
	}
	return out
}

// fromKafkaMessage converts a kafka-go message into this package's type.
func fromKafkaMessage(m kafkago.Message) Message {
	headers := make([]Header, 0, len(m.Headers))
	for _, h := range m.Headers {
		headers = append(headers, Header{Key: h.Key, Value: h.Value})
	}
	return Message{
		Topic:     m.Topic,
		Partition: m.Partition,
		Offset:    m.Offset,
		Key:       m.Key,
		Value:     m.Value,
		Headers:   headers,
		Time:      m.Time,
	}
}
