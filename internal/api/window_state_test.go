package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// windowFullscreenState reads GET /api/window/fullscreen.
func windowFullscreenState(t *testing.T, baseURL string) bool {
	t.Helper()
	resp, err := http.Get(baseURL + "/api/window/fullscreen")
	if err != nil {
		t.Fatalf("GET /api/window/fullscreen: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("GET /api/window/fullscreen = %d %s, want 200", resp.StatusCode, body)
	}
	var body struct {
		Fullscreen bool `json:"fullscreen"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decoding GET response: %v", err)
	}
	return body.Fullscreen
}

// setWindowFullscreen writes PUT /api/window/fullscreen.
func setWindowFullscreen(t *testing.T, baseURL string, fullscreen bool) *http.Response {
	t.Helper()
	payload, err := json.Marshal(map[string]bool{"fullscreen": fullscreen})
	if err != nil {
		t.Fatalf("marshaling payload: %v", err)
	}
	req, err := http.NewRequest(http.MethodPut, baseURL+"/api/window/fullscreen", strings.NewReader(string(payload)))
	if err != nil {
		t.Fatalf("building PUT request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT /api/window/fullscreen: %v", err)
	}
	return resp
}

// TestWindowFullscreen_DefaultsToFalse pins the initial state: a fresh process
// reports windowed, so a close before any UI report saves a windowed state.
func TestWindowFullscreen_DefaultsToFalse(t *testing.T) {
	srv, _ := newShutdownTestServer(t)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	if got := windowFullscreenState(t, ts.URL); got {
		t.Fatalf("initial fullscreen state = %v, want false", got)
	}
}

// TestWindowFullscreen_ReportsUILastKnownState pins the contract the close
// handler relies on: the UI reports cinema mode, and the backend answers with
// exactly that value — without reading GDK state from another thread.
func TestWindowFullscreen_ReportsUILastKnownState(t *testing.T) {
	srv, _ := newShutdownTestServer(t)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := setWindowFullscreen(t, ts.URL, true)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("PUT fullscreen=true = %d, want 200", resp.StatusCode)
	}
	if got := windowFullscreenState(t, ts.URL); !got {
		t.Fatalf("fullscreen state after PUT true = %v, want true", got)
	}

	resp = setWindowFullscreen(t, ts.URL, false)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("PUT fullscreen=false = %d, want 200", resp.StatusCode)
	}
	if got := windowFullscreenState(t, ts.URL); got {
		t.Fatalf("fullscreen state after PUT false = %v, want false", got)
	}
}

// TestWindowFullscreen_RejectsInvalidBody pins that a malformed report cannot
// silently change the state the close handler will persist.
func TestWindowFullscreen_RejectsInvalidBody(t *testing.T) {
	srv, _ := newShutdownTestServer(t)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := setWindowFullscreen(t, ts.URL, true)
	resp.Body.Close()

	req, err := http.NewRequest(http.MethodPut, ts.URL+"/api/window/fullscreen", strings.NewReader("{not json"))
	if err != nil {
		t.Fatalf("building invalid PUT request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	bad, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT invalid body: %v", err)
	}
	bad.Body.Close()
	if bad.StatusCode != http.StatusBadRequest {
		t.Fatalf("PUT invalid body = %d, want 400", bad.StatusCode)
	}

	if got := windowFullscreenState(t, ts.URL); !got {
		t.Fatalf("fullscreen state after invalid body = %v, want unchanged true", got)
	}
}
