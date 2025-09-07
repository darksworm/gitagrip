package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"gitagrip/internal/config"
	"gitagrip/internal/discovery"
	"gitagrip/internal/domain"
	"gitagrip/internal/eventbus"
	"gitagrip/internal/git"
	"gitagrip/internal/groups"
	"gitagrip/internal/logic"
	"gitagrip/internal/ui"
)

// eventReceivedMsg wraps an event for the UI
type eventReceivedMsg struct {
	event interface{}
}

func main() {
	// Parse command line arguments
	var targetDir string
	flag.StringVar(&targetDir, "dir", "", "Directory to scan for repositories")
	flag.StringVar(&targetDir, "d", "", "Directory to scan for repositories (shorthand)")
	flag.Parse()
	
	// If no directory specified, check for remaining args
	if targetDir == "" && flag.NArg() > 0 {
		targetDir = flag.Arg(0)
	}
	
	// If still no directory, use current directory
	if targetDir == "" {
		var err error
		targetDir, err = os.Getwd()
		if err != nil {
			fmt.Printf("Error getting current directory: %v\n", err)
			os.Exit(1)
		}
	}
	
	// Resolve to absolute path
	absDir, err := filepath.Abs(targetDir)
	if err != nil {
		fmt.Printf("Error resolving path: %v\n", err)
		os.Exit(1)
	}
	
	// Set up logging
	logFile, err := os.OpenFile("gitagrip.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Printf("Could not open log file: %v", err)
	} else {
		defer logFile.Close()
		log.SetOutput(logFile)
	}

	// Create context for graceful shutdown
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	// Create event bus
	bus := eventbus.New()

	// Load configuration from the target directory with event bus support
	configPath := filepath.Join(absDir, ".gitagrip.toml")
	configSvc := config.NewConfigServiceWithBus(bus)
	cfg := loadOrCreateConfig(configSvc, absDir)
	
	// Subscribe to config changes to save automatically
	bus.Subscribe(eventbus.EventConfigChanged, func(e eventbus.DomainEvent) {
		if event, ok := e.(eventbus.ConfigChangedEvent); ok {
			// Update config with new groups
			cfg.Groups = event.Groups
			// Save config
			if err := configSvc.SaveToPath(cfg, configPath); err != nil {
				log.Printf("Failed to save config: %v", err)
			} else {
				log.Printf("Config saved to %s", configPath)
			}
		}
	})

	// Initialize services
	_ = discovery.NewDiscoveryService(bus) // Creates service and subscribes to events
	_ = git.NewGitService(bus) // Git service subscribes to events automatically
	_ = groups.NewGroupManager(bus, cfg.Groups) // Group manager subscribes to events automatically

	// Create stores for the new architecture
	repoStore := logic.NewMemoryRepositoryStore()
	groupStore := logic.NewMemoryGroupStore()
	
	// Initialize group store with config data
	for name, paths := range cfg.Groups {
		groupStore.AddGroup(&domain.Group{
			Name:  name,
			Repos: paths,
		})
	}

	// Create event channel for UI
	eventChan := make(chan interface{}, 1000)
	
	// Start channel monitor goroutine
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			log.Printf("[CHANNEL_MONITOR] Event channel status: %d/%d events", len(eventChan), cap(eventChan))
		}
	}()
	
	// Forward events to the event channel
	forwardEvent := func(e interface{}) {
		log.Printf("[FORWARD] Attempting to forward event: %T", e)
		select {
		case eventChan <- e:
			log.Printf("[FORWARD] Successfully forwarded event: %T, channel len: %d/%d", e, len(eventChan), cap(eventChan))
		default:
			log.Printf("[FORWARD] ERROR: Event channel full, dropping event: %T", e)
		}
	}

	// Create UI model using the new architecture
	log.Printf("Creating UI model...")
	uiModel := ui.NewModel(cfg, configSvc, repoStore, groupStore, bus, eventChan)
	log.Printf("UI model created successfully")

	// Create Bubble Tea program
	log.Printf("Creating Bubble Tea program...")
	p := tea.NewProgram(uiModel, tea.WithAltScreen())
	log.Printf("Bubble Tea program created")

	// Subscribe to events and forward to stores and UI
	bus.Subscribe(eventbus.EventRepoDiscovered, func(e eventbus.DomainEvent) {
		if event, ok := e.(eventbus.RepoDiscoveredEvent); ok {
			repo := &domain.Repository{
				Path:   event.Repo.Path,
				Name:   event.Repo.Name,
				Status: domain.RepoStatus{},
			}
			repoStore.AddRepository(repo)
			forwardEvent(logic.RepositoryDiscoveredEvent{Repository: repo})
		}
	})
	
	// Subscribe to batch repository discovery events
	bus.Subscribe(eventbus.EventReposDiscoveredBatch, func(e eventbus.DomainEvent) {
		if event, ok := e.(eventbus.ReposDiscoveredBatchEvent); ok {
			// Process all repositories in the batch
			for _, repoData := range event.Repos {
				repo := &domain.Repository{
					Path:   repoData.Path,
					Name:   repoData.Name,
					Status: domain.RepoStatus{},
				}
				repoStore.AddRepository(repo)
			}
			// Forward the batch event as-is to the UI
			log.Printf("Main: Forwarding batch event with %d repos", len(event.Repos))
			forwardEvent(event)
		}
	})
	
	// Forward scan events
	bus.Subscribe(eventbus.EventScanStarted, func(e eventbus.DomainEvent) {
		forwardEvent(logic.ScanStartedEvent{})
	})
	
	bus.Subscribe(eventbus.EventScanCompleted, func(e eventbus.DomainEvent) {
		if event, ok := e.(eventbus.ScanCompletedEvent); ok {
			log.Printf("Main: Scan completed with %d repos", event.ReposFound)
			forwardEvent(logic.ScanCompletedEvent{Count: event.ReposFound})
		}
	})
	
	bus.Subscribe(eventbus.EventStatusUpdated, func(e eventbus.DomainEvent) {
		if event, ok := e.(eventbus.StatusUpdatedEvent); ok {
			log.Printf("[MAIN] Received StatusUpdatedEvent for %s with branch: %s", 
				filepath.Base(event.RepoPath), event.Status.Branch)
			repo := repoStore.GetRepository(event.RepoPath)
			if repo != nil {
				repo.Status = event.Status
				repoStore.UpdateRepository(repo)
				
				// Send only the status update, not the whole repository
				statusCopy := domain.RepoStatus{
					Branch:       event.Status.Branch,
					IsDirty:      event.Status.IsDirty,
					HasUntracked: event.Status.HasUntracked,
					AheadCount:   event.Status.AheadCount,
					BehindCount:  event.Status.BehindCount,
					StashCount:   event.Status.StashCount,
					Error:        event.Status.Error,
				}
				log.Printf("[MAIN] Forwarding status update for %s with branch: %s (copy: %s)", 
					filepath.Base(repo.Path), event.Status.Branch, statusCopy.Branch)
				forwardEvent(logic.StatusUpdatedEvent{
					Path:   event.RepoPath,
					Status: statusCopy,
				})
			}
		}
	})
	
	bus.Subscribe(eventbus.EventGroupAdded, func(e eventbus.DomainEvent) {
		if event, ok := e.(eventbus.GroupAddedEvent); ok {
			group := &domain.Group{
				Name:  event.Name,
				Repos: []string{},
			}
			existing := groupStore.GetGroup(event.Name)
			if existing != nil {
				// Group already exists
				groupStore.UpdateGroup(existing)
			} else {
				groupStore.AddGroup(group)
			}
			forwardEvent(e)
		}
	})
	
	bus.Subscribe(eventbus.EventGroupRemoved, func(e eventbus.DomainEvent) {
		if event, ok := e.(eventbus.GroupRemovedEvent); ok {
			groupStore.DeleteGroup(event.Name)
			forwardEvent(e)
		}
	})
	bus.Subscribe(eventbus.EventFetchRequested, func(e eventbus.DomainEvent) {
		select {
		case eventChan <- e:
		default:
			log.Println("Event channel full, dropping event")
		}
	})
	bus.Subscribe(eventbus.EventRepoMoved, func(e eventbus.DomainEvent) {
		select {
		case eventChan <- e:
		default:
			log.Println("Event channel full, dropping event")
		}
	})
	bus.Subscribe(eventbus.EventPullRequested, func(e eventbus.DomainEvent) {
		select {
		case eventChan <- e:
		default:
			log.Println("Event channel full, dropping event")
		}
	})
	bus.Subscribe(eventbus.EventStatusRefreshRequested, func(e eventbus.DomainEvent) {
		select {
		case eventChan <- e:
		default:
			log.Println("Event channel full, dropping event")
		}
	})

	// Don't consume events here - the UI reads from eventChan directly via waitForEvent()
	// This was causing a race condition where events could go to either p.Send or waitForEvent

	// Initialize groups from config
	for name := range cfg.Groups {
		bus.Publish(eventbus.GroupAddedEvent{Name: name})
	}

	// Don't start scan here - let UI trigger it once initialized
	// This prevents race conditions with large directories
	// if cfg.BaseDir != "" {
	// 	go discoverySvc.StartScan(ctx, []string{cfg.BaseDir})
	// }

	// Add panic recovery at the top level
	defer func() {
		if r := recover(); r != nil {
			// Log panic to file since Bubble Tea may have taken over terminal
			panicFile, err := os.OpenFile("gitagrip.panic", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
			if err == nil {
				panicFile.WriteString(fmt.Sprintf("PANIC: %v\n", r))
				panicFile.WriteString(fmt.Sprintf("Stack trace:\n%s\n", debug.Stack()))
				panicFile.Close()
			}
			log.Printf("PANIC: %v\nStack: %s", r, debug.Stack())
			panic(r) // Re-panic to get full goroutine dump
		}
	}()
	
	// Run the UI
	log.Printf("Starting UI...")
	if _, err := p.Run(); err != nil {
		log.Printf("Error running program: %v", err)
		fmt.Printf("Error running program: %v\n", err)
		// Write error to separate file too
		errFile, _ := os.OpenFile("gitagrip.error", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
		if errFile != nil {
			errFile.WriteString(fmt.Sprintf("Error: %v\n", err))
			errFile.Close()
		}
		os.Exit(1)
	}
	log.Printf("UI exited normally")

	// Cleanup
	close(eventChan)
	cancel()
}

// loadOrCreateConfig loads config from the directory or creates a new one with auto-generated groups
func loadOrCreateConfig(configSvc config.ConfigService, targetDir string) *config.Config {
	// Try to load config from the target directory
	configPath := filepath.Join(targetDir, ".gitagrip.toml")
	
	// Check if config exists
	if _, err := os.Stat(configPath); err == nil {
		// Config exists, try to load it
		if cfg, err := configSvc.LoadFromPath(configPath); err == nil {
			log.Printf("Loaded config from %s", configPath)
			return cfg
		}
	}
	
	// No config or failed to load - create new one
	log.Printf("Creating new config for %s", targetDir)
	cfg := &config.Config{
		Version: 1,
		BaseDir: targetDir,
		UISettings: config.UISettings{
			ShowAheadBehind: true,
			AutosaveOnExit:  true,
		},
		Groups: generateGroupsFromDirectory(targetDir),
	}
	
	// Save the config
	if err := configSvc.SaveToPath(cfg, configPath); err != nil {
		log.Printf("Failed to save config: %v", err)
	}
	
	return cfg
}

// generateGroupsFromDirectory creates groups based on directory structure
// For now, return empty groups and let the discovery service populate them
func generateGroupsFromDirectory(baseDir string) map[string][]string {
	// Don't do any scanning here - let the background discovery service handle it
	// This prevents the UI from hanging on large directories
	return make(map[string][]string)
}