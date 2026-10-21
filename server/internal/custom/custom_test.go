package custom

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"
	"time"

	"mindhunt/internal/bot"
	"mindhunt/internal/game"
	"mindhunt/internal/store"
)

// office is a realistic small pack: 10 colleagues, three kinds of attribute.
func office() Draft {
	d := Draft{
		Title: "Our office", Noun: "colleague", Description: "Guess who from the third floor.",
		Attributes: []DraftAttr{
			{Label: "Team", Kind: KindChoice, Choices: []string{"Sales", "Tech", "Design"}, Question: "Do they work in {value}"},
			{Label: "Wears glasses", Kind: KindYesNo, Question: "Do they wear glasses?"},
			{Label: "Years here", Kind: KindNumber},
		},
	}
	rows := []string{
		"Asha|Sales|yes|3", "Bilal|Tech|no|5", "Chen|Design|Y|1", "Divya|tech|n|8", "Esha|Sales|no|2",
		"Farhan|Design|yes|6", "Gita|Tech|yes|4", "Hari|Sales|no|10", "Ira|Design|no|", "Jai|Tech||7",
	}
	for _, r := range rows {
		p := strings.Split(r, "|")
		d.Items = append(d.Items, DraftItem{Name: p[0], Values: p[1:]})
	}
	return d
}

func meta() Meta { return Meta{ID: "u-abcdefghij", Version: 1, Author: "Asha", Now: time.Unix(0, 0)} }

func TestBuildOfficePack(t *testing.T) {
	p, is, st := Build(office(), meta())
	if p == nil {
		t.Fatalf("not built: %+v", is)
	}
	if is.HasErrors() || st.Items != 10 || st.Unique != 10 {
		t.Fatalf("issues %+v stats %+v", is, st)
	}
	team, _ := p.Attr("a1")
	if team.Type != game.TypeEnum || team.Templates["eq"] != "Do they work in {value}?" {
		t.Fatalf("team attr %+v", team)
	}
	years, _ := p.Attr("a3")
	if *years.Min != 1 || *years.Max != 10 {
		t.Fatalf("number range %d..%d", *years.Min, *years.Max)
	}
	// "tech" was normalised to the declared choice; blanks are Unknown.
	divya := byName(t, p, "Divya")
	if string(divya.Attrs["a1"]) != `"Tech"` {
		t.Fatalf("choice not normalised: %s", divya.Attrs["a1"])
	}
	if _, has := byName(t, p, "Ira").Attrs["a3"]; has {
		t.Fatal("blank number stored")
	}
	q, err := p.Compile(game.Predicate{Attr: "a1", Op: "eq", Value: []byte(`"Sales"`)})
	if err != nil || q.Text() != "Do they work in Sales?" {
		t.Fatalf("question %q %v", q.Text(), err)
	}
	q, _ = p.Compile(game.Predicate{Attr: "letters", Op: "starts_with", Value: []byte(`"Ha"`)})
	if game.Eval(byName(t, p, "Hari"), q) != game.Yes {
		t.Fatal("name letters question broken")
	}
	if p.Category != "Custom" || p.Author != "Asha" {
		t.Fatal("meta not set")
	}
}

func byName(t *testing.T, p *game.Pack, name string) *game.Candidate {
	for i := range p.Candidates {
		if p.Candidates[i].Name == name {
			return &p.Candidates[i]
		}
	}
	t.Fatalf("no %s", name)
	return nil
}

func TestBuildRejects(t *testing.T) {
	cases := map[string]struct {
		edit func(*Draft)
		path string
	}{
		"too few items":     {func(d *Draft) { d.Items = d.Items[:5] }, "items"},
		"no attributes":     {func(d *Draft) { d.Attributes = nil }, "attributes"},
		"short title":       {func(d *Draft) { d.Title = "ab" }, "title"},
		"link in title":     {func(d *Draft) { d.Title = "see www.example.com" }, "title"},
		"email":             {func(d *Draft) { d.Description = "mail me a@b.co" }, "description"},
		"phone":             {func(d *Draft) { d.Items[0].Name = "Call 98765 43210" }, "items.0.name"},
		"abuse":             {func(d *Draft) { d.Items[1].Name = "B1tch" }, "items.1.name"},
		"html":              {func(d *Draft) { d.Items[2].Name = "<script>" }, "items.2.name"},
		"duplicate name":    {func(d *Draft) { d.Items[3].Name = "asha" }, "items.3.name"},
		"bad yes/no":        {func(d *Draft) { d.Items[0].Values[1] = "maybe" }, "items.0.values.1"},
		"unknown choice":    {func(d *Draft) { d.Items[0].Values[0] = "HR" }, "items.0.values.0"},
		"not a number":      {func(d *Draft) { d.Items[0].Values[2] = "three" }, "items.0.values.2"},
		"one choice":        {func(d *Draft) { d.Attributes[0].Choices = []string{"Sales"} }, "attributes.0.choices"},
		"question no slot":  {func(d *Draft) { d.Attributes[0].Question = "Which team?" }, "attributes.0.question"},
		"label reused":      {func(d *Draft) { d.Attributes[1].Label = "team" }, "attributes.1.label"},
		"label is name":     {func(d *Draft) { d.Attributes[1].Label = "Name" }, "attributes.1.label"},
		"bad kind":          {func(d *Draft) { d.Attributes[1].Kind = "colour" }, "attributes.1.kind"},
		"number span":       {func(d *Draft) { d.Items[0].Values[2] = "5000" }, "attributes.2"},
		"too many values":   {func(d *Draft) { d.Items[0].Values = append(d.Items[0].Values, "x") }, "items.0.values"},
		"too many attrs":    {func(d *Draft) { d.Attributes = append(d.Attributes, make([]DraftAttr, 9)...) }, "attributes"},
		"symbols only name": {func(d *Draft) { d.Items[4].Name = "!!!" }, "items.4.name"},
	}
	for name, c := range cases {
		d := office()
		c.edit(&d)
		p, is, _ := Build(d, meta())
		if p != nil {
			t.Errorf("%s: built", name)
			continue
		}
		found := false
		for _, i := range is {
			if i.Severity == "error" && i.Path == c.path {
				found = true
			}
		}
		if !found {
			t.Errorf("%s: no error at %s: %+v", name, c.path, is)
		}
	}
}

func TestBuildWarnings(t *testing.T) {
	d := office()
	d.Attributes = append(d.Attributes, DraftAttr{Label: "Works here", Kind: KindYesNo})
	for i := range d.Items {
		d.Items[i].Values = append(d.Items[i].Values, "yes")
	}
	d.Items[1].Values = []string{"Sales", "yes", "3", "yes"} // same as Asha
	p, is, st := Build(d, meta())
	if p == nil {
		t.Fatalf("warnings must not block: %+v", is)
	}
	var msgs []string
	for _, i := range is {
		msgs = append(msgs, i.Severity+": "+i.Message)
	}
	all := strings.Join(msgs, "\n")
	if !strings.Contains(all, "same works here") || !strings.Contains(all, "Asha, Bilal look the same") || st.Unique != 8 {
		t.Fatalf("warnings:\n%s\nstats %+v", all, st)
	}
}

func TestBlockedWordsAvoidNames(t *testing.T) {
	for _, ok := range []string{"Dick Cheney", "Lund University", "James Randi", "Scunthorpe United", "Penistone", "Essex"} {
		if w := blocked(ok); w != "" {
			t.Errorf("%q blocked for %q", ok, w)
		}
	}
	for _, bad := range []string{"f u c k i n g idiot", "bh0sdike", "SHIT"} {
		if blocked(bad) == "" {
			t.Errorf("%q not blocked", bad)
		}
	}
}

func TestBotSolvesACustomPack(t *testing.T) {
	d := Draft{Title: "Fruit", Noun: "fruit", Attributes: []DraftAttr{
		{Label: "Colour", Kind: KindChoice, Choices: []string{"Red", "Yellow", "Green", "Orange", "Purple"}},
		{Label: "Grows on trees", Kind: KindYesNo},
		{Label: "Has a stone", Kind: KindYesNo},
		{Label: "Size", Kind: KindChoice, Choices: []string{"Small", "Medium", "Large"}},
	}}
	for _, r := range []string{
		"Apple|Red|y|n|Medium", "Banana|Yellow|n|n|Medium", "Cherry|Red|y|y|Small", "Grape|Purple|n|n|Small",
		"Lemon|Yellow|y|n|Medium", "Lime|Green|y|n|Small", "Mango|Orange|y|y|Large", "Orange|Orange|y|n|Medium",
		"Peach|Orange|y|y|Medium", "Pear|Green|y|n|Medium", "Plum|Purple|y|y|Small", "Strawberry|Red|n|n|Small",
		"Watermelon|Green|n|n|Large", "Kiwi|Green|n|n|Small", "Coconut|Green|y|n|Large", "Papaya|Orange|y|n|Large",
	} {
		p := strings.Split(r, "|")
		d.Items = append(d.Items, DraftItem{Name: p[0], Values: p[1:]})
	}
	p, is, _ := Build(d, meta())
	if p == nil {
		t.Fatal(is)
	}
	total := 0
	rng := rand.New(rand.NewPCG(1, 2))
	for i, secret := range p.Candidates {
		m := game.NewMatch(p, 0, uint64(i+1), game.DefaultRules)
		m.PickSecret(0, m.Pool[0].ID)
		m.PickSecret(1, secret.ID)
		m.Start()
		for steps := 0; m.Status(0) == game.StatusHunting; steps++ {
			if steps > 40 {
				t.Fatalf("bot stuck on %s", secret.Name)
			}
			a := bot.Next(m, 0, rng)
			var err error
			if a.Ask != nil {
				_, err = m.Ask(0, *a.Ask, time.Time{})
			} else {
				_, err = m.Guess(0, a.Guess, time.Time{})
			}
			if err != nil {
				t.Fatalf("illegal bot move: %v", err)
			}
		}
		total += m.Moves(0)
	}
	if avg := float64(total) / float64(len(p.Candidates)); avg > 7 {
		t.Fatalf("bot average %.1f moves", avg)
	}
}

func TestLibraryLifecycle(t *testing.T) {
	ctx := context.Background()
	st := store.NewMemoryCustomPacks()
	lib := NewLibrary(st, Config{HideAfterReports: 2, MaxPerOwner: 2, PerHour: 4})
	asha := store.Guest{ID: "p_asha", Name: "Asha"}

	if _, err := lib.Create(ctx, asha, Draft{Title: "x"}); !errors.Is(err, ErrDraftError) {
		t.Fatalf("invalid draft saved: %v", err)
	}
	sv, err := lib.Create(ctx, asha, office())
	if err != nil {
		t.Fatal(err)
	}
	id := sv.Pack.ID
	if !IsCustomID(id) {
		t.Fatalf("id %q", id)
	}
	p, ok := lib.Cached(id)
	if !ok {
		t.Fatal("not cached after create")
	}
	hariID := byName(t, p, "Hari").ID

	// Someone else can load but not edit or delete it.
	if _, _, err := lib.Load(ctx, id, "p_bilal"); err != nil {
		t.Fatal(err)
	}
	if _, err := lib.Update(ctx, store.Guest{ID: "p_bilal"}, id, office()); !errors.Is(err, ErrNotOwner) {
		t.Fatalf("stranger edited: %v", err)
	}
	if err := lib.Delete(ctx, "p_bilal", id); !errors.Is(err, ErrNotOwner) {
		t.Fatalf("stranger deleted: %v", err)
	}

	// Editing bumps the version and keeps ids for unchanged names.
	d := office()
	d.Items[0].Name = "Asha K"
	sv, err = lib.Update(ctx, asha, id, d)
	if err != nil {
		t.Fatal(err)
	}
	p2, _ := lib.Cached(id)
	if sv.Pack.Version != 2 || p2.Version != 2 || byName(t, p2, "Hari").ID != hariID {
		t.Fatalf("version %d, Hari %s vs %s", p2.Version, byName(t, p2, "Hari").ID, hariID)
	}
	if p.Version != 1 {
		t.Fatal("old pack pointer mutated; running matches must keep their version")
	}

	// Limits: 2 packs per owner here; 4 saves per hour (3 used).
	if _, err := lib.Create(ctx, asha, office()); err != nil {
		t.Fatal(err)
	}
	if _, err := lib.Create(ctx, asha, office()); !errors.Is(err, ErrRateLimit) && !errors.Is(err, ErrTooMany) {
		t.Fatalf("limit not enforced: %v", err)
	}

	// Reports: owner can't report; two distinct reporters hide it.
	if _, err := lib.Report(ctx, "p_asha", id, "mine"); !errors.Is(err, ErrNotOwner) {
		t.Fatal("owner reported own pack")
	}
	if hidden, _ := lib.Report(ctx, "p_r1", id, "mean"); hidden {
		t.Fatal("hidden after one report")
	}
	lib.Report(ctx, "p_r1", id, "again") // same reporter counts once
	if hidden, _ := lib.Report(ctx, "p_r1", id, "again"); hidden {
		t.Fatal("same reporter counted twice")
	}
	if hidden, _ := lib.Report(ctx, "p_r2", id, "mean"); !hidden {
		t.Fatal("not hidden after two reporters")
	}
	if _, ok := lib.Cached(id); ok {
		t.Fatal("hidden pack still offered to rooms")
	}
	if _, _, err := lib.Load(ctx, id, "p_bilal"); !errors.Is(err, ErrNotFound) {
		t.Fatal("hidden pack visible to others")
	}
	if _, cp, err := lib.Load(ctx, id, "p_asha"); err != nil || !cp.Hidden {
		t.Fatal("owner should still see the hidden pack")
	}
	reported, reports, _ := lib.Reported(ctx)
	if len(reported) != 1 || len(reports) != 2 {
		t.Fatalf("reported %d packs %d reports", len(reported), len(reports))
	}
	if err := lib.Moderate(ctx, id, false); err != nil {
		t.Fatal(err)
	}
	if _, _, err := lib.Load(ctx, id, "p_bilal"); err != nil {
		t.Fatal("restored pack not visible")
	}

	// Owner deletes: gone for everyone.
	if err := lib.Delete(ctx, "p_asha", id); err != nil {
		t.Fatal(err)
	}
	if _, _, err := lib.Load(ctx, id, "p_asha"); !errors.Is(err, ErrNotFound) {
		t.Fatal("deleted pack loads")
	}
	mine, _ := lib.Mine(ctx, "p_asha")
	if len(mine) != 1 {
		t.Fatalf("mine = %d", len(mine))
	}
}

func TestCacheEvictsLeastRecentlyUsed(t *testing.T) {
	lib := NewLibrary(store.NewMemoryCustomPacks(), Config{CacheSize: 2})
	clock := time.Unix(0, 0)
	lib.now = func() time.Time { clock = clock.Add(time.Second); return clock }
	for i := range 3 {
		m := meta()
		m.ID = fmt.Sprintf("u-aaaaaaaaa%d", i+2)
		p, _, _ := Build(office(), m)
		lib.put(p)
		if i == 1 {
			lib.Cached("u-aaaaaaaaa2") // touch the first
		}
	}
	if _, ok := lib.Cached("u-aaaaaaaaa3"); ok {
		t.Fatal("least recently used pack kept")
	}
	if _, ok := lib.Cached("u-aaaaaaaaa2"); !ok {
		t.Fatal("recently used pack evicted")
	}
}
