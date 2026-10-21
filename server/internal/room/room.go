// Package room runs one goroutine per room. That goroutine owns all of the
// room's state; connections only send it commands over a channel, so no
// game state is ever shared between goroutines.
package room

import (
	"encoding/json"
	"errors"
	"log/slog"
	"math/rand/v2"
	"strings"
	"sync"
	"time"

	"mindhunt/internal/bot"
	"mindhunt/internal/game"
	"mindhunt/internal/store"
)

// Errors returned by Join.
var (
	ErrRoomFull = errors.New("room is full")
	ErrClosed   = errors.New("room is closed")
)

// BotID is the player id of the practice bot seat.
const BotID = "bot"

// Settings are the room's timing and rule knobs.
type Settings struct {
	Rules          game.Rules
	ReconnectGrace time.Duration
	VoidAfter      time.Duration
	IdleClose      time.Duration
	BotDelay       time.Duration
	Tick           time.Duration
	ActionsPerSec  float64
	ImageBaseURL   string
}

// DefaultSettings match the spec.
func DefaultSettings() Settings {
	return Settings{
		Rules:          game.DefaultRules,
		ReconnectGrace: 90 * time.Second,
		VoidAfter:      10 * time.Minute,
		IdleClose:      30 * time.Minute,
		BotDelay:       2500 * time.Millisecond,
		Tick:           250 * time.Millisecond,
		ActionsPerSec:  5,
		ImageBaseURL:   "/assets",
	}
}

// Conn is one live socket attached to a seat. The transport layer drains
// Outbox and closes the socket when Done fires.
type Conn struct {
	send     chan []byte
	closed   chan struct{}
	once     sync.Once
	lastCseq int64 // touched only by the room goroutine
}

// NewConn makes a connection handle.
func NewConn() *Conn {
	return &Conn{send: make(chan []byte, 128), closed: make(chan struct{})}
}

// Outbox yields encoded server messages.
func (c *Conn) Outbox() <-chan []byte { return c.send }

// Done is closed when the room drops this connection.
func (c *Conn) Done() <-chan struct{} { return c.closed }

// Close drops the connection.
func (c *Conn) Close() { c.once.Do(func() { close(c.closed) }) }

func (c *Conn) push(b []byte) {
	select {
	case <-c.closed:
	case c.send <- b:
	default: // slow client: drop it rather than block the room
		c.Close()
	}
}

type seat struct {
	playerID       string
	name           string
	isBot          bool
	conn           *Conn
	connected      bool
	disconnectedAt time.Time
	ready          bool
	seq            int64

	activeMs    int64
	activeSince time.Time
	deadline    time.Time
	botNextAt   time.Time

	tokens     float64
	lastRefill time.Time
}

// Info is a summary used by the HTTP layer.
type Info struct {
	Phase    Phase `json:"phase"`
	Players  int   `json:"players"`
	Full     bool  `json:"full"`
	Practice bool  `json:"practice"`
}

type cmdJoin struct {
	playerID, name string
	conn           *Conn
	reply          chan error
}
type cmdLeave struct{ conn *Conn }
type cmdMsg struct {
	conn *Conn
	msg  ClientMsg
}
type cmdInfo struct{ reply chan Info }
type cmdInspect struct {
	fn   func(*Room)
	done chan struct{}
}

// Room is a two-seat game room.
type Room struct {
	Code     string
	practice bool
	packs    game.Packs
	pack     *game.Pack // the pack chosen in the lobby; a running match keeps its own
	set      Settings
	rec      store.Recorder
	now      func() time.Time
	rng      *rand.Rand
	onClose  func(code string)

	cmds chan any
	done chan struct{}

	seats      [2]*seat
	host       int
	phase      Phase
	cfg        Config
	match      *game.Match
	matchID    string
	round      int
	result     *ResultView
	rematch    [2]bool
	emptySince time.Time
}

// New creates a room and starts its goroutine.
func New(code string, practice bool, packs game.Packs, pack *game.Pack, set Settings, rec store.Recorder, onClose func(string)) *Room {
	if pack == nil {
		pack = packs.Default()
	}
	r := &Room{
		Code: code, practice: practice, packs: packs, pack: pack, set: set, rec: rec,
		now: time.Now, rng: rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())),
		onClose: onClose, cmds: make(chan any, 64), done: make(chan struct{}),
		phase: PhaseLobby,
		cfg:   Config{PackID: pack.ID, Difficulty: "normal"},
	}
	if practice {
		r.seats[1] = &seat{playerID: BotID, name: "Practice Bot", isBot: true, ready: true}
	}
	go r.run()
	return r
}

// Join attaches conn to the player's seat, taking a free seat if needed.
func (r *Room) Join(playerID, name string, conn *Conn) error {
	reply := make(chan error, 1)
	if !r.post(cmdJoin{playerID: playerID, name: name, conn: conn, reply: reply}) {
		return ErrClosed
	}
	select {
	case err := <-reply:
		return err
	case <-r.done:
		return ErrClosed
	}
}

// Leave detaches a connection (socket closed).
func (r *Room) Leave(conn *Conn) { r.post(cmdLeave{conn: conn}) }

// Deliver passes a client message to the room.
func (r *Room) Deliver(conn *Conn, msg ClientMsg) { r.post(cmdMsg{conn: conn, msg: msg}) }

// Info returns a summary of the room.
func (r *Room) Info() (Info, bool) {
	reply := make(chan Info, 1)
	if !r.post(cmdInfo{reply: reply}) {
		return Info{}, false
	}
	select {
	case i := <-reply:
		return i, true
	case <-r.done:
		return Info{}, false
	}
}

// inspect runs fn on the room goroutine (tests only).
func (r *Room) inspect(fn func(*Room)) {
	done := make(chan struct{})
	if r.post(cmdInspect{fn: fn, done: done}) {
		select {
		case <-done:
		case <-r.done:
		}
	}
}

// Done is closed when the room shuts down.
func (r *Room) Done() <-chan struct{} { return r.done }

func (r *Room) post(c any) bool {
	select {
	case <-r.done:
		return false
	case r.cmds <- c:
		return true
	}
}

func (r *Room) run() {
	t := time.NewTicker(r.set.Tick)
	defer t.Stop()
	defer func() {
		close(r.done)
		for _, s := range r.seats {
			if s != nil && s.conn != nil {
				s.conn.Close()
			}
		}
		if r.onClose != nil {
			r.onClose(r.Code)
		}
	}()
	for {
		select {
		case c := <-r.cmds:
			r.handle(c)
		case <-t.C:
			changed, closeRoom := r.tick(r.now())
			if changed {
				r.broadcast()
			}
			if closeRoom {
				return
			}
		}
	}
}

func (r *Room) handle(c any) {
	switch c := c.(type) {
	case cmdJoin:
		err := r.join(c)
		c.reply <- err
	case cmdLeave:
		r.leave(c.conn)
	case cmdMsg:
		r.onMsg(c)
	case cmdInfo:
		n := 0
		for _, s := range r.seats {
			if s != nil {
				n++
			}
		}
		c.reply <- Info{Phase: r.phase, Players: n, Full: n == 2, Practice: r.practice}
	case cmdInspect:
		c.fn(r)
		close(c.done)
	}
}

func (r *Room) seatOf(playerID string) int {
	for i, s := range r.seats {
		if s != nil && s.playerID == playerID {
			return i
		}
	}
	return -1
}

func (r *Room) seatOfConn(c *Conn) int {
	for i, s := range r.seats {
		if s != nil && s.conn == c {
			return i
		}
	}
	return -1
}

func (r *Room) join(c cmdJoin) error {
	now := r.now()
	idx := r.seatOf(c.playerID)
	if idx < 0 {
		if r.phase != PhaseLobby {
			return ErrRoomFull
		}
		for i, s := range r.seats {
			if s == nil {
				idx = i
				break
			}
		}
		if idx < 0 {
			return ErrRoomFull
		}
		r.seats[idx] = &seat{playerID: c.playerID, lastRefill: now, tokens: 10}
		if r.seats[1-idx] == nil || r.seats[1-idx].isBot {
			r.host = idx
		}
	}
	s := r.seats[idx]
	if s.conn != nil && s.conn != c.conn {
		r.send(idx, EvKicked, map[string]string{"reason": "opened in another tab"}, 0)
		s.conn.Close()
	}
	s.name = cleanName(c.name)
	s.conn = c.conn
	s.connected = true
	s.disconnectedAt = time.Time{}
	c.conn.lastCseq = 0
	r.emptySince = time.Time{}
	r.startClock(idx, now)
	if r.match != nil {
		r.send(idx, EvPool, r.poolView(), 0)
	}
	r.broadcast()
	return nil
}

func (r *Room) leave(conn *Conn) {
	idx := r.seatOfConn(conn)
	if idx < 0 {
		return
	}
	s := r.seats[idx]
	s.conn = nil
	s.connected = false
	s.disconnectedAt = r.now()
	r.stopClock(idx, r.now())
	r.broadcast()
}

func (r *Room) onMsg(c cmdMsg) {
	idx := r.seatOfConn(c.conn)
	if idx < 0 {
		return // stale connection replaced by a newer tab
	}
	if c.msg.Type == MsgPing {
		r.send(idx, EvPong, nil, c.msg.Cseq)
		return
	}
	// Idempotency: each connection's cseq must increase. A repeat (double
	// click, client retry) is dropped without touching state.
	if c.msg.Cseq <= c.conn.lastCseq {
		return
	}
	c.conn.lastCseq = c.msg.Cseq
	now := r.now()
	if !r.seats[idx].allow(now, r.set.ActionsPerSec) {
		r.fail(idx, c.msg.Cseq, game.Errf(game.CodeRateLimited, "slow down"))
		return
	}
	if err := r.dispatch(idx, c.msg, now); err != nil {
		r.fail(idx, c.msg.Cseq, err)
		return
	}
	if c.msg.Type != MsgMatchText { // a preview changes nothing
		r.broadcast()
	}
}

func (s *seat) allow(now time.Time, rate float64) bool {
	if rate <= 0 {
		return true
	}
	s.tokens += now.Sub(s.lastRefill).Seconds() * rate
	s.lastRefill = now
	if s.tokens > 2*rate {
		s.tokens = 2 * rate
	}
	if s.tokens < 1 {
		return false
	}
	s.tokens--
	return true
}

func decode[T any](raw json.RawMessage) (T, error) {
	var v T
	if len(raw) == 0 {
		raw = []byte("{}")
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return v, game.Errf(game.CodeBadRequest, "malformed message")
	}
	return v, nil
}

func (r *Room) needPhase(ps ...Phase) error {
	for _, p := range ps {
		if r.phase == p {
			return nil
		}
	}
	return game.Errf(game.CodeWrongPhase, "not allowed during %s", r.phase)
}

func (r *Room) dispatch(idx int, m ClientMsg, now time.Time) error {
	switch m.Type {
	case MsgSetConfig:
		if err := r.needPhase(PhaseLobby); err != nil {
			return err
		}
		if idx != r.host {
			return game.Errf(game.CodeNotHost, "only the host can change settings")
		}
		cfg, err := decode[Config](m.Data)
		if err != nil {
			return err
		}
		pack := r.pack
		if cfg.PackID != r.pack.ID {
			p, ok := r.packs.Get(cfg.PackID)
			if !ok {
				return game.Errf(game.CodeBadRequest, "unknown pack (open a custom pack's link first so the server loads it)")
			}
			pack = p
		}
		if _, ok := PoolSizes[cfg.Difficulty]; !ok {
			return game.Errf(game.CodeBadRequest, "unknown difficulty")
		}
		if cfg.FreeText && r.practice {
			return game.Errf(game.CodeBadRequest, "free-text questions need a human opponent to answer them")
		}
		if !AllowedTimers[cfg.TimerS] {
			return game.Errf(game.CodeBadRequest, "timer must be off, 30, 60, 90 or 120 seconds")
		}
		r.pack = pack
		if cfg != r.cfg {
			r.cfg = cfg
			if o := r.seats[1-idx]; o != nil && !o.isBot {
				o.ready = false // settings changed under them
			}
		}
		return nil

	case MsgReady:
		if err := r.needPhase(PhaseLobby); err != nil {
			return err
		}
		v, err := decode[struct {
			Ready bool `json:"ready"`
		}](m.Data)
		if err != nil {
			return err
		}
		r.seats[idx].ready = v.Ready
		return nil

	case MsgStart:
		if err := r.needPhase(PhaseLobby); err != nil {
			return err
		}
		if idx != r.host {
			return game.Errf(game.CodeNotHost, "only the host can start")
		}
		o := r.seats[1-idx]
		if o == nil {
			return game.Errf(game.CodeNotReady, "waiting for a second player")
		}
		if !o.ready {
			return game.Errf(game.CodeNotReady, "%s is not ready yet", o.name)
		}
		r.startRound(now)
		return nil

	case MsgPickSecret:
		if err := r.needPhase(PhasePicking); err != nil {
			return err
		}
		v, err := decode[struct {
			CandidateID string `json:"candidate_id"`
		}](m.Data)
		if err != nil {
			return err
		}
		if err := r.match.PickSecret(idx, v.CandidateID); err != nil {
			return err
		}
		if r.match.BothPicked() {
			r.startHunt(now)
		}
		return nil

	case MsgAsk:
		if err := r.needPhase(PhaseHunting); err != nil {
			return err
		}
		v, err := decode[struct {
			Predicate game.Predicate `json:"predicate"`
		}](m.Data)
		if err != nil {
			return err
		}
		mv, err := r.match.Ask(idx, v.Predicate, now)
		if err != nil {
			return err
		}
		r.send(idx, EvAnswer, mv, m.Cseq)
		r.afterMove(idx, mv, now)
		return nil

	case MsgGuess:
		if err := r.needPhase(PhaseHunting); err != nil {
			return err
		}
		v, err := decode[struct {
			CandidateID string `json:"candidate_id"`
		}](m.Data)
		if err != nil {
			return err
		}
		mv, err := r.match.Guess(idx, v.CandidateID, now)
		if err != nil {
			return err
		}
		r.send(idx, EvGuessResult, mv, m.Cseq)
		r.afterMove(idx, mv, now)
		return nil

	case MsgMatchText:
		if err := r.needPhase(PhaseHunting); err != nil {
			return err
		}
		v, err := decode[struct {
			Text string `json:"text"`
		}](m.Data)
		if err != nil {
			return err
		}
		mv := MatchView{}
		if q, ok := r.match.Pack.MatchStrict(v.Text); ok {
			p := q.Predicate()
			mv = MatchView{Text: q.Text(), Predicate: &p, Exact: true}
		} else if q, ok := r.match.Pack.MatchLoose(v.Text); ok {
			p := q.Predicate()
			mv = MatchView{Text: q.Text(), Predicate: &p}
		}
		r.send(idx, EvMatch, mv, m.Cseq)
		return nil

	case MsgAskText:
		if err := r.needPhase(PhaseHunting); err != nil {
			return err
		}
		v, err := decode[struct {
			Text string `json:"text"`
		}](m.Data)
		if err != nil {
			return err
		}
		mv, pend, err := r.match.AskText(idx, v.Text, r.cfg.FreeText && !r.practice, now)
		if err != nil {
			return err
		}
		if pend != nil {
			r.seats[idx].deadline = time.Time{} // your clock pauses while you wait
			r.send(idx, EvAsked, pend, m.Cseq)
			return nil
		}
		r.send(idx, EvAnswer, mv, m.Cseq)
		r.afterMove(idx, mv, now)
		return nil

	case MsgAnswerText:
		if err := r.needPhase(PhaseHunting); err != nil {
			return err
		}
		v, err := decode[struct {
			ID     int    `json:"id"`
			Answer string `json:"answer"`
		}](m.Data)
		if err != nil {
			return err
		}
		mv, err := r.match.AnswerText(idx, v.ID, game.Answer(v.Answer), now)
		if err != nil {
			return err
		}
		r.send(1-idx, EvAnswer, mv, 0)
		r.afterMove(1-idx, mv, now)
		return nil

	case MsgCancelText:
		if err := r.needPhase(PhaseHunting); err != nil {
			return err
		}
		if err := r.match.CancelText(idx); err != nil {
			return err
		}
		r.resetDeadline(idx, now)
		return nil

	case MsgGiveUp:
		if err := r.needPhase(PhaseHunting); err != nil {
			return err
		}
		mv, err := r.match.GiveUp(idx, now)
		if err != nil {
			return err
		}
		r.afterMove(idx, mv, now)
		return nil

	case MsgClaimWin:
		if err := r.needPhase(PhaseHunting); err != nil {
			return err
		}
		o := r.seats[1-idx]
		if r.match.Status(idx) == game.StatusGaveUp || o.isBot || o.connected ||
			now.Sub(o.disconnectedAt) < r.set.ReconnectGrace {
			return game.Errf(game.CodeCannotClaim, "you can claim the win only after your opponent has been away for %d seconds", int(r.set.ReconnectGrace.Seconds()))
		}
		r.finish(r.match.Forfeit(idx), now)
		return nil

	case MsgRematch:
		if err := r.needPhase(PhaseResults); err != nil {
			return err
		}
		r.rematch[idx] = true
		if o := r.seats[1-idx]; o != nil && o.isBot {
			r.rematch[1-idx] = true
		}
		if r.rematch[0] && r.rematch[1] {
			if r.seats[0] == nil || r.seats[1] == nil {
				return game.Errf(game.CodeNotReady, "your opponent left")
			}
			r.startRound(now)
		}
		return nil

	case MsgToLobby:
		if err := r.needPhase(PhasePicking, PhaseResults, PhaseVoid); err != nil {
			return err
		}
		r.toLobby()
		return nil

	case MsgLeave:
		if err := r.needPhase(PhaseLobby, PhaseResults, PhaseVoid); err != nil {
			return err
		}
		s := r.seats[idx]
		r.seats[idx] = nil
		if r.phase != PhaseLobby {
			r.toLobby()
		}
		if o := r.seats[1-idx]; o != nil && !o.isBot {
			r.host = 1 - idx
			o.ready = false
		}
		if s.conn != nil {
			s.conn.Close()
		}
		return nil
	}
	return game.Errf(game.CodeBadRequest, "unknown message type %q", m.Type)
}

func (r *Room) toLobby() {
	r.phase = PhaseLobby
	r.match = nil
	r.result = nil
	r.rematch = [2]bool{}
	for _, s := range r.seats {
		if s != nil {
			s.ready = s.isBot
			s.activeMs, s.activeSince, s.deadline = 0, time.Time{}, time.Time{}
		}
	}
}

func (r *Room) startRound(now time.Time) {
	pack := r.pack
	r.round++
	r.matchID = "m_" + store.RandomHex(8)
	r.match = game.NewMatch(pack, PoolSizes[r.cfg.Difficulty], r.rng.Uint64(), r.set.Rules)
	r.phase = PhasePicking
	r.result = nil
	r.rematch = [2]bool{}
	for i, s := range r.seats {
		s.activeMs, s.activeSince, s.deadline = 0, time.Time{}, time.Time{}
		if s.isBot {
			_ = r.match.PickSecret(i, r.match.Pool[r.rng.IntN(len(r.match.Pool))].ID)
		} else {
			r.send(i, EvPool, r.poolView(), 0)
		}
	}
	slog.Info("round started", "room", r.Code, "round", r.round, "pack", pack.ID, "pool", len(r.match.Pool))
}

func (r *Room) startHunt(now time.Time) {
	r.match.Start()
	r.phase = PhaseHunting
	for i, s := range r.seats {
		r.startClock(i, now)
		if s.isBot {
			s.botNextAt = now.Add(r.set.BotDelay)
		}
	}
	var ids [2]string
	for i, s := range r.seats {
		if !s.isBot {
			ids[i] = s.playerID
		}
	}
	r.rec.MatchStarted(store.MatchStart{
		MatchID: r.matchID, Room: r.Code, PackID: r.match.Pack.ID, PackVersion: r.match.Pack.Version,
		Seed: r.match.Seed, PoolSize: len(r.match.Pool), PlayerIDs: ids,
		Secrets: [2]string{r.match.Secret(0).ID, r.match.Secret(1).ID}, StartedAt: now,
	})
}

// startClock begins counting active hunting time for a connected seat.
func (r *Room) startClock(i int, now time.Time) {
	s := r.seats[i]
	if r.phase != PhaseHunting || s == nil || r.match.Status(i) != game.StatusHunting {
		return
	}
	if !s.connected && !s.isBot {
		return
	}
	if s.activeSince.IsZero() {
		s.activeSince = now
	}
	r.resetDeadline(i, now)
}

// resetDeadline restarts a seat's move timer, unless it is waiting for the
// opponent to answer its typed question.
func (r *Room) resetDeadline(i int, now time.Time) {
	s := r.seats[i]
	if r.cfg.TimerS <= 0 || s.isBot || !s.connected || r.match.Pending(i) != nil || r.match.Status(i) != game.StatusHunting {
		s.deadline = time.Time{}
		return
	}
	s.deadline = now.Add(time.Duration(r.cfg.TimerS) * time.Second)
}

func (r *Room) stopClock(i int, now time.Time) {
	s := r.seats[i]
	if s == nil {
		return
	}
	if !s.activeSince.IsZero() {
		s.activeMs += now.Sub(s.activeSince).Milliseconds()
		s.activeSince = time.Time{}
	}
	s.deadline = time.Time{}
}

func (r *Room) afterMove(i int, mv game.Move, now time.Time) {
	r.rec.MoveRecorded(r.matchID, i, mv)
	if r.match.Done(i) {
		r.stopClock(i, now)
	} else {
		r.resetDeadline(i, now)
	}
	if r.match.BothDone() {
		r.finish(r.match.Decide(r.stopClocks(now)), now)
	}
}

// stopClocks freezes both seats' active time and returns it.
func (r *Room) stopClocks(now time.Time) [2]int64 {
	var active [2]int64
	for j, s := range r.seats {
		if s != nil {
			r.stopClock(j, now)
			active[j] = s.activeMs
		}
	}
	return active
}

func (r *Room) finish(res game.Result, now time.Time) {
	active := r.stopClocks(now)
	r.phase = PhaseResults
	r.result = r.resultView(res, now)
	r.rec.MatchEnded(store.MatchEnd{
		MatchID: r.matchID, Status: "finished", Winner: res.Winner, Reason: res.Reason,
		Moves: [2]int{r.match.Moves(0), r.match.Moves(1)}, Scores: res.Scores, ActiveMs: active, EndedAt: now,
	})
	slog.Info("round finished", "room", r.Code, "winner", res.Winner, "reason", res.Reason, "scores", res.Scores)
}

func (r *Room) tick(now time.Time) (changed, closeRoom bool) {
	if r.phase == PhaseHunting {
		for i, s := range r.seats {
			if r.phase != PhaseHunting {
				break
			}
			switch {
			case s.isBot && r.match.Status(i) == game.StatusHunting && !now.Before(s.botNextAt):
				r.botMove(i, now)
				changed = true
			case !s.isBot && !s.deadline.IsZero() && !now.Before(s.deadline):
				if mv, err := r.match.Timeout(i, now); err == nil {
					r.afterMove(i, mv, now)
					changed = true
				}
			}
		}
	}
	if r.phase == PhaseHunting && r.allHumansAwayFor(now) >= r.set.VoidAfter {
		r.phase = PhaseVoid
		for j := range r.seats {
			r.stopClock(j, now)
		}
		r.rec.MatchEnded(store.MatchEnd{MatchID: r.matchID, Status: "void", Winner: -1, Reason: "abandoned_by_both", EndedAt: now})
		changed = true
	}
	connected := false
	for _, s := range r.seats {
		if s != nil && s.connected {
			connected = true
		}
	}
	if connected {
		r.emptySince = time.Time{}
	} else {
		if r.emptySince.IsZero() {
			r.emptySince = now
		}
		if now.Sub(r.emptySince) >= r.set.IdleClose {
			closeRoom = true
		}
	}
	return changed, closeRoom
}

// allHumansAwayFor is how long every human seat has been disconnected
// (0 if any human is connected).
func (r *Room) allHumansAwayFor(now time.Time) time.Duration {
	var latest time.Time
	for _, s := range r.seats {
		if s == nil || s.isBot {
			continue
		}
		if s.connected {
			return 0
		}
		if s.disconnectedAt.After(latest) {
			latest = s.disconnectedAt
		}
	}
	if latest.IsZero() {
		return 0
	}
	return now.Sub(latest)
}

func (r *Room) botMove(i int, now time.Time) {
	a := bot.Next(r.match, i, r.rng)
	var (
		mv  game.Move
		err error
	)
	if a.Ask != nil {
		mv, err = r.match.Ask(i, *a.Ask, now)
	} else {
		mv, err = r.match.Guess(i, a.Guess, now)
	}
	if err != nil {
		slog.Warn("bot move rejected", "room", r.Code, "err", err)
		mv, _ = r.match.GiveUp(i, now)
	}
	jitter := time.Duration(r.rng.Int64N(int64(r.set.BotDelay/2) + 1))
	r.seats[i].botNextAt = now.Add(r.set.BotDelay + jitter)
	r.afterMove(i, mv, now)
}

func (r *Room) broadcast() {
	for i, s := range r.seats {
		if s != nil && !s.isBot && s.conn != nil {
			r.send(i, EvState, r.viewFor(i), 0)
		}
	}
}

func (r *Room) send(i int, typ string, data any, replyTo int64) {
	s := r.seats[i]
	if s == nil || s.conn == nil {
		return
	}
	s.seq++
	b, err := json.Marshal(ServerMsg{Type: typ, Seq: s.seq, ReplyTo: replyTo, Data: data})
	if err != nil {
		slog.Error("marshal", "err", err)
		return
	}
	s.conn.push(b)
}

func (r *Room) fail(i int, cseq int64, err error) {
	var ge *game.Error
	if !errors.As(err, &ge) {
		ge = game.Errf(game.CodeBadRequest, "%v", err)
	}
	r.send(i, EvError, ge, cseq)
}

func cleanName(n string) string {
	n = strings.TrimSpace(n)
	if n == "" {
		return "Player"
	}
	if r := []rune(n); len(r) > 24 {
		n = string(r[:24])
	}
	return n
}
