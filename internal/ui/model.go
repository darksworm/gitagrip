package ui

import (
	"fmt"
	"log"
	"path/filepath"
	"sync"
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
	
	// Debounce timer for ordered list updates
	updateTimer *time.Timer
	updateMutex sync.Mutex
}

// UIState contains only UI-specific state
type UIState struct {
	Width           int
	Height          int
	ShowHelp        bool
	ShowLog         bool
	LogContent      string
	ShowInfo        bool
	InfoContent     string
	StatusMessage   string
	Repositories    map[string]*domain.Repository // Cache for rendering
	RepoMutex       sync.RWMutex                   // Protects Repositories map
	HelpModel       help.Model
	RefreshingRepos map[string]bool
	FetchingRepos   map[string]bool
	PullingRepos    map[string]bool
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
			ShowHelp:        false,
			Repositories:    make(map[string]*domain.Repository),
			HelpModel:       help.New(),
			RefreshingRepos: make(map[string]bool),
			FetchingRepos:   make(map[string]bool),
			PullingRepos:    make(map[string]bool),
		},
		eventChan: eventChan,
	}
	
	// Update coordinator with initial data
	m.coordinator.UpdateOrderedLists() // Initial load, no debounce needed
	
	// Subscribe to service events
	m.subscribeToEvents()
	
	return m
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	// Trigger initial scan after UI is ready
	if m.config.BaseDir != "" {
		m.eventBus.Publish(eventbus.ScanRequestedEvent{
			Paths: []string{m.config.BaseDir},
		})
	}
	
	return tea.Batch(
		m.waitForEvent(),
		m.inputHandler.Init(),
		tea.EnterAltScreen,
	)
}

// Update handles messages
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Only log non-tick messages to reduce noise
	switch msg.(type) {
	case tickMsg, nil:
		// Don't log these frequent messages
	default:
		log.Printf("[UPDATE] Starting Update with msg type: %T", msg)
	}
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tickMsg:
		// On tick, check for events
		cmds = append(cmds, m.waitForEvent())
		
	case nil:
		// Nil message from waitForEvent - don't do anything special
		// The tick will handle re-checking
		
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
		// Handle single domain event via p.Send()
		switch e := msg.event.(type) {
		case logic.StatusUpdatedEvent:
			log.Printf("[UPDATE] Received StatusUpdatedEvent via p.Send() for %s with branch: %s", 
				filepath.Base(e.Path), e.Status.Branch)
		default:
			log.Printf("[UPDATE] Received event via p.Send(): %T", msg.event)
		}
		m.handleDomainEvent(msg.event)
		log.Printf("[UPDATE] Finished handleDomainEvent")
		
	case batchEventReceivedMsg:
		// Handle multiple events at once
		var statusUpdateCount int
		for _, event := range msg.events {
			if _, ok := event.(logic.StatusUpdatedEvent); ok {
				statusUpdateCount++
			}
		}
		log.Printf("[UPDATE] Processing batch of %d events (including %d status updates)", len(msg.events), statusUpdateCount)
		
		for _, event := range msg.events {
			m.handleDomainEvent(event)
		}
		
		// Log summary of repositories with statuses
		m.state.RepoMutex.RLock()
		reposWithStatus := 0
		for _, repo := range m.state.Repositories {
			if repo.Status.Branch != "" {
				reposWithStatus++
			}
		}
		totalRepos := len(m.state.Repositories)
		m.state.RepoMutex.RUnlock()
		
		log.Printf("[UPDATE] Finished processing batch. Repos with status: %d/%d", reposWithStatus, totalRepos)

	case EventMsg:
		// Handle domain events from external sources
		m.handleDomainEvent(msg.Event)

	}

	// Always tick for animations
	cmds = append(cmds, tickCmd())
	
	// Only log return for non-routine updates
	switch msg.(type) {
	case tickMsg, nil:
		// Don't log these
	default:
		log.Printf("[UPDATE] Returning with %d commands", len(cmds))
	}
	return m, tea.Batch(cmds...)
}

// View renders the UI
func (m *Model) View() string {
	viewStartTime := time.Now()
	defer func() {
		elapsed := time.Since(viewStartTime)
		if elapsed > 50*time.Millisecond {
			log.Printf("[VIEW] WARNING: View() took %v to complete", elapsed)
		}
	}()
	
	// Prevent nil pointer panics
	if m == nil {
		return "Model is nil"
	}
	if m.renderer == nil {
		return "Renderer is nil"
	}
	
	log.Printf("[VIEW] Building view state...")
	// Build view state from services
	viewState := m.buildViewState()
	
	log.Printf("[VIEW] Rendering with %d repos, %d groups", len(viewState.Repositories), len(viewState.Groups))
	// Render using the views package
	result := m.renderer.Render(viewState)
	log.Printf("[VIEW] Render complete, result length: %d", len(result))
	return result
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
		m.coordinator.UpdateOrderedLists() // Immediate update for sort mode changes
	}
	
	// Return to normal mode
	m.inputHandler.ChangeMode(inputtypes.ModeNormal, "")
	return nil
}

// Event handling
func (m *Model) subscribeToEvents() {
	// The internal UI event bus is used for communication between UI services
	// We no longer subscribe to domain events here as they are handled directly
	// in handleDomainEvent to avoid event loops
	
	// Note: If UI services need to communicate, they should use this event bus
	// For now, we don't have any internal UI events to subscribe to
}

// handleDomainEvent processes domain events
func (m *Model) handleDomainEvent(event interface{}) {
	log.Printf("[HANDLE_DOMAIN] Starting handleDomainEvent with type: %T", event)
	startTime := time.Now()
	
	// Process domain events directly based on their type
	switch e := event.(type) {
	case eventbus.ReposDiscoveredBatchEvent:
		log.Printf("[HANDLE_DOMAIN] ReposDiscoveredBatchEvent with %d repos", len(e.Repos))
		// Handle batch directly
		m.handleReposDiscoveredBatch(e)
		
	case logic.StatusUpdatedEvent:
		log.Printf("[HANDLE_DOMAIN] StatusUpdatedEvent for path: %s", e.Path)
		log.Printf("[HANDLE_DOMAIN] Status - Branch: %s, Dirty: %v, Ahead: %d, Behind: %d", 
			e.Status.Branch, e.Status.IsDirty, 
			e.Status.AheadCount, e.Status.BehindCount)
		
		m.state.RepoMutex.Lock()
		if existingRepo, exists := m.state.Repositories[e.Path]; exists {
			// Update only the status of the existing repository
			existingRepo.Status = e.Status
			log.Printf("[HANDLE_DOMAIN] Updated repo %s with branch: %s, dirty: %v", 
				filepath.Base(existingRepo.Path), existingRepo.Status.Branch, existingRepo.Status.IsDirty)
			m.state.RepoMutex.Unlock()
			
			// Trigger UI update after status change
			m.debouncedUpdateOrderedLists()
		} else {
			log.Printf("[HANDLE_DOMAIN] WARNING: Received status update for unknown repo: %s", e.Path)
			m.state.RepoMutex.Unlock()
		}
		
	case logic.RepositoryUpdatedEvent:
		// This is now used only for full repository updates, not status updates
		log.Printf("[HANDLE_DOMAIN] RepositoryUpdatedEvent for path: %s", e.Repository.Path)
		if e.Repository != nil && e.Repository.Path != "" {
			m.state.RepoMutex.Lock()
			m.state.Repositories[e.Repository.Path] = e.Repository
			m.state.RepoMutex.Unlock()
		}
		
	case logic.ScanStartedEvent:
		log.Printf("[HANDLE_DOMAIN] ScanStartedEvent")
		m.state.StatusMessage = "Scanning for repositories..."
		
	case logic.ScanCompletedEvent:
		log.Printf("[HANDLE_DOMAIN] ScanCompletedEvent with count: %d", e.Count)
		m.state.StatusMessage = fmt.Sprintf("Scan completed: %d repositories found", e.Count)
		// Final update after scan
		m.coordinator.UpdateOrderedLists()
		
	case domain.GroupAddedEvent:
		log.Printf("[HANDLE_DOMAIN] GroupAddedEvent")
		// Groups are already handled by the group store, just update UI
		m.coordinator.UpdateOrderedLists()
		
	case eventbus.FetchRequestedEvent:
		log.Printf("[HANDLE_DOMAIN] FetchRequestedEvent for %d repos", len(e.RepoPaths))
		// The git service will handle the fetch and send status updates
		// Show status message
		if len(e.RepoPaths) == 1 {
			m.state.StatusMessage = fmt.Sprintf("Fetching %s...", filepath.Base(e.RepoPaths[0]))
		} else {
			m.state.StatusMessage = fmt.Sprintf("Fetching %d repositories...", len(e.RepoPaths))
		}
		// Mark repos as fetching for UI feedback
		for _, path := range e.RepoPaths {
			m.state.FetchingRepos[path] = true
		}
		
	case eventbus.PullRequestedEvent:
		log.Printf("[HANDLE_DOMAIN] PullRequestedEvent for %d repos", len(e.RepoPaths))
		// The git service will handle the pull and send status updates
		// Show status message
		if len(e.RepoPaths) == 1 {
			m.state.StatusMessage = fmt.Sprintf("Pulling %s...", filepath.Base(e.RepoPaths[0]))
		} else {
			m.state.StatusMessage = fmt.Sprintf("Pulling %d repositories...", len(e.RepoPaths))
		}
		// Mark repos as pulling for UI feedback
		for _, path := range e.RepoPaths {
			m.state.PullingRepos[path] = true
		}
		
	case eventbus.StatusRefreshRequestedEvent:
		log.Printf("[HANDLE_DOMAIN] StatusRefreshRequestedEvent for %d repos", len(e.RepoPaths))
		// The git service will handle the refresh and send status updates
		// Show status message
		if len(e.RepoPaths) == 1 {
			m.state.StatusMessage = fmt.Sprintf("Refreshing %s...", filepath.Base(e.RepoPaths[0]))
		} else {
			m.state.StatusMessage = fmt.Sprintf("Refreshing %d repositories...", len(e.RepoPaths))
		}
		// Mark repos as refreshing for UI feedback
		for _, path := range e.RepoPaths {
			m.state.RefreshingRepos[path] = true
		}
		
	default:
		// For any other events, log but don't process
		log.Printf("[HANDLE_DOMAIN] Unhandled domain event type: %T", e)
	}
	
	elapsed := time.Since(startTime)
	log.Printf("[HANDLE_DOMAIN] Completed in %v", elapsed)
}

// handleReposDiscoveredBatch processes a batch of discovered repositories
func (m *Model) handleReposDiscoveredBatch(event eventbus.ReposDiscoveredBatchEvent) {
	log.Printf("UI: Processing batch with %d repos", len(event.Repos))
	
	// Add all repositories from batch
	m.state.RepoMutex.Lock()
	beforeCount := len(m.state.Repositories)
	for _, repo := range event.Repos {
		if repo.Path != "" {
			repoCopy := repo
			m.state.Repositories[repo.Path] = &repoCopy
		}
	}
	afterCount := len(m.state.Repositories)
	m.state.RepoMutex.Unlock()
	
	log.Printf("UI: Batch processed, repos before: %d, after: %d", beforeCount, afterCount)
	
	// Update ordered lists with debouncing
	m.debouncedUpdateOrderedLists()
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
	
	// Get operation states from model state
	refreshing := m.state.RefreshingRepos
	fetching := m.state.FetchingRepos
	pulling := m.state.PullingRepos
	
	// Make a copy of repositories under lock
	m.state.RepoMutex.RLock()
	repoCount := len(m.state.Repositories)
	reposCopy := make(map[string]*domain.Repository, repoCount)
	statusCount := 0
	for k, v := range m.state.Repositories {
		reposCopy[k] = v
		if v.Status.Branch != "" {
			statusCount++
			// Log first few repos with status
			if statusCount <= 3 {
				log.Printf("[VIEW] Repo %s has status - Branch: %s, Dirty: %v", 
					filepath.Base(v.Path), v.Status.Branch, v.Status.IsDirty)
			}
		}
	}
	m.state.RepoMutex.RUnlock()
	
	log.Printf("[VIEW] Building view state with %d repositories (%d have status)", repoCount, statusCount)
	
	return views.ViewState{
		Width:           m.state.Width,
		Height:          m.state.Height,
		Repositories:    reposCopy,
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

// debouncedUpdateOrderedLists schedules an UpdateOrderedLists call with debouncing
func (m *Model) debouncedUpdateOrderedLists() {
	// Cancel any existing timer
	if m.updateTimer != nil {
		m.updateTimer.Stop()
	}
	
	// Schedule a new update in 100ms
	m.updateTimer = time.AfterFunc(100*time.Millisecond, func() {
		log.Printf("Debounced UpdateOrderedLists triggered")
		m.coordinator.UpdateOrderedLists()
	})
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
		// Collect all available events (up to a limit to prevent blocking UI)
		var events []interface{}
		var statusCount int
		collectLoop:
		for i := 0; i < 500; i++ { // Process up to 500 events at once to handle large bursts
			select {
			case event := <-m.eventChan:
				events = append(events, event)
				// Count status events
				if _, ok := event.(logic.StatusUpdatedEvent); ok {
					statusCount++
				}
				// Log only the first few events to avoid spam
				if i < 3 || (statusCount > 0 && statusCount % 20 == 0) {
					switch e := event.(type) {
					case logic.RepositoryUpdatedEvent:
						log.Printf("[WAIT_EVENT] RepositoryUpdatedEvent for: %s", filepath.Base(e.Repository.Path))
					case logic.StatusUpdatedEvent:
						log.Printf("[WAIT_EVENT] StatusUpdatedEvent #%d for: %s with branch: %s", statusCount, filepath.Base(e.Path), e.Status.Branch)
					default:
						log.Printf("[WAIT_EVENT] Received event: %T", event)
					}
				}
			default:
				// No more events available
				if i > 100 && len(m.eventChan) > 0 {
					// If we've collected many events but channel still has more, continue
					log.Printf("[WAIT_EVENT] Collected %d events so far, channel still has %d", i, len(m.eventChan))
					continue
				}
				break collectLoop
			}
		}
		
		if len(events) > 0 {
			log.Printf("[WAIT_EVENT] Collected %d events (including %d status updates), channel buffer remaining: %d", len(events), statusCount, len(m.eventChan))
			return batchEventReceivedMsg{events: events}
		}
		return nil
	}
}

// batchEventReceivedMsg contains multiple events
type batchEventReceivedMsg struct {
	events []interface{}
}

// Tick for animations
func tickCmd() tea.Cmd {
	return tea.Tick(20*time.Millisecond, func(time.Time) tea.Msg {
		// Return a special tick message instead of nil
		return tickMsg{}
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
	case eventbus.ReposDiscoveredBatchEvent:
		return "eventbus.ReposDiscoveredBatchEvent"
	default:
		return "unknown"
	}
}