package transport

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"mindhunt/internal/custom"
	"mindhunt/internal/game"
	"mindhunt/internal/store"
)

const maxDraftBytes = 256 << 10

func (s *Server) customRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/custom/check", s.customCheck)
	mux.HandleFunc("POST /api/custom", s.customCreate)
	mux.HandleFunc("GET /api/custom", s.customMine)
	mux.HandleFunc("GET /api/custom/{id}", s.customGet)
	mux.HandleFunc("PUT /api/custom/{id}", s.customUpdate)
	mux.HandleFunc("DELETE /api/custom/{id}", s.customDelete)
	mux.HandleFunc("POST /api/custom/{id}/report", s.customReport)
	mux.HandleFunc("GET /api/admin/reports", s.adminReports)
	mux.HandleFunc("POST /api/admin/custom/{id}", s.adminModerate)
}

func readDraft(w http.ResponseWriter, r *http.Request) (custom.Draft, bool) {
	var body struct {
		Draft custom.Draft `json:"draft"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxDraftBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		writeErr(w, 400, "bad_request", "invalid pack JSON (or larger than 256 KB)")
		return custom.Draft{}, false
	}
	return body.Draft, true
}

func (s *Server) customErr(w http.ResponseWriter, err error, saved custom.Saved) {
	switch {
	case errors.Is(err, custom.ErrDraftError):
		writeJSON(w, 422, map[string]any{"code": "invalid_pack", "message": "fix the errors and save again",
			"issues": saved.Issues, "stats": saved.Stats})
	case errors.Is(err, custom.ErrNotFound):
		writeErr(w, 404, "not_found", "pack not found (it may have been deleted or hidden after reports)")
	case errors.Is(err, custom.ErrNotOwner):
		writeErr(w, 403, "forbidden", err.Error())
	case errors.Is(err, custom.ErrTooMany):
		writeErr(w, 409, "too_many", "you can keep up to 30 packs; delete one first")
	case errors.Is(err, custom.ErrRateLimit):
		writeErr(w, 429, "rate_limited", err.Error())
	default:
		writeErr(w, 500, "internal", "could not save the pack")
	}
}

func (s *Server) customCheck(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.auth(w, r, bearer(r)); !ok {
		return
	}
	d, ok := readDraft(w, r)
	if !ok {
		return
	}
	is, st := custom.Check(d)
	if is == nil {
		is = custom.Issues{}
	}
	writeJSON(w, 200, map[string]any{"ok": !is.HasErrors(), "issues": is, "stats": st})
}

func (s *Server) customCreate(w http.ResponseWriter, r *http.Request) {
	g, ok := s.auth(w, r, bearer(r))
	if !ok {
		return
	}
	d, ok := readDraft(w, r)
	if !ok {
		return
	}
	saved, err := s.custom.Create(r.Context(), g, d)
	if err != nil {
		s.customErr(w, err, saved)
		return
	}
	writeJSON(w, 201, s.savedView(r, saved))
}

func (s *Server) customUpdate(w http.ResponseWriter, r *http.Request) {
	g, ok := s.auth(w, r, bearer(r))
	if !ok {
		return
	}
	d, ok := readDraft(w, r)
	if !ok {
		return
	}
	saved, err := s.custom.Update(r.Context(), g, r.PathValue("id"), d)
	if err != nil {
		s.customErr(w, err, saved)
		return
	}
	writeJSON(w, 200, s.savedView(r, saved))
}

func (s *Server) savedView(r *http.Request, sv custom.Saved) map[string]any {
	if sv.Issues == nil {
		sv.Issues = custom.Issues{}
	}
	return map[string]any{"id": sv.Pack.ID, "version": sv.Pack.Version, "issues": sv.Issues, "stats": sv.Stats,
		"share_url": s.baseURL(r) + "/pack/?id=" + sv.Pack.ID}
}

// CustomView is what anyone with the link sees.
type CustomView struct {
	game.PackSummary
	Author     string          `json:"author"`
	Attributes []string        `json:"attributes"`
	Items      []string        `json:"items"`
	IsOwner    bool            `json:"is_owner"`
	Hidden     bool            `json:"hidden"`
	Reports    int             `json:"reports,omitempty"`
	EditDraft  json.RawMessage `json:"edit_draft,omitempty"` // owner only, for editing
	ShareURL   string          `json:"share_url"`
}

func (s *Server) customGet(w http.ResponseWriter, r *http.Request) {
	viewer := ""
	if t := bearer(r); t != "" {
		if g, err := s.guests.ByToken(r.Context(), t); err == nil {
			viewer = g.ID
		}
	}
	p, cp, err := s.custom.Load(r.Context(), r.PathValue("id"), viewer)
	if err != nil {
		s.customErr(w, err, custom.Saved{})
		return
	}
	v := CustomView{
		PackSummary: game.PackSummary{ID: p.ID, Version: p.Version, AsOf: p.AsOf, Category: p.Category, Title: p.Title,
			Noun: p.Noun, Description: p.Description, Candidates: len(p.Candidates)},
		Author: p.Author, IsOwner: viewer != "" && viewer == cp.OwnerID, Hidden: cp.Hidden,
		ShareURL: s.baseURL(r) + "/pack/?id=" + p.ID,
	}
	for _, a := range p.Attributes {
		if a.Key != "letters" {
			v.Attributes = append(v.Attributes, a.Label)
		}
	}
	for _, c := range p.Candidates {
		v.Items = append(v.Items, c.Name)
	}
	sort.Strings(v.Items)
	if v.IsOwner {
		v.EditDraft, v.Reports = cp.Draft, cp.Reports
	}
	writeJSON(w, 200, v)
}

func (s *Server) customMine(w http.ResponseWriter, r *http.Request) {
	g, ok := s.auth(w, r, bearer(r))
	if !ok {
		return
	}
	list, err := s.custom.Mine(r.Context(), g.ID)
	if err != nil {
		writeErr(w, 500, "internal", "could not list packs")
		return
	}
	if list == nil {
		list = []store.CustomPack{}
	}
	writeJSON(w, 200, list)
}

func (s *Server) customDelete(w http.ResponseWriter, r *http.Request) {
	g, ok := s.auth(w, r, bearer(r))
	if !ok {
		return
	}
	if err := s.custom.Delete(r.Context(), g.ID, r.PathValue("id")); err != nil {
		s.customErr(w, err, custom.Saved{})
		return
	}
	w.WriteHeader(204)
}

func (s *Server) customReport(w http.ResponseWriter, r *http.Request) {
	g, ok := s.auth(w, r, bearer(r))
	if !ok {
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	if err := readJSON(r, &body); err != nil || strings.TrimSpace(body.Reason) == "" {
		writeErr(w, 400, "bad_request", "say briefly what is wrong with the pack")
		return
	}
	if !s.allowAction(g.ID, time.Now()) {
		writeErr(w, 429, "rate_limited", "slow down")
		return
	}
	hidden, err := s.custom.Report(r.Context(), g.ID, r.PathValue("id"), body.Reason)
	if errors.Is(err, custom.ErrNotOwner) {
		writeErr(w, 400, "bad_request", "you can't report your own pack; edit or delete it instead")
		return
	}
	if err != nil {
		s.customErr(w, err, custom.Saved{})
		return
	}
	writeJSON(w, 200, map[string]any{"reported": true, "hidden": hidden})
}

// Admin endpoints need X-Admin-Token equal to ADMIN_TOKEN; they are off when
// ADMIN_TOKEN is unset.
func (s *Server) admin(w http.ResponseWriter, r *http.Request) bool {
	want := s.cfg.AdminToken
	got := r.Header.Get("X-Admin-Token")
	if want == "" || subtle.ConstantTimeCompare([]byte(want), []byte(got)) != 1 {
		writeErr(w, 404, "not_found", "not found")
		return false
	}
	return true
}

func (s *Server) adminReports(w http.ResponseWriter, r *http.Request) {
	if !s.admin(w, r) {
		return
	}
	packs, reports, err := s.custom.Reported(r.Context())
	if err != nil {
		writeErr(w, 500, "internal", err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"packs": packs, "reports": reports})
}

func (s *Server) adminModerate(w http.ResponseWriter, r *http.Request) {
	if !s.admin(w, r) {
		return
	}
	var body struct {
		Hidden bool `json:"hidden"`
	}
	if err := readJSON(r, &body); err != nil {
		writeErr(w, 400, "bad_request", "invalid JSON")
		return
	}
	if err := s.custom.Moderate(r.Context(), r.PathValue("id"), body.Hidden); err != nil {
		s.customErr(w, err, custom.Saved{})
		return
	}
	writeJSON(w, 200, map[string]any{"id": r.PathValue("id"), "hidden": body.Hidden})
}
