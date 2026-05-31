package lyrics

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// LRCLibClient implements LyricsProvider for the LRCLIB API.
type LRCLibClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewLRCLibClient creates a new LRCLIB client.
func NewLRCLibClient(baseURL string, timeoutSec int) *LRCLibClient {
	return &LRCLibClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: time.Duration(timeoutSec) * time.Second,
		},
	}
}

// Name returns the provider identifier.
func (c *LRCLibClient) Name() string {
	return "lrclib"
}

// lrclibTrack is the JSON response from LRCLIB get endpoint.
// Duration is float64 because LRCLIB can return floats like 313.0.
type lrclibTrack struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	TrackName    string  `json:"trackName"`
	ArtistName   string  `json:"artistName"`
	AlbumName    string  `json:"albumName"`
	Duration     float64 `json:"duration"`
	SyncedLyrics string `json:"syncedLyrics"`
	PlainLyrics  string `json:"plainLyrics"`
}

// SearchLyrics searches for lyrics using the LRCLIB get endpoint.
func (c *LRCLibClient) SearchLyrics(artist, title string) (*LyricsResult, error) {
	// Try direct get first
	endpoint := fmt.Sprintf("%s/get", c.baseURL)
	params := url.Values{}
	params.Set("artist_name", artist)
	params.Set("track_name", title)

	req, err := http.NewRequest("GET", endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("lrclib request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("lrclib returned %d: %s", resp.StatusCode, string(body))
	}

	var track lrclibTrack
	if err := json.NewDecoder(resp.Body).Decode(&track); err != nil {
		return nil, fmt.Errorf("decoding lrclib response: %w", err)
	}

	result := &LyricsResult{
		Source: "lrclib",
	}

	// Prefer synced lyrics
	if track.SyncedLyrics != "" {
		result.Synced = true
		result.Lyrics = track.SyncedLyrics
		lines, err := ParseLRC(track.SyncedLyrics)
		if err != nil {
			return nil, fmt.Errorf("parsing LRC: %w", err)
		}
		result.Lines = lines
	} else if track.PlainLyrics != "" {
		result.Synced = false
		result.Lyrics = track.PlainLyrics
		// Split plain lyrics into lines
		for _, line := range splitLines(track.PlainLyrics) {
			if line == "" {
				continue
			}
			result.Lines = append(result.Lines, LyricLine{Text: line})
		}
	}

	if len(result.Lines) == 0 {
		return nil, nil
	}

	return result, nil
}

func splitLines(text string) []string {
	var lines []string
	current := ""
	for _, ch := range text {
		if ch == '\n' {
			lines = append(lines, current)
			current = ""
		} else if ch != '\r' {
			current += string(ch)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}
