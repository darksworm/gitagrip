package sorting

import (
	"sort"
	"strings"
	
	"gitagrip/internal/domain"
	"gitagrip/internal/logic"
	"gitagrip/internal/ui/services/events"
)

// Service handles sorting logic
type Service struct {
	state  *State
	bus    events.EventBus
	repoFn func(string) *domain.Repository // Function to get repository
}

// NewService creates a new sorting service
func NewService(bus events.EventBus) *Service {
	return &Service{
		state: &State{
			CurrentMode: logic.SortByName, // Default
		},
		bus: bus,
	}
}

// SetRepositoryFunction sets the function to get repositories
func (s *Service) SetRepositoryFunction(fn func(string) *domain.Repository) {
	s.repoFn = fn
}

// GetCurrentMode returns the current sort mode
func (s *Service) GetCurrentMode() logic.SortMode {
	return s.state.CurrentMode
}

// SetMode sets the sort mode
func (s *Service) SetMode(mode logic.SortMode) {
	if mode == s.state.CurrentMode {
		return
	}
	
	oldMode := s.state.CurrentMode
	s.state.CurrentMode = mode
	
	s.bus.Publish(SortModeChangedEvent{
		OldMode: oldMode,
		NewMode: mode,
	})
}

// NextMode cycles to the next sort mode
func (s *Service) NextMode() {
	modes := []logic.SortMode{
		logic.SortByName,
		logic.SortByStatus,
		logic.SortByBranch,
		logic.SortByPath,
	}
	
	currentIndex := 0
	for i, mode := range modes {
		if mode == s.state.CurrentMode {
			currentIndex = i
			break
		}
	}
	
	nextIndex := (currentIndex + 1) % len(modes)
	s.SetMode(modes[nextIndex])
}

// SortRepositories sorts a list of repository paths
func (s *Service) SortRepositories(paths []string) {
	if s.repoFn == nil {
		return
	}
	
	switch s.state.CurrentMode {
	case logic.SortByName:
		sort.Slice(paths, func(i, j int) bool {
			repoI := s.repoFn(paths[i])
			repoJ := s.repoFn(paths[j])
			if repoI == nil || repoJ == nil {
				return repoI == nil
			}
			return strings.ToLower(repoI.Name) < strings.ToLower(repoJ.Name)
		})
		
	case logic.SortByStatus:
		sort.Slice(paths, func(i, j int) bool {
			repoI := s.repoFn(paths[i])
			repoJ := s.repoFn(paths[j])
			if repoI == nil || repoJ == nil {
				return repoI == nil
			}
			return getStatusPriority(repoI.Status) > getStatusPriority(repoJ.Status)
		})
		
	case logic.SortByBranch:
		sort.Slice(paths, func(i, j int) bool {
			repoI := s.repoFn(paths[i])
			repoJ := s.repoFn(paths[j])
			if repoI == nil || repoJ == nil {
				return repoI == nil
			}
			return strings.ToLower(repoI.Status.Branch) < strings.ToLower(repoJ.Status.Branch)
		})
		
	case logic.SortByPath:
		sort.Strings(paths)
	}
}

// SortGroups sorts group names alphabetically
func (s *Service) SortGroups(groups []string) {
	sort.Slice(groups, func(i, j int) bool {
		return strings.ToLower(groups[i]) < strings.ToLower(groups[j])
	})
}

// GetModeString returns a string representation of the current mode
func (s *Service) GetModeString() string {
	switch s.state.CurrentMode {
	case logic.SortByName:
		return "name"
	case logic.SortByStatus:
		return "status"
	case logic.SortByBranch:
		return "branch"
	case logic.SortByPath:
		return "path"
	default:
		return "unknown"
	}
}

// Helper function for status priority
func getStatusPriority(status domain.RepoStatus) int {
	if status.Error != "" {
		return 4
	}
	if status.IsDirty || status.HasUntracked {
		return 3
	}
	if status.AheadCount > 0 || status.BehindCount > 0 {
		return 2
	}
	return 1
}