package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/imov/lyricsync-translator/internal/cache"
	"github.com/imov/lyricsync-translator/internal/config"
	"github.com/imov/lyricsync-translator/internal/lyrics"
	"github.com/imov/lyricsync-translator/internal/player"
	"github.com/imov/lyricsync-translator/internal/translate"
)

type Server struct {
	cfg                     *config.Config
	store                   *cache.Store
	tracker                 *player.Tracker
	tranSvc                 *translate.Service
	lyricsSvc               *lyrics.Service
	sse                     *SSEBroker
	httpSrv                 *http.Server
	lastTrackPayload        []byte
	lastLyricsPayload       []byte
	lastTranslationsPayload []byte
}

func NewServer(
	cfg *config.Config,
	store *cache.Store,
	tracker *player.Tracker,
	tranSvc *translate.Service,
	lyricsSvc *lyrics.Service,
) *Server {
	sse := NewSSEBroker()
	s := &Server{
		cfg:       cfg,
		store:     store,
		tracker:   tracker,
		tranSvc:   tranSvc,
		lyricsSvc: lyricsSvc,
		sse:       sse,
	}

	// Register callback: when async translations finish, republish updated lyrics
	lyricsSvc.OnUpdate = func(songID string) {
		log.Printf("OnUpdate callback fired for song %s", songID)
		lines, err := store.GetLyricLines(songID)
		if err != nil {
			log.Printf("OnUpdate: error getting lines: %v", err)
			return
		}
		translations, err := store.GetTranslationsBySong(songID)
		if err != nil {
			log.Printf("OnUpdate: error getting translations: %v", err)
			return
		}
		log.Printf("OnUpdate: %d lines, %d translations", len(lines), len(translations))
		type lineData struct {
			ID         int    `json:"id"`
			TimeMs     *int   `json:"time_ms,omitempty"`
			Original   string `json:"original"`
			Romanized  string `json:"romanized,omitempty"`
			Translated string `json:"translated,omitempty"`
		}
		type updateEvent struct {
			Type  string     `json:"type"`
			Lines []lineData `json:"lines"`
		}

		ld := make([]lineData, len(lines))
		for i, l := range lines {
			ld[i] = lineData{ID: l.ID, TimeMs: l.TimeMs, Original: l.Original}
			if t, ok := translations[l.ID]; ok {
				ld[i].Romanized = t.Romanized
				ld[i].Translated = t.TranslatedES
			}
		}
		payload, _ := json.Marshal(updateEvent{Type: "translations", Lines: ld})
		log.Printf("OnUpdate: publishing translations event (%d bytes)", len(payload))
		s.sse.Publish(payload)
		s.lastTranslationsPayload = payload
	}

	r := chi.NewRouter()
	r.Get("/api/now-playing", s.handleNowPlaying)
	r.Get("/api/songs/{hash}/lyrics", s.handleGetLyrics)
	r.Get("/api/lyrics/stream", s.handleSSE)
	r.Get("/api/config", s.handleGetConfig)
	r.Put("/api/config", s.handleUpdateConfig)
	r.Post("/api/player/toggle", s.handleTogglePlayPause)

	// Serve frontend static files in production
	webDir := os.Getenv("WEB_DIR")
	if webDir == "" {
		webDir = "web/dist"
	}
	if absDir, err := filepath.Abs(webDir); err == nil {
		if info, err := os.Stat(absDir); err == nil && info.IsDir() {
			fs := http.FileServer(http.Dir(absDir))
			r.Handle("/*", fs)
			log.Printf("Serving frontend from %s", absDir)
		}
	}

	s.httpSrv = &http.Server{
		Addr:    cfg.Server.Address(),
		Handler: r,
	}

	return s
}

func (s *Server) Start(ctx context.Context) error {
	brokerCtx, brokerCancel := context.WithCancel(ctx)
	defer brokerCancel()
	s.sse.Start(brokerCtx)

	go s.pipeTrackerEvents(ctx)

	fmt.Printf("Server listening on %s\n", s.cfg.Server.Address())
	return s.httpSrv.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpSrv.Shutdown(ctx)
}

func (s *Server) pipeTrackerEvents(ctx context.Context) {
	events := s.tracker.Events(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			s.sse.Publish(event)

			var evt player.TrackerEvent
			if err := json.Unmarshal(event, &evt); err == nil && evt.Type == "track" && evt.Track != nil {
				s.lastTrackPayload = event
				go s.resolveAndPublishLyrics(ctx, evt.Track)
			}
		}
	}
}

func (s *Server) resolveAndPublishLyrics(ctx context.Context, track *player.TrackInfo) {
	log.Printf("Resolving lyrics for: %s - %s", track.Artist, track.Title)

	// Notify frontend that lyrics fetch has started
	loadingPayload, _ := json.Marshal(map[string]string{"type": "lyrics_loading"})
	s.sse.Publish(loadingPayload)

	data, err := s.lyricsSvc.ResolveSong(ctx, track.Artist, track.Title, track.Album)
	if err != nil {
		log.Printf("Error resolving lyrics (attempt 1): %v — retrying in 1s", err)
		time.Sleep(1 * time.Second)
		data, err = s.lyricsSvc.ResolveSong(ctx, track.Artist, track.Title, track.Album)
		if err != nil {
			log.Printf("Error resolving lyrics (attempt 2): %v — giving up", err)
			errorPayload, _ := json.Marshal(map[string]interface{}{
				"type":  "lyrics_error",
				"error": fmt.Sprintf("Failed to load lyrics: %v", err),
				"retry": true,
			})
			s.sse.Publish(errorPayload)
			return
		}
	}
	if data == nil {
		log.Printf("No lyrics found for: %s - %s", track.Artist, track.Title)
		// Send empty lyrics event to clear frontend
		payload, _ := json.Marshal(map[string]interface{}{
			"type": "lyrics",
			"lines": []interface{}{},
			"not_found": true,
		})
		s.sse.Publish(payload)
		return
	}

	payload, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshaling lyrics event: %v", err)
		return
	}
	s.sse.Publish(payload)
	s.lastLyricsPayload = payload
	log.Printf("Lyrics published for: %s - %s (%d lines)", track.Artist, track.Title, len(data.Lines))
}

func (s *Server) handleTogglePlayPause(w http.ResponseWriter, r *http.Request) {
	if err := player.TogglePlayPause(s.cfg.Player.PlayerctlPath, s.tracker.GetActivePlayer()); err != nil {
		log.Printf("Error toggling play-pause: %v", err)
		http.Error(w, `{"error":"failed to toggle"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}
