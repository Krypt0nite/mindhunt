package custom

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"mindhunt/internal/game"
	"mindhunt/internal/store"
)

// Errors returned to the HTTP layer.
var (
	ErrNotFound   = errors.New("pack not found")
	ErrNotOwner   = errors.New("only the pack's creator can change it")
	ErrTooMany    = errors.New("pack limit reached")
	ErrRateLimit  = errors.New("too many packs saved recently; try again later")
	ErrDraftError = errors.New("the pack has errors")
)

// Config sets the library's limits.
type Config struct {
	HideAfterReports int // distinct reports that hide a pack until reviewed
	MaxPerOwner      int
	PerHour          int // creations and edits per owner per hour
	CacheSize        int
}

// DefaultConfig is used when a field is zero.
var DefaultConfig = Config{HideAfterReports: 3, MaxPerOwner: 30, PerHour: 20, CacheSize: 500}

// Library saves player-made packs and keeps recently used ones compiled in
// memory, so rooms can resolve them without touching the database.
type Library struct {
	st  store.CustomPacks
	cfg Config
	now func() time.Time

	mu    sync.Mutex
	cache map[string]*cached
	edits map[string][]time.Time
}

type cached struct {
	pack *game.Pack
	used time.Time
}

// NewLibrary wraps a store.
func NewLibrary(st store.CustomPacks, cfg Config) *Library {
	d := DefaultConfig
	if cfg.HideAfterReports == 0 {
		cfg.HideAfterReports = d.HideAfterReports
	}
	if cfg.MaxPerOwner == 0 {
		cfg.MaxPerOwner = d.MaxPerOwner
	}
	if cfg.PerHour == 0 {
		cfg.PerHour = d.PerHour
	}
	if cfg.CacheSize == 0 {
		cfg.CacheSize = d.CacheSize
	}
	return &Library{st: st, cfg: cfg, now: time.Now, cache: map[string]*cached{}, edits: map[string][]time.Time{}}
}

// Saved is the result of a create or update.
type Saved struct {
	Pack   store.CustomPack `json:"pack"`
	Issues Issues           `json:"issues"`
	Stats  Stats            `json:"stats"`
}

// Create validates and stores a new pack.
func (l *Library) Create(ctx context.Context, owner store.Guest, d Draft) (Saved, error) {
	if err := l.allowEdit(owner.ID); err != nil {
		return Saved{}, err
	}
	mine, err := l.st.PacksByOwner(ctx, owner.ID)
	if err != nil {
		return Saved{}, err
	}
	if len(mine) >= l.cfg.MaxPerOwner {
		return Saved{}, ErrTooMany
	}
	now := l.now().UTC()
	id := NewID()
	pack, is, st := Build(d, Meta{ID: id, Version: 1, Author: owner.Name, Now: now})
	if pack == nil {
		return Saved{Issues: is, Stats: st}, ErrDraftError
	}
	cp, err := record(id, owner, 1, d, pack, now)
	if err != nil {
		return Saved{}, err
	}
	cp.CreatedAt = now
	if err := l.st.CreatePack(ctx, cp); err != nil {
		return Saved{}, err
	}
	l.put(pack)
	return Saved{Pack: cp, Issues: is, Stats: st}, nil
}

// Update saves a new version; item ids are kept for unchanged names.
// Running matches keep the version they started with.
func (l *Library) Update(ctx context.Context, owner store.Guest, id string, d Draft) (Saved, error) {
	old, err := l.st.GetPack(ctx, id)
	if errors.Is(err, store.ErrNotFound) || (err == nil && old.Deleted) {
		return Saved{}, ErrNotFound
	}
	if err != nil {
		return Saved{}, err
	}
	if old.OwnerID != owner.ID {
		return Saved{}, ErrNotOwner
	}
	if err := l.allowEdit(owner.ID); err != nil {
		return Saved{}, err
	}
	prev := map[string]string{}
	var oldPack game.Pack
	if json.Unmarshal(old.Pack, &oldPack) == nil {
		for _, c := range oldPack.Candidates {
			prev[strings.ToLower(c.Name)] = c.ID
		}
	}
	now := l.now().UTC()
	pack, is, st := Build(d, Meta{ID: id, Version: old.Version + 1, Author: owner.Name, PrevIDs: prev, Now: now})
	if pack == nil {
		return Saved{Issues: is, Stats: st}, ErrDraftError
	}
	cp, err := record(id, owner, old.Version+1, d, pack, now)
	if err != nil {
		return Saved{}, err
	}
	cp.CreatedAt, cp.Hidden = old.CreatedAt, old.Hidden
	if err := l.st.UpdatePack(ctx, cp); err != nil {
		return Saved{}, err
	}
	if !old.Hidden {
		l.put(pack)
	}
	return Saved{Pack: cp, Issues: is, Stats: st}, nil
}

func record(id string, owner store.Guest, version int, d Draft, pack *game.Pack, now time.Time) (store.CustomPack, error) {
	draft, err := json.Marshal(d)
	if err != nil {
		return store.CustomPack{}, err
	}
	compiled, err := json.Marshal(pack)
	if err != nil {
		return store.CustomPack{}, err
	}
	return store.CustomPack{ID: id, OwnerID: owner.ID, Author: owner.Name, Title: pack.Title, Version: version,
		Draft: draft, Pack: compiled, UpdatedAt: now}, nil
}

// Load fetches a pack for a viewer and caches it for rooms. Hidden or
// deleted packs are visible only to their owner (hidden) or nobody (deleted).
func (l *Library) Load(ctx context.Context, id, viewerID string) (*game.Pack, store.CustomPack, error) {
	if !IsCustomID(id) {
		return nil, store.CustomPack{}, ErrNotFound
	}
	cp, err := l.st.GetPack(ctx, id)
	if errors.Is(err, store.ErrNotFound) {
		return nil, store.CustomPack{}, ErrNotFound
	}
	if err != nil {
		return nil, store.CustomPack{}, err
	}
	if cp.Deleted || (cp.Hidden && cp.OwnerID != viewerID) {
		l.drop(id)
		return nil, store.CustomPack{}, ErrNotFound
	}
	pack, err := game.ParsePack(cp.Pack)
	if err != nil {
		return nil, store.CustomPack{}, err
	}
	if !cp.Hidden {
		l.put(pack)
	}
	return pack, cp, nil
}

// Cached returns a pack already loaded, for rooms (no I/O).
func (l *Library) Cached(id string) (*game.Pack, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	c, ok := l.cache[id]
	if !ok {
		return nil, false
	}
	c.used = l.now()
	return c.pack, true
}

// Mine lists the owner's packs.
func (l *Library) Mine(ctx context.Context, ownerID string) ([]store.CustomPack, error) {
	return l.st.PacksByOwner(ctx, ownerID)
}

// Delete removes a pack for its owner; rooms already using it keep it.
func (l *Library) Delete(ctx context.Context, ownerID, id string) error {
	cp, err := l.st.GetPack(ctx, id)
	if errors.Is(err, store.ErrNotFound) || (err == nil && cp.Deleted) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if cp.OwnerID != ownerID {
		return ErrNotOwner
	}
	l.drop(id)
	return l.st.SetPackFlags(ctx, id, cp.Hidden, true)
}

// Report records a report; enough distinct reports hide the pack until an
// admin restores it. Owners can't report their own packs.
func (l *Library) Report(ctx context.Context, reporterID, id, reason string) (hidden bool, err error) {
	cp, err := l.st.GetPack(ctx, id)
	if errors.Is(err, store.ErrNotFound) || (err == nil && cp.Deleted) {
		return false, ErrNotFound
	}
	if err != nil {
		return false, err
	}
	if cp.OwnerID == reporterID {
		return false, ErrNotOwner
	}
	reason = strings.TrimSpace(reason)
	if len(reason) > 300 {
		reason = reason[:300]
	}
	n, err := l.st.ReportPack(ctx, store.Report{PackID: id, ReporterID: reporterID, Reason: reason, At: l.now().UTC()})
	if err != nil {
		return false, err
	}
	if n >= l.cfg.HideAfterReports && !cp.Hidden {
		if err := l.st.SetPackFlags(ctx, id, true, false); err != nil {
			return false, err
		}
		l.drop(id)
		return true, nil
	}
	return cp.Hidden, nil
}

// Moderate hides or restores a pack (admin).
func (l *Library) Moderate(ctx context.Context, id string, hidden bool) error {
	cp, err := l.st.GetPack(ctx, id)
	if errors.Is(err, store.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if hidden {
		l.drop(id)
	}
	return l.st.SetPackFlags(ctx, id, hidden, cp.Deleted)
}

// Reported lists reported packs for review (admin).
func (l *Library) Reported(ctx context.Context) ([]store.CustomPack, []store.Report, error) {
	return l.st.ReportedPacks(ctx, 100)
}

func (l *Library) allowEdit(owner string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	keep := l.edits[owner][:0]
	for _, t := range l.edits[owner] {
		if now.Sub(t) < time.Hour {
			keep = append(keep, t)
		}
	}
	if len(keep) >= l.cfg.PerHour {
		l.edits[owner] = keep
		return ErrRateLimit
	}
	l.edits[owner] = append(keep, now)
	return nil
}

func (l *Library) put(p *game.Pack) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cache[p.ID] = &cached{pack: p, used: l.now()}
	for len(l.cache) > l.cfg.CacheSize {
		var oldest string
		var at time.Time
		for id, c := range l.cache {
			if oldest == "" || c.used.Before(at) {
				oldest, at = id, c.used
			}
		}
		delete(l.cache, oldest)
	}
}

func (l *Library) drop(id string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.cache, id)
}
