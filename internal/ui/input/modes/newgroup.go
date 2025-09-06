package modes

import (
	"github.com/charmbracelet/bubbles/textinput"
	"gitagrip/internal/ui/input/types"
)

type NewGroupMode struct {
	TextInputMode
}

func NewNewGroupMode(ti *textinput.Model) *NewGroupMode {
	return &NewGroupMode{
		TextInputMode: NewTextInputMode(types.ModeNewGroup, "new-group", "Enter new group name: ", ti),
	}
}