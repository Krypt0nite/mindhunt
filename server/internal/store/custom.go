package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sort"
	"sync"
	"time"
)

// CustomPack is a player-made pack as stored: the builder's draft (for
// editing) and the compiled pack (for playing).
type CustomPack struct {
	ID        string          `json:"id"`
	OwnerID   string          `json:"-"`
	Author    string          `json:"author"`
	Title     string          `json:"title"`
	Version   int             `json:"version"`
	Draft     json.RawMessage `json:"draft,omitempty"`
	Pack      json.RawMessage `json:"-"`
	Hidden    bool            `json:"hidden"`  // by moderation
	Deleted   bool            `json:"deleted"` // by the owner
	Reports   int             `json:"reports"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// Report is one player's report of a pack.
type Report struct {
	PackID     string    `json:"pack_id"`
	ReporterID string    `json:"reporter_id"`
	Reason     string    `json:"reason"`
	At         time.Time `json:"at"`
}

// CustomPacks stores player-made packs.
type CustomPacks interface {
	CreatePack(ctx context.Context, cp CustomPack) error
	// UpdatePack replaces draft, pack, title and version if the owner matches.
	UpdatePack(ctx context.Context, cp CustomPack) error
	GetPack(ctx context.Context, id string) (CustomPack, error)
	PacksByOwner(ctx context.Context, ownerID string) ([]CustomPack, error)
	CountRecentPacks(ctx context.Context, ownerID string, since time.Time) (int, error)
	// ReportPack records one report per reporter and returns the distinct count.
	ReportPack(ctx context.Context, r Report) (int, error)
	SetPackFlags(ctx context.Context, id string, hidden, deleted bool) error
	ReportedPacks(ctx context.Context, limit int) ([]CustomPack, []Report, error)
}

// ErrNotOwner is returned when someone else's pack is updated.
var ErrNotOwner = errors.New("not the owner")

// MemoryCustomPacks keeps packs in memory.
type MemoryCustomPacks struct {
	mu      sync.Mutex
	packs   map[string]CustomPack
	reports map[string]map[string]Report
}

// NewMemoryCustomPacks returns an empty store.
func NewMemoryCustomPacks() *MemoryCustomPacks {
	return &MemoryCustomPacks{packs: map[string]CustomPack{}, reports: map[string]map[string]Report{}}
}

func (m *MemoryCustomPacks) CreatePack(_ context.Context, cp CustomPack) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, dup := m.packs[cp.ID]; dup {
		return errors.New("duplicate id")
	}
	m.packs[cp.ID] = cp
	return nil
}

func (m *MemoryCustomPacks) UpdatePack(_ context.Context, cp CustomPack) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	old, ok := m.packs[cp.ID]
	if !ok {
		return ErrNotFound
	}
	if old.OwnerID != cp.OwnerID {
		return ErrNotOwner
	}
	old.Draft, old.Pack, old.Title, old.Version, old.Author, old.UpdatedAt = cp.Draft, cp.Pack, cp.Title, cp.Version, cp.Author, cp.UpdatedAt
	m.packs[cp.ID] = old
	return nil
}

func (m *MemoryCustomPacks) GetPack(_ context.Context, id string) (CustomPack, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp, ok := m.packs[id]
	if !ok {
		return CustomPack{}, ErrNotFound
	}
	cp.Reports = len(m.reports[id])
	return cp, nil
}

func (m *MemoryCustomPacks) PacksByOwner(_ context.Context, owner string) ([]CustomPack, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []CustomPack
	for _, cp := range m.packs {
		if cp.OwnerID == owner && !cp.Deleted {
			cp.Reports = len(m.reports[cp.ID])
			cp.Draft, cp.Pack = nil, nil
			out = append(out, cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt.After(out[j].UpdatedAt) })
	return out, nil
}

func (m *MemoryCustomPacks) CountRecentPacks(_ context.Context, owner string, since time.Time) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, cp := range m.packs {
		if cp.OwnerID == owner && !cp.CreatedAt.Before(since) {
			n++
		}
	}
	return n, nil
}

func (m *MemoryCustomPacks) ReportPack(_ context.Context, r Report) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.packs[r.PackID]; !ok {
		return 0, ErrNotFound
	}
	if m.reports[r.PackID] == nil {
		m.reports[r.PackID] = map[string]Report{}
	}
	m.reports[r.PackID][r.ReporterID] = r
	return len(m.reports[r.PackID]), nil
}

func (m *MemoryCustomPacks) SetPackFlags(_ context.Context, id string, hidden, deleted bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp, ok := m.packs[id]
	if !ok {
		return ErrNotFound
	}
	cp.Hidden, cp.Deleted = hidden, deleted
	m.packs[id] = cp
	return nil
}

func (m *MemoryCustomPacks) ReportedPacks(_ context.Context, limit int) ([]CustomPack, []Report, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []CustomPack
	var reps []Report
	for id, rs := range m.reports {
		cp := m.packs[id]
		cp.Reports, cp.Pack = len(rs), nil
		out = append(out, cp)
		for _, r := range rs {
			reps = append(reps, r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Reports > out[j].Reports })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, reps, nil
}

const customSchema = `
CREATE TABLE IF NOT EXISTS custom_packs (
  id         TEXT PRIMARY KEY,
  owner_id   TEXT NOT NULL,
  author     TEXT NOT NULL,
  title      TEXT NOT NULL,
  version    INT NOT NULL,
  draft      JSONB NOT NULL,
  pack       JSONB NOT NULL,
  hidden     BOOLEAN NOT NULL DEFAULT FALSE,
  deleted    BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS custom_packs_owner_idx ON custom_packs (owner_id, updated_at DESC);
CREATE TABLE IF NOT EXISTS custom_pack_reports (
  pack_id     TEXT NOT NULL REFERENCES custom_packs(id),
  reporter_id TEXT NOT NULL,
  reason      TEXT NOT NULL,
  at          TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (pack_id, reporter_id)
);
`

func (p *Postgres) CreatePack(ctx context.Context, cp CustomPack) error {
	_, err := p.db.ExecContext(ctx, `INSERT INTO custom_packs
		(id, owner_id, author, title, version, draft, pack, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		cp.ID, cp.OwnerID, cp.Author, cp.Title, cp.Version, []byte(cp.Draft), []byte(cp.Pack), cp.CreatedAt, cp.UpdatedAt)
	return err
}

func (p *Postgres) UpdatePack(ctx context.Context, cp CustomPack) error {
	res, err := p.db.ExecContext(ctx, `UPDATE custom_packs
		SET author=$3, title=$4, version=$5, draft=$6, pack=$7, updated_at=$8
		WHERE id=$1 AND owner_id=$2`,
		cp.ID, cp.OwnerID, cp.Author, cp.Title, cp.Version, []byte(cp.Draft), []byte(cp.Pack), cp.UpdatedAt)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		if _, err := p.GetPack(ctx, cp.ID); err != nil {
			return err
		}
		return ErrNotOwner
	}
	return nil
}

const packCols = `c.id, c.owner_id, c.author, c.title, c.version, c.hidden, c.deleted, c.created_at, c.updated_at,
	(SELECT count(*) FROM custom_pack_reports r WHERE r.pack_id = c.id)`

func scanPack(sc interface{ Scan(...any) error }, cp *CustomPack, extra ...any) error {
	return sc.Scan(append([]any{&cp.ID, &cp.OwnerID, &cp.Author, &cp.Title, &cp.Version, &cp.Hidden, &cp.Deleted,
		&cp.CreatedAt, &cp.UpdatedAt, &cp.Reports}, extra...)...)
}

func (p *Postgres) GetPack(ctx context.Context, id string) (CustomPack, error) {
	var cp CustomPack
	var draft, pack []byte
	err := scanPack(p.db.QueryRowContext(ctx, `SELECT `+packCols+`, c.draft, c.pack FROM custom_packs c WHERE c.id = $1`, id),
		&cp, &draft, &pack)
	if errors.Is(err, sql.ErrNoRows) {
		return CustomPack{}, ErrNotFound
	}
	cp.Draft, cp.Pack = draft, pack
	return cp, err
}

func (p *Postgres) PacksByOwner(ctx context.Context, owner string) ([]CustomPack, error) {
	rows, err := p.db.QueryContext(ctx, `SELECT `+packCols+` FROM custom_packs c
		WHERE c.owner_id = $1 AND NOT c.deleted ORDER BY c.updated_at DESC LIMIT 200`, owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CustomPack
	for rows.Next() {
		var cp CustomPack
		if err := scanPack(rows, &cp); err != nil {
			return nil, err
		}
		out = append(out, cp)
	}
	return out, rows.Err()
}

func (p *Postgres) CountRecentPacks(ctx context.Context, owner string, since time.Time) (int, error) {
	var n int
	err := p.db.QueryRowContext(ctx, `SELECT count(*) FROM custom_packs WHERE owner_id = $1 AND created_at >= $2`, owner, since).Scan(&n)
	return n, err
}

func (p *Postgres) ReportPack(ctx context.Context, r Report) (int, error) {
	_, err := p.db.ExecContext(ctx, `INSERT INTO custom_pack_reports (pack_id, reporter_id, reason, at) VALUES ($1,$2,$3,$4)
		ON CONFLICT (pack_id, reporter_id) DO UPDATE SET reason = EXCLUDED.reason, at = EXCLUDED.at`,
		r.PackID, r.ReporterID, r.Reason, r.At)
	if err != nil {
		if _, gerr := p.GetPack(ctx, r.PackID); errors.Is(gerr, ErrNotFound) {
			return 0, ErrNotFound
		}
		return 0, err
	}
	var n int
	err = p.db.QueryRowContext(ctx, `SELECT count(*) FROM custom_pack_reports WHERE pack_id = $1`, r.PackID).Scan(&n)
	return n, err
}

func (p *Postgres) SetPackFlags(ctx context.Context, id string, hidden, deleted bool) error {
	res, err := p.db.ExecContext(ctx, `UPDATE custom_packs SET hidden=$2, deleted=$3 WHERE id=$1`, id, hidden, deleted)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *Postgres) ReportedPacks(ctx context.Context, limit int) ([]CustomPack, []Report, error) {
	rows, err := p.db.QueryContext(ctx, `SELECT `+packCols+` FROM custom_packs c
		WHERE EXISTS (SELECT 1 FROM custom_pack_reports r WHERE r.pack_id = c.id)
		ORDER BY 10 DESC, c.updated_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var out []CustomPack
	var ids []string
	for rows.Next() {
		var cp CustomPack
		if err := scanPack(rows, &cp); err != nil {
			return nil, nil, err
		}
		out = append(out, cp)
		ids = append(ids, cp.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	var reps []Report
	for _, id := range ids {
		rr, err := p.db.QueryContext(ctx, `SELECT pack_id, reporter_id, reason, at FROM custom_pack_reports WHERE pack_id = $1 ORDER BY at DESC LIMIT 50`, id)
		if err != nil {
			return nil, nil, err
		}
		for rr.Next() {
			var r Report
			if err := rr.Scan(&r.PackID, &r.ReporterID, &r.Reason, &r.At); err != nil {
				rr.Close()
				return nil, nil, err
			}
			reps = append(reps, r)
		}
		rr.Close()
	}
	return out, reps, nil
}
