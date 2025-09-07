package state

import (
	"gitagrip/internal/domain"
)

// AppState contains all the application state
type AppState struct {
	// Repository data
	Repositories map[string]*domain.Repository // path -> repo
	OrderedRepos []string                      // ordered repo paths for display
	PendingRepos map[string]*domain.Repository // repos discovered during scanning

	// Group data
	Groups             map[string]*domain.Group // name -> group
	OrderedGroups      []string                 // ordered group names
	GroupCreationOrder []string                 // tracks order of group creation
	ExpandedGroups     map[string]bool          // which groups are expanded

	// Selection state
	SelectedIndex int             // currently selected item
	SelectedRepos map[string]bool // selected repository paths

	// Operation states
	RefreshingRepos map[string]bool // repositories currently being refreshed
	FetchingRepos   map[string]bool // repositories currently being fetched
	PullingRepos    map[string]bool // repositories currently being pulled

	// UI state
	ViewportOffset   int  // offset for scrolling
	ViewportHeight   int  // available height for repo list
	Scanning         bool // whether scanning is in progress
	ShowHelp         bool
	HelpScrollOffset int // scroll offset for help popup
	ShowLog          bool
	LogContent       string
	ShowInfo         bool
	InfoContent      string
	StatusMessage    string // status bar message
	LoadingState     string // current loading state description
	LoadingCount     int    // count for loading progress

	// Search and filter state
	SearchQuery     string // current search query
	SearchMatches   []int  // indices of matching items
	SearchIndex     int    // current match index
	SortOptionIndex int    // current selected sort option in sort mode
	FilterQuery     string // current filter query
	IsFiltered      bool   // whether filter is active

	// Cached data
	UngroupedRepos []string // cached ungrouped repos
}

// NewAppState creates a new application state
func NewAppState() *AppState {
	return &AppState{
		Repositories:       make(map[string]*domain.Repository),
		OrderedRepos:       make([]string, 0),
		PendingRepos:       make(map[string]*domain.Repository),
		Groups:             make(map[string]*domain.Group),
		OrderedGroups:      make([]string, 0),
		GroupCreationOrder: make([]string, 0),
		ExpandedGroups:     make(map[string]bool),
		SelectedRepos:      make(map[string]bool),
		RefreshingRepos:    make(map[string]bool),
		FetchingRepos:      make(map[string]bool),
		PullingRepos:       make(map[string]bool),
		UngroupedRepos:     make([]string, 0),
		ViewportHeight:     20, // Default
	}
}

// Repository operations

// AddRepository adds or updates a repository
func (s *AppState) AddRepository(repo *domain.Repository) {
	if repo != nil {
		s.Repositories[repo.Path] = repo
	}
}

// GetRepository retrieves a repository by path
func (s *AppState) GetRepository(path string) (*domain.Repository, bool) {
	repo, ok := s.Repositories[path]
	return repo, ok
}

// RemoveRepository removes a repository
func (s *AppState) RemoveRepository(path string) {
	delete(s.Repositories, path)
	delete(s.SelectedRepos, path)
	delete(s.RefreshingRepos, path)
	delete(s.FetchingRepos, path)
	delete(s.PullingRepos, path)
}

// Group operations

// AddGroup adds a new group
func (s *AppState) AddGroup(name string, repos []string) {
	s.Groups[name] = &domain.Group{
		Name:  name,
		Repos: repos,
	}
	// Hidden group should be collapsed by default
	if name == "_Hidden" {
		s.ExpandedGroups[name] = false
	} else {
		s.ExpandedGroups[name] = true
	}
	// Add to beginning of creation order
	s.GroupCreationOrder = append([]string{name}, s.GroupCreationOrder...)
}

// RemoveGroup removes a group
func (s *AppState) RemoveGroup(name string) {
	delete(s.Groups, name)
	delete(s.ExpandedGroups, name)

	// Remove from creation order
	newOrder := []string{}
	for _, n := range s.GroupCreationOrder {
		if n != name {
			newOrder = append(newOrder, n)
		}
	}
	s.GroupCreationOrder = newOrder
}

// MoveRepoToGroup moves a repository from one group to another
func (s *AppState) MoveRepoToGroup(repoPath, fromGroup, toGroup string) {
	// Remove from old group
	if fromGroup != "" {
		if group, exists := s.Groups[fromGroup]; exists {
			newRepos := make([]string, 0, len(group.Repos))
			for _, path := range group.Repos {
				if path != repoPath {
					newRepos = append(newRepos, path)
				}
			}
			group.Repos = newRepos
		}
	}

	// Add to new group
	if toGroup != "" {
		if group, exists := s.Groups[toGroup]; exists {
			// Check if already in group
			found := false
			for _, path := range group.Repos {
				if path == repoPath {
					found = true
					break
				}
			}
			if !found {
				group.Repos = append(group.Repos, repoPath)
			}
		}
	}
}

// Selection operations

// ToggleRepoSelection toggles the selection state of a repository
func (s *AppState) ToggleRepoSelection(repoPath string) {
	if s.SelectedRepos[repoPath] {
		delete(s.SelectedRepos, repoPath)
	} else {
		s.SelectedRepos[repoPath] = true
	}
}

// ClearSelection clears all selected repositories
func (s *AppState) ClearSelection() {
	s.SelectedRepos = make(map[string]bool)
}

// SelectAll selects all repositories
func (s *AppState) SelectAll() {
	for path := range s.Repositories {
		s.SelectedRepos[path] = true
	}
}

// Operation state management

// SetRefreshing marks repositories as refreshing
func (s *AppState) SetRefreshing(repoPaths []string, refreshing bool) {
	for _, path := range repoPaths {
		if refreshing {
			s.RefreshingRepos[path] = true
		} else {
			delete(s.RefreshingRepos, path)
		}
	}
}

// SetFetching marks repositories as fetching
func (s *AppState) SetFetching(repoPaths []string, fetching bool) {
	for _, path := range repoPaths {
		if fetching {
			s.FetchingRepos[path] = true
		} else {
			delete(s.FetchingRepos, path)
		}
	}
}

// SetPulling marks repositories as pulling
func (s *AppState) SetPulling(repoPaths []string, pulling bool) {
	for _, path := range repoPaths {
		if pulling {
			s.PullingRepos[path] = true
		} else {
			delete(s.PullingRepos, path)
		}
	}
}

// ClearOperationState clears the operation state for a repository
func (s *AppState) ClearOperationState(repoPath string) {
	delete(s.RefreshingRepos, repoPath)
	delete(s.FetchingRepos, repoPath)
	delete(s.PullingRepos, repoPath)
}

// GetGroupsMap returns a copy of groups as a map
func (s *AppState) GetGroupsMap() map[string][]string {
	groups := make(map[string][]string)
	for name, group := range s.Groups {
		groups[name] = append([]string(nil), group.Repos...) // Copy slice
	}
	return groups
}
