// Package daily runs the daily challenge: one secret per day, the same for
// every player, hunted solo against the server.
//
// The day's pool and secret come from an HMAC of the date under a server
// key, so they are stable across restarts but can't be predicted from the
// source code. A player's progress is stored as a move log and the match is
// rebuilt by replaying it, so nothing lives only in memory.
package daily

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"

	"mindhunt/internal/game"
	"mindhunt/internal/room"
	"mindhunt/internal/store"
)

// Config tunes the daily challenge.
type Config struct {
	// PackIDs are used in turn, one per day (a single id means the same pack every day).
	PackIDs   []string
	PoolSize  int
	Key       []byte
	Location  *time.Location
	Epoch     string // date of puzzle #1, YYYY-MM-DD
	Rules     game.Rules
	ImageBase string
}

// Service serves daily puzzles.
type Service struct {
	cfg   Config
	reg   *game.Registry
	store store.Daily
	now   func() time.Time

	locks sync.Map // date|player -> *sync.Mutex
}

// New builds the service.
func New(cfg Config, reg *game.Registry, st store.Daily) (*Service, error) {
	if len(cfg.PackIDs) == 0 {
		return nil, errors.New("daily: no packs")
	}
	for _, id := range cfg.PackIDs {
		if _, ok := reg.Get(id); !ok {
			return nil, fmt.Errorf("daily: unknown pack %q", id)
		}
	}
	if _, err := time.ParseInLocation("2006-01-02", cfg.Epoch, cfg.Location); err != nil {
		return nil, fmt.Errorf("daily: bad epoch: %w", err)
	}
	if len(cfg.Key) == 0 {
		return nil, errors.New("daily: empty key")
	}
	return &Service{cfg: cfg, reg: reg, store: st, now: time.Now}, nil
}

// Puzzle identifies one day's challenge.
type Puzzle struct {
	Date   string
	Number int
	Pack   *game.Pack
	Seed   uint64
	Secret string
	Next   time.Time // when the next puzzle starts
}

// PuzzleAt returns the puzzle for the day containing t.
func (s *Service) PuzzleAt(t time.Time) Puzzle {
	local := t.In(s.cfg.Location)
	date := local.Format("2006-01-02")
	day, _ := time.ParseInLocation("2006-01-02", date, s.cfg.Location)
	epoch, _ := time.ParseInLocation("2006-01-02", s.cfg.Epoch, s.cfg.Location)
	number := int(day.Sub(epoch).Hours()/24) + 1
	n := len(s.cfg.PackIDs)
	pack := s.reg.Get2(s.cfg.PackIDs[((number-1)%n+n)%n])

	mac := hmac.New(sha256.New, s.cfg.Key)
	fmt.Fprintf(mac, "%s|%s", pack.ID, date)
	sum := mac.Sum(nil)
	seed := binary.BigEndian.Uint64(sum[:8])
	pool := pack.SamplePool(s.cfg.PoolSize, seed)
	secret := pool[binary.BigEndian.Uint64(sum[8:16])%uint64(len(pool))].ID

	return Puzzle{
		Date:   date,
		Number: number,
		Pack:   pack,
		Seed:   seed,
		Secret: secret,
		Next:   day.AddDate(0, 0, 1),
	}
}

// match builds a fresh match for a puzzle: seat 0 is the player, seat 1
// holds the day's secret.
func (s *Service) match(p Puzzle) *game.Match {
	rules := s.cfg.Rules
	rules.ContradictionLimit = 0 // no manual answers in the daily
	m := game.NewMatch(p.Pack, s.cfg.PoolSize, p.Seed, rules)
	_ = m.PickSecret(1, p.Secret)
	_ = m.PickSecret(0, m.Pool[0].ID) // unused: nobody hunts the player
	m.Start()
	return m
}

// replay applies a stored log. It stops at the first move that no longer
// applies (for example after a pack change) and reports how many applied.
func replay(m *game.Match, moves []game.Move) int {
	for i, mv := range moves {
		var err error
		switch mv.Kind {
		case game.KindAsk:
			if mv.Predicate == nil {
				return i
			}
			_, err = m.Ask(0, *mv.Predicate, mv.At)
		case game.KindGuess:
			_, err = m.Guess(0, mv.GuessID, mv.At)
		case game.KindGiveUp:
			_, err = m.GiveUp(0, mv.At)
		default:
			return i
		}
		if err != nil {
			return i
		}
	}
	return len(moves)
}

// View is what a player sees. The secret and its card appear only once the
// player has finished.
type View struct {
	Date         string          `json:"date"`
	Number       int             `json:"number"`
	Next         time.Time       `json:"next"`
	PackID       string          `json:"pack_id"`
	PackTitle    string          `json:"pack_title"`
	PackNoun     string          `json:"pack_noun"`
	Category     string          `json:"category"`
	Pool         *room.PoolView  `json:"pool,omitempty"`
	Status       game.SeatStatus `json:"status"`
	Moves        int             `json:"moves"`
	Log          []game.Move     `json:"log"`
	RemainingIDs []string        `json:"remaining_ids"`
	WrongCost    int             `json:"wrong_guess_cost"`
	Last         *game.Move      `json:"last,omitempty"`
	Result       *Result         `json:"result,omitempty"`
	Streak       Streak          `json:"streak"`
}

// Result is revealed when the player finishes.
type Result struct {
	Secret    room.CandidateCard `json:"secret"`
	Score     int                `json:"score"`
	Grid      string             `json:"grid"`
	CrackedBy *game.Move         `json:"cracked_by,omitempty"`
	Stats     store.DailyStats   `json:"stats"`
	// BetterThan is the share of today's solvers who used more moves (0–100).
	BetterThan int `json:"better_than"`
}

// Streak counts consecutive days solved, ending today or yesterday.
type Streak struct {
	Current int `json:"current"`
	Best    int `json:"best"`
	Played  int `json:"played"`
	Solved  int `json:"solved"`
}

type session struct {
	puzzle Puzzle
	match  *game.Match
	play   store.DailyPlay
}

func (s *Service) lock(date, player string) func() {
	v, _ := s.locks.LoadOrStore(date+"|"+player, &sync.Mutex{})
	mu := v.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

func (s *Service) load(ctx context.Context, player string) (*session, error) {
	p := s.PuzzleAt(s.now())
	play, ok, err := s.store.LoadPlay(ctx, p.Date, player)
	if err != nil {
		return nil, err
	}
	if !ok {
		play = store.DailyPlay{Date: p.Date, PlayerID: player, Status: string(game.StatusHunting), StartedAt: s.now().UTC()}
	}
	m := s.match(p)
	if n := replay(m, play.Moves); n != len(play.Moves) {
		slog.Warn("daily replay stopped early", "date", p.Date, "player", player, "applied", n, "stored", len(play.Moves))
	}
	return &session{puzzle: p, match: m, play: play}, nil
}

func (s *Service) save(ctx context.Context, ss *session) error {
	m := ss.match
	ss.play.Moves = m.Log(0)
	ss.play.Status = string(m.Status(0))
	ss.play.Score = m.Score(0)
	if m.Done(0) && ss.play.FinishedAt == nil {
		t := s.now().UTC()
		ss.play.FinishedAt = &t
	}
	return s.store.SavePlay(ctx, ss.play)
}

func (s *Service) view(ctx context.Context, ss *session, withPool bool, last *game.Move) (View, error) {
	m := ss.match
	v := View{
		Date: ss.puzzle.Date, Number: ss.puzzle.Number, Next: ss.puzzle.Next.UTC(),
		PackID: m.Pack.ID, PackTitle: m.Pack.Title, PackNoun: m.Pack.Noun, Category: m.Pack.Category, Status: m.Status(0), Moves: m.Moves(0),
		Log: m.Log(0), WrongCost: m.Rules.WrongGuessCost, Last: last,
	}
	if v.Log == nil {
		v.Log = []game.Move{}
	}
	for _, c := range m.Remaining(0) {
		v.RemainingIDs = append(v.RemainingIDs, c.ID)
	}
	if withPool {
		pv := room.PoolViewOf(m, ss.puzzle.Number)
		v.Pool = &pv
	}
	if m.Done(0) {
		stats, err := s.store.DayStats(ctx, ss.puzzle.Date)
		if err != nil {
			return v, err
		}
		res := &Result{
			Secret: room.CardOf(m.Pack, m.Secret(1), s.cfg.ImageBase), Score: m.Score(0),
			Grid: m.ShareGrid(0), CrackedBy: m.CrackedBy(0), Stats: stats,
		}
		if m.Status(0) == game.StatusSolved && stats.Solved > 1 {
			worse := 0
			for score, n := range stats.Scores {
				if score > res.Score {
					worse += n
				}
			}
			res.BetterThan = 100 * worse / (stats.Solved - 1)
		}
		v.Result = res
	}
	st, err := s.streak(ctx, ss.play.PlayerID, ss.puzzle.Date)
	if err != nil {
		return v, err
	}
	v.Streak = st
	return v, nil
}

func (s *Service) streak(ctx context.Context, player, today string) (Streak, error) {
	hist, err := s.store.PlayHistory(ctx, player, 400)
	if err != nil {
		return Streak{}, err
	}
	solved := map[string]bool{}
	var st Streak
	var dates []string
	for _, p := range hist {
		if p.Status == string(game.StatusHunting) {
			continue
		}
		st.Played++
		if p.Status == string(game.StatusSolved) {
			st.Solved++
			solved[p.Date] = true
			dates = append(dates, p.Date)
		}
	}
	sort.Strings(dates)
	run := 0
	var prev time.Time
	for _, d := range dates {
		t, _ := time.Parse("2006-01-02", d)
		if run > 0 && t.Sub(prev) == 24*time.Hour {
			run++
		} else {
			run = 1
		}
		prev = t
		st.Best = max(st.Best, run)
	}
	day, _ := time.Parse("2006-01-02", today)
	start := day
	if !solved[today] {
		start = day.AddDate(0, 0, -1) // today not solved yet: streak can still continue
	}
	for d := start; solved[d.Format("2006-01-02")]; d = d.AddDate(0, 0, -1) {
		st.Current++
	}
	return st, nil
}

// Get returns today's puzzle and the player's progress.
func (s *Service) Get(ctx context.Context, player string) (View, error) {
	ss, err := s.load(ctx, player)
	if err != nil {
		return View{}, err
	}
	return s.view(ctx, ss, true, nil)
}

// Action is one daily move. Exactly one of the fields is set.
type Action struct {
	Predicate *game.Predicate `json:"predicate,omitempty"`
	Text      string          `json:"text,omitempty"`
	GuessID   string          `json:"candidate_id,omitempty"`
	GiveUp    bool            `json:"give_up,omitempty"`
}

// Play applies an action and returns the new view.
func (s *Service) Play(ctx context.Context, player string, a Action) (View, error) {
	date := s.PuzzleAt(s.now()).Date
	unlock := s.lock(date, player)
	defer unlock()
	ss, err := s.load(ctx, player)
	if err != nil {
		return View{}, err
	}
	now := s.now()
	var mv game.Move
	switch {
	case a.Predicate != nil:
		mv, err = ss.match.Ask(0, *a.Predicate, now)
	case a.Text != "":
		mv, _, err = ss.match.AskText(0, a.Text, false, now)
	case a.GuessID != "":
		mv, err = ss.match.Guess(0, a.GuessID, now)
	case a.GiveUp:
		mv, err = ss.match.GiveUp(0, now)
	default:
		err = game.Errf(game.CodeBadRequest, "nothing to do")
	}
	if err != nil {
		return View{}, err
	}
	if err := s.save(ctx, ss); err != nil {
		return View{}, err
	}
	return s.view(ctx, ss, false, &mv)
}

// Match previews which question a typed text means (no move is made).
func (s *Service) Match(text string) room.MatchView {
	p := s.PuzzleAt(s.now()).Pack
	if q, ok := p.MatchStrict(text); ok {
		pr := q.Predicate()
		return room.MatchView{Text: q.Text(), Predicate: &pr, Exact: true}
	}
	if q, ok := p.MatchLoose(text); ok {
		pr := q.Predicate()
		return room.MatchView{Text: q.Text(), Predicate: &pr}
	}
	return room.MatchView{}
}
