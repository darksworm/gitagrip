package groups

import "gitagrip/internal/domain"

// State holds groups state
type State struct {
	ExpandedGroups map[string]bool
}

// Event types
type GroupCreatedEvent struct {
	Name  string
	Repos []string
}

type GroupDeletedEvent struct {
	Name string
}

type GroupExpandedEvent struct {
	Name string
}

type GroupCollapsedEvent struct {
	Name string
}

type ReposMovedToGroupEvent struct {
	GroupName string
	Repos     []string
	OldGroup  string // Empty if from ungrouped
}

type GroupUpdatedEvent struct {
	Name  string
	Group *domain.Group
}