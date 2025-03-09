package app

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/kafka-explorer-cli/internal/adapters/consumer/kafka"
	"github.com/kafka-explorer-cli/internal/adapters/search"
	"github.com/kafka-explorer-cli/internal/adapters/storage"
	"github.com/kafka-explorer-cli/internal/adapters/ui"
	"github.com/kafka-explorer-cli/internal/core/domain"
	"github.com/kafka-explorer-cli/internal/core/ports"
)

// Config contains application configuration
type Config struct {
	// Kafka parameters
	Brokers    []string
	Topic      string
	Group      string
	Partitions []int

	// Search parameters
	SearchString  string
	CaseSensitive bool
	UseRegex      bool
	ContextSize   int

	// Output parameters
	OutputFile string
	Limit      int

	// Authentication
	SASL ports.SASLConfig
}

// Explorer represents the main application
type Explorer struct {
	config Config
}

// NewExplorer creates a new Explorer instance
func NewExplorer(config Config) *Explorer {
	return &Explorer{
		config: config,
	}
}

// Run starts the application and blocks until completion
func (e *Explorer) Run(ctx context.Context) error {
	// Create a cancelable context for coordinating component shutdown
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Create searcher
	searchConfig := ports.SearchConfig{
		SearchString:  e.config.SearchString,
		CaseSensitive: e.config.CaseSensitive,
		UseRegex:      e.config.UseRegex,
		ContextSize:   e.config.ContextSize,
	}

	searcher, err := search.NewSearcher(searchConfig)
	if err != nil {
		return fmt.Errorf("failed to create searcher: %w", err)
	}

	// Create Kafka consumer
	consumerConfig := ports.ConsumerConfig{
		Brokers:    e.config.Brokers,
		Topic:      e.config.Topic,
		Group:      e.config.Group,
		Partitions: e.config.Partitions,
		SASL:       e.config.SASL,
	}

	consumer, err := kafka.NewConsumer(consumerConfig)
	if err != nil {
		return fmt.Errorf("failed to create Kafka consumer: %w", err)
	}
	defer consumer.Close()

	// Create JSON writer
	jsonWriter, err := storage.NewJSONWriter(
		e.config.OutputFile,
		searchConfig,
		e.config.Brokers,
		e.config.Topic,
		e.config.Limit,
	)
	if err != nil {
		return fmt.Errorf("failed to create JSON writer: %w", err)
	}
	defer jsonWriter.Close()

	// Create UI
	userInterface := ui.NewTUI(searchConfig, e.config.OutputFile, e.config.Limit)

	// Create message channel
	msgChan := make(chan domain.Message, 1000)

	// Create wait group for goroutines
	var wg sync.WaitGroup

	// Start the UI
	uiDone := userInterface.Start()

	// Start a goroutine to update the UI
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				metrics := consumer.GetMetrics()
				userInterface.UpdateMetrics(metrics)
			}
		}
	}()

	// Start a goroutine to search for messages
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-msgChan:
				if !ok {
					// Channel is closed
					return
				}
				if result, found := searcher.Search(msg); found {
					// Write the result to JSON
					if err := jsonWriter.WriteResult(result); err != nil {
						fmt.Fprintf(os.Stderr, "Error writing result: %v\n", err)
					}

					// Update match count
					consumer.UpdateMatchCount(msg.Partition)

					// Send the result to the UI
					userInterface.DisplayResult(result)

					// Check if we've found enough matches
					if e.config.Limit > 0 && jsonWriter.GetCount() >= e.config.Limit {
						// We found enough matches, cancel the context
						cancel()
						return
					}
				}
			}
		}
	}()

	// Start reading messages
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := consumer.ReadMessages(ctx, msgChan); err != nil {
			fmt.Fprintf(os.Stderr, "Error reading messages: %v\n", err)
			cancel()
		}
	}()

	// Wait for UI to finish (user quitting) or context to be canceled
	select {
	case <-uiDone:
		// UI was closed by user
		cancel()
	case <-ctx.Done():
		// Context was canceled elsewhere
		userInterface.Stop()
	}

	// Wait for all goroutines to complete
	wg.Wait()

	return nil
}
