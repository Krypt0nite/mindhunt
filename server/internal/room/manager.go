package room

import (
	"crypto/rand"
	"errors"
	"sync"
	"time"

	"mindhunt/internal/game"
	"mindhunt/internal/store"
)

// codeAlphabet has no ambiguous characters (no 0/O, 1/I/L).
const codeAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

// Manager creates and finds rooms.
type Manager struct {
	packs game.Packs
	set   Settings
	rec   store.Recorder

	mu    sync.Mutex
	rooms map[string]*Room
}

// NewManager returns an empty manager.
func NewManager(packs game.Packs, set Settings, rec store.Recorder) *Manager {
	return &Manager{packs: packs, set: set, rec: rec, rooms: map[string]*Room{}}
}

// Create opens a new room with a unique code. pack is the starting pack
// (nil for the default); the host can change it in the lobby.
func (m *Manager) Create(hostID string, practice bool, pack *game.Pack) (*Room, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for range 20 {
		code := newCode()
		if _, taken := m.rooms[code]; taken {
			continue
		}
		r := New(code, practice, m.packs, pack, m.set, m.rec, m.remove)
		m.rooms[code] = r
		m.rec.RoomCreated(code, hostID, practice, time.Now())
		return r, nil
	}
	return nil, errors.New("could not allocate a room code")
}

// Get finds a room by code.
func (m *Manager) Get(code string) (*Room, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.rooms[code]
	return r, ok
}

// Count returns the number of open rooms.
func (m *Manager) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.rooms)
}

func (m *Manager) remove(code string) {
	m.mu.Lock()
	delete(m.rooms, code)
	m.mu.Unlock()
}

func newCode() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	for i := range b {
		b[i] = codeAlphabet[int(b[i])%len(codeAlphabet)]
	}
	return string(b)
}
