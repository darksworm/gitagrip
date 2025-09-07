package ui

import (
	"fmt"
	"log"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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
var (
	keyUp = key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	)
	keyDown = key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	)
	keyLeft = key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "collapse"),
	)
	keyRight = key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→/l", "expand"),
	)
	keyRefresh = key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	)
	keyFullScan = key.NewBinding(
		key.WithKeys("S"),
		key.WithHelp("S", "full scan"),
	)
	keyFetch = key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "fetch"),
	)
	keyFilter = key.NewBinding(
		key.WithKeys("F"),
		key.WithHelp("F", "filter"),
	)
	keyPull = key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "pull"),
	)
	keyLog = key.NewBinding(
		key.WithKeys("l"),
		key.WithHelp("l", "log"),
	)
	keyHelp = key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	)
	keyQuit = key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	)
	keyTop = key.NewBinding(
		key.WithKeys("g g", "home"),
		key.WithHelp("gg/home", "top"),
	)
	keyBottom = key.NewBinding(
		key.WithKeys("G", "end"),
		key.WithHelp("G/end", "bottom"),
	)
	keyPageUp = key.NewBinding(
		key.WithKeys("pgup", "ctrl+b"),
		key.WithHelp("pgup/ctrl+b", "page up"),
	)
	keyPageDown = key.NewBinding(
		key.WithKeys("pgdown", "ctrl+f", " "),
		key.WithHelp("pgdn/ctrl+f/space", "page down"),
	)
	keyHalfPageUp = key.NewBinding(
		key.WithKeys("ctrl+u"),
		key.WithHelp("ctrl+u", "half page up"),
	)
	keyHalfPageDown = key.NewBinding(
		key.WithKeys("ctrl+d"),
		key.WithHelp("ctrl+d", "half page down"),
	)
	keySelect = key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "select/expand"),
	)
	keySelectAll = key.NewBinding(
		key.WithKeys("cmd+a"),
		key.WithHelp("cmd+a", "select all"),
	)
	keyNewGroup = key.NewBinding(
		key.WithKeys("N"),
		key.WithHelp("N", "new group"),
	)
	keyMoveToGroup = key.NewBinding(
		key.WithKeys("m"),
		key.WithHelp("m", "move to group"),
	)
	keyDelete = key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "delete group"),
	)
	keyCopy = key.NewBinding(
		key.WithKeys("y"),
		key.WithHelp("y", "copy path"),
	)
	keyInfo = key.NewBinding(
		key.WithKeys("i"),
		key.WithHelp("i", "repo info"),
	)
)

// Model represents the UI state
type Model struct {
	bus    eventbus.EventBus
	config *config.Config
	state  *state.AppState // centralized state

	// UI-specific state not in AppState
	width       int
	height      int
	help        help.Model
	lastKeyWasG bool // track 'g' key for 'gg' command
	// Removed: inputMode, textInput, deleteTarget - now handled by input handler
	currentSort logic.SortMode // current sort mode
	// Removed: useNewInput - fully migrated to new input handler

	// Handlers
	searchFilter *logic.SearchFilter          // search and filter handler
	navigator    *logic.Navigator             // navigation and viewport handler
	renderer     *views.Renderer              // view renderer
	eventHandler *handlers.EventHandler       // event processing handler
	viewModel    *viewmodels.ViewModel        // view model for rendering
	store        repositories.RepositoryStore // repository store for data access
	cmdExecutor  *commands.Executor           // command executor
	inputHandler *input.Handler               // input handling
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

	// Create view model with a placeholder text input (actual one is in input handler)
	placeholderTextInput := textinput.New()
	m.viewModel = viewmodels.NewViewModel(appState, cfg, placeholderTextInput)
	m.viewModel.SetHelp(m.help)

	// Initialize groups from config
	for name, repoPaths := range cfg.Groups {
		m.state.AddGroup(name, repoPaths)
	}
	m.updateOrderedLists()

	// Update searchFilter with the actual repositories map
	m.searchFilter = logic.NewSearchFilter(m.state.Repositories)

	return m
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
		// Handle log/info popups first
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
func (m *Model) handleEvent(event eventbus.DomainEvent) (tea.Model, tea.Cmd) {
	cmd := m.eventHandler.HandleEvent(event)
	// Update searchFilter reference
	m.searchFilter = m.eventHandler.GetSearchFilter()
	return m, cmd
}

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

	case logic.SortByGroup:
		// For group sort, we don't sort the repos here, but we sort groups alphabetically
		// Repos will be displayed in their groups

	default:
		// Default to alphabetical by path
		sort.Strings(m.state.OrderedRepos)
	}

	// Update ordered groups
	if m.currentSort == logic.SortByGroup {
		// Sort groups alphabetically
		m.state.OrderedGroups = make([]string, 0, len(m.state.Groups))
		for name := range m.state.Groups {
			if name != HiddenGroupName {
				m.state.OrderedGroups = append(m.state.OrderedGroups, name)
			}
		}
		sort.Strings(m.state.OrderedGroups)
	} else {
		// Use creation order (newest first)
		m.state.OrderedGroups = make([]string, 0, len(m.state.GroupCreationOrder))
		// Only include groups that still exist
		for _, name := range m.state.GroupCreationOrder {
			if _, exists := m.state.Groups[name]; exists {
				if name != HiddenGroupName {
					m.state.OrderedGroups = append(m.state.OrderedGroups, name)
				}
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
	if m.state.ShowHelp {
		// Full help takes more space
		reservedLines += 8
	}
	// Account for input field when active
	// Input mode lines handled by input handler

	m.state.ViewportHeight = m.height - reservedLines
	if m.state.ViewportHeight < 1 {
		m.state.ViewportHeight = 1
	}

	// Ensure viewport offset is still valid
	m.ensureSelectedVisible()
}

// getSelectedGroup returns the group name if a group header is selected
func (m *Model) getSelectedGroup() string {
	currentIndex := 0

	// Check groups first (since they're displayed first now)
	for _, groupName := range m.store.GetOrderedGroups() {
		if currentIndex == m.state.SelectedIndex {
			return groupName // This is the selected group
		}
		currentIndex++

		// Skip group contents
		if m.store.IsGroupExpanded(groupName) {
			group, _ := m.store.GetGroup(groupName)
			currentIndex += len(group.Repos)
		}
		
		// Check if we're on the gap after this group
		gapIndex := currentIndex
		// Account for gap after group (except hidden group at the end)
		if groupName != HiddenGroupName || currentIndex < m.state.SelectedIndex {
			if gapIndex == m.state.SelectedIndex {
				// We're on the gap, return the group before it
				return groupName
			}
			currentIndex++ // Gap after group
		}

		if currentIndex > m.state.SelectedIndex {
			break
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
	info.WriteString(fmt.Sprintf("  Branch: %s\n", repo.Status.Branch))

	// Clean/Dirty status
	if repo.Status.IsDirty {
		info.WriteString("  State: Dirty (uncommitted changes)\n")
	} else if repo.Status.HasUntracked {
		info.WriteString("  State: Has untracked files\n")
	} else {
		info.WriteString("  State: Clean\n")
	}

	// Ahead/Behind
	if repo.Status.AheadCount > 0 || repo.Status.BehindCount > 0 {
		info.WriteString(fmt.Sprintf("  Ahead: %d commits\n", repo.Status.AheadCount))
		info.WriteString(fmt.Sprintf("  Behind: %d commits\n", repo.Status.BehindCount))
	}

	// Stashes
	if repo.Status.StashCount > 0 {
		info.WriteString(fmt.Sprintf("  Stashes: %d\n", repo.Status.StashCount))
	}

	// Error
	if repo.Status.Error != "" {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
		info.WriteString(fmt.Sprintf("  Error: %s\n", errorStyle.Render(repo.Status.Error)))
	}

	info.WriteString("\n")
	info.WriteString("Press ESC or 'i' to close")

	return info.String()
}

// countVisibleItems counts how many items are visible with current filter
func (m *Model) countVisibleItems() int {
	count := 0

	// Count groups and their repos
	for _, groupName := range m.store.GetOrderedGroups() {
		group, _ := m.store.GetGroup(groupName)

		// Check if group or any of its repos match
		groupHasMatches := false
		repoCount := 0

		for _, repoPath := range group.Repos {
			if repo, ok := m.state.Repositories[repoPath]; ok {
				if m.searchFilter.MatchesFilter(repo, groupName, m.state.FilterQuery) {
					groupHasMatches = true
					repoCount++
				}
			}
		}

		if groupHasMatches || m.searchFilter.MatchesGroupFilter(groupName, m.state.FilterQuery) {
			count++ // Count the group header
			if m.store.IsGroupExpanded(groupName) {
				count += repoCount
			}
		}
	}

	// Count ungrouped repos
	ungroupedRepos := m.state.UngroupedRepos
	if len(ungroupedRepos) == 0 {
		ungroupedRepos = m.getUngroupedRepos()
	}
	for _, repoPath := range ungroupedRepos {
		if repo, ok := m.state.Repositories[repoPath]; ok {
			if m.searchFilter.MatchesFilter(repo, "", m.state.FilterQuery) {
				count++
			}
		}
	}

	return count
}

// getCurrentIndexForGroup finds the current display index for a group
func (m *Model) getCurrentIndexForGroup(groupName string) int {
	m.syncNavigatorState()
	return m.navigator.GetCurrentIndexForGroup(groupName)
}

// getCurrentIndexForRepo finds the current display index for a repo
func (m *Model) getCurrentIndexForRepo(repoPath string) int {
	m.syncNavigatorState()
	ungroupedRepos := m.state.UngroupedRepos
	if len(ungroupedRepos) == 0 {
		ungroupedRepos = m.getUngroupedRepos()
	}
	return m.navigator.GetCurrentIndexForRepo(repoPath, ungroupedRepos)
}

// jumpToGroupBoundary jumps to the beginning or end of the current group
func (m *Model) jumpToGroupBoundary(toBeginning bool) {
	m.syncNavigatorState()
	ungroupedRepos := m.state.UngroupedRepos
	if len(ungroupedRepos) == 0 {
		ungroupedRepos = m.getUngroupedRepos()
	}

	needsCrossGroupJump, fromGroup := m.navigator.JumpToGroupBoundary(toBeginning, ungroupedRepos)
	m.state.SelectedIndex = m.navigator.GetSelectedIndex()
	m.state.ViewportOffset = m.navigator.GetViewportOffset()

	if needsCrossGroupJump && fromGroup != "" {
		if toBeginning {
			m.jumpToPreviousGroupStart(fromGroup)
		} else {
			m.jumpToNextGroupEnd(fromGroup)
		}
	}
}

// jumpToNextGroupEnd jumps to the last repo of the next group
func (m *Model) jumpToNextGroupEnd(currentGroupName string) {
	m.syncNavigatorState()
	ungroupedRepos := m.state.UngroupedRepos
	if len(ungroupedRepos) == 0 {
		ungroupedRepos = m.getUngroupedRepos()
	}

	if m.navigator.JumpToNextGroupEnd(currentGroupName, ungroupedRepos) {
		m.state.SelectedIndex = m.navigator.GetSelectedIndex()
		m.state.ViewportOffset = m.navigator.GetViewportOffset()
	}
}

// jumpToPreviousGroupStart jumps to the first repo of the previous group
func (m *Model) jumpToPreviousGroupStart(currentGroupName string) {
	m.syncNavigatorState()

	if m.navigator.JumpToPreviousGroupStart(currentGroupName) {
		m.state.SelectedIndex = m.navigator.GetSelectedIndex()
		m.state.ViewportOffset = m.navigator.GetViewportOffset()
	}
}

// fetchGitLog returns a command that fetches git log for a repository
func (m *Model) fetchGitLog(repoPath string) tea.Cmd {
	return func() tea.Msg {
		// Run git log command
		cmd := exec.Command("git", "log", "--oneline", "-20", "--decorate", "--color=always")
		cmd.Dir = repoPath

		output, err := cmd.Output()
		if err != nil {
			return gitLogMsg{
				repoPath: repoPath,
				err:      err,
			}
		}

		return gitLogMsg{
			repoPath: repoPath,
			content:  string(output),
		}
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
				if m.getRepoPathAtIndex(m.state.SelectedIndex) == "" && m.getSelectedGroup() == "" {
					if m.state.SelectedIndex > 0 {
						m.state.SelectedIndex--
					}
				}
				m.ensureSelectedVisible()
			}
		case "down":
			maxIndex := m.getMaxIndex()
			if m.state.SelectedIndex < maxIndex {
				m.state.SelectedIndex++
				// Skip gaps when moving down
				if m.getRepoPathAtIndex(m.state.SelectedIndex) == "" && m.getSelectedGroup() == "" {
					if m.state.SelectedIndex < maxIndex {
						m.state.SelectedIndex++
					}
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
			
			if a.Direction == "next" {
				// Navigate to next search result
				m.state.SearchIndex = (m.state.SearchIndex + 1) % len(m.state.SearchMatches)
			} else if a.Direction == "prev" {
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
			} else if groupName := m.getSelectedGroup(); groupName != "" {
				// Refresh all repos in the selected group
				if group, ok := m.store.GetGroup(groupName); ok {
					for _, repoPath := range group.Repos {
						repoPaths = append(repoPaths, repoPath)
					}
					m.state.StatusMessage = fmt.Sprintf("Refreshing all repos in '%s'", groupName)
				}
			} else {
				// Refresh current repository
				if repoPath := m.getRepoPathAtIndex(m.state.SelectedIndex); repoPath != "" {
					repoPaths = []string{repoPath}
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
		} else if groupName := m.getSelectedGroup(); groupName != "" {
			// Fetch all repos in the selected group
			if group, ok := m.store.GetGroup(groupName); ok {
				for _, repoPath := range group.Repos {
					repoPaths = append(repoPaths, repoPath)
				}
				m.state.StatusMessage = fmt.Sprintf("Fetching all repos in '%s'", groupName)
			}
		} else {
			// Fetch current repository
			if repoPath := m.getRepoPathAtIndex(m.state.SelectedIndex); repoPath != "" {
				repoPaths = []string{repoPath}
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
		} else if groupName := m.getSelectedGroup(); groupName != "" {
			// Pull all repos in the selected group
			if group, ok := m.store.GetGroup(groupName); ok {
				for _, repoPath := range group.Repos {
					repoPaths = append(repoPaths, repoPath)
				}
				m.state.StatusMessage = fmt.Sprintf("Pulling all repos in '%s'", groupName)
			}
		} else {
			// Pull current repository
			if repoPath := m.getRepoPathAtIndex(m.state.SelectedIndex); repoPath != "" {
				repoPaths = []string{repoPath}
			}
		}
		return m.cmdExecutor.ExecutePull(repoPaths)

	case inputtypes.OpenLogAction:
		// Show git log for current repo
		if repoPath := m.getRepoPathAtIndex(m.state.SelectedIndex); repoPath != "" {
			m.state.ShowLog = true
			return m.fetchGitLog(repoPath)
		}

	case inputtypes.ToggleInfoAction:
		m.state.ShowInfo = !m.state.ShowInfo
		if m.state.ShowInfo {
			// Build info content for current repo
			if repoPath := m.getRepoPathAtIndex(m.state.SelectedIndex); repoPath != "" {
				if repo, ok := m.state.Repositories[repoPath]; ok {
					m.state.InfoContent = m.buildRepoInfo(repo)
				}
			}
		} else {
			m.state.InfoContent = ""
		}

	case inputtypes.ToggleHelpAction:
		m.state.ShowHelp = !m.state.ShowHelp

	case inputtypes.ToggleGroupAction:
		if groupName := m.getSelectedGroup(); groupName != "" {
			m.state.ExpandedGroups[groupName] = !m.state.ExpandedGroups[groupName]
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
				Groups: m.getGroupsMap(),
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
					Groups: m.getGroupsMap(),
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
			log.Printf(a.Text)
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

	case inputtypes.QuitAction:
		if !a.Force && m.config.UISettings.AutosaveOnExit && m.bus != nil {
			m.bus.Publish(eventbus.ConfigChangedEvent{
				Groups: m.getGroupsMap(),
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
		return m, tick()

	case gitLogMsg:
		if msg.err != nil {
			m.state.LogContent = fmt.Sprintf("Error fetching log for %s:\n%v", msg.repoPath, msg.err)
		} else {
			m.state.LogContent = fmt.Sprintf("Git log for %s:\n\n%s", msg.repoPath, msg.content)
		}
		return m, nil

	case quitMsg:
		if msg.saveConfig && m.bus != nil {
			m.bus.Publish(eventbus.ConfigChangedEvent{
				Groups: m.getGroupsMap(),
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
			if m.getRepoPathAtIndex(m.state.SelectedIndex) == "" && m.getSelectedGroup() == "" {
				if m.state.SelectedIndex > 0 {
					m.state.SelectedIndex--
				}
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
			if m.getRepoPathAtIndex(m.state.SelectedIndex) == "" && m.getSelectedGroup() == "" {
				if m.state.SelectedIndex < maxIndex {
					m.state.SelectedIndex++
				}
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
