package groups

import (
	"gitagrip/internal/domain"
	"gitagrip/internal/logic"
	"gitagrip/internal/ui/services/events"
)

// Service handles group management
type Service struct {
	state      *State
	bus        events.EventBus
	groupStore logic.GroupStore
	saveFn     func() // Function to save config
}

// NewService creates a new groups service
func NewService(bus events.EventBus, groupStore logic.GroupStore) *Service {
	return &Service{
		state: &State{
			ExpandedGroups: make(map[string]bool),
		},
		bus:        bus,
		groupStore: groupStore,
	}
}

// SetSaveFunction sets the function to save config
func (s *Service) SetSaveFunction(fn func()) {
	s.saveFn = fn
}

// CreateGroup creates a new group
func (s *Service) CreateGroup(name string, repos []string) error {
	// Create the group
	group := &domain.Group{
		Name:  name,
		Repos: repos,
	}
	
	s.groupStore.AddGroup(group)
	
	// Auto-expand new groups
	s.state.ExpandedGroups[name] = true
	
	// Save config
	if s.saveFn != nil {
		s.saveFn()
	}
	
	s.bus.Publish(GroupCreatedEvent{
		Name:  name,
		Repos: repos,
	})
	
	return nil
}

// DeleteGroup removes a group
func (s *Service) DeleteGroup(name string) error {
	s.groupStore.DeleteGroup(name)
	delete(s.state.ExpandedGroups, name)
	
	// Save config
	if s.saveFn != nil {
		s.saveFn()
	}
	
	s.bus.Publish(GroupDeletedEvent{
		Name: name,
	})
	
	return nil
}

// ToggleExpanded toggles group expansion state
func (s *Service) ToggleExpanded(name string) {
	if s.state.ExpandedGroups[name] {
		s.state.ExpandedGroups[name] = false
		s.bus.Publish(GroupCollapsedEvent{Name: name})
	} else {
		s.state.ExpandedGroups[name] = true
		s.bus.Publish(GroupExpandedEvent{Name: name})
	}
}

// IsExpanded checks if a group is expanded
func (s *Service) IsExpanded(name string) bool {
	return s.state.ExpandedGroups[name]
}

// GetExpandedGroups returns map of expanded groups
func (s *Service) GetExpandedGroups() map[string]bool {
	// Return a copy to prevent external modification
	result := make(map[string]bool)
	for k, v := range s.state.ExpandedGroups {
		result[k] = v
	}
	return result
}

// MoveReposToGroup moves repositories to a group
func (s *Service) MoveReposToGroup(repos []string, targetGroup string) error {
	// First, remove repos from their current groups
	oldGroups := make(map[string]string) // repo -> old group
	
	for _, group := range s.groupStore.GetAllGroups() {
		newRepos := []string{}
		for _, repo := range group.Repos {
			found := false
			for _, movingRepo := range repos {
				if repo == movingRepo {
					oldGroups[repo] = group.Name
					found = true
					break
				}
			}
			if !found {
				newRepos = append(newRepos, repo)
			}
		}
		
		// Update group if repos were removed
		if len(newRepos) != len(group.Repos) {
			group.Repos = newRepos
			s.groupStore.UpdateGroup(group)
		}
	}
	
	// Add repos to target group
	targetGroupObj := s.groupStore.GetGroup(targetGroup)
	if targetGroupObj == nil {
		// Create new group if it doesn't exist
		targetGroupObj = &domain.Group{
			Name:  targetGroup,
			Repos: repos,
		}
		s.groupStore.AddGroup(targetGroupObj)
		s.state.ExpandedGroups[targetGroup] = true
	} else {
		// Add to existing group
		targetGroupObj.Repos = append(targetGroupObj.Repos, repos...)
		s.groupStore.UpdateGroup(targetGroupObj)
	}
	
	// Save config
	if s.saveFn != nil {
		s.saveFn()
	}
	
	// Publish events
	for _, repo := range repos {
		s.bus.Publish(ReposMovedToGroupEvent{
			GroupName: targetGroup,
			Repos:     []string{repo},
			OldGroup:  oldGroups[repo],
		})
	}
	
	return nil
}

// GetGroupForRepo finds which group contains a repository
func (s *Service) GetGroupForRepo(repoPath string) string {
	for _, group := range s.groupStore.GetAllGroups() {
		for _, repo := range group.Repos {
			if repo == repoPath {
				return group.Name
			}
		}
	}
	return "" // Ungrouped
}

// RemoveRepoFromGroup removes a single repo from its group
func (s *Service) RemoveRepoFromGroup(repoPath string) {
	for _, group := range s.groupStore.GetAllGroups() {
		newRepos := []string{}
		found := false
		
		for _, repo := range group.Repos {
			if repo != repoPath {
				newRepos = append(newRepos, repo)
			} else {
				found = true
			}
		}
		
		if found {
			if len(newRepos) == 0 {
				// Delete empty group
				s.DeleteGroup(group.Name)
			} else {
				group.Repos = newRepos
				s.groupStore.UpdateGroup(group)
				s.bus.Publish(GroupUpdatedEvent{
					Name:  group.Name,
					Group: group,
				})
			}
			
			// Save config
			if s.saveFn != nil {
				s.saveFn()
			}
			break
		}
	}
}

// ExpandAll expands all groups
func (s *Service) ExpandAll() {
	for _, group := range s.groupStore.GetAllGroups() {
		if !s.state.ExpandedGroups[group.Name] {
			s.state.ExpandedGroups[group.Name] = true
			s.bus.Publish(GroupExpandedEvent{Name: group.Name})
		}
	}
}

// CollapseAll collapses all groups
func (s *Service) CollapseAll() {
	for name := range s.state.ExpandedGroups {
		if s.state.ExpandedGroups[name] {
			s.state.ExpandedGroups[name] = false
			s.bus.Publish(GroupCollapsedEvent{Name: name})
		}
	}
}