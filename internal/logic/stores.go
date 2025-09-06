package logic

import (
	"sync"
	"gitagrip/internal/domain"
)

// MemoryRepositoryStore is an in-memory implementation of RepositoryStore
type MemoryRepositoryStore struct {
	mu    sync.RWMutex
	repos map[string]*domain.Repository
}

// NewMemoryRepositoryStore creates a new memory-based repository store
func NewMemoryRepositoryStore() *MemoryRepositoryStore {
	return &MemoryRepositoryStore{
		repos: make(map[string]*domain.Repository),
	}
}

func (s *MemoryRepositoryStore) GetRepository(path string) *domain.Repository {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.repos[path]
}

func (s *MemoryRepositoryStore) GetAllRepositories() map[string]*domain.Repository {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// Return a copy to prevent external modification
	result := make(map[string]*domain.Repository)
	for k, v := range s.repos {
		result[k] = v
	}
	return result
}

func (s *MemoryRepositoryStore) AddRepository(repo *domain.Repository) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.repos[repo.Path] = repo
}

func (s *MemoryRepositoryStore) UpdateRepository(repo *domain.Repository) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.repos[repo.Path] = repo
}

func (s *MemoryRepositoryStore) RemoveRepository(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.repos, path)
}

// MemoryGroupStore is an in-memory implementation of GroupStore
type MemoryGroupStore struct {
	mu     sync.RWMutex
	groups map[string]*domain.Group
}

// NewMemoryGroupStore creates a new memory-based group store
func NewMemoryGroupStore() *MemoryGroupStore {
	return &MemoryGroupStore{
		groups: make(map[string]*domain.Group),
	}
}

func (s *MemoryGroupStore) GetGroup(name string) *domain.Group {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.groups[name]
}

func (s *MemoryGroupStore) GetAllGroups() map[string]*domain.Group {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// Return a copy to prevent external modification
	result := make(map[string]*domain.Group)
	for k, v := range s.groups {
		result[k] = v
	}
	return result
}

func (s *MemoryGroupStore) AddGroup(group *domain.Group) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.groups[group.Name] = group
}

func (s *MemoryGroupStore) UpdateGroup(group *domain.Group) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.groups[group.Name] = group
}

func (s *MemoryGroupStore) DeleteGroup(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.groups, name)
}