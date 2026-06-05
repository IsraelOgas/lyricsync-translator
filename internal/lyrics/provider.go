package lyrics

// LyricsProvider is the interface that must be implemented by each lyrics source.
type LyricsProvider interface {
	// Name returns the provider identifier (e.g., "lrclib").
	Name() string

	// SearchLyrics searches for lyrics by artist and title.
	// Returns nil, nil if no lyrics found.
	SearchLyrics(artist, title string) (*LyricsResult, error)
}

// LyricsResult holds parsed lyrics data from a provider.
type LyricsResult struct {
	Source  string      `json:"source"`
	Synced  bool        `json:"synced"`
	Lyrics  string      `json:"lyrics"`
	Lines   []LyricLine `json:"lines,omitempty"`
}

// LyricLine is one parsed line with optional timing.
type LyricLine struct {
	TimeMs *int   `json:"time_ms,omitempty"`
	Text   string `json:"text"`
}

// NewProvider returns the configured lyrics provider by name.
func NewProvider(name, baseURL string, timeoutSec int) LyricsProvider {
	switch name {
	case "lrclib":
		return NewLRCLibClient(baseURL, timeoutSec)
	default:
		return nil
	}
}
