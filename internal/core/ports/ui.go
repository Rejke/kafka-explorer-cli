package ports

import (
	"github.com/kafka-explorer-cli/internal/core/domain"
)

// UI defines the interface for user interface
type UI interface {
	// Start starts the UI and returns a channel to signal when it's done
	Start() <-chan struct{}

	// UpdateMetrics sends updated metrics to the UI
	UpdateMetrics(metrics domain.Metrics)

	// DisplayResult shows a new search result
	DisplayResult(result *domain.MatchResult)

	// Stop gracefully stops the UI
	Stop()
}