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
	
	// Build the main content
	var content strings.Builder
	
	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205"))
	
	content.WriteString(titleStyle.Render("GitaGrip"))
	content.WriteString("\n")
	
	// Show text input if in input mode
	if m.inputMode != InputModeNormal {
		content.WriteString("\n")
		if m.inputMode == InputModeNewGroup {
			content.WriteString("Enter new group name: ")
		} else if m.inputMode == InputModeMoveToGroup {
			content.WriteString("Move to group: ")
		} else if m.inputMode == InputModeDeleteConfirm {
			confirmStyle := lipgloss.NewStyle().Bold(true)
			content.WriteString(confirmStyle.Render(fmt.Sprintf("Delete group '%s'? (y/n): ", m.deleteTarget)))
		} else if m.inputMode == InputModeSearch {
			content.WriteString("Search: ")
		} else if m.inputMode == InputModeFilter {
			content.WriteString("Filter: ")
		}
		if m.inputMode != InputModeDeleteConfirm {
			content.WriteString(m.textInput.View())
		}
		content.WriteString("\n")
	}
	content.WriteString("\n")
	
	// Repository list
	if m.scanning && len(m.repositories) == 0 {
		// Show scanning animation
		scanStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("33"))
		spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		frame := (time.Now().UnixMilli() / 80) % int64(len(spinner))
		content.WriteString(scanStyle.Render(fmt.Sprintf("%s Scanning for repositories...", spinner[frame])))
		content.WriteString("\n")
	} else if len(m.repositories) == 0 && !m.scanning {
		dimStyle := lipgloss.NewStyle().Faint(true)
		content.WriteString(dimStyle.Render("No repositories found. Press F for full scan."))
	} else {
		content.WriteString(m.renderRepositoryList())
	}
	
	// Status bar
	content.WriteString("\n\n")
	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))
	
	// Build status components
	var statusParts []string
	
	// Selection count
	if len(m.selectedRepos) > 0 {
		statusParts = append(statusParts, fmt.Sprintf("%d selected", len(m.selectedRepos)))
	}
	
	// Progress indicators for operations
	refreshingCount := len(m.refreshingRepos)
	fetchingCount := len(m.fetchingRepos)
	pullingCount := len(m.pullingRepos)
	
	if refreshingCount > 0 {
		statusParts = append(statusParts, fmt.Sprintf("Refreshing %d", refreshingCount))
	}
	if fetchingCount > 0 {
		statusParts = append(statusParts, fmt.Sprintf("Fetching %d", fetchingCount))
	}
	if pullingCount > 0 {
		statusParts = append(statusParts, fmt.Sprintf("Pulling %d", pullingCount))
	}
	
	// Scanning indicator
	if m.scanning {
		spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		frame := (time.Now().UnixMilli() / 80) % int64(len(spinner))
		scanStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("33"))
		statusParts = append(statusParts, scanStyle.Render(fmt.Sprintf("%s Scanning...", spinner[frame])))
	}
	
	// Filter indicator
	if m.isFiltered {
		filterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214")) // yellow
		filterText := fmt.Sprintf("Filter: %s", m.filterQuery)
		statusParts = append(statusParts, filterStyle.Render(filterText))
	}
	
	// Status message (if any)
	if m.statusMessage != "" && refreshingCount == 0 && fetchingCount == 0 && pullingCount == 0 && !m.scanning {
		statusParts = append(statusParts, m.statusMessage)
	}
	
	// Join and render status
	if len(statusParts) > 0 {
		content.WriteString(statusStyle.Render(strings.Join(statusParts, " | ")))
	}
	
	
	// Log popup
	if m.showLog && m.logContent != "" {
		// Create a box for the log content
		logBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("205")).
			Padding(1).
			MaxHeight(m.height - 4).
			MaxWidth(m.width - 4).
			Render(m.logContent)
		
		// Center the log box
		centeredLog := lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			logBox)
		
		return centeredLog
	}
	
	// Info popup
	if m.showInfo && m.infoContent != "" {
		// Create a box for the info content
		infoBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("78")).
			Padding(1).
			MaxHeight(m.height - 4).
			MaxWidth(m.width - 4).
			Render(m.infoContent)
		
		// Center the info box
		centeredInfo := lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			infoBox)
		
		return centeredInfo
	}
	
	// Help
	if m.showHelp {
		content.WriteString("\n\n")
		content.WriteString(m.help.View(m.keyBindings()))
	} else {
		content.WriteString("\n")
		helpStyle := lipgloss.NewStyle().Faint(true)
		content.WriteString(helpStyle.Render("Press ? for help"))
	}
	
	// Apply padding and size constraints
	mainStyle := lipgloss.NewStyle().
		Padding(1, 2).
		Width(m.width).
		Height(m.height)
		
	return mainStyle.Render(content.String())
}

// renderRepositoryList renders the grouped repository list
func (m *Model) renderRepositoryList() string {
	var visibleLines []string
	currentIndex := 0
	totalItems := 0
	
	// Calculate the total number of items first
	// Groups first
	for _, groupName := range m.orderedGroups {
		totalItems++ // Group header
		if m.expandedGroups[groupName] {
			group := m.groups[groupName]
			totalItems += len(group.Repos)
		}
	}
	// Then ungrouped repos
	totalItems += len(m.getUngroupedRepos())
	
	// Determine if we need scroll indicators
	needsTopIndicator := m.viewportOffset > 0
	needsBottomIndicator := m.viewportOffset + m.viewportHeight < totalItems
	
	// Adjust effective viewport to account for scroll indicators
	effectiveViewportHeight := m.viewportHeight
	effectiveViewportOffset := m.viewportOffset
	
	// Reserve space for indicators within the viewport
	if needsTopIndicator {
		effectiveViewportHeight--
	}
	if needsBottomIndicator {
		effectiveViewportHeight--
	}
	
	// Ensure we have at least 1 line for content
	if effectiveViewportHeight < 1 {
		effectiveViewportHeight = 1
	}
	
	// Reset index
	currentIndex = 0
	
	// Render groups first
	for _, groupName := range m.orderedGroups {
		group := m.groups[groupName]
		isExpanded := m.expandedGroups[groupName]
		
		// Skip group if filtered and has no matching repos
		groupHasMatches := false
		if m.isFiltered {
			for _, repoPath := range group.Repos {
				if repo, ok := m.repositories[repoPath]; ok {
					if m.searchFilter.MatchesFilter(repo, groupName, m.filterQuery) {
						groupHasMatches = true
						break
					}
				}
			}
			if !groupHasMatches && !m.searchFilter.MatchesGroupFilter(groupName, m.filterQuery) {
				continue
			}
		}
		
		// Render group header
		if currentIndex >= effectiveViewportOffset && len(visibleLines) < effectiveViewportHeight {
			isSelected := currentIndex == m.selectedIndex
			line := m.renderGroupHeader(group, isExpanded, isSelected)
			visibleLines = append(visibleLines, line)
		}
		currentIndex++
		
		// Render group contents if expanded
		if isExpanded {
			// Sort repos within the group based on current sort mode
			sortedRepos := make([]string, len(group.Repos))
			copy(sortedRepos, group.Repos)
			
			switch m.currentSort {
			case logic.SortByStatus:
				sort.Slice(sortedRepos, func(i, j int) bool {
					repoI, okI := m.repositories[sortedRepos[i]]
					repoJ, okJ := m.repositories[sortedRepos[j]]
					if !okI || !okJ {
						return !okI // Put missing repos at the end
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
						return !okI // Put missing repos at the end
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
						return !okI // Put missing repos at the end
					}
					return strings.ToLower(repoI.Name) < strings.ToLower(repoJ.Name)
				})
			}
			
			for _, repoPath := range sortedRepos {
				if repo, ok := m.repositories[repoPath]; ok {
					// Skip if filtered and doesn't match
					if m.isFiltered && !m.searchFilter.MatchesFilter(repo, groupName, m.filterQuery) {
						continue
					}
					
					if currentIndex >= effectiveViewportOffset && len(visibleLines) < effectiveViewportHeight {
						isSelected := currentIndex == m.selectedIndex
						line := m.renderRepository(repo, isSelected, 1)
						visibleLines = append(visibleLines, line)
					}
					currentIndex++
				}
			}
		}
	}
	
	// Then render ungrouped repositories
	ungroupedRepos := m.ungroupedRepos
	if len(ungroupedRepos) == 0 {
		ungroupedRepos = m.getUngroupedRepos()
	}
	for _, repoPath := range ungroupedRepos {
		if repo, ok := m.repositories[repoPath]; ok {
			// Skip if filtered and doesn't match
			if m.isFiltered && !m.searchFilter.MatchesFilter(repo, "", m.filterQuery) {
				continue
			}
			
			if currentIndex >= effectiveViewportOffset && len(visibleLines) < effectiveViewportHeight {
				isSelected := currentIndex == m.selectedIndex
				line := m.renderRepository(repo, isSelected, 0)
				visibleLines = append(visibleLines, line)
			}
			currentIndex++
		}
	}
	
	// Build final result with indicators
	var result []string
	
	// Add top scroll indicator if needed
	if needsTopIndicator {
		scrollStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true)
		result = append(result, scrollStyle.Render(fmt.Sprintf("↑ %d more above ↑", m.viewportOffset)))
	}
	
	// Add visible lines
	result = append(result, visibleLines...)
	
	// Add bottom scroll indicator if needed
	if needsBottomIndicator {
		itemsBelow := totalItems - (m.viewportOffset + m.viewportHeight)
		scrollStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true)
		result = append(result, scrollStyle.Render(fmt.Sprintf("↓ %d more below ↓", itemsBelow)))
	}
	
	return strings.Join(result, "\n")
}

// renderGroupHeader renders a group header line
func (m *Model) renderGroupHeader(group *domain.Group, isExpanded bool, isSelected bool) string {
	arrow := "▶"
	if isExpanded {
		arrow = "▼"
	}
	
	count := 0
	for _, repoPath := range group.Repos {
		if _, ok := m.repositories[repoPath]; ok {
			count++
		}
	}
	
	// Check if this group is in search results
	isSearchMatch := false
	if m.searchQuery != "" && len(m.searchMatches) > 0 {
		currentIndex := m.getCurrentIndexForGroup(group.Name)
		for _, matchIndex := range m.searchMatches {
			if currentIndex == matchIndex {
				isSearchMatch = true
				break
			}
		}
	}
	
	// Build the content
	groupName := group.Name
	if isSearchMatch && m.searchQuery != "" && !strings.HasPrefix(m.searchQuery, "status:") {
		// Highlight matching text
		highlightStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true)
		groupName = m.highlightMatch(group.Name, m.searchQuery, highlightStyle, lipgloss.NewStyle())
	}
	
	content := fmt.Sprintf("%s %s (%d)", arrow, groupName, count)
	
	// Calculate padding for full-width highlighting
	contentWidth := lipgloss.Width(content)
	availableWidth := m.width - 4 // Account for outer padding
	if availableWidth < 1 {
		availableWidth = 80 // Fallback width
	}
	
	if isSelected {
		// Apply background to the entire line
		padding := availableWidth - contentWidth
		if padding < 0 {
			padding = 0
		}
		paddingStr := strings.Repeat(" ", padding)
		fullLine := content + paddingStr
		
		// Apply background style to the full line
		highlightStyle := lipgloss.NewStyle().Background(lipgloss.Color("238"))
		return highlightStyle.Render(fullLine)
	}
	
	return content
}

// renderRepository renders a repository line
func (m *Model) renderRepository(repo *domain.Repository, isSelected bool, indent int) string {
	// Check if this repo is in search results
	isSearchMatch := false
	if m.searchQuery != "" && len(m.searchMatches) > 0 {
		currentIndex := m.getCurrentIndexForRepo(repo.Path)
		for _, matchIndex := range m.searchMatches {
			if currentIndex == matchIndex {
				isSearchMatch = true
				break
			}
		}
	}
	// Selection indicator
	selectionIndicator := "[ ]"
	if m.selectedRepos[repo.Path] {
		selectionIndicator = "[✓]"
	}
	
	
	// Check if this repo is currently refreshing, fetching, or pulling
	isRefreshing := m.refreshingRepos[repo.Path]
	isFetching := m.fetchingRepos[repo.Path]
	isPulling := m.pullingRepos[repo.Path]
	
	// Status indicator
	var status string
	if isPulling {
		status = "⤓" // Pulling indicator (stronger down arrow)
	} else if isFetching {
		status = "⇣" // Fetching indicator (down arrow)
	} else if isRefreshing {
		status = "⟳" // Refreshing indicator
	} else if repo.Status.Error != "" {
		status = "⚠"
	} else if repo.Status.IsDirty || repo.Status.HasUntracked {
		status = "●"
	} else if repo.Status.Branch == "⋯" {
		status = "⋯"
	} else {
		status = "✓"
	}
	
	// Status color
	statusStyle := lipgloss.NewStyle()
	if isFetching {
		statusStyle = statusStyle.Foreground(lipgloss.Color("214")) // yellow for fetching
	} else if isRefreshing {
		statusStyle = statusStyle.Foreground(lipgloss.Color("51")) // cyan for refreshing
	} else if repo.Status.Error != "" {
		statusStyle = statusStyle.Foreground(lipgloss.Color("203")) // red
	} else if repo.Status.IsDirty || repo.Status.HasUntracked {
		statusStyle = statusStyle.Foreground(lipgloss.Color("214")) // yellow
	} else if repo.Status.Branch == "⋯" {
		statusStyle = statusStyle.Foreground(lipgloss.Color("241")) // gray for loading
	} else {
		statusStyle = statusStyle.Foreground(lipgloss.Color("78")) // green
	}
	
	// Apply status color even when selected
	if isSelected {
		statusStyle = statusStyle.Background(lipgloss.Color("238"))
	}
	
	// Branch info with color
	branchName := repo.Status.Branch
	if branchName == "" {
		branchName = "?"
	}
	
	// Apply branch color
	branchColor, isBold := m.branchColor(branchName)
	branchStyle := lipgloss.NewStyle().Foreground(branchColor)
	if isBold {
		branchStyle = branchStyle.Bold(true)
	}
	if isSelected {
		branchStyle = branchStyle.Background(lipgloss.Color("238"))
	}
	coloredBranch := branchStyle.Render(branchName)
	
	// Ahead/Behind indicators
	var aheadBehind string
	if repo.Status.AheadCount > 0 || repo.Status.BehindCount > 0 {
		aheadBehind = fmt.Sprintf(" (%d↑ %d↓)", repo.Status.AheadCount, repo.Status.BehindCount)
	}
	
	// Stash indicator
	var stashIndicator string
	if repo.Status.StashCount > 0 {
		stashIndicator = fmt.Sprintf(" 📦%d", repo.Status.StashCount)
	}
	
	// Build the line content
	indentStr := strings.Repeat("  ", indent)
	
	if isSelected {
		// When selected, we need to carefully construct the line with backgrounds
		bgColor := lipgloss.Color("238")
		
		// Build each component with its styling but without rendering yet
		var parts []string
		
		// Indent
		parts = append(parts, indentStr)
		
		// Selection indicator with background
		selectionStyle := lipgloss.NewStyle().Background(bgColor)
		parts = append(parts, selectionStyle.Render(selectionIndicator))
		
		// Space
		parts = append(parts, " ")
		
		// Status with its color and background
		parts = append(parts, statusStyle.Render(status))
		
		// Space
		parts = append(parts, " ")
		
		// Repo name with background and potential search highlight
		nameStyle := lipgloss.NewStyle().Background(bgColor)
		repoName := repo.Name
		if isSearchMatch && m.searchQuery != "" && !strings.HasPrefix(m.searchQuery, "status:") {
			// Highlight matching text
			highlightStyle := nameStyle.Copy().Foreground(lipgloss.Color("226")) // bright yellow
			repoName = m.highlightMatch(repo.Name, m.searchQuery, highlightStyle, nameStyle)
		}
		parts = append(parts, nameStyle.Render(repoName))
		
		// Opening paren with background
		parenStyle := lipgloss.NewStyle().Background(bgColor)
		parts = append(parts, parenStyle.Render(" ("))
		
		// Branch (already has background from earlier)
		parts = append(parts, coloredBranch)
		
		// Ahead/behind with closing paren
		if aheadBehind != "" {
			// aheadBehind already includes the space before it
			aheadBehindWithBg := lipgloss.NewStyle().Background(bgColor).Render(aheadBehind)
			parts = append(parts, aheadBehindWithBg)
		}
		
		// Closing paren with background
		parts = append(parts, parenStyle.Render(")"))
		
		// Stash indicator with background
		if stashIndicator != "" {
			stashStyle := lipgloss.NewStyle().Background(bgColor)
			parts = append(parts, stashStyle.Render(stashIndicator))
		}
		
		// Join all parts
		content := strings.Join(parts, "")
		
		// Calculate padding needed to fill the width
		contentWidth := lipgloss.Width(content)
		availableWidth := m.width - 4 // Account for outer padding
		if availableWidth < 1 {
			availableWidth = 80 // Fallback width
		}
		
		// Add padding to fill the entire row
		padding := availableWidth - contentWidth
		if padding < 0 {
			padding = 0
		}
		paddingStr := strings.Repeat(" ", padding)
		
		// Apply background to padding as well
		paddingStyle := lipgloss.NewStyle().Background(bgColor)
		return content + paddingStyle.Render(paddingStr)
	}
	
	// Not selected - render normally
	repoName := repo.Name
	if isSearchMatch && m.searchQuery != "" && !strings.HasPrefix(m.searchQuery, "status:") {
		// Highlight matching text
		highlightStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true)
		repoName = m.highlightMatch(repo.Name, m.searchQuery, highlightStyle, lipgloss.NewStyle())
	}
	
	content := fmt.Sprintf("%s%s %s %s (%s%s)%s", 
		indentStr,
		selectionIndicator,
		statusStyle.Render(status),
		repoName,
		coloredBranch,
		aheadBehind,
		stashIndicator,
	)
	
	return content
}

// branchColor returns the color and bold flag for a branch name
func (m *Model) branchColor(branchName string) (lipgloss.Color, bool) {
	// Main and master get special treatment - bold green
	if branchName == "main" || branchName == "master" {
		return lipgloss.Color("78"), true // bold green
	}
	
	// Special cases for loading and unknown
	if branchName == "⋯" || branchName == "?" {
		return lipgloss.Color("241"), false // gray, not bold
	}
	
	// Use a simple hash function to assign consistent colors to branch names
	var hash uint32
	for _, b := range branchName {
		hash = hash*31 + uint32(b)
	}
	
	// Map to a set of colors (avoiding red which might indicate errors)
	// Using a wider range of ANSI 256 colors for better variety
	colors := []lipgloss.Color{
		lipgloss.Color("51"),   // Cyan
		lipgloss.Color("214"),  // Yellow/Orange
		lipgloss.Color("33"),   // Blue
		lipgloss.Color("205"),  // Magenta/Pink
		lipgloss.Color("87"),   // Light Cyan
		lipgloss.Color("228"),  // Light Yellow
		lipgloss.Color("111"),  // Light Blue
		lipgloss.Color("213"),  // Light Magenta
		lipgloss.Color("45"),   // Turquoise
		lipgloss.Color("39"),   // Deep Sky Blue
		lipgloss.Color("171"),  // Purple
		lipgloss.Color("220"),  // Gold
		lipgloss.Color("208"),  // Dark Orange
		lipgloss.Color("159"),  // Pale Cyan
		lipgloss.Color("141"),  // Light Purple
		lipgloss.Color("117"),  // Sky Blue
		lipgloss.Color("183"),  // Plum
		lipgloss.Color("186"),  // Khaki
		lipgloss.Color("222"),  // Light Salmon
		lipgloss.Color("156"),  // Light Green
		lipgloss.Color("48"),   // Spring Green
		lipgloss.Color("85"),   // Sea Green
		lipgloss.Color("120"),  // Light Green
		lipgloss.Color("135"),  // Purple Blue
		lipgloss.Color("177"),  // Violet
		lipgloss.Color("215"),  // Sandy Brown
	}
	
	color := colors[hash%uint32(len(colors))]
	return color, false // regular weight
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
	count := len(m.orderedGroups) + len(m.getUngroupedRepos())
	for groupName, group := range m.groups {
		if m.expandedGroups[groupName] {
			count += len(group.Repos)
		}
	}
	return count - 1
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
	// Calculate total items using the same logic as renderRepositoryList
	totalItems := 0
	// Groups first
	for _, groupName := range m.orderedGroups {
		totalItems++ // Group header
		if m.expandedGroups[groupName] {
			group := m.groups[groupName]
			totalItems += len(group.Repos)
		}
	}
	// Then ungrouped repos
	totalItems += len(m.getUngroupedRepos())
	
	// If selected item is above viewport, scroll up
	if m.selectedIndex < m.viewportOffset {
		m.viewportOffset = m.selectedIndex
	}
	
	// If selected item is below viewport, we need to calculate the effective visible area
	// This must match the logic in renderRepositoryList exactly
	
	// Determine if we'll have scroll indicators using the same logic as rendering
	needsTopIndicator := m.viewportOffset > 0
	needsBottomIndicator := m.viewportOffset + m.viewportHeight < totalItems
	
	// Special case: if we're showing items but can't fit them all even without bottom indicator,
	// we still need the bottom indicator
	if !needsBottomIndicator && needsTopIndicator {
		// Check if all remaining items can fit
		remainingItems := totalItems - m.viewportOffset
		availableSpace := m.viewportHeight - 1 // -1 for top indicator
		if remainingItems > availableSpace {
			needsBottomIndicator = true
		}
	}
	
	// Calculate effective visible area (same as in renderRepositoryList)
	effectiveHeight := m.viewportHeight
	if needsTopIndicator {
		effectiveHeight--
	}
	if needsBottomIndicator {
		effectiveHeight--
	}
	
	// Ensure we have at least 1 line for content
	if effectiveHeight < 1 {
		effectiveHeight = 1
	}
	
	// Check if selected item is beyond the effective visible area
	// The rendering uses "len(visibleLines) < effectiveHeight" which means
	// it will render effectiveHeight items (0 through effectiveHeight-1)
	// So the last visible item is at viewportOffset + effectiveHeight - 1
	lastVisibleIndex := m.viewportOffset + effectiveHeight - 1
	
	if m.selectedIndex > lastVisibleIndex {
		// Selected item is below visible area, need to scroll down
		// Calculate how much to scroll
		scrollAmount := m.selectedIndex - lastVisibleIndex
		m.viewportOffset += scrollAmount
		
		// Ensure we don't scroll past the end
		maxOffset := totalItems - m.viewportHeight
		if maxOffset < 0 {
			maxOffset = 0
		}
		if m.viewportOffset > maxOffset {
			m.viewportOffset = maxOffset
			
			// Double-check: if we're at max offset and still can't see the selected item,
			// it means our effective height calculation is wrong
			// Force the viewport to show the last item
			if m.selectedIndex >= m.viewportOffset + effectiveHeight {
				// Adjust viewport to ensure selected item is visible
				m.viewportOffset = m.selectedIndex - effectiveHeight + 1
				if m.viewportOffset < 0 {
					m.viewportOffset = 0
				}
			}
		}
	}
	
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
	currentIndex := 0
	
	for _, name := range m.orderedGroups {
		if name == groupName {
			return currentIndex
		}
		currentIndex++
		
		if m.expandedGroups[name] {
			group := m.groups[name]
			currentIndex += len(group.Repos)
		}
	}
	
	return -1
}

// getCurrentIndexForRepo finds the current display index for a repo
func (m *Model) getCurrentIndexForRepo(repoPath string) int {
	currentIndex := 0
	
	// Check groups first
	for _, groupName := range m.orderedGroups {
		currentIndex++ // Group header
		
		if m.expandedGroups[groupName] {
			group := m.groups[groupName]
			for _, path := range group.Repos {
				if path == repoPath {
					return currentIndex
				}
				currentIndex++
			}
		}
	}
	
	// Check ungrouped repos
	ungroupedRepos := m.ungroupedRepos
	if len(ungroupedRepos) == 0 {
		ungroupedRepos = m.getUngroupedRepos()
	}
	for _, path := range ungroupedRepos {
		if path == repoPath {
			return currentIndex
		}
		currentIndex++
	}
	
	return -1
}

// highlightMatch highlights matching text within a string
func (m *Model) highlightMatch(text, query string, highlightStyle, normalStyle lipgloss.Style) string {
	lowerText := strings.ToLower(text)
	lowerQuery := strings.ToLower(query)
	
	index := strings.Index(lowerText, lowerQuery)
	if index == -1 {
		return text
	}
	
	// Build the highlighted string
	var result string
	if index > 0 {
		result += text[:index]
	}
	result += highlightStyle.Render(text[index:index+len(query)])
	if index+len(query) < len(text) {
		result += text[index+len(query):]
	}
	
	return result
}

// jumpToGroupBoundary jumps to the beginning or end of the current group
func (m *Model) jumpToGroupBoundary(toBeginning bool) {
	// Find which group we're currently in
	currentIndex := 0
	inGroup := false
	
	for _, groupName := range m.orderedGroups {
		groupHeaderIndex := currentIndex
		
		// Check if we're on the group header
		if currentIndex == m.selectedIndex {
			inGroup = true
			// On header - jump to first or last repo in group
			if m.expandedGroups[groupName] {
				group := m.groups[groupName]
				if len(group.Repos) > 0 {
					if toBeginning {
						// Jump to first repo
						m.selectedIndex = groupHeaderIndex + 1
					} else {
						// Jump to last repo
						m.selectedIndex = groupHeaderIndex + len(group.Repos)
					}
					m.ensureSelectedVisible()
					return
				}
			}
			return // Group is collapsed or empty
		}
		currentIndex++
		
		// Check repos in group if expanded
		if m.expandedGroups[groupName] {
			group := m.groups[groupName]
			groupFirstRepoIndex := currentIndex
			groupLastRepoIndex := currentIndex + len(group.Repos) - 1
			
			// Check if we're inside this group
			for range group.Repos {
				if currentIndex == m.selectedIndex {
					inGroup = true
					
					if toBeginning {
						// If at the first repo in the group, jump to previous group
						if currentIndex == groupFirstRepoIndex {
							// Jump to first repo of previous group
							m.jumpToPreviousGroupStart(groupName)
							return
						}
						// Otherwise jump to first repo in current group
						m.selectedIndex = groupFirstRepoIndex
						m.ensureSelectedVisible()
						return
					} else {
						// Check if we're already at the last repo in the group
						if currentIndex == groupLastRepoIndex {
							// Jump to last repo of next group
							m.jumpToNextGroupEnd(groupName)
							return
						}
						// Jump to last repo in group
						m.selectedIndex = groupLastRepoIndex
						m.ensureSelectedVisible()
						return
					}
				}
				currentIndex++
			}
		}
	}
	
	// Check ungrouped repos
	if !inGroup {
		ungroupedRepos := m.ungroupedRepos
		if len(ungroupedRepos) == 0 {
			ungroupedRepos = m.getUngroupedRepos()
		}
		
		if len(ungroupedRepos) > 0 {
			ungroupedStartIndex := currentIndex
			ungroupedEndIndex := currentIndex + len(ungroupedRepos) - 1
			
			// Check if we're in ungrouped section
			for i := 0; i < len(ungroupedRepos); i++ {
				if currentIndex == m.selectedIndex {
					if toBeginning {
						m.selectedIndex = ungroupedStartIndex
					} else {
						m.selectedIndex = ungroupedEndIndex
					}
					m.ensureSelectedVisible()
					return
				}
				currentIndex++
			}
		}
	}
}

// jumpToNextGroupEnd jumps to the last repo of the next group
func (m *Model) jumpToNextGroupEnd(currentGroupName string) {
	// Find current group index
	currentGroupIndex := -1
	for i, groupName := range m.orderedGroups {
		if groupName == currentGroupName {
			currentGroupIndex = i
			break
		}
	}
	
	// If we found the current group and there's a next group
	if currentGroupIndex != -1 && currentGroupIndex < len(m.orderedGroups)-1 {
		// Find the next expanded group with repos
		for i := currentGroupIndex + 1; i < len(m.orderedGroups); i++ {
			nextGroupName := m.orderedGroups[i]
			if m.expandedGroups[nextGroupName] {
				group := m.groups[nextGroupName]
				if len(group.Repos) > 0 {
					// Calculate the index of the last repo in this group
					currentIndex := 0
					for j, groupName := range m.orderedGroups {
						currentIndex++ // Group header
						if j < i {
							// Count all repos in previous groups
							if m.expandedGroups[groupName] {
								g := m.groups[groupName]
								currentIndex += len(g.Repos)
							}
						} else if j == i {
							// We're at the target group, add repos to get to last one
							currentIndex += len(group.Repos) - 1
							m.selectedIndex = currentIndex
							m.ensureSelectedVisible()
							return
						}
					}
				}
			}
		}
	}
	
	// If no next group found, check ungrouped repos
	ungroupedRepos := m.ungroupedRepos
	if len(ungroupedRepos) == 0 {
		ungroupedRepos = m.getUngroupedRepos()
	}
	if len(ungroupedRepos) > 0 {
		// Jump to last ungrouped repo
		currentIndex := 0
		// Count all groups and their repos
		for _, groupName := range m.orderedGroups {
			currentIndex++ // Group header
			if m.expandedGroups[groupName] {
				group := m.groups[groupName]
				currentIndex += len(group.Repos)
			}
		}
		// Add ungrouped repos
		currentIndex += len(ungroupedRepos) - 1
		m.selectedIndex = currentIndex
		m.ensureSelectedVisible()
	}
}

// jumpToPreviousGroupStart jumps to the first repo of the previous group
func (m *Model) jumpToPreviousGroupStart(currentGroupName string) {
	// Find current group index
	currentGroupIndex := -1
	for i, groupName := range m.orderedGroups {
		if groupName == currentGroupName {
			currentGroupIndex = i
			break
		}
	}
	
	// If we found the current group and there's a previous group
	if currentGroupIndex > 0 {
		// Find the previous expanded group with repos
		for i := currentGroupIndex - 1; i >= 0; i-- {
			prevGroupName := m.orderedGroups[i]
			if m.expandedGroups[prevGroupName] {
				group := m.groups[prevGroupName]
				if len(group.Repos) > 0 {
					// Calculate the index of the first repo in this group
					currentIndex := 0
					for j, groupName := range m.orderedGroups {
						currentIndex++ // Group header
						if j < i {
							// Count all repos in previous groups
							if m.expandedGroups[groupName] {
								g := m.groups[groupName]
								currentIndex += len(g.Repos)
							}
						} else if j == i {
							// We're at the target group, we're already at first repo
							m.selectedIndex = currentIndex
							m.ensureSelectedVisible()
							return
						}
					}
				}
			}
		}
	}
	
	// If no previous group found, stay where we are (we're at the first group)
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