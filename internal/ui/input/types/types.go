package types

import tea "github.com/charmbracelet/bubbletea"

// Mode represents an input mode
type Mode int

const (
	ModeNormal Mode = iota
	ModeSearch
	ModeFilter
	ModeNewGroup
	ModeMoveToGroup
	ModeDeleteConfirm
	ModeSort
)

// Action represents a command the model should execute
type Action interface {
	Type() string
}

// Context provides read-only access to model state needed for input handling
type Context interface {
	CurrentIndex() int
	TotalItems() int
	HasSelection() bool
	SelectedCount() int
	CurrentRepositoryPath() string
	GetRepoPathAtIndex(index int) string
	IsOnGroup() bool
	CurrentGroupName() string
}

// ModeHandler handles input for a specific mode
type ModeHandler interface {
	// HandleKey processes a key message and returns actions and whether to consume the event
	HandleKey(msg tea.KeyMsg, ctx Context) ([]Action, bool)
	
	// Enter is called when entering this mode
	Enter(ctx Context) []Action
	
	// Exit is called when leaving this mode
	Exit(ctx Context) []Action
	
	// Name returns the mode name for display
	Name() string
}