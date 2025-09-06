package navigation

// State holds all navigation-related state
type State struct {
	Cursor         int
	ViewportOffset int
	ViewportHeight int
	MaxIndex       int
}

// Direction represents movement directions
type Direction string

const (
	DirectionUp       Direction = "up"
	DirectionDown     Direction = "down"
	DirectionLeft     Direction = "left"
	DirectionRight    Direction = "right"
	DirectionPageUp   Direction = "pageup"
	DirectionPageDown Direction = "pagedown"
	DirectionHome     Direction = "home"
	DirectionEnd      Direction = "end"
)

// Event types for navigation changes
type CursorMovedEvent struct {
	OldIndex int
	NewIndex int
}

type ViewportChangedEvent struct {
	Offset int
	Height int
}