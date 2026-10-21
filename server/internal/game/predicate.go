package game

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
)

// Answer is the result of evaluating a predicate, or of a guess.
type Answer string

const (
	Yes     Answer = "yes"
	No      Answer = "no"
	Unknown Answer = "unknown"
	Correct Answer = "correct"
	Wrong   Answer = "wrong"
)

// Predicate is a question as sent by a client.
type Predicate struct {
	Attr  string          `json:"attr"`
	Op    string          `json:"op"`
	Value json.RawMessage `json:"value"`
}

// Question is a validated predicate bound to its attribute.
type Question struct {
	Attr *Attribute
	Op   string
	Str  string
	Bool bool
	N1   int
	N2   int
}

var opsByType = map[AttrType][]string{
	TypeEnum:   {"eq"},
	TypeBool:   {"is"},
	TypeInt:    {"eq", "gt", "lt", "between"},
	TypeSet:    {"contains"},
	TypeString: {"starts_with"},
}

// OpsFor lists the operators allowed for an attribute type.
func OpsFor(t AttrType) []string { return opsByType[t] }

func opAllowed(t AttrType, op string) bool {
	for _, o := range opsByType[t] {
		if o == op {
			return true
		}
	}
	return false
}

// Compile validates a predicate against a pack.
func (p *Pack) Compile(pr Predicate) (Question, error) {
	if reservedKeys[strings.ToLower(pr.Attr)] {
		return Question{}, Errf(CodeForbiddenAttr, "questions cannot name a candidate; make a guess instead")
	}
	a, ok := p.attrIndex[pr.Attr]
	if !ok {
		return Question{}, Errf(CodeInvalidPredicate, "unknown attribute %q", pr.Attr)
	}
	if !opAllowed(a.Type, pr.Op) {
		return Question{}, Errf(CodeInvalidPredicate, "operator %q not allowed for %s", pr.Op, a.Label)
	}
	q := Question{Attr: a, Op: pr.Op}
	bad := func(format string, args ...any) (Question, error) {
		return Question{}, Errf(CodeInvalidPredicate, format, args...)
	}
	switch a.Type {
	case TypeEnum, TypeSet:
		if err := json.Unmarshal(pr.Value, &q.Str); err != nil {
			return bad("value must be a string")
		}
		if !contains(a.Values, q.Str) {
			return bad("%q is not a valid value for %s", q.Str, a.Label)
		}
	case TypeBool:
		if err := json.Unmarshal(pr.Value, &q.Bool); err != nil {
			return bad("value must be true or false")
		}
	case TypeInt:
		if pr.Op == "between" {
			var pair []int
			if err := json.Unmarshal(pr.Value, &pair); err != nil || len(pair) != 2 {
				return bad("between needs [low, high]")
			}
			q.N1, q.N2 = pair[0], pair[1]
			if q.N1 > q.N2 {
				return bad("low must not exceed high")
			}
			if q.N1 < *a.Min || q.N2 > *a.Max {
				return bad("range outside [%d, %d]", *a.Min, *a.Max)
			}
		} else {
			if err := json.Unmarshal(pr.Value, &q.N1); err != nil {
				return bad("value must be an integer")
			}
			if q.N1 < *a.Min || q.N1 > *a.Max {
				return bad("value outside [%d, %d]", *a.Min, *a.Max)
			}
		}
		if a.Step > 0 && ((q.N1-*a.Min)%a.Step != 0 || (pr.Op == "between" && (q.N2-*a.Min)%a.Step != 0)) {
			return bad("value must be in steps of %d", a.Step)
		}
	case TypeString:
		if err := json.Unmarshal(pr.Value, &q.Str); err != nil {
			return bad("value must be a string")
		}
		q.Str = strings.TrimSpace(q.Str)
		if n := len([]rune(q.Str)); n < 1 || n > 3 {
			return bad("prefix must be 1 to 3 characters")
		}
		for _, r := range q.Str {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
				return bad("prefix may only contain letters or digits")
			}
		}
	}
	return q, nil
}

// Canonical is a stable key used to detect repeated questions.
func (q Question) Canonical() string {
	switch q.Attr.Type {
	case TypeBool:
		return fmt.Sprintf("%s|%s|%t", q.Attr.Key, q.Op, q.Bool)
	case TypeInt:
		if q.Op == "between" {
			return fmt.Sprintf("%s|between|%d|%d", q.Attr.Key, q.N1, q.N2)
		}
		return fmt.Sprintf("%s|%s|%d", q.Attr.Key, q.Op, q.N1)
	case TypeString:
		return fmt.Sprintf("%s|%s|%s", q.Attr.Key, q.Op, strings.ToLower(q.Str))
	default:
		return fmt.Sprintf("%s|%s|%s", q.Attr.Key, q.Op, q.Str)
	}
}

// Predicate converts a compiled question back to wire form.
func (q Question) Predicate() Predicate {
	var v any
	switch q.Attr.Type {
	case TypeBool:
		v = q.Bool
	case TypeInt:
		if q.Op == "between" {
			v = []int{q.N1, q.N2}
		} else {
			v = q.N1
		}
	default:
		v = q.Str
	}
	raw, _ := json.Marshal(v)
	return Predicate{Attr: q.Attr.Key, Op: q.Op, Value: raw}
}

// Eval answers a question about a candidate. Unknown means the candidate
// has no data for the attribute.
func Eval(c *Candidate, q Question) Answer {
	v, ok := c.vals[q.Attr.Key]
	if !ok {
		return Unknown
	}
	var hit bool
	switch q.Op {
	case "eq":
		if q.Attr.Type == TypeInt {
			hit = v.n == q.N1
		} else {
			hit = v.str == q.Str
		}
	case "is":
		hit = v.b == q.Bool
	case "gt":
		hit = v.n > q.N1
	case "lt":
		hit = v.n < q.N1
	case "between":
		hit = v.n >= q.N1 && v.n <= q.N2
	case "contains":
		hit = v.set[q.Str]
	case "starts_with":
		hit = strings.HasPrefix(alnum(v.str), alnum(q.Str))
	}
	if hit {
		return Yes
	}
	return No
}

var defaultTemplates = map[string]string{
	"eq_enum":     "Is their {label_lc} {value}?",
	"eq_int":      "Is their {label_lc} exactly {value}?",
	"is":          "{label}?",
	"is_false":    "Not {label_lc}?",
	"gt":          "Is their {label_lc} more than {value}?",
	"lt":          "Is their {label_lc} less than {value}?",
	"between":     "Is their {label_lc} between {value} and {value2}?",
	"contains":    "Does their {label_lc} include {value}?",
	"starts_with": "Does their {label_lc} start with “{value}”?",
}

// Text renders a question as a readable sentence using pack templates.
func (q Question) Text() string {
	key := q.Op
	if q.Op == "is" && !q.Bool {
		key = "is_false"
	}
	tpl, ok := q.Attr.Templates[key+":"+q.Str]
	if !ok || q.Str == "" {
		tpl, ok = q.Attr.Templates[key]
	}
	if !ok {
		switch {
		case q.Op == "eq" && q.Attr.Type == TypeInt:
			tpl = defaultTemplates["eq_int"]
		case q.Op == "eq":
			tpl = defaultTemplates["eq_enum"]
		default:
			tpl = defaultTemplates[key]
		}
	}
	val, val2 := q.Str, ""
	switch q.Attr.Type {
	case TypeInt:
		val, val2 = fmt.Sprint(q.N1), fmt.Sprint(q.N2)
	case TypeString:
		val = strings.ToUpper(q.Str)
	}
	r := strings.NewReplacer(
		"{label}", q.Attr.Label,
		"{label_lc}", strings.ToLower(q.Attr.Label),
		"{value}", val,
		"{value_lc}", strings.ToLower(val),
		"{value2}", val2,
	)
	return r.Replace(tpl)
}

// alnum lower-cases s and drops everything but letters and digits.
func alnum(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Filter keeps the candidates still consistent with an answered question.
// An Unknown answer eliminates nothing, and a candidate with no data for the
// attribute is never eliminated.
func Filter(cands []*Candidate, q Question, ans Answer) []*Candidate {
	if ans != Yes && ans != No {
		return cands
	}
	out := cands[:0:0]
	for _, c := range cands {
		if e := Eval(c, q); e == ans || e == Unknown {
			out = append(out, c)
		}
	}
	return out
}
