package domain

import (
	"encoding/json"
	"time"
)

// Message represents a Kafka message with metadata
type Message struct {
	Key       string            `json:"key"`
	Value     string            `json:"value"`
	Topic     string            `json:"topic"`
	Partition int               `json:"partition"`
	Offset    int64             `json:"offset"`
	Time      time.Time         `json:"timestamp"`
	Headers   map[string]string `json:"headers,omitempty"`
}

// MatchResult represents a search match in a message
type MatchResult struct {
	Key       string            `json:"key"`
	Value     json.RawMessage   `json:"value"`
	Topic     string            `json:"topic"`
	Partition int               `json:"partition"`
	Offset    int64             `json:"offset"`
	Time      time.Time         `json:"timestamp"`
	Headers   map[string]string `json:"headers,omitempty"`
}

// PartitionMetrics tracks the metrics for a single partition
type PartitionMetrics struct {
	Partition        int       `json:"partition"`
	MessagesRead     int64     `json:"messages_read"`
	LastMessagesRead int64     `json:"-"` // For speed calculation
	MatchesFound     int       `json:"matches_found"`
	MessagesTotal    int64     `json:"messages_total,omitempty"` // May be unknown for some partitions
	StartOffset      int64     `json:"start_offset"`
	CurrentOffset    int64     `json:"current_offset"`
	EndOffset        int64     `json:"end_offset,omitempty"` // May be unknown for some partitions
	MsgPerSecond     float64   `json:"messages_per_second"`
	LastUpdated      time.Time `json:"-"`                 // For internal use
	LastMessageTime  time.Time `json:"last_message_time"` // Time of last read message
}

// Metrics tracks overall metrics for the search process
type Metrics struct {
	TotalMessagesRead int64              `json:"total_messages_read"`
	TotalMatchesFound int                `json:"total_matches_found"`
	AvgMsgPerSecond   float64            `json:"avg_messages_per_second"`
	StartTime         time.Time          `json:"start_time"`
	ElapsedTime       time.Duration      `json:"elapsed_time"`
	PartitionMetrics  []PartitionMetrics `json:"partition_metrics"`
}

// SearchResults represents the final output format for search results
type SearchResults struct {
	Brokers         []string       `json:"brokers"`
	Topic           string         `json:"topic"`
	SearchString    string         `json:"searchString"`
	Limit           int            `json:"limit"`
	IsCaseSensitive bool           `json:"isCaseSensitive"`
	IsRegexpSearch  bool           `json:"isRegexpSearch"`
	Matches         []*MatchResult `json:"matches"`
}
