package modes

import (
	"github.com/charmbracelet/bubbles/textinput"
	"gitagrip/internal/ui/input/types"
)

type SearchMode struct {
	TextInputMode
}

func NewSearchMode(ti *textinput.Model) *SearchMode {
	return &SearchMode{
		TextInputMode: NewTextInputMode(types.ModeSearch, "search", "Search: ", ti),
	}
}