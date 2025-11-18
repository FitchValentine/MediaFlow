package kafka

import (
	"context"
	"strings"
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

// Producer wraps kafka-go Writer with MediaFlow defaults.
type Producer struct {
	writer *kafkago.Writer
}

type ProducerConfig struct {
	Brokers      []string
	Topic        string
	BatchSize    int
	BatchTimeout time.Duration
	Compression  kafkago.Compression
	RequiredAcks kafkago.RequiredAcks
	MaxAttempts  int
}

// NewProducer constructs a Producer from the given configuration.
func NewProducer(cfg ProducerConfig) *Producer {
	return &Producer{
		writer: &kafkago.Writer{
			Addr:         kafkago.TCP(cfg.Brokers...),
			Topic:        cfg.Topic,
			Balancer:     &kafkago.Hash{},
			BatchSize:    cfg.BatchSize,
			BatchTimeout: cfg.BatchTimeout,
			RequiredAcks: cfg.RequiredAcks,
			Compression:  cfg.Compression,
			MaxAttempts:  cfg.MaxAttempts,
		},
	}
}

// Publish sends a Kafka message with optional headers.
func (p *Producer) Publish(ctx context.Context, key []byte, value []byte, headers map[string]string) error {
	msg := kafkago.Message{
		Key:   key,
		Value: value,
		Time:  time.Now().UTC(),
	}

	for k, v := range headers {
		msg.Headers = append(msg.Headers, kafkago.Header{Key: k, Value: []byte(v)})
	}

	return p.writer.WriteMessages(ctx, msg)
}

// Close flushes and closes the underlying writer.
func (p *Producer) Close(ctx context.Context) error {
	return p.writer.Close()
}

// CompressionFromString maps textual codec to kafka-go value.
func CompressionFromString(name string) kafkago.Compression {
	switch strings.ToLower(name) {
	case "gzip":
		return kafkago.Gzip
	case "snappy":
		return kafkago.Snappy
	case "lz4":
		return kafkago.Lz4
	case "zstd":
		return kafkago.Zstd
	default:
		return kafkago.Snappy
	}
}
