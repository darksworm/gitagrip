package viewmodels

import (
	"github.com/charmbracelet/bubbles/textinput"
)

// InputMode represents the different input modes
type InputMode int

const (
	InputModeNormal InputMode = iota
	InputModeNewGroup
	InputModeMoveToGroup
	InputModeDeleteConfirm
	InputModeSearch
	InputModeFilter
	InputModeSort
	InputModeRenameGroup
)

// InputTransformer handles input mode transformations
type InputTransformer struct {
	mode      InputMode
	textInput textinput.Model
}

// NewInputTransformer creates a new input transformer
func NewInputTransformer(textInput textinput.Model) *InputTransformer {
	return &InputTransformer{
		mode:      InputModeNormal,
		textInput: textInput,
	}
}

// SetMode sets the current input mode
func (it *InputTransformer) SetMode(mode InputMode) {
	it.mode = mode
}

// GetInputText returns the current text input string for the view
func (it *InputTransformer) GetInputText() string {
	if it.mode == InputModeNormal {
		return ""
	}
	
	switch it.mode {
	case InputModeDeleteConfirm:
		return "Disband group? (y/n): "
	case InputModeNewGroup:
		return "Enter new group name: " + it.textInput.View()
	case InputModeMoveToGroup:
		return "Move to group: " + it.textInput.View()
	case InputModeSearch:
		return "Search: " + it.textInput.View()
	case InputModeFilter:
		return "Filter: " + it.textInput.View()
	case InputModeSort:
		// Sort mode now uses interactive selection, not text input
		return ""
	case InputModeRenameGroup:
		return "Rename group to: " + it.textInput.View()
	default:
		return it.textInput.View()
	}
}

// GetInputModeString returns the string representation of the input mode
func (it *InputTransformer) GetInputModeString() string {
	switch it.mode {
	case InputModeNormal:
		return ""
	case InputModeNewGroup:
		return "new-group"
	case InputModeMoveToGroup:
		return "move-to-group"
	case InputModeDeleteConfirm:
		return "delete-confirm"
	case InputModeSearch:
		return "search"
	case InputModeFilter:
		return "filter"
	case InputModeSort:
		return "sort"
	case InputModeRenameGroup:
		return "rename-group"
	default:
		return ""
	}
}