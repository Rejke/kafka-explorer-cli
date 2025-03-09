package kafka

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kafka-explorer-cli/internal/core/domain"
	"github.com/kafka-explorer-cli/internal/core/ports"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"
)

// Consumer implements the ports.Consumer interface for Kafka
type Consumer struct {
	config        ports.ConsumerConfig
	readers       map[int]*kafka.Reader
	readerMutex   sync.Mutex
	metrics       *domain.Metrics
	metricMutex   sync.RWMutex
	channelClosed bool
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(config ports.ConsumerConfig) (ports.Consumer, error) {
	c := &Consumer{
		config:  config,
		readers: make(map[int]*kafka.Reader),
		metrics: &domain.Metrics{
			StartTime:        time.Now(),
			PartitionMetrics: []domain.PartitionMetrics{},
		},
		channelClosed: false,
	}

	// Initialize metrics for each partition
	partitions, err := c.getPartitions()
	if err != nil {
		return nil, err
	}

	// Filter partitions if specific ones are requested
	usePartitions := partitions
	if len(config.Partitions) > 0 {
		usePartitions = make([]int, 0)
		for _, p := range partitions {
			for _, requestedP := range config.Partitions {
				if p == requestedP {
					usePartitions = append(usePartitions, p)
					break
				}
			}
		}
	}

	// Initialize metrics for each partition
	for _, partition := range usePartitions {
		c.metrics.PartitionMetrics = append(c.metrics.PartitionMetrics, domain.PartitionMetrics{
			Partition:   partition,
			LastUpdated: time.Now(),
		})
	}

	return c, nil
}

// createSASLMechanism creates the appropriate SASL mechanism based on configuration
func createSASLMechanism(config ports.SASLConfig) (sasl.Mechanism, error) {
	switch config.Mechanism {
	case "PLAIN":
		return plain.Mechanism{
			Username: config.Username,
			Password: config.Password,
		}, nil

	case "SCRAM-SHA-256":
		mechanism, err := scram.Mechanism(scram.SHA256, config.Username, config.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to create SCRAM-SHA-256 mechanism: %w", err)
		}
		return mechanism, nil

	case "SCRAM-SHA-512":
		mechanism, err := scram.Mechanism(scram.SHA512, config.Username, config.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to create SCRAM-SHA-512 mechanism: %w", err)
		}
		return mechanism, nil

	default:
		return nil, fmt.Errorf("unsupported SASL mechanism: %s", config.Mechanism)
	}
}

// getPartitions retrieves all partitions for the topic
func (c *Consumer) getPartitions() ([]int, error) {
	var logBuffer strings.Builder

	logBuffer.WriteString("========= ATTEMPTING TO CONNECT TO KAFKA ==========\n")
	logBuffer.WriteString(fmt.Sprintf("Topic: %s, Brokers: %s\n", c.config.Topic, strings.Join(c.config.Brokers, ",")))

	if c.config.SASL.Enabled {
		logBuffer.WriteString(fmt.Sprintf("SASL: enabled, Mechanism: %s, Username: %s, Protocol: %s\n",
			c.config.SASL.Mechanism, c.config.SASL.Username, c.config.SASL.Protocol))
	} else {
		logBuffer.WriteString("SASL: disabled\n")
	}

	// Create dialer with longer timeout
	dialer := &kafka.Dialer{
		Timeout: 30 * time.Second,
	}

	// Configure SASL authentication
	if c.config.SASL.Enabled {
		mechanism, err := createSASLMechanism(c.config.SASL)
		if err != nil {
			return nil, err
		}

		// Set the SASL mechanism on the dialer
		dialer.SASLMechanism = mechanism
		logBuffer.WriteString("SASL mechanism configured successfully\n")
	}

	// Set TLS config for SSL protocols
	if c.config.SASL.Protocol == "SSL" || c.config.SASL.Protocol == "SASL_SSL" {
		dialer.TLS = &tls.Config{
			InsecureSkipVerify: true, // Note: In production, verify certificates
		}
		logBuffer.WriteString("TLS configured for SSL connection\n")
	}

	// Output buffered logs
	fmt.Fprint(os.Stderr, logBuffer.String())

	// Try to connect to each broker until successful
	var lastErr error
	for _, broker := range c.config.Brokers {
		// Connect to Kafka
		fmt.Fprintf(os.Stderr, "Attempting to connect to broker: %s\n", broker)

		// Create context with timeout for connection attempt
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Determine network protocol
		network := "tcp"
		if c.config.SASL.Protocol == "SSL" || c.config.SASL.Protocol == "SASL_SSL" {
			network = "ssl"
		}

		fmt.Fprintf(os.Stderr, "Using network protocol: %s\n", network)

		conn, err := dialer.DialLeader(ctx, network, broker, c.config.Topic, 0)
		if err != nil {
			lastErr = fmt.Errorf("failed to connect to broker %s: %w", broker, err)
			fmt.Fprintf(os.Stderr, "Connection error: %v, trying next broker...\n", err)
			continue
		}

		// Successfully connected
		fmt.Fprintf(os.Stderr, "Successfully connected to broker: %s\n", broker)

		// Set read/write deadlines to prevent hanging
		if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to set read deadline: %w", err)
		}

		partitions, err := conn.ReadPartitions()
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to get partitions: %w", err)
		}

		// Reset deadline and close properly
		if err := conn.SetReadDeadline(time.Time{}); err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to reset read deadline: %w", err)
		}
		conn.Close()

		result := make([]int, 0, len(partitions))
		for _, p := range partitions {
			if p.Topic == c.config.Topic {
				result = append(result, p.ID)
			}
		}

		if len(result) == 0 {
			fmt.Fprintf(os.Stderr, "Warning: No partitions found for topic %s\n", c.config.Topic)
		} else {
			fmt.Fprintf(os.Stderr, "Found %d partitions for topic %s\n", len(result), c.config.Topic)
		}

		return result, nil
	}

	// All brokers failed
	return nil, fmt.Errorf("failed to connect to any Kafka broker: %w", lastErr)
}

// getReader creates or returns an existing Kafka reader for a partition
func (c *Consumer) getReader(partition int) (*kafka.Reader, error) {
	c.readerMutex.Lock()
	defer c.readerMutex.Unlock()

	if reader, ok := c.readers[partition]; ok {
		return reader, nil
	}

	// Configure the reader with optimized settings
	config := kafka.ReaderConfig{
		Brokers:         c.config.Brokers,
		Topic:           c.config.Topic,
		Partition:       partition,
		MinBytes:        10e3,                   // 10KB minimum packet size
		MaxBytes:        10e6,                   // 10MB maximum packet size
		MaxWait:         500 * time.Millisecond, // Maximum wait time for packet filling
		ReadLagInterval: -1,                     // Disable lag tracking for better performance
		CommitInterval:  0,                      // Disable automatic commit for better performance
		QueueCapacity:   100,                    // Increase internal queue size
		ReadBackoffMin:  100 * time.Millisecond,
		ReadBackoffMax:  1 * time.Second,
	}

	// Configure SASL authentication for the reader
	if c.config.SASL.Enabled {
		mechanism, err := createSASLMechanism(c.config.SASL)
		if err != nil {
			return nil, err
		}

		dialer := &kafka.Dialer{
			Timeout:       30 * time.Second,
			SASLMechanism: mechanism,
		}

		// Set TLS for SSL protocols
		if c.config.SASL.Protocol == "SSL" || c.config.SASL.Protocol == "SASL_SSL" {
			dialer.TLS = &tls.Config{
				InsecureSkipVerify: true, // Note: In production, verify certificates
			}
		}

		config.Dialer = dialer
	}

	reader := kafka.NewReader(config)
	c.readers[partition] = reader
	return reader, nil
}

// ReadMessages reads messages from Kafka topic partitions
func (c *Consumer) ReadMessages(ctx context.Context, msgChan chan<- domain.Message) error {
	// Get partitions to read
	partitions, err := c.getPartitions()
	if err != nil {
		return fmt.Errorf("failed to get partitions: %w", err)
	}

	// Filter partitions if specific ones are requested
	usePartitions := partitions
	if len(c.config.Partitions) > 0 {
		usePartitions = make([]int, 0)
		for _, p := range partitions {
			for _, requestedP := range c.config.Partitions {
				if p == requestedP {
					usePartitions = append(usePartitions, p)
					break
				}
			}
		}
	}

	// Check if there are partitions to read
	if len(usePartitions) == 0 {
		return fmt.Errorf("no partitions available for topic %s", c.config.Topic)
	}

	// Start metrics update goroutine
	go c.updateMetrics(ctx)

	// Create buffered internal channel for messages between goroutines
	// This prevents blocking when passing messages
	bufferSize := 1000 // Buffer size for internal channel
	internalMsgChan := make(chan domain.Message, bufferSize)

	// Create WaitGroup to wait for all goroutines
	var wg sync.WaitGroup

	// Start a goroutine for each partition
	for _, partition := range usePartitions {
		wg.Add(1)
		go func(p int) {
			defer wg.Done()
			c.readPartition(ctx, p, internalMsgChan)
		}(partition)
	}

	// Start a goroutine to pass messages from internal channel to user channel
	// This prevents blocking when reading from partitions
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-internalMsgChan:
				if !ok {
					return
				}
				select {
				case <-ctx.Done():
					return
				case msgChan <- msg:
					// Message sent successfully
				}
			}
		}
	}()

	// Wait for all goroutines to complete or context to be done
	go func() {
		wg.Wait()
		// Close internal message channel after all goroutines are done
		c.readerMutex.Lock()
		defer c.readerMutex.Unlock()

		// Close internal channel
		close(internalMsgChan)

		// Check if channel wasn't already closed
		if !c.channelClosed {
			close(msgChan)
			c.channelClosed = true
		}
	}()

	return nil
}

// readPartition reads messages from a single Kafka partition
func (c *Consumer) readPartition(ctx context.Context, partition int, msgChan chan<- domain.Message) {
	// Get reader for partition
	reader, err := c.getReader(partition)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting reader for partition %d: %v\n", partition, err)
		return
	}

	// Set timeout for reading messages
	readTimeout := 5 * time.Second // Reduce timeout from 10 to 5 seconds
	batchSize := 100               // Batch size for reading

	// Ensure reader is closed when function exits
	defer func() {
		c.readerMutex.Lock()
		defer c.readerMutex.Unlock()

		// Close reader for this partition
		if reader != nil {
			if err := reader.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "Error closing reader for partition %d: %v\n", partition, err)
			}
		}
	}()

	// Buffer for batch reading
	messageBatch := make([]kafka.Message, 0, batchSize)

	for {
		// Check if context is done
		select {
		case <-ctx.Done():
			return
		default:
			// Continue reading
		}

		// Create context with timeout for reading
		readCtx, cancel := context.WithTimeout(ctx, readTimeout)

		// Try to read batch of messages
		messageBatch = messageBatch[:0] // Clear buffer

		// Read up to batchSize messages or until timeout
		for i := 0; i < batchSize; i++ {
			msg, err := reader.ReadMessage(readCtx)

			if err != nil {
				// If this is first iteration and error occurred, handle as before
				if i == 0 {
					// Check if context is done
					if ctx.Err() != nil {
						cancel()
						return
					}

					// Check if timeout expired
					if err == context.DeadlineExceeded {
						// Just continue reading
						break
					}

					// Log other errors
					fmt.Fprintf(os.Stderr, "Error reading from partition %d: %v\n", partition, err)

					// Short pause before retry
					time.Sleep(500 * time.Millisecond) // Reduce pause from 1 second to 500 ms
					break
				}

				// If already read some messages, just stop reading batch
				break
			}

			// Add message to batch
			messageBatch = append(messageBatch, msg)
		}

		cancel() // Free context resources

		// Process read messages
		for _, msg := range messageBatch {
			// Convert message to our format
			kafkaMsg := convertMessage(msg)

			// Update metrics
			c.updatePartitionMetrics(partition, msg.Offset, msg.Time)

			// Send message to channel
			select {
			case <-ctx.Done():
				return
			case msgChan <- kafkaMsg:
				// Message sent successfully
			}
		}
	}
}

// convertMessage converts a kafka-go message to our internal representation
func convertMessage(msg kafka.Message) domain.Message {
	var headers map[string]string

	if len(msg.Headers) > 0 {
		headers = make(map[string]string, len(msg.Headers))
		for _, h := range msg.Headers {
			if len(h.Key) > 0 {
				headers[h.Key] = string(h.Value)
			}
		}
	}

	return domain.Message{
		Key:       string(msg.Key),
		Value:     string(msg.Value),
		Topic:     msg.Topic,
		Partition: msg.Partition,
		Offset:    msg.Offset,
		Time:      msg.Time,
		Headers:   headers,
	}
}

// updatePartitionMetrics updates metrics for a single partition with optimized locking
func (c *Consumer) updatePartitionMetrics(partition int, offset int64, msgTime time.Time) {
	// Do local calculations without locking when possible
	now := time.Now()

	// Use atomic operations for counters and batch metric updates
	// First check if detailed metrics need to be updated
	// Update detailed metrics only every 100 ms to reduce locking
	updateDetailed := false

	// Use shorter read lock
	c.metricMutex.RLock()
	var pm *domain.PartitionMetrics
	for i, metric := range c.metrics.PartitionMetrics {
		if metric.Partition == partition {
			pm = &c.metrics.PartitionMetrics[i]
			// Check if enough time passed since last update
			if now.Sub(pm.LastUpdated) > 100*time.Millisecond {
				updateDetailed = true
			}
			break
		}
	}
	c.metricMutex.RUnlock()

	if pm == nil {
		return // Partition not found
	}

	// Increment message counter atomically
	atomic.AddInt64(&pm.MessagesRead, 1)

	// If detailed metrics need update, do it with full lock
	if updateDetailed {
		c.metricMutex.Lock()
		defer c.metricMutex.Unlock()

		// Update offset only if valid
		if offset >= 0 {
			pm.CurrentOffset = offset
			// Update time of last message from Kafka
			pm.LastMessageTime = msgTime
		}

		// Update speed if enough time passed
		timeSinceLastUpdate := now.Sub(pm.LastUpdated)
		if timeSinceLastUpdate > time.Second {
			messagesDelta := pm.MessagesRead - pm.LastMessagesRead
			pm.MsgPerSecond = float64(messagesDelta) / timeSinceLastUpdate.Seconds()
			pm.LastMessagesRead = pm.MessagesRead
			pm.LastUpdated = now
		}
	}
}

// updateMetrics periodically updates the global metrics with optimized locking
func (c *Consumer) updateMetrics(ctx context.Context) {
	// Use faster metrics update interval for more accurate data
	ticker := time.NewTicker(250 * time.Millisecond) // Reduce interval from 500 to 250 ms
	defer ticker.Stop()

	// Buffer for temporary calculations without locking
	var lastMessagesRead int64
	var lastCalculationTime time.Time

	// Create channel for metrics updates with rate limiting
	// This prevents too frequent updates under high load
	metricUpdateChan := make(chan struct{}, 1)

	// Start goroutine to process metrics updates
	go func() {
		for range metricUpdateChan {
			// Update global metrics with full lock
			c.updateGlobalMetrics()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			close(metricUpdateChan) // Close channel when done
			return
		case <-ticker.C:
			now := time.Now()

			// First read metrics with minimal locking
			var totalMessagesRead int64
			var startTime time.Time

			func() {
				c.metricMutex.RLock()
				defer c.metricMutex.RUnlock()

				// Sum messages from all partitions for more accurate counting
				totalMessagesRead = 0
				for _, pm := range c.metrics.PartitionMetrics {
					totalMessagesRead += pm.MessagesRead
				}

				startTime = c.metrics.StartTime
			}()

			// Calculate metrics without locking
			elapsed := now.Sub(startTime)
			var avgMsgPerSecond float64

			if elapsed > 0 {
				// Calculate overall rate for entire period
				overallAvg := float64(totalMessagesRead) / elapsed.Seconds()

				// Calculate rate for last update interval
				if !lastCalculationTime.IsZero() {
					intervalElapsed := now.Sub(lastCalculationTime)
					messagesDelta := totalMessagesRead - lastMessagesRead

					if intervalElapsed.Seconds() > 0 {
						intervalAvg := float64(messagesDelta) / intervalElapsed.Seconds()
						// Mix two metrics for smoother rate changes
						// Give more weight (70%) to last interval for faster reaction to changes
						avgMsgPerSecond = intervalAvg*0.7 + overallAvg*0.3
					} else {
						avgMsgPerSecond = overallAvg
					}
				} else {
					avgMsgPerSecond = overallAvg
				}
			}

			// Update buffer values for next calculation
			lastMessagesRead = totalMessagesRead
			lastCalculationTime = now

			// Update global metrics with minimal locking
			func() {
				c.metricMutex.Lock()
				defer c.metricMutex.Unlock()

				// Update only necessary fields
				c.metrics.TotalMessagesRead = totalMessagesRead
				c.metrics.AvgMsgPerSecond = avgMsgPerSecond
				c.metrics.ElapsedTime = elapsed
			}()

			// Send signal to update global metrics with rate limiting
			select {
			case metricUpdateChan <- struct{}{}:
				// Signal sent
			default:
				// Channel full, skip update
			}
		}
	}
}

// updateGlobalMetrics updates global metrics based on partition metrics
func (c *Consumer) updateGlobalMetrics() {
	c.metricMutex.Lock()
	defer c.metricMutex.Unlock()

	// Update total message count
	var totalMessagesRead int64
	var totalMatchesFound int

	for _, pm := range c.metrics.PartitionMetrics {
		totalMessagesRead += pm.MessagesRead
		totalMatchesFound += pm.MatchesFound
	}

	c.metrics.TotalMessagesRead = totalMessagesRead
	c.metrics.TotalMatchesFound = totalMatchesFound
}

// GetMetrics returns a copy of the current metrics
func (c *Consumer) GetMetrics() domain.Metrics {
	c.metricMutex.RLock()
	defer c.metricMutex.RUnlock()

	// Make a deep copy of the metrics
	metrics := domain.Metrics{
		TotalMessagesRead: c.metrics.TotalMessagesRead,
		TotalMatchesFound: c.metrics.TotalMatchesFound,
		AvgMsgPerSecond:   c.metrics.AvgMsgPerSecond,
		StartTime:         c.metrics.StartTime,
		ElapsedTime:       c.metrics.ElapsedTime,
		PartitionMetrics:  make([]domain.PartitionMetrics, 0, len(c.metrics.PartitionMetrics)),
	}

	// Use map to prevent partition duplication
	partitionMap := make(map[int]domain.PartitionMetrics)

	// Fill map with latest metrics for each partition
	for _, pm := range c.metrics.PartitionMetrics {
		partitionMap[pm.Partition] = pm
	}

	// Copy unique partition metrics to result
	for _, pm := range partitionMap {
		metrics.PartitionMetrics = append(metrics.PartitionMetrics, pm)
	}

	return metrics
}

// UpdateMatchCount updates the match count for a partition
func (c *Consumer) UpdateMatchCount(partition int) {
	c.metricMutex.Lock()
	defer c.metricMutex.Unlock()

	// Find the partition metrics
	for i, pm := range c.metrics.PartitionMetrics {
		if pm.Partition == partition {
			c.metrics.PartitionMetrics[i].MatchesFound++
			c.metrics.TotalMatchesFound++
			break
		}
	}
}

// Close closes all Kafka readers
func (c *Consumer) Close() error {
	c.readerMutex.Lock()
	defer c.readerMutex.Unlock()

	// Set channel closed flag
	c.channelClosed = true

	// Close all readers
	for _, reader := range c.readers {
		if err := reader.Close(); err != nil {
			return err
		}
	}
	return nil
}