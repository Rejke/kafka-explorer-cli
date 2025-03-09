package ports

import (
	"context"

	"github.com/kafka-explorer-cli/internal/core/domain"
)

// ConsumerConfig holds configuration for message consumer
type ConsumerConfig struct {
	Brokers    []string
	Topic      string
	Group      string
	Partitions []int
	SASL       SASLConfig
}

// SASLConfig holds SASL authentication configuration
type SASLConfig struct {
	Enabled   bool
	Mechanism string
	Username  string
	Password  string
	Protocol  string
}

// Consumer defines the interface for consuming messages
type Consumer interface {
	// ReadMessages reads messages from the source into the provided channel
	ReadMessages(ctx context.Context, msgChan chan<- domain.Message) error

	// GetMetrics returns current consumer metrics
	GetMetrics() domain.Metrics

	// UpdateMatchCount updates the match counter for a partition
	UpdateMatchCount(partition int)

	// Close closes the consumer and releases resources
	Close() error
}