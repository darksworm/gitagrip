package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"gitagrip/internal/config"
	"gitagrip/internal/discovery"
	"gitagrip/internal/eventbus"
	"gitagrip/internal/git"
	"gitagrip/internal/groups"
	"gitagrip/internal/ui"
    tea "github.com/charmbracelet/bubbletea/v2"
)

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
		defer func() {
			_ = logFile.Close()
		}()
		log.SetOutput(logFile)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
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
			// Update config with new groups and order
			cfg.Groups = event.Groups
			cfg.GroupOrder = event.GroupOrder
			// Save config
			if err := configSvc.SaveToPath(cfg, configPath); err != nil {
				log.Printf("Failed to save config: %v", err)
			} else {
				log.Printf("Config saved to %s", configPath)
			}
		}
	})

	// Initialize services
	discoverySvc := discovery.NewDiscoveryService(bus)
	_ = git.NewGitService(bus)                  // Git service subscribes to events automatically
	_ = groups.NewGroupManager(bus, cfg.Groups) // Group manager subscribes to events automatically

	// Create UI model
	uiModel := ui.NewModel(bus, cfg)

	// Create Bubble Tea program
	p := tea.NewProgram(uiModel, tea.WithAltScreen())

	// Set program reference in model and gitOps for terminal management
	uiModel.SetProgram(p)

	// Signal ready for E2E tests (only in test mode)
	if os.Getenv("GITAGRIP_E2E_TEST") == "1" {
		fmt.Println("__READY__")
	}

	// Set up event forwarding to UI
	eventChan := make(chan eventbus.DomainEvent, 100)
	bus.Subscribe(eventbus.EventRepoDiscovered, func(e eventbus.DomainEvent) {
		select {
		case eventChan <- e:
		default:
			// Channel full, drop event
			log.Println("Event channel full, dropping event")
		}
	})
	bus.Subscribe(eventbus.EventStatusUpdated, func(e eventbus.DomainEvent) {
		select {
		case eventChan <- e:
		default:
			log.Println("Event channel full, dropping event")
		}
	})
	bus.Subscribe(eventbus.EventError, func(e eventbus.DomainEvent) {
		select {
		case eventChan <- e:
		default:
			log.Println("Event channel full, dropping event")
		}
	})
	bus.Subscribe(eventbus.EventGroupAdded, func(e eventbus.DomainEvent) {
		select {
		case eventChan <- e:
		default:
			log.Println("Event channel full, dropping event")
		}
	})
	bus.Subscribe(eventbus.EventGroupRemoved, func(e eventbus.DomainEvent) {
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
	bus.Subscribe(eventbus.EventFetchCompleted, func(e eventbus.DomainEvent) {
		select {
		case eventChan <- e:
		default:
			log.Println("Event channel full, dropping event")
		}
	})
	bus.Subscribe(eventbus.EventPullCompleted, func(e eventbus.DomainEvent) {
		select {
		case eventChan <- e:
		default:
			log.Println("Event channel full, dropping event")
		}
	})
	bus.Subscribe(eventbus.EventCommandExecuted, func(e eventbus.DomainEvent) {
		select {
		case eventChan <- e:
		default:
			log.Println("Event channel full, dropping event")
		}
	})

	// Start forwarding events to UI in background
	go func() {
		for event := range eventChan {
			p.Send(ui.EventMsg{Event: event})
		}
	}()

	// Initialize groups from config
	for name := range cfg.Groups {
		bus.Publish(eventbus.GroupAddedEvent{Name: name})
	}

	// Start initial scan
	if cfg.BaseDir != "" {
		go func() {
			_ = discoverySvc.StartScan(ctx, []string{cfg.BaseDir})
		}()
	}

	// Run the UI
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}

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
	groups := make(map[string][]string)

	// Do a quick scan to find git repos and group them by immediate parent directory
	// We'll limit depth to avoid hanging on large directories
	maxDepth := 3
	reposByParent := make(map[string][]string)

	_ = filepath.WalkDir(baseDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Continue walking
		}

		// Check depth
		relPath, _ := filepath.Rel(baseDir, path)
		depth := strings.Count(relPath, string(filepath.Separator))
		if depth > maxDepth {
			return filepath.SkipDir
		}

		// Skip common non-repo directories
		if d.IsDir() {
			name := d.Name()
			if name == "node_modules" || name == ".npm" || name == "__pycache__" ||
				name == ".pytest_cache" || name == "venv" || name == ".venv" ||
				name == "target" || name == "build" || name == "dist" {
				return filepath.SkipDir
			}
		}

		// Check if this is a .git directory
		if d.IsDir() && d.Name() == ".git" {
			repoPath := filepath.Dir(path)

			// Get the parent directory relative to base
			relRepo, _ := filepath.Rel(baseDir, repoPath)
			parentDir := filepath.Dir(relRepo)

			// If repo is directly in base dir, don't create a group
			if parentDir == "." {
				return filepath.SkipDir
			}

			// Use the immediate parent directory as the group name
			groupName := filepath.Base(parentDir)
			reposByParent[groupName] = append(reposByParent[groupName], repoPath)

			return filepath.SkipDir
		}

		return nil
	})

	// Only create groups that have 2 or more repos
	for groupName, repos := range reposByParent {
		if len(repos) >= 2 {
			groups[groupName] = repos
		}
	}

	return groups
}
