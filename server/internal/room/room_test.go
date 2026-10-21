package room

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"mindhunt/internal/game"
	"mindhunt/internal/store"
)

const roomTestPack = `{
  "id": "colors", "version": 2, "as_of": "2026-01-01", "category": "Test",
  "title": "Colors", "description": "d", "draft": false,
  "attributes": [
    {"key": "color", "label": "Color", "type": "enum", "values": ["red", "blue", "green"]},
    {"key": "big", "label": "Is big", "type": "bool"},
    {"key": "year", "label": "Year", "type": "int", "min": 1900, "max": 2000, "step": 10}
  ],
  "candidates": [
    {"id": "c_00000a", "name": "Alpha",   "attrs": {"color": "red",   "big": true,  "year": 1950}, "image": {"asset": "test/a_111111.webp", "license": "CC0", "credit": "x", "source_url": "https://example.org/a"}},
    {"id": "c_00000b", "name": "Bravo",   "attrs": {"color": "blue",  "big": false, "year": 1960}, "image": {"asset": "test/a_222222.webp", "license": "CC0", "credit": "x", "source_url": "https://example.org/b"}},
    {"id": "c_00000c", "name": "Charlie", "attrs": {"color": "green", "big": true,  "year": 1970}},
    {"id": "c_00000d", "name": "Delta",   "attrs": {"color": "red",   "big": false, "year": 1980}},
    {"id": "c_00000e", "name": "Echo",    "attrs": {"color": "blue",  "big": true,  "year": 1990}},
    {"id": "c_00000f", "name": "Foxtrot", "attrs": {"color": "green", "big": false, "year": 2000}},
    {"id": "c_000010", "name": "Golf",    "attrs": {"color": "red",   "big": true,  "year": 1900}},
    {"id": "c_000011", "name": "Hotel",   "attrs": {"color": "blue",  "big": false, "year": 1910}}
  ]
}`

func testRegistry(t *testing.T) *game.Registry {
	t.Helper()
	reg, err := game.LoadRegistry(fstest.MapFS{"colors.json": {Data: []byte(roomTestPack)}})
	if err != nil {
		t.Fatal(err)
	}
	return reg
}

func fastSettings() Settings {
	s := DefaultSettings()
	s.ReconnectGrace = 300 * time.Millisecond
	s.VoidAfter = 400 * time.Millisecond
	s.IdleClose = time.Hour
	s.BotDelay = 20 * time.Millisecond
	s.Tick = 10 * time.Millisecond
	s.ActionsPerSec = 0 // rate limit tested separately
	return s
}

type msg struct {
	Type    string          `json:"type"`
	Seq     int64           `json:"seq"`
	ReplyTo int64           `json:"reply_to"`
	Data    json.RawMessage `json:"data"`
	raw     string
}

type client struct {
	t        *testing.T
	room     *Room
	id, name string
	conn     *Conn
	cseq     int64

	mu   sync.Mutex
	all  []msg
	wake chan struct{}
}

func connect(t *testing.T, r *Room, id, name string) *client {
	t.Helper()
	c := &client{t: t, room: r, id: id, name: name, wake: make(chan struct{}, 1)}
	c.attach()
	return c
}

func (c *client) attach() {
	c.t.Helper()
	c.conn = NewConn()
	conn := c.conn
	c.mu.Lock()
	c.all = nil // only judge messages from the new connection
	c.mu.Unlock()
	go func() {
		for {
			select {
			case b := <-conn.Outbox():
				var m msg
				if err := json.Unmarshal(b, &m); err != nil {
					c.t.Errorf("bad json: %v", err)
				}
				m.raw = string(b)
				c.mu.Lock()
				c.all = append(c.all, m)
				c.mu.Unlock()
				select {
				case c.wake <- struct{}{}:
				default:
				}
			case <-conn.Done():
				return
			}
		}
	}()
	if err := c.room.Join(c.id, c.name, conn); err != nil {
		c.t.Fatalf("join %s: %v", c.id, err)
	}
	c.cseq = 0
}

func (c *client) drop() { c.room.Leave(c.conn); c.conn.Close() }

func (c *client) send(typ string, data any) int64 {
	c.cseq++
	raw, _ := json.Marshal(data)
	c.room.Deliver(c.conn, ClientMsg{Type: typ, Cseq: c.cseq, Data: raw})
	return c.cseq
}

// waitFor blocks until a message matching f arrives (searching from the start).
func (c *client) waitFor(desc string, f func(msg) bool) msg {
	c.t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		c.mu.Lock()
		for _, m := range c.all {
			if f(m) {
				c.mu.Unlock()
				return m
			}
		}
		c.mu.Unlock()
		select {
		case <-c.wake:
		case <-time.After(20 * time.Millisecond):
		case <-deadline:
			c.t.Fatalf("%s: timed out waiting for %s; last msgs: %s", c.id, desc, c.tail())
		}
	}
}

func (c *client) tail() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []string
	for i := max(0, len(c.all)-4); i < len(c.all); i++ {
		out = append(out, c.all[i].raw)
	}
	return strings.Join(out, "\n")
}

// state waits for a state message satisfying f and returns it decoded.
func (c *client) state(desc string, f func(StateView) bool) StateView {
	c.t.Helper()
	var sv StateView
	c.waitLatest(desc, func(m msg) bool {
		if m.Type != EvState {
			return false
		}
		var v StateView
		if err := json.Unmarshal(m.Data, &v); err != nil {
			c.t.Fatal(err)
		}
		if f(v) {
			sv = v
			return true
		}
		return false
	})
	return sv
}

// waitLatest checks only the newest message of each poll.
func (c *client) waitLatest(desc string, f func(msg) bool) {
	c.t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		c.mu.Lock()
		var last *msg
		for i := len(c.all) - 1; i >= 0; i-- {
			if c.all[i].Type == EvState {
				last = &c.all[i]
				break
			}
		}
		c.mu.Unlock()
		if last != nil && f(*last) {
			return
		}
		select {
		case <-c.wake:
		case <-time.After(20 * time.Millisecond):
		case <-deadline:
			c.t.Fatalf("%s: timed out waiting for %s; last msgs: %s", c.id, desc, c.tail())
		}
	}
}

func (c *client) reply(cseq int64) msg {
	c.t.Helper()
	return c.waitFor("reply", func(m msg) bool { return m.ReplyTo == cseq && m.Type != EvPong })
}

func (c *client) pool() PoolView {
	c.t.Helper()
	var pv PoolView
	m := c.waitFor("pool", func(m msg) bool { return m.Type == EvPool })
	json.Unmarshal(m.Data, &pv)
	return pv
}

func newRoom(t *testing.T, practice bool) *Room {
	t.Helper()
	r := New("TEST01", practice, testRegistry(t), nil, fastSettings(), store.Noop{}, nil)
	return r
}

func ask(attr, op string, v any) map[string]any {
	raw, _ := json.Marshal(v)
	return map[string]any{"predicate": game.Predicate{Attr: attr, Op: op, Value: raw}}
}

// startHunt brings a fresh room to the hunting phase with chosen secrets.
func startHunt(t *testing.T, r *Room, secretA, secretB string) (*client, *client) {
	t.Helper()
	a := connect(t, r, "pA", "Asha")
	b := connect(t, r, "pB", "Bilal")
	a.send(MsgSetConfig, Config{PackID: "colors", Difficulty: "hard"})
	b.send(MsgReady, map[string]bool{"ready": true})
	a.state("guest ready", func(s StateView) bool { return s.Opponent != nil && s.Opponent.Ready })
	a.send(MsgStart, nil)
	a.pool()
	b.pool()
	a.send(MsgPickSecret, map[string]string{"candidate_id": secretA})
	b.send(MsgPickSecret, map[string]string{"candidate_id": secretB})
	a.state("hunting", func(s StateView) bool { return s.Phase == PhaseHunting })
	b.state("hunting", func(s StateView) bool { return s.Phase == PhaseHunting })
	return a, b
}

func TestLobbySeatsAndReconnect(t *testing.T) {
	r := newRoom(t, false)
	a := connect(t, r, "pA", "Asha")
	s := a.state("alone", func(s StateView) bool { return s.Phase == PhaseLobby })
	if !s.You.Host || s.Opponent != nil {
		t.Fatalf("first player should be host alone: %+v", s)
	}
	b := connect(t, r, "pB", "Bilal")
	a.state("opponent joined", func(s StateView) bool { return s.Opponent != nil && s.Opponent.Name == "Bilal" })

	// A third player is refused.
	if err := r.Join("pC", "Chen", NewConn()); err != ErrRoomFull {
		t.Fatalf("third player: %v", err)
	}

	// Guest cannot change settings or start.
	if m := b.reply(b.send(MsgStart, nil)); !strings.Contains(m.raw, game.CodeNotHost) {
		t.Fatalf("guest start: %s", m.raw)
	}
	// Host cannot start before guest is ready.
	if m := a.reply(a.send(MsgStart, nil)); !strings.Contains(m.raw, game.CodeNotReady) {
		t.Fatalf("early start: %s", m.raw)
	}

	// Refreshing a tab reconnects to the same seat and kicks the old socket.
	old := b.conn
	b.attach()
	select {
	case <-old.Done():
	case <-time.After(time.Second):
		t.Fatal("old connection not closed")
	}
	s = b.state("reconnected", func(s StateView) bool { return s.You.Seat == 1 })
	if s.You.Host {
		t.Fatal("guest became host")
	}

	// Disconnect is visible to the other player.
	b.drop()
	a.state("opponent offline", func(s StateView) bool { return s.Opponent != nil && !s.Opponent.Connected })
}

// stripAllowed removes fields that legitimately contain candidate ids.
func stripAllowed(t *testing.T, m msg) string {
	var v map[string]any
	json.Unmarshal([]byte(m.raw), &v)
	if m.Type == EvPool {
		return "" // the pool lists every candidate by design
	}
	if d, ok := v["data"].(map[string]any); ok {
		if h, ok := d["hunt"].(map[string]any); ok {
			delete(h, "remaining_ids")
			delete(h, "your_log")
		}
		delete(d, "guess_id") // own guess echoed back
	}
	out, _ := json.Marshal(v)
	return string(out)
}

func TestSecretNeverLeaksBeforeResults(t *testing.T) {
	r := newRoom(t, false)
	a, b := startHunt(t, r, "c_00000a", "c_00000b")

	// Play a few moves on both sides, including a correct guess by A.
	a.reply(a.send(MsgAsk, ask("color", "eq", "blue")))
	b.reply(b.send(MsgAsk, ask("big", "is", true)))
	b.reply(b.send(MsgGuess, map[string]string{"candidate_id": "c_00000c"}))
	a.reply(a.send(MsgGuess, map[string]string{"candidate_id": "c_00000b"}))
	sb := b.state("A solved", func(s StateView) bool { return s.Opponent.Status == "solved" })

	// While hunting, B must not see A's move count, log or secret.
	opp, _ := json.Marshal(sb.Opponent)
	for _, bad := range []string{"moves", "log", "secret"} {
		if strings.Contains(string(opp), bad) {
			t.Fatalf("opponent view leaks %q: %s", bad, opp)
		}
	}

	check := func(c *client, opponentSecret, opponentImage string) {
		c.mu.Lock()
		defer c.mu.Unlock()
		for _, m := range c.all {
			if strings.Contains(m.raw, `"phase":"results"`) {
				break
			}
			s := stripAllowed(t, m)
			if strings.Contains(s, opponentSecret) {
				t.Errorf("%s saw opponent secret before results: %s", c.id, m.raw)
			}
			if strings.Contains(m.raw, opponentImage) || strings.Contains(m.raw, `"image"`) {
				t.Errorf("%s saw an image reference before results: %s", c.id, m.raw)
			}
		}
	}
	check(b, "c_00000a", "a_111111")

	b.reply(b.send(MsgGuess, map[string]string{"candidate_id": "c_00000a"}))
	sa := a.state("results", func(s StateView) bool { return s.Phase == PhaseResults })
	check(a, "c_00000b", "a_222222")

	res := sa.Result
	if res.Players[0].Secret.ID != "c_00000a" || res.Players[1].Secret.ID != "c_00000b" {
		t.Fatalf("reveal wrong: %+v", res.Players)
	}
	if res.Players[1].Secret.Image == nil || !strings.Contains(res.Players[1].Secret.Image.URL, "a_222222") {
		t.Fatalf("image missing at reveal: %+v", res.Players[1].Secret)
	}
	// A: 1 ask + correct guess = 2. B: ask + wrong guess(3) + correct = 5.
	if res.Winner != 0 || res.Players[0].Moves != 2 || res.Players[1].Moves != 5 {
		t.Fatalf("scoring: winner=%d moves=%d,%d", res.Winner, res.Players[0].Moves, res.Players[1].Moves)
	}
	if res.Players[0].CrackedBy == nil {
		t.Fatal("cracked_by missing")
	}
}

func TestDuplicateSubmitIgnored(t *testing.T) {
	r := newRoom(t, false)
	a, _ := startHunt(t, r, "c_00000a", "c_00000b")
	raw, _ := json.Marshal(ask("color", "eq", "red"))
	a.cseq++
	m := ClientMsg{Type: MsgAsk, Cseq: a.cseq, Data: raw}
	r.Deliver(a.conn, m)
	r.Deliver(a.conn, m) // double click with same cseq
	r.Deliver(a.conn, m)
	a.reply(a.cseq)
	time.Sleep(50 * time.Millisecond)
	s := a.state("after ask", func(s StateView) bool { return s.Hunt.YourMoves >= 1 })
	if s.Hunt.YourMoves != 1 || len(s.Hunt.YourLog) != 1 {
		t.Fatalf("duplicate counted: moves=%d log=%d", s.Hunt.YourMoves, len(s.Hunt.YourLog))
	}
}

func TestInvalidMovesRejected(t *testing.T) {
	r := newRoom(t, false)
	a, b := startHunt(t, r, "c_00000a", "c_00000b")
	cases := []struct {
		typ  string
		data any
		code string
	}{
		{MsgAsk, ask("name", "eq", "Bravo"), game.CodeForbiddenAttr},
		{MsgAsk, ask("color", "eq", "pink"), game.CodeInvalidPredicate},
		{MsgGuess, map[string]string{"candidate_id": "c_ffffff"}, game.CodeNotInPool},
		{MsgPickSecret, map[string]string{"candidate_id": "c_00000c"}, game.CodeWrongPhase},
		{MsgSetConfig, Config{PackID: "colors", Difficulty: "easy"}, game.CodeWrongPhase},
		{MsgRematch, nil, game.CodeWrongPhase},
		{MsgClaimWin, nil, game.CodeCannotClaim},
		{"bogus", nil, game.CodeBadRequest},
	}
	for _, tc := range cases {
		m := a.reply(a.send(tc.typ, tc.data))
		if m.Type != EvError || !strings.Contains(m.raw, tc.code) {
			t.Errorf("%s: want %s, got %s", tc.typ, tc.code, m.raw)
		}
	}
	a.reply(a.send(MsgAsk, ask("color", "eq", "blue")))
	if m := a.reply(a.send(MsgAsk, ask("color", "eq", "blue"))); !strings.Contains(m.raw, game.CodeDuplicateQuestion) {
		t.Errorf("duplicate question: %s", m.raw)
	}
	a.reply(a.send(MsgGuess, map[string]string{"candidate_id": "c_00000b"}))
	if m := a.reply(a.send(MsgAsk, ask("big", "is", true))); !strings.Contains(m.raw, game.CodeAlreadyDone) {
		t.Errorf("ask after solved: %s", m.raw)
	}
	_ = b
	// Moves charged: 1 ask + 1 correct guess; rejected moves cost nothing.
	s := a.state("moves", func(s StateView) bool { return s.Hunt.YourStatus == "solved" })
	if s.Hunt.YourMoves != 2 {
		t.Fatalf("rejected moves were charged: %d", s.Hunt.YourMoves)
	}
}

func TestTieDrawAndRematch(t *testing.T) {
	r := newRoom(t, false)
	a, b := startHunt(t, r, "c_00000a", "c_00000b")
	a.reply(a.send(MsgGuess, map[string]string{"candidate_id": "c_00000b"}))
	b.reply(b.send(MsgGuess, map[string]string{"candidate_id": "c_00000a"}))
	s := a.state("results", func(s StateView) bool { return s.Phase == PhaseResults })
	if s.Result.Winner != -1 || s.Result.Reason != game.ReasonTie {
		t.Fatalf("expected draw: %+v", s.Result)
	}
	if s.Result.Players[0].Grid != "✅" {
		t.Fatalf("grid %q", s.Result.Players[0].Grid)
	}

	a.send(MsgRematch, nil)
	sb := b.state("rematch vote visible", func(s StateView) bool { return s.RematchOpp })
	if sb.Phase != PhaseResults {
		t.Fatal("rematch started with one vote")
	}
	b.send(MsgRematch, nil)
	s = a.state("round 2", func(s StateView) bool { return s.Phase == PhasePicking && s.Round == 2 })
	if s.Room != "TEST01" || s.Hunt == nil || s.Hunt.YourSecretID != "" || s.Hunt.YourMoves != 0 {
		t.Fatalf("round 2 not fresh: %+v", s.Hunt)
	}
	// A new pool is sent for round 2.
	b.waitFor("pool round 2", func(m msg) bool { return m.Type == EvPool && strings.Contains(m.raw, `"round":2`) })

	// Back to lobby from picking keeps both seats.
	a.send(MsgToLobby, nil)
	s = b.state("lobby", func(s StateView) bool { return s.Phase == PhaseLobby })
	if s.Opponent == nil || s.You.Ready {
		t.Fatalf("lobby after round: %+v", s)
	}
}

func TestDisconnectAndClaimWin(t *testing.T) {
	r := newRoom(t, false)
	a, b := startHunt(t, r, "c_00000a", "c_00000b")
	b.drop()
	s := a.state("opponent away", func(s StateView) bool { return s.Opponent != nil && !s.Opponent.Connected })
	if s.Opponent.ClaimableAt == nil {
		t.Fatal("claimable_at missing")
	}
	if m := a.reply(a.send(MsgClaimWin, nil)); !strings.Contains(m.raw, game.CodeCannotClaim) {
		t.Fatalf("claim before grace: %s", m.raw)
	}
	time.Sleep(fastSettings().ReconnectGrace + 50*time.Millisecond)
	a.send(MsgClaimWin, nil)
	s = a.state("forfeit", func(s StateView) bool { return s.Phase == PhaseResults })
	if s.Result.Winner != 0 || s.Result.Reason != game.ReasonAbandoned {
		t.Fatalf("forfeit: %+v", s.Result)
	}
	// The disconnected player sees the result when they come back.
	b.attach()
	sb := b.state("sees result", func(s StateView) bool { return s.Phase == PhaseResults })
	if sb.Result.Winner != 0 {
		t.Fatal("result differs for returning player")
	}
}

func TestReconnectMidHuntKeepsProgress(t *testing.T) {
	r := newRoom(t, false)
	a, b := startHunt(t, r, "c_00000a", "c_00000b")
	a.reply(a.send(MsgAsk, ask("color", "eq", "blue")))
	a.drop()
	b.state("A away", func(s StateView) bool { return !s.Opponent.Connected })
	a.attach()
	a.pool() // pool resent on reconnect
	s := a.state("restored", func(s StateView) bool { return s.Phase == PhaseHunting })
	if s.Hunt.YourMoves != 1 || s.Hunt.YourSecretID != "c_00000a" || len(s.Hunt.YourLog) != 1 {
		t.Fatalf("progress lost: %+v", s.Hunt)
	}
	// cseq restarts at 1 on the new connection and is accepted.
	if m := a.reply(a.send(MsgAsk, ask("big", "is", true))); m.Type != EvAnswer {
		t.Fatalf("ask after reconnect: %s", m.raw)
	}
}

func TestVoidWhenBothAway(t *testing.T) {
	r := newRoom(t, false)
	a, b := startHunt(t, r, "c_00000a", "c_00000b")
	a.drop()
	b.drop()
	time.Sleep(fastSettings().VoidAfter + 100*time.Millisecond)
	var phase Phase
	r.inspect(func(r *Room) { phase = r.phase })
	if phase != PhaseVoid {
		t.Fatalf("phase %s, want void", phase)
	}
}

func TestTurnTimerChargesAMove(t *testing.T) {
	r := newRoom(t, false)
	a, _ := startHunt(t, r, "c_00000a", "c_00000b")
	r.inspect(func(r *Room) {
		r.cfg.TimerS = 1
		r.startClock(0, r.now())
		r.seats[0].deadline = r.now().Add(150 * time.Millisecond)
	})
	s := a.state("timeout", func(s StateView) bool { return s.Hunt.YourMoves == 1 })
	if s.Hunt.YourLog[0].Kind != game.KindTimeout {
		t.Fatalf("log: %+v", s.Hunt.YourLog)
	}
}

func TestRateLimit(t *testing.T) {
	set := fastSettings()
	set.ActionsPerSec = 5
	r := New("RATE01", false, testRegistry(t), nil, set, store.Noop{}, nil)
	a := connect(t, r, "pA", "Asha")
	limited := false
	for range 30 {
		a.send(MsgReady, map[string]bool{"ready": true})
	}
	time.Sleep(100 * time.Millisecond)
	a.mu.Lock()
	for _, m := range a.all {
		if strings.Contains(m.raw, game.CodeRateLimited) {
			limited = true
		}
	}
	a.mu.Unlock()
	if !limited {
		t.Fatal("30 instant actions were not rate limited")
	}
}

func TestPracticeBotPlaysFullRound(t *testing.T) {
	r := newRoom(t, true)
	a := connect(t, r, "pA", "Asha")
	s := a.state("lobby with bot", func(s StateView) bool { return s.Opponent != nil })
	if !s.Opponent.Bot || !s.Opponent.Ready || !s.You.Host {
		t.Fatalf("practice lobby: %+v", s)
	}
	a.send(MsgSetConfig, Config{PackID: "colors", Difficulty: "hard"})
	a.send(MsgStart, nil)
	a.pool()
	a.send(MsgPickSecret, map[string]string{"candidate_id": "c_00000c"})
	a.state("hunting", func(s StateView) bool { return s.Phase == PhaseHunting })
	s = a.state("bot solved", func(s StateView) bool { return s.Opponent.Status == "solved" })
	// Find the bot's secret by elimination and solve.
	for _, id := range []string{"c_00000a", "c_00000b", "c_00000c", "c_00000d", "c_00000e", "c_00000f", "c_000010", "c_000011"} {
		m := a.reply(a.send(MsgGuess, map[string]string{"candidate_id": id}))
		if strings.Contains(m.raw, `"answer":"correct"`) {
			break
		}
	}
	s = a.state("results", func(s StateView) bool { return s.Phase == PhaseResults })
	bp := s.Result.Players[1]
	if !bp.Bot || bp.Status != "solved" || bp.Moves < 2 {
		t.Fatalf("bot result: %+v", bp)
	}
	// Rematch against the bot needs only the human's vote.
	a.send(MsgRematch, nil)
	a.state("round 2", func(s StateView) bool { return s.Phase == PhasePicking && s.Round == 2 })
}

func TestLeaveFreesSeatAndPassesHost(t *testing.T) {
	r := newRoom(t, false)
	a := connect(t, r, "pA", "Asha")
	b := connect(t, r, "pB", "Bilal")
	a.state("both", func(s StateView) bool { return s.Opponent != nil })
	a.send(MsgLeave, nil)
	s := b.state("alone", func(s StateView) bool { return s.Opponent == nil })
	if !s.You.Host {
		t.Fatal("host not passed on")
	}
	c := connect(t, r, "pC", "Chen")
	c.state("joined", func(s StateView) bool { return s.Opponent != nil && s.Opponent.Name == "Bilal" })
}

func TestFreeTextQuestionsAndAudit(t *testing.T) {
	r := newRoom(t, false)
	a := connect(t, r, "pA", "Asha")
	b := connect(t, r, "pB", "Bilal")
	a.send(MsgSetConfig, Config{PackID: "colors", Difficulty: "hard", FreeText: true})
	b.send(MsgReady, map[string]bool{"ready": true})
	a.state("ready", func(s StateView) bool { return s.Opponent != nil && s.Opponent.Ready && s.Config.FreeText })
	a.send(MsgStart, nil)
	a.pool()
	a.send(MsgPickSecret, map[string]string{"candidate_id": "c_00000a"})
	b.send(MsgPickSecret, map[string]string{"candidate_id": "c_00000b"}) // Bravo: blue, not big
	a.state("hunting", func(s StateView) bool { return s.Phase == PhaseHunting })

	// Preview: exact, loose and nothing; previews don't broadcast.
	if m := a.reply(a.send(MsgMatchText, map[string]string{"text": "is their color red"})); !strings.Contains(m.raw, `"exact":true`) {
		t.Fatalf("exact preview: %s", m.raw)
	}
	if m := a.reply(a.send(MsgMatchText, map[string]string{"text": "a red one?"})); !strings.Contains(m.raw, `"exact":false`) || !strings.Contains(m.raw, "red") {
		t.Fatalf("loose preview: %s", m.raw)
	}
	if m := a.reply(a.send(MsgMatchText, map[string]string{"text": "is it shiny"})); strings.Contains(m.raw, "predicate") {
		t.Fatalf("empty preview: %s", m.raw)
	}

	// A known question typed in is answered by the data.
	m := a.reply(a.send(MsgAskText, map[string]string{"text": "Is their color red?"}))
	if m.Type != EvAnswer || !strings.Contains(m.raw, `"answer":"no"`) || strings.Contains(m.raw, `"manual"`) {
		t.Fatalf("auto answer: %s", m.raw)
	}

	// An unknown question goes to the opponent.
	m = a.reply(a.send(MsgAskText, map[string]string{"text": "Is it shiny?"}))
	if m.Type != EvAsked {
		t.Fatalf("expected asked, got %s", m.raw)
	}
	sb := b.state("incoming", func(s StateView) bool { return s.Hunt.Incoming != nil })
	if sb.Hunt.Incoming.Text != "Is it shiny?" || sb.Hunt.UnknownsLeft != 3 {
		t.Fatalf("incoming: %+v", sb.Hunt)
	}
	sa := a.state("pending", func(s StateView) bool { return s.Hunt.Pending != nil })
	if sa.Hunt.YourMoves != 1 {
		t.Fatalf("charged before answer: %d", sa.Hunt.YourMoves)
	}
	b.send(MsgAnswerText, map[string]any{"id": sb.Hunt.Incoming.ID, "answer": "unknown"})
	a.waitFor("manual answer", func(m msg) bool { return m.Type == EvAnswer && strings.Contains(m.raw, `"manual":true`) })
	sb = b.state("unknown used", func(s StateView) bool { return s.Hunt.Incoming == nil })
	if sb.Hunt.UnknownsLeft != 2 {
		t.Fatalf("unknowns left %d", sb.Hunt.UnknownsLeft)
	}

	// Two lies about Bravo (blue): "blue" → No, "green" → Yes.
	for _, q := range []struct{ text, ans string }{{"is it blue", "no"}, {"is it green", "yes"}} {
		a.reply(a.send(MsgAskText, map[string]string{"text": q.text}))
		s := b.state("incoming "+q.text, func(s StateView) bool { return s.Hunt.Incoming != nil && s.Hunt.Incoming.Text == q.text })
		b.send(MsgAnswerText, map[string]any{"id": s.Hunt.Incoming.ID, "answer": q.ans})
		a.state("answered "+q.text, func(s StateView) bool { return s.Hunt.Pending == nil && len(s.Hunt.YourLog) >= 3 })
	}

	// Cancel a waiting question at no cost.
	a.reply(a.send(MsgAskText, map[string]string{"text": "Is it heavy?"}))
	before := a.state("pending heavy", func(s StateView) bool { return s.Hunt.Pending != nil }).Hunt.YourMoves
	a.send(MsgCancelText, nil)
	if s := a.state("cancelled", func(s StateView) bool { return s.Hunt.Pending == nil }); s.Hunt.YourMoves != before {
		t.Fatal("cancel cost a move")
	}

	// B wins on moves (1 vs 5) but gave 2 contradicting answers.
	a.reply(a.send(MsgGuess, map[string]string{"candidate_id": "c_00000b"}))
	b.reply(b.send(MsgGuess, map[string]string{"candidate_id": "c_00000a"}))
	s := a.state("results", func(s StateView) bool { return s.Phase == PhaseResults })
	if s.Result.Winner != 0 || s.Result.Reason != game.ReasonContradictions {
		t.Fatalf("result: winner=%d reason=%s", s.Result.Winner, s.Result.Reason)
	}
	if flags := s.Result.Players[1].AnswerFlags; len(flags) != 2 {
		t.Fatalf("flags: %+v", flags)
	}
}

func TestFreeTextOffAndPractice(t *testing.T) {
	r := newRoom(t, false)
	a, _ := startHunt(t, r, "c_00000a", "c_00000b")
	if m := a.reply(a.send(MsgAskText, map[string]string{"text": "Is it shiny?"})); !strings.Contains(m.raw, game.CodeNoMatch) {
		t.Fatalf("free text off: %s", m.raw)
	}
	if m := a.reply(a.send(MsgAskText, map[string]string{"text": "is their year more than 1950"})); m.Type != EvAnswer {
		t.Fatalf("matched text with free text off: %s", m.raw)
	}

	p := newRoom(t, true)
	h := connect(t, p, "pH", "Solo")
	if m := h.reply(h.send(MsgSetConfig, Config{PackID: "colors", Difficulty: "hard", FreeText: true})); m.Type != EvError {
		t.Fatalf("practice free text allowed: %s", m.raw)
	}
}
