package eventbus

import (
	"gitagrip/internal/domain"
	"log"
	"runtime/debug"
	"sync"
)

// Re-export domain types for convenience
type DomainEvent = domain.DomainEvent
type EventType = domain.EventType

// Event type constants
const (
	EventRepoDiscovered = domain.EventRepoDiscovered
	EventReposDiscoveredBatch = domain.EventReposDiscoveredBatch // Add batch event
	EventStatusUpdated  = domain.EventStatusUpdated
	EventError          = domain.EventError
	EventGroupAdded     = domain.EventGroupAdded
	EventGroupRemoved   = domain.EventGroupRemoved
	EventRepoMoved      = domain.EventRepoMoved
	EventScanStarted    = domain.EventScanStarted
	EventScanCompleted  = domain.EventScanCompleted
	EventScanRequested  = domain.EventScanRequested
	EventStatusRefreshRequested = domain.EventStatusRefreshRequested
	EventFetchRequested = domain.EventFetchRequested
	EventPullRequested  = domain.EventPullRequested
	EventConfigLoaded   = domain.EventConfigLoaded
	EventConfigSaved    = domain.EventConfigSaved
	EventConfigChanged  = domain.EventConfigChanged
)

// Re-export domain event types
type RepoDiscoveredEvent = domain.RepoDiscoveredEvent
type ReposDiscoveredBatchEvent = domain.ReposDiscoveredBatchEvent // Add batch event type
type StatusUpdatedEvent = domain.StatusUpdatedEvent
type ErrorEvent = domain.ErrorEvent
type GroupAddedEvent = domain.GroupAddedEvent
type GroupRemovedEvent = domain.GroupRemovedEvent
type RepoMovedEvent = domain.RepoMovedEvent
type ScanStartedEvent = domain.ScanStartedEvent
type ScanCompletedEvent = domain.ScanCompletedEvent
type ScanRequestedEvent = domain.ScanRequestedEvent
type StatusRefreshRequestedEvent = domain.StatusRefreshRequestedEvent
type FetchRequestedEvent = domain.FetchRequestedEvent
type PullRequestedEvent = domain.PullRequestedEvent
type ConfigLoadedEvent = domain.ConfigLoadedEvent
type ConfigSavedEvent = domain.ConfigSavedEvent
type ConfigChangedEvent = domain.ConfigChangedEvent

// EventHandler is a function that handles domain events
type EventHandler func(DomainEvent)

// EventBus is the interface for the event bus
type EventBus interface {
	Publish(event DomainEvent)
	Subscribe(eventType EventType, handler EventHandler) func()
}

// bus is the concrete implementation of EventBus
type bus struct {
	mu        sync.RWMutex
	handlers  map[EventType][]EventHandler
	eventChan chan DomainEvent
	wg        sync.WaitGroup
	quit      chan struct{}
}

// New creates a new event bus
func New() EventBus {
	b := &bus{
		handlers:  make(map[EventType][]EventHandler),
		eventChan: make(chan DomainEvent, 1000), // Increased from 100 to 1000
		quit:      make(chan struct{}),
	}
	
	// Start the event dispatcher
	b.wg.Add(1)
	go b.dispatch()
	
	return b
}

// Publish publishes an event to all subscribers
func (b *bus) Publish(event DomainEvent) {
	// Skip logging for high-frequency events
	switch event.Type() {
	case EventStatusUpdated:
		// Don't log status updates as they're too frequent
	default:
		log.Printf("EventBus: Publishing event %s", event.Type())
	}
	
	select {
	case b.eventChan <- event:
		// Event sent successfully
	default:
		// Channel full, log and drop
		log.Printf("Event bus channel full, dropping event: %v", event.Type())
	}
}

// Subscribe subscribes to events of a specific type
// Returns an unsubscribe function
func (b *bus) Subscribe(eventType EventType, handler EventHandler) func() {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	// Add handler to the list
	b.handlers[eventType] = append(b.handlers[eventType], handler)
	
	// Return unsubscribe function
	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		
		// Find and remove the handler
		handlers := b.handlers[eventType]
		for i, h := range handlers {
			// Compare function pointers
			if &h == &handler {
				// Remove handler by slicing
				b.handlers[eventType] = append(handlers[:i], handlers[i+1:]...)
				break
			}
		}
	}
}

// dispatch handles event distribution to subscribers
func (b *bus) dispatch() {
	defer b.wg.Done()
	
	for {
		select {
		case event := <-b.eventChan:
			// Get handlers for this event type
			b.mu.RLock()
			handlers := b.handlers[event.Type()]
			// Make a copy to avoid holding lock during handler execution
			handlersCopy := make([]EventHandler, len(handlers))
			copy(handlersCopy, handlers)
			b.mu.RUnlock()
			
			// Call each handler
			for _, handler := range handlersCopy {
				// Call handler in a goroutine to avoid blocking
				go func(h EventHandler, eventType EventType) {
					defer func() {
						if r := recover(); r != nil {
							log.Printf("Event handler panic for %s: %v\nStack: %s", eventType, r, debug.Stack())
						}
					}()
					h(event)
				}(handler, event.Type())
			}
			
		case <-b.quit:
			// Drain remaining events
			for {
				select {
				case <-b.eventChan:
					// Discard event
				default:
					return
				}
			}
		}
	}
}
