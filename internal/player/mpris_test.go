//go:build linux

package player

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// writeFakePlayerctl creates an executable stand-in for `playerctl --follow`
// that records its own PID and then blocks, so the test can prove the process
// is terminated when the context is cancelled.
func writeFakePlayerctl(t *testing.T) (scriptPath, pidPath string) {
	t.Helper()
	dir := t.TempDir()
	pidPath = filepath.Join(dir, "playerctl.pid")
	scriptPath = filepath.Join(dir, "fake-playerctl")

	// Args without --follow (the one-shot GetCurrentTrack probe) exit at once,
	// so only the long-lived follower records and holds the PID.
	script := fmt.Sprintf(`#!/bin/sh
case "$*" in
  *--follow*) echo $$ > %q; exec sleep 300 ;;
esac
exit 0
`, pidPath)
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("writing fake playerctl: %v", err)
	}
	return scriptPath, pidPath
}

// waitForPID waits until the fake playerctl recorded its PID.
func waitForPID(t *testing.T, pidPath string) int {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		data, err := os.ReadFile(pidPath)
		if err == nil {
			pid, convErr := strconv.Atoi(strings.TrimSpace(string(data)))
			if convErr == nil && pid > 0 {
				return pid
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("fake playerctl never recorded its PID (last error: %v)", err)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// processAlive reports whether pid is still running.
func processAlive(pid int) bool {
	_, err := os.Stat(fmt.Sprintf("/proc/%d", pid))
	return err == nil
}

// TestStart_KillsChildOnContextCancel pins the shutdown rule: cancelling the
// tracker context must terminate the playerctl child instead of leaving an
// orphan process behind after the app closes.
func TestStart_KillsChildOnContextCancel(t *testing.T) {
	scriptPath, pidPath := writeFakePlayerctl(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var active string
	if _, _, _, err := Start(ctx, scriptPath, &active); err != nil {
		t.Fatalf("Start: %v", err)
	}

	pid := waitForPID(t, pidPath)
	if !processAlive(pid) {
		t.Fatalf("fake playerctl (pid %d) died before the test cancelled the context", pid)
	}

	cancel()

	deadline := time.Now().Add(5 * time.Second)
	for processAlive(pid) {
		if time.Now().After(deadline) {
			t.Fatalf("playerctl child (pid %d) survived context cancellation: shutdown leaves an orphan process", pid)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// TestStart_ClosesChannelsOnContextCancel pins that the returned channels are
// closed after cancellation, so Tracker.Events never leaves its consumers
// waiting on a channel that will never produce.
func TestStart_ClosesChannelsOnContextCancel(t *testing.T) {
	scriptPath, _ := writeFakePlayerctl(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	trackCh, statusCh, posCh, err := Start(ctx, scriptPath, nil)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	cancel()

	trackClosed := make(chan struct{})
	go func() {
		defer close(trackClosed)
		for range trackCh { //nolint:revive // draining until close is the point
		}
	}()
	statusClosed := make(chan struct{})
	go func() {
		defer close(statusClosed)
		for range statusCh { //nolint:revive // draining until close is the point
		}
	}()
	posClosed := make(chan struct{})
	go func() {
		defer close(posClosed)
		for range posCh { //nolint:revive // draining until close is the point
		}
	}()

	for name, done := range map[string]<-chan struct{}{
		"track":  trackClosed,
		"status": statusClosed,
		"pos":    posClosed,
	} {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatalf("%s channel stayed open after context cancellation", name)
		}
	}
}

// TestStart_MissingPlayerctlStillReturnsNoPlayerError pins existing behavior:
// a missing playerctl binary is reported as an error, not a panic or a hang.
func TestStart_MissingPlayerctlStillReturnsNoPlayerError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var active string
	_, _, _, err := Start(ctx, filepath.Join(t.TempDir(), "does-not-exist"), &active)
	if err == nil {
		t.Fatal("Start with a missing playerctl binary returned no error")
	}
}

// TestTrackerEvents_KillsChildOnContextCancel is the end-to-end rule for the
// tracker: after the app cancels the tracker context, the playerctl follower
// must be gone. This is the behavior that used to leak an orphan process on
// every app close.
func TestTrackerEvents_KillsChildOnContextCancel(t *testing.T) {
	scriptPath, pidPath := writeFakePlayerctl(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tracker := NewTracker(scriptPath)
	tracker.Events(ctx)

	pid := waitForPID(t, pidPath)

	cancel()

	deadline := time.Now().Add(5 * time.Second)
	for processAlive(pid) {
		if time.Now().After(deadline) {
			t.Fatalf("playerctl follower (pid %d) still alive after the tracker context was cancelled", pid)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
