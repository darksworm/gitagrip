package ui

import (
	"fmt"
	"log"
	"time"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"

	"gitagrip/internal/config"
	"gitagrip/internal/domain"
	"gitagrip/internal/eventbus"
	"gitagrip/internal/logic"
	"gitagrip/internal/ui/coordinator"
	"gitagrip/internal/ui/input"
	inputtypes "gitagrip/internal/ui/input/types"
	"gitagrip/internal/ui/services/events"
	"gitagrip/internal/ui/services/navigation"
	"gitagrip/internal/ui/views"
)

// Model is the refactored model using services
type Model struct {
	// Core dependencies
	config      *config.Config
	configService config.ConfigService
	store       logic.RepositoryStore
	groupStore  logic.GroupStore
	eventBus    eventbus.EventBus
	coordinator *coordinator.Coordinator
	
	// UI components
	inputHandler *input.Handler
	renderer     *views.Renderer
	
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
	configService config.ConfigService,
	store logic.RepositoryStore,
	groupStore logic.GroupStore,
	eventBus eventbus.EventBus,
	eventChan <-chan interface{},
) *Model {
	log.Printf("NewModel: Starting creation")
	
	// Create UI event bus for internal service communication
	log.Printf("NewModel: Creating UI event bus")
	uiEventBus := events.NewBus()
	
	// Create coordinator with all services
	log.Printf("NewModel: Creating coordinator")
	coord := coordinator.NewCoordinator(uiEventBus, store, groupStore)
	
	// Set up groups save function
	coord.Groups.SetSaveFunction(func() {
		// Convert groups to config format and save
		groupsConfig := make(map[string][]string)
		for _, group := range groupStore.GetAllGroups() {
			groupsConfig[group.Name] = group.Repos
		}
		cfg.Groups = groupsConfig
		
		if err := configService.Save(cfg); err != nil {
			log.Printf("Error saving config: %v", err)
		}
	})
	
	// Initialize expanded groups from existing data
	for name := range groupStore.GetAllGroups() {
		coord.Groups.ToggleExpanded(name) // Default to expanded
	}
	
	// Create UI components
	log.Printf("NewModel: Creating renderer")
	renderer := views.NewRenderer(cfg.UISettings.ShowAheadBehind)
	
	log.Printf("NewModel: Creating model struct")
	m := &Model{
		config:       cfg,
		configService: configService,
		store:        store,
		groupStore:   groupStore,
		eventBus:     eventBus,
		coordinator:  coord,
		inputHandler: input.New(),
		renderer:     renderer,
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
		ctx := &contextAdapter{model: m}
		actions, cmd := m.inputHandler.HandleKey(msg, ctx)
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

	case EventMsg:
		// Handle domain events from external sources
		m.handleDomainEvent(msg.Event)

	}

	// Always tick for animations
	cmds = append(cmds, tickCmd())
	
	return m, tea.Batch(cmds...)
}

// View renders the UI
func (m *Model) View() string {
	// Prevent nil pointer panics
	if m == nil {
		return "Model is nil"
	}
	if m.renderer == nil {
		return "Renderer is nil"
	}
	
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
		// Type assert data to string if needed
		data := ""
		if s, ok := a.Data.(string); ok {
			data = s
		}
		m.inputHandler.ChangeMode(a.Mode, data)
		
	case inputtypes.QuitAction:
		if a.Force || !m.coordinator.Selection.HasSelection() {
			return tea.Quit
		}
		
	case inputtypes.SubmitTextAction:
		// Handle input submission
		if cmd := m.handleInputSubmit(a); cmd != nil {
			return cmd
		}
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
			// Use domain event bus event type
			m.eventBus.Publish(eventbus.ScanRequestedEvent{})
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
		m.eventBus.Publish(eventbus.StatusRefreshRequestedEvent{RepoPaths: []string{path}})
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
		m.eventBus.Publish(eventbus.FetchRequestedEvent{RepoPaths: []string{path}})
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
		m.eventBus.Publish(eventbus.PullRequestedEvent{RepoPaths: []string{path}})
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

// handleInputSubmit processes input submission
func (m *Model) handleInputSubmit(a inputtypes.SubmitTextAction) tea.Cmd {
	text := a.Text
	switch m.inputHandler.GetMode() {
	case inputtypes.ModeSearch:
		m.coordinator.Search.StartSearch(text)
		
	case inputtypes.ModeFilter:
		// Filter is handled by view state building
		
	case inputtypes.ModeNewGroup:
		if text != "" && m.coordinator.Selection.HasSelection() {
			selected := m.coordinator.Selection.GetSelected()
			m.coordinator.Groups.CreateGroup(text, selected)
			m.coordinator.Selection.DeselectAll()
		}
		
	case inputtypes.ModeMoveToGroup:
		if text != "" {
			var repos []string
			if m.coordinator.Selection.HasSelection() {
				repos = m.coordinator.Selection.GetSelected()
			} else if path := m.coordinator.GetCurrentRepositoryPath(); path != "" {
				repos = []string{path}
			}
			
			if len(repos) > 0 {
				m.coordinator.Groups.MoveReposToGroup(repos, text)
				m.coordinator.Selection.DeselectAll()
			}
		}
		
	case inputtypes.ModeDeleteConfirm:
		groupName := m.inputHandler.GetModeData()
		if text == "y" && groupName != "" {
			m.coordinator.Groups.DeleteGroup(groupName)
		}
		
	case inputtypes.ModeSort:
		// Sort mode selection
		switch text {
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
	m.eventBus.Subscribe("logic.RepositoryDiscoveredEvent", func(e eventbus.DomainEvent) {
		if adapter, ok := e.(eventAdapter); ok {
			if event, ok := adapter.event.(logic.RepositoryDiscoveredEvent); ok {
				if event.Repository != nil && event.Repository.Path != "" {
					m.state.Repositories[event.Repository.Path] = event.Repository
					m.coordinator.UpdateOrderedLists()
				}
			}
		}
	})
	
	m.eventBus.Subscribe("logic.RepositoryUpdatedEvent", func(e eventbus.DomainEvent) {
		if adapter, ok := e.(eventAdapter); ok {
			if event, ok := adapter.event.(logic.RepositoryUpdatedEvent); ok {
				if event.Repository != nil && event.Repository.Path != "" {
					m.state.Repositories[event.Repository.Path] = event.Repository
				}
			}
		}
	})
	
	// Status messages
	m.eventBus.Subscribe("logic.ScanStartedEvent", func(e eventbus.DomainEvent) {
		m.state.StatusMessage = "Scanning for repositories..."
	})
	
	m.eventBus.Subscribe("logic.ScanCompletedEvent", func(e eventbus.DomainEvent) {
		if adapter, ok := e.(eventAdapter); ok {
			if event, ok := adapter.event.(logic.ScanCompletedEvent); ok {
				m.state.StatusMessage = fmt.Sprintf("Scan completed: %d repositories found", event.Count)
			}
		}
	})
}

// handleDomainEvent processes domain events
func (m *Model) handleDomainEvent(event interface{}) {
	// Wrap the event in an adapter and publish to event bus
	adapter := eventAdapter{event: event}
	m.eventBus.Publish(adapter)
}

// buildViewState creates view state from services
func (m *Model) buildViewState() views.ViewState {
	// Default values to prevent panics
	if m.coordinator == nil {
		log.Printf("ERROR: coordinator is nil in buildViewState")
		return views.ViewState{}
	}
	
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
		ShowAheadBehind: m.config.UISettings.ShowAheadBehind,
		HelpModel:       m.state.HelpModel,
		DeleteTarget:    m.inputHandler.GetModeData(),
		TextInput:       func() string {
			if ti := m.inputHandler.GetTextInput(); ti != nil {
				return ti.Value()
			}
			return ""
		}(),
		InputMode:       string(m.inputHandler.GetMode()),
		UngroupedRepos:  m.coordinator.Query.GetUngroupedRepos(),
	}
}

// Helper methods
func (m *Model) showGitLog(repo *domain.Repository) {
	// TODO: Implement git log functionality
	m.state.LogContent = "Git log not yet implemented"
	m.state.ShowLog = true
}

func (m *Model) showRepositoryInfo(repo *domain.Repository) {
	// Format repository info
	info := fmt.Sprintf(
		"Repository: %s\n"+
			"Path: %s\n"+
			"Branch: %s\n"+
			"Status: %s\n"+
			"Ahead: %d, Behind: %d\n"+
			"Stashes: %d",
		repo.Name,
		repo.Path,
		repo.Status.Branch,
		getStatusText(repo.Status),
		repo.Status.AheadCount,
		repo.Status.BehindCount,
		repo.Status.StashCount,
	)
	
	m.state.InfoContent = info
	m.state.ShowInfo = true
}

// getStatusText returns a text description of the repository status
func getStatusText(status domain.RepoStatus) string {
	if status.Error != "" {
		return "Error: " + status.Error
	}
	if status.IsDirty {
		return "Modified"
	}
	if status.HasUntracked {
		return "Untracked files"
	}
	return "Clean"
}

// contextAdapter implements input.Context for the input handler
type contextAdapter struct {
	model *Model
}

func (c *contextAdapter) CurrentIndex() int {
	return c.model.coordinator.Navigation.GetCursor()
}

func (c *contextAdapter) TotalItems() int {
	return c.model.coordinator.Query.GetMaxIndex() + 1
}

func (c *contextAdapter) HasSelection() bool {
	return c.model.coordinator.Selection.HasSelection()
}

func (c *contextAdapter) SelectedCount() int {
	return c.model.coordinator.Selection.GetCount()
}

func (c *contextAdapter) CurrentRepositoryPath() string {
	return c.model.coordinator.GetCurrentRepositoryPath()
}

func (c *contextAdapter) GetRepoPathAtIndex(index int) string {
	return c.model.coordinator.Query.GetRepositoryPathAtIndex(index)
}

func (c *contextAdapter) IsOnGroup() bool {
	return c.model.coordinator.IsOnGroup()
}

func (c *contextAdapter) CurrentGroupName() string {
	return c.model.coordinator.GetCurrentGroupName()
}

func (c *contextAdapter) SearchQuery() string {
	return c.model.coordinator.Search.GetQuery()
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

// eventAdapter wraps logic events to implement DomainEvent interface
type eventAdapter struct {
	event interface{}
}

func (e eventAdapter) Type() eventbus.EventType {
	switch e.event.(type) {
	case logic.RepositoryDiscoveredEvent:
		return "logic.RepositoryDiscoveredEvent"
	case logic.RepositoryUpdatedEvent:
		return "logic.RepositoryUpdatedEvent"
	case logic.ScanStartedEvent:
		return "logic.ScanStartedEvent"
	case logic.ScanCompletedEvent:
		return "logic.ScanCompletedEvent"
	default:
		return "unknown"
	}
}