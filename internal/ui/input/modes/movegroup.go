package modes

import (
	"github.com/charmbracelet/bubbles/textinput"
	"gitagrip/internal/ui/input/types"
)

type MoveToGroupMode struct {
	TextInputMode
}

func NewMoveToGroupMode(ti *textinput.Model) *MoveToGroupMode {
	return &MoveToGroupMode{
		TextInputMode: NewTextInputMode(types.ModeMoveToGroup, "move-to-group", "Move to group: ", ti),
	}
}