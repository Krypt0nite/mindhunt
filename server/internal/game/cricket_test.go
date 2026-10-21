package game

import (
	"encoding/json"
	"testing"
)

// These pin the verified cricket data (see tools/cricket_evidence.py and the
// Audit sheet of mindhunt-cricket-players-verified.xlsx).

func TestCricketRoster(t *testing.T) {
	p := cricket(t)
	if len(p.Candidates) != 72 {
		t.Fatalf("want 72 players, have %d", len(p.Candidates))
	}
	ids := map[string]bool{}
	for _, c := range p.Candidates {
		if ids[c.ID] {
			t.Errorf("duplicate id %s", c.ID)
		}
		ids[c.ID] = true
	}
	// IDs survive the spelling fix and the corrections.
	for id, name := range map[string]string{
		"c_44c51f": "Muttiah Muralitharan", "c_12ae0c": "Don Bradman", "c_dbbd1b": "Clive Lloyd",
	} {
		c, ok := p.Candidate(id)
		if !ok || c.Name != name {
			t.Errorf("%s: want %s, got %+v", id, name, c)
		}
	}
	if p.Candidates[0].Name != "Sachin Tendulkar" || p.Candidates[71].Name != "Rashid Khan" {
		t.Error("roster order changed")
	}
}

func TestCricketODIDebutDecade(t *testing.T) {
	p := cricket(t)
	a, _ := p.Attr("debut_decade")
	if a == nil || a.Label != "ODI debut decade" || a.Min == nil || *a.Min != 1970 {
		t.Fatalf("debut attribute not ODI-based: %+v", a)
	}
	for _, c := range p.Candidates {
		raw, has := c.Attrs["debut_decade"]
		if c.Name == "Don Bradman" {
			if has {
				t.Error("Bradman never played an ODI; debut decade must be N/A")
			}
			continue
		}
		var n int
		if !has || json.Unmarshal(raw, &n) != nil || n < 1970 || n > 2020 || n%10 != 0 {
			t.Errorf("%s: bad ODI debut decade %s", c.Name, raw)
		}
	}
	if got := decade(t, p, "c_dbbd1b"); got != 1970 { // Lloyd: Test 1966, first ODI 1973
		t.Errorf("Clive Lloyd ODI decade %d", got)
	}
	// Questions about Bradman's ODI debut answer Unknown and never eliminate him.
	q, err := p.Compile(Predicate{Attr: "debut_decade", Op: "lt", Value: json.RawMessage("1990")})
	if err != nil {
		t.Fatal(err)
	}
	if ans := Eval(cand(t, p, "c_12ae0c"), q); ans != Unknown {
		t.Errorf("Bradman ODI question answered %v", ans)
	}
}

func decade(t *testing.T, p *Pack, id string) int {
	var n int
	if err := json.Unmarshal(cand(t, p, id).Attrs["debut_decade"], &n); err != nil {
		t.Fatal(err)
	}
	return n
}

func cand(t *testing.T, p *Pack, id string) *Candidate {
	c, ok := p.Candidate(id)
	if !ok {
		t.Fatalf("no candidate %s", id)
	}
	return c
}

func TestImageValidation(t *testing.T) {
	ok := Image{Asset: "cricket-men-intl/c_44c51f.webp", License: "CC BY-SA 4.0", Credit: "A. Photographer",
		SourceURL: "https://commons.wikimedia.org/wiki/File:X.jpg"}
	if err := ok.validate(); err != nil {
		t.Fatal(err)
	}
	for name, bad := range map[string]Image{
		"traversal":  {Asset: "../etc/passwd", License: "CC0", Credit: "x", SourceURL: "https://x"},
		"absolute":   {Asset: "/cricket/c_1.webp", License: "CC0", Credit: "x", SourceURL: "https://x"},
		"remote":     {Asset: "https://upload.wikimedia.org/x.jpg", License: "CC0", Credit: "x", SourceURL: "https://x"},
		"no licence": {Asset: "p/c_1.webp", Credit: "x", SourceURL: "https://x"},
		"no credit":  {Asset: "p/c_1.webp", License: "CC0", SourceURL: "https://x"},
		"no source":  {Asset: "p/c_1.webp", License: "CC0", Credit: "x"},
	} {
		if bad.validate() == nil {
			t.Errorf("%s: accepted %+v", name, bad)
		}
	}
}

func TestDropImagesFallsBackToInitials(t *testing.T) {
	p := cricket(t)
	c := cand(t, p, "c_44c51f")
	saved := c.Image
	defer func() { c.Image = saved }()
	c.Image = &Image{Asset: "cricket-men-intl/c_44c51f.webp", License: "CC0", Credit: "x", SourceURL: "https://x"}
	if n := p.DropImages(func(string) bool { return false }); n < 1 || c.Image != nil {
		t.Fatalf("missing asset kept (dropped %d)", n)
	}
}
