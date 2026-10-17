package player

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// TrackInfo holds metadata about the currently playing track.
type TrackInfo struct {
	Artist      string `json:"artist"`
	Title       string `json:"title"`
	Album       string `json:"album,omitempty"`
	CoverArtURL string `json:"cover_art_url,omitempty"`
	DurationMs  int    `json:"duration_ms"`
}

// PlayerStatus represents the player state.
type PlayerStatus int

const (
	StatusPlaying PlayerStatus = iota
	StatusPaused
	StatusStopped
	StatusNoPlayer
)

func (s PlayerStatus) String() string {
	switch s {
	case StatusPlaying:
		return "playing"
	case StatusPaused:
		return "paused"
	case StatusStopped:
		return "stopped"
	case StatusNoPlayer:
		return "no_player"
	default:
		return "unknown"
	}
}

// Start launches playerctl --follow and returns channels for track info,
// player status, and position updates.
//
// The format string uses pipe as separator: artist||title||album||duration||position||status||playerName
// Position is updated roughly every 500ms by polling.
//
// The child process lives until ctx is cancelled: cancelling ctx kills
// playerctl and closes the three channels, so shutting the app down never
// leaves an orphan `playerctl --follow` behind.
//
// Returns an error if playerctl cannot be started (likely not installed).
// The caller should receive StatusNoPlayer on the status channel in that case.
func Start(ctx context.Context, playerctlPath string, activePlayer *string) (<-chan TrackInfo, <-chan PlayerStatus, <-chan int64, error) {
	format := "{{artist}}||{{title}}||{{album}}||{{mpris:length}}||{{position}}||{{status}}||{{playerName}}||{{mpris:artUrl}}"

	cmd := exec.CommandContext(ctx, playerctlPath, "--follow", "-a", "--format", format, "metadata", "position", "status")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("creating stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, nil, fmt.Errorf("starting playerctl: %w (is it installed?)", err)
	}

	trackCh := make(chan TrackInfo, 8)
	statusCh := make(chan PlayerStatus, 8)
	posCh := make(chan int64, 64)

	// The sends below stop on cancellation: Tracker.Events returns on ctx.Done
	// and stops draining, so an unguarded send would block this goroutine and
	// keep the reader alive after shutdown.
	sendTrack := func(t TrackInfo) bool {
		select {
		case trackCh <- t:
			return true
		case <-ctx.Done():
			return false
		}
	}
	sendStatus := func(s PlayerStatus) bool {
		select {
		case statusCh <- s:
			return true
		case <-ctx.Done():
			return false
		}
	}
	sendPosition := func(ms int64) bool {
		select {
		case posCh <- ms:
			return true
		case <-ctx.Done():
			return false
		}
	}
	go func() {
		defer cmd.Wait()
		defer close(trackCh)
		defer close(statusCh)
		defer close(posCh)

		var localActive string // track which player we're following
		if activePlayer != nil {
			*activePlayer = localActive
		}

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			if ctx.Err() != nil {
				return
			}
			line := scanner.Text()
			parts := strings.Split(line, "||")
			if len(parts) < 8 {
				continue
			}

			artist := parts[0]
			title := parts[1]
			artUrl := parts[len(parts)-1]
			playerName := parts[len(parts)-2]
			statusStr := parts[len(parts)-3]
			posStr := parts[len(parts)-4]
			durStr := parts[len(parts)-5]
			album := strings.Join(parts[2:len(parts)-5], "||")

			// Duration
			durationMs := 0
			if durStr != "" {
				if d, err := strconv.Atoi(durStr); err == nil {
					durationMs = d / 1000 // convert microseconds to milliseconds
				}
			}

			// Parse status for player-switching logic
			var status PlayerStatus
			switch strings.ToLower(statusStr) {
			case "playing":
				status = StatusPlaying
			case "paused":
				status = StatusPaused
			case "stopped":
				status = StatusStopped
			}

			// When any player starts playing, switch active player to it.
			if status == StatusPlaying && playerName != "" {
				if localActive != playerName {
					localActive = playerName
					if activePlayer != nil {
						*activePlayer = localActive
					}
				}
			}

			// If active player stopped, clear it so we can pick up another.
			if playerName == localActive && status == StatusStopped {
				localActive = ""
				if activePlayer != nil {
					*activePlayer = ""
				}
			}

			// Adopt first player with content if nothing is active yet.
			if localActive == "" && artist != "" && title != "" {
				localActive = playerName
				if activePlayer != nil {
					*activePlayer = localActive
				}
			}

			// Ignore events from players we are not tracking.
			if playerName != localActive {
				continue
			}

			// Skip empty artist/title (transitional output)
			if artist == "" && title == "" {
				if posStr != "" {
					if pos, err := strconv.Atoi(posStr); err == nil {
						if !sendPosition(int64(pos) / 1000) {
							return
						}
					}
				}
				continue
			}

			// Track info
			if artist != "" && title != "" {
				if !sendTrack(TrackInfo{
					Artist:      artist,
					Title:       title,
					Album:       album,
					CoverArtURL: artUrl,
					DurationMs:  durationMs,
				}) {
					return
				}
			}

			// Position
			if posStr != "" {
				if pos, err := strconv.Atoi(posStr); err == nil {
					if !sendPosition(int64(pos) / 1000) {
						return
					}
				}
			}

			// Status
			switch status {
			case StatusPlaying:
				if !sendStatus(StatusPlaying) {
					return
				}
			case StatusPaused:
				if !sendStatus(StatusPaused) {
					return
				}
			case StatusStopped:
				if !sendStatus(StatusStopped) {
					return
				}
			}
		}
	}()

	return trackCh, statusCh, posCh, nil
}

// TogglePlayPause sends play-pause to the given MPRIS player via playerctl.
// If playerName is empty, toggles the first available player.
func TogglePlayPause(playerctlPath string, playerName string) error {
	args := []string{"play-pause"}
	if playerName != "" {
		args = append([]string{"-p", playerName}, args...)
	}
	cmd := exec.Command(playerctlPath, args...)
	return cmd.Run()
}

// SetPosition seeks the given MPRIS player to a position in milliseconds.
// If playerName is empty, targets the first available player.
func SetPosition(playerctlPath string, playerName string, positionMs int) error {
	seconds := float64(positionMs) / 1000.0
	args := []string{"position", fmt.Sprintf("%.3f", seconds)}
	if playerName != "" {
		args = append([]string{"-p", playerName}, args...)
	}
	cmd := exec.Command(playerctlPath, args...)
	return cmd.Run()
}

// Next skips to the next track.
func Next(playerctlPath string, playerName string) error {
	args := []string{"next"}
	if playerName != "" {
		args = append([]string{"-p", playerName}, args...)
	}
	return exec.Command(playerctlPath, args...).Run()
}

// Previous skips to the previous track.
func Previous(playerctlPath string, playerName string) error {
	args := []string{"previous"}
	if playerName != "" {
		args = append([]string{"-p", playerName}, args...)
	}
	return exec.Command(playerctlPath, args...).Run()
}

// SetVolume adjusts the player volume by a relative delta (e.g., +0.05 or -0.05).
func SetVolume(playerctlPath string, playerName string, delta float64) error {
	sign := "+"
	if delta < 0 {
		sign = "-"
		delta = -delta
	}
	val := fmt.Sprintf("%.2f%s", delta, sign)
	args := []string{"volume", val}
	if playerName != "" {
		args = append([]string{"-p", playerName}, args...)
	}
	return exec.Command(playerctlPath, args...).Run()
}

// GetVolume returns the current player volume as a float between 0.0 and 1.0.
func GetVolume(playerctlPath string, playerName string) (float64, error) {
	args := []string{"volume"}
	if playerName != "" {
		args = append([]string{"-p", playerName}, args...)
	}
	out, err := exec.Command(playerctlPath, args...).Output()
	if err != nil {
		return 0, fmt.Errorf("playerctl volume: %w", err)
	}
	vol, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil {
		return 0, fmt.Errorf("parse volume %q: %w", strings.TrimSpace(string(out)), err)
	}
	return vol, nil
}

// SetAbsoluteVolume sets the player volume to an absolute value between 0.0 and 1.0.
func SetAbsoluteVolume(playerctlPath string, playerName string, vol float64) error {
	args := []string{"volume", fmt.Sprintf("%.2f", vol)}
	if playerName != "" {
		args = append([]string{"-p", playerName}, args...)
	}
	return exec.Command(playerctlPath, args...).Run()
}

// Shuffle toggles the player shuffle state.
func Shuffle(playerctlPath string, playerName string) error {
	args := []string{"shuffle", "Toggle"}
	if playerName != "" {
		args = append([]string{"-p", playerName}, args...)
	}
	return exec.Command(playerctlPath, args...).Run()
}

// GetShuffle returns whether shuffle is enabled ("On") or not ("Off").
func GetShuffle(playerctlPath string, playerName string) (string, error) {
	args := []string{"shuffle"}
	if playerName != "" {
		args = append([]string{"-p", playerName}, args...)
	}
	out, err := exec.Command(playerctlPath, args...).Output()
	if err != nil {
		return "", fmt.Errorf("playerctl shuffle: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// GetLoop returns the current loop state: "None", "Track", or "Playlist".
func GetLoop(playerctlPath string, playerName string) (string, error) {
	args := []string{"loop"}
	if playerName != "" {
		args = append([]string{"-p", playerName}, args...)
	}
	out, err := exec.Command(playerctlPath, args...).Output()
	if err != nil {
		return "", fmt.Errorf("playerctl loop: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// Loop cycles the player loop status: None → Track → Playlist → None.
// Returns the new state string ("None", "Track", or "Playlist").
func Loop(playerctlPath string, playerName string) (string, error) {
	// Read current state
	readArgs := []string{"loop"}
	if playerName != "" {
		readArgs = append([]string{"-p", playerName}, readArgs...)
	}
	out, err := exec.Command(playerctlPath, readArgs...).Output()
	if err != nil {
		return "", fmt.Errorf("reading loop state: %w", err)
	}
	current := strings.TrimSpace(string(out))

	// Cycle: None → Track → Playlist → None
	next := "Track"
	switch current {
	case "Track":
		next = "Playlist"
	case "Playlist":
		next = "None"
	}

	writeArgs := []string{"loop", next}
	if playerName != "" {
		writeArgs = append([]string{"-p", playerName}, writeArgs...)
	}
	if err := exec.Command(playerctlPath, writeArgs...).Run(); err != nil {
		return "", fmt.Errorf("setting loop: %w", err)
	}
	return next, nil
}

// GetCurrentTrack queries playerctl for the currently playing track (one-shot).
func GetCurrentTrack(playerctlPath string) (string, *TrackInfo, PlayerStatus) {
	format := "{{artist}}||{{title}}||{{album}}||{{mpris:length}}||{{status}}||{{playerName}}||{{mpris:artUrl}}"

	cmd := exec.Command(playerctlPath, "-a", "--format", format, "metadata", "status")
	out, err := cmd.Output()
	if err != nil {
		return "", nil, StatusNoPlayer
	}

	// There may be multiple lines (one per player).
	// Prioritize: Playing > Paused > any track.
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")

	type candidate struct {
		playerName string
		track      TrackInfo
		status     PlayerStatus
	}

	var playing, paused, any *candidate

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "||")
		if len(parts) < 7 {
			continue
		}

		artist := parts[0]
		title := parts[1]
		artUrl := parts[len(parts)-1]
		playerName := parts[len(parts)-2]
		statusStr := parts[len(parts)-3]
		durStr := parts[len(parts)-4]
		album := strings.Join(parts[2:len(parts)-4], "||")

		if artist == "" && title == "" {
			continue
		}

		var status PlayerStatus
		switch strings.ToLower(statusStr) {
		case "playing":
			status = StatusPlaying
		case "paused":
			status = StatusPaused
		default:
			status = StatusStopped
		}

		durationMs := 0
		if durStr != "" {
			if d, err := strconv.Atoi(durStr); err == nil {
				durationMs = d / 1000
			}
		}

		c := &candidate{
			playerName: playerName,
			track: TrackInfo{
				Artist:      artist,
				Title:       title,
				Album:       album,
				CoverArtURL: artUrl,
				DurationMs:  durationMs,
			},
			status: status,
		}

		switch status {
		case StatusPlaying:
			if playing == nil {
				playing = c
			}
		case StatusPaused:
			if paused == nil {
				paused = c
			}
		default:
			if any == nil {
				any = c
			}
		}
	}

	// Return best: Playing > Paused > first with any status
	if playing != nil {
		return playing.playerName, &playing.track, playing.status
	}
	if paused != nil {
		return paused.playerName, &paused.track, paused.status
	}
	if any != nil {
		return any.playerName, &any.track, any.status
	}

	// Fallback: try each player individually
	players, _ := ListPlayers(playerctlPath)
	for _, playerName := range players {
		track, status := getPlayerTrack(playerctlPath, playerName)
		if track != nil {
			return playerName, track, status
		}
	}
	return "", nil, StatusNoPlayer
}

func getPlayerTrack(playerctlPath, playerName string) (*TrackInfo, PlayerStatus) {
	format := "{{artist}}||{{title}}||{{album}}||{{mpris:length}}||{{status}}||{{playerName}}||{{mpris:artUrl}}"
	cmd := exec.Command(playerctlPath, "-p", playerName, "--format", format, "metadata", "status")
	out, err := cmd.Output()
	if err != nil {
		return nil, StatusNoPlayer
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "||")
		if len(parts) < 7 {
			continue
		}

		artist := parts[0]
		title := parts[1]
		artUrl := parts[len(parts)-1]
		_ = parts[len(parts)-2] // playerName — unused here, param already known
		statusStr := parts[len(parts)-3]
		durStr := parts[len(parts)-4]
		album := strings.Join(parts[2:len(parts)-4], "||")

		if artist == "" && title == "" {
			continue
		}

		var status PlayerStatus
		switch strings.ToLower(statusStr) {
		case "playing":
			status = StatusPlaying
		case "paused":
			status = StatusPaused
		default:
			status = StatusStopped
		}

		durationMs := 0
		if durStr != "" {
			if d, err := strconv.Atoi(durStr); err == nil {
				durationMs = d / 1000
			}
		}

		return &TrackInfo{
			Artist:      artist,
			Title:       title,
			Album:       album,
			CoverArtURL: artUrl,
			DurationMs:  durationMs,
		}, status
	}
	return nil, StatusNoPlayer
}

// ListPlayers returns all available MPRIS players.
func ListPlayers(playerctlPath string) ([]string, error) {
	cmd := exec.Command(playerctlPath, "-a", "-l")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("listing players: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var players []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			players = append(players, line)
		}
	}
	return players, nil
}
