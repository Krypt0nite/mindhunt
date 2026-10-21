package daily

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"mindhunt/internal/game"
	"mindhunt/internal/store"
	"mindhunt/packs"
)

func newService(t *testing.T, st store.Daily, key string, now *time.Time) *Service {
	t.Helper()
	reg, err := game.LoadRegistry(packs.FS)
	if err != nil {
		t.Fatal(err)
	}
	loc, _ := time.LoadLocation("Asia/Kolkata")
	s, err := New(Config{PackIDs: []string{"cricket-men-intl"}, PoolSize: 48, Key: []byte(key), Location: loc,
		Epoch: "2026-10-18", Rules: game.DefaultRules, ImageBase: "/assets"}, reg, st)
	if err != nil {
		t.Fatal(err)
	}
	s.now = func() time.Time { return *now }
	return s
}

func at(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestPuzzleIsStablePerDayAndKey(t *testing.T) {
	now := at("2026-10-18T10:00:00+05:30")
	a := newService(t, store.NewMemoryDaily(), "k1", &now)
	b := newService(t, store.NewMemoryDaily(), "k1", &now)
	c := newService(t, store.NewMemoryDaily(), "k2", &now)
	pa, pb := a.PuzzleAt(now), b.PuzzleAt(now)
	if pa.Secret != pb.Secret || pa.Seed != pb.Seed || pa.Number != 1 || pa.Date != "2026-10-18" {
		t.Fatalf("not stable: %+v %+v", pa, pb)
	}
	// Different key, different puzzle (with overwhelming probability).
	if pc := c.PuzzleAt(now); pc.Seed == pa.Seed {
		t.Fatal("key does not affect the puzzle")
	}
	// Day boundary is local midnight in IST: 23:59 IST is still day 1.
	if p := a.PuzzleAt(at("2026-10-18T18:29:00Z")); p.Date != "2026-10-18" {
		t.Fatalf("boundary: %s", p.Date)
	}
	if p := a.PuzzleAt(at("2026-10-18T18:31:00Z")); p.Date != "2026-10-19" || p.Number != 2 {
		t.Fatalf("next day: %+v", p)
	}
	// Secrets vary across days.
	seen := map[string]bool{}
	for d := 0; d < 30; d++ {
		seen[a.PuzzleAt(now.AddDate(0, 0, d)).Secret] = true
	}
	if len(seen) < 15 {
		t.Fatalf("only %d distinct secrets in 30 days", len(seen))
	}
}

func pred(attr, op string, v any) *game.Predicate {
	raw, _ := json.Marshal(v)
	return &game.Predicate{Attr: attr, Op: op, Value: raw}
}

func TestPlaySolveRevealAndResume(t *testing.T) {
	ctx := context.Background()
	now := at("2026-10-18T10:00:00+05:30")
	st := store.NewMemoryDaily()
	s := newService(t, st, "k", &now)
	secret := s.PuzzleAt(now).Secret

	v, err := s.Get(ctx, "p1")
	if err != nil || v.Pool == nil || len(v.Pool.Candidates) != 48 || v.Status != game.StatusHunting {
		t.Fatalf("get: %+v %v", v.Status, err)
	}
	raw, _ := json.Marshal(v)
	if strings.Contains(string(raw), `"result"`) || strings.Contains(string(raw), `"image"`) {
		t.Fatal("result or image before finishing")
	}

	v, err = s.Play(ctx, "p1", Action{Predicate: pred("country", "eq", "India")})
	if err != nil || v.Moves != 1 || v.Last == nil || v.Pool != nil {
		t.Fatalf("ask: %+v %v", v, err)
	}
	if _, err := s.Play(ctx, "p1", Action{Predicate: pred("country", "eq", "India")}); err == nil {
		t.Fatal("duplicate allowed")
	}
	v, err = s.Play(ctx, "p1", Action{Text: "did they bat left handed"})
	if err != nil || v.Moves != 2 {
		t.Fatalf("typed: %+v %v", v.Moves, err)
	}
	if _, err := s.Play(ctx, "p1", Action{Text: "is he married"}); err == nil {
		t.Fatal("unmatched text should be refused in the daily")
	}

	// A new service over the same store (a restart) resumes from the log.
	s2 := newService(t, st, "k", &now)
	v, _ = s2.Get(ctx, "p1")
	if v.Moves != 2 || len(v.Log) != 2 {
		t.Fatalf("resume: moves=%d log=%d", v.Moves, len(v.Log))
	}

	v, err = s2.Play(ctx, "p1", Action{GuessID: secret})
	if err != nil || v.Status != game.StatusSolved || v.Result == nil || v.Result.Secret.ID != secret {
		t.Fatalf("solve: %+v %v", v.Status, err)
	}
	if v.Result.Score != 3 || v.Result.Stats.Solved != 1 || v.Streak.Current != 1 {
		t.Fatalf("result: score=%d stats=%+v streak=%+v", v.Result.Score, v.Result.Stats, v.Streak)
	}
	if _, err := s2.Play(ctx, "p1", Action{GiveUp: true}); err == nil {
		t.Fatal("played after finishing")
	}
}

func TestStatsAndStreaks(t *testing.T) {
	ctx := context.Background()
	now := at("2026-10-18T10:00:00+05:30")
	st := store.NewMemoryDaily()
	s := newService(t, st, "k", &now)

	solveIn := func(player string, extra int) View {
		secret := s.PuzzleAt(now).Secret
		for i, v := range []string{"India", "Australia", "England", "Pakistan"}[:extra] {
			_ = i
			s.Play(ctx, player, Action{Predicate: pred("country", "eq", v)})
		}
		v, err := s.Play(ctx, player, Action{GuessID: secret})
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	solveIn("fast", 0)    // 1 move
	solveIn("mid", 2)     // 3 moves
	v := solveIn("me", 1) // 2 moves
	if v.Result.Stats.Solved != 3 || v.Result.BetterThan != 50 {
		t.Fatalf("stats: %+v better=%d", v.Result.Stats, v.Result.BetterThan)
	}
	s.Play(ctx, "quitter", Action{GiveUp: true})
	v, _ = s.Get(ctx, "me")
	if v.Result.Stats.Finished != 4 || v.Result.Stats.Solved != 3 {
		t.Fatalf("finished: %+v", v.Result.Stats)
	}

	// Streak: solve three days in a row, skip one, solve again.
	for _, d := range []int{1, 2} {
		now = at("2026-10-18T10:00:00+05:30").AddDate(0, 0, d)
		solveIn("me", 0)
	}
	now = at("2026-10-18T10:00:00+05:30").AddDate(0, 0, 3)
	v, _ = s.Get(ctx, "me")
	if v.Streak.Current != 3 || v.Streak.Best != 3 {
		t.Fatalf("streak before today's play: %+v", v.Streak)
	}
	now = at("2026-10-18T10:00:00+05:30").AddDate(0, 0, 5) // missed a day
	v = solveIn("me", 0)
	if v.Streak.Current != 1 || v.Streak.Best != 3 || v.Streak.Played != 4 {
		t.Fatalf("streak after gap: %+v", v.Streak)
	}
	if v.Number != 6 {
		t.Fatalf("puzzle number %d", v.Number)
	}
}
