package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	
	"gitagrip/internal/config"
	"gitagrip/internal/domain"
	"gitagrip/internal/eventbus"
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
		key.WithKeys("F"),
		key.WithHelp("F", "full scan"),
	)
	keyFetch = key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "fetch"),
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
)

// EventMsg wraps a domain event for the UI
type EventMsg struct {
	Event eventbus.DomainEvent
}

// Model represents the UI state
type Model struct {
	bus          eventbus.EventBus
	config       *config.Config
	repositories map[string]*domain.Repository // path -> repo
	groups       map[string]*domain.Group      // name -> group
	orderedRepos []string                      // ordered repo paths for display
	orderedGroups []string                     // ordered group names
	selectedIndex int                          // currently selected item
	expandedGroups map[string]bool             // which groups are expanded
	scanning      bool                         // whether scanning is in progress
	statusMessage string                       // status bar message
	width         int
	height        int
	showHelp      bool
	help          help.Model
}

// NewModel creates a new UI model
func NewModel(bus eventbus.EventBus, cfg *config.Config) *Model {
	m := &Model{
		bus:            bus,
		config:         cfg,
		repositories:   make(map[string]*domain.Repository),
		groups:         make(map[string]*domain.Group),
		orderedRepos:   make([]string, 0),
		orderedGroups:  make([]string, 0),
		expandedGroups: make(map[string]bool),
		help:           help.New(),
	}
	
	// Initialize groups from config
	for name, repoPaths := range cfg.Groups {
		m.groups[name] = &domain.Group{
			Name:  name,
			Repos: repoPaths,
		}
		m.expandedGroups[name] = true // Start with groups expanded
	}
	m.updateOrderedLists()
	
	return m
}

// Init returns an initial command
func (m *Model) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keyQuit):
			return m, tea.Quit
			
		case key.Matches(msg, keyUp):
			if m.selectedIndex > 0 {
				m.selectedIndex--
			}
			
		case key.Matches(msg, keyDown):
			maxIndex := m.getMaxIndex()
			if m.selectedIndex < maxIndex {
				m.selectedIndex++
			}
			
		case key.Matches(msg, keyRefresh):
			m.statusMessage = "Refreshing repository statuses..."
			// TODO: Trigger status refresh
			
		case key.Matches(msg, keyFullScan):
			m.statusMessage = "Starting full repository scan..."
			if m.bus != nil && m.config.BaseDir != "" {
				m.bus.Publish(eventbus.ScanRequestedEvent{
					Paths: []string{m.config.BaseDir},
				})
			}
			
		case key.Matches(msg, keyHelp):
			m.showHelp = !m.showHelp
		}
		
	case EventMsg:
		return m.handleEvent(msg.Event)
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
		
	case eventbus.StatusUpdatedEvent:
		// Update repository status
		if repo, ok := m.repositories[e.RepoPath]; ok {
			repo.Status = e.Status
		}
		
	case eventbus.ErrorEvent:
		m.statusMessage = fmt.Sprintf("Error: %s", e.Message)
		
	case eventbus.GroupAddedEvent:
		if _, exists := m.groups[e.Name]; !exists {
			m.groups[e.Name] = &domain.Group{
				Name:  e.Name,
				Repos: []string{},
			}
			m.expandedGroups[e.Name] = true
			m.updateOrderedLists()
		}
		
	case eventbus.ScanStartedEvent:
		m.scanning = true
		m.statusMessage = "Scanning for repositories..."
		
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
	content.WriteString("\n\n")
	
	// Repository list
	if len(m.repositories) == 0 && !m.scanning {
		dimStyle := lipgloss.NewStyle().Faint(true)
		content.WriteString(dimStyle.Render("No repositories found. Press F for full scan."))
	} else {
		content.WriteString(m.renderRepositoryList())
	}
	
	// Status bar
	content.WriteString("\n\n")
	if m.statusMessage != "" {
		statusStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
		content.WriteString(statusStyle.Render(m.statusMessage))
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
	var lines []string
	currentIndex := 0
	
	// Render ungrouped repositories first
	ungroupedRepos := m.getUngroupedRepos()
	if len(ungroupedRepos) > 0 {
		for _, repoPath := range ungroupedRepos {
			repo := m.repositories[repoPath]
			isSelected := currentIndex == m.selectedIndex
			lines = append(lines, m.renderRepository(repo, isSelected, 0))
			currentIndex++
		}
	}
	
	// Render groups
	for _, groupName := range m.orderedGroups {
		group := m.groups[groupName]
		isExpanded := m.expandedGroups[groupName]
		
		// Render group header
		isSelected := currentIndex == m.selectedIndex
		lines = append(lines, m.renderGroupHeader(group, isExpanded, isSelected))
		currentIndex++
		
		// Render group contents if expanded
		if isExpanded {
			for _, repoPath := range group.Repos {
				if repo, ok := m.repositories[repoPath]; ok {
					isSelected := currentIndex == m.selectedIndex
					lines = append(lines, m.renderRepository(repo, isSelected, 1))
					currentIndex++
				}
			}
		}
	}
	
	return strings.Join(lines, "\n")
}

// renderGroupHeader renders a group header line
func (m *Model) renderGroupHeader(group *domain.Group, isExpanded bool, isSelected bool) string {
	arrow := "▶"
	if isExpanded {
		arrow = "▼"
	}
	
	style := lipgloss.NewStyle()
	if isSelected {
		style = style.Background(lipgloss.Color("238"))
	}
	
	count := 0
	for _, repoPath := range group.Repos {
		if _, ok := m.repositories[repoPath]; ok {
			count++
		}
	}
	
	return style.Render(fmt.Sprintf("%s %s (%d)", arrow, group.Name, count))
}

// renderRepository renders a repository line
func (m *Model) renderRepository(repo *domain.Repository, isSelected bool, indent int) string {
	style := lipgloss.NewStyle()
	if isSelected {
		style = style.Background(lipgloss.Color("238"))
	}
	
	// Status indicator
	var status string
	if repo.Status.Error != "" {
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
	if repo.Status.Error != "" {
		statusStyle = statusStyle.Foreground(lipgloss.Color("203")) // red
	} else if repo.Status.IsDirty || repo.Status.HasUntracked {
		statusStyle = statusStyle.Foreground(lipgloss.Color("214")) // yellow
	} else if repo.Status.Branch == "⋯" {
		statusStyle = statusStyle.Foreground(lipgloss.Color("241")) // gray for loading
	} else {
		statusStyle = statusStyle.Foreground(lipgloss.Color("78")) // green
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
	coloredBranch := branchStyle.Render(branchName)
	
	// Ahead/Behind indicators
	var aheadBehind string
	if repo.Status.AheadCount > 0 || repo.Status.BehindCount > 0 {
		aheadBehind = fmt.Sprintf(" (%d↑ %d↓)", repo.Status.AheadCount, repo.Status.BehindCount)
	}
	
	// Build the line
	indentStr := strings.Repeat("  ", indent)
	line := fmt.Sprintf("%s%s %s (%s%s)", 
		indentStr,
		statusStyle.Render(status),
		repo.Name,
		coloredBranch,
		aheadBehind,
	)
	
	return style.Render(line)
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
	colors := []lipgloss.Color{
		lipgloss.Color("51"),  // Cyan
		lipgloss.Color("214"), // Yellow
		lipgloss.Color("33"),  // Blue
		lipgloss.Color("205"), // Magenta
		lipgloss.Color("87"),  // LightCyan
		lipgloss.Color("228"), // LightYellow
		lipgloss.Color("111"), // LightBlue
		lipgloss.Color("213"), // LightMagenta
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
	sort.Strings(m.orderedRepos)
	
	// Update ordered groups
	m.orderedGroups = make([]string, 0, len(m.groups))
	for name := range m.groups {
		m.orderedGroups = append(m.orderedGroups, name)
	}
	sort.Strings(m.orderedGroups)
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
	count := len(m.getUngroupedRepos()) + len(m.orderedGroups)
	for groupName, group := range m.groups {
		if m.expandedGroups[groupName] {
			count += len(group.Repos)
		}
	}
	return count - 1
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
		{keyRefresh, keyFullScan, keyFetch, keyLog},
		{keyHelp, keyQuit},
	}
}