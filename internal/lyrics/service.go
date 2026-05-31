package lyrics

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/imov/lyricsync-translator/internal/cache"
	"github.com/imov/lyricsync-translator/internal/translate"
)

// Service orchestrates lyrics fetching, caching, and translation.
type Service struct {
	provider LyricsProvider
	store    *cache.Store
	tranSvc  *translate.Service
	OnUpdate func(songID string) // called after async translations complete
}

// NewService creates a new lyrics orchestrator.
func NewService(provider LyricsProvider, store *cache.Store, tranSvc *translate.Service) *Service {
	return &Service{provider: provider, store: store, tranSvc: tranSvc}
}

// ResolveSong gets or fetches lyrics for a track, returning full SongData.
// Returns cached data immediately if available. Otherwise fetches from provider,
// caches, and triggers async translation.
func (s *Service) ResolveSong(ctx context.Context, artist, title, album string) (*SongData, error) {
	hashKey := HashKey(artist, title, album)

	// Check cache first
	song, err := s.store.GetSongByHash(hashKey)
	if err != nil {
		return nil, fmt.Errorf("cache lookup: %w", err)
	}

	if song != nil {
		lines, _ := s.store.GetLyricLines(song.ID)
		translations, _ := s.store.GetTranslationsBySong(song.ID)

		// Retry translation if any lines are missing translations
		var missingTexts []string
		var missingLines []cache.LyricLine
		for _, l := range lines {
			if _, ok := translations[l.ID]; !ok {
				missingLines = append(missingLines, l)
				missingTexts = append(missingTexts, l.Original)
			}
		}
		if len(missingLines) > 0 {
			log.Printf("Retrying translation for %d/%d lines of %s", len(missingLines), len(lines), title)
			go s.translateLines(context.Background(), missingLines, missingTexts)
		}

		return buildSongData(song, lines, translations, len(missingLines) > 0), nil
	}

	// Fetch from provider
	log.Printf("Fetching lyrics for: %s - %s", artist, title)
	result, err := s.provider.SearchLyrics(artist, title)
	if err != nil {
		return nil, fmt.Errorf("lyrics search: %w", err)
	}
	if result == nil || len(result.Lines) == 0 {
		return nil, nil
	}

	// Save song to DB
	song = &cache.Song{
		HashKey: hashKey,
		Artist:  artist,
		Title:   title,
		Album:   album,
		Source:  result.Source,
	}
	if err := s.store.SaveSong(song); err != nil {
		return nil, fmt.Errorf("saving song: %w", err)
	}

	// Save lyric lines
	var cacheLines []cache.LyricLine
	var origTexts []string
	for i, l := range result.Lines {
		timeMs := new(int)
		*timeMs = l.TimeMs
		lang := translate.DetectLanguage(l.Text)

		cacheLines = append(cacheLines, cache.LyricLine{
			SongID:   song.ID,
			LineNum:  i + 1,
			TimeMs:   timeMs,
			Original: l.Text,
			Lang:     lang,
		})
		origTexts = append(origTexts, l.Text)
	}
	if err := s.store.SaveLyricLines(song.ID, cacheLines); err != nil {
		return nil, fmt.Errorf("saving lyric lines: %w", err)
	}

	// Reload lines to get their DB IDs
	storedLines, err := s.store.GetLyricLines(song.ID)
	if err != nil {
		return nil, fmt.Errorf("reloading lines: %w", err)
	}

	// Start async translation
	go s.translateLines(context.Background(), storedLines, origTexts)

	return buildSongData(song, storedLines, nil, true), nil
}

func (s *Service) translateLines(ctx context.Context, storedLines []cache.LyricLine, origTexts []string) {
	results, err := s.tranSvc.ProcessLines(ctx, origTexts)
	if err != nil {
		log.Printf("Translation error: %v", err)
		return
	}

	for i, r := range results {
		if i >= len(storedLines) {
			break
		}
		t := &cache.Translation{
			LyricLineID:  storedLines[i].ID,
			Romanized:    r.Romanized,
			TranslatedES: r.Translated,
		}
		if err := s.store.SaveTranslation(t); err != nil {
			log.Printf("Error saving translation for line %d: %v", i, err)
		}
	}
	log.Printf("Translations saved for %d lines", len(results))
	if s.OnUpdate != nil {
		s.OnUpdate(storedLines[0].SongID)
	}
}

// SongData is the full data returned by ResolveSong, serialized as JSON for SSE.
type SongData struct {
	Type        string     `json:"type"`
	Song        *SongInfo  `json:"song"`
	Lines       []LineData `json:"lines"`
	Translating bool       `json:"translating,omitempty"`
}

// SongInfo holds song metadata for the SSE event.
type SongInfo struct {
	ID         string `json:"id"`
	HashKey    string `json:"hash_key"`
	Artist     string `json:"artist"`
	Title      string `json:"title"`
	Album      string `json:"album,omitempty"`
	DurationMs int    `json:"duration_ms,omitempty"`
	Source     string `json:"source"`
}

// LineData holds one line of lyrics with optional romanization and translation.
type LineData struct {
	ID         int    `json:"id"`
	TimeMs     *int   `json:"time_ms,omitempty"`
	Original   string `json:"original"`
	Romanized  string `json:"romanized,omitempty"`
	Translated string `json:"translated,omitempty"`
}

func buildSongData(song *cache.Song, lines []cache.LyricLine, translations map[int]*cache.Translation, translating bool) *SongData {
	data := &SongData{
		Type:        "lyrics",
		Translating: translating,
		Song: &SongInfo{
			ID:         song.ID,
			HashKey:    song.HashKey,
			Artist:     song.Artist,
			Title:      song.Title,
			Album:      song.Album,
			DurationMs: song.DurationMs,
			Source:     song.Source,
		},
		Lines: make([]LineData, len(lines)),
	}

	for i, l := range lines {
		ld := LineData{
			ID:       l.ID,
			TimeMs:   l.TimeMs,
			Original: l.Original,
		}
		if translations != nil {
			if t, ok := translations[l.ID]; ok {
				ld.Romanized = t.Romanized
				ld.Translated = t.TranslatedES
			}
		}
		data.Lines[i] = ld
	}

	return data
}

// HashKey creates a deterministic hash from artist, title, and album.
func HashKey(artist, title, album string) string {
	h := sha256.New()
	h.Write([]byte(artist + "|" + title + "|" + album))
	return hex.EncodeToString(h.Sum(nil))
}


