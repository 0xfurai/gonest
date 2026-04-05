package gonest

import "sync"

// EventEmitter provides a simple pub/sub event system.
// Equivalent to NestJS @nestjs/event-emitter.
type EventEmitter struct {
	mu       sync.RWMutex
	handlers map[string][]EventHandler
}

// EventHandler processes an emitted event.
type EventHandler func(data any) error

// NewEventEmitter creates a new event emitter.
func NewEventEmitter() *EventEmitter {
	return &EventEmitter{
		handlers: make(map[string][]EventHandler),
	}
}

// On registers a handler for the given event.
func (e *EventEmitter) On(event string, handler EventHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.handlers[event] = append(e.handlers[event], handler)
}

// Emit fires an event, calling all registered handlers.
func (e *EventEmitter) Emit(event string, data any) error {
	e.mu.RLock()
	handlers := e.handlers[event]
	e.mu.RUnlock()

	for _, h := range handlers {
		if err := h(data); err != nil {
			return err
		}
	}
	return nil
}

// EmitAsync fires an event asynchronously.
func (e *EventEmitter) EmitAsync(event string, data any) {
	e.mu.RLock()
	handlers := e.handlers[event]
	e.mu.RUnlock()

	for _, h := range handlers {
		go h(data)
	}
}

// RemoveAll removes all handlers for the given event.
func (e *EventEmitter) RemoveAll(event string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.handlers, event)
}

// ListenerCount returns the number of handlers for an event.
func (e *EventEmitter) ListenerCount(event string) int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.handlers[event])
}
