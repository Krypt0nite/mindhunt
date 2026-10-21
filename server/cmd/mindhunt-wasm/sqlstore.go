//go:build js && wasm

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"syscall/js"
	"time"

	"mindhunt/internal/store"
)

// sqlDB runs statements on the Durable Object's SQLite storage, which is
// synchronous, so no call here waits on the JavaScript event loop.
type sqlDB struct{ fn js.Value }

// tsFormat is fixed width (always UTC, nine fraction digits), so stored
// timestamps compare correctly as text.
const tsFormat = "2006-01-02T15:04:05.000000000Z07:00"

type sqlResult struct {
	Rows    [][]json.RawMessage `json:"rows"`
	Written int                 `json:"written"`
	Error   string              `json:"error"`
}

func (d *sqlDB) exec(q string, args ...any) (sqlResult, error) {
	for i, a := range args {
		switch v := a.(type) {
		case bool: // SQLite has no booleans
			if v {
				args[i] = 1
			} else {
				args[i] = 0
			}
		case time.Time:
			args[i] = v.UTC().Format(tsFormat)
		case *time.Time:
			if v == nil {
				args[i] = nil
			} else {
				args[i] = v.UTC().Format(tsFormat)
			}
		case json.RawMessage:
			args[i] = string(v)
		}
	}
	if args == nil {
		args = []any{}
	}
	ab, err := json.Marshal(args)
	if err != nil {
		return sqlResult{}, err
	}
	var res sqlResult
	if err := json.Unmarshal([]byte(d.fn.Invoke(q, string(ab)).String()), &res); err != nil {
		return sqlResult{}, err
	}
	if res.Error != "" {
		return res, fmt.Errorf("sql: %s", res.Error)
	}
	return res, nil
}

// scan decodes one row into dst: *string, *int, *bool (0/1), *time.Time,
// **time.Time (NULL allowed) or *json.RawMessage (TEXT holding JSON).
func scan(row []json.RawMessage, dst ...any) error {
	if len(row) < len(dst) {
		return errors.New("sql: short row")
	}
	for i, d := range dst {
		raw := row[i]
		switch p := d.(type) {
		case *bool:
			var n int
			if err := json.Unmarshal(raw, &n); err != nil {
				return err
			}
			*p = n != 0
		case *time.Time:
			t, err := parseTime(raw)
			if err != nil {
				return err
			}
			*p = t
		case **time.Time:
			if string(raw) == "null" {
				*p = nil
				continue
			}
			t, err := parseTime(raw)
			if err != nil {
				return err
			}
			*p = &t
		case *json.RawMessage:
			var s string
			if err := json.Unmarshal(raw, &s); err != nil {
				return err
			}
			*p = json.RawMessage(s)
		default:
			if err := json.Unmarshal(raw, d); err != nil {
				return err
			}
		}
	}
	return nil
}

func parseTime(raw json.RawMessage) (time.Time, error) {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339Nano, s)
}

const sqliteSchema = `
CREATE TABLE IF NOT EXISTS players (
  id           TEXT PRIMARY KEY,
  token_hash   TEXT NOT NULL UNIQUE,
  display_name TEXT NOT NULL,
  created_at   TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS daily_plays (
  day         TEXT NOT NULL,
  player_id   TEXT NOT NULL,
  moves       TEXT NOT NULL,
  status      TEXT NOT NULL,
  score       INTEGER NOT NULL,
  started_at  TEXT NOT NULL,
  finished_at TEXT,
  PRIMARY KEY (day, player_id)
);
CREATE INDEX IF NOT EXISTS daily_plays_player_idx ON daily_plays (player_id, day DESC);
CREATE TABLE IF NOT EXISTS custom_packs (
  id         TEXT PRIMARY KEY,
  owner_id   TEXT NOT NULL,
  author     TEXT NOT NULL,
  title      TEXT NOT NULL,
  version    INTEGER NOT NULL,
  draft      TEXT NOT NULL,
  pack       TEXT NOT NULL,
  hidden     INTEGER NOT NULL DEFAULT 0,
  deleted    INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS custom_packs_owner_idx ON custom_packs (owner_id, updated_at DESC);
CREATE TABLE IF NOT EXISTS custom_pack_reports (
  pack_id     TEXT NOT NULL REFERENCES custom_packs(id),
  reporter_id TEXT NOT NULL,
  reason      TEXT NOT NULL,
  at          TEXT NOT NULL,
  PRIMARY KEY (pack_id, reporter_id)
);
`

// sqlStore implements store.Guests, store.Daily and store.CustomPacks.
// Match events are not recorded (store.Noop) to stay well inside the free
// plan's daily row-write allowance.
type sqlStore struct {
	db *sqlDB

	mu    sync.Mutex
	cache map[string]store.Guest // token -> guest
}

func newSQLStore(db *sqlDB) (*sqlStore, error) {
	for _, stmt := range strings.Split(sqliteSchema, ";") {
		if strings.TrimSpace(stmt) == "" {
			continue
		}
		if _, err := db.exec(stmt); err != nil {
			return nil, err
		}
	}
	return &sqlStore{db: db, cache: map[string]store.Guest{}}, nil
}

func hashToken(t string) string {
	h := sha256.Sum256([]byte(t))
	return hex.EncodeToString(h[:])
}

// Guests

func (s *sqlStore) Create(_ context.Context, name string) (store.Guest, error) {
	g := store.Guest{ID: "p_" + store.RandomHex(8), Token: store.RandomHex(24), Name: name, CreatedAt: time.Now().UTC()}
	if _, err := s.db.exec(`INSERT INTO players (id, token_hash, display_name, created_at) VALUES (?, ?, ?, ?)`,
		g.ID, hashToken(g.Token), g.Name, g.CreatedAt); err != nil {
		return store.Guest{}, err
	}
	s.mu.Lock()
	s.cache[g.Token] = g
	s.mu.Unlock()
	return g, nil
}

func (s *sqlStore) ByToken(_ context.Context, token string) (store.Guest, error) {
	if token == "" {
		return store.Guest{}, store.ErrNotFound
	}
	s.mu.Lock()
	g, ok := s.cache[token]
	s.mu.Unlock()
	if ok {
		return g, nil
	}
	res, err := s.db.exec(`SELECT id, display_name, created_at FROM players WHERE token_hash = ?`, hashToken(token))
	if err != nil {
		return store.Guest{}, err
	}
	if len(res.Rows) == 0 {
		return store.Guest{}, store.ErrNotFound
	}
	g = store.Guest{Token: token}
	if err := scan(res.Rows[0], &g.ID, &g.Name, &g.CreatedAt); err != nil {
		return store.Guest{}, err
	}
	s.mu.Lock()
	s.cache[token] = g
	s.mu.Unlock()
	return g, nil
}

func (s *sqlStore) Rename(_ context.Context, id, name string) error {
	if _, err := s.db.exec(`UPDATE players SET display_name = ? WHERE id = ?`, name, id); err != nil {
		return err
	}
	s.mu.Lock()
	for tok, g := range s.cache {
		if g.ID == id {
			g.Name = name
			s.cache[tok] = g
		}
	}
	s.mu.Unlock()
	return nil
}

// Daily

func (s *sqlStore) LoadPlay(_ context.Context, date, playerID string) (store.DailyPlay, bool, error) {
	res, err := s.db.exec(`SELECT day, player_id, moves, status, score, started_at, finished_at
		FROM daily_plays WHERE day = ? AND player_id = ?`, date, playerID)
	if err != nil || len(res.Rows) == 0 {
		return store.DailyPlay{}, false, err
	}
	var dp store.DailyPlay
	var moves json.RawMessage
	if err := scan(res.Rows[0], &dp.Date, &dp.PlayerID, &moves, &dp.Status, &dp.Score, &dp.StartedAt, &dp.FinishedAt); err != nil {
		return store.DailyPlay{}, false, err
	}
	if err := json.Unmarshal(moves, &dp.Moves); err != nil {
		return store.DailyPlay{}, false, err
	}
	return dp, true, nil
}

func (s *sqlStore) SavePlay(_ context.Context, dp store.DailyPlay) error {
	moves, err := json.Marshal(dp.Moves)
	if err != nil {
		return err
	}
	_, err = s.db.exec(`
		INSERT INTO daily_plays (day, player_id, moves, status, score, started_at, finished_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (day, player_id) DO UPDATE
		SET moves = excluded.moves, status = excluded.status, score = excluded.score, finished_at = excluded.finished_at`,
		dp.Date, dp.PlayerID, string(moves), dp.Status, dp.Score, dp.StartedAt, dp.FinishedAt)
	return err
}

func (s *sqlStore) DayStats(_ context.Context, date string) (store.DailyStats, error) {
	st := store.DailyStats{Scores: map[int]int{}}
	res, err := s.db.exec(`SELECT status, score, count(*) FROM daily_plays
		WHERE day = ? AND status <> 'hunting' GROUP BY status, score`, date)
	if err != nil {
		return st, err
	}
	for _, row := range res.Rows {
		var status string
		var score, n int
		if err := scan(row, &status, &score, &n); err != nil {
			return st, err
		}
		st.Finished += n
		if status == "solved" {
			st.Solved += n
			st.Scores[score] += n
		}
	}
	return st, nil
}

func (s *sqlStore) PlayHistory(_ context.Context, playerID string, limit int) ([]store.DailyPlay, error) {
	res, err := s.db.exec(`SELECT day, status, score FROM daily_plays WHERE player_id = ? ORDER BY day DESC LIMIT ?`,
		playerID, limit)
	if err != nil {
		return nil, err
	}
	var out []store.DailyPlay
	for _, row := range res.Rows {
		dp := store.DailyPlay{PlayerID: playerID}
		if err := scan(row, &dp.Date, &dp.Status, &dp.Score); err != nil {
			return nil, err
		}
		out = append(out, dp)
	}
	return out, nil
}

// Custom packs

func (s *sqlStore) CreatePack(_ context.Context, cp store.CustomPack) error {
	_, err := s.db.exec(`INSERT INTO custom_packs
		(id, owner_id, author, title, version, draft, pack, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		cp.ID, cp.OwnerID, cp.Author, cp.Title, cp.Version, cp.Draft, cp.Pack, cp.CreatedAt, cp.UpdatedAt)
	return err
}

func (s *sqlStore) UpdatePack(ctx context.Context, cp store.CustomPack) error {
	res, err := s.db.exec(`UPDATE custom_packs
		SET author = ?, title = ?, version = ?, draft = ?, pack = ?, updated_at = ?
		WHERE id = ? AND owner_id = ?`,
		cp.Author, cp.Title, cp.Version, cp.Draft, cp.Pack, cp.UpdatedAt, cp.ID, cp.OwnerID)
	if err != nil {
		return err
	}
	if res.Written == 0 {
		if _, err := s.GetPack(ctx, cp.ID); err != nil {
			return err
		}
		return store.ErrNotOwner
	}
	return nil
}

const packCols = `c.id, c.owner_id, c.author, c.title, c.version, c.hidden, c.deleted, c.created_at, c.updated_at,
	(SELECT count(*) FROM custom_pack_reports r WHERE r.pack_id = c.id)`

func scanPack(row []json.RawMessage, cp *store.CustomPack, extra ...any) error {
	return scan(row, append([]any{&cp.ID, &cp.OwnerID, &cp.Author, &cp.Title, &cp.Version, &cp.Hidden, &cp.Deleted,
		&cp.CreatedAt, &cp.UpdatedAt, &cp.Reports}, extra...)...)
}

func (s *sqlStore) GetPack(_ context.Context, id string) (store.CustomPack, error) {
	res, err := s.db.exec(`SELECT `+packCols+`, c.draft, c.pack FROM custom_packs c WHERE c.id = ?`, id)
	if err != nil {
		return store.CustomPack{}, err
	}
	if len(res.Rows) == 0 {
		return store.CustomPack{}, store.ErrNotFound
	}
	var cp store.CustomPack
	err = scanPack(res.Rows[0], &cp, &cp.Draft, &cp.Pack)
	return cp, err
}

func (s *sqlStore) packList(q string, args ...any) ([]store.CustomPack, error) {
	res, err := s.db.exec(q, args...)
	if err != nil {
		return nil, err
	}
	var out []store.CustomPack
	for _, row := range res.Rows {
		var cp store.CustomPack
		if err := scanPack(row, &cp); err != nil {
			return nil, err
		}
		out = append(out, cp)
	}
	return out, nil
}

func (s *sqlStore) PacksByOwner(_ context.Context, owner string) ([]store.CustomPack, error) {
	return s.packList(`SELECT `+packCols+` FROM custom_packs c
		WHERE c.owner_id = ? AND NOT c.deleted ORDER BY c.updated_at DESC LIMIT 200`, owner)
}

func (s *sqlStore) CountRecentPacks(_ context.Context, owner string, since time.Time) (int, error) {
	res, err := s.db.exec(`SELECT count(*) FROM custom_packs WHERE owner_id = ? AND created_at >= ?`, owner, since)
	if err != nil || len(res.Rows) == 0 {
		return 0, err
	}
	var n int
	err = scan(res.Rows[0], &n)
	return n, err
}

func (s *sqlStore) ReportPack(ctx context.Context, r store.Report) (int, error) {
	if _, err := s.GetPack(ctx, r.PackID); err != nil {
		return 0, err
	}
	if _, err := s.db.exec(`INSERT INTO custom_pack_reports (pack_id, reporter_id, reason, at) VALUES (?, ?, ?, ?)
		ON CONFLICT (pack_id, reporter_id) DO UPDATE SET reason = excluded.reason, at = excluded.at`,
		r.PackID, r.ReporterID, r.Reason, r.At); err != nil {
		return 0, err
	}
	res, err := s.db.exec(`SELECT count(*) FROM custom_pack_reports WHERE pack_id = ?`, r.PackID)
	if err != nil || len(res.Rows) == 0 {
		return 0, err
	}
	var n int
	err = scan(res.Rows[0], &n)
	return n, err
}

func (s *sqlStore) SetPackFlags(_ context.Context, id string, hidden, deleted bool) error {
	res, err := s.db.exec(`UPDATE custom_packs SET hidden = ?, deleted = ? WHERE id = ?`, hidden, deleted, id)
	if err != nil {
		return err
	}
	if res.Written == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (s *sqlStore) ReportedPacks(_ context.Context, limit int) ([]store.CustomPack, []store.Report, error) {
	out, err := s.packList(`SELECT `+packCols+` FROM custom_packs c
		WHERE EXISTS (SELECT 1 FROM custom_pack_reports r WHERE r.pack_id = c.id)
		ORDER BY 10 DESC, c.updated_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, nil, err
	}
	var reps []store.Report
	for _, cp := range out {
		res, err := s.db.exec(`SELECT pack_id, reporter_id, reason, at FROM custom_pack_reports
			WHERE pack_id = ? ORDER BY at DESC LIMIT 50`, cp.ID)
		if err != nil {
			return nil, nil, err
		}
		for _, row := range res.Rows {
			var r store.Report
			if err := scan(row, &r.PackID, &r.ReporterID, &r.Reason, &r.At); err != nil {
				return nil, nil, err
			}
			reps = append(reps, r)
		}
	}
	return out, reps, nil
}
