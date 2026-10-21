package game

import (
	"strings"
	"time"
)

// SeatStatus is a player's progress in a round.
type SeatStatus string

const (
	StatusPicking SeatStatus = "picking"
	StatusPicked  SeatStatus = "picked"
	StatusHunting SeatStatus = "hunting"
	StatusSolved  SeatStatus = "solved"
	StatusGaveUp  SeatStatus = "gave_up"
)

// Move kinds.
const (
	KindAsk     = "ask"
	KindGuess   = "guess"
	KindGiveUp  = "give_up"
	KindTimeout = "timeout"
	KindAskFree = "ask_free" // typed question answered by the opponent
)

// Rules are the tunable costs of a round.
type Rules struct {
	WrongGuessCost int
	GiveUpPenalty  int
	// UnknownCap is how many manual "Unknown" answers each player may give.
	UnknownCap int
	// ContradictionLimit is how many audited wrong answers void a win.
	ContradictionLimit int
}

// DefaultRules match the spec.
var DefaultRules = Rules{WrongGuessCost: 3, GiveUpPenalty: 10, UnknownCap: 3, ContradictionLimit: 2}

// Pending is a typed question waiting for the opponent's manual answer.
type Pending struct {
	ID   int       `json:"id"`
	Text string    `json:"text"`
	At   time.Time `json:"at"`
}

// Flag is a manual answer that the reveal-time audit found contradicts the data.
type Flag struct {
	N        int    `json:"n"`
	Text     string `json:"text"`
	Answered Answer `json:"answered"`
	Data     Answer `json:"data"`
	ReadAs   string `json:"read_as"`
}

// Move is one entry in a player's log.
type Move struct {
	N         int        `json:"n"`
	Kind      string     `json:"kind"`
	Predicate *Predicate `json:"predicate,omitempty"`
	Text      string     `json:"text"`
	GuessID   string     `json:"guess_id,omitempty"`
	Answer    Answer     `json:"answer,omitempty"`
	Cost      int        `json:"cost"`
	Total     int        `json:"total"`
	At        time.Time  `json:"at"`
	Removed   int        `json:"removed,omitempty"`
	Manual    bool       `json:"manual,omitempty"` // answered by the opponent, not the data
	Asked     string     `json:"asked,omitempty"`  // the player's own wording, if typed

	q *Question
}

type seatState struct {
	secret  *Candidate
	status  SeatStatus
	moves   int
	log     []Move
	asked   map[string]bool
	guessed map[string]bool
	pending *Pending
	// unknownsGiven counts this seat's manual Unknown answers.
	unknownsGiven int
}

// Match is one round between two seats. Seat i hunts the secret of seat 1-i.
type Match struct {
	Pack  *Pack
	Pool  []*Candidate
	Seed  uint64
	Rules Rules

	inPool    map[string]bool
	seats     [2]seatState
	pendingID int
}

// NewMatch samples a pool and prepares a round.
func NewMatch(p *Pack, poolSize int, seed uint64, rules Rules) *Match {
	m := &Match{Pack: p, Seed: seed, Rules: rules, Pool: p.SamplePool(poolSize, seed)}
	m.inPool = make(map[string]bool, len(m.Pool))
	for _, c := range m.Pool {
		m.inPool[c.ID] = true
	}
	for i := range m.seats {
		m.seats[i] = seatState{status: StatusPicking, asked: map[string]bool{}, guessed: map[string]bool{}}
	}
	return m
}

// InPool reports whether id is a candidate of this round.
func (m *Match) InPool(id string) bool { return m.inPool[id] }

// PickSecret sets (or changes) a seat's secret during picking.
func (m *Match) PickSecret(seat int, id string) error {
	s := &m.seats[seat]
	if s.status != StatusPicking && s.status != StatusPicked {
		return Errf(CodeWrongPhase, "secret is locked")
	}
	if !m.inPool[id] {
		return Errf(CodeNotInPool, "that candidate is not in this round")
	}
	s.secret, _ = m.Pack.Candidate(id)
	s.status = StatusPicked
	return nil
}

// BothPicked reports whether both secrets are chosen.
func (m *Match) BothPicked() bool {
	return m.seats[0].secret != nil && m.seats[1].secret != nil
}

// Start locks both secrets and begins the hunt.
func (m *Match) Start() {
	for i := range m.seats {
		m.seats[i].status = StatusHunting
	}
}

// Secret is the candidate seat chose (nil before picking).
func (m *Match) Secret(seat int) *Candidate { return m.seats[seat].secret }

// Status of a seat.
func (m *Match) Status(seat int) SeatStatus { return m.seats[seat].status }

// Moves used by a seat so far.
func (m *Match) Moves(seat int) int { return m.seats[seat].moves }

// Log returns a copy of a seat's moves.
func (m *Match) Log(seat int) []Move {
	return append([]Move(nil), m.seats[seat].log...)
}

// Done reports whether a seat has finished hunting.
func (m *Match) Done(seat int) bool {
	st := m.seats[seat].status
	return st == StatusSolved || st == StatusGaveUp
}

// BothDone reports whether the round can be scored.
func (m *Match) BothDone() bool { return m.Done(0) && m.Done(1) }

func (m *Match) hunting(seat int) error {
	switch m.seats[seat].status {
	case StatusHunting:
		return nil
	case StatusSolved, StatusGaveUp:
		return Errf(CodeAlreadyDone, "you have already finished this round")
	default:
		return Errf(CodeWrongPhase, "the hunt has not started")
	}
}

func (m *Match) record(seat int, mv Move) Move {
	s := &m.seats[seat]
	s.moves += mv.Cost
	mv.N = len(s.log) + 1
	mv.Total = s.moves
	s.log = append(s.log, mv)
	return mv
}

// Ask evaluates a question about the opponent's secret.
func (m *Match) Ask(seat int, pr Predicate, at time.Time) (Move, error) {
	if err := m.hunting(seat); err != nil {
		return Move{}, err
	}
	q, err := m.Pack.Compile(pr)
	if err != nil {
		return Move{}, err
	}
	return m.askQuestion(seat, q, "", at)
}

func (m *Match) askQuestion(seat int, q Question, asked string, at time.Time) (Move, error) {
	k := q.Canonical()
	if m.seats[seat].asked[k] {
		return Move{}, Errf(CodeDuplicateQuestion, "you already asked that")
	}
	m.seats[seat].asked[k] = true
	ans := Eval(m.seats[1-seat].secret, q)
	p := q.Predicate()
	return m.record(seat, Move{Kind: KindAsk, Predicate: &p, Text: q.Text(), Asked: asked, Answer: ans, Cost: 1, At: at, q: &q}), nil
}

// AskText handles a typed or spoken question. When its words match a known
// question it is answered from the data straight away. Otherwise, if
// allowManual, it waits for the opponent to answer by hand.
func (m *Match) AskText(seat int, text string, allowManual bool, at time.Time) (Move, *Pending, error) {
	if err := m.hunting(seat); err != nil {
		return Move{}, nil, err
	}
	text = strings.Join(strings.Fields(text), " ")
	if n := len([]rune(text)); n < 3 || n > 140 {
		return Move{}, nil, Errf(CodeInvalidText, "questions must be 3 to 140 characters")
	}
	if q, ok := m.Pack.MatchStrict(text); ok {
		mv, err := m.askQuestion(seat, q, text, at)
		return mv, nil, err
	}
	if !allowManual {
		return Move{}, nil, Errf(CodeNoMatch, "that doesn't match a question the game can answer; use the question builder")
	}
	s := &m.seats[seat]
	if s.pending != nil {
		return Move{}, nil, Errf(CodePendingQuestion, "wait for your last question to be answered, or cancel it")
	}
	k := "free|" + key(normalize(text))
	if s.asked[k] {
		return Move{}, nil, Errf(CodeDuplicateQuestion, "you already asked that")
	}
	m.pendingID++
	s.pending = &Pending{ID: m.pendingID, Text: text, At: at}
	return Move{}, s.pending, nil
}

// AnswerText records the answerer's manual answer to the opponent's
// pending question. The move (and its cost) goes to the asker.
func (m *Match) AnswerText(answerer, id int, ans Answer, at time.Time) (Move, error) {
	asker := 1 - answerer
	p := m.seats[asker].pending
	if p == nil || p.ID != id {
		return Move{}, Errf(CodeNoPending, "that question is no longer waiting for an answer")
	}
	if ans != Yes && ans != No && ans != Unknown {
		return Move{}, Errf(CodeBadRequest, "answer must be yes, no or unknown")
	}
	if ans == Unknown {
		if m.seats[answerer].unknownsGiven >= m.Rules.UnknownCap {
			return Move{}, Errf(CodeUnknownCap, "you have used all %d Unknown answers; answer Yes or No", m.Rules.UnknownCap)
		}
		m.seats[answerer].unknownsGiven++
	}
	m.seats[asker].pending = nil
	m.seats[asker].asked["free|"+key(normalize(p.Text))] = true
	return m.record(asker, Move{Kind: KindAskFree, Text: p.Text, Answer: ans, Cost: 1, Manual: true, At: at}), nil
}

// CancelText withdraws a seat's pending question at no cost.
func (m *Match) CancelText(seat int) error {
	if m.seats[seat].pending == nil {
		return Errf(CodeNoPending, "no question is waiting")
	}
	m.seats[seat].pending = nil
	return nil
}

// Pending is the seat's own question waiting for an answer.
func (m *Match) Pending(seat int) *Pending { return m.seats[seat].pending }

// Incoming is the opponent's question this seat must answer.
func (m *Match) Incoming(seat int) *Pending { return m.seats[1-seat].pending }

// UnknownsLeft is how many manual Unknown answers the seat may still give.
func (m *Match) UnknownsLeft(seat int) int {
	return max(0, m.Rules.UnknownCap-m.seats[seat].unknownsGiven)
}

// Audit checks the manual answers the answerer gave against the data for
// their own secret. Only questions MatchLoose can read are checked, and only
// a Yes/No answer opposite to the data counts.
func (m *Match) Audit(answerer int) []Flag {
	var out []Flag
	secret := m.seats[answerer].secret
	for _, mv := range m.seats[1-answerer].log {
		if !mv.Manual || (mv.Answer != Yes && mv.Answer != No) {
			continue
		}
		q, ok := m.Pack.MatchLoose(mv.Text)
		if !ok {
			continue
		}
		if truth := Eval(secret, q); (truth == Yes || truth == No) && truth != mv.Answer {
			out = append(out, Flag{N: mv.N, Text: mv.Text, Answered: mv.Answer, Data: truth, ReadAs: q.Text()})
		}
	}
	return out
}

// Guess names a candidate. A wrong guess costs Rules.WrongGuessCost moves.
func (m *Match) Guess(seat int, id string, at time.Time) (Move, error) {
	if err := m.hunting(seat); err != nil {
		return Move{}, err
	}
	if !m.inPool[id] {
		return Move{}, Errf(CodeNotInPool, "that candidate is not in this round")
	}
	s := &m.seats[seat]
	if s.guessed[id] {
		return Move{}, Errf(CodeDuplicateGuess, "you already guessed that")
	}
	s.guessed[id] = true
	c, _ := m.Pack.Candidate(id)
	mv := Move{Kind: KindGuess, GuessID: id, Text: "Is it " + c.Name + "?", At: at}
	if id == m.seats[1-seat].secret.ID {
		mv.Answer, mv.Cost = Correct, 1
		s.status = StatusSolved
		s.pending = nil
	} else {
		mv.Answer, mv.Cost = Wrong, m.Rules.WrongGuessCost
	}
	return m.record(seat, mv), nil
}

// GiveUp ends a seat's hunt with the give-up score.
func (m *Match) GiveUp(seat int, at time.Time) (Move, error) {
	if err := m.hunting(seat); err != nil {
		return Move{}, err
	}
	m.seats[seat].status = StatusGaveUp
	m.seats[seat].pending = nil
	return m.record(seat, Move{Kind: KindGiveUp, Text: "Gave up", At: at}), nil
}

// Timeout charges one move when the optional turn timer expires.
func (m *Match) Timeout(seat int, at time.Time) (Move, error) {
	if err := m.hunting(seat); err != nil {
		return Move{}, err
	}
	return m.record(seat, Move{Kind: KindTimeout, Text: "Timer ran out", Cost: 1, At: at}), nil
}

// Score is the number used to decide the round (lower is better).
func (m *Match) Score(seat int) int {
	s := m.seats[seat]
	if s.status == StatusGaveUp {
		return max(len(m.Pool), s.moves) + m.Rules.GiveUpPenalty
	}
	return s.moves
}

// Result outcome reasons.
const (
	ReasonFewerMoves = "fewer_moves"
	ReasonFaster     = "faster"
	ReasonTie        = "tie"
	ReasonBothGaveUp = "both_gave_up"
	ReasonAbandoned  = "abandoned"
	// ReasonContradictions: the would-be winner gave too many manual answers
	// that contradict the data.
	ReasonContradictions = "contradictions"
)

// Result is the outcome of a round. Winner is -1 for a draw.
type Result struct {
	Winner int    `json:"winner"`
	Reason string `json:"reason"`
	Scores [2]int `json:"scores"`
}

// Decide scores a finished round. activeMs is each seat's connected hunting time.
func (m *Match) Decide(activeMs [2]int64) Result {
	r := Result{Winner: -1, Scores: [2]int{m.Score(0), m.Score(1)}}
	switch {
	case m.seats[0].status == StatusGaveUp && m.seats[1].status == StatusGaveUp:
		r.Reason = ReasonBothGaveUp
	case r.Scores[0] != r.Scores[1]:
		r.Reason = ReasonFewerMoves
		if r.Scores[0] < r.Scores[1] {
			r.Winner = 0
		} else {
			r.Winner = 1
		}
	default:
		d := activeMs[0] - activeMs[1]
		switch {
		case d < -1000:
			r.Winner, r.Reason = 0, ReasonFaster
		case d > 1000:
			r.Winner, r.Reason = 1, ReasonFaster
		default:
			r.Reason = ReasonTie
		}
	}
	if lim := m.Rules.ContradictionLimit; lim > 0 {
		bad := [2]bool{len(m.Audit(0)) >= lim, len(m.Audit(1)) >= lim}
		switch {
		case bad[0] && bad[1]:
			r.Winner, r.Reason = -1, ReasonContradictions
		case r.Winner >= 0 && bad[r.Winner]:
			r.Winner, r.Reason = 1-r.Winner, ReasonContradictions
		case r.Winner == -1 && (bad[0] || bad[1]):
			if bad[0] {
				r.Winner = 1
			} else {
				r.Winner = 0
			}
			r.Reason = ReasonContradictions
		}
	}
	return r
}

// Forfeit scores a round won by abandonment.
func (m *Match) Forfeit(winner int) Result {
	return Result{Winner: winner, Reason: ReasonAbandoned, Scores: [2]int{m.Score(0), m.Score(1)}}
}

// Remaining replays a seat's log over the pool and returns the candidates
// still consistent with every answer.
func (m *Match) Remaining(seat int) []*Candidate {
	rem, _ := m.replay(seat)
	return rem
}

// CrackedBy returns the question that removed the most candidates, or nil.
func (m *Match) CrackedBy(seat int) *Move {
	_, best := m.replay(seat)
	return best
}

func (m *Match) replay(seat int) ([]*Candidate, *Move) {
	rem := append([]*Candidate(nil), m.Pool...)
	var best *Move
	for _, mv := range m.seats[seat].log {
		before := len(rem)
		switch {
		case mv.Kind == KindAsk && mv.q != nil:
			rem = Filter(rem, *mv.q, mv.Answer)
		case mv.Kind == KindGuess && mv.Answer == Wrong:
			rem = without(rem, mv.GuessID)
		case mv.Kind == KindGuess && mv.Answer == Correct:
			rem = only(rem, mv.GuessID)
		}
		if mv.Kind == KindAsk {
			removed := before - len(rem)
			if removed > 0 && (best == nil || removed > best.Removed) {
				cp := mv
				cp.Removed = removed
				best = &cp
			}
		}
	}
	return rem, best
}

func without(cs []*Candidate, id string) []*Candidate {
	out := cs[:0:0]
	for _, c := range cs {
		if c.ID != id {
			out = append(out, c)
		}
	}
	return out
}

func only(cs []*Candidate, id string) []*Candidate {
	for _, c := range cs {
		if c.ID == id {
			return []*Candidate{c}
		}
	}
	return nil
}

// ShareGrid renders a spoiler-free emoji line for a seat's log.
func (m *Match) ShareGrid(seat int) string {
	var b strings.Builder
	for _, mv := range m.seats[seat].log {
		switch {
		case (mv.Kind == KindAsk || mv.Kind == KindAskFree) && mv.Answer == Yes:
			b.WriteString("🟩")
		case (mv.Kind == KindAsk || mv.Kind == KindAskFree) && mv.Answer == No:
			b.WriteString("🟥")
		case mv.Kind == KindAsk || mv.Kind == KindAskFree:
			b.WriteString("⬜")
		case mv.Kind == KindGuess && mv.Answer == Correct:
			b.WriteString("✅")
		case mv.Kind == KindGuess:
			b.WriteString("❌")
		case mv.Kind == KindTimeout:
			b.WriteString("⏱️")
		case mv.Kind == KindGiveUp:
			b.WriteString("🏳️")
		}
	}
	return b.String()
}
