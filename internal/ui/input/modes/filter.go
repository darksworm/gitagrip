package modes

import (
	"github.com/charmbracelet/bubbles/textinput"
	"gitagrip/internal/ui/input/types"
)

type FilterMode struct {
	TextInputMode
}

func NewFilterMode(ti *textinput.Model) *FilterMode {
	return &FilterMode{
		TextInputMode: NewTextInputMode(types.ModeFilter, "filter", "Filter: ", ti),
	}
}