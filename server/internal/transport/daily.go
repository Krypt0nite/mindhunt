package transport

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"mindhunt/internal/daily"
	"mindhunt/internal/game"
)

type bucket struct {
	tokens float64
	last   time.Time
}

// allowAction is a per-player token bucket (5 per second, burst 10) for
// daily moves and previews.
func (s *Server) allowAction(id string, now time.Time) bool {
	s.limMu.Lock()
	defer s.limMu.Unlock()
	b, ok := s.actions[id]
	if !ok {
		b = &bucket{tokens: 10, last: now}
		s.actions[id] = b
	}
	b.tokens = min(10, b.tokens+now.Sub(b.last).Seconds()*5)
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func (s *Server) dailyOff(w http.ResponseWriter) bool {
	if s.daily == nil {
		writeErr(w, http.StatusNotFound, "daily_off", "the daily challenge is not enabled on this server")
		return true
	}
	return false
}

func (s *Server) dailyGet(w http.ResponseWriter, r *http.Request) {
	if s.dailyOff(w) {
		return
	}
	g, ok := s.auth(w, r, bearer(r))
	if !ok {
		return
	}
	v, err := s.daily.Get(r.Context(), g.ID)
	if err != nil {
		slog.Error("daily get", "err", err)
		writeErr(w, 500, "internal", "couldn't load today's challenge")
		return
	}
	writeJSON(w, 200, v)
}

func (s *Server) dailyPlay(w http.ResponseWriter, r *http.Request) {
	if s.dailyOff(w) {
		return
	}
	g, ok := s.auth(w, r, bearer(r))
	if !ok {
		return
	}
	if !s.allowAction(g.ID, time.Now()) {
		writeErr(w, 429, game.CodeRateLimited, "slow down")
		return
	}
	var a daily.Action
	if err := readJSON(r, &a); err != nil {
		writeErr(w, 400, "bad_request", "invalid JSON")
		return
	}
	v, err := s.daily.Play(r.Context(), g.ID, a)
	var ge *game.Error
	if errors.As(err, &ge) {
		writeJSON(w, http.StatusUnprocessableEntity, ge)
		return
	}
	if err != nil {
		slog.Error("daily play", "err", err)
		writeErr(w, 500, "internal", "couldn't save your move; try again")
		return
	}
	writeJSON(w, 200, v)
}

func (s *Server) dailyMatch(w http.ResponseWriter, r *http.Request) {
	if s.dailyOff(w) {
		return
	}
	text := r.URL.Query().Get("text")
	if len([]rune(text)) > 140 {
		writeErr(w, 400, "bad_request", "text too long")
		return
	}
	if !s.allowAction("ip:"+clientIP(r), time.Now()) {
		writeErr(w, 429, game.CodeRateLimited, "slow down")
		return
	}
	writeJSON(w, 200, s.daily.Match(text))
}
