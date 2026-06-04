package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleNowPlaying(w http.ResponseWriter, r *http.Request) {
	track := s.tracker.GetCurrent()
	status := s.tracker.GetStatus()
	pos := s.tracker.GetPositionMs()

	resp := map[string]interface{}{
		"track":       track,
		"status":      status.String(),
		"position_ms": pos,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleGetLyrics(w http.ResponseWriter, r *http.Request) {
	hash := chi.URLParam(r, "hash")

	song, err := s.store.GetSongByHash(hash)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if song == nil {
		http.Error(w, "song not found", http.StatusNotFound)
		return
	}

	lines, err := s.store.GetLyricLines(song.ID)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	translations, err := s.store.GetTranslationsBySong(song.ID, s.tranSvc.TargetLang())
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	type lineResp struct {
		ID         int    `json:"id"`
		TimeMs     *int   `json:"time_ms,omitempty"`
		Original   string `json:"original"`
		Romanized  string `json:"romanized,omitempty"`
		Translated string `json:"translated,omitempty"`
	}

	var responseLines []lineResp
	for _, l := range lines {
		lr := lineResp{
			ID:       l.ID,
			TimeMs:   l.TimeMs,
			Original: l.Original,
		}
		if t, ok := translations[l.ID]; ok {
			lr.Romanized = t.Romanized
			lr.Translated = t.TranslatedText
		}
		responseLines = append(responseLines, lr)
	}

	resp := map[string]interface{}{
		"song":  song,
		"lines": responseLines,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.cfg)
}

func (s *Server) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TargetLang *string `json:"target_lang"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}
	if body.TargetLang != nil && *body.TargetLang != "" {
		s.tranSvc.SetTargetLang(*body.TargetLang)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"target_lang": s.tranSvc.TargetLang(),
	})
}

func (s *Server) handleGetOffset(w http.ResponseWriter, r *http.Request) {
	hash := chi.URLParam(r, "hash")
	offset, err := s.store.GetSongOffset(hash)
	if err != nil {
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"offset_ms": offset})
}

func (s *Server) handleUpdateOffset(w http.ResponseWriter, r *http.Request) {
	hash := chi.URLParam(r, "hash")
	var body struct {
		OffsetMs int `json:"offset_ms"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}
	if err := s.store.UpdateSongOffset(hash, body.OffsetMs); err != nil {
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"offset_ms": body.OffsetMs})
}
