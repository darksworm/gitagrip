package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
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

	// Load configuration
	configSvc := config.NewConfigService()
	cfg, err := configSvc.Load()
	if err != nil {
		log.Printf("Error loading config: %v", err)
		// Use default config
		cfg = config.DefaultConfig()
	}

	// Create event bus
	bus := eventbus.New()

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