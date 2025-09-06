package selection

// State holds selection state
type State struct {
	SelectedRepos map[string]bool
	LastSelected  int // For shift-selection
}

// Event types
type SelectionChangedEvent struct {
	Added   []string
	Removed []string
	Total   int
}

type SelectionClearedEvent struct{}

type AllSelectedEvent struct {
	Paths []string
}