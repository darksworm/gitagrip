package modes

import (
	"github.com/charmbracelet/bubbles/textinput"
	"gitagrip/internal/ui/input/types"
)

type SearchMode struct {
	TextInputMode
}

func NewSearchMode(ti *textinput.Model) *SearchMode {
	return &SearchMode{
		TextInputMode: NewTextInputMode(types.ModeSearch, "search", "Search: ", ti),
	}
}

// Enter overrides the base Enter to expand all groups for better search visibility
func (m *SearchMode) Enter(ctx types.Context) []types.Action {
	// First call the base Enter to handle text input setup
	actions := m.TextInputMode.Enter(ctx)
	
	// Then expand all groups
	return append(actions, types.ExpandAllGroupsAction{})
}