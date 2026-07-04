package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
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
	router                  chi.Router
	httpSrv                 *http.Server
	cancel                  context.CancelFunc
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
	r := chi.NewRouter()

	// CORS: the Wails webview loads from an internal scheme (e.g. wails://)
	// but makes HTTP requests to the API server. Without CORS headers,
	// fetch() calls are blocked by the webview.
	r.Use(corsMiddleware)

	s := &Server{
		cfg:       cfg,
		store:     store,
		tracker:   tracker,
		tranSvc:   tranSvc,
		lyricsSvc: lyricsSvc,
		sse:       sse,
		router:    r,
	}

	// Register callback: when async translations finish, republish updated lyrics
	lyricsSvc.OnUpdate = func(songID string) {
		log.Printf("OnUpdate callback fired for song %s", songID)
		lines, err := store.GetLyricLines(songID)
		if err != nil {
			log.Printf("OnUpdate: error getting lines: %v", err)
			return
		}
		targetLang := tranSvc.TargetLang()
		translations, err := store.GetTranslationsBySong(songID, targetLang)
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
				ld[i].Translated = t.TranslatedText
			}
		}
		payload, _ := json.Marshal(updateEvent{Type: "translations", Lines: ld})
		log.Printf("OnUpdate: publishing translations event (%d bytes)", len(payload))
		s.sse.Publish(payload)
		s.lastTranslationsPayload = payload
	}

	// Register error callback: emits a lyrics_error event so the frontend
	// can show an error message instead of leaving the shimmer visible.
	lyricsSvc.OnError = func(songID string, errMsg string) {
		log.Printf("OnError callback: song=%s error=%s", songID, errMsg)
		payload, _ := json.Marshal(map[string]interface{}{
			"type":  "lyrics_error",
			"error": fmt.Sprintf("Translation failed: %s", errMsg),
			"retry": true,
		})
		s.sse.Publish(payload)
	}

	// API routes
	r.Get("/api/now-playing", s.handleNowPlaying)
	r.Get("/api/songs", s.handleListSongs)
	r.Get("/api/songs/{hash}/lyrics", s.handleGetLyrics)
	r.Get("/api/lyrics/stream", s.handleSSE)
	r.Post("/api/lyrics/retry", s.handleRetryLyrics)
	r.Get("/api/config", s.handleGetConfig)
	r.Put("/api/config", s.handleUpdateConfig)
	r.Put("/api/config/provider", s.handleUpdateProvider)
	r.Get("/api/songs/{hash}/offset", s.handleGetOffset)
	r.Put("/api/songs/{hash}/offset", s.handleUpdateOffset)
	r.Post("/api/player/toggle", s.handleTogglePlayPause)
	r.Post("/api/player/seek", s.handleSeek)
	r.Post("/api/player/next", s.handleNext)
	r.Post("/api/player/previous", s.handlePrevious)
	r.Get("/api/player/volume", s.handleGetVolume)
	r.Post("/api/player/volume", s.handleVolume)
	r.Post("/api/player/shuffle", s.handleShuffle)
	r.Get("/api/player/shuffle", s.handleGetShuffle)
	r.Post("/api/player/loop", s.handleLoop)
	r.Get("/api/player/loop", s.handleGetLoop)

	// Player selection
	r.Get("/api/players", s.handleListPlayers)
	r.Get("/api/players/active", s.handleGetActivePlayer)
	r.Post("/api/players/active", s.handleSetActivePlayer)

	return s
}

// EnableEmbeddedAssets registers the SPA catch-all handler on the chi router.
// Must be called before Start().
//
// In production (devServerURL is empty): serves frontend from the embedded
// web/dist filesystem with SPA fallback and API base URL injection.
//
// In dev mode (devServerURL is set): reverse-proxies non-API requests to
// the Vite dev server so HMR works. API routes (/api/*) are matched by
// chi before reaching this catch-all, so they still hit the Go backend.
func (s *Server) EnableEmbeddedAssets(assetsFS fs.FS, apiBase string, devServerURL string) {
	if devServerURL != "" {
		s.router.Handle("/*", s.devProxyHandler(devServerURL))
	} else {
		s.router.Handle("/*", s.spaHandler(assetsFS, apiBase))
	}
}

// devProxyHandler creates a reverse proxy to the Vite dev server.
// Used in dev mode so HMR works while API routes still hit Go.
func (s *Server) devProxyHandler(targetURL string) http.HandlerFunc {
	target, err := url.Parse(targetURL)
	if err != nil {
		panic("invalid dev server URL: " + targetURL)
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	return func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r)
	}
}

// Handler returns the chi router for use as Wails AssetServer.Handler.
func (s *Server) Handler() http.Handler {
	return s.router
}

// spaHandler serves embedded frontend assets with SPA fallback.
// If the requested file exists in assetsFS, it is served directly.
// Otherwise, index.html is served with the API base URL injected via
// string replacement of the {{.APIBase}} template placeholder.
func (s *Server) spaHandler(assetsFS fs.FS, apiBase string) http.HandlerFunc {
	fileServer := http.FileServer(http.FS(assetsFS))

	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Try to serve the requested file from embedded assets.
		f, err := assetsFS.Open(strings.TrimPrefix(path, "/"))
		if err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		// SPA fallback: serve index.html with API base injected.
		indexContent, err := fs.ReadFile(assetsFS, "index.html")
		if err != nil {
			http.Error(w, "index.html not found in embedded assets", http.StatusInternalServerError)
			return
		}

		// Replace the Go template placeholder with the actual API base.
		content := strings.Replace(string(indexContent), "{{.APIBase}}", apiBase, 1)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(content))
	}
}

func (s *Server) Start(ctx context.Context) {
	s.httpSrv = &http.Server{
		Addr:    s.cfg.Server.Address(),
		Handler: s.router,
	}

	sseCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.sse.Start(sseCtx)
	go s.pipeTrackerEvents(sseCtx)

	go func() {
		fmt.Printf("Server listening on %s\n", s.cfg.Server.Address())
		if err := s.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Server error: %v", err)
		}
	}()
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.cancel != nil {
		s.cancel()
	}
	if s.httpSrv != nil {
		return s.httpSrv.Shutdown(ctx)
	}
	return nil
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

func (s *Server) handleSeek(w http.ResponseWriter, r *http.Request) {
	var body struct {
		PositionMs int `json:"position_ms"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}
	// Guard: don't seek to negative positions
	if body.PositionMs < 0 {
		http.Error(w, `{"error":"position must be >= 0"}`, http.StatusBadRequest)
		return
	}
	if err := player.SetPosition(s.cfg.Player.PlayerctlPath, s.tracker.GetActivePlayer(), body.PositionMs); err != nil {
		log.Printf("Error seeking: %v", err)
		http.Error(w, `{"error":"failed to seek"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

func (s *Server) handleNext(w http.ResponseWriter, r *http.Request) {
	if err := player.Next(s.cfg.Player.PlayerctlPath, s.tracker.GetActivePlayer()); err != nil {
		log.Printf("Error skipping next: %v", err)
		http.Error(w, `{"error":"failed to skip"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

func (s *Server) handlePrevious(w http.ResponseWriter, r *http.Request) {
	if err := player.Previous(s.cfg.Player.PlayerctlPath, s.tracker.GetActivePlayer()); err != nil {
		log.Printf("Error skipping previous: %v", err)
		http.Error(w, `{"error":"failed to skip"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

func (s *Server) handleGetVolume(w http.ResponseWriter, r *http.Request) {
	vol, err := player.GetVolume(s.cfg.Player.PlayerctlPath, s.tracker.GetActivePlayer())
	if err != nil {
		log.Printf("Error getting volume: %v", err)
		http.Error(w, `{"error":"failed to get volume"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"volume":%.2f}`, vol)
}

func (s *Server) handleVolume(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Delta    float64  `json:"delta"`
		Absolute *float64 `json:"absolute"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}

	var err error
	if body.Absolute != nil {
		err = player.SetAbsoluteVolume(s.cfg.Player.PlayerctlPath, s.tracker.GetActivePlayer(), *body.Absolute)
	} else {
		err = player.SetVolume(s.cfg.Player.PlayerctlPath, s.tracker.GetActivePlayer(), body.Delta)
	}

	if err != nil {
		log.Printf("Error setting volume: %v", err)
		http.Error(w, `{"error":"failed to set volume"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

func (s *Server) handleShuffle(w http.ResponseWriter, r *http.Request) {
	if err := player.Shuffle(s.cfg.Player.PlayerctlPath, s.tracker.GetActivePlayer()); err != nil {
		log.Printf("Error toggling shuffle: %v", err)
		http.Error(w, `{"error":"failed to toggle shuffle"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

func (s *Server) handleLoop(w http.ResponseWriter, r *http.Request) {
	state, err := player.Loop(s.cfg.Player.PlayerctlPath, s.tracker.GetActivePlayer())
	if err != nil {
		log.Printf("Error cycling loop: %v", err)
		http.Error(w, `{"error":"failed to cycle loop"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"loop": state})
}

func (s *Server) handleGetShuffle(w http.ResponseWriter, r *http.Request) {
	state, err := player.GetShuffle(s.cfg.Player.PlayerctlPath, s.tracker.GetActivePlayer())
	if err != nil {
		log.Printf("Error reading shuffle state: %v", err)
		http.Error(w, `{"error":"failed to read shuffle"}`, http.StatusInternalServerError)
		return
	}
	on := state == "On"
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"shuffle":%t}`, on)
}

func (s *Server) handleGetLoop(w http.ResponseWriter, r *http.Request) {
	state, err := player.GetLoop(s.cfg.Player.PlayerctlPath, s.tracker.GetActivePlayer())
	if err != nil {
		log.Printf("Error reading loop state: %v", err)
		http.Error(w, `{"error":"failed to read loop"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"loop": state})
}

// corsMiddleware adds permissive CORS headers for the Wails webview.
// The webview loads pages from an internal scheme (wails://) but
// makes real HTTP requests to the API server, which are cross-origin.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
