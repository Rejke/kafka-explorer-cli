package search

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/kafka-explorer-cli/internal/core/domain"
	"github.com/kafka-explorer-cli/internal/core/ports"
)

// RegexSearcher implements Searcher interface using regular expressions
type RegexSearcher struct {
	config ports.SearchConfig
	regex  *regexp.Regexp
}

// NewSearcher creates a new Searcher based on the provided configuration
func NewSearcher(config ports.SearchConfig) (ports.Searcher, error) {
	s := &RegexSearcher{
		config: config,
	}

	if config.UseRegex {
		// Compile regex with or without case sensitivity
		var err error
		if config.CaseSensitive {
			s.regex, err = regexp.Compile(config.SearchString)
		} else {
			s.regex, err = regexp.Compile("(?i)" + config.SearchString)
		}

		if err != nil {
			return nil, err
		}
	}

	return s, nil
}

// Search implements the Searcher interface
func (s *RegexSearcher) Search(message domain.Message) (*domain.MatchResult, bool) {
	if !s.hasMatch(message.Value) {
		return nil, false
	}

	// Prepare the value as json.RawMessage
	valueToStore := []byte(message.Value)

	// If it's valid JSON, format it nicely
	var jsonObj interface{}
	if err := json.Unmarshal(valueToStore, &jsonObj); err == nil {
		if formattedBytes, err := json.MarshalIndent(jsonObj, "", "  "); err == nil {
			valueToStore = formattedBytes
		}
	}

	return &domain.MatchResult{
		Key:       message.Key,
		Value:     valueToStore,
		Topic:     message.Topic,
		Partition: message.Partition,
		Offset:    message.Offset,
		Time:      message.Time,
		Headers:   message.Headers,
	}, true
}

func (s *RegexSearcher) hasMatch(value string) bool {
	if s.config.UseRegex {
		return s.regex.FindStringIndex(value) != nil
	}

	searchStr := s.config.SearchString
	valueToSearch := value

	if !s.config.CaseSensitive {
		searchStr = strings.ToLower(searchStr)
		valueToSearch = strings.ToLower(value)
	}

	return strings.Contains(valueToSearch, searchStr)
}
