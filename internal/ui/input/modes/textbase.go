package modes

import (
	"gitagrip/internal/ui/input/types"
	"github.com/charmbracelet/bubbles/v2/textinput"
	tea "github.com/charmbracelet/bubbletea/v2"
)

// TextInputMode is a base for modes that accept text input
type TextInputMode struct {
	mode      types.Mode
	name      string
	prompt    string
	textInput *textinput.Model
}

func NewTextInputMode(mode types.Mode, name, prompt string, ti *textinput.Model) TextInputMode {
	return TextInputMode{
		mode:      mode,
		name:      name,
		prompt:    prompt,
		textInput: ti,
	}
}

func (m TextInputMode) Name() string {
	return m.name
}

func (m TextInputMode) Enter(ctx types.Context) []types.Action {
	if m.textInput != nil {
		m.textInput.Reset()
		m.textInput.Focus()
		m.textInput.Prompt = "" // Prompt is handled in the UI layer
	}
	return nil
}

func (m TextInputMode) Exit(ctx types.Context) []types.Action {
	if m.textInput != nil {
		m.textInput.Blur()
		m.textInput.Reset()
	}
	return nil
}

func (m TextInputMode) HandleKey(msg tea.KeyMsg, ctx types.Context) ([]types.Action, bool) {
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
		// Submit the text
		text := ""
		if m.textInput != nil {
			text = m.textInput.Value()
		}
		return []types.Action{
			types.SubmitTextAction{Text: text, Mode: m.mode},
			types.ChangeModeAction{Mode: types.ModeNormal},
		}, true
	default:
		// Let the main handler update the text input
		// Returning false here means the input handler will process it
		return nil, false
	}
}
