package kafka

import (
	"context"
	"fmt"

	kafkago "github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const tracerName = "github.com/alvaromartinez0713/fleet-trace-ms/internal/platform/kafka"

// Producer publishes messages to Kafka. Messages with the same key go to the same partition.
type Producer struct {
	writer *kafkago.Writer
}

// NewProducer validates cfg and returns a producer. It does not connect until the first publish.
func NewProducer(cfg Config) (*Producer, error) {
	if err := cfg.validate(false); err != nil {
		return nil, err
	}
	return &Producer{
		writer: &kafkago.Writer{
			Addr:                   kafkago.TCP(cfg.Brokers...),
			Balancer:               &kafkago.Hash{},
			RequiredAcks:           kafkago.RequireAll,
			AllowAutoTopicCreation: false,
			WriteTimeout:           cfg.WriteTimeout,
			ReadTimeout:            cfg.ReadTimeout,
			Transport: &kafkago.Transport{
				ClientID:    cfg.ClientID,
				DialTimeout: cfg.DialTimeout,
			},
		},
	}, nil
}

// Publish writes one message to topic. The trace context from ctx is injected into the headers,
// so a consumer can continue the same trace. The caller's headers slice is not modified.
func (p *Producer) Publish(ctx context.Context, topic string, key, value []byte, headers []Header) error {
	ctx, span := otel.Tracer(tracerName).Start(ctx, "kafka publish",
		oteltrace.WithSpanKind(oteltrace.SpanKindProducer),
		oteltrace.WithAttributes(attribute.String("messaging.destination.name", topic)),
	)
	defer span.End()

	out := append([]Header(nil), headers...)
	otel.GetTextMapPropagator().Inject(ctx, headerCarrier{headers: &out})

	err := p.writer.WriteMessages(ctx, kafkago.Message{
		Topic:   topic,
		Key:     key,
		Value:   value,
		Headers: toKafkaHeaders(out),
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "publish failed")
		return fmt.Errorf("kafka: publish to %s: %w", topic, err)
	}
	return nil
}

// Close flushes pending writes and closes the connection.
func (p *Producer) Close() error {
	return p.writer.Close()
}
