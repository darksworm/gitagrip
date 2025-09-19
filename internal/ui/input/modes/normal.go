package modes

import (
	"gitagrip/internal/ui/input/types"
	tea "github.com/charmbracelet/bubbletea"
	"time"
)

type NormalMode struct {
	lastKeyWasG bool
	lastGTime   time.Time
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
		// Enter toggles group when on a group header; otherwise open lazygit for the repository
		if ctx.IsOnGroup() {
			return []types.Action{types.ToggleGroupAction{}}, true
		}
		if ctx.CurrentRepositoryPath() != "" {
			return []types.Action{types.OpenLazygitAction{}}, true
		}
		return nil, false
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

	case "z":
		// z toggles group expansion (works on group header or repo in group)
		if ctx.IsOnGroup() || ctx.GetRepoPathAtIndex(ctx.CurrentIndex()) != "" {
			return []types.Action{types.ToggleGroupAction{}}, true
		}
		return nil, false

	case "J":
		// Shift+J moves group down
		if ctx.IsOnGroup() || ctx.GetRepoPathAtIndex(ctx.CurrentIndex()) != "" {
			return []types.Action{types.MoveGroupDownAction{}}, true
		}
		return nil, false

	case "K":
		// Shift+K moves group up
		if ctx.IsOnGroup() || ctx.GetRepoPathAtIndex(ctx.CurrentIndex()) != "" {
			return []types.Action{types.MoveGroupUpAction{}}, true
		}
		return nil, false

	case "shift+up":
		// Shift+Up moves group up
		if ctx.IsOnGroup() || ctx.GetRepoPathAtIndex(ctx.CurrentIndex()) != "" {
			return []types.Action{types.MoveGroupUpAction{}}, true
		}
		return nil, false

	case "shift+down":
		// Shift+Down moves group down
		if ctx.IsOnGroup() || ctx.GetRepoPathAtIndex(ctx.CurrentIndex()) != "" {
			return []types.Action{types.MoveGroupDownAction{}}, true
		}
		return nil, false

	case " ":
		// Space toggles selection on repo or selects/deselects all in group
		if ctx.IsOnGroup() {
			return []types.Action{types.SelectGroupAction{GroupName: ctx.CurrentGroupName()}}, true
		}
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
		// Rename group (only if on a group)
		if ctx.IsOnGroup() {
			return []types.Action{types.ChangeModeAction{
				Mode: types.ModeRenameGroup,
				Data: ctx.CurrentGroupName(),
			}}, true
		}
		return nil, false

	case "f":
		// Fetch selected repos, current repo, or all repos in group
		if ctx.HasSelection() || ctx.CurrentRepositoryPath() != "" || ctx.IsOnGroup() {
			return []types.Action{types.FetchAction{}}, true
		}
		return nil, false

	case "p", "P":
		// Pull selected repos, current repo, or all repos in group
		if ctx.HasSelection() || ctx.CurrentRepositoryPath() != "" || ctx.IsOnGroup() {
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
		// Navigate to next search result
		if ctx.SearchQuery() != "" {
			return []types.Action{types.SearchNavigateAction{Direction: "next"}}, true
		}
		return nil, true // Consume the key even if no action

	case "N":
		// New group (only if selection)
		if ctx.HasSelection() {
			return []types.Action{types.ChangeModeAction{Mode: types.ModeNewGroup}}, true
		}
		// Otherwise, navigate to previous search result
		if ctx.SearchQuery() != "" {
			return []types.Action{types.SearchNavigateAction{Direction: "prev"}}, true
		}
		return nil, true // Consume the key even if no action

	case "m":
		// Move to group (only if selection or on repo)
		if ctx.HasSelection() || (ctx.CurrentRepositoryPath() != "" && !ctx.IsOnGroup()) {
			return []types.Action{types.ChangeModeAction{Mode: types.ModeMoveToGroup}}, true
		}
		return nil, false

	case "H":
		// Open commit history (git log) for the current repository
		if ctx.CurrentRepositoryPath() != "" && !ctx.IsOnGroup() {
			return []types.Action{types.OpenLogAction{}}, true
		}
		return nil, false

	case "d":
		// Delete group (only if on a group)
		if ctx.IsOnGroup() {
			return []types.Action{types.ChangeModeAction{Mode: types.ModeDeleteConfirm}}, true
		}
		return nil, false

	case "D":
		// Show git diff for current repo
		if ctx.CurrentRepositoryPath() != "" && !ctx.IsOnGroup() {
			return []types.Action{types.OpenDiffAction{}}, true
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

	case "esc":
		// Clear selection if any, otherwise do nothing
		if ctx.HasSelection() {
			return []types.Action{types.DeselectAllAction{}}, true
		}
		return nil, true // Consume the key even if no action

	case "q":
		// Quit
		return []types.Action{types.QuitAction{Force: false}}, true

	case "g":
		if m.lastKeyWasG && time.Since(m.lastGTime) < 500*time.Millisecond {
			// gg - go to top (within timeout)
			m.lastKeyWasG = false
			return []types.Action{types.NavigateAction{Direction: "home"}}, true
		} else {
			// First g, wait for next key
			m.lastKeyWasG = true
			m.lastGTime = time.Now()
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
		// Also cancel if too much time has passed since first 'g'
		if m.lastKeyWasG && time.Since(m.lastGTime) >= 500*time.Millisecond {
			m.lastKeyWasG = false
		}
	}

	return nil, false
}
