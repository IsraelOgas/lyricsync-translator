package lyrics

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	OnUpdate func(songID string)                // called after async translations complete
	OnError  func(songID string, errMsg string) // called when translation fails
}

// NewService creates a new lyrics orchestrator.
func NewService(provider LyricsProvider, store *cache.Store, tranSvc *translate.Service) *Service {
	return &Service{provider: provider, store: store, tranSvc: tranSvc}
}

// ResolveOptions controls a lyrics resolution.
type ResolveOptions struct {
	// DurationMs is the track duration in milliseconds (0 if unknown).
	DurationMs int
	// Level is the highest sync level to request: "line" (default) or "word".
	Level string
	// Force bypasses the cache and re-fetches from the provider.
	Force bool
}

// ResolveSong gets or fetches lyrics for a track, returning full SongData.
// Returns cached data immediately if available. Otherwise fetches from provider,
// caches, and triggers async translation.
func (s *Service) ResolveSong(ctx context.Context, artist, title, album string, opts ResolveOptions) (*SongData, error) {
	hashKey := HashKey(artist, title, album)

	// Check cache first
	song, err := s.store.GetSongByHash(hashKey)
	if err != nil {
		return nil, fmt.Errorf("cache lookup: %w", err)
	}

	if song != nil && !opts.Force {
		lines, _ := s.store.GetLyricLines(song.ID)

		// Translation disabled: return lyrics without any translations/romanization
		// and never trigger provider calls (no API cost).
		if !s.tranSvc.Enabled() {
			return buildSongData(song, lines, nil, false), nil
		}

		targetLang := s.tranSvc.TargetLang()
		translations, _ := s.store.GetTranslationsBySong(song.ID, targetLang)

		// Retry translation if any lines are missing translations or have empty text
		// (empty translations happen when the provider failed but we cached them anyway).
		// Empty lines (instrumental markers) never need translation.
		var missingTexts []string
		var missingLines []cache.LyricLine
		for _, l := range lines {
			if l.Original == "" {
				continue
			}
			t, ok := translations[l.ID]
			if !ok || t.TranslatedText == "" {
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
	log.Printf("Fetching lyrics for: %s - %s (provider: %s)", artist, title, s.provider.Name())
	// Reuse the stored ISRC when re-fetching (it takes priority in lrcmux).
	// song is loaded by GetSongByHash above even when Force is set, so it may
	// carry an ISRC from a previous fetch.
	searchISRC := ""
	if song != nil {
		searchISRC = song.ISRC
	}
	result, err := s.provider.SearchLyrics(artist, title, opts.DurationMs, opts.Level, searchISRC)
	if err != nil {
		return nil, fmt.Errorf("lyrics search: %w", err)
	}
	if result == nil {
		return nil, nil
	}
	if result.Instrumental {
		result.Lines = []LyricLine{{Text: ""}}
	}
	if len(result.Lines) == 0 {
		return nil, nil
	}

	// Derive sync level when the provider doesn't report it explicitly.
	if result.SyncLevel == "" {
		switch {
		case hasWords(result.Lines):
			result.SyncLevel = "word"
		case hasTimestamps(result.Lines):
			result.SyncLevel = "line"
		default:
			result.SyncLevel = "none"
		}
	}

	// Save song to DB. If the song already exists in cache (Force re-fetch),
	// keep its ID so lyric lines UPSERT onto the same song_id and existing
	// translations stay attached instead of being orphaned.
	songID := ""
	if song != nil {
		songID = song.ID
	}
	song = &cache.Song{
		ID:         songID,
		HashKey:    hashKey,
		Artist:     artist,
		Title:      title,
		Album:      album,
		Source:     result.Source,
		ISRC:       result.ISRC,
		SyncLevel:  result.SyncLevel,
		DurationMs: result.DurationMs,
	}
	// Fall back to the player-reported duration when the provider doesn't
	// return a canonical one.
	if song.DurationMs == 0 && opts.DurationMs > 0 {
		song.DurationMs = opts.DurationMs
	}
	// Serialize cover art to JSON for storage.
	if result.CoverArt != nil {
		if b, err := json.Marshal(result.CoverArt); err == nil {
			song.CoverArtJSON = string(b)
		}
	}
	if err := s.store.SaveSong(song); err != nil {
		return nil, fmt.Errorf("saving song: %w", err)
	}
	log.Printf("Cached lyrics for %s - %s (source: %s, lines: %d)", artist, title, result.Source, len(result.Lines))

	// Save lyric lines
	var cacheLines []cache.LyricLine
	var origTexts []string
	for i, l := range result.Lines {
		var timeMs *int
		if l.TimeMs != nil {
			t := new(int)
			*t = *l.TimeMs
			timeMs = t
		}
		lang := translate.DetectLanguage(l.Text)

		cl := cache.LyricLine{
			SongID:   song.ID,
			LineNum:  i + 1,
			TimeMs:   timeMs,
			EndMs:    l.EndMs,
			Original: l.Text,
			Lang:     lang,
		}
		// Serialize word-level timestamps to JSON.
		if len(l.Words) > 0 {
			if b, err := json.Marshal(l.Words); err == nil {
				cl.WordsJSON = string(b)
			}
		}
		cacheLines = append(cacheLines, cl)
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

	// Start async translation (only when enabled, and never for instrumentals).
	if s.tranSvc.Enabled() && !result.Instrumental {
		go s.translateLines(context.Background(), storedLines, origTexts)
		return buildSongData(song, storedLines, nil, true), nil
	}

	return buildSongData(song, storedLines, nil, false), nil
}

func (s *Service) translateLines(ctx context.Context, storedLines []cache.LyricLine, origTexts []string) {
	results, err := s.tranSvc.ProcessLines(ctx, origTexts)
	if err != nil {
		log.Printf("Translation error: %v", err)
		// Notify frontend even on error so it clears the shimmer.
		if s.OnUpdate != nil {
			s.OnUpdate(storedLines[0].SongID)
		}
		if s.OnError != nil {
			s.OnError(storedLines[0].SongID, err.Error())
		}
		return
	}

	for i, r := range results {
		if i >= len(storedLines) {
			break
		}
		t := &cache.Translation{
			LyricLineID:    storedLines[i].ID,
			Romanized:      r.Romanized,
			TranslatedText: r.Translated,
			TargetLang:     s.tranSvc.TargetLang(),
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
	ID         string    `json:"id"`
	HashKey    string    `json:"hash_key"`
	Artist     string    `json:"artist"`
	Title      string    `json:"title"`
	Album      string    `json:"album,omitempty"`
	DurationMs int       `json:"duration_ms,omitempty"`
	ISRC       string    `json:"isrc,omitempty"`
	SyncLevel  string    `json:"sync_level,omitempty"`
	OffsetMs   int       `json:"offset_ms"`
	Source     string    `json:"source"`
	CoverArt   *CoverArt `json:"cover_art,omitempty"`
}

// LineData holds one line of lyrics with optional romanization and translation.
type LineData struct {
	ID         int         `json:"id"`
	TimeMs     *int        `json:"time_ms,omitempty"`
	EndMs      *int        `json:"end_ms,omitempty"`
	Original   string      `json:"original"`
	Romanized  string      `json:"romanized,omitempty"`
	Translated string      `json:"translated,omitempty"`
	Words      []LyricWord `json:"words,omitempty"`
}

func buildSongData(song *cache.Song, lines []cache.LyricLine, translations map[int]*cache.Translation, translating bool) *SongData {
	// Deserialize cover art from cache.
	var coverArt *CoverArt
	if song.CoverArtJSON != "" {
		var ca CoverArt
		if err := json.Unmarshal([]byte(song.CoverArtJSON), &ca); err == nil {
			coverArt = &ca
		}
	}

	// Derive sync level for legacy cached songs that predate sync_level.
	if song.SyncLevel == "" {
		switch {
		case hasWordsFromCache(lines):
			song.SyncLevel = "word"
		case hasTimestampsFromCache(lines):
			song.SyncLevel = "line"
		default:
			song.SyncLevel = "none"
		}
	}

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
			ISRC:       song.ISRC,
			SyncLevel:  song.SyncLevel,
			OffsetMs:   song.OffsetMs,
			Source:     song.Source,
			CoverArt:   coverArt,
		},
		Lines: make([]LineData, len(lines)),
	}

	for i, l := range lines {
		ld := LineData{
			ID:       l.ID,
			TimeMs:   l.TimeMs,
			EndMs:    l.EndMs,
			Original: l.Original,
		}
		// Only include timestamps if they're real positive values.
		// Zero means "unsynced" (either fresh nil or stale 0 from old cache).
		if l.TimeMs != nil && *l.TimeMs <= 0 {
			ld.TimeMs = nil
		}
		if l.EndMs != nil && *l.EndMs <= 0 {
			ld.EndMs = nil
		}
		if translations != nil {
			if t, ok := translations[l.ID]; ok {
				ld.Romanized = t.Romanized
				ld.Translated = t.TranslatedText
			}
		}
		// Deserialize word-level timestamps from cache.
		if l.WordsJSON != "" {
			var words []LyricWord
			if err := json.Unmarshal([]byte(l.WordsJSON), &words); err == nil {
				ld.Words = words
			}
		}
		data.Lines[i] = ld
	}

	return data
}

// hasWords reports whether any line carries word-level timestamps.
func hasWords(lines []LyricLine) bool {
	for _, l := range lines {
		if len(l.Words) > 0 {
			return true
		}
	}
	return false
}

// hasTimestamps reports whether any line carries a line-level timestamp.
func hasTimestamps(lines []LyricLine) bool {
	for _, l := range lines {
		if l.TimeMs != nil && *l.TimeMs > 0 {
			return true
		}
	}
	return false
}

// hasWordsFromCache reports whether any cached line carries word-level timestamps.
func hasWordsFromCache(lines []cache.LyricLine) bool {
	for _, l := range lines {
		if l.WordsJSON != "" {
			return true
		}
	}
	return false
}

// hasTimestampsFromCache reports whether any cached line carries a line-level timestamp.
func hasTimestampsFromCache(lines []cache.LyricLine) bool {
	for _, l := range lines {
		if l.TimeMs != nil && *l.TimeMs > 0 {
			return true
		}
	}
	return false
}

// HashKey creates a deterministic hash from artist, title, and album.
func HashKey(artist, title, album string) string {
	h := sha256.New()
	h.Write([]byte(artist + "|" + title + "|" + album))
	return hex.EncodeToString(h.Sum(nil))
}
