package modes

import (
	tea "github.com/charmbracelet/bubbletea"
	"gitagrip/internal/ui/input/types"
)

type NormalMode struct{
	lastKeyWasG bool
}

func NewNormalMode() *NormalMode {
	return &NormalMode{}
}

func (m *NormalMode) Name() string {
	return "normal"
}

func (m *NormalMode) Enter(ctx types.Context) []types.Action {
	return nil // No special actions on enter
}

func (m *NormalMode) Exit(ctx types.Context) []types.Action {
	return nil // No special actions on exit
}

func (m *NormalMode) HandleKey(msg tea.KeyMsg, ctx types.Context) ([]types.Action, bool) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return []types.Action{types.QuitAction{Force: true}}, true
		
	case tea.KeyEsc:
		// In normal mode, Esc doesn't do anything
		return nil, false
		
	case tea.KeyUp:
		return []types.Action{types.NavigateAction{Direction: "up"}}, true
		
	case tea.KeyDown:
		return []types.Action{types.NavigateAction{Direction: "down"}}, true
		
	case tea.KeyLeft:
		return []types.Action{types.NavigateAction{Direction: "left"}}, true
		
	case tea.KeyRight:
		return []types.Action{types.NavigateAction{Direction: "right"}}, true
		
	case tea.KeyPgUp:
		return []types.Action{types.NavigateAction{Direction: "pageup"}}, true
		
	case tea.KeyPgDown:
		return []types.Action{types.NavigateAction{Direction: "pagedown"}}, true
		
	case tea.KeyHome:
		return []types.Action{types.NavigateAction{Direction: "home"}}, true
		
	case tea.KeyEnd:
		return []types.Action{types.NavigateAction{Direction: "end"}}, true
		
	case tea.KeyEnter:
		// Enter toggles group or opens log on repository
		if ctx.IsOnGroup() {
			return []types.Action{types.ToggleGroupAction{}}, true
		}
		return []types.Action{types.OpenLogAction{}}, true
	}
	
	// Handle string keys
	switch msg.String() {
	case "j":
		return []types.Action{types.NavigateAction{Direction: "down"}}, true
		
	case "k":
		return []types.Action{types.NavigateAction{Direction: "up"}}, true
		
	case "h":
		return []types.Action{types.NavigateAction{Direction: "left"}}, true
		
	case "l":
		return []types.Action{types.NavigateAction{Direction: "right"}}, true
		
	case " ":
		// Space toggles selection
		return []types.Action{types.SelectAction{Index: -1}}, true
		
	case "a", "A":
		// Toggle select all
		if ctx.HasSelection() {
			return []types.Action{types.DeselectAllAction{}}, true
		}
		return []types.Action{types.SelectAllAction{}}, true
		
	case "r":
		// Refresh status
		return []types.Action{types.RefreshAction{All: false}}, true
		
	case "R":
		// Full refresh (rescan)
		return []types.Action{types.RefreshAction{All: true}}, true
		
	case "f":
		// Fetch selected repos
		if ctx.HasSelection() || ctx.CurrentRepositoryPath() != "" {
			return []types.Action{types.FetchAction{}}, true
		}
		return nil, false
		
	case "p", "P":
		// Pull selected repos
		if ctx.HasSelection() || ctx.CurrentRepositoryPath() != "" {
			return []types.Action{types.PullAction{}}, true
		}
		return nil, false
		
	case "/":
		// Enter search mode
		return []types.Action{types.ChangeModeAction{Mode: types.ModeSearch}}, true
		
	case "ctrl+f", "F":
		// Enter filter mode
		return []types.Action{types.ChangeModeAction{Mode: types.ModeFilter}}, true
		
	case "n":
		// New group
		return []types.Action{types.ChangeModeAction{Mode: types.ModeNewGroup}}, true
		
	case "m":
		// Move to group (only if selection or on repo)
		if ctx.HasSelection() || (ctx.CurrentRepositoryPath() != "" && !ctx.IsOnGroup()) {
			return []types.Action{types.ChangeModeAction{Mode: types.ModeMoveToGroup}}, true
		}
		return nil, false
		
	case "d":
		// Delete group (only if on group)
		if ctx.IsOnGroup() {
			return []types.Action{types.ChangeModeAction{
				Mode: types.ModeDeleteConfirm,
				Data: ctx.CurrentGroupName(),
			}}, true
		}
		return nil, false
		
	case "s":
		// Sort mode
		return []types.Action{types.ChangeModeAction{Mode: types.ModeSort}}, true
		
	case "?":
		// Toggle help
		return []types.Action{types.ToggleHelpAction{}}, true
		
	case "i", "I":
		// Toggle info
		return []types.Action{types.ToggleInfoAction{}}, true
		
	case "L":
		// Open log for current repo
		if ctx.CurrentRepositoryPath() != "" && !ctx.IsOnGroup() {
			return []types.Action{types.OpenLogAction{}}, true
		}
		return nil, false
		
	case "q":
		// Quit
		return []types.Action{types.QuitAction{Force: false}}, true
		
	case "g":
		if m.lastKeyWasG {
			// gg - go to top
			m.lastKeyWasG = false
			return []types.Action{types.NavigateAction{Direction: "home"}}, true
		} else {
			// First g, wait for next key
			m.lastKeyWasG = true
			return nil, true // consume the key but don't do anything
		}
		
	case "G":
		// G - go to bottom
		m.lastKeyWasG = false
		return []types.Action{types.NavigateAction{Direction: "end"}}, true
		
	default:
		// Any other key cancels the 'g' prefix
		if m.lastKeyWasG && msg.String() != "g" {
			m.lastKeyWasG = false
		}
	}
	
	return nil, false
}