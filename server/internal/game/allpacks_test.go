package game

import (
	"strings"
	"testing"

	"mindhunt/packs"
)

// Checks every built-in pack, so a new pack can't ship broken.
func TestEveryBuiltinPack(t *testing.T) {
	reg, err := LoadRegistry(packs.FS)
	if err != nil {
		t.Fatal(err)
	}
	if len(reg.IDs()) < 8 {
		t.Fatalf("expected at least 8 packs, have %v", reg.IDs())
	}
	for _, id := range reg.IDs() {
		p := reg.Get2(id)
		t.Run(id, func(t *testing.T) {
			if p.Noun == "" || p.Category == "" {
				t.Error("noun and category are required")
			}
			if len(p.Candidates) < 20 {
				t.Errorf("only %d candidates", len(p.Candidates))
			}
			// Every candidate can be told apart from every other.
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
			// Every attribute is filled for most candidates and actually splits them.
			for _, a := range p.Attributes {
				filled, distinct := 0, map[string]bool{}
				for _, c := range p.Candidates {
					if v := c.AttrString(&a); v != "" {
						filled++
						distinct[v] = true
					}
				}
				if filled*10 < len(p.Candidates)*4 { // some attributes apply to a subset (bowling arm)
					t.Errorf("%s filled for only %d of %d", a.Key, filled, len(p.Candidates))
				}
				if len(distinct) < 2 {
					t.Errorf("%s has a single value for everyone", a.Key)
				}
			}
			// Every question the game renders reads back to itself.
			all := make([]*Candidate, len(p.Candidates))
			for i := range p.Candidates {
				all[i] = &p.Candidates[i]
			}
			for _, q := range AllQuestions(p, all, 3, true) {
				got, ok := p.MatchStrict(q.Text())
				if !ok {
					t.Errorf("%q: not matched", q.Text())
				} else if got.Canonical() != q.Canonical() {
					t.Errorf("%q: matched %s, want %s", q.Text(), got.Canonical(), q.Canonical())
				}
			}
		})
	}
}
