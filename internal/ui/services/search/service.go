package search

import (
	"log"
	"strings"
	
	"gitagrip/internal/domain"
	"gitagrip/internal/eventbus"
)

// Service handles search functionality
type Service struct {
	state       *State
	bus         eventbus.EventBus
	matcherFn   func(string) []MatchResult // Function to find matches
	navigateFn  func(int)                   // Function to navigate to index
}

// MatchResult represents a search match
type MatchResult struct {
	Index      int
	Name       string
	Path       string
	Repository *domain.Repository
	IsGroup    bool
}

// NewService creates a new search service
func NewService(bus eventbus.EventBus) *Service {
	return &Service{
		state: &State{
			Query:        "",
			Matches:      nil,
			CurrentMatch: 0,
		},
		bus: bus,
	}
}

// SetMatcherFunction sets the function to find matches
func (s *Service) SetMatcherFunction(fn func(string) []MatchResult) {
	s.matcherFn = fn
}

// SetNavigateFunction sets the function to navigate to an index
func (s *Service) SetNavigateFunction(fn func(int)) {
	s.navigateFn = fn
}

// StartSearch begins a new search
func (s *Service) StartSearch(query string) {
	if query == s.state.Query {
		return // Same search
	}
	
	s.state.Query = query
	s.bus.Publish(SearchStartedEvent{Query: query})
	
	if query == "" {
		s.clearSearch()
		return
	}
	
	s.performSearch()
}

// ClearSearch clears the current search
func (s *Service) ClearSearch() {
	s.clearSearch()
}

// NavigateNext moves to the next search result
func (s *Service) NavigateNext() {
	if len(s.state.Matches) == 0 {
		return
	}
	
	oldMatch := s.state.CurrentMatch
	s.state.CurrentMatch = (s.state.CurrentMatch + 1) % len(s.state.Matches)
	
	s.navigateToCurrentMatch()
	
	s.bus.Publish(SearchNavigatedEvent{
		OldIndex: s.state.Matches[oldMatch],
		NewIndex: s.state.Matches[s.state.CurrentMatch],
	})
}

// NavigatePrevious moves to the previous search result
func (s *Service) NavigatePrevious() {
	if len(s.state.Matches) == 0 {
		return
	}
	
	oldMatch := s.state.CurrentMatch
	s.state.CurrentMatch--
	if s.state.CurrentMatch < 0 {
		s.state.CurrentMatch = len(s.state.Matches) - 1
	}
	
	s.navigateToCurrentMatch()
	
	s.bus.Publish(SearchNavigatedEvent{
		OldIndex: s.state.Matches[oldMatch],
		NewIndex: s.state.Matches[s.state.CurrentMatch],
	})
}

// GetQuery returns the current search query
func (s *Service) GetQuery() string {
	return s.state.Query
}

// GetMatchCount returns the number of matches
func (s *Service) GetMatchCount() int {
	return len(s.state.Matches)
}

// GetCurrentMatchIndex returns the current match index in results
func (s *Service) GetCurrentMatchIndex() int {
	if len(s.state.Matches) == 0 {
		return -1
	}
	return s.state.Matches[s.state.CurrentMatch]
}

// IsMatch checks if an index is a search match
func (s *Service) IsMatch(index int) bool {
	for _, match := range s.state.Matches {
		if match == index {
			return true
		}
	}
	return false
}

// Internal methods
func (s *Service) performSearch() {
	if s.matcherFn == nil {
		return
	}
	
	// Store old matches to detect changes
	oldMatches := make([]int, len(s.state.Matches))
	copy(oldMatches, s.state.Matches)
	
	// Get matches from matcher function
	results := s.matcherFn(s.state.Query)
	
	// Extract indices
	s.state.Matches = nil
	for _, result := range results {
		s.state.Matches = append(s.state.Matches, result.Index)
	}
	
	// Check if matches changed
	matchesChanged := len(oldMatches) != len(s.state.Matches)
	if !matchesChanged && len(oldMatches) > 0 {
		for i, match := range oldMatches {
			if i >= len(s.state.Matches) || match != s.state.Matches[i] {
				matchesChanged = true
				break
			}
		}
	}
	
	// Reset current match if matches changed
	if matchesChanged {
		s.state.CurrentMatch = 0
	} else if s.state.CurrentMatch >= len(s.state.Matches) {
		s.state.CurrentMatch = 0
	}
	
	log.Printf("Search completed for '%s': found %d matches", s.state.Query, len(s.state.Matches))
	
	firstMatch := -1
	if len(s.state.Matches) > 0 {
		firstMatch = s.state.Matches[0]
	}
	
	s.bus.Publish(SearchCompletedEvent{
		Query:      s.state.Query,
		MatchCount: len(s.state.Matches),
		FirstMatch: firstMatch,
	})
}

func (s *Service) clearSearch() {
	s.state.Query = ""
	s.state.Matches = nil
	s.state.CurrentMatch = 0
	
	s.bus.Publish(SearchClearedEvent{})
}

func (s *Service) navigateToCurrentMatch() {
	if s.navigateFn == nil || len(s.state.Matches) == 0 {
		return
	}
	
	targetIndex := s.state.Matches[s.state.CurrentMatch]
	s.navigateFn(targetIndex)
}

// Highlight helpers for UI
func (s *Service) ShouldHighlight(text string) bool {
	if s.state.Query == "" {
		return false
	}
	return strings.Contains(strings.ToLower(text), strings.ToLower(s.state.Query))
}