// Package kafka wraps segmentio/kafka-go with a generic producer and consumer group.
//
// The package holds no topic names and no business types. Callers pass topics and
// handlers in. Trace context travels in message headers using the W3C TraceContext format.
package kafka

import (
	"errors"
	"time"
)

// Config holds the connection settings used by the producer and the consumer.
type Config struct {
	Brokers      []string
	ClientID     string
	GroupID      string // Required for the consumer only.
	DialTimeout  time.Duration
	WriteTimeout time.Duration
	ReadTimeout  time.Duration
}

// validate checks the settings. It requires GroupID only when needGroup is true.
func (c Config) validate(needGroup bool) error {
	if len(c.Brokers) == 0 {
		return errors.New("kafka: at least one broker is required")
	}
	if needGroup && c.GroupID == "" {
		return errors.New("kafka: consumer group id is required")
	}
	if c.DialTimeout <= 0 {
		return errors.New("kafka: dial timeout must be positive")
	}
	if c.WriteTimeout <= 0 {
		return errors.New("kafka: write timeout must be positive")
	}
	if c.ReadTimeout <= 0 {
		return errors.New("kafka: read timeout must be positive")
	}
	return nil
}
