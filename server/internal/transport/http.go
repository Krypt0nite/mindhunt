// Package transport exposes the REST API, the WebSocket endpoint and the
// static web client.
package transport

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"mindhunt/internal/config"
	"mindhunt/internal/custom"
	"mindhunt/internal/daily"
	"mindhunt/internal/game"
	"mindhunt/internal/room"
	"mindhunt/internal/store"
)

// Server wires HTTP handlers to the room manager.
type Server struct {
	cfg    config.Config
	guests store.Guests
	mgr    *room.Manager
	reg    *game.Registry
	daily  *daily.Service
	custom *custom.Library

	limMu   sync.Mutex
	created map[string][]time.Time
	actions map[string]*bucket
}

// New builds a server. daily may be nil to turn the daily challenge off.
func New(cfg config.Config, guests store.Guests, mgr *room.Manager, reg *game.Registry, d *daily.Service, lib *custom.Library) *Server {
	return &Server{cfg: cfg, guests: guests, mgr: mgr, reg: reg, daily: d, custom: lib,
		created: map[string][]time.Time{}, actions: map[string]*bucket{}}
}

// Handler returns the full HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("GET /api/config", s.publicConfig)
	mux.HandleFunc("POST /api/guest", s.createGuest)
	mux.HandleFunc("GET /api/me", s.me)
	mux.HandleFunc("PATCH /api/me", s.rename)
	mux.HandleFunc("GET /api/packs", s.packs)
	mux.HandleFunc("POST /api/rooms", s.createRoom)
	mux.HandleFunc("GET /api/rooms/{code}", s.roomInfo)
	mux.HandleFunc("GET /ws", s.ws)
	mux.HandleFunc("GET /api/daily", s.dailyGet)
	mux.HandleFunc("POST /api/daily/play", s.dailyPlay)
	mux.HandleFunc("GET /api/daily/match", s.dailyMatch)
	s.customRoutes(mux)
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeErr(w, http.StatusNotFound, "not_found", "no such endpoint")
	})
	mux.Handle("GET /assets/", s.assets())
	mux.Handle("/", s.static())
	return securityHeaders(mux)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]string{"code": code, "message": msg})
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, 200, map[string]any{"ok": true, "rooms": s.mgr.Count()})
}

func (s *Server) publicConfig(w http.ResponseWriter, _ *http.Request) {
	timers := []int{0, 30, 60, 90, 120}
	writeJSON(w, 200, map[string]any{
		"app_name":          s.cfg.AppName,
		"wrong_guess_cost":  s.cfg.WrongGuessCost,
		"reconnect_grace_s": int(s.cfg.ReconnectGrace.Seconds()),
		"pool_sizes":        room.PoolSizes,
		"timers":            timers,
	})
}

func cleanName(n string) (string, bool) {
	n = strings.Join(strings.Fields(n), " ")
	r := []rune(n)
	if len(r) < 1 || len(r) > 24 {
		return "", false
	}
	for _, c := range r {
		if c < 32 || c == 127 {
			return "", false
		}
	}
	return n, true
}

func readJSON(r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, 4096)
	return json.NewDecoder(r.Body).Decode(v)
}

func (s *Server) createGuest(w http.ResponseWriter, r *http.Request) {
	var body struct {
		DisplayName string `json:"display_name"`
	}
	if err := readJSON(r, &body); err != nil {
		writeErr(w, 400, "bad_request", "invalid JSON")
		return
	}
	name, ok := cleanName(body.DisplayName)
	if !ok {
		writeErr(w, 400, "bad_name", "name must be 1 to 24 characters")
		return
	}
	g, err := s.guests.Create(r.Context(), name)
	if err != nil {
		slog.Error("create guest", "err", err)
		writeErr(w, 500, "internal", "could not create guest")
		return
	}
	writeJSON(w, 201, g)
}

func bearer(r *http.Request) string {
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return ""
}

func (s *Server) auth(w http.ResponseWriter, r *http.Request, token string) (store.Guest, bool) {
	if token == "" {
		writeErr(w, 401, "unauthorized", "missing guest token")
		return store.Guest{}, false
	}
	g, err := s.guests.ByToken(r.Context(), token)
	if errors.Is(err, store.ErrNotFound) {
		writeErr(w, 401, "unauthorized", "unknown guest token")
		return g, false
	}
	if err != nil {
		slog.Error("auth", "err", err)
		writeErr(w, 500, "internal", "auth failed")
		return g, false
	}
	return g, true
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	g, ok := s.auth(w, r, bearer(r))
	if !ok {
		return
	}
	g.Token = ""
	writeJSON(w, 200, g)
}

func (s *Server) rename(w http.ResponseWriter, r *http.Request) {
	g, ok := s.auth(w, r, bearer(r))
	if !ok {
		return
	}
	var body struct {
		DisplayName string `json:"display_name"`
	}
	if err := readJSON(r, &body); err != nil {
		writeErr(w, 400, "bad_request", "invalid JSON")
		return
	}
	name, valid := cleanName(body.DisplayName)
	if !valid {
		writeErr(w, 400, "bad_name", "name must be 1 to 24 characters")
		return
	}
	if err := s.guests.Rename(r.Context(), g.ID, name); err != nil {
		writeErr(w, 500, "internal", "rename failed")
		return
	}
	g.Name, g.Token = name, ""
	writeJSON(w, 200, g)
}

func (s *Server) packs(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, 200, s.reg.Summaries())
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// allowRoom applies the per-IP room creation limit.
func (s *Server) allowRoom(ip string, now time.Time) bool {
	if s.cfg.RoomsPerIPHour <= 0 {
		return true
	}
	s.limMu.Lock()
	defer s.limMu.Unlock()
	recent := s.created[ip][:0]
	for _, t := range s.created[ip] {
		if now.Sub(t) < time.Hour {
			recent = append(recent, t)
		}
	}
	if len(recent) >= s.cfg.RoomsPerIPHour {
		s.created[ip] = recent
		return false
	}
	s.created[ip] = append(recent, now)
	return true
}

func (s *Server) createRoom(w http.ResponseWriter, r *http.Request) {
	g, ok := s.auth(w, r, bearer(r))
	if !ok {
		return
	}
	var body struct {
		Practice bool   `json:"practice"`
		PackID   string `json:"pack_id"`
	}
	if r.ContentLength != 0 {
		if err := readJSON(r, &body); err != nil {
			writeErr(w, 400, "bad_request", "invalid JSON")
			return
		}
	}
	if !s.allowRoom(clientIP(r), time.Now()) {
		writeErr(w, 429, "rate_limited", "too many rooms created; try again later")
		return
	}
	var pack *game.Pack
	if body.PackID != "" {
		var ok bool
		if pack, ok = s.reg.Get(body.PackID); !ok {
			p, _, err := s.custom.Load(r.Context(), body.PackID, g.ID)
			if err != nil {
				writeErr(w, 404, "not_found", "pack not found")
				return
			}
			pack = p
		}
	}
	rm, err := s.mgr.Create(g.ID, body.Practice, pack)
	if err != nil {
		writeErr(w, 503, "unavailable", err.Error())
		return
	}
	writeJSON(w, 201, map[string]any{
		"code": rm.Code, "practice": body.Practice,
		"invite_url": s.baseURL(r) + "/room/?code=" + rm.Code,
	})
}

func (s *Server) baseURL(r *http.Request) string {
	if s.cfg.PublicURL != "" {
		return s.cfg.PublicURL
	}
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

func (s *Server) roomInfo(w http.ResponseWriter, r *http.Request) {
	rm, ok := s.mgr.Get(strings.ToUpper(r.PathValue("code")))
	if !ok {
		writeErr(w, 404, "not_found", "room not found")
		return
	}
	info, ok := rm.Info()
	if !ok {
		writeErr(w, 404, "not_found", "room closed")
		return
	}
	writeJSON(w, 200, info)
}

// assets serves licensed images from ASSETS_DIR. Only image files are
// served, never directory listings; names are candidate ids, so a URL only
// reaches a client in the results view, after the reveal.
func (s *Server) assets() http.Handler {
	dir := s.cfg.AssetsDir
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rel := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/assets/")
		if !AssetPathOK(rel) {
			http.NotFound(w, r)
			return
		}
		full := filepath.Join(dir, filepath.FromSlash(rel))
		st, err := os.Stat(full)
		if err != nil || st.IsDir() {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=604800")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		http.ServeFile(w, r, full)
	})
}

var assetPathRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*/[A-Za-z0-9_-]+\.(webp|jpg|jpeg|png)$`)

// AssetPathOK reports whether rel is a servable asset path (one directory,
// an image extension, no traversal).
func AssetPathOK(rel string) bool { return assetPathRe.MatchString(rel) }

// static serves the exported Next.js client. /room/ resolves to
// room/index.html; unknown paths get 404.html.
func (s *Server) static() http.Handler {
	dir := s.cfg.WebDir
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "web client not built (WEB_DIR="+dir+")", http.StatusNotFound)
		})
	}
	fs := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := path.Clean("/" + r.URL.Path)
		full := filepath.Join(dir, filepath.FromSlash(p))
		if st, err := os.Stat(full); err == nil {
			if st.IsDir() {
				if _, err := os.Stat(filepath.Join(full, "index.html")); err != nil {
					s.notFound(w, dir)
					return
				}
			}
			if strings.HasPrefix(p, "/_next/static/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			} else {
				w.Header().Set("Cache-Control", "no-cache")
			}
			fs.ServeHTTP(w, r)
			return
		}
		if _, err := os.Stat(full + ".html"); err == nil {
			r.URL.Path = p + ".html"
			fs.ServeHTTP(w, r)
			return
		}
		s.notFound(w, dir)
	})
}

func (s *Server) notFound(w http.ResponseWriter, dir string) {
	b, err := os.ReadFile(filepath.Join(dir, "404.html"))
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write(b)
}
