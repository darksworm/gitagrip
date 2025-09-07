package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"gitagrip/internal/config"
	"gitagrip/internal/discovery"
	"gitagrip/internal/eventbus"
	"gitagrip/internal/git"
	"gitagrip/internal/groups"
	"gitagrip/internal/ui"
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
		defer logFile.Close()
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
	discoverySvc := discovery.NewDiscoveryService(bus)
	_ = git.NewGitService(bus) // Git service subscribes to events automatically
	_ = groups.NewGroupManager(bus, cfg.Groups) // Group manager subscribes to events automatically

	// Create UI model
	uiModel := ui.NewModel(bus, cfg)

	// Create Bubble Tea program
	p := tea.NewProgram(uiModel, tea.WithAltScreen())

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
		go discoverySvc.StartScan(ctx, []string{cfg.BaseDir})
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
	// Don't do any scanning here - let the background discovery service handle it
	// This prevents the UI from hanging on large directories
	return make(map[string][]string)
}