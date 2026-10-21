package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"mindhunt/internal/game"
)

const schema = `
CREATE TABLE IF NOT EXISTS players (
  id           TEXT PRIMARY KEY,
  token_hash   TEXT NOT NULL UNIQUE,
  display_name TEXT NOT NULL,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS rooms (
  id             BIGSERIAL PRIMARY KEY,
  code           TEXT NOT NULL,
  host_player_id TEXT NOT NULL,
  practice       BOOLEAN NOT NULL DEFAULT false,
  created_at     TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS rooms_code_idx ON rooms (code);
CREATE TABLE IF NOT EXISTS matches (
  id           TEXT PRIMARY KEY,
  room_code    TEXT NOT NULL,
  pack_id      TEXT NOT NULL,
  pack_version INT NOT NULL,
  seed         NUMERIC(20) NOT NULL,
  pool_size    INT NOT NULL,
  status       TEXT NOT NULL DEFAULT 'hunting',
  winner_seat  SMALLINT,
  reason       TEXT,
  started_at   TIMESTAMPTZ NOT NULL,
  ended_at     TIMESTAMPTZ
);
CREATE TABLE IF NOT EXISTS match_players (
  match_id            TEXT NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
  seat                SMALLINT NOT NULL,
  player_id           TEXT,
  secret_candidate_id TEXT NOT NULL,
  moves               INT,
  score               INT,
  active_ms           BIGINT,
  PRIMARY KEY (match_id, seat)
);
CREATE TABLE IF NOT EXISTS moves (
  match_id   TEXT NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
  seat       SMALLINT NOT NULL,
  n          INT NOT NULL,
  kind       TEXT NOT NULL,
  predicate  JSONB,
  text       TEXT NOT NULL,
  guess_id   TEXT,
  answer     TEXT,
  cost       INT NOT NULL,
  total      INT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (match_id, seat, n)
);
`

// Postgres implements Guests (synchronously, with an in-memory cache) and
// Recorder (asynchronously, through a single ordered writer goroutine).
type Postgres struct {
	db    *sql.DB
	cache *MemoryGuests
	jobs  chan job
	wg    sync.WaitGroup

	dropMu  sync.Mutex
	dropped int
}

type job struct {
	name string
	fn   func(ctx context.Context, db *sql.DB) error
}

// OpenPostgres connects, applies the schema and starts the writer.
func OpenPostgres(ctx context.Context, url string) (*Postgres, error) {
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(8)
	db.SetConnMaxIdleTime(5 * time.Minute)
	pctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := db.PingContext(pctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect: %w", err)
	}
	if _, err := db.ExecContext(pctx, schema+dailySchema+customSchema); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	p := &Postgres{db: db, cache: NewMemoryGuests(), jobs: make(chan job, 4096)}
	p.wg.Add(1)
	go p.writer()
	return p, nil
}

func hashToken(t string) string {
	h := sha256.Sum256([]byte(t))
	return hex.EncodeToString(h[:])
}

// Create inserts a guest. Only a hash of the token is stored.
func (p *Postgres) Create(ctx context.Context, name string) (Guest, error) {
	g := Guest{ID: "p_" + RandomHex(8), Token: RandomHex(24), Name: name, CreatedAt: time.Now().UTC()}
	_, err := p.db.ExecContext(ctx,
		`INSERT INTO players (id, token_hash, display_name, created_at) VALUES ($1, $2, $3, $4)`,
		g.ID, hashToken(g.Token), g.Name, g.CreatedAt)
	if err != nil {
		return Guest{}, err
	}
	p.cache.put(g)
	return g, nil
}

// ByToken looks in the cache first, then the database.
func (p *Postgres) ByToken(ctx context.Context, token string) (Guest, error) {
	if token == "" {
		return Guest{}, ErrNotFound
	}
	if g, err := p.cache.ByToken(ctx, token); err == nil {
		return g, nil
	}
	g := Guest{Token: token}
	err := p.db.QueryRowContext(ctx,
		`SELECT id, display_name, created_at FROM players WHERE token_hash = $1`, hashToken(token),
	).Scan(&g.ID, &g.Name, &g.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Guest{}, ErrNotFound
	}
	if err != nil {
		return Guest{}, err
	}
	p.cache.put(g)
	return g, nil
}

// Rename updates a guest's display name.
func (p *Postgres) Rename(ctx context.Context, id, name string) error {
	if _, err := p.db.ExecContext(ctx, `UPDATE players SET display_name = $2 WHERE id = $1`, id, name); err != nil {
		return err
	}
	_ = p.cache.Rename(ctx, id, name)
	return nil
}

func (p *Postgres) enqueue(name string, fn func(ctx context.Context, db *sql.DB) error) {
	select {
	case p.jobs <- job{name: name, fn: fn}:
	default:
		p.dropMu.Lock()
		p.dropped++
		n := p.dropped
		p.dropMu.Unlock()
		slog.Error("db queue full; dropping write", "job", name, "dropped_total", n)
	}
}

func (p *Postgres) writer() {
	defer p.wg.Done()
	for j := range p.jobs {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := j.fn(ctx, p.db); err != nil {
			slog.Error("db write failed", "job", j.name, "err", err)
		}
		cancel()
	}
}

// RoomCreated records a room.
func (p *Postgres) RoomCreated(code, hostID string, practice bool, at time.Time) {
	p.enqueue("room", func(ctx context.Context, db *sql.DB) error {
		_, err := db.ExecContext(ctx,
			`INSERT INTO rooms (code, host_player_id, practice, created_at) VALUES ($1, $2, $3, $4)`,
			code, hostID, practice, at)
		return err
	})
}

// MatchStarted records a match and both locked secrets.
func (p *Postgres) MatchStarted(m MatchStart) {
	p.enqueue("match_start", func(ctx context.Context, db *sql.DB) error {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO matches (id, room_code, pack_id, pack_version, seed, pool_size, started_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			m.MatchID, m.Room, m.PackID, m.PackVersion, strconv.FormatUint(m.Seed, 10), m.PoolSize, m.StartedAt); err != nil {
			return err
		}
		for seat := range 2 {
			var pid any
			if m.PlayerIDs[seat] != "" {
				pid = m.PlayerIDs[seat]
			}
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO match_players (match_id, seat, player_id, secret_candidate_id) VALUES ($1, $2, $3, $4)`,
				m.MatchID, seat, pid, m.Secrets[seat]); err != nil {
				return err
			}
		}
		return tx.Commit()
	})
}

// MoveRecorded appends to the move log.
func (p *Postgres) MoveRecorded(matchID string, seat int, mv game.Move) {
	var pred []byte
	if mv.Predicate != nil {
		pred, _ = json.Marshal(mv.Predicate)
	}
	p.enqueue("move", func(ctx context.Context, db *sql.DB) error {
		var predArg, guessArg, ansArg any
		if pred != nil {
			predArg = string(pred)
		}
		if mv.GuessID != "" {
			guessArg = mv.GuessID
		}
		if mv.Answer != "" {
			ansArg = string(mv.Answer)
		}
		_, err := db.ExecContext(ctx,
			`INSERT INTO moves (match_id, seat, n, kind, predicate, text, guess_id, answer, cost, total, created_at)
			 VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7, $8, $9, $10, $11)`,
			matchID, seat, mv.N, mv.Kind, predArg, mv.Text, guessArg, ansArg, mv.Cost, mv.Total, mv.At)
		return err
	})
}

// MatchEnded records the outcome.
func (p *Postgres) MatchEnded(e MatchEnd) {
	p.enqueue("match_end", func(ctx context.Context, db *sql.DB) error {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		var winner any
		if e.Winner >= 0 {
			winner = e.Winner
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE matches SET status = $2, winner_seat = $3, reason = $4, ended_at = $5 WHERE id = $1`,
			e.MatchID, e.Status, winner, e.Reason, e.EndedAt); err != nil {
			return err
		}
		if e.Status == "finished" {
			for seat := range 2 {
				if _, err := tx.ExecContext(ctx,
					`UPDATE match_players SET moves = $3, score = $4, active_ms = $5 WHERE match_id = $1 AND seat = $2`,
					e.MatchID, seat, e.Moves[seat], e.Scores[seat], e.ActiveMs[seat]); err != nil {
					return err
				}
			}
		}
		return tx.Commit()
	})
}

// Close flushes queued writes (up to timeout) and closes the database.
func (p *Postgres) Close(timeout time.Duration) {
	close(p.jobs)
	done := make(chan struct{})
	go func() { p.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(timeout):
		slog.Warn("db writer did not drain before shutdown")
	}
	p.db.Close()
}
