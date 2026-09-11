package lyrics

// userAgent identifies this app to lyrics APIs (lrcmux, LRCLIB) as recommended
// by their docs. Keep the repo URL up to date.
const userAgent = "lyricsync-translator/1.0 (https://github.com/IsraelOgas/lyricsync-translator)"

// LyricsProvider is the interface that must be implemented by each lyrics source.
type LyricsProvider interface {
	// Name returns the provider identifier (e.g., "lrclib").
	Name() string

	// SearchLyrics searches for lyrics by artist, title, track duration, and ISRC.
	// durationMs is the track duration in milliseconds (0 if unknown).
	// level is the highest sync level to request: "line" (default) or "word"
	// (only meaningful for lrcmux; LRCLIB ignores it).
	// isrc is the track ISRC if known (takes priority in lrcmux; empty string if unknown).
	// Returns nil, nil if no lyrics found.
	SearchLyrics(artist, title string, durationMs int, level string, isrc string) (*LyricsResult, error)
}

// LyricsResult holds parsed lyrics data from a provider.
type LyricsResult struct {
	Source       string      `json:"source"`
	Synced       bool        `json:"synced"`
	Lyrics       string      `json:"lyrics"`
	Lines        []LyricLine `json:"lines,omitempty"`
	CoverArt     *CoverArt   `json:"cover_art,omitempty"`
	Instrumental bool        `json:"instrumental,omitempty"`
	ISRC         string      `json:"isrc,omitempty"`
	DurationMs   int         `json:"duration_ms,omitempty"`
	// SyncLevel is the actual sync level of the returned lyrics:
	// "word", "line", or "none". Empty when the provider doesn't report it.
	SyncLevel string `json:"sync_level,omitempty"`
}

// CoverArt holds cover image URLs from a provider (typically Deezer CDN).
type CoverArt struct {
	Small  string `json:"small"`
	Medium string `json:"medium"`
	Big    string `json:"big"`
}

// LyricLine is one parsed line with optional timing.
type LyricLine struct {
	TimeMs *int        `json:"time_ms,omitempty"`
	EndMs  *int        `json:"end_ms,omitempty"`
	Text   string      `json:"text"`
	Words  []LyricWord `json:"words,omitempty"`
}

// LyricWord is one word within a lyric line with optional word-level timing.
type LyricWord struct {
	Text    string `json:"text"`
	StartMs int    `json:"start_ms"`
	EndMs   int    `json:"end_ms"`
}

// NewProvider returns the configured lyrics provider by name.
func NewProvider(name, baseURL string, timeoutSec int) LyricsProvider {
	switch name {
	case "lrclib":
		return NewLRCLibClient(baseURL, timeoutSec)
	case "lrcmux":
		return NewLrcMuxClient(baseURL, timeoutSec)
	default:
		return nil
	}
}
