package ui

import (
	"fmt"
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
	"gitagrip/internal/ui/logic"
	"gitagrip/internal/ui/views"
)

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
		key.WithKeys("n"),
		key.WithHelp("n", "new group"),
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

// InputMode represents different input modes
type InputMode int

const (
	InputModeNormal InputMode = iota
	InputModeNewGroup
	InputModeMoveToGroup
	InputModeDeleteConfirm
	InputModeSearch
	InputModeSort
	InputModeFilter
)

// Model represents the UI state
type Model struct {
	bus          eventbus.EventBus
	config       *config.Config
	repositories map[string]*domain.Repository // path -> repo
	groups       map[string]*domain.Group      // name -> group
	orderedRepos []string                      // ordered repo paths for display
	orderedGroups []string                     // ordered group names
	groupCreationOrder []string                // tracks order of group creation
	selectedIndex int                          // currently selected item
	selectedRepos map[string]bool              // selected repository paths
	refreshingRepos map[string]bool            // repositories currently being refreshed
	fetchingRepos map[string]bool              // repositories currently being fetched
	pullingRepos map[string]bool               // repositories currently being pulled
	expandedGroups map[string]bool             // which groups are expanded
	scanning      bool                         // whether scanning is in progress
	statusMessage string                       // status bar message
	width         int
	height        int
	showHelp      bool
	showLog       bool
	logContent    string
	showInfo      bool
	infoContent   string
	help          help.Model
	viewportOffset int                         // offset for scrolling
	viewportHeight int                         // available height for repo list
	lastKeyWasG    bool                        // track 'g' key for 'gg' command
	inputMode      InputMode                   // current input mode
	textInput      textinput.Model             // text input for group names
	deleteTarget   string                      // group name being deleted
	searchQuery    string                      // current search query
	searchMatches  []int                       // indices of matching items
	searchIndex    int                         // current match index
	currentSort    logic.SortMode                    // current sort mode
	ungroupedRepos []string                    // cached ungrouped repos
	filterQuery    string                      // current filter query
	isFiltered     bool                        // whether filter is active
	searchFilter   *logic.SearchFilter         // search and filter handler
	navigator      *logic.Navigator            // navigation and viewport handler
	renderer       *views.Renderer             // view renderer
}

// NewModel creates a new UI model
func NewModel(bus eventbus.EventBus, cfg *config.Config) *Model {
	ti := textinput.New()
	ti.Placeholder = "Enter group name..."
	ti.CharLimit = 50
	
	m := &Model{
		bus:            bus,
		config:         cfg,
		repositories:   make(map[string]*domain.Repository),
		groups:         make(map[string]*domain.Group),
		orderedRepos:   make([]string, 0),
		orderedGroups:  make([]string, 0),
		groupCreationOrder: make([]string, 0),
		selectedRepos:  make(map[string]bool),
		refreshingRepos: make(map[string]bool),
		fetchingRepos:  make(map[string]bool),
		pullingRepos:   make(map[string]bool),
		expandedGroups: make(map[string]bool),
		ungroupedRepos: make([]string, 0),
		help:           help.New(),
		textInput:      ti,
		inputMode:      InputModeNormal,
		selectedIndex:  0,
		currentSort:    logic.SortByName,
		searchFilter:   logic.NewSearchFilter(nil), // Will be updated when repos are added
		navigator:      logic.NewNavigator(),
		renderer:       views.NewRenderer(cfg.UISettings.ShowAheadBehind),
	}
	
	// Initialize groups from config
	for name, repoPaths := range cfg.Groups {
		m.groups[name] = &domain.Group{
			Name:  name,
			Repos: repoPaths,
		}
		m.expandedGroups[name] = true // Start with groups expanded
		m.groupCreationOrder = append(m.groupCreationOrder, name)
	}
	m.updateOrderedLists()
	
	// Update searchFilter with the actual repositories map
	m.searchFilter = logic.NewSearchFilter(m.repositories)
	
	return m
}

// syncNavigatorState updates the navigator with current model state
func (m *Model) syncNavigatorState() {
	m.navigator.UpdateState(
		m.selectedIndex,
		m.viewportOffset,
		m.viewportHeight,
		m.expandedGroups,
		m.orderedGroups,
		m.groups,
		m.repositories,
	)
}

// Init returns an initial command
func (m *Model) Init() tea.Cmd {
	// Initialize viewport with reasonable defaults
	m.viewportHeight = 20 // Will be updated on first WindowSizeMsg
	return tea.Tick(time.Millisecond*80, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Update handles messages
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle input mode first
	if m.inputMode != InputModeNormal {
		return m.handleInputMode(msg)
	}
	
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		m.updateViewportHeight()
		
	case tea.KeyMsg:
		// Handle log popup shortcuts
		if m.showLog {
			switch msg.String() {
			case "esc", "l", "q":
				m.showLog = false
				m.logContent = ""
				return m, nil
			}
		}
		
		// Handle info popup shortcuts
		if m.showInfo {
			switch msg.String() {
			case "esc", "i", "q":
				m.showInfo = false
				m.infoContent = ""
				return m, nil
			}
		}
		
		switch {
		case key.Matches(msg, keyQuit):
			// If autosave is enabled, emit config changed event before quitting
			if m.config.UISettings.AutosaveOnExit && m.bus != nil {
				m.bus.Publish(eventbus.ConfigChangedEvent{
					Groups: m.getGroupsMap(),
				})
			}
			return m, tea.Quit
			
		case key.Matches(msg, keyUp):
			if m.selectedIndex > 0 {
				m.selectedIndex--
				m.ensureSelectedVisible()
			}
			
		case key.Matches(msg, keyDown):
			maxIndex := m.getMaxIndex()
			if m.selectedIndex < maxIndex {
				m.selectedIndex++
				m.ensureSelectedVisible()
			}
			
		case key.Matches(msg, keyLeft):
			// Collapse group
			if groupName := m.getSelectedGroup(); groupName != "" {
				m.expandedGroups[groupName] = false
				m.ensureSelectedVisible()
			}
			
		case key.Matches(msg, keyRight):
			// Expand group
			if groupName := m.getSelectedGroup(); groupName != "" {
				m.expandedGroups[groupName] = true
			}
			
		case key.Matches(msg, keyRefresh):
			// Refresh selected repositories, group, or current one
			var repoPaths []string
			if len(m.selectedRepos) > 0 {
				// Refresh selected repos
				for path := range m.selectedRepos {
					repoPaths = append(repoPaths, path)
					m.refreshingRepos[path] = true
				}
			} else if groupName := m.getSelectedGroup(); groupName != "" {
				// Refresh all repos in the selected group
				if group, ok := m.groups[groupName]; ok {
					for _, repoPath := range group.Repos {
						repoPaths = append(repoPaths, repoPath)
						m.refreshingRepos[repoPath] = true
					}
					m.statusMessage = fmt.Sprintf("Refreshing all repos in '%s'", groupName)
				}
			} else {
				// Refresh current repository
				if repoPath := m.getRepoPathAtIndex(m.selectedIndex); repoPath != "" {
					repoPaths = []string{repoPath}
					m.refreshingRepos[repoPath] = true
				}
			}
			
			if len(repoPaths) > 0 && m.bus != nil {
				m.bus.Publish(eventbus.StatusRefreshRequestedEvent{
					RepoPaths: repoPaths,
				})
			}
			
		case key.Matches(msg, keyFilter):
			// Enter filter mode
			m.inputMode = InputModeFilter
			m.textInput.Reset()
			m.textInput.Focus()
			m.updateViewportHeight()
			return m, textinput.Blink
			
		case key.Matches(msg, keyFetch):
			// Fetch selected repositories, group, or current one
			var repoPaths []string
			if len(m.selectedRepos) > 0 {
				// Fetch selected repos
				for path := range m.selectedRepos {
					repoPaths = append(repoPaths, path)
					m.fetchingRepos[path] = true
				}
			} else if groupName := m.getSelectedGroup(); groupName != "" {
				// Fetch all repos in the selected group
				if group, ok := m.groups[groupName]; ok {
					for _, repoPath := range group.Repos {
						repoPaths = append(repoPaths, repoPath)
						m.fetchingRepos[repoPath] = true
					}
					m.statusMessage = fmt.Sprintf("Fetching all repos in '%s'", groupName)
				}
			} else {
				// Fetch current repository
				if repoPath := m.getRepoPathAtIndex(m.selectedIndex); repoPath != "" {
					repoPaths = []string{repoPath}
					m.fetchingRepos[repoPath] = true
				}
			}
			
			if len(repoPaths) > 0 && m.bus != nil {
				m.bus.Publish(eventbus.FetchRequestedEvent{
					RepoPaths: repoPaths,
				})
			}
			
		case key.Matches(msg, keyPull):
			// Pull selected repositories, group, or current one
			var repoPaths []string
			if len(m.selectedRepos) > 0 {
				// Pull selected repos
				for path := range m.selectedRepos {
					repoPaths = append(repoPaths, path)
					m.pullingRepos[path] = true
				}
			} else if groupName := m.getSelectedGroup(); groupName != "" {
				// Pull all repos in the selected group
				if group, ok := m.groups[groupName]; ok {
					for _, repoPath := range group.Repos {
						repoPaths = append(repoPaths, repoPath)
						m.pullingRepos[repoPath] = true
					}
					m.statusMessage = fmt.Sprintf("Pulling all repos in '%s'", groupName)
				}
			} else {
				// Pull current repository
				if repoPath := m.getRepoPathAtIndex(m.selectedIndex); repoPath != "" {
					repoPaths = []string{repoPath}
					m.pullingRepos[repoPath] = true
				}
			}
			
			if len(repoPaths) > 0 && m.bus != nil {
				m.bus.Publish(eventbus.PullRequestedEvent{
					RepoPaths: repoPaths,
				})
			}
			
		case key.Matches(msg, keyFullScan):
			m.statusMessage = "Starting full repository scan..."
			if m.bus != nil && m.config.BaseDir != "" {
				m.bus.Publish(eventbus.ScanRequestedEvent{
					Paths: []string{m.config.BaseDir},
				})
			}
			
		case key.Matches(msg, keyHelp):
			m.showHelp = !m.showHelp
			
		case msg.String() == "/":
			// Enter search mode
			m.inputMode = InputModeSearch
			m.textInput.Reset()
			m.textInput.Focus()
			m.searchQuery = ""
			m.searchMatches = nil
			m.searchIndex = 0
			m.updateViewportHeight()
			return m, textinput.Blink
			
		case key.Matches(msg, keyLog):
			// Show log for selected repository
			if !m.showLog {
				if repoPath := m.getRepoPathAtIndex(m.selectedIndex); repoPath != "" {
					// Get git log asynchronously
					return m, m.fetchGitLog(repoPath)
				}
			} else {
				m.showLog = false
				m.logContent = ""
			}
			
		case key.Matches(msg, keySelect):
			// Check if a group is selected
			if groupName := m.getSelectedGroup(); groupName != "" {
				// Toggle group expansion
				m.expandedGroups[groupName] = !m.expandedGroups[groupName]
				m.ensureSelectedVisible()
			} else if repoPath := m.getRepoPathAtIndex(m.selectedIndex); repoPath != "" {
				// Toggle selection for repository
				if m.selectedRepos[repoPath] {
					delete(m.selectedRepos, repoPath)
				} else {
					m.selectedRepos[repoPath] = true
				}
			}
			
		case key.Matches(msg, keySelectAll):
			// Toggle select all
			if len(m.selectedRepos) == len(m.repositories) {
				// All selected, deselect all
				m.selectedRepos = make(map[string]bool)
			} else {
				// Select all repositories
				for path := range m.repositories {
					m.selectedRepos[path] = true
				}
			}
			
		case msg.String() == "n":
			// Next search result (if in normal mode with search results)
			if m.inputMode == InputModeNormal && len(m.searchMatches) > 0 {
				m.searchIndex = (m.searchIndex + 1) % len(m.searchMatches)
				m.selectedIndex = m.searchMatches[m.searchIndex]
				m.ensureSelectedVisible()
			} else if key.Matches(msg, keyNewGroup) {
				// Create new group
				m.inputMode = InputModeNewGroup
				m.textInput.Reset()
				m.textInput.Focus()
				m.statusMessage = ""
				m.updateViewportHeight()
				return m, textinput.Blink
			}
			
		case msg.String() == "N":
			// Previous search result
			if m.inputMode == InputModeNormal && len(m.searchMatches) > 0 {
				m.searchIndex--
				if m.searchIndex < 0 {
					m.searchIndex = len(m.searchMatches) - 1
				}
				m.selectedIndex = m.searchMatches[m.searchIndex]
				m.ensureSelectedVisible()
			}
			
		case key.Matches(msg, keyMoveToGroup):
			// Move selected repositories to a group
			if len(m.selectedRepos) > 0 && len(m.orderedGroups) > 0 {
				// For now, just move to the first group
				// TODO: Implement group selection UI
				targetGroup := m.orderedGroups[0]
				movedCount := 0
				
				for repoPath := range m.selectedRepos {
					// Find current group (if any)
					var fromGroup string
					for _, group := range m.groups {
						for _, path := range group.Repos {
							if path == repoPath {
								fromGroup = group.Name
								break
							}
						}
					}
					
					// Publish move event
					if m.bus != nil {
						m.bus.Publish(eventbus.RepoMovedEvent{
							RepoPath:  repoPath,
							FromGroup: fromGroup,
							ToGroup:   targetGroup,
						})
						movedCount++
					}
				}
				
				m.statusMessage = fmt.Sprintf("Moved %d repos to '%s'", movedCount, targetGroup)
				m.selectedRepos = make(map[string]bool) // Clear selection
				
				// Emit config changed event
				if m.bus != nil && movedCount > 0 {
					m.bus.Publish(eventbus.ConfigChangedEvent{
						Groups: m.getGroupsMap(),
					})
				}
			} else if len(m.orderedGroups) == 0 {
				m.statusMessage = "No groups available. Press 'n' to create one."
			} else {
				m.statusMessage = "No repositories selected"
			}
			
		case key.Matches(msg, keyDelete):
			// Delete group if a group is selected
			if groupName := m.getSelectedGroup(); groupName != "" {
				m.deleteTarget = groupName
				m.inputMode = InputModeDeleteConfirm
				m.statusMessage = ""
				m.updateViewportHeight()
				return m, nil
			}
			
		case msg.String() == "s":
			// Show sort options
			m.inputMode = InputModeSort
			m.statusMessage = "Sort by: (n)ame (s)tatus (b)ranch (g)roup"
			return m, nil
			
		case key.Matches(msg, keyCopy):
			// Copy repository path to clipboard
			if repoPath := m.getRepoPathAtIndex(m.selectedIndex); repoPath != "" {
				if repo, ok := m.repositories[repoPath]; ok {
					// Use pbcopy on macOS, xclip on Linux
					var cmd *exec.Cmd
					cmd = exec.Command("pbcopy")
					stdin, err := cmd.StdinPipe()
					if err == nil {
						go func() {
							defer stdin.Close()
							stdin.Write([]byte(repo.Path))
						}()
						if err := cmd.Start(); err == nil {
							m.statusMessage = fmt.Sprintf("Copied: %s", repo.Path)
						} else {
							m.statusMessage = "Failed to copy path"
						}
					} else {
						m.statusMessage = "Failed to copy path"
					}
				}
			}
			
		case key.Matches(msg, keyInfo):
			// Show info for selected repository
			if !m.showInfo {
				if repoPath := m.getRepoPathAtIndex(m.selectedIndex); repoPath != "" {
					if repo, ok := m.repositories[repoPath]; ok {
						m.showInfo = true
						m.infoContent = m.buildRepoInfo(repo)
					}
				}
			} else {
				m.showInfo = false
				m.infoContent = ""
			}
			
		// Navigation keys
		case msg.String() == "g":
			if m.lastKeyWasG {
				// gg - go to top
				m.selectedIndex = 0
				m.viewportOffset = 0
				m.lastKeyWasG = false
			} else {
				m.lastKeyWasG = true
				// Don't do anything yet, wait for next key
			}
			
		case key.Matches(msg, keyBottom):
			// G - go to bottom
			m.selectedIndex = m.getMaxIndex()
			m.ensureSelectedVisible()
			m.lastKeyWasG = false
			
		case key.Matches(msg, keyPageDown):
			// Page down
			pageSize := m.viewportHeight - 2 // Leave some overlap
			if pageSize < 1 {
				pageSize = 1
			}
			for i := 0; i < pageSize; i++ {
				if m.selectedIndex < m.getMaxIndex() {
					m.selectedIndex++
				}
			}
			m.ensureSelectedVisible()
			m.lastKeyWasG = false
			
		case key.Matches(msg, keyPageUp):
			// Page up
			pageSize := m.viewportHeight - 2 // Leave some overlap
			if pageSize < 1 {
				pageSize = 1
			}
			for i := 0; i < pageSize; i++ {
				if m.selectedIndex > 0 {
					m.selectedIndex--
				}
			}
			m.ensureSelectedVisible()
			m.lastKeyWasG = false
			
		case key.Matches(msg, keyHalfPageDown):
			// Half page down
			halfPage := m.viewportHeight / 2
			if halfPage < 1 {
				halfPage = 1
			}
			for i := 0; i < halfPage; i++ {
				if m.selectedIndex < m.getMaxIndex() {
					m.selectedIndex++
				}
			}
			m.ensureSelectedVisible()
			m.lastKeyWasG = false
			
		case key.Matches(msg, keyHalfPageUp):
			// Half page up
			halfPage := m.viewportHeight / 2
			if halfPage < 1 {
				halfPage = 1
			}
			for i := 0; i < halfPage; i++ {
				if m.selectedIndex > 0 {
					m.selectedIndex--
				}
			}
			m.ensureSelectedVisible()
			m.lastKeyWasG = false
			
		default:
			// Any other key cancels the 'g' prefix
			if m.lastKeyWasG && msg.String() != "g" {
				m.lastKeyWasG = false
			}
			
		case msg.String() == "{":
			// Jump to beginning of current group
			m.jumpToGroupBoundary(true)
			
		case msg.String() == "}":
			// Jump to end of current group
			m.jumpToGroupBoundary(false)
		}
		
	case EventMsg:
		return m.handleEvent(msg.Event)
		
	case tickMsg:
		// Only return a new tick if we're scanning
		if m.scanning {
			return m, tea.Tick(time.Millisecond*80, func(t time.Time) tea.Msg {
				return tickMsg(t)
			})
		}
		return m, nil
		
	case gitLogMsg:
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Failed to get log: %v", msg.err)
		} else {
			m.showLog = true
			m.logContent = msg.content
		}
		return m, nil
	}
	
	return m, nil
}

// handleEvent processes domain events
func (m *Model) handleEvent(event eventbus.DomainEvent) (tea.Model, tea.Cmd) {
	switch e := event.(type) {
	case eventbus.RepoDiscoveredEvent:
		// Add or update repository
		m.repositories[e.Repo.Path] = &e.Repo
		m.updateOrderedLists()
		// Update searchFilter with new repositories
		m.searchFilter = logic.NewSearchFilter(m.repositories)
		
	case eventbus.StatusUpdatedEvent:
		// Update repository status
		if repo, ok := m.repositories[e.RepoPath]; ok {
			repo.Status = e.Status
		}
		// Clear refreshing, fetching, and pulling states
		delete(m.refreshingRepos, e.RepoPath)
		delete(m.fetchingRepos, e.RepoPath)
		delete(m.pullingRepos, e.RepoPath)
		
		// If all operations completed, show a completion message
		if len(m.refreshingRepos) == 0 && len(m.fetchingRepos) == 0 && len(m.pullingRepos) == 0 {
			m.statusMessage = "All operations completed"
		}
		
	case eventbus.ErrorEvent:
		m.statusMessage = fmt.Sprintf("Error: %s", e.Message)
		// If this is a refresh error for a specific repo, we might need to clear its refreshing state
		// This would require extending the ErrorEvent to include optional repo path
		
	case eventbus.GroupAddedEvent:
		if _, exists := m.groups[e.Name]; !exists {
			m.groups[e.Name] = &domain.Group{
				Name:  e.Name,
				Repos: []string{},
			}
			m.expandedGroups[e.Name] = true // Start expanded so user can see the contents
			// Add to beginning of creation order so new groups appear first
			m.groupCreationOrder = append([]string{e.Name}, m.groupCreationOrder...)
			m.updateOrderedLists()
		}
		
	case eventbus.GroupRemovedEvent:
		if _, exists := m.groups[e.Name]; exists {
			delete(m.groups, e.Name)
			delete(m.expandedGroups, e.Name)
			// Remove from creation order
			newOrder := []string{}
			for _, name := range m.groupCreationOrder {
				if name != e.Name {
					newOrder = append(newOrder, name)
				}
			}
			m.groupCreationOrder = newOrder
			m.updateOrderedLists()
		}
		
	case eventbus.RepoMovedEvent:
		// Remove from old group
		if e.FromGroup != "" {
			if group, exists := m.groups[e.FromGroup]; exists {
				newRepos := make([]string, 0, len(group.Repos))
				for _, path := range group.Repos {
					if path != e.RepoPath {
						newRepos = append(newRepos, path)
					}
				}
				group.Repos = newRepos
			}
		}
		
		// Add to new group
		if e.ToGroup != "" {
			if group, exists := m.groups[e.ToGroup]; exists {
				// Check if already in group
				found := false
				for _, path := range group.Repos {
					if path == e.RepoPath {
						found = true
						break
					}
				}
				if !found {
					group.Repos = append(group.Repos, e.RepoPath)
				}
			}
		}
		
	case eventbus.ScanStartedEvent:
		m.scanning = true
		m.statusMessage = "Scanning for repositories..."
		// Return a tick command to start the spinner animation
		return m, tea.Tick(time.Millisecond*80, func(t time.Time) tea.Msg {
			return tickMsg(t)
		})
		
	case eventbus.ScanCompletedEvent:
		m.scanning = false
		m.statusMessage = fmt.Sprintf("Scan complete. Found %d repositories.", e.ReposFound)
	}
	
	return m, nil
}

// View renders the UI
func (m *Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}
	
	// Create view state
	state := views.ViewState{
		Width:            m.width,
		Height:           m.height,
		Repositories:     m.repositories,
		Groups:           m.groups,
		OrderedGroups:    m.orderedGroups,
		SelectedIndex:    m.selectedIndex,
		SelectedRepos:    m.selectedRepos,
		RefreshingRepos:  m.refreshingRepos,
		FetchingRepos:    m.fetchingRepos,
		PullingRepos:     m.pullingRepos,
		ExpandedGroups:   m.expandedGroups,
		Scanning:         m.scanning,
		StatusMessage:    m.statusMessage,
		ShowHelp:         m.showHelp,
		ShowLog:          m.showLog,
		LogContent:       m.logContent,
		ShowInfo:         m.showInfo,
		InfoContent:      m.infoContent,
		ViewportOffset:   m.viewportOffset,
		ViewportHeight:   m.viewportHeight,
		SearchQuery:      m.searchQuery,
		FilterQuery:      m.filterQuery,
		IsFiltered:       m.isFiltered,
		ShowAheadBehind:  m.config.UISettings.ShowAheadBehind,
		HelpModel:        m.help,
		DeleteTarget:     m.deleteTarget,
		TextInput:        m.getInputText(),
		InputMode:        m.getInputModeString(),
		UngroupedRepos:   m.getUngroupedRepos(),
	}
	
	return m.renderer.Render(state)
}

// getInputText returns the current text input string for the view
func (m *Model) getInputText() string {
	if m.inputMode == InputModeNormal || m.inputMode == InputModeDeleteConfirm {
		return ""
	}
	
	var prefix string
	switch m.inputMode {
	case InputModeNewGroup:
		prefix = "Enter new group name: "
	case InputModeMoveToGroup:
		prefix = "Move to group: "
	case InputModeSearch:
		prefix = "Search: "
	case InputModeFilter:
		prefix = "Filter: "
	case InputModeSort:
		prefix = "Sort by: "
	}
	
	return prefix + m.textInput.View()
}

// getInputModeString returns the string representation of the input mode
func (m *Model) getInputModeString() string {
	switch m.inputMode {
	case InputModeNormal:
		return ""
	case InputModeNewGroup:
		return "new-group"
	case InputModeMoveToGroup:
		return "move-to-group"
	case InputModeDeleteConfirm:
		return "delete-confirm"
	case InputModeSearch:
		return "search"
	case InputModeFilter:
		return "filter"
	case InputModeSort:
		return "sort"
	default:
		return ""
	}
}

// updateOrderedLists updates the ordered lists for display
func (m *Model) updateOrderedLists() {
	// Update ordered repos
	m.orderedRepos = make([]string, 0, len(m.repositories))
	for path := range m.repositories {
		m.orderedRepos = append(m.orderedRepos, path)
	}
	
	// Sort repositories based on current sort mode
	switch m.currentSort {
	case logic.SortByName:
		sort.Slice(m.orderedRepos, func(i, j int) bool {
			repoI, okI := m.repositories[m.orderedRepos[i]]
			repoJ, okJ := m.repositories[m.orderedRepos[j]]
			if !okI || !okJ {
				return !okI
			}
			return strings.ToLower(repoI.Name) < strings.ToLower(repoJ.Name)
		})
		
	case logic.SortByStatus:
		sort.Slice(m.orderedRepos, func(i, j int) bool {
			repoI, okI := m.repositories[m.orderedRepos[i]]
			repoJ, okJ := m.repositories[m.orderedRepos[j]]
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
		sort.Slice(m.orderedRepos, func(i, j int) bool {
			repoI, okI := m.repositories[m.orderedRepos[i]]
			repoJ, okJ := m.repositories[m.orderedRepos[j]]
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
		sort.Strings(m.orderedRepos)
	}
	
	// Update ordered groups
	if m.currentSort == logic.SortByGroup {
		// Sort groups alphabetically
		m.orderedGroups = make([]string, 0, len(m.groups))
		for name := range m.groups {
			m.orderedGroups = append(m.orderedGroups, name)
		}
		sort.Strings(m.orderedGroups)
	} else {
		// Use creation order (newest first)
		m.orderedGroups = make([]string, 0, len(m.groupCreationOrder))
		// Only include groups that still exist
		for _, name := range m.groupCreationOrder {
			if _, exists := m.groups[name]; exists {
				m.orderedGroups = append(m.orderedGroups, name)
			}
		}
	}
	
	// Update ungrouped repos cache
	m.ungroupedRepos = m.getUngroupedRepos()
	
	// Sort ungrouped repos if needed
	if m.currentSort != logic.SortByName {
		// Apply the same sort to ungrouped repos
		switch m.currentSort {
		case logic.SortByStatus:
			sort.Slice(m.ungroupedRepos, func(i, j int) bool {
				repoI, okI := m.repositories[m.ungroupedRepos[i]]
				repoJ, okJ := m.repositories[m.ungroupedRepos[j]]
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
			sort.Slice(m.ungroupedRepos, func(i, j int) bool {
				repoI, okI := m.repositories[m.ungroupedRepos[i]]
				repoJ, okJ := m.repositories[m.ungroupedRepos[j]]
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
}

// getUngroupedRepos returns repositories not in any group
func (m *Model) getUngroupedRepos() []string {
	grouped := make(map[string]bool)
	for _, group := range m.groups {
		for _, repoPath := range group.Repos {
			grouped[repoPath] = true
		}
	}
	
	var ungrouped []string
	for _, repoPath := range m.orderedRepos {
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
	if m.showHelp {
		// Full help takes more space
		reservedLines += 8
	}
	// Account for input field when active
	if m.inputMode != InputModeNormal {
		reservedLines += 2 // Input prompt and field
	}
	
	m.viewportHeight = m.height - reservedLines
	if m.viewportHeight < 1 {
		m.viewportHeight = 1
	}
	
	// Ensure viewport offset is still valid
	m.ensureSelectedVisible()
}

// getSelectedGroup returns the group name if a group header is selected
func (m *Model) getSelectedGroup() string {
	currentIndex := 0
	
	// Check groups first (since they're displayed first now)
	for _, groupName := range m.orderedGroups {
		if currentIndex == m.selectedIndex {
			return groupName // This is the selected group
		}
		currentIndex++
		
		// Skip group contents
		if m.expandedGroups[groupName] {
			group := m.groups[groupName]
			currentIndex += len(group.Repos)
		}
		
		if currentIndex > m.selectedIndex {
			break
		}
	}
	
	return ""
}

// getRepoPathAtIndex returns the repository path at the given index
func (m *Model) getRepoPathAtIndex(index int) string {
	currentIndex := 0
	
	// Check groups first (since they're displayed first now)
	for _, groupName := range m.orderedGroups {
		// Group header itself is not a repo
		if currentIndex == index {
			return "" // This is a group header, not a repo
		}
		currentIndex++
		
		// Check repos in group if expanded
		if m.expandedGroups[groupName] {
			group := m.groups[groupName]
			// Apply the same sorting as in renderRepositoryList
			sortedRepos := make([]string, len(group.Repos))
			copy(sortedRepos, group.Repos)
			
			switch m.currentSort {
			case logic.SortByStatus:
				sort.Slice(sortedRepos, func(i, j int) bool {
					repoI, okI := m.repositories[sortedRepos[i]]
					repoJ, okJ := m.repositories[sortedRepos[j]]
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
				sort.Slice(sortedRepos, func(i, j int) bool {
					repoI, okI := m.repositories[sortedRepos[i]]
					repoJ, okJ := m.repositories[sortedRepos[j]]
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
				
			case logic.SortByName, logic.SortByGroup:
				sort.Slice(sortedRepos, func(i, j int) bool {
					repoI, okI := m.repositories[sortedRepos[i]]
					repoJ, okJ := m.repositories[sortedRepos[j]]
					if !okI || !okJ {
						return !okI
					}
					return strings.ToLower(repoI.Name) < strings.ToLower(repoJ.Name)
				})
			}
			
			for _, repoPath := range sortedRepos {
				if currentIndex == index {
					return repoPath
				}
				currentIndex++
			}
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

// handleInputMode handles input when in text input mode
func (m *Model) handleInputMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	
	// Handle sort mode separately as it doesn't use text input
	if m.inputMode == InputModeSort {
		return m.handleSortMode(msg)
	}
	
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			// Process the input based on mode
			switch m.inputMode {
			case InputModeNewGroup:
				groupName := strings.TrimSpace(m.textInput.Value())
				if groupName != "" {
					// Create the new group
					if m.bus != nil {
						m.bus.Publish(eventbus.GroupAddedEvent{
							Name: groupName,
						})
					}
					
					// Move selected repos to the new group
					movedCount := 0
					for repoPath := range m.selectedRepos {
						// Find current group (if any)
						var fromGroup string
						for _, group := range m.groups {
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
					m.selectedRepos = make(map[string]bool)
					
					// Set cursor to the new group
					// Groups are displayed first and new group will be at the beginning
					m.selectedIndex = 0 // New group is at the top
					m.viewportOffset = 0
					m.ensureSelectedVisible()
					
					if movedCount > 0 {
						m.statusMessage = fmt.Sprintf("Created group '%s' with %d repositories", groupName, movedCount)
					} else {
						m.statusMessage = fmt.Sprintf("Created empty group '%s'", groupName)
					}
					
					// Emit config changed event
					if m.bus != nil {
						m.bus.Publish(eventbus.ConfigChangedEvent{
							Groups: m.getGroupsMap(),
						})
					}
				}
				
			case InputModeSearch:
				searchQuery := strings.TrimSpace(m.textInput.Value())
				if searchQuery != "" {
					m.searchQuery = searchQuery
					m.performSearch()
					if len(m.searchMatches) > 0 {
						m.searchIndex = 0
						m.selectedIndex = m.searchMatches[m.searchIndex]
						m.ensureSelectedVisible()
						m.statusMessage = fmt.Sprintf("Found %d matches (n/N to navigate)", len(m.searchMatches))
					} else {
						m.statusMessage = fmt.Sprintf("No matches found for '%s'", searchQuery)
					}
				}
				
			case InputModeFilter:
				filterQuery := strings.TrimSpace(m.textInput.Value())
				m.filterQuery = filterQuery
				m.isFiltered = filterQuery != ""
				m.updateOrderedLists() // Rebuild lists with filter applied
				if m.isFiltered {
					// Count visible items
					visibleCount := m.countVisibleItems()
					m.statusMessage = fmt.Sprintf("Showing %d items matching filter", visibleCount)
				} else {
					m.statusMessage = "Filter cleared"
				}
			}
			// Return to normal mode
			m.inputMode = InputModeNormal
			m.textInput.Blur()
			m.updateViewportHeight()
			return m, nil
			
		case tea.KeyEsc:
			// Cancel input
			m.inputMode = InputModeNormal
			m.textInput.Blur()
			m.statusMessage = ""
			m.deleteTarget = ""
			// Clear search if we were searching
			if m.inputMode == InputModeSearch {
				m.searchQuery = ""
				m.searchMatches = nil
				m.searchIndex = 0
			}
			// Clear filter if we were filtering
			if m.inputMode == InputModeFilter {
				m.filterQuery = ""
				m.isFiltered = false
				m.updateOrderedLists()
			}
			m.updateViewportHeight()
			return m, nil
		}
		
		// Handle delete confirmation
		if m.inputMode == InputModeDeleteConfirm {
			switch msg.String() {
			case "y", "Y":
				// Confirm delete
				if m.deleteTarget != "" {
					// Get repos in this group before deletion
					group := m.groups[m.deleteTarget]
					repoCount := len(group.Repos)
					
					// Move repos back to ungrouped
					for _, repoPath := range group.Repos {
						if m.bus != nil {
							m.bus.Publish(eventbus.RepoMovedEvent{
								RepoPath:  repoPath,
								FromGroup: m.deleteTarget,
								ToGroup:   "",
							})
						}
					}
					
					// Remove the group
					if m.bus != nil {
						m.bus.Publish(eventbus.GroupRemovedEvent{
							Name: m.deleteTarget,
						})
					}
					
					// Emit config changed event
					if m.bus != nil {
						m.bus.Publish(eventbus.ConfigChangedEvent{
							Groups: m.getGroupsMap(),
						})
					}
					
					m.statusMessage = fmt.Sprintf("Deleted group '%s' (%d repos moved to ungrouped)", m.deleteTarget, repoCount)
					
					// Adjust selection if needed
					if m.selectedIndex > 0 {
						m.selectedIndex--
					}
					m.ensureSelectedVisible()
				}
				m.inputMode = InputModeNormal
				m.deleteTarget = ""
				m.updateViewportHeight()
				return m, nil
				
			case "n", "N":
				// Cancel delete
				m.inputMode = InputModeNormal
				m.statusMessage = "Delete cancelled"
				m.deleteTarget = ""
				m.updateViewportHeight()
				return m, nil
			}
		}
	}
	
	// Update text input
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

// ensureSelectedVisible adjusts the viewport to keep the selected item visible
func (m *Model) ensureSelectedVisible() {
	// Sync state with navigator
	m.syncNavigatorState()
	
	// Let navigator handle the viewport adjustment
	m.selectedIndex, m.viewportOffset = m.navigator.SetSelectedIndex(m.selectedIndex)
}

// keyBindings returns the help key bindings
func (m *Model) keyBindings() help.KeyMap {
	return keyMap{}
}

// keyMap defines our key bindings
type keyMap struct{}

// ShortHelp returns keybindings to be shown in the mini help view
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{keyHelp, keyQuit}
}

// FullHelp returns keybindings for the expanded help view
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{keyUp, keyDown, keyLeft, keyRight},
		{keyPageUp, keyPageDown, keyHalfPageUp, keyHalfPageDown},
		{keyTop, keyBottom},
		{keySelect, keySelectAll},
		{keyNewGroup, keyMoveToGroup, keyDelete},
		{keyRefresh, keyFetch, keyFilter, keyPull},
		{keyFullScan, keyLog, keyCopy, keyInfo},
		{keyHelp, keyQuit},
	}
}

// getGroupsMap returns the current groups as a map
func (m *Model) getGroupsMap() map[string][]string {
	groups := make(map[string][]string)
	for name, group := range m.groups {
		groups[name] = append([]string(nil), group.Repos...) // Copy slice
	}
	return groups
}

// performSearch searches for repositories matching the query
func (m *Model) performSearch() {
	// Update searchFilter with current repositories
	m.searchFilter = logic.NewSearchFilter(m.repositories)
	
	// Get ungrouped repos
	ungroupedRepos := m.ungroupedRepos
	if len(ungroupedRepos) == 0 {
		ungroupedRepos = m.getUngroupedRepos()
	}
	
	// Perform search using the filter logic
	results := m.searchFilter.PerformSearch(m.searchQuery, m.orderedGroups, m.groups, m.expandedGroups, ungroupedRepos)
	
	// Convert results to match indices
	m.searchMatches = nil
	for _, result := range results {
		m.searchMatches = append(m.searchMatches, result.Index)
	}
	
	// Jump to first match if any
	if len(m.searchMatches) > 0 {
		m.searchIndex = 0
		m.selectedIndex = m.searchMatches[0]
		m.ensureSelectedVisible()
	}
}

// handleSortMode handles input when in sort mode
func (m *Model) handleSortMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "n":
			// Sort by name
			m.currentSort = logic.SortByName
			m.updateOrderedLists()
			m.statusMessage = "Sorted by name"
			m.inputMode = InputModeNormal
			return m, nil
			
		case "s":
			// Sort by status
			m.currentSort = logic.SortByStatus
			m.updateOrderedLists()
			m.statusMessage = "Sorted by status"
			m.inputMode = InputModeNormal
			return m, nil
			
		case "b":
			// Sort by branch
			m.currentSort = logic.SortByBranch
			m.updateOrderedLists()
			m.statusMessage = "Sorted by branch"
			m.inputMode = InputModeNormal
			return m, nil
			
		case "g":
			// Sort by group
			m.currentSort = logic.SortByGroup
			m.updateOrderedLists()
			m.statusMessage = "Sorted by group"
			m.inputMode = InputModeNormal
			return m, nil
			
		case "esc", "q":
			// Cancel sort
			m.inputMode = InputModeNormal
			m.statusMessage = ""
			return m, nil
		}
	}
	
	return m, nil
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
	for _, group := range m.groups {
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
	for _, groupName := range m.orderedGroups {
		group := m.groups[groupName]
		
		// Check if group or any of its repos match
		groupHasMatches := false
		repoCount := 0
		
		for _, repoPath := range group.Repos {
			if repo, ok := m.repositories[repoPath]; ok {
				if m.searchFilter.MatchesFilter(repo, groupName, m.filterQuery) {
					groupHasMatches = true
					repoCount++
				}
			}
		}
		
		if groupHasMatches || m.searchFilter.MatchesGroupFilter(groupName, m.filterQuery) {
			count++ // Count the group header
			if m.expandedGroups[groupName] {
				count += repoCount
			}
		}
	}
	
	// Count ungrouped repos
	ungroupedRepos := m.ungroupedRepos
	if len(ungroupedRepos) == 0 {
		ungroupedRepos = m.getUngroupedRepos()
	}
	for _, repoPath := range ungroupedRepos {
		if repo, ok := m.repositories[repoPath]; ok {
			if m.searchFilter.MatchesFilter(repo, "", m.filterQuery) {
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
	ungroupedRepos := m.ungroupedRepos
	if len(ungroupedRepos) == 0 {
		ungroupedRepos = m.getUngroupedRepos()
	}
	return m.navigator.GetCurrentIndexForRepo(repoPath, ungroupedRepos)
}

// jumpToGroupBoundary jumps to the beginning or end of the current group
func (m *Model) jumpToGroupBoundary(toBeginning bool) {
	m.syncNavigatorState()
	ungroupedRepos := m.ungroupedRepos
	if len(ungroupedRepos) == 0 {
		ungroupedRepos = m.getUngroupedRepos()
	}
	
	needsCrossGroupJump, fromGroup := m.navigator.JumpToGroupBoundary(toBeginning, ungroupedRepos)
	m.selectedIndex = m.navigator.GetSelectedIndex()
	m.viewportOffset = m.navigator.GetViewportOffset()
	
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
	ungroupedRepos := m.ungroupedRepos
	if len(ungroupedRepos) == 0 {
		ungroupedRepos = m.getUngroupedRepos()
	}
	
	if m.navigator.JumpToNextGroupEnd(currentGroupName, ungroupedRepos) {
		m.selectedIndex = m.navigator.GetSelectedIndex()
		m.viewportOffset = m.navigator.GetViewportOffset()
	}
}

// jumpToPreviousGroupStart jumps to the first repo of the previous group
func (m *Model) jumpToPreviousGroupStart(currentGroupName string) {
	m.syncNavigatorState()
	
	if m.navigator.JumpToPreviousGroupStart(currentGroupName) {
		m.selectedIndex = m.navigator.GetSelectedIndex()
		m.viewportOffset = m.navigator.GetViewportOffset()
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