package modes

import (
	"gitagrip/internal/ui/input/types"
	tea "github.com/charmbracelet/bubbletea/v2"
)

// SortOptions available for sorting
var SortOptions = []struct {
	Key         string
	Name        string
	Description string
}{
	{"name", "Name", "Sort by repository name"},
	{"status", "Status", "Sort by status (dirty, clean)"},
	{"branch", "Branch", "Sort by branch name"},
}

type SortSelectMode struct {
	sortIndex     int
	originalIndex int // Remember the original sort when entering
}

func NewSortSelectMode() *SortSelectMode {
	return &SortSelectMode{
		sortIndex: 0,
	}
}

func (m *SortSelectMode) Name() string {
	return "sort"
}

func (m *SortSelectMode) Enter(ctx types.Context) []types.Action {
	// Start with the current sort option
	currentSort := ctx.GetCurrentSort()
	m.sortIndex = 0
	m.originalIndex = 0

	// Find the index of the current sort
	for i, option := range SortOptions {
		if option.Key == string(currentSort) {
			m.sortIndex = i
			m.originalIndex = i
			break
		}
	}

	return []types.Action{types.UpdateSortIndexAction{Index: m.sortIndex}}
}

func (m *SortSelectMode) Exit(ctx types.Context) []types.Action {
	return nil // No special actions on exit
}

// HandleKey processes key messages for sort selection
func (m *SortSelectMode) HandleKey(msg tea.KeyMsg, ctx types.Context) ([]types.Action, bool) {
	switch msg.String() {
	case "esc":
		// Cancel and restore original sort
		return []types.Action{
			types.SortByAction{Criteria: SortOptions[m.originalIndex].Key},
			types.ChangeModeAction{Mode: types.ModeNormal},
		}, true

	case "enter":
		// Accept current sort and return to normal mode
		return []types.Action{
			types.ChangeModeAction{Mode: types.ModeNormal},
		}, true

	case "up", "down":
		// Navigate through sort options and apply immediately
		if msg.String() == "up" {
			m.sortIndex--
			if m.sortIndex < 0 {
				m.sortIndex = len(SortOptions) - 1
			}
		} else {
			m.sortIndex++
			if m.sortIndex >= len(SortOptions) {
				m.sortIndex = 0
			}
		}
		// Update the UI and apply sort immediately
		return []types.Action{
			types.UpdateSortIndexAction{Index: m.sortIndex},
			types.SortByAction{Criteria: SortOptions[m.sortIndex].Key},
		}, true
	}

	// Handle string keys
	switch msg.String() {
	case "j":
		// Down
		m.sortIndex++
		if m.sortIndex >= len(SortOptions) {
			m.sortIndex = 0
		}
		return []types.Action{
			types.UpdateSortIndexAction{Index: m.sortIndex},
			types.SortByAction{Criteria: SortOptions[m.sortIndex].Key},
		}, true

	case "k":
		// Up
		m.sortIndex--
		if m.sortIndex < 0 {
			m.sortIndex = len(SortOptions) - 1
		}
		return []types.Action{
			types.UpdateSortIndexAction{Index: m.sortIndex},
			types.SortByAction{Criteria: SortOptions[m.sortIndex].Key},
		}, true

	case "q":
		// Cancel and restore original sort
		return []types.Action{
			types.SortByAction{Criteria: SortOptions[m.originalIndex].Key},
			types.ChangeModeAction{Mode: types.ModeNormal},
		}, true
	}

	return nil, false
}

// GetCurrentIndex returns the current sort option index
func (m *SortSelectMode) GetCurrentIndex() int {
	return m.sortIndex
}
