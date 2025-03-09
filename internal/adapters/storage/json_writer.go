package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/kafka-explorer-cli/internal/core/domain"
	"github.com/kafka-explorer-cli/internal/core/ports"
)

// JSONWriter implements ports.ResultWriter for writing results to a JSON file
type JSONWriter struct {
	file          *os.File
	mutex         sync.Mutex
	config        ports.SearchConfig
	brokers       []string
	topic         string
	limit         int
	results       domain.SearchResults
	matchesBuffer []*domain.MatchResult
}

// NewJSONWriter creates a new JSONWriter
func NewJSONWriter(filename string, config ports.SearchConfig, brokers []string, topic string, limit int) (ports.ResultWriter, error) {
	file, err := os.Create(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create output file: %w", err)
	}

	// Start the JSON array
	if _, err := file.WriteString(""); err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to write to output file: %w", err)
	}

	return &JSONWriter{
		file:    file,
		config:  config,
		brokers: brokers,
		topic:   topic,
		limit:   limit,
		results: domain.SearchResults{
			Brokers:         brokers,
			Topic:           topic,
			SearchString:    config.SearchString,
			Limit:           limit,
			IsCaseSensitive: config.CaseSensitive,
			IsRegexpSearch:  config.UseRegex,
			Matches:         []*domain.MatchResult{},
		},
		matchesBuffer: make([]*domain.MatchResult, 0, 100), // Буфер для оптимизации
	}, nil
}

// WriteResult writes a search result to the buffer
func (w *JSONWriter) WriteResult(result *domain.MatchResult) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	// Добавляем результат в буфер
	w.matchesBuffer = append(w.matchesBuffer, result)

	// Если буфер достаточно большой, добавляем его к результатам
	if len(w.matchesBuffer) >= 100 {
		w.results.Matches = append(w.results.Matches, w.matchesBuffer...)
		w.matchesBuffer = make([]*domain.MatchResult, 0, 100)
	}

	return nil
}

// Close closes the JSONWriter and finalizes the JSON file
func (w *JSONWriter) Close() error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	// Добавляем оставшиеся результаты из буфера
	if len(w.matchesBuffer) > 0 {
		w.results.Matches = append(w.results.Matches, w.matchesBuffer...)
	}

	// Сериализуем все результаты в один JSON-объект
	encoder := json.NewEncoder(w.file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(w.results); err != nil {
		return fmt.Errorf("failed to encode results: %w", err)
	}

	return w.file.Close()
}

// GetCount returns the number of results written
func (w *JSONWriter) GetCount() int {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	return len(w.results.Matches) + len(w.matchesBuffer)
}
