package modes

import (
	"gitagrip/internal/ui/input/types"
	"github.com/charmbracelet/bubbles/v2/textinput"
	tea "github.com/charmbracelet/bubbletea/v2"
)

type RenameGroupMode struct {
	textInput *textinput.Model
	oldName   string
}

func NewRenameGroupMode(ti *textinput.Model) *RenameGroupMode {
	return &RenameGroupMode{
		textInput: ti,
	}
}

func (m *RenameGroupMode) Name() string {
	return "rename group"
}

func (m *RenameGroupMode) Enter(ctx types.Context) []types.Action {
	if m.textInput != nil {
		m.textInput.Reset()
		m.textInput.Focus()
		// Pre-fill with current group name
		if groupName := ctx.CurrentGroupName(); groupName != "" {
			m.oldName = groupName
			m.textInput.SetValue(groupName)
			m.textInput.CursorEnd()
		}
	}
	return nil
}

func (m *RenameGroupMode) Exit(ctx types.Context) []types.Action {
	if m.textInput != nil {
		m.textInput.Blur()
		m.textInput.Reset()
	}
	m.oldName = ""
	return nil
}

func (m *RenameGroupMode) HandleKey(msg tea.KeyMsg, ctx types.Context) ([]types.Action, bool) {
	switch msg.String() {
	case "ctrl+c":
		return []types.Action{types.QuitAction{Force: true}}, true

	case "esc":
		// Cancel and return to normal mode
		return []types.Action{
			types.CancelTextAction{},
			types.ChangeModeAction{Mode: types.ModeNormal},
		}, true

	case "enter":
		// Submit the rename
		newName := ""
		if m.textInput != nil {
			newName = m.textInput.Value()
		}

		// Only rename if the name changed and is not empty
		if newName != "" && newName != m.oldName {
			return []types.Action{
				types.RenameGroupAction{OldName: m.oldName, NewName: newName},
				types.ChangeModeAction{Mode: types.ModeNormal},
			}, true
		}

		// Just cancel if no change
		return []types.Action{
			types.CancelTextAction{},
			types.ChangeModeAction{Mode: types.ModeNormal},
		}, true

	default:
		// Let the main handler update the text input
		return nil, false
	}
}
