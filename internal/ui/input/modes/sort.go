package modes

import (
	"github.com/charmbracelet/bubbles/textinput"
	"gitagrip/internal/ui/input/types"
)

type SortMode struct {
	TextInputMode
}

func NewSortMode(ti *textinput.Model) *SortMode {
	return &SortMode{
		TextInputMode: NewTextInputMode(types.ModeSort, "sort", "Sort by: ", ti),
	}
}