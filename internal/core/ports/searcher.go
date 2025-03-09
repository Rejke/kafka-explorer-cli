package ports

import (
	"github.com/kafka-explorer-cli/internal/core/domain"
)

// SearchConfig contains configuration for searching
type SearchConfig struct {
	SearchString  string
	CaseSensitive bool
	UseRegex      bool
	ContextSize   int // Number of characters to include before and after the match
}

// Searcher defines the interface for search functionality
type Searcher interface {
	// Search looks for matches in a message and returns a result if found
	Search(message domain.Message) (*domain.MatchResult, bool)
}