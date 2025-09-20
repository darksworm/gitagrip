package modes

import (
	"gitagrip/internal/ui/input/types"
	"github.com/charmbracelet/bubbles/v2/textinput"
)

// NewBranchMode prompts for a new branch name
type NewBranchMode struct {
	TextInputMode
}

func NewNewBranchMode(ti *textinput.Model) *NewBranchMode {
	return &NewBranchMode{TextInputMode: NewTextInputMode(types.ModeNewBranch, "new-branch", "New branch name: ", ti)}
}

// SwitchBranchMode prompts for an existing branch name to switch to
type SwitchBranchMode struct {
	TextInputMode
}

func NewSwitchBranchMode(ti *textinput.Model) *SwitchBranchMode {
	return &SwitchBranchMode{TextInputMode: NewTextInputMode(types.ModeSwitchBranch, "switch-branch", "Switch to branch: ", ti)}
}
