package game

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"mindhunt/packs"
)

const testPackJSON = `{
  "id": "test-pack", "version": 1, "as_of": "2026-01-01", "category": "Test",
  "title": "Test", "description": "d", "draft": false,
  "attributes": [
    {"key": "color", "label": "Color", "type": "enum", "values": ["red", "blue", "green"]},
    {"key": "big", "label": "Is big", "type": "bool"},
    {"key": "year", "label": "Year", "type": "int", "min": 1900, "max": 2000, "step": 10},
    {"key": "tags", "label": "Tags", "type": "set", "values": ["a", "b", "c"]},
    {"key": "word", "label": "Word", "type": "string"}
  ],
  "candidates": [
    {"id": "c_000001", "name": "One",   "attrs": {"color": "red",   "big": true,  "year": 1950, "tags": ["a"],      "word": "Apple"}},
    {"id": "c_000002", "name": "Two",   "attrs": {"color": "blue",  "big": false, "year": 1960, "tags": ["a", "b"], "word": "Banana"}},
    {"id": "c_000003", "name": "Three", "attrs": {"color": "green", "big": true,  "year": 1970, "tags": [],         "word": "Cherry"}},
    {"id": "c_000004", "name": "Four",  "attrs": {"color": "red",   "big": false, "year": 1980, "tags": ["c"],      "word": "de Silva"}},
    {"id": "c_000005", "name": "Five",  "attrs": {"color": "blue",  "big": true,  "year": 1990, "tags": ["b", "c"], "word": "Egg"}},
    {"id": "c_000006", "name": "Six",   "attrs": {"color": "green", "big": false, "year": 2000, "tags": ["a"],      "word": "Fig"}},
    {"id": "c_000007", "name": "Seven", "attrs": {"color": "red",   "big": true,  "year": 1900, "tags": ["b"]}},
    {"id": "c_000008", "name": "Eight", "attrs": {"color": "blue"}}
  ]
}`

func testPack(t *testing.T) *Pack {
	t.Helper()
	p, err := ParsePack([]byte(testPackJSON))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func pred(attr, op string, v any) Predicate {
	raw, _ := json.Marshal(v)
	return Predicate{Attr: attr, Op: op, Value: raw}
}

func code(err error) string {
	var ge *Error
	if errors.As(err, &ge) {
		return ge.Code
	}
	if err != nil {
		return "other"
	}
	return ""
}

func TestEval(t *testing.T) {
	p := testPack(t)
	cases := []struct {
		cand string
		pr   Predicate
		want Answer
	}{
		{"c_000001", pred("color", "eq", "red"), Yes},
		{"c_000002", pred("color", "eq", "red"), No},
		{"c_000001", pred("big", "is", true), Yes},
		{"c_000002", pred("big", "is", true), No},
		{"c_000002", pred("big", "is", false), Yes},
		{"c_000001", pred("year", "eq", 1950), Yes},
		{"c_000001", pred("year", "gt", 1950), No},
		{"c_000002", pred("year", "gt", 1950), Yes},
		{"c_000001", pred("year", "lt", 1960), Yes},
		{"c_000003", pred("year", "between", []int{1960, 1980}), Yes},
		{"c_000006", pred("year", "between", []int{1960, 1980}), No},
		{"c_000002", pred("tags", "contains", "b"), Yes},
		{"c_000003", pred("tags", "contains", "a"), No},
		{"c_000001", pred("word", "starts_with", "ap"), Yes},
		{"c_000001", pred("word", "starts_with", "AP"), Yes},
		{"c_000004", pred("word", "starts_with", "des"), Yes}, // spaces ignored
		{"c_000002", pred("word", "starts_with", "a"), No},
		{"c_000007", pred("word", "starts_with", "a"), Unknown}, // missing attr
		{"c_000008", pred("big", "is", true), Unknown},
		{"c_000008", pred("year", "gt", 1900), Unknown},
	}
	for _, tc := range cases {
		q, err := p.Compile(tc.pr)
		if err != nil {
			t.Fatalf("compile %+v: %v", tc.pr, err)
		}
		c, _ := p.Candidate(tc.cand)
		if got := Eval(c, q); got != tc.want {
			t.Errorf("%s %s: got %s want %s", tc.cand, q.Canonical(), got, tc.want)
		}
	}
}

func TestCompileRejects(t *testing.T) {
	p := testPack(t)
	cases := []struct {
		pr   Predicate
		code string
	}{
		{pred("name", "eq", "One"), CodeForbiddenAttr},
		{pred("id", "eq", "c_000001"), CodeForbiddenAttr},
		{pred("NAME", "eq", "One"), CodeForbiddenAttr},
		{pred("nope", "eq", "x"), CodeInvalidPredicate},
		{pred("color", "gt", "red"), CodeInvalidPredicate},
		{pred("color", "eq", "purple"), CodeInvalidPredicate},
		{pred("color", "eq", 3), CodeInvalidPredicate},
		{pred("big", "is", "yes"), CodeInvalidPredicate},
		{pred("year", "gt", 1955), CodeInvalidPredicate}, // off step
		{pred("year", "gt", 2100), CodeInvalidPredicate},
		{pred("year", "between", []int{1980, 1960}), CodeInvalidPredicate},
		{pred("year", "between", []int{1960}), CodeInvalidPredicate},
		{pred("tags", "contains", "z"), CodeInvalidPredicate},
		{pred("word", "starts_with", ""), CodeInvalidPredicate},
		{pred("word", "starts_with", "abcd"), CodeInvalidPredicate},
		{pred("word", "starts_with", "a*"), CodeInvalidPredicate},
	}
	for _, tc := range cases {
		_, err := p.Compile(tc.pr)
		if code(err) != tc.code {
			t.Errorf("%+v (%s): got %v want %s", tc.pr, tc.pr.Value, err, tc.code)
		}
	}
}

func TestParsePackRejects(t *testing.T) {
	bad := map[string]string{
		"slug id":       strings.Replace(testPackJSON, `"c_000001"`, `"c_one"`, 1),
		"dup id":        strings.Replace(testPackJSON, `"c_000002"`, `"c_000001"`, 1),
		"name attr":     strings.Replace(testPackJSON, `"key": "word"`, `"key": "name"`, 1),
		"enum value":    strings.Replace(testPackJSON, `"color": "red",   "big": true,  "year": 1950`, `"color": "pink", "big": true, "year": 1950`, 1),
		"int range":     strings.Replace(testPackJSON, `"year": 1950`, `"year": 2050`, 1),
		"unknown attr":  strings.Replace(testPackJSON, `"word": "Apple"`, `"wurd": "Apple"`, 1),
		"unknown field": strings.Replace(testPackJSON, `"draft": false`, `"draft": false, "extra": 1`, 1),
	}
	for name, js := range bad {
		if _, err := ParsePack([]byte(js)); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestText(t *testing.T) {
	p := testPack(t)
	q, _ := p.Compile(pred("year", "between", []int{1960, 1980}))
	if got := q.Text(); got != "Is their year between 1960 and 1980?" {
		t.Errorf("got %q", got)
	}
	q, _ = p.Compile(pred("word", "starts_with", "ap"))
	if got := q.Text(); !strings.Contains(got, "AP") {
		t.Errorf("got %q", got)
	}
}

func TestFilterUnknownNeverEliminates(t *testing.T) {
	p := testPack(t)
	all := p.SamplePool(0, 1)
	q, _ := p.Compile(pred("big", "is", true))
	got := Filter(all, q, Yes)
	names := map[string]bool{}
	for _, c := range got {
		names[c.Name] = true
	}
	if !names["Eight"] { // no data for "big"
		t.Error("candidate with missing data was eliminated")
	}
	if names["Two"] {
		t.Error("No candidate kept on Yes answer")
	}
	if len(Filter(all, q, Unknown)) != len(all) {
		t.Error("Unknown answer eliminated candidates")
	}
}

func TestSamplePoolDeterministic(t *testing.T) {
	p := testPack(t)
	a, b := p.SamplePool(5, 42), p.SamplePool(5, 42)
	if len(a) != 5 {
		t.Fatalf("len %d", len(a))
	}
	for i := range a {
		if a[i].ID != b[i].ID {
			t.Fatal("pool not deterministic for a seed")
		}
	}
	if len(p.SamplePool(0, 1)) != 8 || len(p.SamplePool(100, 1)) != 8 {
		t.Error("full pool size wrong")
	}
}

func newTestMatch(t *testing.T) *Match {
	m := NewMatch(testPack(t), 0, 7, DefaultRules)
	if err := m.PickSecret(0, "c_000001"); err != nil {
		t.Fatal(err)
	}
	if err := m.PickSecret(1, "c_000002"); err != nil {
		t.Fatal(err)
	}
	m.Start()
	return m
}

func TestMatchFlow(t *testing.T) {
	m := newTestMatch(t)
	now := time.Now()

	// Seat 0 hunts c_000002 (blue, not big).
	mv, err := m.Ask(0, pred("color", "eq", "blue"), now)
	if err != nil || mv.Answer != Yes || mv.Total != 1 {
		t.Fatalf("ask: %+v %v", mv, err)
	}
	if _, err := m.Ask(0, pred("color", "eq", "blue"), now); code(err) != CodeDuplicateQuestion {
		t.Fatalf("duplicate not rejected: %v", err)
	}
	if m.Moves(0) != 1 {
		t.Fatalf("duplicate cost a move: %d", m.Moves(0))
	}
	mv, _ = m.Guess(0, "c_000005", now)
	if mv.Answer != Wrong || m.Moves(0) != 1+DefaultRules.WrongGuessCost {
		t.Fatalf("wrong guess: %+v moves=%d", mv, m.Moves(0))
	}
	if _, err := m.Guess(0, "c_000005", now); code(err) != CodeDuplicateGuess {
		t.Fatalf("duplicate guess: %v", err)
	}
	if _, err := m.Guess(0, "c_999999", now); code(err) != CodeNotInPool {
		t.Fatalf("not in pool: %v", err)
	}
	mv, _ = m.Guess(0, "c_000002", now)
	if mv.Answer != Correct || m.Status(0) != StatusSolved || m.Moves(0) != 5 {
		t.Fatalf("correct guess: %+v moves=%d", mv, m.Moves(0))
	}
	if _, err := m.Ask(0, pred("big", "is", true), now); code(err) != CodeAlreadyDone {
		t.Fatalf("ask after solve: %v", err)
	}
	if m.BothDone() {
		t.Fatal("round done too early")
	}
	// Secrets are locked once the hunt starts.
	if err := m.PickSecret(1, "c_000003"); code(err) != CodeWrongPhase {
		t.Fatalf("secret change allowed: %v", err)
	}
	// Seat 1 solves in 1.
	if mv, _ := m.Guess(1, "c_000001", now); mv.Answer != Correct {
		t.Fatal("seat 1 guess")
	}
	r := m.Decide([2]int64{10000, 20000})
	if r.Winner != 1 || r.Reason != ReasonFewerMoves || r.Scores != [2]int{5, 1} {
		t.Fatalf("decide: %+v", r)
	}
	if g := m.ShareGrid(0); g != "🟩❌✅" {
		t.Errorf("grid %q", g)
	}
}

func TestDecideTiesAndGiveUp(t *testing.T) {
	m := newTestMatch(t)
	now := time.Now()
	m.Guess(0, "c_000002", now)
	m.Guess(1, "c_000001", now)
	if r := m.Decide([2]int64{5000, 5500}); r.Winner != -1 || r.Reason != ReasonTie {
		t.Errorf("tie within 1s: %+v", r)
	}
	if r := m.Decide([2]int64{5000, 9000}); r.Winner != 0 || r.Reason != ReasonFaster {
		t.Errorf("faster: %+v", r)
	}

	m = newTestMatch(t)
	m.GiveUp(0, now)
	m.Guess(1, "c_000001", now)
	if r := m.Decide([2]int64{}); r.Winner != 1 || r.Scores[0] != 8+10 {
		t.Errorf("give up: %+v", r)
	}
	m = newTestMatch(t)
	m.GiveUp(0, now)
	m.GiveUp(1, now)
	if r := m.Decide([2]int64{}); r.Winner != -1 || r.Reason != ReasonBothGaveUp {
		t.Errorf("both gave up: %+v", r)
	}
}

func TestCrackedBy(t *testing.T) {
	m := newTestMatch(t)
	now := time.Now()
	m.Ask(0, pred("tags", "contains", "c"), now) // No: removes Four, Five -> 2
	m.Ask(0, pred("big", "is", false), now)      // Yes: removes One, Three, Seven, (Five gone) -> 3
	best := m.CrackedBy(0)
	if best == nil || best.Predicate.Attr != "big" || best.Removed != 3 {
		t.Fatalf("cracked by: %+v", best)
	}
	rem := m.Remaining(0)
	names := []string{}
	for _, c := range rem {
		names = append(names, c.Name)
	}
	// Eight has no data for tags or big, so it is never eliminated.
	if strings.Join(names, ",") != "Eight,Six,Two" {
		t.Fatalf("remaining %v", names)
	}
}

func TestBuiltinPacks(t *testing.T) {
	reg, err := LoadRegistry(packs.FS)
	if err != nil {
		t.Fatal(err)
	}
	p, ok := reg.Get("cricket-men-intl")
	if !ok {
		t.Fatal("cricket pack missing")
	}
	// Every attribute except bowling_arm (bowlers only) and ODI debut decade
	// (Bradman never played an ODI) is filled for every candidate.
	for _, c := range p.Candidates {
		for _, a := range p.Attributes {
			if a.Key == "bowling_arm" || (a.Key == "debut_decade" && c.Name == "Don Bradman") {
				continue
			}
			if c.AttrString(&a) == "" {
				t.Errorf("%s missing %s", c.Name, a.Key)
			}
		}
		_, arm := c.vals["bowling_arm"]
		bowls := c.vals["bowling"].str != "Not a regular bowler"
		if arm != bowls {
			t.Errorf("%s: bowling_arm presence %v but bowls %v", c.Name, arm, bowls)
		}
	}
	// No two candidates share every attribute, so every secret is identifiable.
	seen := map[string]string{}
	for _, c := range p.Candidates {
		var sig []string
		for _, a := range p.Attributes {
			sig = append(sig, c.AttrString(&a))
		}
		k := strings.Join(sig, "|")
		if other, dup := seen[k]; dup {
			t.Errorf("%s and %s are indistinguishable", c.Name, other)
		}
		seen[k] = c.Name
	}
}
