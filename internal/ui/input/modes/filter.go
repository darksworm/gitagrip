package modes

import (
    "gitagrip/internal/ui/input/types"
    "github.com/charmbracelet/bubbles/v2/textinput"
)

type FilterMode struct {
	TextInputMode
}

func NewFilterMode(ti *textinput.Model) *FilterMode {
	return &FilterMode{
		TextInputMode: NewTextInputMode(types.ModeFilter, "filter", "Filter: ", ti),
	}
}

// Enter overrides the base Enter to expand all groups for better filter visibility
func (m *FilterMode) Enter(ctx types.Context) []types.Action {
	// First call the base Enter to handle text input setup
	actions := m.TextInputMode.Enter(ctx)

	// Then expand all groups
	return append(actions, types.ExpandAllGroupsAction{})
}
