package events

// EventBus is a simple interface for publishing events
type EventBus interface {
	Publish(event interface{})
	Subscribe(eventType string, handler func(interface{}))
}

// NullBus is a no-op implementation of EventBus
type NullBus struct{}

func (n *NullBus) Publish(event interface{}) {}
func (n *NullBus) Subscribe(eventType string, handler func(interface{})) {}