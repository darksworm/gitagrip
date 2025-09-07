package modes

import (
	"gitagrip/internal/ui/input/types"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type NewGroupMode struct {
	textInputMode TextInputMode
}

func NewNewGroupMode(ti *textinput.Model) *NewGroupMode {
	return &NewGroupMode{
		textInputMode: NewTextInputMode(types.ModeNewGroup, "new-group", "Enter new group name: ", ti),
	}
}

func (m *NewGroupMode) Name() string {
	return m.textInputMode.Name()
}

func (m *NewGroupMode) Enter(ctx types.Context) []types.Action {
	return m.textInputMode.Enter(ctx)
}

func (m *NewGroupMode) Exit(ctx types.Context) []types.Action {
	return m.textInputMode.Exit(ctx)
}

func (m *NewGroupMode) HandleKey(msg tea.KeyMsg, ctx types.Context) ([]types.Action, bool) {
	// Let the base TextInputMode handle all keys including Enter
	// It will send a SubmitTextAction when Enter is pressed
	return m.textInputMode.HandleKey(msg, ctx)
}
