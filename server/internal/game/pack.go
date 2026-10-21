// Package game holds the pure rules of MindHunt: topic packs, predicates,
// evaluation and scoring. Nothing in here does I/O or keeps time.
package game

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"regexp"
	"sort"
	"strings"
)

// AttrType is the value type of a pack attribute.
type AttrType string

const (
	TypeEnum   AttrType = "enum"
	TypeBool   AttrType = "bool"
	TypeInt    AttrType = "int"
	TypeSet    AttrType = "set"
	TypeString AttrType = "string"
)

// Attribute describes one askable property of every candidate in a pack.
type Attribute struct {
	Key       string            `json:"key"`
	Label     string            `json:"label"`
	Type      AttrType          `json:"type"`
	Values    []string          `json:"values,omitempty"`
	Min       *int              `json:"min,omitempty"`
	Max       *int              `json:"max,omitempty"`
	Step      int               `json:"step,omitempty"`
	Help      string            `json:"help,omitempty"`
	Templates map[string]string `json:"templates,omitempty"`
	// Aliases are extra words for a value, used only to suggest questions
	// from typed text and to audit manual answers (never to answer).
	Aliases map[string][]string `json:"aliases,omitempty"`
}

// Image is a licensed picture shown only on the results screen.
//
// Asset is a path relative to the server's ASSETS_DIR (for example
// "cricket-men-intl/c_44c51f.webp"); images are served locally so a match
// never depends on a third-party host.
type Image struct {
	Asset      string `json:"asset"`
	License    string `json:"license"`
	LicenseURL string `json:"license_url,omitempty"`
	Credit     string `json:"credit"`
	SourceURL  string `json:"source_url"`
}

// Candidate is one possible secret.
type Candidate struct {
	ID    string                     `json:"id"`
	Name  string                     `json:"name"`
	Attrs map[string]json.RawMessage `json:"attrs"`
	Image *Image                     `json:"image,omitempty"`
	Facts []string                   `json:"facts,omitempty"`

	vals map[string]value
}

// value is a typed, validated attribute value.
type value struct {
	str  string
	b    bool
	n    int
	set  map[string]bool
	kind AttrType
}

// Pack is a versioned topic pack.
type Pack struct {
	ID          string      `json:"id"`
	Version     int         `json:"version"`
	AsOf        string      `json:"as_of"`
	Category    string      `json:"category"`
	Title       string      `json:"title"`
	Noun        string      `json:"noun,omitempty"` // singular, e.g. "country"
	Description string      `json:"description"`
	Draft       bool        `json:"draft"`
	Notes       string      `json:"notes,omitempty"`
	Author      string      `json:"author,omitempty"` // display name, for player-made packs
	Attributes  []Attribute `json:"attributes"`
	Candidates  []Candidate `json:"candidates"`

	attrIndex map[string]*Attribute
	candIndex map[string]*Candidate
}

var (
	packIDRe  = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,62}$`)
	attrKeyRe = regexp.MustCompile(`^[a-z][a-z0-9_]{0,31}$`)
	candIDRe  = regexp.MustCompile(`^c_[0-9a-f]{6}$`)
)

// reservedKeys can never be attributes: asking about them would be a guess.
var reservedKeys = map[string]bool{"id": true, "name": true}

// ParsePack decodes and validates a pack.
func ParsePack(data []byte) (*Pack, error) {
	var p Pack
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil {
		return nil, fmt.Errorf("decode pack: %w", err)
	}
	if err := p.validate(); err != nil {
		return nil, fmt.Errorf("pack %q: %w", p.ID, err)
	}
	return &p, nil
}

func (p *Pack) validate() error {
	if !packIDRe.MatchString(p.ID) {
		return fmt.Errorf("invalid id")
	}
	if p.Version < 1 {
		return fmt.Errorf("version must be >= 1")
	}
	if p.Title == "" {
		return fmt.Errorf("title required")
	}
	if len(p.Attributes) == 0 {
		return fmt.Errorf("no attributes")
	}
	if len(p.Candidates) < 8 {
		return fmt.Errorf("need at least 8 candidates, have %d", len(p.Candidates))
	}
	p.attrIndex = make(map[string]*Attribute, len(p.Attributes))
	for i := range p.Attributes {
		a := &p.Attributes[i]
		if !attrKeyRe.MatchString(a.Key) || reservedKeys[a.Key] {
			return fmt.Errorf("invalid attribute key %q", a.Key)
		}
		if _, dup := p.attrIndex[a.Key]; dup {
			return fmt.Errorf("duplicate attribute %q", a.Key)
		}
		if a.Label == "" {
			return fmt.Errorf("attribute %q: label required", a.Key)
		}
		switch a.Type {
		case TypeEnum, TypeSet:
			if len(a.Values) < 2 {
				return fmt.Errorf("attribute %q: needs at least 2 values", a.Key)
			}
			seen := map[string]bool{}
			for _, v := range a.Values {
				if v == "" || seen[v] {
					return fmt.Errorf("attribute %q: empty or duplicate value %q", a.Key, v)
				}
				seen[v] = true
			}
		case TypeInt:
			if a.Min == nil || a.Max == nil || *a.Min > *a.Max {
				return fmt.Errorf("attribute %q: int needs min <= max", a.Key)
			}
			if a.Step < 0 {
				return fmt.Errorf("attribute %q: negative step", a.Key)
			}
		case TypeBool, TypeString:
		default:
			return fmt.Errorf("attribute %q: unknown type %q", a.Key, a.Type)
		}
		for key := range a.Templates {
			op, val, perValue := strings.Cut(key, ":")
			if !opAllowed(a.Type, op) && op != "is_false" {
				return fmt.Errorf("attribute %q: template for disallowed op %q", a.Key, op)
			}
			if perValue && !contains(a.Values, val) {
				return fmt.Errorf("attribute %q: template for unknown value %q", a.Key, val)
			}
		}
		for v := range a.Aliases {
			if !contains(a.Values, v) {
				return fmt.Errorf("attribute %q: aliases for unknown value %q", a.Key, v)
			}
		}
		p.attrIndex[a.Key] = a
	}

	p.candIndex = make(map[string]*Candidate, len(p.Candidates))
	names := map[string]bool{}
	for i := range p.Candidates {
		c := &p.Candidates[i]
		if !candIDRe.MatchString(c.ID) {
			return fmt.Errorf("candidate %q: id must match c_xxxxxx (random hex)", c.ID)
		}
		if _, dup := p.candIndex[c.ID]; dup {
			return fmt.Errorf("duplicate candidate id %q", c.ID)
		}
		if strings.TrimSpace(c.Name) == "" {
			return fmt.Errorf("candidate %q: name required", c.ID)
		}
		if names[strings.ToLower(c.Name)] {
			return fmt.Errorf("duplicate candidate name %q", c.Name)
		}
		names[strings.ToLower(c.Name)] = true
		if err := c.Image.validate(); err != nil {
			return fmt.Errorf("candidate %q image: %w", c.Name, err)
		}
		c.vals = make(map[string]value, len(c.Attrs))
		for key, raw := range c.Attrs {
			a, ok := p.attrIndex[key]
			if !ok {
				return fmt.Errorf("candidate %q: unknown attribute %q", c.Name, key)
			}
			v, err := parseValue(a, raw)
			if err != nil {
				return fmt.Errorf("candidate %q attribute %q: %w", c.Name, key, err)
			}
			c.vals[key] = v
		}
		p.candIndex[c.ID] = c
	}
	return nil
}

func parseValue(a *Attribute, raw json.RawMessage) (value, error) {
	v := value{kind: a.Type}
	switch a.Type {
	case TypeEnum:
		if err := json.Unmarshal(raw, &v.str); err != nil {
			return v, fmt.Errorf("want string")
		}
		if !contains(a.Values, v.str) {
			return v, fmt.Errorf("value %q not in allowed values", v.str)
		}
	case TypeString:
		if err := json.Unmarshal(raw, &v.str); err != nil || v.str == "" {
			return v, fmt.Errorf("want non-empty string")
		}
	case TypeBool:
		if err := json.Unmarshal(raw, &v.b); err != nil {
			return v, fmt.Errorf("want bool")
		}
	case TypeInt:
		if err := json.Unmarshal(raw, &v.n); err != nil {
			return v, fmt.Errorf("want integer")
		}
		if v.n < *a.Min || v.n > *a.Max {
			return v, fmt.Errorf("%d outside [%d, %d]", v.n, *a.Min, *a.Max)
		}
	case TypeSet:
		var items []string
		if err := json.Unmarshal(raw, &items); err != nil {
			return v, fmt.Errorf("want string array")
		}
		v.set = make(map[string]bool, len(items))
		for _, it := range items {
			if !contains(a.Values, it) {
				return v, fmt.Errorf("set value %q not in allowed values", it)
			}
			v.set[it] = true
		}
	}
	return v, nil
}

// Attr returns the attribute with key k.
func (p *Pack) Attr(k string) (*Attribute, bool) {
	a, ok := p.attrIndex[k]
	return a, ok
}

// Candidate returns the candidate with id.
func (p *Pack) Candidate(id string) (*Candidate, bool) {
	c, ok := p.candIndex[id]
	return c, ok
}

// SamplePool picks size candidates deterministically from seed, sorted by name.
// size <= 0 or >= len(candidates) returns every candidate.
func (p *Pack) SamplePool(size int, seed uint64) []*Candidate {
	all := make([]*Candidate, len(p.Candidates))
	for i := range p.Candidates {
		all[i] = &p.Candidates[i]
	}
	if size > 0 && size < len(all) {
		rng := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
		rng.Shuffle(len(all), func(i, j int) { all[i], all[j] = all[j], all[i] })
		all = all[:size]
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Name < all[j].Name })
	return all
}

// AttrString renders a candidate attribute for display ("" if missing).
func (c *Candidate) AttrString(a *Attribute) string {
	v, ok := c.vals[a.Key]
	if !ok {
		return ""
	}
	switch v.kind {
	case TypeBool:
		if v.b {
			return "Yes"
		}
		return "No"
	case TypeInt:
		return fmt.Sprint(v.n)
	case TypeSet:
		var out []string
		for _, val := range a.Values {
			if v.set[val] {
				out = append(out, val)
			}
		}
		return strings.Join(out, ", ")
	default:
		return v.str
	}
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

var assetRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*/[A-Za-z0-9_-]+\.(webp|jpg|jpeg|png)$`)

// validate rejects an image without the attribution a reuse licence needs,
// or whose asset path could escape ASSETS_DIR.
func (im *Image) validate() error {
	if im == nil {
		return nil
	}
	if !assetRe.MatchString(im.Asset) {
		return fmt.Errorf("asset %q must look like <pack>/<file>.webp", im.Asset)
	}
	if strings.TrimSpace(im.License) == "" || strings.TrimSpace(im.Credit) == "" {
		return fmt.Errorf("licence and credit are required")
	}
	if !strings.HasPrefix(im.SourceURL, "https://") {
		return fmt.Errorf("source_url must be an https file page")
	}
	return nil
}

// DropImages removes image references for which keep returns false (for
// example, files missing from ASSETS_DIR), so the client falls back to
// initials instead of a broken picture. It returns the number dropped.
func (p *Pack) DropImages(keep func(asset string) bool) int {
	n := 0
	for i := range p.Candidates {
		if im := p.Candidates[i].Image; im != nil && !keep(im.Asset) {
			p.Candidates[i].Image = nil
			n++
		}
	}
	return n
}
