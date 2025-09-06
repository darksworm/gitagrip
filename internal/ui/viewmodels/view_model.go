package viewmodels

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/textinput"
	
	"gitagrip/internal/config"
	"gitagrip/internal/ui/state"
	"gitagrip/internal/ui/views"
)

// ViewModel transforms application state into view-ready data
type ViewModel struct {
	state            *state.AppState
	config           *config.Config
	width            int
	height           int
	help             help.Model
	deleteTarget     string
	ungroupedRepos   []string
	inputTransformer *InputTransformer
}

// NewViewModel creates a new view model
func NewViewModel(appState *state.AppState, cfg *config.Config, textInput textinput.Model) *ViewModel {
	return &ViewModel{
		state:            appState,
		config:           cfg,
		inputTransformer: NewInputTransformer(textInput),
	}
}

// SetDimensions sets the current terminal dimensions
func (vm *ViewModel) SetDimensions(width, height int) {
	vm.width = width
	vm.height = height
}

// SetHelp sets the help model
func (vm *ViewModel) SetHelp(helpModel help.Model) {
	vm.help = helpModel
}

// SetDeleteTarget sets the current delete target
func (vm *ViewModel) SetDeleteTarget(target string) {
	vm.deleteTarget = target
}

// SetInputMode sets the current input mode
func (vm *ViewModel) SetInputMode(mode InputMode) {
	vm.inputTransformer.SetMode(mode)
}

// UpdateTextInput updates the text input model
func (vm *ViewModel) UpdateTextInput(textInput textinput.Model) {
	vm.inputTransformer.textInput = textInput
}

// SetUngroupedRepos sets the ungrouped repositories
func (vm *ViewModel) SetUngroupedRepos(repos []string) {
	vm.ungroupedRepos = repos
}

// BuildViewState creates a ViewState for rendering
func (vm *ViewModel) BuildViewState() views.ViewState {
	return views.ViewState{
		Width:            vm.width,
		Height:           vm.height,
		Repositories:     vm.state.Repositories,
		Groups:           vm.state.Groups,
		OrderedGroups:    vm.state.OrderedGroups,
		SelectedIndex:    vm.state.SelectedIndex,
		SelectedRepos:    vm.state.SelectedRepos,
		RefreshingRepos:  vm.state.RefreshingRepos,
		FetchingRepos:    vm.state.FetchingRepos,
		PullingRepos:     vm.state.PullingRepos,
		ExpandedGroups:   vm.state.ExpandedGroups,
		Scanning:         vm.state.Scanning,
		StatusMessage:    vm.state.StatusMessage,
		ShowHelp:         vm.state.ShowHelp,
		ShowLog:          vm.state.ShowLog,
		LogContent:       vm.state.LogContent,
		ShowInfo:         vm.state.ShowInfo,
		InfoContent:      vm.state.InfoContent,
		ViewportOffset:   vm.state.ViewportOffset,
		ViewportHeight:   vm.state.ViewportHeight,
		SearchQuery:      vm.state.SearchQuery,
		FilterQuery:      vm.state.FilterQuery,
		IsFiltered:       vm.state.IsFiltered,
		ShowAheadBehind:  vm.config.UISettings.ShowAheadBehind,
		HelpModel:        vm.help,
		DeleteTarget:     vm.deleteTarget,
		TextInput:        vm.inputTransformer.GetInputText(),
		InputMode:        vm.inputTransformer.GetInputModeString(),
		UngroupedRepos:   vm.ungroupedRepos,
	}
}