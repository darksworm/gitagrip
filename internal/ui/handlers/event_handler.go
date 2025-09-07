package handlers

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"gitagrip/internal/eventbus"
	"gitagrip/internal/ui/logic"
	"gitagrip/internal/ui/state"
)

// TickMsg is a tick message for animations
type TickMsg time.Time

// EventHandler handles domain events and updates state
type EventHandler struct {
	state         *state.AppState
	searchFilter  *logic.SearchFilter
	updateOrderedLists func()
}

// NewEventHandler creates a new event handler
func NewEventHandler(appState *state.AppState, updateOrderedLists func()) *EventHandler {
	return &EventHandler{
		state:         appState,
		searchFilter:  logic.NewSearchFilter(appState.Repositories),
		updateOrderedLists: updateOrderedLists,
	}
}

// HandleEvent processes domain events and returns any necessary commands
func (h *EventHandler) HandleEvent(event eventbus.DomainEvent) tea.Cmd {
	switch e := event.(type) {
	case eventbus.RepoDiscoveredEvent:
		// Add or update repository
		h.state.AddRepository(&e.Repo)
		h.updateOrderedLists()
		// Update searchFilter with new repositories
		h.searchFilter = logic.NewSearchFilter(h.state.Repositories)

	case eventbus.StatusUpdatedEvent:
		// Update repository status
		if repo, ok := h.state.Repositories[e.RepoPath]; ok {
			repo.Status = e.Status
		}
		// Clear operation states
		h.state.ClearOperationState(e.RepoPath)

	case eventbus.ErrorEvent:
		h.state.StatusMessage = fmt.Sprintf("Error: %s", e.Message)
		// If this is a refresh error for a specific repo, we might need to clear its refreshing state
		// This would require extending the ErrorEvent to include optional repo path

	case eventbus.GroupAddedEvent:
		if _, exists := h.state.Groups[e.Name]; !exists {
			h.state.AddGroup(e.Name, []string{})
			h.updateOrderedLists()
		}

	case eventbus.GroupRemovedEvent:
		if _, exists := h.state.Groups[e.Name]; exists {
			h.state.RemoveGroup(e.Name)
			h.updateOrderedLists()
		}

	case eventbus.RepoMovedEvent:
		h.state.MoveRepoToGroup(e.RepoPath, e.FromGroup, e.ToGroup)
		h.updateOrderedLists()

	case eventbus.ScanStartedEvent:
		h.state.Scanning = true
		h.state.StatusMessage = "Scanning for repositories..."
		// Return a tick command to start the spinner animation
		return tea.Tick(time.Millisecond*80, func(t time.Time) tea.Msg {
			// Return tick event to trigger animation update
			return TickMsg(t)
		})

	case eventbus.ScanCompletedEvent:
		h.state.Scanning = false
		h.state.StatusMessage = fmt.Sprintf("Scan complete. Found %d repositories.", e.ReposFound)

	case eventbus.FetchCompletedEvent:
		// Clear fetching state for this repo
		h.state.SetFetching([]string{e.RepoPath}, false)
		
		// Update status message
		if e.Success {
			h.state.StatusMessage = fmt.Sprintf("Fetch completed for %s", e.RepoPath)
		} else {
			h.state.StatusMessage = fmt.Sprintf("Fetch failed for %s: %v", e.RepoPath, e.Error)
		}

	case eventbus.PullCompletedEvent:
		// Clear pulling state for this repo
		h.state.SetPulling([]string{e.RepoPath}, false)
		
		// Update status message
		if e.Success {
			h.state.StatusMessage = fmt.Sprintf("Pull completed for %s", e.RepoPath)
		} else {
			h.state.StatusMessage = fmt.Sprintf("Pull failed for %s: %v", e.RepoPath, e.Error)
		}
	}

	return nil
}

// GetSearchFilter returns the current search filter
func (h *EventHandler) GetSearchFilter() *logic.SearchFilter {
	return h.searchFilter
}

// UpdateSearchFilter updates the search filter with current repositories
func (h *EventHandler) UpdateSearchFilter() {
	h.searchFilter = logic.NewSearchFilter(h.state.Repositories)
}