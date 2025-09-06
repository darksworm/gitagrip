package sorting

import "gitagrip/internal/logic"

// State holds sorting state
type State struct {
	CurrentMode logic.SortMode
}

// Event types
type SortModeChangedEvent struct {
	OldMode logic.SortMode
	NewMode logic.SortMode
}