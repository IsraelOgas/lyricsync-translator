package lyrics

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// LrcMuxClient implements LyricsProvider for the lrcmux aggregator API.
// lrcmux fans out to multiple providers (LRCLIB, Kugou, NetEase, etc.) and
// returns word-level synced lyrics with cover art metadata.
type LrcMuxClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewLrcMuxClient creates a new lrcmux client.
func NewLrcMuxClient(baseURL string, timeoutSec int) *LrcMuxClient {
	return &LrcMuxClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: time.Duration(timeoutSec) * time.Second,
		},
	}
}

// Name returns the provider identifier.
func (c *LrcMuxClient) Name() string {
	return "lrcmux"
}

// lrcmuxResponse maps the lrcmux native /get endpoint JSON.
type lrcmuxResponse struct {
	Track lrcmuxTrack  `json:"track"`
	Lines []lrcmuxLine `json:"lines"`
	Meta  lrcmuxMeta   `json:"meta"`
}

type lrcmuxTrack struct {
	Title    string       `json:"title"`
	Artist   string       `json:"artist"`
	Album    string       `json:"album"`
	ISRC     string       `json:"isrc"`
	Duration int          `json:"duration"`
	Cover    *lrcmuxCover `json:"cover"`
}

type lrcmuxCover struct {
	Small  string `json:"small"`
	Medium string `json:"medium"`
	Big    string `json:"big"`
}

type lrcmuxLine struct {
	Text  string       `json:"text"`
	Start int          `json:"start"`
	End   int          `json:"end"`
	Words []lrcmuxWord `json:"words"`
}

type lrcmuxWord struct {
	Text  string `json:"text"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}

type lrcmuxMeta struct {
	Source       lrcmuxSource `json:"source"`
	Level        string       `json:"level"`
	Instrumental bool         `json:"instrumental"`
}

type lrcmuxSource struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

// SearchLyrics fetches lyrics from the lrcmux native endpoint.
// level is the requested sync level: "line" (default) or "word".
// isrc is the track ISRC if known; it takes priority over artist/title.
func (c *LrcMuxClient) SearchLyrics(artist, title string, durationMs int, level string, isrc string) (*LyricsResult, error) {
	endpoint := fmt.Sprintf("%s/get", c.baseURL)
	params := url.Values{}
	params.Set("artist", artist)
	params.Set("title", title)
	if level == "" {
		level = "line"
	}
	params.Set("level", level) // "line" (default) or "word"
	if durationMs > 0 {
		params.Set("duration", strconv.Itoa(durationMs/1000))
	}
	if isrc != "" {
		params.Set("isrc", isrc)
	}

	req, err := http.NewRequest("GET", endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("lrcmux: creating request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("lrcmux: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("lrcmux: returned %d: %s", resp.StatusCode, string(body))
	}

	var data lrcmuxResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("lrcmux: decoding response: %w", err)
	}

	// Prefer the provider's official source name for the badge; fall back to
	// the source id when the API omits the name.
	srcName := data.Meta.Source.Name
	if srcName == "" {
		srcName = data.Meta.Source.ID
	}

	// Instrumental tracks come back with meta.instrumental=true and empty
	// lines. Return an instrumental result so the app renders the "— ♪ —"
	// placeholder instead of "Lyrics not found".
	if data.Meta.Instrumental {
		return &LyricsResult{
			Source:       fmt.Sprintf("lrcmux/%s", srcName),
			Instrumental: true,
			SyncLevel:    "none",
			ISRC:         data.Track.ISRC,
			DurationMs:   int(data.Track.Duration) * 1000,
		}, nil
	}

	if len(data.Lines) == 0 {
		return nil, nil
	}

	result := &LyricsResult{
		Source:     fmt.Sprintf("lrcmux/%s", srcName),
		Synced:     true,
		SyncLevel:  data.Meta.Level,
		ISRC:       data.Track.ISRC,
		DurationMs: int(data.Track.Duration) * 1000,
	}

	// Map cover art from lrcmux response (Deezer CDN).
	if data.Track.Cover != nil {
		result.CoverArt = &CoverArt{
			Small:  data.Track.Cover.Small,
			Medium: data.Track.Cover.Medium,
			Big:    data.Track.Cover.Big,
		}
	}

	// Build line-level LRC text from the response, and LyricLine entries.
	var lrcLines []string
	for _, l := range data.Lines {
		if l.Text == "" {
			continue
		}
		timeMs := l.Start
		t := &timeMs
		// If start is 0 (unsynced line), use nil.
		if timeMs <= 0 {
			t = nil
		}

		// Map the line end timestamp when present (karaoke fill uses it).
		var endMs *int
		if l.End > 0 {
			e := l.End
			endMs = &e
		}

		// Map word-level timestamps.
		var words []LyricWord
		for _, w := range l.Words {
			words = append(words, LyricWord{
				Text:    w.Text,
				StartMs: w.Start,
				EndMs:   w.End,
			})
		}

		result.Lines = append(result.Lines, LyricLine{
			TimeMs: t,
			EndMs:  endMs,
			Text:   l.Text,
			Words:  words,
		})

		if t != nil {
			minutes := *t / 60000
			seconds := (*t % 60000) / 1000
			millis := *t % 1000
			lrcLines = append(lrcLines, fmt.Sprintf("[%02d:%02d.%02d]%s", minutes, seconds, millis/10, l.Text))
		} else {
			lrcLines = append(lrcLines, l.Text)
		}
	}

	// If all lines had timestamps, mark as synced and include LRC text.
	if result.Synced && len(lrcLines) > 0 {
		result.Lyrics = joinLRC(lrcLines)
	} else {
		result.Synced = false
		for _, l := range data.Lines {
			if l.Text != "" {
				result.Lyrics += l.Text + "\n"
			}
		}
	}

	return result, nil
}

func joinLRC(lines []string) string {
	var result string
	for i, l := range lines {
		if i > 0 {
			result += "\n"
		}
		result += l
	}
	return result
}
