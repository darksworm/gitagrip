package modes

import (
	"gitagrip/internal/ui/input/types"
	"github.com/charmbracelet/bubbles/textinput"
)

type SortMode struct {
	TextInputMode
}

func NewSortMode(ti *textinput.Model) *SortMode {
	return &SortMode{
		TextInputMode: NewTextInputMode(types.ModeSort, "sort", "Sort by: ", ti),
	}
}
