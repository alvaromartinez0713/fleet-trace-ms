package kafka

import (
	"context"
	"errors"
	"fmt"
	"io"

	kafkago "github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// Consumer reads messages as a member of a consumer group.
type Consumer struct {
	reader *kafkago.Reader
}

// NewConsumer validates cfg and returns a consumer group reader for the given topics.
// Offsets are committed synchronously after each message is handled.
func NewConsumer(cfg Config, topics []string) (*Consumer, error) {
	if err := cfg.validate(true); err != nil {
		return nil, err
	}
	if len(topics) == 0 {
		return nil, errors.New("kafka: at least one topic is required")
	}
	for _, t := range topics {
		if t == "" {
			return nil, errors.New("kafka: topic names must not be empty")
		}
	}

	readerCfg := kafkago.ReaderConfig{
		Brokers:          cfg.Brokers,
		GroupID:          cfg.GroupID,
		GroupTopics:      topics,
		Dialer:           &kafkago.Dialer{ClientID: cfg.ClientID, Timeout: cfg.DialTimeout},
		ReadBatchTimeout: cfg.ReadTimeout,
	}
	// NewReader panics on an invalid config, so validate first and return the error instead.
	if err := readerCfg.Validate(); err != nil {
		return nil, fmt.Errorf("kafka: invalid reader config: %w", err)
	}

	return &Consumer{reader: kafkago.NewReader(readerCfg)}, nil
}

// Consume fetches messages and calls handler for each one, until ctx is cancelled.
//
// The offset is committed only after handler returns nil. If handler returns an error, Consume
// stops and returns it without committing, so the message is redelivered after a restart.
// Retry and dead-letter policy belong to the caller. Consume returns nil when ctx is cancelled
// or the reader is closed.
func (c *Consumer) Consume(ctx context.Context, handler Handler) error {
	for {
		raw, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("kafka: fetch message: %w", err)
		}

		msg := fromKafkaMessage(raw)
		msgCtx := otel.GetTextMapPropagator().Extract(ctx, headerCarrier{headers: &msg.Headers})
		msgCtx, span := otel.Tracer(tracerName).Start(msgCtx, "kafka consume",
			oteltrace.WithSpanKind(oteltrace.SpanKindConsumer),
			oteltrace.WithAttributes(attribute.String("messaging.destination.name", msg.Topic)),
		)

		if err := handler(msgCtx, msg); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "handler failed")
			span.End()
			return fmt.Errorf("kafka: handle %s[%d]@%d: %w", raw.Topic, raw.Partition, raw.Offset, err)
		}
		span.End()

		if err := c.reader.CommitMessages(ctx, raw); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("kafka: commit %s[%d]@%d: %w", raw.Topic, raw.Partition, raw.Offset, err)
		}
	}
}

// Close stops the reader and leaves the consumer group.
func (c *Consumer) Close() error {
	return c.reader.Close()
}
