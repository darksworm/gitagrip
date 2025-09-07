package selection

import (
	"gitagrip/internal/ui/services/events"
)

// Service handles selection logic
type Service struct {
	state   *State
	bus     events.EventBus
	queryFn func(int) string // Function to get repo path at index
}

// NewService creates a new selection service
func NewService(bus events.EventBus) *Service {
	return &Service{
		state: &State{
			SelectedRepos: make(map[string]bool),
			LastSelected:  -1,
		},
		bus: bus,
	}
}

// SetQueryFunction sets the function to query repo paths
func (s *Service) SetQueryFunction(fn func(int) string) {
	s.queryFn = fn
}

// Toggle toggles selection at index
func (s *Service) Toggle(index int) {
	if s.queryFn == nil {
		return
	}
	
	repoPath := s.queryFn(index)
	if repoPath == "" {
		return // Not a repository
	}
	
	var added, removed []string
	
	if s.state.SelectedRepos[repoPath] {
		delete(s.state.SelectedRepos, repoPath)
		removed = append(removed, repoPath)
	} else {
		s.state.SelectedRepos[repoPath] = true
		added = append(added, repoPath)
	}
	
	s.state.LastSelected = index
	
	s.bus.Publish(SelectionChangedEvent{
		Added:   added,
		Removed: removed,
		Total:   len(s.state.SelectedRepos),
	})
}

// SelectRange selects a range from last selected to current
func (s *Service) SelectRange(toIndex int) {
	if s.queryFn == nil || s.state.LastSelected < 0 {
		return
	}
	
	start, end := s.state.LastSelected, toIndex
	if start > end {
		start, end = end, start
	}
	
	var added []string
	for i := start; i <= end; i++ {
		repoPath := s.queryFn(i)
		if repoPath != "" && !s.state.SelectedRepos[repoPath] {
			s.state.SelectedRepos[repoPath] = true
			added = append(added, repoPath)
		}
	}
	
	if len(added) > 0 {
		s.bus.Publish(SelectionChangedEvent{
			Added: added,
			Total: len(s.state.SelectedRepos),
		})
	}
}

// SelectAll selects all visible repositories
func (s *Service) SelectAll(repoPaths []string) {
	s.state.SelectedRepos = make(map[string]bool)
	for _, path := range repoPaths {
		s.state.SelectedRepos[path] = true
	}
	
	s.bus.Publish(AllSelectedEvent{
		Paths: repoPaths,
	})
}

// DeselectAll clears all selections
func (s *Service) DeselectAll() {
	s.state.SelectedRepos = make(map[string]bool)
	s.state.LastSelected = -1
	
	s.bus.Publish(SelectionClearedEvent{})
}

// IsSelected checks if a repo is selected
func (s *Service) IsSelected(repoPath string) bool {
	return s.state.SelectedRepos[repoPath]
}

// GetSelected returns all selected paths
func (s *Service) GetSelected() []string {
	var selected []string
	for path := range s.state.SelectedRepos {
		selected = append(selected, path)
	}
	return selected
}

// GetCount returns the number of selected items
func (s *Service) GetCount() int {
	return len(s.state.SelectedRepos)
}

// HasSelection returns true if anything is selected
func (s *Service) HasSelection() bool {
	return len(s.state.SelectedRepos) > 0
}

// Clear specific repos from selection (e.g., when they're removed)
func (s *Service) RemoveFromSelection(paths []string) {
	var removed []string
	for _, path := range paths {
		if s.state.SelectedRepos[path] {
			delete(s.state.SelectedRepos, path)
			removed = append(removed, path)
		}
	}
	
	if len(removed) > 0 {
		s.bus.Publish(SelectionChangedEvent{
			Removed: removed,
			Total:   len(s.state.SelectedRepos),
		})
	}
}