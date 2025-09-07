package coordinator

import (
	"gitagrip/internal/domain"
	"gitagrip/internal/logic"
	"gitagrip/internal/ui/services/events"
	"gitagrip/internal/ui/services/groups"
	"gitagrip/internal/ui/services/navigation"
	"gitagrip/internal/ui/services/query"
	"gitagrip/internal/ui/services/search"
	"gitagrip/internal/ui/services/selection"
	"gitagrip/internal/ui/services/sorting"
)

// Coordinator manages all UI services and their interactions
type Coordinator struct {
	// Services
	Navigation *navigation.Service
	Query      *query.Service
	Selection  *selection.Service
	Search     *search.Service
	Groups     *groups.Service
	Sorting    *sorting.Service
	
	// Dependencies
	bus        events.EventBus
	repoStore  logic.RepositoryStore
	groupStore logic.GroupStore
}

// NewCoordinator creates a new coordinator with all services
func NewCoordinator(bus events.EventBus, repoStore logic.RepositoryStore, groupStore logic.GroupStore) *Coordinator {
	c := &Coordinator{
		Navigation: navigation.NewService(bus),
		Query:      query.NewService(repoStore, groupStore),
		Selection:  selection.NewService(bus),
		Search:     search.NewService(bus),
		Groups:     groups.NewService(bus, groupStore),
		Sorting:    sorting.NewService(bus),
		bus:        bus,
		repoStore:  repoStore,
		groupStore: groupStore,
	}
	
	// Wire up service dependencies
	c.wireServices()
	
	// Subscribe to events
	c.subscribeToEvents()
	
	return c
}

// wireServices connects services with their dependencies
func (c *Coordinator) wireServices() {
	// Navigation needs to query max index
	c.Navigation.SetQueryFunction(func() int {
		return c.Query.GetMaxIndex()
	})
	
	// Selection needs to query repo paths
	c.Selection.SetQueryFunction(func(index int) string {
		return c.Query.GetRepositoryPathAtIndex(index)
	})
	
	// Search needs to find matches and navigate
	c.Search.SetMatcherFunction(func(searchQuery string) []search.MatchResult {
		matches := c.Query.GetRepositoriesMatching(searchQuery)
		
		// Convert to search results
		var results []search.MatchResult
		for _, info := range matches {
			result := search.MatchResult{
				Index:      c.Query.GetIndexForRepository(info.Path),
				Path:       info.Path,
				Repository: info.Repository,
				IsGroup:    info.Type == query.IndexTypeGroup,
			}
			if info.Repository != nil {
				result.Name = info.Repository.Name
			} else if info.Type == query.IndexTypeGroup {
				result.Name = info.GroupName
			}
			results = append(results, result)
		}
		return results
	})
	
	c.Search.SetNavigateFunction(func(index int) {
		c.Navigation.MoveToIndex(index)
	})
	
	// Sorting needs to get repositories
	c.Sorting.SetRepositoryFunction(func(path string) *domain.Repository {
		return c.repoStore.GetRepository(path)
	})
	
	// Groups needs save function (will be set by Model)
}

// subscribeToEvents sets up event handlers
func (c *Coordinator) subscribeToEvents() {
	// When groups change, update query service
	c.bus.Subscribe("groups.GroupCreatedEvent", func(e interface{}) {
		c.UpdateOrderedLists()
	})
	
	c.bus.Subscribe("groups.GroupDeletedEvent", func(e interface{}) {
		c.UpdateOrderedLists()
	})
	
	c.bus.Subscribe("groups.GroupExpandedEvent", func(e interface{}) {
		expanded := c.Groups.GetExpandedGroups()
		c.Query.SetExpandedGroups(expanded)
	})
	
	c.bus.Subscribe("groups.GroupCollapsedEvent", func(e interface{}) {
		expanded := c.Groups.GetExpandedGroups()
		c.Query.SetExpandedGroups(expanded)
	})
	
	// When sort mode changes, re-sort
	c.bus.Subscribe("sorting.SortModeChangedEvent", func(e interface{}) {
		c.UpdateOrderedLists()
	})
	
	// When navigation changes and there's an active search, check if we're still on a match
	c.bus.Subscribe("navigation.CursorMovedEvent", func(e interface{}) {
		// Could implement search match tracking here if needed
	})
}

// UpdateStores updates the underlying data stores
func (c *Coordinator) UpdateStores(repoStore logic.RepositoryStore, groupStore logic.GroupStore) {
	c.repoStore = repoStore
	c.groupStore = groupStore
	
	// Update services that need the stores
	c.Query = query.NewService(repoStore, groupStore)
	c.UpdateOrderedLists()
}

// UpdateOrderedLists updates the ordered lists in query service
func (c *Coordinator) UpdateOrderedLists() {
	// Get all repos and sort them
	repos := make([]string, 0, len(c.repoStore.GetAllRepositories()))
	for path := range c.repoStore.GetAllRepositories() {
		repos = append(repos, path)
	}
	c.Sorting.SortRepositories(repos)
	c.Query.SetOrderedRepos(repos)
	
	// Get all groups and sort them
	groups := make([]string, 0, len(c.groupStore.GetAllGroups()))
	for _, group := range c.groupStore.GetAllGroups() {
		groups = append(groups, group.Name)
	}
	c.Sorting.SortGroups(groups)
	c.Query.SetOrderedGroups(groups)
	
	// Update expanded groups
	c.Query.SetExpandedGroups(c.Groups.GetExpandedGroups())
}

// GetCurrentIndex returns the current navigation index
func (c *Coordinator) GetCurrentIndex() int {
	return c.Navigation.GetCursor()
}

// GetCurrentRepository returns the repository at current index
func (c *Coordinator) GetCurrentRepository() *domain.Repository {
	index := c.Navigation.GetCursor()
	return c.Query.GetRepositoryAtIndex(index)
}

// GetCurrentRepositoryPath returns the repository path at current index
func (c *Coordinator) GetCurrentRepositoryPath() string {
	index := c.Navigation.GetCursor()
	return c.Query.GetRepositoryPathAtIndex(index)
}

// IsOnGroup checks if current index is on a group header
func (c *Coordinator) IsOnGroup() bool {
	index := c.Navigation.GetCursor()
	info := c.Query.GetIndexInfo(index)
	return info != nil && info.Type == query.IndexTypeGroup
}

// GetCurrentGroupName returns the group name at current index
func (c *Coordinator) GetCurrentGroupName() string {
	index := c.Navigation.GetCursor()
	info := c.Query.GetIndexInfo(index)
	if info != nil && info.Type == query.IndexTypeGroup {
		return info.GroupName
	}
	return ""
}

// SetViewportHeight updates viewport height across services
func (c *Coordinator) SetViewportHeight(height int) {
	c.Navigation.SetViewportHeight(height)
}