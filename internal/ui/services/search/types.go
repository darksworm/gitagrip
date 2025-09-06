package search

// State holds search state
type State struct {
	Query        string
	Matches      []int  // Indices of matching items
	CurrentMatch int    // Current match index in Matches slice
}

// Event types
type SearchStartedEvent struct {
	Query string
}

type SearchCompletedEvent struct {
	Query       string
	MatchCount  int
	FirstMatch  int // Index of first match (-1 if none)
}

type SearchClearedEvent struct{}

type SearchNavigatedEvent struct {
	OldIndex int
	NewIndex int
}