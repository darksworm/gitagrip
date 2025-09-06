package adapters

import (
	"gitagrip/internal/domain"
	"gitagrip/internal/logic"
	"gitagrip/internal/ui/state"
)

// StateToStoreAdapter adapts the legacy AppState to the new store interfaces
type StateToStoreAdapter struct {
	appState *state.AppState
}

// NewStateToStoreAdapter creates a new adapter
func NewStateToStoreAdapter(appState *state.AppState) *StateToStoreAdapter {
	return &StateToStoreAdapter{appState: appState}
}

// AsRepositoryStore returns the adapter as a RepositoryStore
func (a *StateToStoreAdapter) AsRepositoryStore() logic.RepositoryStore {
	return &repositoryStoreAdapter{appState: a.appState}
}

// AsGroupStore returns the adapter as a GroupStore
func (a *StateToStoreAdapter) AsGroupStore() logic.GroupStore {
	return &groupStoreAdapter{appState: a.appState}
}

// repositoryStoreAdapter implements logic.RepositoryStore
type repositoryStoreAdapter struct {
	appState *state.AppState
}

func (a *repositoryStoreAdapter) GetRepository(path string) *domain.Repository {
	repo, _ := a.appState.GetRepository(path)
	return repo
}

func (a *repositoryStoreAdapter) GetAllRepositories() map[string]*domain.Repository {
	return a.appState.GetAllRepositories()
}

func (a *repositoryStoreAdapter) AddRepository(repo *domain.Repository) {
	a.appState.AddRepository(repo)
}

func (a *repositoryStoreAdapter) UpdateRepository(repo *domain.Repository) {
	a.appState.UpdateRepository(repo)
}

func (a *repositoryStoreAdapter) RemoveRepository(path string) {
	// Legacy state doesn't have remove, so we'll skip this
}

// groupStoreAdapter implements logic.GroupStore
type groupStoreAdapter struct {
	appState *state.AppState
}

func (a *groupStoreAdapter) GetGroup(name string) *domain.Group {
	group, _ := a.appState.GetGroup(name)
	return group
}

func (a *groupStoreAdapter) GetAllGroups() map[string]*domain.Group {
	return a.appState.GetAllGroups()
}

func (a *groupStoreAdapter) AddGroup(group *domain.Group) {
	a.appState.AddGroup(group.Name, group.Repos)
}

func (a *groupStoreAdapter) UpdateGroup(group *domain.Group) {
	// Update by removing and re-adding
	a.appState.RemoveGroup(group.Name)
	a.appState.AddGroup(group.Name, group.Repos)
}

func (a *groupStoreAdapter) DeleteGroup(name string) {
	a.appState.RemoveGroup(name)
}