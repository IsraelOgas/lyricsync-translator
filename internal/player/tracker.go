package player

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

// Tracker combines MPRIS channels into a coherent state and emits events.
//
// State is written by the Events goroutine (and, for the active player, by the
// playerctl reader goroutine) and read by HTTP handlers, so every field is
// guarded by mu.
type Tracker struct {
	mu           sync.RWMutex
	currentTrack *TrackInfo
	status       PlayerStatus
	positionMs   int64
	activePlayer string
	playerctl    string
}

// TrackerEvent is a JSON-serializable update sent to SSE clients.
type TrackerEvent struct {
	Type       string     `json:"type"`
	Track      *TrackInfo `json:"track,omitempty"`
	Status     string     `json:"status,omitempty"`
	PositionMs int64      `json:"position_ms,omitempty"`
	Timestamp  int64      `json:"timestamp"`
}

// NewTracker creates a tracker using the given playerctl binary path.
func NewTracker(playerctlPath string) *Tracker {
	return &Tracker{
		playerctl: playerctlPath,
		status:    StatusNoPlayer,
	}
}

// Events starts the MPRIS watcher and emits TrackerEvent JSON on the returned channel.
func (t *Tracker) Events(ctx context.Context) <-chan []byte {
	out := make(chan []byte, 64)

	trackCh, statusCh, posCh, err := Start(ctx, t.playerctl, t.setActivePlayer)
	if err != nil {
		go func() {
			defer close(out)
			msg, _ := json.Marshal(TrackerEvent{
				Type:      "status",
				Status:    StatusNoPlayer.String(),
				Timestamp: time.Now().UnixMilli(),
			})
			out <- msg
		}()
		return out
	}

	go func() {
		defer close(out)

		// Check if a track is already playing on startup
		if playerName, initialTrack, initialStatus := GetCurrentTrack(t.playerctl); initialTrack != nil {
			t.setActivePlayer(playerName)
			t.setCurrentTrack(initialTrack)
			t.setStatus(initialStatus)
			msg, _ := json.Marshal(TrackerEvent{
				Type:      "track",
				Track:     initialTrack,
				Timestamp: time.Now().UnixMilli(),
			})
			out <- msg
			// Also emit the initial status so the frontend shows it
			statusMsg, _ := json.Marshal(TrackerEvent{
				Type:      "status",
				Status:    initialStatus.String(),
				Timestamp: time.Now().UnixMilli(),
			})
			out <- statusMsg
		}

		// Periodically emit position even without updates (fallback polling)
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return

			case track, ok := <-trackCh:
				if !ok {
					return
				}
				previous := t.GetCurrent()
				changed := previous == nil ||
					previous.Artist != track.Artist ||
					previous.Title != track.Title
				t.setCurrentTrack(&track)
				if changed {
					msg, _ := json.Marshal(TrackerEvent{
						Type:      "track",
						Track:     &track,
						Timestamp: time.Now().UnixMilli(),
					})
					out <- msg
				}

			case status, ok := <-statusCh:
				if !ok {
					return
				}
				if t.GetStatus() != status {
					t.setStatus(status)
					msg, _ := json.Marshal(TrackerEvent{
						Type:      "status",
						Status:    status.String(),
						Timestamp: time.Now().UnixMilli(),
					})
					out <- msg
				}

			case pos, ok := <-posCh:
				if !ok {
					return
				}
				t.setPositionMs(pos)

			case <-ticker.C:
				current, playing, position := t.state()
				if current != nil && playing == StatusPlaying {
					msg, _ := json.Marshal(TrackerEvent{
						Type:       "position",
						PositionMs: position,
						Timestamp:  time.Now().UnixMilli(),
					})
					out <- msg
				}
			}
		}
	}()

	return out
}

// GetCurrent returns the current track info.
func (t *Tracker) GetCurrent() *TrackInfo {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.currentTrack
}

// GetStatus returns the current player status.
func (t *Tracker) GetStatus() PlayerStatus {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.status
}

// GetPositionMs returns the current position in milliseconds.
func (t *Tracker) GetPositionMs() int64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.positionMs
}

// GetActivePlayer returns the name of the currently tracked MPRIS player.
func (t *Tracker) GetActivePlayer() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.activePlayer
}

// setCurrentTrack records the track most recently seen on the active player.
func (t *Tracker) setCurrentTrack(track *TrackInfo) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.currentTrack = track
}

// setStatus records the player status.
func (t *Tracker) setStatus(status PlayerStatus) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.status = status
}

// setPositionMs records the last known playback position.
func (t *Tracker) setPositionMs(ms int64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.positionMs = ms
}

// setActivePlayer records the MPRIS player currently being followed.
func (t *Tracker) setActivePlayer(name string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.activePlayer = name
}

// state returns a consistent snapshot of track, status and position.
func (t *Tracker) state() (*TrackInfo, PlayerStatus, int64) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.currentTrack, t.status, t.positionMs
}
