package repositories

import (
	"gitagrip/internal/domain"
	"gitagrip/internal/ui/state"
)

// RepositoryStore provides access to repository data
type RepositoryStore interface {
	// Repository operations
	GetRepository(path string) (*domain.Repository, bool)
	GetAllRepositories() map[string]*domain.Repository
	GetOrderedRepositories() []string

	// Group operations
	GetGroup(name string) (*domain.Group, bool)
	GetAllGroups() map[string]*domain.Group
	GetOrderedGroups() []string
	GetGroupCreationOrder() []string

	// Selection operations
	IsRepositorySelected(path string) bool
	GetSelectedRepositories() map[string]bool
	GetSelectionCount() int

	// Operation state queries
	IsRepositoryRefreshing(path string) bool
	IsRepositoryFetching(path string) bool
	IsRepositoryPulling(path string) bool
	GetRefreshingCount() int
	GetFetchingCount() int
	GetPullingCount() int

	// UI state queries
	IsScanning() bool
	GetStatusMessage() string
	IsGroupExpanded(name string) bool

	// Search and filter state
	GetSearchQuery() string
	GetFilterQuery() string
	IsFiltered() bool
}

// StateRepositoryStore implements RepositoryStore using AppState
type StateRepositoryStore struct {
	state *state.AppState
}

// NewStateRepositoryStore creates a new repository store backed by AppState
func NewStateRepositoryStore(appState *state.AppState) *StateRepositoryStore {
	return &StateRepositoryStore{
		state: appState,
	}
}

// Repository operations
func (s *StateRepositoryStore) GetRepository(path string) (*domain.Repository, bool) {
	return s.state.GetRepository(path)
}

func (s *StateRepositoryStore) GetAllRepositories() map[string]*domain.Repository {
	return s.state.Repositories
}

func (s *StateRepositoryStore) GetOrderedRepositories() []string {
	return s.state.OrderedRepos
}

// Group operations
func (s *StateRepositoryStore) GetGroup(name string) (*domain.Group, bool) {
	group, ok := s.state.Groups[name]
	return group, ok
}

func (s *StateRepositoryStore) GetAllGroups() map[string]*domain.Group {
	return s.state.Groups
}

func (s *StateRepositoryStore) GetOrderedGroups() []string {
	return s.state.OrderedGroups
}

func (s *StateRepositoryStore) GetGroupCreationOrder() []string {
	return s.state.GroupCreationOrder
}

// Selection operations
func (s *StateRepositoryStore) IsRepositorySelected(path string) bool {
	return s.state.SelectedRepos[path]
}

func (s *StateRepositoryStore) GetSelectedRepositories() map[string]bool {
	return s.state.SelectedRepos
}

func (s *StateRepositoryStore) GetSelectionCount() int {
	return len(s.state.SelectedRepos)
}

// Operation state queries
func (s *StateRepositoryStore) IsRepositoryRefreshing(path string) bool {
	return s.state.RefreshingRepos[path]
}

func (s *StateRepositoryStore) IsRepositoryFetching(path string) bool {
	return s.state.FetchingRepos[path]
}

func (s *StateRepositoryStore) IsRepositoryPulling(path string) bool {
	return s.state.PullingRepos[path]
}

func (s *StateRepositoryStore) GetRefreshingCount() int {
	return len(s.state.RefreshingRepos)
}

func (s *StateRepositoryStore) GetFetchingCount() int {
	return len(s.state.FetchingRepos)
}

func (s *StateRepositoryStore) GetPullingCount() int {
	return len(s.state.PullingRepos)
}

// UI state queries
func (s *StateRepositoryStore) IsScanning() bool {
	return s.state.Scanning
}

func (s *StateRepositoryStore) GetStatusMessage() string {
	return s.state.StatusMessage
}

func (s *StateRepositoryStore) IsGroupExpanded(name string) bool {
	return s.state.ExpandedGroups[name]
}

// Search and filter state
func (s *StateRepositoryStore) GetSearchQuery() string {
	return s.state.SearchQuery
}

func (s *StateRepositoryStore) GetFilterQuery() string {
	return s.state.FilterQuery
}

func (s *StateRepositoryStore) IsFiltered() bool {
	return s.state.IsFiltered
}
