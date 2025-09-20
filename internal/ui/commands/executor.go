package commands

import (
	tea "github.com/charmbracelet/bubbletea/v2"

	"gitagrip/internal/eventbus"
	"gitagrip/internal/ui/state"
)

// Executor handles command execution
type Executor struct {
	ctx *CommandContext
}

// NewExecutor creates a new command executor
func NewExecutor(state *state.AppState, bus eventbus.EventBus) *Executor {
	return &Executor{
		ctx: &CommandContext{
			State: state,
			Bus:   bus,
		},
	}
}

// ExecuteRefresh creates and executes a refresh command
func (e *Executor) ExecuteRefresh(repoPaths []string) tea.Cmd {
	cmd := NewRefreshCommand(e.ctx, repoPaths)
	return cmd.Execute()
}

// ExecuteFetch creates and executes a fetch command
func (e *Executor) ExecuteFetch(repoPaths []string) tea.Cmd {
	cmd := NewFetchCommand(e.ctx, repoPaths)
	return cmd.Execute()
}

// ExecutePull creates and executes a pull command
func (e *Executor) ExecutePull(repoPaths []string) tea.Cmd {
	cmd := NewPullCommand(e.ctx, repoPaths)
	return cmd.Execute()
}

// ExecuteFullScan creates and executes a full scan command
func (e *Executor) ExecuteFullScan(scanPath string) tea.Cmd {
	cmd := NewFullScanCommand(e.ctx, scanPath)
	return cmd.Execute()
}

// ExecuteToggleSelection creates and executes a toggle selection command
func (e *Executor) ExecuteToggleSelection(repoPath string) tea.Cmd {
	cmd := NewToggleSelectionCommand(e.ctx, repoPath)
	return cmd.Execute()
}

// ExecuteSelectAll creates and executes a select all command
func (e *Executor) ExecuteSelectAll(totalRepos int) tea.Cmd {
	cmd := NewSelectAllCommand(e.ctx, totalRepos)
	return cmd.Execute()
}

// ExecuteMoveToGroup creates and executes a move to group command
func (e *Executor) ExecuteMoveToGroup(repoPaths []string, fromGroups map[string]string, toGroup string) tea.Cmd {
	cmd := NewMoveToGroupCommand(e.ctx, repoPaths, fromGroups, toGroup)
	return cmd.Execute()
}
