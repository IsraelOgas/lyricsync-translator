package cache

import "time"

// Song represents a cached track in the database.
type Song struct {
	ID         string    `json:"id"`
	HashKey    string    `json:"hash_key"`
	Artist     string    `json:"artist"`
	Title      string    `json:"title"`
	Album      string    `json:"album,omitempty"`
	DurationMs int       `json:"duration_ms,omitempty"`
	Source     string    `json:"source"`
	CreatedAt  time.Time `json:"created_at"`
}

// LyricLine is one line of a song's lyrics, optionally synchronized.
type LyricLine struct {
	ID       int    `json:"id"`
	SongID   string `json:"song_id"`
	LineNum  int    `json:"line_num"`
	TimeMs   *int   `json:"time_ms,omitempty"`
	Original string `json:"original"`
	Lang     string `json:"lang,omitempty"`
}

// Translation holds romanization and Spanish translation for a lyric line.
type Translation struct {
	ID           int    `json:"id"`
	LyricLineID  int    `json:"lyric_line_id"`
	Romanized    string `json:"romanized,omitempty"`
	TranslatedES string `json:"translated_es,omitempty"`
}
