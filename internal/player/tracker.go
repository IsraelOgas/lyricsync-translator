package player

import (
	"context"
	"encoding/json"
	"time"
)

// Tracker combines MPRIS channels into a coherent state and emits events.
type Tracker struct {
	currentTrack *TrackInfo
	status       PlayerStatus
	positionMs   int64
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

	trackCh, statusCh, posCh, err := Start(t.playerctl)
	if err != nil {
		go func() {
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
		if initialTrack, initialStatus := GetCurrentTrack(t.playerctl); initialTrack != nil {
			t.currentTrack = initialTrack
			t.status = initialStatus
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
				changed := t.currentTrack == nil ||
					t.currentTrack.Artist != track.Artist ||
					t.currentTrack.Title != track.Title
				t.currentTrack = &track
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
				if t.status != status {
					t.status = status
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
				t.positionMs = pos

			case <-ticker.C:
				if t.currentTrack != nil && t.status == StatusPlaying {
					msg, _ := json.Marshal(TrackerEvent{
						Type:       "position",
						PositionMs: t.positionMs,
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
	return t.currentTrack
}

// GetStatus returns the current player status.
func (t *Tracker) GetStatus() PlayerStatus {
	return t.status
}

// GetPositionMs returns the current position in milliseconds.
func (t *Tracker) GetPositionMs() int64 {
	return t.positionMs
}
