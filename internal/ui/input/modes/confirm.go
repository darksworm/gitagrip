package modes

import (
	"gitagrip/internal/ui/input/types"
	tea "github.com/charmbracelet/bubbletea/v2"
)

type ConfirmMode struct {
	groupName string
}

func NewConfirmMode() *ConfirmMode {
	return &ConfirmMode{}
}

func (m *ConfirmMode) Name() string {
	return "delete-confirm"
}

func (m *ConfirmMode) Enter(ctx types.Context) []types.Action {
	// Store the group name when entering the mode
	if ctx.IsOnGroup() {
		m.groupName = ctx.CurrentGroupName()
	}
	return nil
}

func (m *ConfirmMode) Exit(ctx types.Context) []types.Action {
	return nil
}

func (m *ConfirmMode) HandleKey(msg tea.KeyMsg, ctx types.Context) ([]types.Action, bool) {
	switch msg.String() {
	case "ctrl+c":
		return []types.Action{types.QuitAction{Force: true}}, true
	case "esc":
		// Cancel and return to normal mode
		return []types.Action{types.ChangeModeAction{Mode: types.ModeNormal}}, true
	case "y", "Y":
		// Confirm deletion
		return []types.Action{
			types.DeleteGroupAction{GroupName: m.groupName},
			types.ChangeModeAction{Mode: types.ModeNormal},
		}, true

	case "n", "N":
		// Cancel deletion
		return []types.Action{types.ChangeModeAction{Mode: types.ModeNormal}}, true
	}

	return nil, false
}
