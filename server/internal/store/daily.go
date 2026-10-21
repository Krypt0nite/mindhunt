package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sort"
	"sync"
	"time"

	"mindhunt/internal/game"
)

// DailyPlay is one guest's attempt at one day's puzzle. The move log is the
// source of truth; the match is rebuilt from it by replay.
type DailyPlay struct {
	Date       string      `json:"date"` // YYYY-MM-DD in the daily time zone
	PlayerID   string      `json:"player_id"`
	Moves      []game.Move `json:"moves"`
	Status     string      `json:"status"` // hunting | solved | gave_up
	Score      int         `json:"score"`
	StartedAt  time.Time   `json:"started_at"`
	FinishedAt *time.Time  `json:"finished_at,omitempty"`
}

// DailyStats summarises a day across all players.
type DailyStats struct {
	Finished int         `json:"finished"`
	Solved   int         `json:"solved"`
	Scores   map[int]int `json:"scores"` // moves -> number of solvers
}

// Daily stores daily plays.
type Daily interface {
	LoadPlay(ctx context.Context, date, playerID string) (DailyPlay, bool, error)
	SavePlay(ctx context.Context, p DailyPlay) error
	DayStats(ctx context.Context, date string) (DailyStats, error)
	// PlayHistory returns the player's plays, newest first.
	PlayHistory(ctx context.Context, playerID string, limit int) ([]DailyPlay, error)
}

// MemoryDaily keeps daily plays in memory.
type MemoryDaily struct {
	mu    sync.Mutex
	plays map[string]DailyPlay // date|player
}

// NewMemoryDaily returns an empty store.
func NewMemoryDaily() *MemoryDaily { return &MemoryDaily{plays: map[string]DailyPlay{}} }

func (m *MemoryDaily) LoadPlay(_ context.Context, date, playerID string) (DailyPlay, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.plays[date+"|"+playerID]
	p.Moves = append([]game.Move(nil), p.Moves...)
	return p, ok, nil
}

func (m *MemoryDaily) SavePlay(_ context.Context, p DailyPlay) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p.Moves = append([]game.Move(nil), p.Moves...)
	m.plays[p.Date+"|"+p.PlayerID] = p
	return nil
}

func (m *MemoryDaily) DayStats(_ context.Context, date string) (DailyStats, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	st := DailyStats{Scores: map[int]int{}}
	for _, p := range m.plays {
		if p.Date != date || p.Status == string(game.StatusHunting) {
			continue
		}
		st.Finished++
		if p.Status == string(game.StatusSolved) {
			st.Solved++
			st.Scores[p.Score]++
		}
	}
	return st, nil
}

func (m *MemoryDaily) PlayHistory(_ context.Context, playerID string, limit int) ([]DailyPlay, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []DailyPlay
	for _, p := range m.plays {
		if p.PlayerID == playerID {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date > out[j].Date })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

const dailySchema = `
CREATE TABLE IF NOT EXISTS daily_plays (
  day         DATE NOT NULL,
  player_id   TEXT NOT NULL,
  moves       JSONB NOT NULL,
  status      TEXT NOT NULL,
  score       INT NOT NULL,
  started_at  TIMESTAMPTZ NOT NULL,
  finished_at TIMESTAMPTZ,
  PRIMARY KEY (day, player_id)
);
CREATE INDEX IF NOT EXISTS daily_plays_player_idx ON daily_plays (player_id, day DESC);
`

// Daily plays are written synchronously: a player's progress must survive a
// restart, and each request already waits for its own result.

func (p *Postgres) LoadPlay(ctx context.Context, date, playerID string) (DailyPlay, bool, error) {
	var (
		dp    DailyPlay
		moves []byte
		day   time.Time
	)
	err := p.db.QueryRowContext(ctx,
		`SELECT day, player_id, moves, status, score, started_at, finished_at FROM daily_plays WHERE day = $1 AND player_id = $2`,
		date, playerID).Scan(&day, &dp.PlayerID, &moves, &dp.Status, &dp.Score, &dp.StartedAt, &dp.FinishedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return DailyPlay{}, false, nil
	}
	if err != nil {
		return DailyPlay{}, false, err
	}
	dp.Date = day.Format("2006-01-02")
	if err := json.Unmarshal(moves, &dp.Moves); err != nil {
		return DailyPlay{}, false, err
	}
	return dp, true, nil
}

func (p *Postgres) SavePlay(ctx context.Context, dp DailyPlay) error {
	moves, err := json.Marshal(dp.Moves)
	if err != nil {
		return err
	}
	_, err = p.db.ExecContext(ctx, `
		INSERT INTO daily_plays (day, player_id, moves, status, score, started_at, finished_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (day, player_id) DO UPDATE
		SET moves = EXCLUDED.moves, status = EXCLUDED.status, score = EXCLUDED.score, finished_at = EXCLUDED.finished_at`,
		dp.Date, dp.PlayerID, moves, dp.Status, dp.Score, dp.StartedAt, dp.FinishedAt)
	return err
}

func (p *Postgres) DayStats(ctx context.Context, date string) (DailyStats, error) {
	st := DailyStats{Scores: map[int]int{}}
	rows, err := p.db.QueryContext(ctx,
		`SELECT status, score, count(*) FROM daily_plays WHERE day = $1 AND status <> 'hunting' GROUP BY status, score`, date)
	if err != nil {
		return st, err
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var score, n int
		if err := rows.Scan(&status, &score, &n); err != nil {
			return st, err
		}
		st.Finished += n
		if status == string(game.StatusSolved) {
			st.Solved += n
			st.Scores[score] += n
		}
	}
	return st, rows.Err()
}

func (p *Postgres) PlayHistory(ctx context.Context, playerID string, limit int) ([]DailyPlay, error) {
	rows, err := p.db.QueryContext(ctx,
		`SELECT day, status, score FROM daily_plays WHERE player_id = $1 ORDER BY day DESC LIMIT $2`, playerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DailyPlay
	for rows.Next() {
		var dp DailyPlay
		var day time.Time
		if err := rows.Scan(&day, &dp.Status, &dp.Score); err != nil {
			return nil, err
		}
		dp.Date, dp.PlayerID = day.Format("2006-01-02"), playerID
		out = append(out, dp)
	}
	return out, rows.Err()
}
