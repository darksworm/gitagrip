package input

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
	"gitagrip/internal/ui/input/modes"
	"gitagrip/internal/ui/input/types"
)

type Handler struct {
	currentMode types.Mode
	modes       map[types.Mode]types.ModeHandler
	textInput   textinput.Model // Shared text input for text modes
}

func New() *Handler {
	ti := textinput.New()
	
	h := &Handler{
		currentMode: types.ModeNormal,
		textInput:   ti,
		modes:       make(map[types.Mode]types.ModeHandler),
	}
	
	// Register all mode handlers
	h.modes[types.ModeNormal] = modes.NewNormalMode()
	h.modes[types.ModeSearch] = modes.NewSearchMode(&ti)
	h.modes[types.ModeFilter] = modes.NewFilterMode(&ti)
	h.modes[types.ModeNewGroup] = modes.NewNewGroupMode(&ti)
	h.modes[types.ModeMoveToGroup] = modes.NewMoveToGroupMode(&ti)
	h.modes[types.ModeDeleteConfirm] = modes.NewConfirmMode()
	h.modes[types.ModeSort] = modes.NewSortMode(&ti)
	
	return h
}

func (h *Handler) HandleKey(msg tea.KeyMsg, ctx types.Context) ([]types.Action, tea.Cmd) {
	handler := h.modes[h.currentMode]
	if handler == nil {
		return nil, nil
	}
	
	actions, consumed := handler.HandleKey(msg, ctx)
	
	var cmd tea.Cmd
	var allActions []types.Action
	
	// If not consumed and we're in text mode, we'll handle it below
	if !consumed && !h.isTextMode(h.currentMode) {
		return nil, nil
	}
	
	// Handle mode changes
	for _, action := range actions {
		if changeMode, ok := action.(types.ChangeModeAction); ok {
			// Exit current mode
			if h.modes[h.currentMode] != nil {
				exitActions := h.modes[h.currentMode].Exit(ctx)
				allActions = append(allActions, exitActions...)
			}
			
			// Change mode
			oldMode := h.currentMode
			h.currentMode = changeMode.Mode
			
			// Enter new mode
			if h.modes[h.currentMode] != nil {
				enterActions := h.modes[h.currentMode].Enter(ctx)
				allActions = append(allActions, enterActions...)
			}
			
			// Handle text input focus
			if h.isTextMode(h.currentMode) {
				h.textInput.Reset()
				h.textInput.Focus()
				cmd = textinput.Blink
			} else if h.isTextMode(oldMode) {
				h.textInput.Blur()
			}
		} else {
			allActions = append(allActions, action)
		}
	}
	
	// If we're in a text mode and didn't handle the key, pass it to text input
	if h.isTextMode(h.currentMode) && (!consumed || len(actions) == 0) {
		var textCmd tea.Cmd
		h.textInput, textCmd = h.textInput.Update(msg)
		cmd = textCmd
		// Always append an update action when in text mode to keep view in sync
		allActions = append(allActions, types.UpdateTextAction{Text: h.textInput.Value()})
	}
	
	return allActions, cmd
}

func (h *Handler) CurrentMode() types.Mode {
	return h.currentMode
}

func (h *Handler) TextInput() *textinput.Model {
	if h.isTextMode(h.currentMode) {
		return &h.textInput
	}
	return nil
}

func (h *Handler) RegisterMode(mode types.Mode, handler types.ModeHandler) {
	h.modes[mode] = handler
}

func (h *Handler) isTextMode(mode types.Mode) bool {
	switch mode {
	case types.ModeSearch, types.ModeFilter, types.ModeNewGroup, types.ModeMoveToGroup, types.ModeSort:
		return true
	default:
		return false
	}
}

func (h *Handler) Reset() {
	h.currentMode = types.ModeNormal
	h.textInput.Reset()
	h.textInput.Blur()
}

// Update handles non-keyboard messages for text input
func (h *Handler) Update(msg tea.Msg) tea.Cmd {
	if h.isTextMode(h.currentMode) {
		var cmd tea.Cmd
		h.textInput, cmd = h.textInput.Update(msg)
		return cmd
	}
	return nil
}