package game

import (
	"testing"
	"time"

	"mindhunt/packs"
)

func cricket(t *testing.T) *Pack {
	t.Helper()
	reg, err := LoadRegistry(packs.FS)
	if err != nil {
		t.Fatal(err)
	}
	p, _ := reg.Get("cricket-men-intl")
	return p
}

func TestMatchStrictCanonicalTexts(t *testing.T) {
	p := cricket(t)
	all := make([]*Candidate, len(p.Candidates))
	for i := range p.Candidates {
		all[i] = &p.Candidates[i]
	}
	// Every question the game can render is matched back to itself.
	for _, q := range AllQuestions(p, all, 3, true) {
		got, ok := p.MatchStrict(q.Text())
		if !ok {
			t.Errorf("%q: no match, want %s", q.Text(), q.Canonical())
		} else if got.Canonical() != q.Canonical() {
			t.Errorf("%q: got %s, want %s", q.Text(), got.Canonical(), q.Canonical())
		}
	}
}

func TestMatchStrictParaphrases(t *testing.T) {
	p := cricket(t)
	cases := map[string]string{
		"did they represent India":                   "country|eq|India",
		"have they represented india ???":            "country|eq|India",
		"Have they captained their country?":         "captained_country|is|true",
		"has he captained the country":               "captained_country|is|true",
		"did he make his odi debut before the 1990s": "debut_decade|lt|1990",
		"Do they bat left handed":                    "batting_hand|eq|Left",
		"does their surname start with K":            "surname|starts_with|k",
		"have they never captained the country":      "captained_country|is|false",
		"Does his surname begin with an?":            "surname|starts_with|an",
		"surname starts with 'A'":                    "surname|starts_with|a",
		"did he make his odi debut after 1990":       "debut_decade|gt|1990",
	}
	for text, want := range cases {
		q, ok := p.MatchStrict(text)
		if !ok {
			t.Errorf("%q: no match, want %s", text, want)
		} else if q.Canonical() != want {
			t.Errorf("%q: got %s, want %s", text, q.Canonical(), want)
		}
	}
	for _, text := range []string{"is he left handed", "is he a famous player", "did he play IPL", "is he a spinner from india"} {
		if q, ok := p.MatchStrict(text); ok {
			t.Errorf("%q should not match strictly, got %s", text, q.Canonical())
		}
	}
}

func TestMatchLoose(t *testing.T) {
	p := cricket(t)
	ok := map[string]string{
		"is he left handed":            "batting_hand|eq|Left",
		"is he a lefty?":               "batting_hand|eq|Left",
		"is he a spinner":              "bowling|eq|Spin",
		"is he a fast bowler":          "bowling|eq|Pace",
		"is he a keeper":               "role|eq|Wicketkeeper",
		"is he Indian":                 "country|eq|India",
		"does he play for the windies": "country|eq|West Indies",
		"is he an all rounder":         "role|eq|All-rounder",
		"does he bowl left arm":        "bowling_arm|eq|Left",
		"debuted in the 1990s":         "debut_decade|eq|1990",
	}
	for text, want := range ok {
		q, matched := p.MatchLoose(text)
		if !matched {
			t.Errorf("%q: no match, want %s", text, want)
		} else if q.Canonical() != want {
			t.Errorf("%q: got %s, want %s", text, q.Canonical(), want)
		}
	}
	// Ambiguous, negated or comparative questions are never read.
	for _, text := range []string{
		"is he a left arm spinner",   // two facts
		"is he not Indian",           // negation
		"isn't he from India",        // negation via n't
		"debuted later than 1990",    // comparator
		"is he a spinner from India", // two facts
		"is he left",                 // batting or bowling arm?
		"was he good at fielding",    // nothing known
		"did he play in 1990",        // number without the attribute word
	} {
		if q, matched := p.MatchLoose(text); matched {
			t.Errorf("%q should not match, got %s", text, q.Canonical())
		}
	}
}

func freeMatch(t *testing.T, secretA, secretB string) *Match {
	t.Helper()
	p := cricket(t)
	m := NewMatch(p, 0, 1, DefaultRules)
	var a, b string
	for _, c := range p.Candidates {
		if c.Name == secretA {
			a = c.ID
		}
		if c.Name == secretB {
			b = c.ID
		}
	}
	if err := m.PickSecret(0, a); err != nil {
		t.Fatal(err)
	}
	if err := m.PickSecret(1, b); err != nil {
		t.Fatal(err)
	}
	m.Start()
	return m
}

func TestAskTextAutoAnswersKnownQuestions(t *testing.T) {
	m := freeMatch(t, "Virat Kohli", "Shane Warne")
	mv, pend, err := m.AskText(0, "did they represent Australia?", false, time.Now())
	if err != nil || pend != nil || mv.Answer != Yes || mv.Kind != KindAsk || mv.Asked == "" || mv.Manual {
		t.Fatalf("auto: %+v %+v %v", mv, pend, err)
	}
	// Builder and typed versions of the same question are the same question.
	if _, err := m.Ask(0, pred("country", "eq", "Australia"), time.Now()); code(err) != CodeDuplicateQuestion {
		t.Fatalf("duplicate across builder/text: %v", err)
	}
	// Unmatched text with free text off is refused at no cost.
	if _, _, err := m.AskText(0, "is he blond", false, time.Now()); code(err) != CodeNoMatch {
		t.Fatalf("no match: %v", err)
	}
	if m.Moves(0) != 1 {
		t.Fatalf("moves %d", m.Moves(0))
	}
}

func TestManualAnswersCapAndCancel(t *testing.T) {
	m := freeMatch(t, "Virat Kohli", "Shane Warne")
	now := time.Now()
	_, pend, err := m.AskText(0, "Is he blond?", true, now)
	if err != nil || pend == nil {
		t.Fatalf("pending: %v", err)
	}
	if m.Incoming(1) == nil || m.Incoming(1).Text != "Is he blond?" || m.Moves(0) != 0 {
		t.Fatal("incoming not visible or charged early")
	}
	if _, _, err := m.AskText(0, "Is he tall?", true, now); code(err) != CodePendingQuestion {
		t.Fatalf("second pending: %v", err)
	}
	if _, err := m.AnswerText(1, pend.ID+1, Yes, now); code(err) != CodeNoPending {
		t.Fatalf("wrong id: %v", err)
	}
	mv, err := m.AnswerText(1, pend.ID, Yes, now)
	if err != nil || !mv.Manual || mv.Kind != KindAskFree || m.Moves(0) != 1 {
		t.Fatalf("answer: %+v %v", mv, err)
	}
	if _, _, err := m.AskText(0, "is he   BLOND", true, now); code(err) != CodeDuplicateQuestion {
		t.Fatalf("repeat free text: %v", err)
	}
	// Unknown is capped at 3 per answerer.
	for i, q := range []string{"Is he tall?", "Does he sing?", "Likes coffee?", "Owns a dog?"} {
		_, p, _ := m.AskText(0, q, true, now)
		_, err := m.AnswerText(1, p.ID, Unknown, now)
		if i < 3 && err != nil {
			t.Fatalf("unknown %d: %v", i, err)
		}
		if i == 3 {
			if code(err) != CodeUnknownCap {
				t.Fatalf("cap not enforced: %v", err)
			}
			if m.UnknownsLeft(1) != 0 {
				t.Fatal("unknowns left")
			}
			if err := m.CancelText(0); err != nil || m.Pending(0) != nil {
				t.Fatalf("cancel: %v", err)
			}
		}
	}
	// Solving clears any pending question.
	m.AskText(0, "Is he funny?", true, now)
	m.Guess(0, m.Secret(1).ID, now)
	if m.Pending(0) != nil || m.Incoming(1) != nil {
		t.Fatal("pending survived solve")
	}
}

func TestAuditAndContradictionRule(t *testing.T) {
	// Seat 1 hides Shane Warne (Australia, spinner, right-handed).
	m := freeMatch(t, "Virat Kohli", "Shane Warne")
	now := time.Now()
	answer := func(text string, a Answer) {
		_, p, err := m.AskText(0, text, true, now)
		if err != nil || p == nil {
			t.Fatalf("%q: %v", text, err)
		}
		if _, err := m.AnswerText(1, p.ID, a, now); err != nil {
			t.Fatal(err)
		}
	}
	answer("is he a spinner", No)    // lie (loose match: Spin)
	answer("is he Australian", Yes)  // honest
	answer("is he a lefty", Yes)     // lie (Warne bats right)
	answer("does he like chess", No) // unreadable, ignored
	flags := m.Audit(1)
	if len(flags) != 2 || flags[0].Data != Yes || flags[1].ReadAs != "Do they bat left-handed?" {
		t.Fatalf("flags: %+v", flags)
	}
	if len(m.Audit(0)) != 0 {
		t.Fatal("seat 0 gave no answers")
	}
	// Seat 1 would win on moves (1 vs 5), but 2 contradictions void it.
	m.Guess(0, m.Secret(1).ID, now)
	m.Guess(1, m.Secret(0).ID, now)
	r := m.Decide([2]int64{})
	if r.Winner != 0 || r.Reason != ReasonContradictions {
		t.Fatalf("decide: %+v", r)
	}
}
