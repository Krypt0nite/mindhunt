// Package custom turns player-made pack drafts into validated game packs,
// with limits and text checks, and keeps the made packs loaded for rooms.
package custom

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"mindhunt/internal/game"
)

// Attribute kinds a player can make.
const (
	KindYesNo  = "yesno"
	KindChoice = "choice"
	KindNumber = "number"
)

// Limits keep packs playable and the server safe.
const (
	MinItems      = 8
	MaxItems      = 150
	MaxAttributes = 10
	MaxChoices    = 12
	MaxNumberSpan = 400 // the question builder lists every value in range
)

// Draft is what the pack builder sends: a table of items and one column per
// attribute, every cell a string ("" = not sure, the game answers Unknown).
type Draft struct {
	Title       string      `json:"title"`
	Noun        string      `json:"noun"`
	Description string      `json:"description"`
	Attributes  []DraftAttr `json:"attributes"`
	Items       []DraftItem `json:"items"`
}

// DraftAttr is one column.
type DraftAttr struct {
	Label    string   `json:"label"`
	Kind     string   `json:"kind"`
	Choices  []string `json:"choices,omitempty"`
	Question string   `json:"question,omitempty"` // optional wording; {value} for choices and numbers
}

// DraftItem is one row: a name and a value per attribute, in column order.
type DraftItem struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}

// Issue is a problem found in a draft. Path points at the field
// ("title", "attributes.2.label", "items.5.values.1").
type Issue struct {
	Path     string `json:"path"`
	Message  string `json:"message"`
	Severity string `json:"severity"` // error | warning
}

// Issues collects problems.
type Issues []Issue

func (is *Issues) err(path, f string, a ...any) {
	*is = append(*is, Issue{Path: path, Message: fmt.Sprintf(f, a...), Severity: "error"})
}

func (is *Issues) warn(path, f string, a ...any) {
	*is = append(*is, Issue{Path: path, Message: fmt.Sprintf(f, a...), Severity: "warning"})
}

// HasErrors reports whether any issue blocks saving.
func (is Issues) HasErrors() bool {
	for _, i := range is {
		if i.Severity == "error" {
			return true
		}
	}
	return false
}

// Stats describes how playable a valid draft is.
type Stats struct {
	Items       int     `json:"items"`
	Attributes  int     `json:"attributes"`
	Unique      int     `json:"unique"`        // items that no other item matches on every attribute
	AvgMovesEst float64 `json:"avg_moves_est"` // rough: log2 of items, plus a penalty for look-alikes
}

// Meta identifies a pack being built.
type Meta struct {
	ID      string
	Version int
	Author  string
	// PrevIDs maps lower-cased item names to their candidate ids in the
	// previous version, so editing a pack keeps ids stable.
	PrevIDs map[string]string
	Now     time.Time
}

var (
	urlRe   = regexp.MustCompile(`(?i)(https?://|www\.|\b[a-z0-9-]+\.(com|net|org|in|io|co|xyz|ly|me|gg)\b)`)
	emailRe = regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`)
	phoneRe = regexp.MustCompile(`(\+?\d[\d\s-]{8,}\d)`)
)

// text checks one free-text field: length, characters, links, contact
// details and the word list. It returns the cleaned value.
func text(is *Issues, path, label, v string, min, max int) string {
	v = strings.Join(strings.Fields(v), " ")
	n := utf8.RuneCountInString(v)
	switch {
	case n < min && min == 1:
		is.err(path, "%s is required", label)
		return v
	case n < min:
		is.err(path, "%s needs at least %d characters", label, min)
	case n > max:
		is.err(path, "%s can be at most %d characters", label, max)
	}
	for _, r := range v {
		if unicode.IsControl(r) || r == '<' || r == '>' {
			is.err(path, "%s contains characters that are not allowed", label)
			break
		}
	}
	switch {
	case urlRe.MatchString(v):
		is.err(path, "%s can't contain links", label)
	case emailRe.MatchString(v):
		is.err(path, "%s can't contain email addresses", label)
	case phoneRe.MatchString(v):
		is.err(path, "%s can't contain phone numbers", label)
	}
	if w := blocked(v); w != "" {
		is.err(path, "%s contains a word that isn't allowed", label)
	}
	return v
}

// Check validates a draft without building it.
func Check(d Draft) (Issues, Stats) {
	_, is, st := Build(d, Meta{ID: "u-check00000", Version: 1, Now: time.Now()})
	return is, st
}

// Build validates a draft and turns it into a pack. The pack is nil when
// there are errors.
func Build(d Draft, m Meta) (*game.Pack, Issues, Stats) {
	var is Issues
	title := text(&is, "title", "Title", d.Title, 3, 60)
	noun := text(&is, "noun", "Noun", d.Noun, 0, 30)
	if noun == "" {
		noun = "item"
	}
	desc := text(&is, "description", "Description", d.Description, 0, 200)

	if len(d.Attributes) == 0 {
		is.err("attributes", "add at least one attribute to ask about")
	}
	if len(d.Attributes) > MaxAttributes {
		is.err("attributes", "at most %d attributes", MaxAttributes)
		d.Attributes = d.Attributes[:MaxAttributes]
	}
	if len(d.Items) < MinItems {
		is.err("items", "add at least %d %ss (you have %d)", MinItems, noun, len(d.Items))
	}
	if len(d.Items) > MaxItems {
		is.err("items", "at most %d %ss", MaxItems, noun)
		d.Items = d.Items[:MaxItems]
	}

	attrs := make([]game.Attribute, 0, len(d.Attributes)+1)
	labels := map[string]bool{"name": true}
	for i, a := range d.Attributes {
		p := fmt.Sprintf("attributes.%d", i)
		label := text(&is, p+".label", "Attribute name", a.Label, 2, 40)
		if labels[strings.ToLower(label)] {
			is.err(p+".label", "two attributes are called %q", label)
		}
		labels[strings.ToLower(label)] = true
		q := text(&is, p+".question", "Question", a.Question, 0, 120)
		ga := game.Attribute{Key: fmt.Sprintf("a%d", i+1), Label: label, Templates: map[string]string{}}
		switch a.Kind {
		case KindYesNo:
			ga.Type = game.TypeBool
			if q != "" {
				ga.Templates["is"] = ensureQ(q)
			}
		case KindChoice:
			ga.Type = game.TypeEnum
			seen := map[string]bool{}
			for j, c := range a.Choices {
				c = text(&is, fmt.Sprintf("%s.choices.%d", p, j), "Choice", c, 1, 30)
				if c == "" {
					continue
				}
				if seen[strings.ToLower(c)] {
					is.err(fmt.Sprintf("%s.choices.%d", p, j), "choice %q is listed twice", c)
					continue
				}
				seen[strings.ToLower(c)] = true
				ga.Values = append(ga.Values, c)
			}
			if len(ga.Values) < 2 {
				is.err(p+".choices", "a choice attribute needs at least 2 choices")
			}
			if len(ga.Values) > MaxChoices {
				is.err(p+".choices", "at most %d choices", MaxChoices)
			}
			if q != "" {
				if !strings.Contains(q, "{value}") {
					is.err(p+".question", "the question must contain {value} where the choice goes")
				} else {
					ga.Templates["eq"] = ensureQ(q)
				}
			}
		case KindNumber:
			ga.Type = game.TypeInt
			if q != "" {
				if !strings.Contains(q, "{value}") {
					is.err(p+".question", "the question must contain {value} where the number goes")
				} else {
					// "Is their age {value}?" -> also more/less/between forms
					ga.Templates["eq"] = ensureQ(q)
				}
			}
		default:
			is.err(p+".kind", "kind must be yesno, choice or number")
		}
		if len(ga.Templates) == 0 {
			ga.Templates = nil
		}
		attrs = append(attrs, ga)
	}

	cands := make([]game.Candidate, 0, len(d.Items))
	names := map[string]int{}
	used := map[string]bool{}
	nums := make([][]int, len(attrs))
	for i, it := range d.Items {
		p := fmt.Sprintf("items.%d", i)
		name := text(&is, p+".name", "Name", it.Name, 1, 60)
		if name != "" && !strings.ContainsFunc(name, func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }) {
			is.err(p+".name", "name needs a letter or digit")
		}
		if j, dup := names[strings.ToLower(name)]; dup && name != "" {
			is.err(p+".name", "%q is also row %d", name, j+1)
		}
		names[strings.ToLower(name)] = i
		if len(it.Values) > len(attrs) {
			is.err(p+".values", "more values than attributes")
		}
		vals := map[string]json.RawMessage{}
		for j, a := range attrs {
			raw := ""
			if j < len(it.Values) {
				raw = strings.TrimSpace(it.Values[j])
			}
			if raw == "" {
				continue
			}
			vp := fmt.Sprintf("%s.values.%d", p, j)
			switch a.Type {
			case game.TypeBool:
				b, ok := yesNo(raw)
				if !ok {
					is.err(vp, "%q: use yes, no, or leave blank if unsure", raw)
					continue
				}
				vals[a.Key] = json.RawMessage(strconv.FormatBool(b))
			case game.TypeEnum:
				c := canonical(a.Values, raw)
				if c == "" {
					is.err(vp, "%q is not one of the choices for %s", raw, a.Label)
					continue
				}
				vals[a.Key], _ = json.Marshal(c)
			case game.TypeInt:
				n, err := strconv.Atoi(strings.ReplaceAll(raw, ",", ""))
				if err != nil || n < -1_000_000 || n > 1_000_000 {
					is.err(vp, "%q is not a whole number", raw)
					continue
				}
				vals[a.Key] = json.RawMessage(strconv.Itoa(n))
				nums[j] = append(nums[j], n)
			}
		}
		id := m.PrevIDs[strings.ToLower(name)]
		if id == "" || used[id] {
			id = newCandID(used)
		}
		used[id] = true
		vals["letters"], _ = json.Marshal(name)
		cands = append(cands, game.Candidate{ID: id, Name: name, Attrs: vals})
	}

	for j := range attrs {
		a := &attrs[j]
		p := fmt.Sprintf("attributes.%d", j)
		if a.Type == game.TypeInt {
			if len(nums[j]) == 0 {
				is.err(p, "%s has no numbers filled in", a.Label)
				lo, hi := 0, 0
				a.Min, a.Max = &lo, &hi
				continue
			}
			lo, hi := nums[j][0], nums[j][0]
			for _, n := range nums[j] {
				lo, hi = min(lo, n), max(hi, n)
			}
			if hi-lo > MaxNumberSpan {
				is.err(p, "%s spans %d to %d; keep the range within %d (use bigger units or a choice attribute)", a.Label, lo, hi, MaxNumberSpan)
			}
			a.Min, a.Max, a.Step = &lo, &hi, 1
		}
		filled, distinct := 0, map[string]bool{}
		for _, c := range cands {
			if v, ok := c.Attrs[a.Key]; ok {
				filled++
				distinct[string(v)] = true
			}
		}
		switch {
		case len(cands) == 0:
		case filled > 0 && len(distinct) < 2:
			is.warn(p, "every %s has the same %s, so asking about it tells players nothing", noun, strings.ToLower(a.Label))
		case filled*2 < len(cands):
			is.warn(p, "%s is blank for most %ss; blanks are answered Unknown", a.Label, noun)
		}
	}
	attrs = append(attrs, game.Attribute{Key: "letters", Label: "Name", Type: game.TypeString,
		Help:      "Ask about the first 1 to 3 letters (spaces and hyphens ignored).",
		Templates: map[string]string{"starts_with": "Does the name start with “{value}”?"}})

	st := Stats{Items: len(cands), Attributes: len(d.Attributes)}
	sigs := map[string][]string{}
	for _, c := range cands {
		var parts []string
		for _, a := range attrs[:len(attrs)-1] {
			parts = append(parts, string(c.Attrs[a.Key]))
		}
		k := strings.Join(parts, "|")
		sigs[k] = append(sigs[k], c.Name)
	}
	var keys []string
	for k := range sigs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if g := sigs[k]; len(g) == 1 {
			st.Unique++
		} else if len(d.Attributes) > 0 {
			is.warn("items", "%s look the same on every attribute; players can only tell them apart by name letters", strings.Join(g, ", "))
		}
	}
	if n := len(cands); n > 0 {
		st.AvgMovesEst = log2(float64(n)) + float64(n-st.Unique)/float64(n)*3
	}

	if is.HasErrors() {
		return nil, is, st
	}
	pack := &game.Pack{
		ID: m.ID, Version: m.Version, AsOf: m.Now.Format("2006-01-02"), Category: "Custom",
		Title: title, Noun: noun, Description: desc, Author: m.Author,
		Attributes: attrs, Candidates: cands,
	}
	// The game's own validator is the last word.
	data, err := json.Marshal(pack)
	if err == nil {
		pack, err = game.ParsePack(data)
	}
	if err != nil {
		is.err("", "pack rejected: %v", err)
		return nil, is, st
	}
	return pack, is, st
}

func ensureQ(q string) string {
	if !strings.HasSuffix(q, "?") {
		q += "?"
	}
	return q
}

func yesNo(s string) (bool, bool) {
	switch strings.ToLower(s) {
	case "yes", "y", "true", "1", "haan", "ha":
		return true, true
	case "no", "n", "false", "0", "nahi":
		return false, true
	}
	return false, false
}

func canonical(choices []string, v string) string {
	for _, c := range choices {
		if strings.EqualFold(c, v) {
			return c
		}
	}
	return ""
}

func newCandID(used map[string]bool) string {
	for {
		var b [3]byte
		_, _ = rand.Read(b[:])
		id := "c_" + hex.EncodeToString(b[:])
		if !used[id] {
			return id
		}
	}
}

// NewID returns a fresh id for a player-made pack.
func NewID() string {
	const alphabet = "abcdefghijkmnpqrstuvwxyz23456789"
	var b [10]byte
	_, _ = rand.Read(b[:])
	out := []byte("u-")
	for _, x := range b {
		out = append(out, alphabet[int(x)%len(alphabet)])
	}
	return string(out)
}

// IsCustomID reports whether id names a player-made pack.
func IsCustomID(id string) bool { return customIDRe.MatchString(id) }

var customIDRe = regexp.MustCompile(`^u-[a-z2-9]{10}$`)

func log2(x float64) float64 {
	n := 0.0
	for x > 1 {
		x /= 2
		n++
	}
	return n
}
