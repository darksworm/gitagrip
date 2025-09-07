package commands

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"gitagrip/internal/eventbus"
	"gitagrip/internal/ui/state"
)

// Command represents an executable action
type Command interface {
	Execute() tea.Cmd
}

// CommandContext provides context for command execution
type CommandContext struct {
	State *state.AppState
	Bus   eventbus.EventBus
}

// RefreshCommand refreshes repository status
type RefreshCommand struct {
	ctx       *CommandContext
	repoPaths []string
}

// NewRefreshCommand creates a new refresh command
func NewRefreshCommand(ctx *CommandContext, repoPaths []string) *RefreshCommand {
	return &RefreshCommand{
		ctx:       ctx,
		repoPaths: repoPaths,
	}
}

// Execute performs the refresh operation
func (c *RefreshCommand) Execute() tea.Cmd {
	if len(c.repoPaths) > 0 {
		c.ctx.State.SetRefreshing(c.repoPaths, true)
		if c.ctx.Bus != nil {
			c.ctx.Bus.Publish(eventbus.StatusRefreshRequestedEvent{
				RepoPaths: c.repoPaths,
			})
		}
	}
	return nil
}

// FetchCommand fetches from remote repositories
type FetchCommand struct {
	ctx       *CommandContext
	repoPaths []string
}

// NewFetchCommand creates a new fetch command
func NewFetchCommand(ctx *CommandContext, repoPaths []string) *FetchCommand {
	return &FetchCommand{
		ctx:       ctx,
		repoPaths: repoPaths,
	}
}

// Execute performs the fetch operation
func (c *FetchCommand) Execute() tea.Cmd {
	if len(c.repoPaths) > 0 {
		c.ctx.State.SetFetching(c.repoPaths, true)
		if c.ctx.Bus != nil {
			c.ctx.Bus.Publish(eventbus.FetchRequestedEvent{
				RepoPaths: c.repoPaths,
			})
		}
	}
	return nil
}

// PullCommand pulls from remote repositories
type PullCommand struct {
	ctx       *CommandContext
	repoPaths []string
}

// NewPullCommand creates a new pull command
func NewPullCommand(ctx *CommandContext, repoPaths []string) *PullCommand {
	return &PullCommand{
		ctx:       ctx,
		repoPaths: repoPaths,
	}
}

// Execute performs the pull operation
func (c *PullCommand) Execute() tea.Cmd {
	if len(c.repoPaths) > 0 {
		c.ctx.State.SetPulling(c.repoPaths, true)
		if c.ctx.Bus != nil {
			c.ctx.Bus.Publish(eventbus.PullRequestedEvent{
				RepoPaths: c.repoPaths,
			})
		}
	}
	return nil
}

// FullScanCommand initiates a full repository scan
type FullScanCommand struct {
	ctx      *CommandContext
	scanPath string
}

// NewFullScanCommand creates a new full scan command
func NewFullScanCommand(ctx *CommandContext, scanPath string) *FullScanCommand {
	return &FullScanCommand{
		ctx:      ctx,
		scanPath: scanPath,
	}
}

// Execute performs the full scan
func (c *FullScanCommand) Execute() tea.Cmd {
	c.ctx.State.Scanning = true
	c.ctx.State.StatusMessage = "Starting full scan..."
	if c.ctx.Bus != nil && c.scanPath != "" {
		c.ctx.Bus.Publish(eventbus.ScanRequestedEvent{
			Paths: []string{c.scanPath},
		})
	}
	return nil
}

// ToggleSelectionCommand toggles repository selection
type ToggleSelectionCommand struct {
	ctx      *CommandContext
	repoPath string
}

// NewToggleSelectionCommand creates a new toggle selection command
func NewToggleSelectionCommand(ctx *CommandContext, repoPath string) *ToggleSelectionCommand {
	return &ToggleSelectionCommand{
		ctx:      ctx,
		repoPath: repoPath,
	}
}

// Execute toggles the selection
func (c *ToggleSelectionCommand) Execute() tea.Cmd {
	if c.repoPath != "" {
		c.ctx.State.ToggleRepoSelection(c.repoPath)
	}
	return nil
}

// SelectAllCommand toggles select all repositories
type SelectAllCommand struct {
	ctx        *CommandContext
	totalRepos int
}

// NewSelectAllCommand creates a new select all command
func NewSelectAllCommand(ctx *CommandContext, totalRepos int) *SelectAllCommand {
	return &SelectAllCommand{
		ctx:        ctx,
		totalRepos: totalRepos,
	}
}

// Execute toggles select all
func (c *SelectAllCommand) Execute() tea.Cmd {
	if len(c.ctx.State.SelectedRepos) == c.totalRepos {
		c.ctx.State.ClearSelection()
	} else {
		c.ctx.State.SelectAll()
	}
	return nil
}

// MoveToGroupCommand moves repositories to a group
type MoveToGroupCommand struct {
	ctx        *CommandContext
	repoPaths  []string
	fromGroups map[string]string // repoPath -> fromGroup
	toGroup    string
}

// NewMoveToGroupCommand creates a new move to group command
func NewMoveToGroupCommand(ctx *CommandContext, repoPaths []string, fromGroups map[string]string, toGroup string) *MoveToGroupCommand {
	return &MoveToGroupCommand{
		ctx:        ctx,
		repoPaths:  repoPaths,
		fromGroups: fromGroups,
		toGroup:    toGroup,
	}
}

// Execute moves the repositories
func (c *MoveToGroupCommand) Execute() tea.Cmd {
	movedCount := 0
	for _, repoPath := range c.repoPaths {
		fromGroup := c.fromGroups[repoPath]
		c.ctx.State.MoveRepoToGroup(repoPath, fromGroup, c.toGroup)

		if c.ctx.Bus != nil {
			c.ctx.Bus.Publish(eventbus.RepoMovedEvent{
				RepoPath:  repoPath,
				FromGroup: fromGroup,
				ToGroup:   c.toGroup,
			})
			movedCount++
		}
	}

	if movedCount > 0 {
		c.ctx.State.StatusMessage = fmt.Sprintf("Moved %d repos to '%s'", movedCount, c.toGroup)
		c.ctx.State.ClearSelection()

		if c.ctx.Bus != nil {
			c.ctx.Bus.Publish(eventbus.ConfigChangedEvent{
				Groups: c.ctx.State.GetGroupsMap(),
			})
		}
	}

	return nil
}
