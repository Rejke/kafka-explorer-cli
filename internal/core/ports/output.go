package ports

import (
	"github.com/kafka-explorer-cli/internal/core/domain"
)

// ResultWriter defines the interface for writing search results
type ResultWriter interface {
	// WriteResult writes a search result
	WriteResult(result *domain.MatchResult) error

	// GetCount returns the number of results written so far
	GetCount() int

	// Close finalizes and closes the writer
	Close() error
}