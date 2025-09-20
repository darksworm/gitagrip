package ui

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

    "github.com/charmbracelet/bubbles/v2/help"
    "github.com/charmbracelet/bubbles/v2/textinput"
    tea "github.com/charmbracelet/bubbletea/v2"
    "github.com/charmbracelet/lipgloss/v2"

	"gitagrip/internal/config"
	"gitagrip/internal/domain"
	"gitagrip/internal/eventbus"
	"gitagrip/internal/ui/commands"
	"gitagrip/internal/ui/handlers"
	"gitagrip/internal/ui/input"
	inputtypes "gitagrip/internal/ui/input/types"
	"gitagrip/internal/ui/logic"
	"gitagrip/internal/ui/repositories"
	"gitagrip/internal/ui/state"
	"gitagrip/internal/ui/viewmodels"
	"gitagrip/internal/ui/views"
)

// Special group name for hidden repositories
const HiddenGroupName = "_Hidden"

// Key bindings
// Removed unused key bindings - they're now handled by the input system

// Model represents the UI state
type Model struct {
	bus    eventbus.EventBus
	config *config.Config
	state  *state.AppState // centralized state

	// UI-specific state not in AppState
	width  int
	height int
	help   help.Model
	// Removed: inputMode, textInput, deleteTarget - now handled by input handler
	currentSort logic.SortMode // current sort mode
	// Removed: useNewInput - fully migrated to new input handler
	inPagerMode bool // tracks if we're currently in pager mode

	// Handlers
	searchFilter *logic.SearchFilter          // search and filter handler
	navigator    *logic.Navigator             // navigation and viewport handler
	renderer     *views.Renderer              // view renderer
	eventHandler *handlers.EventHandler       // event processing handler
	viewModel    *viewmodels.ViewModel        // view model for rendering
	store        repositories.RepositoryStore // repository store for data access
	cmdExecutor  *commands.Executor           // command executor
	inputHandler *input.Handler               // input handling
	gitOps       *GitOps                      // git operations handler

	// Program reference for terminal management
	program *tea.Program
}

// NewModel creates a new UI model
func NewModel(bus eventbus.EventBus, cfg *config.Config) *Model {
	appState := state.NewAppState()

	m := &Model{
		bus:    bus,
		config: cfg,
		state:  appState,
		help:   help.New(),
		// Removed: textInput - now handled by input handler
		// Removed: inputMode - now handled by input handler
		currentSort:  logic.SortByName,
		searchFilter: logic.NewSearchFilter(nil), // Will be updated when repos are added
		navigator:    logic.NewNavigator(),
		renderer:     views.NewRenderer(cfg.UISettings.ShowAheadBehind),
		inputHandler: input.New(),
	}

	// Create event handler with reference to updateOrderedLists method
	m.eventHandler = handlers.NewEventHandler(appState, m.updateOrderedLists)

	// Create repository store
	m.store = repositories.NewStateRepositoryStore(appState)

	// Create command executor
	m.cmdExecutor = commands.NewExecutor(appState, bus)

	// Create git operations handler
	m.gitOps = NewGitOps()

	// Create view model with a placeholder text input (actual one is in input handler)
	placeholderTextInput := textinput.New()
	m.viewModel = viewmodels.NewViewModel(appState, cfg, placeholderTextInput)
	m.viewModel.SetHelp(m.help)

	// Initialize groups from config
	for name, repoPaths := range cfg.Groups {
		m.state.AddGroup(name, repoPaths)
	}

	// If we have a saved group order, use it
	if len(cfg.GroupOrder) > 0 {
		// Reset GroupCreationOrder to match the saved order
		m.state.GroupCreationOrder = make([]string, 0, len(cfg.GroupOrder))
		for _, groupName := range cfg.GroupOrder {
			if _, exists := m.state.Groups[groupName]; exists {
				m.state.GroupCreationOrder = append(m.state.GroupCreationOrder, groupName)
			}
		}
		// Add any new groups that aren't in the saved order
		for groupName := range m.state.Groups {
			found := false
			for _, savedName := range cfg.GroupOrder {
				if savedName == groupName {
					found = true
					break
				}
			}
			if !found && groupName != HiddenGroupName {
				m.state.GroupCreationOrder = append(m.state.GroupCreationOrder, groupName)
			}
		}
	}

	// Ensure hidden group is collapsed if it exists
	if _, exists := m.state.Groups[HiddenGroupName]; exists {
		m.state.ExpandedGroups[HiddenGroupName] = false
	}
	m.updateOrderedLists()

	// Update searchFilter with the actual repositories map
	m.searchFilter = logic.NewSearchFilter(m.state.Repositories)

	return m
}

// SetProgram sets the program reference for terminal management
func (m *Model) SetProgram(p *tea.Program) {
	m.program = p
	// Also set it in gitOps
	if m.gitOps != nil {
		m.gitOps.SetProgram(p)
	}
}

// syncNavigatorState updates the navigator with current model state
func (m *Model) syncNavigatorState() {
	ungroupedCount := len(m.getUngroupedRepos())
	m.navigator.UpdateState(
		m.state.SelectedIndex,
		m.state.ViewportOffset,
		m.state.ViewportHeight,
		m.state.ExpandedGroups,
		m.store.GetOrderedGroups(),
		m.state.Groups,
		m.state.Repositories,
		ungroupedCount,
	)
}

// Init returns an initial command
func (m *Model) Init() tea.Cmd {
	// Initialize viewport with reasonable defaults
	m.state.ViewportHeight = 20 // Will be updated on first WindowSizeMsg
	return tea.Tick(time.Millisecond*80, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Update handles messages
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		m.updateViewportHeight()

	case tea.KeyMsg:
		// Handle log/info/help popups first
		if m.state.ShowLog {
			switch msg.String() {
			case "esc", "l", "q":
				m.state.ShowLog = false
				m.state.LogContent = ""
				return m, nil
			}
		}

		if m.state.ShowInfo {
			switch msg.String() {
			case "esc", "i", "q":
				m.state.ShowInfo = false
				m.state.InfoContent = ""
				return m, nil
			}
		}

		// Create context for input handler
		ctx := &input.ModelContext{
			State:       m.state,
			Store:       m.store,
			Navigator:   m.navigator,
			CurrentSort: m.currentSort,
		}

		// Handle input through the new handler
		actions, cmd := m.inputHandler.HandleKey(msg, ctx)

		// Process actions
		cmds := []tea.Cmd{}
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		for _, action := range actions {
			if actionCmd := m.processAction(action); actionCmd != nil {
				cmds = append(cmds, actionCmd)
			}
		}

		// Update text input in view model if in text mode
		if m.inputHandler.TextInput() != nil {
			m.viewModel.UpdateTextInput(*m.inputHandler.TextInput())
		}

		return m, tea.Batch(cmds...)

	default:
		// Handle non-keyboard messages
		if cmd := m.inputHandler.Update(msg); cmd != nil {
			// Update text input in view model if in text mode
			if m.inputHandler.TextInput() != nil {
				m.viewModel.UpdateTextInput(*m.inputHandler.TextInput())
			}
			return m, cmd
		}
		return m.handleNonKeyboardMsg(msg)
	}

	return m, nil
}

// handleEvent processes domain events
// func (m *Model) handleEvent(event eventbus.DomainEvent) (tea.Model, tea.Cmd) {
// 	cmd := m.eventHandler.HandleEvent(event)
// 	// Update searchFilter reference
// 	m.searchFilter = m.eventHandler.GetSearchFilter()
// 	return m, cmd
// }

// View renders the UI
func (m *Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Update view model with current UI state
	m.viewModel.SetDimensions(m.width, m.height)
	// deleteTarget now handled by input handler

	// Use input handler's state
	if m.inputHandler != nil {
		// Convert input.Mode to viewmodels.InputMode
		inputHandlerMode := m.inputHandler.CurrentMode()
		var viewModelMode viewmodels.InputMode
		switch inputHandlerMode {
		case inputtypes.ModeNormal:
			viewModelMode = viewmodels.InputModeNormal
		case inputtypes.ModeSearch:
			viewModelMode = viewmodels.InputModeSearch
		case inputtypes.ModeFilter:
			viewModelMode = viewmodels.InputModeFilter
		case inputtypes.ModeNewGroup:
			viewModelMode = viewmodels.InputModeNewGroup
		case inputtypes.ModeMoveToGroup:
			viewModelMode = viewmodels.InputModeMoveToGroup
		case inputtypes.ModeDeleteConfirm:
			viewModelMode = viewmodels.InputModeDeleteConfirm
		case inputtypes.ModeSort:
			viewModelMode = viewmodels.InputModeSort
		case inputtypes.ModeRenameGroup:
			viewModelMode = viewmodels.InputModeRenameGroup
		}
		m.viewModel.SetInputMode(viewModelMode)

		// Use input handler's text input if available
		if ti := m.inputHandler.TextInput(); ti != nil {
			m.viewModel.UpdateTextInput(*ti)
		}
	}

	m.viewModel.SetUngroupedRepos(m.getUngroupedRepos())

	// Build view state and render
	state := m.viewModel.BuildViewState()
	return m.renderer.Render(state)
}

// updateOrderedLists updates the ordered lists for display
func (m *Model) updateOrderedLists() {
	// Update ordered repos
	m.state.OrderedRepos = make([]string, 0, len(m.store.GetAllRepositories()))
	for path := range m.store.GetAllRepositories() {
		m.state.OrderedRepos = append(m.state.OrderedRepos, path)
	}

	// Sort repositories based on current sort mode
	switch m.currentSort {
	case logic.SortByName:
		sort.Slice(m.state.OrderedRepos, func(i, j int) bool {
			repoI, okI := m.state.Repositories[m.state.OrderedRepos[i]]
			repoJ, okJ := m.state.Repositories[m.state.OrderedRepos[j]]
			if !okI || !okJ {
				return !okI
			}
			return strings.ToLower(repoI.Name) < strings.ToLower(repoJ.Name)
		})

	case logic.SortByStatus:
		sort.Slice(m.state.OrderedRepos, func(i, j int) bool {
			repoI, okI := m.state.Repositories[m.state.OrderedRepos[i]]
			repoJ, okJ := m.state.Repositories[m.state.OrderedRepos[j]]
			if !okI || !okJ {
				return !okI
			}
			// Order: error, dirty, clean
			statusI := logic.GetStatusPriority(repoI)
			statusJ := logic.GetStatusPriority(repoJ)
			if statusI != statusJ {
				return statusI > statusJ // Higher priority first
			}
			return strings.ToLower(repoI.Name) < strings.ToLower(repoJ.Name)
		})

	case logic.SortByBranch:
		sort.Slice(m.state.OrderedRepos, func(i, j int) bool {
			repoI, okI := m.state.Repositories[m.state.OrderedRepos[i]]
			repoJ, okJ := m.state.Repositories[m.state.OrderedRepos[j]]
			if !okI || !okJ {
				return !okI
			}
			branchI := strings.ToLower(repoI.Status.Branch)
			branchJ := strings.ToLower(repoJ.Status.Branch)
			if branchI != branchJ {
				// Put main/master first
				if branchI == "main" || branchI == "master" {
					return true
				}
				if branchJ == "main" || branchJ == "master" {
					return false
				}
				return branchI < branchJ
			}
			return strings.ToLower(repoI.Name) < strings.ToLower(repoJ.Name)
		})

	default:
		// Default to alphabetical by path
		sort.Strings(m.state.OrderedRepos)
	}

	// Update ordered groups - always use creation order
	m.state.OrderedGroups = make([]string, 0, len(m.state.GroupCreationOrder))
	// Only include groups that still exist
	for _, name := range m.state.GroupCreationOrder {
		if _, exists := m.state.Groups[name]; exists {
			if name != HiddenGroupName {
				m.state.OrderedGroups = append(m.state.OrderedGroups, name)
			}
		}
	}

	// Always put hidden group at the end if it exists
	if _, exists := m.state.Groups[HiddenGroupName]; exists {
		m.state.OrderedGroups = append(m.state.OrderedGroups, HiddenGroupName)
	}

	// Update ungrouped repos cache
	m.state.UngroupedRepos = m.getUngroupedRepos()

	// Detect and handle duplicate repository names
	m.updateDuplicateRepoNames()

	// Sort ungrouped repos if needed
	if m.currentSort != logic.SortByName {
		// Apply the same sort to ungrouped repos
		switch m.currentSort {
		case logic.SortByStatus:
			sort.Slice(m.state.UngroupedRepos, func(i, j int) bool {
				repoI, okI := m.store.GetRepository(m.state.UngroupedRepos[i])
				repoJ, okJ := m.store.GetRepository(m.state.UngroupedRepos[j])
				if !okI || !okJ {
					return !okI
				}
				statusI := logic.GetStatusPriority(repoI)
				statusJ := logic.GetStatusPriority(repoJ)
				if statusI != statusJ {
					return statusI > statusJ
				}
				return strings.ToLower(repoI.Name) < strings.ToLower(repoJ.Name)
			})

		case logic.SortByBranch:
			sort.Slice(m.state.UngroupedRepos, func(i, j int) bool {
				repoI, okI := m.store.GetRepository(m.state.UngroupedRepos[i])
				repoJ, okJ := m.store.GetRepository(m.state.UngroupedRepos[j])
				if !okI || !okJ {
					return !okI
				}
				branchI := strings.ToLower(repoI.Status.Branch)
				branchJ := strings.ToLower(repoJ.Status.Branch)
				if branchI != branchJ {
					if branchI == "main" || branchI == "master" {
						return true
					}
					if branchJ == "main" || branchJ == "master" {
						return false
					}
					return branchI < branchJ
				}
				return strings.ToLower(repoI.Name) < strings.ToLower(repoJ.Name)
			})
		}
	}

	// Sort repositories within each group
	for _, group := range m.state.Groups {
		// Create a copy of the repo paths to sort
		sortedRepos := make([]string, len(group.Repos))
		copy(sortedRepos, group.Repos)

		switch m.currentSort {
		case logic.SortByName:
			sort.Slice(sortedRepos, func(i, j int) bool {
				repoI, okI := m.state.Repositories[sortedRepos[i]]
				repoJ, okJ := m.state.Repositories[sortedRepos[j]]
				if !okI || !okJ {
					return !okI
				}
				return strings.ToLower(repoI.Name) < strings.ToLower(repoJ.Name)
			})

		case logic.SortByStatus:
			sort.Slice(sortedRepos, func(i, j int) bool {
				repoI, okI := m.state.Repositories[sortedRepos[i]]
				repoJ, okJ := m.state.Repositories[sortedRepos[j]]
				if !okI || !okJ {
					return !okI
				}
				statusI := logic.GetStatusPriority(repoI)
				statusJ := logic.GetStatusPriority(repoJ)
				if statusI != statusJ {
					return statusI > statusJ // Higher priority first
				}
				return strings.ToLower(repoI.Name) < strings.ToLower(repoJ.Name)
			})

		case logic.SortByBranch:
			sort.Slice(sortedRepos, func(i, j int) bool {
				repoI, okI := m.state.Repositories[sortedRepos[i]]
				repoJ, okJ := m.state.Repositories[sortedRepos[j]]
				if !okI || !okJ {
					return !okI
				}
				branchI := strings.ToLower(repoI.Status.Branch)
				branchJ := strings.ToLower(repoJ.Status.Branch)
				if branchI != branchJ {
					// Put main/master first
					if branchI == "main" || branchI == "master" {
						return true
					}
					if branchJ == "main" || branchJ == "master" {
						return false
					}
					return branchI < branchJ
				}
				return strings.ToLower(repoI.Name) < strings.ToLower(repoJ.Name)
			})
		}

		// Update the group's repo list with sorted order
		group.Repos = sortedRepos
	}
}

// getUngroupedRepos returns repositories not in any group
func (m *Model) getUngroupedRepos() []string {
	grouped := make(map[string]bool)
	for _, group := range m.state.Groups {
		for _, repoPath := range group.Repos {
			grouped[repoPath] = true
		}
	}

	var ungrouped []string
	for _, repoPath := range m.state.OrderedRepos {
		if !grouped[repoPath] {
			ungrouped = append(ungrouped, repoPath)
		}
	}

	return ungrouped
}

// getMaxIndex returns the maximum selectable index
func (m *Model) getMaxIndex() int {
	m.syncNavigatorState()
	return m.navigator.GetMaxIndex(len(m.getUngroupedRepos()))
}

// updateViewportHeight calculates the available height for the repository list
func (m *Model) updateViewportHeight() {
	// Account for title (2 lines), status (2 lines), help (1 line), and padding
	reservedLines := 7

	// Account for input field when active
	// Input mode lines handled by input handler

	m.state.ViewportHeight = m.height - reservedLines
	if m.state.ViewportHeight < 1 {
		m.state.ViewportHeight = 1
	}

	// Ensure viewport offset is still valid
	m.ensureSelectedVisible()
}

// isOnGap returns true if the given index is on a gap (empty line) between groups
func (m *Model) isOnGap(index int) bool {
	currentIndex := 0
	// Check groups first
	for i, groupName := range m.store.GetOrderedGroups() {
		// Group header is not a gap
		if currentIndex == index {
			return false
		}
		currentIndex++

		// Skip group contents if expanded
		if m.store.IsGroupExpanded(groupName) {
			group, _ := m.store.GetGroup(groupName)
			// Repository entries are not gaps
			for j := 0; j < len(group.Repos); j++ {
				if currentIndex == index {
					return false
				}
				currentIndex++
			}
		}

		// Check if there should be a gap after this group
		isLastGroup := i == len(m.store.GetOrderedGroups())-1
		isHiddenGroup := groupName == HiddenGroupName

		// Add gap after group unless it's the hidden group at the end
		if !isHiddenGroup || !isLastGroup {
			if currentIndex == index {
				// This is a gap
				return true
			}
			currentIndex++ // Gap after group
		}

		if currentIndex > index {
			break
		}
	}

	// Check ungrouped section
	ungroupedRepos := m.getUngroupedRepos()
	for range ungroupedRepos {
		if currentIndex == index {
			return false // Ungrouped repos are not gaps
		}
		currentIndex++
	}

	return false
}

// getSelectedGroup returns the group name if a group header is selected
func (m *Model) getSelectedGroup() string {
	currentIndex := 0

	// Check groups first (since they're displayed first now)
	for i, groupName := range m.store.GetOrderedGroups() {
		if currentIndex == m.state.SelectedIndex {
			return groupName // This is the selected group
		}
		currentIndex++

		// Skip group contents if expanded
		if m.store.IsGroupExpanded(groupName) {
			group, _ := m.store.GetGroup(groupName)
			currentIndex += len(group.Repos)
		}

		// Check if there should be a gap after this group
		isLastGroup := i == len(m.store.GetOrderedGroups())-1
		isHiddenGroup := groupName == HiddenGroupName

		// Add gap after group unless it's the hidden group at the end
		if !isHiddenGroup || !isLastGroup {
			currentIndex++ // Gap after group
		}

		if currentIndex > m.state.SelectedIndex {
			break
		}
	}

	return ""
}

// getGroupAtIndex returns the group name for the item at the given index
// This returns the group name whether the index is on a group header or a repo within that group
func (m *Model) getGroupAtIndex(index int) string {
	currentIndex := 0

	// Check groups
	for i, groupName := range m.store.GetOrderedGroups() {
		// Group header
		if currentIndex == index {
			return groupName
		}
		currentIndex++

		// Check repos in group if expanded
		if m.store.IsGroupExpanded(groupName) {
			group, _ := m.store.GetGroup(groupName)
			for range group.Repos {
				if currentIndex == index {
					return groupName // Repo belongs to this group
				}
				currentIndex++
			}
		}

		// Account for gap after group (except hidden group at the end)
		isLastGroup := i == len(m.store.GetOrderedGroups())-1
		isHiddenGroup := groupName == HiddenGroupName
		if !isHiddenGroup || !isLastGroup {
			if currentIndex == index {
				// On a gap - return empty
				return ""
			}
			currentIndex++ // Gap after group
		}

		if currentIndex > index {
			break
		}
	}

	// Check if in ungrouped section
	ungroupedRepos := m.getUngroupedRepos()
	if len(ungroupedRepos) > 0 {
		// All remaining items are ungrouped
		if index >= currentIndex && index < currentIndex+len(ungroupedRepos) {
			return "Ungrouped"
		}
	}

	return ""
}

// getRepoPathAtIndex returns the repository path at the given index
func (m *Model) getRepoPathAtIndex(index int) string {
	currentIndex := 0

	// Check groups first (since they're displayed first now)
	for _, groupName := range m.store.GetOrderedGroups() {
		// Group header itself is not a repo
		if currentIndex == index {
			return "" // This is a group header, not a repo
		}
		currentIndex++

		// Check repos in group if expanded
		if m.store.IsGroupExpanded(groupName) {
			group, _ := m.store.GetGroup(groupName)
			// Use the repos in the same order as displayed (no sorting)
			for _, repoPath := range group.Repos {
				if currentIndex == index {
					return repoPath
				}
				currentIndex++
			}
		}

		// Account for gap after group (except hidden group at the end)
		if groupName != HiddenGroupName || currentIndex < index {
			if currentIndex == index {
				return "" // This is a gap, not a repo
			}
			currentIndex++ // Gap after group
		}

		if currentIndex > index {
			break
		}
	}

	// Then check ungrouped repos
	ungroupedRepos := m.getUngroupedRepos()
	for _, repoPath := range ungroupedRepos {
		if currentIndex == index {
			return repoPath
		}
		currentIndex++
	}

	return ""
}

// buildRepoInfo builds detailed information about a repository
func (m *Model) buildRepoInfo(repo *domain.Repository) string {
	var info strings.Builder

	// Repository name and path
	info.WriteString(lipgloss.NewStyle().Bold(true).Render(repo.Name))
	info.WriteString("\n\n")

	// Path
	info.WriteString(fmt.Sprintf("Path: %s\n", repo.Path))

	// Group
	groupName := "Ungrouped"
	for _, group := range m.state.Groups {
		for _, path := range group.Repos {
			if path == repo.Path {
				groupName = group.Name
				break
			}
		}
	}
	info.WriteString(fmt.Sprintf("Group: %s\n\n", groupName))

    // Status information
    info.WriteString(lipgloss.NewStyle().Bold(true).Render("Status:"))
    info.WriteString("\n")
    // Colorize branch like in list view
    branchColor := views.GetBranchColor(repo.Status.Branch)
    branchStyled := lipgloss.NewStyle().Foreground(lipgloss.Color(branchColor))
    // Make main/master bold for emphasis
    if repo.Status.Branch == "main" || repo.Status.Branch == "master" {
        branchStyled = branchStyled.Bold(true)
    }
    info.WriteString("  Branch: ")
    info.WriteString(branchStyled.Render(repo.Status.Branch))
    info.WriteString("\n")

	// Clean/Dirty status
    if repo.Status.IsDirty {
        // Yellow for changes
        info.WriteString("  State: ")
        info.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("Dirty (uncommitted changes)"))
        info.WriteString("\n")
    } else if repo.Status.HasUntracked {
        info.WriteString("  State: ")
        info.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("Has untracked files"))
        info.WriteString("\n")
    } else {
        // Green for clean
        info.WriteString("  State: ")
        info.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("78")).Render("Clean"))
        info.WriteString("\n")
    }

	// Ahead/Behind
	if repo.Status.AheadCount > 0 || repo.Status.BehindCount > 0 {
		info.WriteString(fmt.Sprintf("  Ahead: %d commits\n", repo.Status.AheadCount))
		info.WriteString(fmt.Sprintf("  Behind: %d commits\n", repo.Status.BehindCount))
	}

	// Error
	if repo.Status.Error != "" {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
		info.WriteString(fmt.Sprintf("  Error: %s\n", errorStyle.Render(repo.Status.Error)))
	}

	// Command logs
	if len(repo.CommandLogs) > 0 {
		info.WriteString("\n")
		info.WriteString(lipgloss.NewStyle().Bold(true).Render("Command History:"))
		info.WriteString("\n")

		// Show last 10 logs in reverse order (most recent first)
		start := len(repo.CommandLogs) - 10
		if start < 0 {
			start = 0
		}

		for i := len(repo.CommandLogs) - 1; i >= start; i-- {
			log := repo.CommandLogs[i]

			// Format timestamp and command on same line
			info.WriteString(fmt.Sprintf("\n[%s] ", log.Timestamp))

			// Command name with appropriate styling
			if !log.Success {
				cmdStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
				info.WriteString(cmdStyle.Render(log.Command))
			} else {
				info.WriteString(log.Command)
			}

			// Duration
			info.WriteString(fmt.Sprintf(" (%dms)\n", log.Duration))

			// Output/Error
			if !log.Success {
				if log.Output != "" {
					// Show the actual git output which contains the real error message
					errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
					output := strings.TrimSpace(log.Output)
					// Replace any error: prefix to avoid duplication
					output = strings.TrimPrefix(output, "error: ")
					info.WriteString("  Output: ")
					info.WriteString(errorStyle.Render(output))
					info.WriteString("\n")
				} else if log.Error != "" {
					// Fallback to error field if no output
					errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
					info.WriteString("  Error: ")
					info.WriteString(errorStyle.Render(log.Error))
					info.WriteString("\n")
				}
			} else if log.Output != "" && len(log.Output) < 200 {
				// Show short outputs for successful commands
				info.WriteString("  Output: ")
				info.WriteString(strings.TrimSpace(log.Output))
				info.WriteString("\n")
			}
		}
	}

	info.WriteString("\n")
	info.WriteString("Press ESC or 'i' to close")

	return info.String()
}

// buildRepoLogsContent generates a plain text log report for the repository suitable for pager display
func (m *Model) buildRepoLogsContent(repo *domain.Repository) string {
	var b strings.Builder
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99")).Render("Repository Logs")
	b.WriteString(title)
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("Name: %s\n", repo.Name))
	b.WriteString(fmt.Sprintf("Path: %s\n", repo.Path))
	b.WriteString("\n")

	if len(repo.CommandLogs) == 0 {
		b.WriteString("No command logs yet. Try fetch (f) or pull (p).\n")
		return b.String()
	}

	// Show all available logs, most recent first
	for i := len(repo.CommandLogs) - 1; i >= 0; i-- {
		entry := repo.CommandLogs[i]
		status := "OK"
		if !entry.Success {
			status = "FAIL"
		}
		b.WriteString(fmt.Sprintf("[%s] %s (%dms) — %s\n", entry.Timestamp, entry.Command, entry.Duration, status))
		if entry.Output != "" {
			b.WriteString("Output:\n")
			b.WriteString(strings.TrimSpace(entry.Output))
			b.WriteString("\n")
		}
		if entry.Error != "" {
			b.WriteString("Error:\n")
			b.WriteString(strings.TrimSpace(entry.Error))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("Press q to close")
	return b.String()
}

// countVisibleItems counts how many items are visible with current filter
// getCurrentIndexForGroup finds the current display index for a group
func (m *Model) getCurrentIndexForGroup(groupName string) int {
	m.syncNavigatorState()
	return m.navigator.GetCurrentIndexForGroup(groupName)
}

// fetchGitLog returns a command that fetches git log for a repository
func (m *Model) fetchGitLog(repoPath string) tea.Cmd {
	return func() tea.Msg {
		content, err := m.gitOps.FetchGitLog(repoPath)
		if err != nil {
			return gitLogMsg{
				repoPath: repoPath,
				err:      err,
			}
		}

		return gitLogMsg{
			repoPath: repoPath,
			content:  content,
		}
	}
}

// fetchGitDiff returns a command that fetches git diff for a repository
func (m *Model) fetchGitDiff(repoPath string) tea.Cmd {
	return func() tea.Msg {
		content, err := m.gitOps.FetchGitDiff(repoPath)
		if err != nil {
			return gitDiffMsg{
				repoPath: repoPath,
				err:      err,
			}
		}

		return gitDiffMsg{
			repoPath: repoPath,
			content:  content,
		}
	}
}

// fetchGitLogPager returns a command that shows git log using ov pager
func (m *Model) fetchGitLogPager(repoPath string) tea.Cmd {
	return func() tea.Msg {
		// Send pause message to stop rendering
		m.program.Send(pauseRenderingMsg{})

		err := m.gitOps.ShowGitLogInPager(repoPath)

		// Send resume message to restart rendering
		m.program.Send(resumeRenderingMsg{})

		return gitLogPagerMsg{
			repoPath: repoPath,
			err:      err,
		}
	}
}

// fetchGitDiffPager returns a command that shows git diff using ov pager
func (m *Model) fetchGitDiffPager(repoPath string) tea.Cmd {
	return func() tea.Msg {
		// Send pause message to stop rendering
		m.program.Send(pauseRenderingMsg{})

		err := m.gitOps.ShowGitDiffInPager(repoPath)

		// Send resume message to restart rendering
		m.program.Send(resumeRenderingMsg{})

		return gitDiffPagerMsg{
			repoPath: repoPath,
			err:      err,
		}
	}
}

// fetchHelpPager returns a command that shows help using ov pager
func (m *Model) fetchHelpPager(helpContent string) tea.Cmd {
	return func() tea.Msg {
		// Send pause message to stop rendering
		m.program.Send(pauseRenderingMsg{})

		err := m.gitOps.ShowHelpInPager(helpContent)

		// Send resume message to restart rendering
		m.program.Send(resumeRenderingMsg{})

		return helpPagerMsg{
			err: err,
		}
	}
}

// fetchLazygit returns a command that runs lazygit for the given repo, pausing and resuming rendering
func (m *Model) fetchLazygit(repoPath string) tea.Cmd {
	return func() tea.Msg {
		// Pause rendering while external TUI is active
		m.program.Send(pauseRenderingMsg{})

		err := m.gitOps.RunLazygit(repoPath)

		// Resume rendering afterwards
		m.program.Send(resumeRenderingMsg{})

		return lazygitExitMsg{repoPath: repoPath, err: err}
	}
}

// processAction processes an action from the input handler
func (m *Model) processAction(action inputtypes.Action) tea.Cmd {
	log.Printf("processAction: %T", action)
	switch a := action.(type) {
	case inputtypes.NavigateAction:
		switch a.Direction {
		case "up":
			if m.state.SelectedIndex > 0 {
				m.state.SelectedIndex--
				// Skip gaps when moving up
				for m.state.SelectedIndex > 0 && m.isOnGap(m.state.SelectedIndex) {
					m.state.SelectedIndex--
				}
				m.ensureSelectedVisible()
			}
		case "down":
			maxIndex := m.getMaxIndex()
			if m.state.SelectedIndex < maxIndex {
				m.state.SelectedIndex++
				// Skip gaps when moving down
				for m.state.SelectedIndex < maxIndex && m.isOnGap(m.state.SelectedIndex) {
					m.state.SelectedIndex++
				}
				m.ensureSelectedVisible()
			}
		case "left":
			// Collapse group
			if groupName := m.getSelectedGroup(); groupName != "" {
				m.state.ExpandedGroups[groupName] = false
				m.ensureSelectedVisible()
			}
		case "right":
			// Expand group
			if groupName := m.getSelectedGroup(); groupName != "" {
				m.state.ExpandedGroups[groupName] = true
			}
		case "home":
			m.state.SelectedIndex = 0
			m.ensureSelectedVisible()
		case "end":
			m.state.SelectedIndex = m.getMaxIndex()
			m.ensureSelectedVisible()
		case "pageup":
			m.pageUp()
		case "pagedown":
			m.pageDown()
		}

	case inputtypes.SelectAction:
		if a.Index < 0 {
			// Toggle selection at current index
			if repoPath := m.getRepoPathAtIndex(m.state.SelectedIndex); repoPath != "" {
				return m.cmdExecutor.ExecuteToggleSelection(repoPath)
			}
		} else {
			// Toggle selection at specific index
			if repoPath := m.getRepoPathAtIndex(a.Index); repoPath != "" {
				return m.cmdExecutor.ExecuteToggleSelection(repoPath)
			}
		}

	case inputtypes.SelectGroupAction:
		// Toggle selection for all repos in the group
		if group, ok := m.store.GetGroup(a.GroupName); ok {
			// Check if all repos in group are already selected
			allSelected := true
			for _, repoPath := range group.Repos {
				if !m.state.SelectedRepos[repoPath] {
					allSelected = false
					break
				}
			}

			// Toggle selection for all repos in group
			for _, repoPath := range group.Repos {
				if allSelected {
					// Deselect all
					delete(m.state.SelectedRepos, repoPath)
				} else {
					// Select all
					m.state.SelectedRepos[repoPath] = true
				}
			}

			// Update status message
			if allSelected {
				m.state.StatusMessage = fmt.Sprintf("Deselected all repos in '%s'", a.GroupName)
			} else {
				m.state.StatusMessage = fmt.Sprintf("Selected all repos in '%s'", a.GroupName)
			}
		}

	case inputtypes.SearchNavigateAction:
		log.Printf("SearchNavigateAction: direction=%s, query=%s, matches=%v, currentSearchIndex=%d",
			a.Direction, m.state.SearchQuery, m.state.SearchMatches, m.state.SearchIndex)

		if m.state.SearchQuery != "" && len(m.state.SearchMatches) > 0 {
			oldIndex := m.state.SearchIndex

			switch a.Direction {
			case "next":
				// Navigate to next search result
				m.state.SearchIndex = (m.state.SearchIndex + 1) % len(m.state.SearchMatches)
			case "prev":
				// Navigate to previous search result
				m.state.SearchIndex--
				if m.state.SearchIndex < 0 {
					m.state.SearchIndex = len(m.state.SearchMatches) - 1
				}
			}

			// Jump to the match
			if m.state.SearchIndex >= 0 && m.state.SearchIndex < len(m.state.SearchMatches) {
				m.state.SelectedIndex = m.state.SearchMatches[m.state.SearchIndex]
				m.ensureSelectedVisible()
				log.Printf("SearchNavigate: moved from match %d to %d, jumped to index %d",
					oldIndex, m.state.SearchIndex, m.state.SelectedIndex)
			}
		} else {
			log.Printf("SearchNavigate: no action - query empty or no matches")
		}

	case inputtypes.SelectAllAction:
		totalRepos := m.countVisibleRepos()
		return m.cmdExecutor.ExecuteSelectAll(totalRepos)

	case inputtypes.DeselectAllAction:
		m.state.ClearSelection()

	case inputtypes.RefreshAction:
		if a.All {
			// Full scan
			return m.cmdExecutor.ExecuteFullScan(m.config.BaseDir)
		} else {
			// Refresh status
			var repoPaths []string
			if m.store.GetSelectionCount() > 0 {
				// Refresh selected repos
				for path := range m.store.GetSelectedRepositories() {
					repoPaths = append(repoPaths, path)
				}
			} else {
				// Check if we're on a group header
				groupName := m.getSelectedGroup()
				if groupName != "" && groupName != "Ungrouped" {
					// Refresh all repos in the group
					if group, ok := m.store.GetGroup(groupName); ok {
						repoPaths = append(repoPaths, group.Repos...)
						m.state.StatusMessage = fmt.Sprintf("Refreshing all repos in '%s'", groupName)
					}
				} else {
					// Refresh current repository
					if repoPath := m.getRepoPathAtIndex(m.state.SelectedIndex); repoPath != "" {
						repoPaths = []string{repoPath}
					}
				}
			}
			return m.cmdExecutor.ExecuteRefresh(repoPaths)
		}

	case inputtypes.FetchAction:
		var repoPaths []string
		if m.store.GetSelectionCount() > 0 {
			// Fetch selected repos
			for path := range m.store.GetSelectedRepositories() {
				repoPaths = append(repoPaths, path)
			}
		} else {
			// Check if we're on a group header
			groupName := m.getSelectedGroup()
			if groupName != "" && groupName != "Ungrouped" {
				// Fetch all repos in the group
				if group, ok := m.store.GetGroup(groupName); ok {
					repoPaths = append(repoPaths, group.Repos...)
					m.state.StatusMessage = fmt.Sprintf("Fetching all repos in '%s'", groupName)
				}
			} else {
				// Fetch current repository
				if repoPath := m.getRepoPathAtIndex(m.state.SelectedIndex); repoPath != "" {
					repoPaths = []string{repoPath}
				}
			}
		}
		return m.cmdExecutor.ExecuteFetch(repoPaths)

	case inputtypes.PullAction:
		var repoPaths []string
		if m.store.GetSelectionCount() > 0 {
			// Pull selected repos
			for path := range m.store.GetSelectedRepositories() {
				repoPaths = append(repoPaths, path)
			}
		} else {
			// Check if we're on a group header
			groupName := m.getSelectedGroup()
			if groupName != "" && groupName != "Ungrouped" {
				// Pull all repos in the group
				if group, ok := m.store.GetGroup(groupName); ok {
					repoPaths = append(repoPaths, group.Repos...)
					m.state.StatusMessage = fmt.Sprintf("Pulling all repos in '%s'", groupName)
				}
			} else {
				// Pull current repository
				if repoPath := m.getRepoPathAtIndex(m.state.SelectedIndex); repoPath != "" {
					repoPaths = []string{repoPath}
				}
			}
		}
		return m.cmdExecutor.ExecutePull(repoPaths)

	case inputtypes.OpenLogAction:
		// Show git log for current repo
		if repoPath := m.getRepoPathAtIndex(m.state.SelectedIndex); repoPath != "" {
			// Try pager first if available, fall back to popup
			if m.gitOps.IsOvAvailable() {
				return m.fetchGitLogPager(repoPath)
			} else {
				m.state.ShowLog = true
				return m.fetchGitLog(repoPath)
			}
		}

	case inputtypes.OpenDiffAction:
		// Show git diff for current repo
		if repoPath := m.getRepoPathAtIndex(m.state.SelectedIndex); repoPath != "" {
			// First check if there's any diff content
			content, err := m.gitOps.FetchGitDiff(repoPath)
			if err != nil {
				// Show error as status message
				m.state.StatusMessage = fmt.Sprintf("Error fetching diff: %v", err)
				return nil
			}

			// If no changes, show status message instead of opening pager/popup
			if content == "" {
				m.state.StatusMessage = "No uncommitted changes"
				// Clear the status message after 3 seconds
				return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
					return clearStatusMsg{}
				})
			}

			// There are changes, proceed with pager or popup
			if m.gitOps.IsOvAvailable() {
				return m.fetchGitDiffPager(repoPath)
			} else {
				m.state.ShowLog = true
				return m.fetchGitDiff(repoPath)
			}
		}

	case inputtypes.ToggleInfoAction:
		m.state.ShowInfo = !m.state.ShowInfo
		if m.state.ShowInfo {
			// Build info content for current repo
			repoPath := m.getRepoPathAtIndex(m.state.SelectedIndex)
			log.Printf("ToggleInfoAction: ShowInfo=%v, repoPath=%s", m.state.ShowInfo, repoPath)
			if repoPath != "" {
				if repo, ok := m.state.Repositories[repoPath]; ok {
					m.state.InfoContent = m.buildRepoInfo(repo)
					log.Printf("Built info content, length=%d", len(m.state.InfoContent))
				} else {
					log.Printf("Repository not found for path: %s", repoPath)
				}
			} else {
				log.Printf("No repo path at index %d", m.state.SelectedIndex)
			}
		} else {
			m.state.InfoContent = ""
		}

	case inputtypes.ToggleHelpAction:
		// Generate plain text help content for pager
		helpContent := m.renderer.RenderHelpContentPlain()
		// Show help using ov pager
		return m.fetchHelpPager(helpContent)

	case inputtypes.OpenRepoLogsAction:
		// Build logs content for the current repo and show in pager
		if repoPath := m.getRepoPathAtIndex(m.state.SelectedIndex); repoPath != "" {
			if repo, ok := m.state.Repositories[repoPath]; ok {
				content := m.buildRepoLogsContent(repo)
				return m.fetchHelpPager(content)
			}
		}
		return nil

	case inputtypes.OpenLazygitAction:
		// Open lazygit for current repo (if available)
		if repoPath := m.getRepoPathAtIndex(m.state.SelectedIndex); repoPath != "" {
			if m.gitOps.IsLazygitAvailable() {
				return m.fetchLazygit(repoPath)
			}
			// If not available, show a helpful status message with guidance
			m.state.StatusMessage = "lazygit not found. Install with: brew install lazygit (or see github.com/jesseduffield/lazygit). Tip: press H for git log."
			return tea.Tick(5*time.Second, func(t time.Time) tea.Msg { return clearStatusMsg{} })
		}
		return nil

	case inputtypes.ExpandAllGroupsAction:
		// Expand all groups (except hidden)
		for groupName := range m.state.Groups {
			if groupName != HiddenGroupName {
				m.state.ExpandedGroups[groupName] = true
			}
		}
		m.ensureSelectedVisible()

	case inputtypes.ToggleGroupAction:
		// First try to get the group if we're on a group header
		groupName := m.getSelectedGroup()
		wasOnGroupHeader := groupName != ""

		if groupName == "" {
			// If not on a group header, check if we're on a repo within a group
			groupName = m.getGroupAtIndex(m.state.SelectedIndex)
		}

		if groupName != "" && groupName != "Ungrouped" {
			// Check if we're closing the group while inside it
			isClosing := m.state.ExpandedGroups[groupName]
			wasInsideGroup := !wasOnGroupHeader && isClosing

			// Toggle the group expansion state
			m.state.ExpandedGroups[groupName] = !m.state.ExpandedGroups[groupName]

			// If we just collapsed a group and we were inside it, move selection to the group header
			if wasInsideGroup {
				// Find the group header position
				newIndex := m.getCurrentIndexForGroup(groupName)
				if newIndex >= 0 {
					m.state.SelectedIndex = newIndex
				}
			}
			m.ensureSelectedVisible()
		}

	case inputtypes.CreateGroupAction:
		log.Printf("processAction: CreateGroupAction received with name: %s", a.Name)
		// Create the new group
		if m.bus != nil {
			m.bus.Publish(eventbus.GroupAddedEvent{
				Name: a.Name,
			})
		}

		// Move selected repos to the new group
		movedCount := 0
		for repoPath := range m.store.GetSelectedRepositories() {
			// Find current group (if any)
			var fromGroup string
			for _, group := range m.state.Groups {
				for _, path := range group.Repos {
					if path == repoPath {
						fromGroup = group.Name
						break
					}
				}
			}

			// Publish move event
			m.bus.Publish(eventbus.RepoMovedEvent{
				RepoPath:  repoPath,
				FromGroup: fromGroup,
				ToGroup:   a.Name,
			})
			movedCount++
		}

		// Clear selection
		m.state.ClearSelection()

		// Set cursor to the new group
		for _, groupName := range m.state.OrderedGroups {
			if groupName == a.Name {
				m.state.SelectedIndex = m.getCurrentIndexForGroup(groupName)
				m.ensureSelectedVisible()
				break
			}
		}

		// Status message
		if movedCount > 0 {
			m.state.StatusMessage = fmt.Sprintf("Created group '%s' with %d repositories", a.Name, movedCount)
		} else {
			m.state.StatusMessage = fmt.Sprintf("Created empty group '%s'", a.Name)
		}

		// Publish config changed event
		if m.bus != nil {
			m.bus.Publish(eventbus.ConfigChangedEvent{
				Groups:     m.getGroupsMap(),
				GroupOrder: m.getGroupOrder(),
			})
		}

	case inputtypes.MoveToGroupAction:
		var repoPaths []string
		fromGroups := make(map[string]string)

		if m.store.GetSelectionCount() > 0 {
			// Move selected repos
			for path := range m.store.GetSelectedRepositories() {
				repoPaths = append(repoPaths, path)
				// Find current group
				for gName, group := range m.state.Groups {
					for _, p := range group.Repos {
						if p == path {
							fromGroups[path] = gName
							break
						}
					}
				}
			}
		} else {
			// Move current repo
			if repoPath := m.getRepoPathAtIndex(m.state.SelectedIndex); repoPath != "" {
				repoPaths = []string{repoPath}
				// Find current group
				for gName, group := range m.state.Groups {
					for _, p := range group.Repos {
						if p == repoPath {
							fromGroups[repoPath] = gName
							break
						}
					}
				}
			}
		}

		return m.cmdExecutor.ExecuteMoveToGroup(repoPaths, fromGroups, a.GroupName)

	case inputtypes.RenameGroupAction:
		if a.OldName != "" && a.NewName != "" && a.OldName != a.NewName {
			// Check if new name already exists
			if _, exists := m.state.Groups[a.NewName]; exists {
				m.state.StatusMessage = fmt.Sprintf("Group '%s' already exists", a.NewName)
				return nil
			}

			// Get the old group
			oldGroup, exists := m.state.Groups[a.OldName]
			if !exists {
				m.state.StatusMessage = fmt.Sprintf("Group '%s' not found", a.OldName)
				return nil
			}

			// Create new group with same repos
			m.state.Groups[a.NewName] = &domain.Group{
				Name:  a.NewName,
				Repos: oldGroup.Repos,
			}

			// Copy expansion state
			if expanded, ok := m.state.ExpandedGroups[a.OldName]; ok {
				m.state.ExpandedGroups[a.NewName] = expanded
			}

			// Delete old group
			delete(m.state.Groups, a.OldName)
			delete(m.state.ExpandedGroups, a.OldName)

			// Update ordered groups
			for i, groupName := range m.state.OrderedGroups {
				if groupName == a.OldName {
					m.state.OrderedGroups[i] = a.NewName
					break
				}
			}

			// Update GroupCreationOrder
			for i, groupName := range m.state.GroupCreationOrder {
				if groupName == a.OldName {
					m.state.GroupCreationOrder[i] = a.NewName
					break
				}
			}

			m.state.StatusMessage = fmt.Sprintf("Renamed group '%s' to '%s'", a.OldName, a.NewName)

			// Save config
			if m.bus != nil {
				m.bus.Publish(eventbus.ConfigChangedEvent{
					Groups:     m.getGroupsMap(),
					GroupOrder: m.getGroupOrder(),
				})
			}
		}

	case inputtypes.DeleteGroupAction:
		if a.GroupName != "" && a.GroupName != "Ungrouped" {
			// Remove the group
			delete(m.state.Groups, a.GroupName)

			// Remove from ordered groups
			newOrderedGroups := []string{}
			for _, g := range m.state.OrderedGroups {
				if g != a.GroupName {
					newOrderedGroups = append(newOrderedGroups, g)
				}
			}
			m.state.OrderedGroups = newOrderedGroups

			m.state.StatusMessage = fmt.Sprintf("Deleted group '%s'", a.GroupName)

			// Publish config changed event
			if m.bus != nil {
				m.bus.Publish(eventbus.ConfigChangedEvent{
					Groups:     m.getGroupsMap(),
					GroupOrder: m.getGroupOrder(),
				})
			}
		}

	case inputtypes.SubmitTextAction:
		// Handle text submission based on mode
		switch a.Mode {
		case inputtypes.ModeSearch:
			m.state.SearchQuery = a.Text
			m.performSearch()
			// Jump to first match if any
			if len(m.state.SearchMatches) > 0 {
				m.state.SearchIndex = 0
				m.state.SelectedIndex = m.state.SearchMatches[0]
				m.ensureSelectedVisible()
			}

		case inputtypes.ModeFilter:
			m.state.FilterQuery = a.Text
			m.state.IsFiltered = a.Text != ""
			// TODO: Implement filter
			// if a.Text == "" {
			// 	m.searchFilter.ClearFilter()
			// 	m.state.IsFiltered = false
			// } else {
			// 	m.searchFilter.Filter(a.Text)
			// 	m.state.IsFiltered = true
			// }
			m.updateOrderedLists()
			m.ensureSelectedVisible()

		case inputtypes.ModeSort:
			m.handleSortInput(a.Text)
		case inputtypes.ModeNewGroup:
			log.Printf("New group input: %s", a.Text)
			groupName := strings.TrimSpace(a.Text)
			if groupName != "" {
				// Create the new group
				if m.bus != nil {
					m.bus.Publish(eventbus.GroupAddedEvent{
						Name: groupName,
					})
				}

				// Move selected repos to the new group
				movedCount := 0
				for repoPath := range m.store.GetSelectedRepositories() {
					// Find current group (if any)
					var fromGroup string
					for _, group := range m.state.Groups {
						for _, path := range group.Repos {
							if path == repoPath {
								fromGroup = group.Name
								break
							}
						}
					}

					// Publish move event
					m.bus.Publish(eventbus.RepoMovedEvent{
						RepoPath:  repoPath,
						FromGroup: fromGroup,
						ToGroup:   groupName,
					})
					movedCount++
				}

				// Clear selection
				if movedCount > 0 {
					m.state.ClearSelection()
				}

				// Update status message
				m.state.StatusMessage = fmt.Sprintf("Created group '%s' with %d repo(s)", groupName, movedCount)
			}

		case inputtypes.ModeMoveToGroup:
			// TODO: Implement move to group
		}

	case inputtypes.CancelTextAction:
		// Clear any partial input
		m.state.SearchQuery = ""
		m.state.SearchMatches = nil
		m.state.SearchIndex = 0
		m.state.FilterQuery = ""

	case inputtypes.UpdateTextAction:
		// Update text in view model is handled in the main Update method

	case inputtypes.SortByAction:
		m.handleSortInput(a.Criteria)

	case inputtypes.UpdateSortIndexAction:
		m.state.SortOptionIndex = a.Index

	case inputtypes.HideAction:
		// Ensure hidden group exists
		if _, exists := m.state.Groups[HiddenGroupName]; !exists {
			m.state.AddGroup(HiddenGroupName, []string{})
			// Keep hidden group collapsed by default
			m.state.ExpandedGroups[HiddenGroupName] = false
			// Add to end of ordered groups
			m.state.OrderedGroups = append(m.state.OrderedGroups, HiddenGroupName)
			// Publish group added event
			if m.bus != nil {
				m.bus.Publish(eventbus.GroupAddedEvent{Name: HiddenGroupName})
			}
		}

		// Move repos to hidden group
		var repoPaths []string
		fromGroups := make(map[string]string)

		if m.store.GetSelectionCount() > 0 {
			// Hide selected repos
			for path := range m.store.GetSelectedRepositories() {
				repoPaths = append(repoPaths, path)
				// Find current group
				for gName, group := range m.state.Groups {
					for _, p := range group.Repos {
						if p == path {
							fromGroups[path] = gName
							break
						}
					}
				}
			}
			m.state.ClearSelection()
		} else {
			// Hide current repo
			if repoPath := m.getRepoPathAtIndex(m.state.SelectedIndex); repoPath != "" {
				repoPaths = []string{repoPath}
				// Find current group
				for gName, group := range m.state.Groups {
					for _, p := range group.Repos {
						if p == repoPath {
							fromGroups[repoPath] = gName
							break
						}
					}
				}
			}
		}

		// Move repos to hidden group
		if len(repoPaths) > 0 {
			return m.cmdExecutor.ExecuteMoveToGroup(repoPaths, fromGroups, HiddenGroupName)
		}

	case inputtypes.MoveGroupUpAction:
		// Get the group at current position
		groupName := m.getGroupAtIndex(m.state.SelectedIndex)
		if groupName != "" && groupName != "Ungrouped" && groupName != HiddenGroupName {
			// Find current position in OrderedGroups
			currentIdx := -1
			for i, g := range m.state.OrderedGroups {
				if g == groupName {
					currentIdx = i
					break
				}
			}

			// Move up if possible
			if currentIdx > 0 {
				// Swap with previous group
				m.state.OrderedGroups[currentIdx], m.state.OrderedGroups[currentIdx-1] =
					m.state.OrderedGroups[currentIdx-1], m.state.OrderedGroups[currentIdx]

				// Update GroupCreationOrder to match
				m.updateGroupCreationOrder()

				// Update display
				m.updateOrderedLists()

				// Move cursor to follow the group
				newIdx := m.getCurrentIndexForGroup(groupName)
				if newIdx >= 0 {
					m.state.SelectedIndex = newIdx
					m.ensureSelectedVisible()
				}

				// Save config
				if m.bus != nil {
					m.bus.Publish(eventbus.ConfigChangedEvent{
						Groups: m.getGroupsMap(),
					})
				}
			}
		}

	case inputtypes.MoveGroupDownAction:
		// Get the group at current position
		groupName := m.getGroupAtIndex(m.state.SelectedIndex)
		if groupName != "" && groupName != "Ungrouped" && groupName != HiddenGroupName {
			// Find current position in OrderedGroups
			currentIdx := -1
			for i, g := range m.state.OrderedGroups {
				if g == groupName {
					currentIdx = i
					break
				}
			}

			// Move down if possible (not counting hidden group at the end)
			maxIdx := len(m.state.OrderedGroups) - 1
			if _, hasHidden := m.state.Groups[HiddenGroupName]; hasHidden {
				maxIdx-- // Don't count hidden group
			}

			if currentIdx >= 0 && currentIdx < maxIdx {
				// Swap with next group
				m.state.OrderedGroups[currentIdx], m.state.OrderedGroups[currentIdx+1] =
					m.state.OrderedGroups[currentIdx+1], m.state.OrderedGroups[currentIdx]

				// Update GroupCreationOrder to match
				m.updateGroupCreationOrder()

				// Update display
				m.updateOrderedLists()

				// Move cursor to follow the group
				newIdx := m.getCurrentIndexForGroup(groupName)
				if newIdx >= 0 {
					m.state.SelectedIndex = newIdx
					m.ensureSelectedVisible()
				}

				// Save config
				if m.bus != nil {
					m.bus.Publish(eventbus.ConfigChangedEvent{
						Groups: m.getGroupsMap(),
					})
				}
			}
		}

	case inputtypes.QuitAction:
		if !a.Force && m.config.UISettings.AutosaveOnExit && m.bus != nil {
			m.bus.Publish(eventbus.ConfigChangedEvent{
				Groups:     m.getGroupsMap(),
				GroupOrder: m.getGroupOrder(),
			})
		}
		return tea.Quit
	}

	return nil
}

// handleNonKeyboardMsg handles non-keyboard messages
func (m *Model) handleNonKeyboardMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	log.Printf("handleNonKeyboardMsg: %T", msg)
	switch msg := msg.(type) {
	case EventMsg:
		// Process domain events
		cmd := m.eventHandler.HandleEvent(msg.Event)
		return m, cmd

	case tickMsg:
		// Don't clear loading state automatically - let scan completion handle it
		// Don't continue tick loop if we're in pager mode
		if m.inPagerMode {
			return m, nil
		}
		return m, tick()

	case gitLogMsg:
		if msg.err != nil {
			// Log error and mark repository error state instead of surfacing at top
			log.Printf("Error fetching log for %s: %v", msg.repoPath, msg.err)
			if repo, ok := m.state.Repositories[msg.repoPath]; ok {
				repo.HasError = true
				repo.LastError = fmt.Sprintf("log: %v", msg.err)
			}
			// Keep popup content minimal or empty; do not set StatusMessage here
			m.state.LogContent = ""
		} else {
			m.state.LogContent = fmt.Sprintf("Git log for %s:\n\n%s", msg.repoPath, msg.content)
		}
		return m, nil

	case gitDiffMsg:
		if msg.err != nil {
			// Log error and mark repository error state; do not show in status bar
			log.Printf("Error fetching diff for %s: %v", msg.repoPath, msg.err)
			if repo, ok := m.state.Repositories[msg.repoPath]; ok {
				repo.HasError = true
				repo.LastError = fmt.Sprintf("diff: %v", msg.err)
			}
		} else {
			if msg.content == "" {
				// No changes - show status message instead of opening popup
				m.state.StatusMessage = "No uncommitted changes"
			} else {
				// Show diff in popup
				m.state.LogContent = fmt.Sprintf("Git diff for %s:\n\n%s", msg.repoPath, msg.content)
				m.state.ShowLog = true
			}
		}
		return m, nil

	case gitLogPagerMsg:
		if msg.err != nil {
			// Pager failed, log and fall back to popup silently
			log.Printf("Log pager failed for %s: %v — falling back to popup", msg.repoPath, msg.err)
			return m, m.fetchGitLog(msg.repoPath)
		}
		// Pager succeeded, RestoreTerminal() should have restored the screen
		return m, nil

	case gitDiffPagerMsg:
		if msg.err != nil {
			// Pager failed, log and fall back to popup silently
			log.Printf("Diff pager failed for %s: %v — falling back to popup", msg.repoPath, msg.err)
			return m, m.fetchGitDiff(msg.repoPath)
		}
		// Pager succeeded, RestoreTerminal() should have restored the screen
		return m, nil

	case helpPagerMsg:
		if msg.err != nil {
			// Pager failed: log only; do not surface in status bar
			log.Printf("Help pager failed: %v", msg.err)
		}
		// Pager succeeded, RestoreTerminal() should have restored the screen
		return m, nil

	case lazygitExitMsg:
		if msg.err != nil {
			m.state.StatusMessage = fmt.Sprintf("Failed to run lazygit: %v", msg.err)
			return m, tea.Tick(3*time.Second, func(t time.Time) tea.Msg { return clearStatusMsg{} })
		}
		return m, nil

	case pauseRenderingMsg:
		// Signal that rendering should be paused for external pager
		m.inPagerMode = true
		return m, nil

	case resumeRenderingMsg:
		// Signal that rendering should resume after external pager
		// Bubble Tea's RestoreTerminal() should handle the actual resuming
		m.inPagerMode = false
		return m, nil

	case clearStatusMsg:
		// Clear the status message
		m.state.StatusMessage = ""
		return m, nil

	case quitMsg:
		if msg.saveConfig && m.bus != nil {
			m.bus.Publish(eventbus.ConfigChangedEvent{
				Groups:     m.getGroupsMap(),
				GroupOrder: m.getGroupOrder(),
			})
		}
		return m, tea.Quit

	default:
		// Other messages are handled elsewhere
		return m, nil
	}
}

// handleSortInput processes sort criteria input
func (m *Model) handleSortInput(criteria string) {
	criteria = strings.ToLower(strings.TrimSpace(criteria))
	switch criteria {
	case "name", "n":
		m.currentSort = logic.SortByName
		m.state.StatusMessage = "Sorting by name"
	case "status", "s":
		m.currentSort = logic.SortByStatus
		m.state.StatusMessage = "Sorting by status"
	case "branch", "b":
		m.currentSort = logic.SortByBranch
		m.state.StatusMessage = "Sorting by branch"
	case "modified", "m":
		m.currentSort = logic.SortByStatus
		m.state.StatusMessage = "Sorting by status"
	default:
		m.state.StatusMessage = fmt.Sprintf("Unknown sort criteria: %s", criteria)
		return
	}

	// Update the sort order
	m.updateOrderedLists()
	m.ensureSelectedVisible()
}

// ensureSelectedVisible ensures the selected item is visible in the viewport
func (m *Model) ensureSelectedVisible() {
	// Sync state with navigator
	m.syncNavigatorState()

	// Let navigator handle the viewport adjustment
	m.state.SelectedIndex, m.state.ViewportOffset = m.navigator.SetSelectedIndex(m.state.SelectedIndex)
}

// pageUp moves the selection up by one page
func (m *Model) pageUp() {
	// Page up
	pageSize := m.state.ViewportHeight - 2 // Leave some overlap
	if pageSize < 1 {
		pageSize = 1
	}
	for i := 0; i < pageSize; i++ {
		if m.state.SelectedIndex > 0 {
			m.state.SelectedIndex--
			// Skip gaps
			for m.state.SelectedIndex > 0 && m.isOnGap(m.state.SelectedIndex) {
				m.state.SelectedIndex--
			}
		}
	}
	m.ensureSelectedVisible()
}

// pageDown moves the selection down by one page
func (m *Model) pageDown() {
	// Page down
	pageSize := m.state.ViewportHeight - 2 // Leave some overlap
	if pageSize < 1 {
		pageSize = 1
	}
	maxIndex := m.getMaxIndex()
	for i := 0; i < pageSize; i++ {
		if m.state.SelectedIndex < maxIndex {
			m.state.SelectedIndex++
			// Skip gaps
			for m.state.SelectedIndex < maxIndex && m.isOnGap(m.state.SelectedIndex) {
				m.state.SelectedIndex++
			}
		}
	}
	m.ensureSelectedVisible()
}

// countVisibleRepos counts the total number of repositories
func (m *Model) countVisibleRepos() int {
	return len(m.state.Repositories)
}

// tick returns a command that sends a tick message after a delay
func tick() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// getGroupsMap returns a map of group names to repository paths
func (m *Model) getGroupsMap() map[string][]string {
	return m.state.GetGroupsMap()
}

// getGroupOrder returns the ordered list of group names (excluding hidden)
func (m *Model) getGroupOrder() []string {
	order := make([]string, 0, len(m.state.OrderedGroups))
	for _, groupName := range m.state.OrderedGroups {
		if groupName != HiddenGroupName {
			order = append(order, groupName)
		}
	}
	return order
}

// updateGroupCreationOrder updates the GroupCreationOrder to match OrderedGroups
func (m *Model) updateGroupCreationOrder() {
	// Create new creation order based on current ordered groups
	newOrder := make([]string, 0, len(m.state.OrderedGroups))

	// Add all non-hidden groups in their current order
	for _, groupName := range m.state.OrderedGroups {
		if groupName != HiddenGroupName {
			newOrder = append(newOrder, groupName)
		}
	}

	// Update the creation order
	m.state.GroupCreationOrder = newOrder
}

// updateDuplicateRepoNames updates DisplayName for repos with duplicate names
func (m *Model) updateDuplicateRepoNames() {
	// Count occurrences of each repository name
	nameCount := make(map[string]int)
	for _, repo := range m.state.Repositories {
		nameCount[repo.Name]++
	}

	// For repos with duplicate names, update their DisplayName
	for path, repo := range m.state.Repositories {
		if nameCount[repo.Name] > 1 {
			// Calculate relative path from base directory
			relativePath := strings.TrimPrefix(path, m.config.BaseDir)
			relativePath = strings.TrimPrefix(relativePath, "/")

			// Use the relative path as the display name
			repo.DisplayName = relativePath
		} else {
			// No duplicates, use regular name
			repo.DisplayName = repo.Name
		}
	}
}

// performSearch searches for repositories matching the search query
func (m *Model) performSearch() {
	// Save the old matches to see if they changed
	oldMatches := m.state.SearchMatches
	m.state.SearchMatches = nil

	if m.state.SearchQuery == "" {
		m.state.SearchIndex = 0
		return
	}

	query := strings.ToLower(m.state.SearchQuery)
	currentIdx := 0

	// Search through ALL repositories in the display order
	// This should match exactly what the UI renders

	// First, check all groups
	for _, groupName := range m.state.OrderedGroups {
		group := m.state.Groups[groupName]
		if group == nil {
			continue
		}

		// Check if group name matches
		if strings.Contains(strings.ToLower(groupName), query) {
			m.state.SearchMatches = append(m.state.SearchMatches, currentIdx)
			log.Printf("Search match found at index %d: Group %s", currentIdx, groupName)
		}

		currentIdx++ // Move past group header

		// Only process repos if group is expanded
		if m.state.ExpandedGroups[groupName] {
			for _, repoPath := range group.Repos {
				// Get repository from the main repositories map
				if repo, exists := m.state.Repositories[repoPath]; exists {
					if strings.Contains(strings.ToLower(repo.Name), query) {
						m.state.SearchMatches = append(m.state.SearchMatches, currentIdx)
						log.Printf("Search match found at index %d: %s (in group %s)", currentIdx, repo.Name, groupName)
					}
				}
				currentIdx++
			}
		}

		// Account for gap after group (except hidden at the end)
		if groupName != HiddenGroupName {
			currentIdx++ // Gap after group
		}
	}

	// Now handle ungrouped repos - get them the same way the UI does
	ungroupedRepos := m.getUngroupedRepos()
	if len(ungroupedRepos) > 0 {
		// Check if we should show ungrouped header
		hasUngroupedHeader := false
		for _, repoPath := range ungroupedRepos {
			if _, exists := m.state.Repositories[repoPath]; exists {
				hasUngroupedHeader = true
				break
			}
		}

		if hasUngroupedHeader {
			// Check ungrouped header
			if strings.Contains(strings.ToLower("Ungrouped"), query) {
				m.state.SearchMatches = append(m.state.SearchMatches, currentIdx)
				log.Printf("Search match found at index %d: Ungrouped", currentIdx)
			}
			// No expansion check needed - ungrouped repos are always visible
			// Process ungrouped repos
			for _, repoPath := range ungroupedRepos {
				// Get repository from the main repositories map
				if repo, exists := m.state.Repositories[repoPath]; exists {
					if strings.Contains(strings.ToLower(repo.Name), query) {
						m.state.SearchMatches = append(m.state.SearchMatches, currentIdx)
						log.Printf("Search match found at index %d: %s (ungrouped)", currentIdx, repo.Name)
					}
				}
				currentIdx++
			}
		}
	}

	// Only reset search index if the matches changed
	matchesChanged := len(oldMatches) != len(m.state.SearchMatches)
	if !matchesChanged && len(oldMatches) > 0 {
		for i, match := range oldMatches {
			if i >= len(m.state.SearchMatches) || match != m.state.SearchMatches[i] {
				matchesChanged = true
				break
			}
		}
	}

	if matchesChanged {
		m.state.SearchIndex = 0
	} else {
		// Keep current index but ensure it's within bounds
		if m.state.SearchIndex >= len(m.state.SearchMatches) {
			m.state.SearchIndex = 0
		}
	}

	log.Printf("Search completed for '%s': found %d matches, searchIndex=%d", query, len(m.state.SearchMatches), m.state.SearchIndex)

	// Debug: Log all repositories in state
	log.Printf("Total repositories in state: %d", len(m.state.Repositories))
	for path, repo := range m.state.Repositories {
		log.Printf("  Repo: %s (name: %s)", path, repo.Name)
	}

	// Debug: Log what's in groups
	totalInGroups := 0
	for name, group := range m.state.Groups {
		totalInGroups += len(group.Repos)
		log.Printf("  Group %s has %d repos", name, len(group.Repos))
	}
	log.Printf("Total repos in groups: %d", totalInGroups)

	// Debug: Log ungrouped
	log.Printf("Ungrouped repos: %d", len(ungroupedRepos))
	for _, path := range ungroupedRepos {
		log.Printf("  Ungrouped: %s", path)
	}
}
