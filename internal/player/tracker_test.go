//go:build linux

package player

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// writeChattyPlayerctl writes a fake playerctl that streams metadata lines on
// --follow, so the tracker keeps updating its state while the test reads it.
func writeChattyPlayerctl(t *testing.T) string {
	t.Helper()
	scriptPath := filepath.Join(t.TempDir(), "chatty-playerctl")
	script := `#!/bin/sh
case "$*" in
  *--follow*)
    i=0
    while [ $i -lt 60 ]; do
      printf '%s\n' 'Artist||Title||Album||180000000||1000000||Playing||harness||file://cover.jpg'
      i=$((i+1))
      sleep 0.02
    done
    ;;
esac
exit 0
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("writing fake playerctl: %v", err)
	}
	return scriptPath
}

// TestTracker_ConcurrentStateAccess pins that tracker state can be read from
// other goroutines while the tracker updates it. Run with -race: without the
// mutex this test reports a data race between the Events goroutine and the
// readers (the same access pattern HTTP handlers use).
func TestTracker_ConcurrentStateAccess(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tracker := NewTracker(writeChattyPlayerctl(t))
	tracker.Events(ctx)

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				_ = tracker.GetCurrent()
				_ = tracker.GetStatus()
				_ = tracker.GetPositionMs()
				_ = tracker.GetActivePlayer()
				time.Sleep(time.Millisecond)
			}
		}()
	}
	wg.Wait()

	// The tracker must still answer coherently after the concurrent reads.
	switch status := tracker.GetStatus(); status {
	case StatusPlaying, StatusPaused, StatusStopped, StatusNoPlayer:
	default:
		t.Fatalf("GetStatus returned an out-of-range PlayerStatus: %v", status)
	}
}
