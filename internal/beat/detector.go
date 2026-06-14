package beat

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// BeatEvent represents a detected beat or audio energy update.
type BeatEvent struct {
	Type      string  `json:"type"`
	Timestamp int64   `json:"timestamp"`
	Energy    float64 `json:"energy"`
	IsOnset   bool    `json:"is_onset"`
	BPM       float64 `json:"bpm,omitempty"`
	BeatPhase float64 `json:"beat_phase,omitempty"`
}

// Detector captures system audio and detects beats in real-time.
// Uses ffmpeg to capture from PulseAudio/PipeWire monitor source.
type Detector struct {
	ffmpegPath string
	pythonPath string
	scriptPath string
}

// NewDetector creates a beat detector. Looks for the Python script next to the binary.
func NewDetector() *Detector {
	exe, _ := os.Executable()
	dir := filepath.Dir(exe)

	// In dev mode, script is in internal/beat/
	scriptPaths := []string{
		filepath.Join(dir, "beat_detect.py"),
		filepath.Join(dir, "..", "internal", "beat", "beat_detect.py"),
		filepath.Join("internal", "beat", "beat_detect.py"),
	}

	var script string
	for _, p := range scriptPaths {
		if _, err := os.Stat(p); err == nil {
			script = p
			break
		}
	}

	return &Detector{
		ffmpegPath: findBinary("ffmpeg"),
		pythonPath: findBinary("python3"),
		scriptPath: script,
	}
}

// Events starts capturing system audio and returns a channel of BeatEvents.
func (d *Detector) Events(ctx context.Context) <-chan BeatEvent {
	out := make(chan BeatEvent, 128)

	if d.ffmpegPath == "" || d.pythonPath == "" || d.scriptPath == "" {
		close(out)
		return out
	}

	go func() {
		defer close(out)

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			d.runCapture(ctx, out)

			select {
			case <-ctx.Done():
				return
			default:
			}

			// Restart after brief pause
			time.Sleep(2 * time.Second)
		}
	}()

	return out
}

func (d *Detector) runCapture(ctx context.Context, out chan<- BeatEvent) {
	// Auto-detect the PulseAudio monitor source name
	monitorSource := findMonitorSource()
	if monitorSource == "" {
		return
	}

	// ffmpeg captures from PulseAudio monitor and outputs raw float32 mono
	ffmpegArgs := []string{
		"-loglevel", "error",
		"-f", "pulse",
		"-i", monitorSource,
		"-ar", "44100",
		"-ac", "1",
		"-f", "f32le",
		"pipe:1",
	}

	ffmpegCmd := exec.CommandContext(ctx, d.ffmpegPath, ffmpegArgs...)
	ffmpegCmd.Stderr = nil

	ffmpegStdout, err := ffmpegCmd.StdoutPipe()
	if err != nil {
		return
	}

	// Python script reads raw audio from stdin and outputs JSON beat events
	pyCmd := exec.CommandContext(ctx, d.pythonPath, d.scriptPath)
	pyCmd.Stdin = ffmpegStdout
	pyCmd.Stderr = os.Stderr

	pyStdout, err := pyCmd.StdoutPipe()
	if err != nil {
		return
	}

	if err := ffmpegCmd.Start(); err != nil {
		return
	}
	if err := pyCmd.Start(); err != nil {
		ffmpegCmd.Process.Kill()
		return
	}

	// Read JSON events from Python output
	scanner := bufio.NewScanner(pyStdout)
	scanner.Buffer(make([]byte, 1024*64), 1024*64)

	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			var event BeatEvent
			if err := json.Unmarshal([]byte(line), &event); err != nil {
				continue
			}

			event.Type = "beat"

			select {
			case out <- event:
			case <-ctx.Done():
				return
			}
		}
	}()

	_ = ffmpegCmd.Wait()
	_ = pyCmd.Wait()
}

// findMonitorSource uses ffmpeg to list PulseAudio sources and finds the monitor.
func findMonitorSource() string {
	cmd := exec.Command("ffmpeg", "-sources", "pulse")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, ".monitor") && strings.Contains(line, "Monitor of") {
			// Extract source name: first token before space
			parts := strings.Fields(line)
			if len(parts) > 0 {
				return parts[0]
			}
		}
	}
	return ""
}

func findBinary(name string) string {
	if path, err := exec.LookPath(name); err == nil {
		return path
	}
	return ""
}
