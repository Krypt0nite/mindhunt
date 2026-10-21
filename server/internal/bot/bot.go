// Package bot is the practice opponent: it asks the question that minimises
// the expected number of remaining candidates, and guesses only when one
// candidate is left (or nothing else can narrow the field).
package bot

import (
	"math/rand/v2"

	"mindhunt/internal/game"
)

// Action is what the bot does next: exactly one field is set.
type Action struct {
	Ask   *game.Predicate
	Guess string
}

// Next picks the bot's next action for seat.
func Next(m *game.Match, seat int, rng *rand.Rand) Action {
	rem := m.Remaining(seat)
	if len(rem) == 0 { // cannot happen with consistent data; fall back to the pool
		rem = m.Pool
	}
	if len(rem) == 1 {
		return Action{Guess: rem[0].ID}
	}
	asked := map[string]bool{}
	for _, mv := range m.Log(seat) {
		if mv.Predicate != nil {
			if q, err := m.Pack.Compile(*mv.Predicate); err == nil {
				asked[q.Canonical()] = true
			}
		}
	}

	var best []game.Question
	bestE := float64(len(rem))
	for _, q := range questions(m.Pack, rem) {
		if asked[q.Canonical()] {
			continue
		}
		e := expectedRemaining(rem, q)
		switch {
		case e < bestE-1e-9:
			bestE, best = e, []game.Question{q}
		case e < bestE+1e-9 && e < float64(len(rem)):
			best = append(best, q)
		}
	}
	if len(best) == 0 {
		return Action{Guess: rem[rng.IntN(len(rem))].ID}
	}
	p := best[rng.IntN(len(best))].Predicate()
	return Action{Ask: &p}
}

func expectedRemaining(rem []*game.Candidate, q game.Question) float64 {
	var yes, no, unk float64
	for _, c := range rem {
		switch game.Eval(c, q) {
		case game.Yes:
			yes++
		case game.No:
			no++
		default:
			unk++
		}
	}
	if yes == 0 || no == 0 {
		return float64(len(rem)) // no information
	}
	n := float64(len(rem))
	return (yes*(yes+unk) + no*(no+unk) + unk*n) / n
}

// questions lists every askable question worth considering for rem.
func questions(p *game.Pack, rem []*game.Candidate) []game.Question {
	return game.AllQuestions(p, rem, 2, false)
}
