package gonest

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// SSEEvent represents a Server-Sent Event.
type SSEEvent struct {
	ID    string `json:"id,omitempty"`
	Event string `json:"event,omitempty"`
	Data  any    `json:"data"`
}

// SSEStream provides a channel-based interface for sending SSE events.
type SSEStream struct {
	events chan SSEEvent
	done   chan struct{}
}

// NewSSEStream creates a new SSE stream.
func NewSSEStream(bufferSize int) *SSEStream {
	if bufferSize <= 0 {
		bufferSize = 10
	}
	return &SSEStream{
		events: make(chan SSEEvent, bufferSize),
		done:   make(chan struct{}),
	}
}

// Send sends an event to the stream.
func (s *SSEStream) Send(event SSEEvent) {
	select {
	case s.events <- event:
	case <-s.done:
	}
}

// Close closes the stream.
func (s *SSEStream) Close() {
	close(s.done)
}

// SSE creates a handler that streams Server-Sent Events.
// The provided function receives an SSEStream and should send events to it.
// The function runs in a goroutine. Close the stream when done.
func SSE(fn func(stream *SSEStream)) HandlerFunc {
	return func(ctx Context) error {
		w := ctx.ResponseWriter()
		r := ctx.Request()

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Transfer-Encoding", "chunked")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		if !ok {
			return NewInternalServerError("streaming not supported")
		}

		stream := NewSSEStream(10)
		go fn(stream)

		for {
			select {
			case event, ok := <-stream.events:
				if !ok {
					return nil
				}
				if event.ID != "" {
					fmt.Fprintf(w, "id: %s\n", event.ID)
				}
				if event.Event != "" {
					fmt.Fprintf(w, "event: %s\n", event.Event)
				}
				dataBytes, _ := json.Marshal(event.Data)
				fmt.Fprintf(w, "data: %s\n\n", string(dataBytes))
				flusher.Flush()
			case <-stream.done:
				return nil
			case <-r.Context().Done():
				stream.Close()
				return nil
			}
		}
	}
}
