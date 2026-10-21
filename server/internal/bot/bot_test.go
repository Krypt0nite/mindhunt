package bot

import (
	"math/rand/v2"
	"testing"
	"time"

	"mindhunt/internal/game"
	"mindhunt/packs"
)

// TestBotSolvesEfficiently plays 1,000 simulated hunts on the cricket pack
// at Normal difficulty (48 candidates) and checks the average move count.
func TestBotSolvesEfficiently(t *testing.T) {
	reg, err := game.LoadRegistry(packs.FS)
	if err != nil {
		t.Fatal(err)
	}
	pack, _ := reg.Get("cricket-men-intl")
	rng := rand.New(rand.NewPCG(1, 2))
	const games = 1000
	total, worst := 0, 0
	for g := range games {
		m := game.NewMatch(pack, 48, uint64(g), game.DefaultRules)
		m.PickSecret(0, m.Pool[rng.IntN(len(m.Pool))].ID)
		m.PickSecret(1, m.Pool[rng.IntN(len(m.Pool))].ID)
		m.Start()
		for steps := 0; m.Status(0) == game.StatusHunting; steps++ {
			if steps > 60 {
				t.Fatalf("game %d: bot did not finish", g)
			}
			a := Next(m, 0, rng)
			var err error
			if a.Ask != nil {
				_, err = m.Ask(0, *a.Ask, time.Time{})
			} else {
				_, err = m.Guess(0, a.Guess, time.Time{})
			}
			if err != nil {
				t.Fatalf("game %d: bot made an illegal move: %v", g, err)
			}
		}
		total += m.Moves(0)
		worst = max(worst, m.Moves(0))
	}
	avg := float64(total) / games
	t.Logf("average %.2f moves, worst %d, over %d games", avg, worst, games)
	if avg < 4 || avg > 8.5 {
		t.Errorf("average %.2f outside expected 4–8.5", avg)
	}
}

// TestBotOnEveryPack checks every pack is solvable in a sensible number of
// moves, which also catches packs whose attributes barely split candidates.
func TestBotOnEveryPack(t *testing.T) {
	reg, err := game.LoadRegistry(packs.FS)
	if err != nil {
		t.Fatal(err)
	}
	rng := rand.New(rand.NewPCG(3, 4))
	for _, id := range reg.IDs() {
		pack := reg.Get2(id)
		total := 0
		const games = 200
		for g := range games {
			m := game.NewMatch(pack, 48, uint64(g), game.DefaultRules)
			m.PickSecret(0, m.Pool[0].ID)
			m.PickSecret(1, m.Pool[rng.IntN(len(m.Pool))].ID)
			m.Start()
			for steps := 0; m.Status(0) == game.StatusHunting; steps++ {
				if steps > 80 {
					t.Fatalf("%s game %d: bot stuck", id, g)
				}
				a := Next(m, 0, rng)
				var err error
				if a.Ask != nil {
					_, err = m.Ask(0, *a.Ask, time.Time{})
				} else {
					_, err = m.Guess(0, a.Guess, time.Time{})
				}
				if err != nil {
					t.Fatalf("%s: illegal bot move: %v", id, err)
				}
			}
			total += m.Moves(0)
		}
		avg := float64(total) / games
		t.Logf("%-24s %2d candidates  average %.2f moves", id, len(pack.Candidates), avg)
		if avg > 10 {
			t.Errorf("%s: average %.2f moves is too many", id, avg)
		}
	}
}
