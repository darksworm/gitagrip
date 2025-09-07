package navigation

import (
	"gitagrip/internal/ui/services/events"
)

// Service handles all navigation logic
type Service struct {
	state    *State
	bus      events.EventBus
	queryFn  func() int // Function to get max index from query service
}

// NewService creates a new navigation service
func NewService(bus events.EventBus) *Service {
	return &Service{
		state: &State{
			Cursor:         0,
			ViewportOffset: 0,
			ViewportHeight: 20, // Default, will be updated
			MaxIndex:       0,
		},
		bus: bus,
	}
}

// SetQueryFunction sets the function to query max index
func (s *Service) SetQueryFunction(fn func() int) {
	s.queryFn = fn
}

// GetCursor returns current cursor position
func (s *Service) GetCursor() int {
	return s.state.Cursor
}

// GetViewportOffset returns current viewport offset
func (s *Service) GetViewportOffset() int {
	return s.state.ViewportOffset
}

// GetViewportHeight returns current viewport height
func (s *Service) GetViewportHeight() int {
	return s.state.ViewportHeight
}

// SetViewportHeight updates viewport height
func (s *Service) SetViewportHeight(height int) {
	// Reserve space for header, status bar, help
	effectiveHeight := height - 8
	if effectiveHeight < 1 {
		effectiveHeight = 1
	}
	s.state.ViewportHeight = effectiveHeight
	s.ensureVisible()
}

// Navigate handles navigation in a direction
func (s *Service) Navigate(direction Direction) {
	oldCursor := s.state.Cursor
	
	switch direction {
	case DirectionUp:
		s.moveUp()
	case DirectionDown:
		s.moveDown()
	case DirectionPageUp:
		s.pageUp()
	case DirectionPageDown:
		s.pageDown()
	case DirectionHome:
		s.moveToStart()
	case DirectionEnd:
		s.moveToEnd()
	}
	
	if oldCursor != s.state.Cursor {
		s.bus.Publish(CursorMovedEvent{
			OldIndex: oldCursor,
			NewIndex: s.state.Cursor,
		})
	}
}

// MoveToIndex moves cursor to specific index
func (s *Service) MoveToIndex(index int) {
	if s.queryFn != nil {
		s.state.MaxIndex = s.queryFn()
	}
	
	oldCursor := s.state.Cursor
	s.state.Cursor = s.clampIndex(index)
	s.ensureVisible()
	
	if oldCursor != s.state.Cursor {
		s.bus.Publish(CursorMovedEvent{
			OldIndex: oldCursor,
			NewIndex: s.state.Cursor,
		})
	}
}

// Internal navigation methods
func (s *Service) moveUp() {
	if s.state.Cursor > 0 {
		s.state.Cursor--
		s.ensureVisible()
	}
}

func (s *Service) moveDown() {
	if s.queryFn != nil {
		s.state.MaxIndex = s.queryFn()
	}
	if s.state.Cursor < s.state.MaxIndex {
		s.state.Cursor++
		s.ensureVisible()
	}
}

func (s *Service) pageUp() {
	pageSize := s.state.ViewportHeight - 1
	target := s.state.Cursor - pageSize
	s.state.Cursor = s.clampIndex(target)
	
	// Also scroll viewport up
	s.state.ViewportOffset -= pageSize
	if s.state.ViewportOffset < 0 {
		s.state.ViewportOffset = 0
	}
	s.ensureVisible()
}

func (s *Service) pageDown() {
	if s.queryFn != nil {
		s.state.MaxIndex = s.queryFn()
	}
	
	pageSize := s.state.ViewportHeight - 1
	target := s.state.Cursor + pageSize
	s.state.Cursor = s.clampIndex(target)
	s.ensureVisible()
}

func (s *Service) moveToStart() {
	s.state.Cursor = 0
	s.state.ViewportOffset = 0
}

func (s *Service) moveToEnd() {
	if s.queryFn != nil {
		s.state.MaxIndex = s.queryFn()
	}
	s.state.Cursor = s.state.MaxIndex
	s.ensureVisible()
}

// Helper methods
func (s *Service) clampIndex(index int) int {
	if index < 0 {
		return 0
	}
	if s.queryFn != nil {
		s.state.MaxIndex = s.queryFn()
	}
	if index > s.state.MaxIndex {
		return s.state.MaxIndex
	}
	return index
}

func (s *Service) ensureVisible() {
	// Ensure cursor is visible within viewport
	if s.state.Cursor < s.state.ViewportOffset {
		s.state.ViewportOffset = s.state.Cursor
		s.bus.Publish(ViewportChangedEvent{
			Offset: s.state.ViewportOffset,
			Height: s.state.ViewportHeight,
		})
	} else if s.state.Cursor >= s.state.ViewportOffset+s.state.ViewportHeight {
		s.state.ViewportOffset = s.state.Cursor - s.state.ViewportHeight + 1
		s.bus.Publish(ViewportChangedEvent{
			Offset: s.state.ViewportOffset,
			Height: s.state.ViewportHeight,
		})
	}
}