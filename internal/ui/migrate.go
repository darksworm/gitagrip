package ui

import (
	"gitagrip/internal/config"
	"gitagrip/internal/eventbus"
	"gitagrip/internal/logic"
)

// CreateModel creates the appropriate model based on a feature flag
// This allows us to test the new model while keeping the old one
func CreateModel(
	cfg *config.Config,
	store logic.RepositoryStore,
	groupStore logic.GroupStore,
	eventBus eventbus.EventBus,
	eventChan <-chan interface{},
	useNewModel bool,
) interface{} {
	if useNewModel {
		return NewModel(cfg, store, groupStore, eventBus, eventChan)
	}
	
	// Use the old model by default for now
	return NewLegacyModel(cfg, store, groupStore, eventBus, eventChan)
}