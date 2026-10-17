package api

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/imov/lyricsync-translator/internal/cache"
	"github.com/imov/lyricsync-translator/internal/config"
	"github.com/imov/lyricsync-translator/internal/lyrics"
	"github.com/imov/lyricsync-translator/internal/player"
	"github.com/imov/lyricsync-translator/internal/translate"
)

// newShutdownTestServer builds a Server backed by a temp SQLite file so the
// shutdown path can be exercised end to end without touching user data.
func newShutdownTestServer(t *testing.T) (*Server, *config.Config) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "cache.db")
	store, err := cache.NewStore(dbPath)
	if err != nil {
		t.Fatalf("opening test store: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	if err := store.Migrate(); err != nil {
		t.Fatalf("migrating test store: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Server.Host = "127.0.0.1"
	cfg.Server.Port = 0 // replaced by a free port below
	// Keep tests hermetic: never attach to the developer's real MPRIS session.
	cfg.Player.PlayerctlPath = filepath.Join(t.TempDir(), "no-such-playerctl")

	tracker := player.NewTracker("playerctl")
	tranSvc := translate.NewService(
		translate.NewLibreTranslateClient("http://127.0.0.1:1", 1, ""),
		"es",
	)
	lyricsSvc := lyrics.NewService(
		lyrics.NewProvider("lrclib", "http://127.0.0.1:1", 1),
		store,
		tranSvc,
	)

	return NewServer(cfg, store, tracker, tranSvc, lyricsSvc), cfg
}

// freePort reserves and releases a local TCP port so Start can listen on it.
func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserving port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	return port
}

// TestSSEBroker_SubscribeAfterStopDoesNotBlock pins that a stopped broker
// never blocks callers: handling connections during shutdown must not hang.
func TestSSEBroker_SubscribeAfterStopDoesNotBlock(t *testing.T) {
	b := NewSSEBroker()
	ctx, cancel := context.WithCancel(context.Background())
	b.Start(ctx)

	// Simulate app shutdown: the broker context is cancelled.
	cancel()
	time.Sleep(50 * time.Millisecond) // let the broker loop observe ctx.Done

	done := make(chan struct{})
	go func() {
		defer close(done)
		ch := b.Subscribe()
		b.Unsubscribe(ch)
		b.Publish([]byte(`{"type":"status"}`))
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("SSEBroker blocked after its context was cancelled: Subscribe/Unsubscribe/Publish must never hang during shutdown")
	}
}

// TestSSEBroker_PublishAfterStopDoesNotBlock covers a publish racing with
// shutdown, which is the normal case while the app is closing.
func TestSSEBroker_PublishAfterStopDoesNotBlock(t *testing.T) {
	b := NewSSEBroker()
	ctx, cancel := context.WithCancel(context.Background())
	b.Start(ctx)
	client := b.Subscribe()

	cancel()
	time.Sleep(50 * time.Millisecond)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 200; i++ {
			b.Publish([]byte(fmt.Sprintf(`{"type":"position","n":%d}`, i)))
		}
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Publish blocked after the broker stopped")
	}

	// The client channel must be released so its SSE handler can return.
	select {
	case _, ok := <-client:
		if ok {
			// Drain any queued event, then require the channel to close.
			select {
			case _, ok2 := <-client:
				if ok2 {
					t.Fatal("expected client channel to be closed after stop")
				}
			case <-time.After(time.Second):
				t.Fatal("client channel was never closed after stop")
			}
		}
	case <-time.After(time.Second):
		t.Fatal("client channel was never closed after stop")
	}
}

// TestServerShutdown_DrainsActiveSSE pins the reported bug: closing the
// window while the UI holds an SSE stream open must not freeze shutdown.
func TestServerShutdown_DrainsActiveSSE(t *testing.T) {
	srv, cfg := newShutdownTestServer(t)
	port := freePort(t)
	cfg.Server.Port = port

	appCtx, appCancel := context.WithCancel(context.Background())
	defer appCancel()
	srv.Start(appCtx)

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	// Wait for the listener to accept connections.
	deadline := time.Now().Add(3 * time.Second)
	for {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 100*time.Millisecond)
		if err == nil {
			conn.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("server never started listening on %s: %v", baseURL, err)
		}
		time.Sleep(20 * time.Millisecond)
	}

	// Open an SSE stream and consume the initial snapshot, so the handler is
	// registered in the broker before shutdown starts.
	resp, err := http.Get(baseURL + "/api/lyrics/stream")
	if err != nil {
		t.Fatalf("opening SSE stream: %v", err)
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	if _, err := reader.ReadString('\n'); err != nil {
		t.Fatalf("reading first SSE line: %v", err)
	}

	// Shutdown exactly like OnShutdown does: an unbounded context.
	shutdownDone := make(chan error, 1)
	go func() { shutdownDone <- srv.Shutdown(context.Background()) }()

	select {
	case err := <-shutdownDone:
		if err != nil {
			t.Fatalf("Shutdown returned error with an active SSE client: %v", err)
		}
	case <-time.After(3 * time.Second):
		// Release the blocked shutdown so the test binary can exit.
		t.Cleanup(func() {
			if srv.httpSrv != nil {
				srv.httpSrv.Close()
			}
		})
		t.Fatal("Server.Shutdown blocked while an SSE client was connected: the window freezes on close")
	}

	// The stream must end, otherwise the webview keeps a dead connection.
	streamClosed := make(chan error, 1)
	go func() {
		for {
			if _, err := reader.ReadString('\n'); err != nil {
				streamClosed <- err
				return
			}
		}
	}()
	select {
	case <-streamClosed:
	case <-time.After(2 * time.Second):
		t.Fatal("SSE stream stayed open after Shutdown")
	}
}

// TestSSEHandler_SendsCurrentStateSnapshot pins the initial snapshot that a
// newly connected client receives: track, status, lyrics and translations,
// in that order. This is existing behavior that the shutdown fix must keep.
func TestSSEHandler_SendsCurrentStateSnapshot(t *testing.T) {
	srv, _ := newShutdownTestServer(t)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// Prime the snapshot the same way the tracker and lyrics service do.
	srv.lastTrackPayload = []byte(`{"type":"track","track":{"artist":"A","title":"T"}}`)
	srv.lastLyricsPayload = []byte(`{"type":"lyrics","lines":[{"id":1,"original":"one"}]}`)
	srv.lastTranslationsPayload = []byte(`{"type":"translations","lines":[{"id":1,"translated":"uno"}]}`)

	resp, err := http.Get(ts.URL + "/api/lyrics/stream")
	if err != nil {
		t.Fatalf("opening SSE stream: %v", err)
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	want := []string{
		`data: {"type":"track","track":{"artist":"A","title":"T"}}`,
		`data: {"type":"status","status":"no_player"}`,
		`data: {"type":"lyrics","lines":[{"id":1,"original":"one"}]}`,
		`data: {"type":"translations","lines":[{"id":1,"translated":"uno"}]}`,
	}
	for i, line := range want {
		got, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("reading SSE line %d: %v", i+1, err)
		}
		if strings.TrimRight(got, "\n") != line {
			t.Fatalf("SSE line %d = %q, want %q", i+1, strings.TrimRight(got, "\n"), line)
		}
		if blank, err := reader.ReadString('\n'); err != nil || strings.TrimSpace(blank) != "" {
			t.Fatalf("expected blank separator after SSE line %d, got %q (err=%v)", i+1, blank, err)
		}
	}
}
