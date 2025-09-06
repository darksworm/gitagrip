package logic

import "gitagrip/internal/domain"

// RepositoryStore provides access to repository data
type RepositoryStore interface {
	GetRepository(path string) *domain.Repository
	GetAllRepositories() map[string]*domain.Repository
	AddRepository(repo *domain.Repository)
	UpdateRepository(repo *domain.Repository)
	RemoveRepository(path string)
}

// GroupStore provides access to group data
type GroupStore interface {
	GetGroup(name string) *domain.Group
	GetAllGroups() map[string]*domain.Group
	AddGroup(group *domain.Group)
	UpdateGroup(group *domain.Group)
	DeleteGroup(name string)
}

// Sort modes
type SortMode int

const (
	SortByName SortMode = iota
	SortByStatus
	SortByBranch
	SortByPath
)

// Event types
type RescanRequestedEvent struct{}

type RefreshRequestedEvent struct {
	Path string
}

type RepositoryDiscoveredEvent struct {
	Repository *domain.Repository
}

type RepositoryUpdatedEvent struct {
	Repository *domain.Repository
}

type ScanStartedEvent struct{}

type ScanCompletedEvent struct {
	Count int
}