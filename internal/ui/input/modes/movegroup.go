package modes

import (
	"gitagrip/internal/ui/input/types"
	"github.com/charmbracelet/bubbles/v2/textinput"
)

type MoveToGroupMode struct {
	TextInputMode
}

func NewMoveToGroupMode(ti *textinput.Model) *MoveToGroupMode {
	return &MoveToGroupMode{
		TextInputMode: NewTextInputMode(types.ModeMoveToGroup, "move-to-group", "Move to group: ", ti),
	}
}
