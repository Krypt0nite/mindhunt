package store

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"mindhunt/internal/game"
)

// Runs only when MINDHUNT_TEST_DATABASE_URL points at a scratch database.
func TestPostgresStore(t *testing.T) {
	url := os.Getenv("MINDHUNT_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("MINDHUNT_TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	p, err := OpenPostgres(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.db.Exec(`TRUNCATE players, rooms, matches CASCADE`); err != nil {
		t.Fatal(err)
	}

	g, err := p.Create(ctx, "Asha")
	if err != nil {
		t.Fatal(err)
	}
	var stored string
	p.db.QueryRow(`SELECT token_hash FROM players WHERE id = $1`, g.ID).Scan(&stored)
	if stored == g.Token || stored != hashToken(g.Token) {
		t.Fatal("token must be stored hashed")
	}
	// A fresh store (simulating a restart) still finds the guest.
	p2 := &Postgres{db: p.db, cache: NewMemoryGuests()}
	got, err := p2.ByToken(ctx, g.Token)
	if err != nil || got.ID != g.ID || got.Name != "Asha" {
		t.Fatalf("by token after restart: %+v %v", got, err)
	}
	if _, err := p2.ByToken(ctx, "missing"); err != ErrNotFound {
		t.Fatalf("missing token: %v", err)
	}
	if err := p.Rename(ctx, g.ID, "Asha K"); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC().Truncate(time.Millisecond)
	p.RoomCreated("ABC234", g.ID, false, now)
	p.MatchStarted(MatchStart{MatchID: "m_test", Room: "ABC234", PackID: "cricket-men-intl", PackVersion: 1,
		Seed: ^uint64(0), PoolSize: 48, PlayerIDs: [2]string{g.ID, ""}, Secrets: [2]string{"c_aaaaaa", "c_bbbbbb"}, StartedAt: now})
	raw, _ := json.Marshal("India")
	p.MoveRecorded("m_test", 0, game.Move{N: 1, Kind: game.KindAsk, Text: "Have they represented India?",
		Predicate: &game.Predicate{Attr: "country", Op: "eq", Value: raw}, Answer: game.Yes, Cost: 1, Total: 1, At: now})
	p.MoveRecorded("m_test", 0, game.Move{N: 2, Kind: game.KindGuess, GuessID: "c_bbbbbb", Text: "Is it X?", Answer: game.Correct, Cost: 1, Total: 2, At: now})
	p.MatchEnded(MatchEnd{MatchID: "m_test", Status: "finished", Winner: 0, Reason: "fewer_moves",
		Moves: [2]int{2, 7}, Scores: [2]int{2, 7}, ActiveMs: [2]int64{1000, 2000}, EndedAt: now})
	p.Close(5 * time.Second)

	db, _ := OpenPostgres(ctx, url)
	defer db.Close(time.Second)
	var status, reason, seed string
	var winner, nMoves, score int
	db.db.QueryRow(`SELECT status, reason, winner_seat, seed::text FROM matches WHERE id = 'm_test'`).Scan(&status, &reason, &winner, &seed)
	db.db.QueryRow(`SELECT count(*) FROM moves WHERE match_id = 'm_test'`).Scan(&nMoves)
	db.db.QueryRow(`SELECT score FROM match_players WHERE match_id = 'm_test' AND seat = 1`).Scan(&score)
	if status != "finished" || reason != "fewer_moves" || winner != 0 || nMoves != 2 || score != 7 || seed != "18446744073709551615" {
		t.Fatalf("persisted: status=%s reason=%s winner=%d moves=%d score=%d seed=%s", status, reason, winner, nMoves, score, seed)
	}
	var botPlayer *string
	db.db.QueryRow(`SELECT player_id FROM match_players WHERE match_id = 'm_test' AND seat = 1`).Scan(&botPlayer)
	if botPlayer != nil {
		t.Fatal("bot seat should have NULL player_id")
	}
}

func TestPostgresDaily(t *testing.T) {
	url := os.Getenv("MINDHUNT_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("MINDHUNT_TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	p, err := OpenPostgres(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close(time.Second)
	p.db.Exec(`TRUNCATE daily_plays`)
	now := time.Now().UTC().Truncate(time.Millisecond)
	raw, _ := json.Marshal("India")
	play := DailyPlay{Date: "2026-10-18", PlayerID: "p_1", Status: "hunting", StartedAt: now,
		Moves: []game.Move{{N: 1, Kind: game.KindAsk, Predicate: &game.Predicate{Attr: "country", Op: "eq", Value: raw}, Answer: game.Yes, Cost: 1, Total: 1, At: now}}}
	if err := p.SavePlay(ctx, play); err != nil {
		t.Fatal(err)
	}
	play.Status, play.Score, play.FinishedAt = "solved", 4, &now
	if err := p.SavePlay(ctx, play); err != nil { // upsert
		t.Fatal(err)
	}
	p.SavePlay(ctx, DailyPlay{Date: "2026-10-18", PlayerID: "p_2", Status: "gave_up", Score: 58, StartedAt: now, Moves: []game.Move{}})
	p.SavePlay(ctx, DailyPlay{Date: "2026-10-17", PlayerID: "p_1", Status: "solved", Score: 6, StartedAt: now, Moves: []game.Move{}})
	got, ok, err := p.LoadPlay(ctx, "2026-10-18", "p_1")
	if err != nil || !ok || got.Status != "solved" || len(got.Moves) != 1 || got.Moves[0].Predicate.Attr != "country" {
		t.Fatalf("load: %+v %v", got, err)
	}
	st, _ := p.DayStats(ctx, "2026-10-18")
	if st.Finished != 2 || st.Solved != 1 || st.Scores[4] != 1 {
		t.Fatalf("stats: %+v", st)
	}
	hist, _ := p.PlayHistory(ctx, "p_1", 10)
	if len(hist) != 2 || hist[0].Date != "2026-10-18" {
		t.Fatalf("history: %+v", hist)
	}
}
