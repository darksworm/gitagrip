package events

import (
	"fmt"
	"sync"
)

// Bus is a simple event bus for UI services
type Bus struct {
	mu        sync.RWMutex
	listeners map[string][]func(interface{})
}

// NewBus creates a new event bus
func NewBus() *Bus {
	return &Bus{
		listeners: make(map[string][]func(interface{})),
	}
}

// Subscribe registers a listener for an event type
func (b *Bus) Subscribe(eventType string, handler func(interface{})) {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	b.listeners[eventType] = append(b.listeners[eventType], handler)
}

// Publish sends an event to all listeners
func (b *Bus) Publish(event interface{}) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	// Get event type from the event's type name
	eventType := getEventType(event)
	
	if handlers, ok := b.listeners[eventType]; ok {
		for _, handler := range handlers {
			// Run handlers in goroutines to avoid blocking
			go handler(event)
		}
	}
}

// getEventType extracts the type name from an event
func getEventType(event interface{}) string {
	// Simple type name extraction
	switch event.(type) {
	default:
		// Use the full type name as the event type
		return fmt.Sprintf("%T", event)
	}
}