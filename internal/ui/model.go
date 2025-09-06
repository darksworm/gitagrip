package ui

import (
	"fmt"
	"log"
	"time"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"gitagrip/internal/config"
	"gitagrip/internal/domain"
	"gitagrip/internal/eventbus"
	"gitagrip/internal/git"
	"gitagrip/internal/logic"
	"gitagrip/internal/ui/coordinator"
	"gitagrip/internal/ui/input"
	inputtypes "gitagrip/internal/ui/input/types"
	"gitagrip/internal/ui/services/navigation"
	"gitagrip/internal/ui/viewmodels"
	"gitagrip/internal/ui/views"
)

// Model is the refactored model using services
type Model struct {
	// Core dependencies
	config      *config.Config
	store       logic.RepositoryStore
	groupStore  logic.GroupStore
	eventBus    eventbus.EventBus
	coordinator *coordinator.Coordinator
	
	// UI components
	inputHandler *input.Handler
	renderer     *views.Renderer
	transformer  *viewmodels.ViewModelTransformer
	
	// UI state (minimal - most state is in services)
	state UIState
	
	// Channels
	eventChan <-chan interface{}
}

// UIState contains only UI-specific state
type UIState struct {
	Width         int
	Height        int
	ShowHelp      bool
	ShowLog       bool
	LogContent    string
	ShowInfo      bool
	InfoContent   string
	StatusMessage string
	Repositories  map[string]*domain.Repository // Cache for rendering
	HelpModel     help.Model
}

// NewModel creates a new refactored model
func NewModel(
	cfg *config.Config,
	store logic.RepositoryStore,
	groupStore logic.GroupStore,
	eventBus eventbus.EventBus,
	eventChan <-chan interface{},
) *Model {
	// Create coordinator with all services
	coord := coordinator.NewCoordinator(eventBus, store, groupStore)
	
	// Set up groups save function
	coord.Groups.SetSaveFunction(func() {
		// Convert groups to config format and save
		groupsConfig := make(map[string][]string)
		for _, group := range groupStore.GetAllGroups() {
			groupsConfig[group.Name] = group.Repos
		}
		cfg.Groups = groupsConfig
		
		if err := config.SaveConfig(cfg); err != nil {
			log.Printf("Error saving config: %v", err)
		}
	})
	
	// Initialize expanded groups from existing data
	for name := range groupStore.GetAllGroups() {
		coord.Groups.ToggleExpanded(name) // Default to expanded
	}
	
	// Create UI components
	renderer := views.NewRenderer(cfg.UI.ShowAheadBehind)
	transformer := viewmodels.NewViewModelTransformer(cfg.UI.ShowAheadBehind)
	
	m := &Model{
		config:       cfg,
		store:        store,
		groupStore:   groupStore,
		eventBus:     eventBus,
		coordinator:  coord,
		inputHandler: input.NewHandler(),
		renderer:     renderer,
		transformer:  transformer,
		state: UIState{
			ShowHelp:     false,
			Repositories: make(map[string]*domain.Repository),
			HelpModel:    help.New(),
		},
		eventChan: eventChan,
	}
	
	// Update coordinator with initial data
	m.coordinator.UpdateOrderedLists()
	
	// Subscribe to service events
	m.subscribeToEvents()
	
	return m
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.waitForEvent(),
		m.inputHandler.Init(),
		tea.EnterAltScreen,
	)
}

// Update handles messages
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.state.Width = msg.Width
		m.state.Height = msg.Height
		m.coordinator.SetViewportHeight(msg.Height)
		m.state.HelpModel.Width = msg.Width

	case tea.KeyMsg:
		// Let input handler process the key
		actions, cmd := m.inputHandler.HandleKey(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		
		// Process resulting actions
		for _, action := range actions {
			if cmd := m.handleAction(action); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

	case eventReceivedMsg:
		// Handle domain events
		m.handleDomainEvent(msg.event)
		cmds = append(cmds, m.waitForEvent())

	case inputtypes.InputCompleteMsg:
		// Handle input completion
		if cmd := m.handleInputComplete(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// Always tick for animations
	cmds = append(cmds, tickCmd())
	
	return m, tea.Batch(cmds...)
}

// View renders the UI
func (m *Model) View() string {
	// Build view state from services
	viewState := m.buildViewState()
	
	// Render using the views package
	return m.renderer.Render(viewState)
}

// handleAction processes an action from the input handler
func (m *Model) handleAction(action inputtypes.Action) tea.Cmd {
	switch a := action.(type) {
	case inputtypes.NavigateAction:
		m.handleNavigate(a)
		
	case inputtypes.SelectAction:
		m.handleSelect(a)
		
	case inputtypes.SelectAllAction:
		m.handleSelectAll()
		
	case inputtypes.DeselectAllAction:
		m.coordinator.Selection.DeselectAll()
		
	case inputtypes.ToggleGroupAction:
		m.handleToggleGroup()
		
	case inputtypes.RefreshAction:
		return m.handleRefresh(a)
		
	case inputtypes.FetchAction:
		return m.handleFetch()
		
	case inputtypes.PullAction:
		return m.handlePull()
		
	case inputtypes.OpenLogAction:
		m.handleOpenLog()
		
	case inputtypes.ToggleHelpAction:
		m.state.ShowHelp = !m.state.ShowHelp
		
	case inputtypes.ToggleInfoAction:
		m.handleToggleInfo()
		
	case inputtypes.SearchNavigateAction:
		m.handleSearchNavigate(a)
		
	case inputtypes.ChangeModeAction:
		m.inputHandler.ChangeMode(a.Mode, a.Data)
		
	case inputtypes.QuitAction:
		if a.Force || !m.coordinator.Selection.HasSelection() {
			return tea.Quit
		}
		
	case inputtypes.SubmitAction:
		// Input completion will be handled via InputCompleteMsg
	}
	
	return nil
}

// handleNavigate processes navigation
func (m *Model) handleNavigate(a inputtypes.NavigateAction) {
	switch a.Direction {
	case "up":
		m.coordinator.Navigation.Navigate(navigation.DirectionUp)
	case "down":
		m.coordinator.Navigation.Navigate(navigation.DirectionDown)
	case "left":
		if m.coordinator.IsOnGroup() {
			groupName := m.coordinator.GetCurrentGroupName()
			m.coordinator.Groups.ToggleExpanded(groupName)
		}
	case "right":
		if m.coordinator.IsOnGroup() {
			groupName := m.coordinator.GetCurrentGroupName()
			if !m.coordinator.Groups.IsExpanded(groupName) {
				m.coordinator.Groups.ToggleExpanded(groupName)
			}
		}
	case "pageup":
		m.coordinator.Navigation.Navigate(navigation.DirectionPageUp)
	case "pagedown":
		m.coordinator.Navigation.Navigate(navigation.DirectionPageDown)
	case "home":
		m.coordinator.Navigation.Navigate(navigation.DirectionHome)
	case "end":
		m.coordinator.Navigation.Navigate(navigation.DirectionEnd)
	}
}

// handleSelect processes selection
func (m *Model) handleSelect(a inputtypes.SelectAction) {
	if a.Index == -1 {
		// Toggle at current position
		m.coordinator.Selection.Toggle(m.coordinator.GetCurrentIndex())
	} else {
		m.coordinator.Selection.Toggle(a.Index)
	}
}

// handleSelectAll selects all visible repositories
func (m *Model) handleSelectAll() {
	paths := m.coordinator.Query.GetVisibleRepositoryPaths()
	m.coordinator.Selection.SelectAll(paths)
}

// handleToggleGroup toggles group expansion
func (m *Model) handleToggleGroup() {
	if m.coordinator.IsOnGroup() {
		groupName := m.coordinator.GetCurrentGroupName()
		m.coordinator.Groups.ToggleExpanded(groupName)
	}
}

// handleRefresh refreshes repositories
func (m *Model) handleRefresh(a inputtypes.RefreshAction) tea.Cmd {
	if a.All {
		// Full rescan
		return func() tea.Msg {
			m.eventBus.Publish(logic.RescanRequestedEvent{})
			return nil
		}
	}
	
	// Refresh selected or current
	var reposToRefresh []string
	if m.coordinator.Selection.HasSelection() {
		reposToRefresh = m.coordinator.Selection.GetSelected()
	} else if path := m.coordinator.GetCurrentRepositoryPath(); path != "" {
		reposToRefresh = []string{path}
	}
	
	for _, path := range reposToRefresh {
		m.eventBus.Publish(logic.RefreshRequestedEvent{Path: path})
	}
	
	return nil
}

// handleFetch fetches repositories
func (m *Model) handleFetch() tea.Cmd {
	var reposToFetch []string
	
	if m.coordinator.Selection.HasSelection() {
		reposToFetch = m.coordinator.Selection.GetSelected()
	} else if path := m.coordinator.GetCurrentRepositoryPath(); path != "" {
		reposToFetch = []string{path}
	}
	
	for _, path := range reposToFetch {
		m.eventBus.Publish(git.FetchRequestedEvent{Path: path})
	}
	
	return nil
}

// handlePull pulls repositories
func (m *Model) handlePull() tea.Cmd {
	var reposToPull []string
	
	if m.coordinator.Selection.HasSelection() {
		reposToPull = m.coordinator.Selection.GetSelected()
	} else if path := m.coordinator.GetCurrentRepositoryPath(); path != "" {
		reposToPull = []string{path}
	}
	
	for _, path := range reposToPull {
		m.eventBus.Publish(git.PullRequestedEvent{Path: path})
	}
	
	return nil
}

// handleOpenLog opens git log
func (m *Model) handleOpenLog() {
	if !m.coordinator.IsOnGroup() {
		if repo := m.coordinator.GetCurrentRepository(); repo != nil {
			m.showGitLog(repo)
		}
	}
}

// handleToggleInfo toggles info display
func (m *Model) handleToggleInfo() {
	if m.state.ShowInfo {
		m.state.ShowInfo = false
		m.state.InfoContent = ""
	} else if repo := m.coordinator.GetCurrentRepository(); repo != nil {
		m.showRepositoryInfo(repo)
	}
}

// handleSearchNavigate navigates search results
func (m *Model) handleSearchNavigate(a inputtypes.SearchNavigateAction) {
	if a.Direction == "next" {
		m.coordinator.Search.NavigateNext()
	} else if a.Direction == "prev" {
		m.coordinator.Search.NavigatePrevious()
	}
}

// handleInputComplete processes completed input
func (m *Model) handleInputComplete(msg inputtypes.InputCompleteMsg) tea.Cmd {
	switch m.inputHandler.GetMode() {
	case inputtypes.ModeSearch:
		m.coordinator.Search.StartSearch(msg.Text)
		
	case inputtypes.ModeFilter:
		// Filter is handled by view state building
		
	case inputtypes.ModeNewGroup:
		if msg.Text != "" && m.coordinator.Selection.HasSelection() {
			selected := m.coordinator.Selection.GetSelected()
			m.coordinator.Groups.CreateGroup(msg.Text, selected)
			m.coordinator.Selection.DeselectAll()
		}
		
	case inputtypes.ModeMoveToGroup:
		if msg.Text != "" {
			var repos []string
			if m.coordinator.Selection.HasSelection() {
				repos = m.coordinator.Selection.GetSelected()
			} else if path := m.coordinator.GetCurrentRepositoryPath(); path != "" {
				repos = []string{path}
			}
			
			if len(repos) > 0 {
				m.coordinator.Groups.MoveReposToGroup(repos, msg.Text)
				m.coordinator.Selection.DeselectAll()
			}
		}
		
	case inputtypes.ModeDeleteConfirm:
		groupName := m.inputHandler.GetModeData()
		if msg.Text == "y" && groupName != "" {
			m.coordinator.Groups.DeleteGroup(groupName)
		}
		
	case inputtypes.ModeSort:
		// Sort mode selection
		switch msg.Text {
		case "n":
			m.coordinator.Sorting.SetMode(logic.SortByName)
		case "s":
			m.coordinator.Sorting.SetMode(logic.SortByStatus)
		case "b":
			m.coordinator.Sorting.SetMode(logic.SortByBranch)
		case "p":
			m.coordinator.Sorting.SetMode(logic.SortByPath)
		}
		m.coordinator.UpdateOrderedLists()
	}
	
	// Return to normal mode
	m.inputHandler.ChangeMode(inputtypes.ModeNormal, "")
	return nil
}

// Event handling
func (m *Model) subscribeToEvents() {
	// Repository updates
	m.eventBus.Subscribe(logic.RepositoryDiscoveredEvent{}, func(e interface{}) {
		event := e.(logic.RepositoryDiscoveredEvent)
		m.state.Repositories[event.Repository.Path] = event.Repository
		m.coordinator.UpdateOrderedLists()
	})
	
	m.eventBus.Subscribe(logic.RepositoryUpdatedEvent{}, func(e interface{}) {
		event := e.(logic.RepositoryUpdatedEvent)
		m.state.Repositories[event.Repository.Path] = event.Repository
	})
	
	// Status messages
	m.eventBus.Subscribe(logic.ScanStartedEvent{}, func(e interface{}) {
		m.state.StatusMessage = "Scanning for repositories..."
	})
	
	m.eventBus.Subscribe(logic.ScanCompletedEvent{}, func(e interface{}) {
		event := e.(logic.ScanCompletedEvent)
		m.state.StatusMessage = fmt.Sprintf("Scan completed: %d repositories found", event.Count)
	})
}

// handleDomainEvent processes domain events
func (m *Model) handleDomainEvent(event interface{}) {
	// The event bus subscriptions handle most events
	// This is for any additional UI-specific handling
}

// buildViewState creates view state from services
func (m *Model) buildViewState() views.ViewState {
	// Get current states from services
	cursor := m.coordinator.Navigation.GetCursor()
	viewport := m.coordinator.Navigation.GetViewportOffset()
	viewportHeight := m.coordinator.Navigation.GetViewportHeight()
	
	// Build selection map
	selectedRepos := make(map[string]bool)
	for _, path := range m.coordinator.Selection.GetSelected() {
		selectedRepos[path] = true
	}
	
	// Get groups for display
	groups := make(map[string]*domain.Group)
	orderedGroups := []string{}
	for _, g := range m.groupStore.GetAllGroups() {
		groups[g.Name] = g
		orderedGroups = append(orderedGroups, g.Name)
	}
	m.coordinator.Sorting.SortGroups(orderedGroups)
	
	// Get operation states
	refreshing := make(map[string]bool)
	fetching := make(map[string]bool)
	pulling := make(map[string]bool)
	
	// TODO: Track operation states from git service events
	
	return views.ViewState{
		Width:           m.state.Width,
		Height:          m.state.Height,
		Repositories:    m.state.Repositories,
		Groups:          groups,
		OrderedGroups:   orderedGroups,
		SelectedIndex:   cursor,
		SelectedRepos:   selectedRepos,
		RefreshingRepos: refreshing,
		FetchingRepos:   fetching,
		PullingRepos:    pulling,
		ExpandedGroups:  m.coordinator.Groups.GetExpandedGroups(),
		Scanning:        false, // TODO: Track from scan events
		StatusMessage:   m.state.StatusMessage,
		ShowHelp:        m.state.ShowHelp,
		ShowLog:         m.state.ShowLog,
		LogContent:      m.state.LogContent,
		ShowInfo:        m.state.ShowInfo,
		InfoContent:     m.state.InfoContent,
		ViewportOffset:  viewport,
		ViewportHeight:  viewportHeight,
		SearchQuery:     m.coordinator.Search.GetQuery(),
		FilterQuery:     m.inputHandler.GetFilterQuery(),
		IsFiltered:      m.inputHandler.GetFilterQuery() != "",
		ShowAheadBehind: m.config.UI.ShowAheadBehind,
		HelpModel:       m.state.HelpModel,
		DeleteTarget:    m.inputHandler.GetModeData(),
		TextInput:       m.inputHandler.GetTextInput(),
		InputMode:       string(m.inputHandler.GetMode()),
		UngroupedRepos:  m.coordinator.Query.GetUngroupedRepos(),
	}
}

// Helper methods
func (m *Model) showGitLog(repo *domain.Repository) {
	log, err := git.GetGitLog(repo.Path, 20)
	if err != nil {
		m.state.LogContent = fmt.Sprintf("Error getting git log: %v", err)
	} else {
		m.state.LogContent = log
	}
	m.state.ShowLog = true
}

func (m *Model) showRepositoryInfo(repo *domain.Repository) {
	info := m.transformer.FormatRepositoryInfo(repo)
	m.state.InfoContent = info
	m.state.ShowInfo = true
}

// Event handling
type eventReceivedMsg struct {
	event interface{}
}

func (m *Model) waitForEvent() tea.Cmd {
	return func() tea.Msg {
		event := <-m.eventChan
		return eventReceivedMsg{event: event}
	}
}

// Tick for animations
func tickCmd() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(time.Time) tea.Msg {
		return nil
	})
}