package api

import (
	"context"
	"fmt"
	"net/http"
)

// SSEBroker manages Server-Sent Events connections.
type SSEBroker struct {
	clients    map[chan []byte]bool
	register   chan chan []byte
	unregister chan chan []byte
	broadcast  chan []byte
}

// NewSSEBroker creates a new SSE broker.
func NewSSEBroker() *SSEBroker {
	return &SSEBroker{
		clients:    make(map[chan []byte]bool),
		register:   make(chan chan []byte),
		unregister: make(chan chan []byte),
		broadcast:  make(chan []byte, 64),
	}
}

// Start begins the broker event loop.
func (b *SSEBroker) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case client := <-b.register:
				b.clients[client] = true
			case client := <-b.unregister:
				if _, ok := b.clients[client]; ok {
					delete(b.clients, client)
					close(client)
				}
			case msg := <-b.broadcast:
				for client := range b.clients {
					select {
					case client <- msg:
					default:
						// skip slow clients
					}
				}
			}
		}
	}()
}

// Subscribe registers a new SSE client and returns its message channel.
func (b *SSEBroker) Subscribe() chan []byte {
	ch := make(chan []byte, 64)
	b.register <- ch
	return ch
}

// Unsubscribe removes a client.
func (b *SSEBroker) Unsubscribe(ch chan []byte) {
	b.unregister <- ch
}

// Publish sends a message to all connected SSE clients.
func (b *SSEBroker) Publish(event []byte) {
	select {
	case b.broadcast <- event:
	default:
	}
}

// handleSSE handles the GET /api/lyrics/stream endpoint.
func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Send current state immediately so the client doesn't start empty
	s.sendCurrentState(w, flusher)

	ch := s.sse.Subscribe()
	defer s.sse.Unsubscribe(ch)

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		}
	}
}

func (s *Server) sendCurrentState(w http.ResponseWriter, flusher http.Flusher) {
	// Send current track
	if s.lastTrackPayload != nil {
		fmt.Fprintf(w, "data: %s\n\n", s.lastTrackPayload)
	}

	// Send current status
	status := s.tracker.GetStatus()
	statusPayload := fmt.Sprintf(`{"type":"status","status":"%s"}`, status.String())
	fmt.Fprintf(w, "data: %s\n\n", statusPayload)

	// Send current lyrics
	if s.lastLyricsPayload != nil {
		fmt.Fprintf(w, "data: %s\n\n", s.lastLyricsPayload)
	}

	// Send current translations
	if s.lastTranslationsPayload != nil {
		fmt.Fprintf(w, "data: %s\n\n", s.lastTranslationsPayload)
	}

	flusher.Flush()
}
