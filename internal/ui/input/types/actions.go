package types

// Navigation actions
type NavigateAction struct {
	Direction string // "up", "down", "pageup", "pagedown", "home", "end", "left", "right"
}

func (a NavigateAction) Type() string { return "navigate" }

// Selection actions
type SelectAction struct {
	Index int // -1 for current
}

func (a SelectAction) Type() string { return "select" }

type SelectAllAction struct{}

func (a SelectAllAction) Type() string { return "select_all" }

type DeselectAllAction struct{}

func (a DeselectAllAction) Type() string { return "deselect_all" }

type SelectGroupAction struct {
	GroupName string
}

func (a SelectGroupAction) Type() string { return "select_group" }

// Mode transition actions
type ChangeModeAction struct {
	Mode Mode
	Data interface{} // Optional data for the mode
}

func (a ChangeModeAction) Type() string { return "change_mode" }

// Text input actions
type UpdateTextAction struct {
	Text string
}

func (a UpdateTextAction) Type() string { return "update_text" }

type SubmitTextAction struct {
	Text string
	Mode Mode // Which mode submitted the text
}

func (a SubmitTextAction) Type() string { return "submit_text" }

type CancelTextAction struct{}

func (a CancelTextAction) Type() string { return "cancel_text" }

// Command actions
type RefreshAction struct {
	All bool // true for full scan, false for status refresh
}

func (a RefreshAction) Type() string { return "refresh" }

type FetchAction struct{}

func (a FetchAction) Type() string { return "fetch" }

type PullAction struct{}

func (a PullAction) Type() string { return "pull" }

type OpenLogAction struct{}

func (a OpenLogAction) Type() string { return "open_log" }

type ToggleInfoAction struct{}

func (a ToggleInfoAction) Type() string { return "toggle_info" }

type ToggleHelpAction struct{}

func (a ToggleHelpAction) Type() string { return "toggle_help" }

type CreateGroupAction struct {
	Name string
}

func (a CreateGroupAction) Type() string { return "create_group" }

type MoveToGroupAction struct {
	GroupName string
}

func (a MoveToGroupAction) Type() string { return "move_to_group" }

type DeleteGroupAction struct {
	GroupName string
}

func (a DeleteGroupAction) Type() string { return "delete_group" }

type RenameGroupAction struct {
	OldName string
	NewName string
}

func (a RenameGroupAction) Type() string { return "rename_group" }

type ToggleGroupAction struct{}

func (a ToggleGroupAction) Type() string { return "toggle_group" }

type ExpandAllGroupsAction struct{}

func (a ExpandAllGroupsAction) Type() string { return "expand_all_groups" }

type MoveGroupUpAction struct{}

func (a MoveGroupUpAction) Type() string { return "move_group_up" }

type MoveGroupDownAction struct{}

func (a MoveGroupDownAction) Type() string { return "move_group_down" }

type SearchNavigateAction struct {
	Direction string // "next" or "prev"
}

func (a SearchNavigateAction) Type() string { return "search_navigate" }

type QuitAction struct {
	Force bool // true for Ctrl+C, false for 'q'
}

func (a QuitAction) Type() string { return "quit" }

type HideAction struct{}

func (a HideAction) Type() string { return "hide" }

// Sort actions
type SortByAction struct {
	Criteria string
}

func (a SortByAction) Type() string { return "sort_by" }

type UpdateSortIndexAction struct {
	Index int
}

func (a UpdateSortIndexAction) Type() string { return "update_sort_index" }
