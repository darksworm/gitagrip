package groups

import (
	"fmt"
	"sync"

	"gitagrip/internal/domain"
	"gitagrip/internal/eventbus"
)

// GroupManager manages repository grouping
type GroupManager interface {
	CreateGroup(name string) error
	RemoveGroup(name string) error
	AddRepoToGroup(repoPath string, groupName string) error
	RemoveRepoFromGroup(repoPath string, groupName string) error
	GetGroups() map[string]*domain.Group
	GetRepoGroup(repoPath string) string
}

// groupManager is the concrete implementation
type groupManager struct {
	bus         eventbus.EventBus
	mu          sync.RWMutex
	groups      map[string]*domain.Group // group name -> group
	repoToGroup map[string]string        // repo path -> group name
}

// NewGroupManager creates a new group manager
func NewGroupManager(bus eventbus.EventBus, initialGroups map[string][]string) GroupManager {
	gm := &groupManager{
		bus:         bus,
		groups:      make(map[string]*domain.Group),
		repoToGroup: make(map[string]string),
	}

	// Initialize with groups from config
	for name, repoPaths := range initialGroups {
		gm.groups[name] = &domain.Group{
			Name:  name,
			Repos: repoPaths,
		}

		// Update reverse mapping
		for _, repoPath := range repoPaths {
			gm.repoToGroup[repoPath] = name
		}
	}

	// Subscribe to group-related events
	bus.Subscribe(eventbus.EventGroupAdded, func(e eventbus.DomainEvent) {
		if event, ok := e.(eventbus.GroupAddedEvent); ok {
			gm.CreateGroup(event.Name)
		}
	})

	bus.Subscribe(eventbus.EventGroupRemoved, func(e eventbus.DomainEvent) {
		if event, ok := e.(eventbus.GroupRemovedEvent); ok {
			gm.RemoveGroup(event.Name)
		}
	})

	bus.Subscribe(eventbus.EventRepoMoved, func(e eventbus.DomainEvent) {
		if event, ok := e.(eventbus.RepoMovedEvent); ok {
			if event.FromGroup != "" {
				gm.RemoveRepoFromGroup(event.RepoPath, event.FromGroup)
			}
			if event.ToGroup != "" {
				gm.AddRepoToGroup(event.RepoPath, event.ToGroup)
			}
		}
	})

	return gm
}

// CreateGroup creates a new group
func (gm *groupManager) CreateGroup(name string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	if _, exists := gm.groups[name]; exists {
		return fmt.Errorf("group %s already exists", name)
	}

	gm.groups[name] = &domain.Group{
		Name:  name,
		Repos: []string{},
	}

	// Publish event if we're not already handling one
	// (to avoid circular events)
	if gm.bus != nil {
		go gm.bus.Publish(eventbus.GroupAddedEvent{Name: name})
	}

	return nil
}

// RemoveGroup removes a group
func (gm *groupManager) RemoveGroup(name string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	group, exists := gm.groups[name]
	if !exists {
		return fmt.Errorf("group %s does not exist", name)
	}

	// Remove all repos from the group
	for _, repoPath := range group.Repos {
		delete(gm.repoToGroup, repoPath)
	}

	// Remove the group
	delete(gm.groups, name)

	// Publish event
	if gm.bus != nil {
		go gm.bus.Publish(eventbus.GroupRemovedEvent{Name: name})
	}

	return nil
}

// AddRepoToGroup adds a repository to a group
func (gm *groupManager) AddRepoToGroup(repoPath string, groupName string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	group, exists := gm.groups[groupName]
	if !exists {
		return fmt.Errorf("group %s does not exist", groupName)
	}

	// Check if repo is already in this group
	currentGroup, hasGroup := gm.repoToGroup[repoPath]
	if hasGroup && currentGroup == groupName {
		return nil // Already in the group
	}

	// Remove from current group if any
	if hasGroup {
		gm.removeRepoFromGroupLocked(repoPath, currentGroup)
	}

	// Add to new group
	group.Repos = append(group.Repos, repoPath)
	gm.repoToGroup[repoPath] = groupName

	// Publish event
	if gm.bus != nil {
		go gm.bus.Publish(eventbus.RepoMovedEvent{
			RepoPath:  repoPath,
			FromGroup: currentGroup,
			ToGroup:   groupName,
		})
	}

	return nil
}

// RemoveRepoFromGroup removes a repository from a group
func (gm *groupManager) RemoveRepoFromGroup(repoPath string, groupName string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	return gm.removeRepoFromGroupLocked(repoPath, groupName)
}

// removeRepoFromGroupLocked removes a repo from a group (must be called with lock held)
func (gm *groupManager) removeRepoFromGroupLocked(repoPath string, groupName string) error {
	group, exists := gm.groups[groupName]
	if !exists {
		return fmt.Errorf("group %s does not exist", groupName)
	}

	// Find and remove the repo
	for i, path := range group.Repos {
		if path == repoPath {
			group.Repos = append(group.Repos[:i], group.Repos[i+1:]...)
			delete(gm.repoToGroup, repoPath)

			// Publish event
			if gm.bus != nil {
				go gm.bus.Publish(eventbus.RepoMovedEvent{
					RepoPath:  repoPath,
					FromGroup: groupName,
					ToGroup:   "",
				})
			}

			return nil
		}
	}

	return fmt.Errorf("repository %s not found in group %s", repoPath, groupName)
}

// GetGroups returns all groups
func (gm *groupManager) GetGroups() map[string]*domain.Group {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	// Return a copy to prevent concurrent modification
	result := make(map[string]*domain.Group, len(gm.groups))
	for name, group := range gm.groups {
		// Deep copy the group
		reposCopy := make([]string, len(group.Repos))
		copy(reposCopy, group.Repos)

		result[name] = &domain.Group{
			Name:  group.Name,
			Repos: reposCopy,
		}
	}

	return result
}

// GetRepoGroup returns the group a repository belongs to
func (gm *groupManager) GetRepoGroup(repoPath string) string {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	return gm.repoToGroup[repoPath]
}
