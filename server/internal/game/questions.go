package game

import "encoding/json"

// AllQuestions lists every askable question for a pack. String prefixes are
// drawn from cands (1 to maxPrefix letters). withNegations adds "is false"
// questions for booleans.
func AllQuestions(p *Pack, cands []*Candidate, maxPrefix int, withNegations bool) []Question {
	var out []Question
	add := func(attr, op string, v any) {
		raw, _ := json.Marshal(v)
		if q, err := p.Compile(Predicate{Attr: attr, Op: op, Value: raw}); err == nil {
			out = append(out, q)
		}
	}
	for _, a := range p.Attributes {
		switch a.Type {
		case TypeEnum:
			for _, v := range a.Values {
				add(a.Key, "eq", v)
			}
		case TypeSet:
			for _, v := range a.Values {
				add(a.Key, "contains", v)
			}
		case TypeBool:
			add(a.Key, "is", true)
			if withNegations {
				add(a.Key, "is", false)
			}
		case TypeInt:
			step := a.Step
			if step <= 0 {
				step = 1
			}
			if (*a.Max-*a.Min)/step > 200 {
				step = (*a.Max - *a.Min) / 200
			}
			for v := *a.Min; v <= *a.Max; v += step {
				add(a.Key, "eq", v)
				add(a.Key, "gt", v)
				add(a.Key, "lt", v)
			}
		case TypeString:
			seen := map[string]bool{}
			for _, c := range cands {
				s := alnum(c.AttrString(&a))
				for n := 1; n <= maxPrefix && n <= len([]rune(s)); n++ {
					pre := string([]rune(s)[:n])
					if !seen[pre] {
						seen[pre] = true
						add(a.Key, "starts_with", pre)
					}
				}
			}
		}
	}
	return out
}
