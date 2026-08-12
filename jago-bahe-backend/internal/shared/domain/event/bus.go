// Package event defines the domain-event bus interface used across contexts.
// Contexts publish events (e.g. ProblemValidated) without knowing who consumes
// them. B0 ships the interface plus a synchronous in-memory bus; a durable
// implementation can replace it later without touching domain code.
package event

import "context"

// Event is any domain event. Name identifies the event type for routing/logging.
type Event interface {
	Name() string
}

// Handler reacts to a published event.
type Handler func(ctx context.Context, e Event) error

// Bus publishes events to subscribed handlers.
type Bus interface {
	Publish(ctx context.Context, e Event) error
	Subscribe(name string, h Handler)
}

// InMemoryBus is a simple synchronous Bus for a single-process deployment.
type InMemoryBus struct {
	handlers map[string][]Handler
}

// NewInMemoryBus constructs an empty in-memory bus.
func NewInMemoryBus() *InMemoryBus {
	return &InMemoryBus{handlers: make(map[string][]Handler)}
}

// Subscribe registers a handler for events with the given name.
func (b *InMemoryBus) Subscribe(name string, h Handler) {
	b.handlers[name] = append(b.handlers[name], h)
}

// Publish dispatches e to every handler subscribed to its name, synchronously.
// The first handler error short-circuits and is returned.
func (b *InMemoryBus) Publish(ctx context.Context, e Event) error {
	for _, h := range b.handlers[e.Name()] {
		if err := h(ctx, e); err != nil {
			return err
		}
	}
	return nil
}
