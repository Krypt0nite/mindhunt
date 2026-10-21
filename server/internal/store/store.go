// Package store persists guests, rooms, matches and move logs.
//
// Gameplay never waits on the database: match events go through Recorder,
// whose Postgres implementation queues writes to a background goroutine.
package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"mindhunt/internal/game"
)

// ErrNotFound is returned for unknown tokens.
var ErrNotFound = errors.New("not found")

// Guest is an anonymous player identity. Token is a bearer secret kept in
// the browser; ID is safe to show.
type Guest struct {
	ID        string    `json:"player_id"`
	Token     string    `json:"token,omitempty"`
	Name      string    `json:"display_name"`
	CreatedAt time.Time `json:"created_at"`
}

// Guests creates and looks up guest identities.
type Guests interface {
	Create(ctx context.Context, name string) (Guest, error)
	ByToken(ctx context.Context, token string) (Guest, error)
	Rename(ctx context.Context, id, name string) error
}

// MatchStart is recorded when a hunt begins (secrets locked).
type MatchStart struct {
	MatchID     string
	Room        string
	PackID      string
	PackVersion int
	Seed        uint64
	PoolSize    int
	PlayerIDs   [2]string // "" for the practice bot
	Secrets     [2]string
	StartedAt   time.Time
}

// MatchEnd is recorded when a round finishes or is voided.
type MatchEnd struct {
	MatchID  string
	Status   string // finished | void
	Winner   int    // -1 draw or void
	Reason   string
	Moves    [2]int
	Scores   [2]int
	ActiveMs [2]int64
	EndedAt  time.Time
}

// Recorder receives match events. Implementations must not block.
type Recorder interface {
	RoomCreated(code, hostID string, practice bool, at time.Time)
	MatchStarted(MatchStart)
	MoveRecorded(matchID string, seat int, mv game.Move)
	MatchEnded(MatchEnd)
}

// Noop discards every event.
type Noop struct{}

func (Noop) RoomCreated(string, string, bool, time.Time) {}
func (Noop) MatchStarted(MatchStart)                     {}
func (Noop) MoveRecorded(string, int, game.Move)         {}
func (Noop) MatchEnded(MatchEnd)                         {}

// MemoryGuests keeps guests in memory (lost on restart).
type MemoryGuests struct {
	mu      sync.RWMutex
	byToken map[string]Guest
	byID    map[string]string // id -> token
}

// NewMemoryGuests returns an empty in-memory guest store.
func NewMemoryGuests() *MemoryGuests {
	return &MemoryGuests{byToken: map[string]Guest{}, byID: map[string]string{}}
}

// Create makes a new guest.
func (m *MemoryGuests) Create(_ context.Context, name string) (Guest, error) {
	g := Guest{ID: "p_" + RandomHex(8), Token: RandomHex(24), Name: name, CreatedAt: time.Now().UTC()}
	m.put(g)
	return g, nil
}

func (m *MemoryGuests) put(g Guest) {
	m.mu.Lock()
	m.byToken[g.Token] = g
	m.byID[g.ID] = g.Token
	m.mu.Unlock()
}

// ByToken finds a guest by bearer token.
func (m *MemoryGuests) ByToken(_ context.Context, token string) (Guest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	g, ok := m.byToken[token]
	if !ok {
		return Guest{}, ErrNotFound
	}
	return g, nil
}

// Rename changes a guest's display name.
func (m *MemoryGuests) Rename(_ context.Context, id, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	tok, ok := m.byID[id]
	if !ok {
		return ErrNotFound
	}
	g := m.byToken[tok]
	g.Name = name
	m.byToken[tok] = g
	return nil
}

// RandomHex returns n random bytes hex-encoded.
func RandomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
