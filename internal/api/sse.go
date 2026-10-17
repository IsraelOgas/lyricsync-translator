package api

import (
	"context"
	"fmt"
	"net/http"
	"sync"
)

// SSEBroker manages Server-Sent Events connections.
//
// Subscribe, Unsubscribe and Publish are safe to call at any moment, including
// while the app is shutting down: they never block and never send on a closed
// channel. That matters because the Wails close handler waits for the HTTP
// server to drain, and a streaming handler only returns once its channel is
// closed or its request context is cancelled.
type SSEBroker struct {
	mu      sync.Mutex
	clients map[chan []byte]struct{}
	stopped bool
	done    chan struct{}
}

// NewSSEBroker creates a new SSE broker.
func NewSSEBroker() *SSEBroker {
	return &SSEBroker{
		clients: make(map[chan []byte]struct{}),
		done:    make(chan struct{}),
	}
}

// Start begins the broker. Cancelling ctx stops it, which releases every
// connected client so their handlers can return.
func (b *SSEBroker) Start(ctx context.Context) {
	go func() {
		select {
		case <-ctx.Done():
			b.Stop()
		case <-b.done:
		}
	}()
}

// Stop closes every client channel and refuses new subscriptions.
// It is idempotent: calling it twice (context cancellation plus an explicit
// shutdown) is harmless.
func (b *SSEBroker) Stop() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.stopped {
		return
	}
	b.stopped = true
	for client := range b.clients {
		delete(b.clients, client)
		close(client)
	}
	close(b.done)
}

// Subscribe registers a new SSE client and returns its message channel.
// After Stop the returned channel is already closed, so the caller's stream
// terminates instead of hanging forever.
func (b *SSEBroker) Subscribe() chan []byte {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan []byte, 64)
	if b.stopped {
		close(ch)
		return ch
	}
	b.clients[ch] = struct{}{}
	return ch
}

// Unsubscribe removes a client. Safe to call multiple times and after Stop.
func (b *SSEBroker) Unsubscribe(ch chan []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.clients[ch]; !ok {
		return
	}
	delete(b.clients, ch)
	close(ch)
}

// Publish fans a message out to all connected SSE clients without blocking:
// slow clients are skipped, and events published after Stop are dropped.
func (b *SSEBroker) Publish(event []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for client := range b.clients {
		select {
		case client <- event:
		default:
			// skip slow clients
		}
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
	snap := s.lastPayloads()

	// Send current track
	if snap.track != nil {
		fmt.Fprintf(w, "data: %s\n\n", snap.track)
	}

	// Send current status
	status := s.tracker.GetStatus()
	statusPayload := fmt.Sprintf(`{"type":"status","status":"%s"}`, status.String())
	fmt.Fprintf(w, "data: %s\n\n", statusPayload)

	// Send current lyrics
	if snap.lyrics != nil {
		fmt.Fprintf(w, "data: %s\n\n", snap.lyrics)
	}

	// Send current translations
	if snap.translations != nil {
		fmt.Fprintf(w, "data: %s\n\n", snap.translations)
	}

	flusher.Flush()
}
