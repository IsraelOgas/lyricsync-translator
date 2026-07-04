package player

import (
	"context"
	"encoding/json"
	"time"
)

// Tracker combines MPRIS channels into a coherent state and emits events.
type Tracker struct {
	currentTrack    *TrackInfo
	status          PlayerStatus
	positionMs      int64
	activePlayer    string
	manualPlayer    string // user-selected player override (empty = auto)
	playerctl       string
	availableCached []string
	cacheExpiry     time.Time
}

// TrackerEvent is a JSON-serializable update sent to SSE clients.
type TrackerEvent struct {
	Type       string     `json:"type"`
	Track      *TrackInfo `json:"track,omitempty"`
	Status     string     `json:"status,omitempty"`
	PositionMs int64      `json:"position_ms,omitempty"`
	PlayerName string     `json:"player_name,omitempty"`
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

	trackCh, statusCh, posCh, err := Start(t.playerctl, &t.activePlayer)
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
		if playerName, initialTrack, initialStatus := GetCurrentTrack(t.playerctl); initialTrack != nil {
			if t.manualPlayer == "" {
				t.activePlayer = playerName
			} else {
				t.activePlayer = t.manualPlayer
			}
			t.currentTrack = initialTrack
			t.status = initialStatus
			msg, _ := json.Marshal(TrackerEvent{
				Type:       "track",
				Track:      initialTrack,
				PlayerName: t.activePlayer,
				Timestamp:  time.Now().UnixMilli(),
			})
			out <- msg
			// Also emit the initial status so the frontend shows it
			statusMsg, _ := json.Marshal(TrackerEvent{
				Type:       "status",
				Status:     initialStatus.String(),
				PlayerName: t.activePlayer,
				Timestamp:  time.Now().UnixMilli(),
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
			// Update active player from track event when in auto mode
			if t.manualPlayer == "" && track.PlayerName != "" {
				t.activePlayer = track.PlayerName
			}
			if changed {
				msg, _ := json.Marshal(TrackerEvent{
					Type:       "track",
					Track:      &track,
					PlayerName: t.activePlayer,
					Timestamp:  time.Now().UnixMilli(),
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
					Type:       "status",
					Status:     status.String(),
					PlayerName: t.activePlayer,
					Timestamp:  time.Now().UnixMilli(),
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

// GetActivePlayer returns the name of the currently tracked MPRIS player.
func (t *Tracker) GetActivePlayer() string {
	return t.activePlayer
}

// SetManualPlayer overrides the auto-detected active player with the given name.
// Subsequent auto-switching is disabled until ClearManualPlayer is called.
func (t *Tracker) SetManualPlayer(name string) {
	t.manualPlayer = name
	t.activePlayer = name
}

// ClearManualPlayer re-enables automatic player switching.
func (t *Tracker) ClearManualPlayer() {
	t.manualPlayer = ""
}

// IsManual returns true if the active player was set manually by the user.
func (t *Tracker) IsManual() bool {
	return t.manualPlayer != ""
}

// ListAvailablePlayers returns all MPRIS players with their current status.
func (t *Tracker) ListAvailablePlayers() ([]PlayerInfo, error) {
	players, err := ListPlayers(t.playerctl)
	if err != nil {
		return nil, err
	}

	var result []PlayerInfo
	for _, name := range players {
		track, status := GetPlayerTrack(t.playerctl, name)
		pi := PlayerInfo{
			Name:   name,
			Status: status.String(),
		}
		if track != nil {
			pi.Track = track
		}
		result = append(result, pi)
	}
	return result, nil
}

// PlayerInfo holds a player name and its current state.
type PlayerInfo struct {
	Name   string     `json:"name"`
	Status string     `json:"status"`
	Track  *TrackInfo `json:"track,omitempty"`
}
