package game

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"
)

// Text matching turns a typed or spoken question into a pack question.
//
//   - MatchStrict only succeeds when the words are the same as a known
//     question's (ignoring filler words and simple word endings). It is
//     safe to answer automatically.
//   - MatchLoose also recognises a single mentioned value ("is he a
//     keeper from India" is not matched; "did he play for India" is). It is
//     used to suggest a question the player confirms, and to audit manual
//     answers at the reveal. It never answers anything on its own.

var stopwords = map[string]bool{
	"a": true, "an": true, "the": true, "is": true, "are": true, "was": true, "were": true,
	"do": true, "does": true, "did": true, "have": true, "has": true, "had": true,
	"they": true, "their": true, "them": true, "he": true, "his": true, "him": true,
	"she": true, "her": true, "it": true, "this": true, "that": true, "person": true,
	"player": true, "your": true, "you": true, "of": true, "in": true, "for": true,
	"ever": true, "be": true, "been": true, "true": true, "or": true, "secret": true,
}

var (
	negations   = map[string]bool{"not": true, "never": true, "no": true, "nt": true, "neither": true, "nor": true, "without": true}
	comparators = map[string]bool{"before": true, "after": true, "more": true, "less": true, "than": true, "between": true,
		"over": true, "under": true, "above": true, "below": true, "older": true, "newer": true, "earlier": true, "later": true}
	numberRe = regexp.MustCompile(`\b(\d{1,4})s?\b`)
	// "…start with K", "…begin with “AN”?" — the prefix may be a filler
	// word, so it is read before filler words are dropped.
	prefixRe = regexp.MustCompile(`(?i)\b(?:start|starts|begin|begins)\s+with\s+(?:the\s+letters?\s+)?["“'‘]?([\p{L}\p{N}]{1,3})["”'’]?\s*[?.!]*\s*$`)
)

// normalize lower-cases, turns punctuation into spaces and splits "don't"
// into "don nt" so negations survive.
func normalize(s string) []string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "n't", " nt")
	s = strings.ReplaceAll(s, "n’t", " nt")
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}
	return strings.Fields(b.String())
}

// stem drops common English endings so "represented" ~ "represent".
func stem(w string) string {
	for _, suf := range []string{"ing", "ed", "es", "s"} {
		if len(w) > len(suf)+3 && strings.HasSuffix(w, suf) {
			return w[:len(w)-len(suf)]
		}
	}
	return w
}

// key is the order-independent content of a sentence.
func key(words []string) string {
	set := map[string]bool{}
	for _, w := range words {
		if !stopwords[w] {
			set[stem(w)] = true
		}
	}
	out := make([]string, 0, len(set))
	for w := range set {
		out = append(out, w)
	}
	sort.Strings(out)
	return strings.Join(out, " ")
}

type textIndex struct {
	strict map[string]*Question // nil value = ambiguous key
}

var indexes sync.Map // *Pack -> *textIndex

func (p *Pack) index() *textIndex {
	if v, ok := indexes.Load(p); ok {
		return v.(*textIndex)
	}
	cands := make([]*Candidate, len(p.Candidates))
	for i := range p.Candidates {
		cands[i] = &p.Candidates[i]
	}
	idx := &textIndex{strict: map[string]*Question{}}
	for _, q := range AllQuestions(p, cands, 3, true) {
		q := q
		if q.Attr.Type == TypeString {
			continue // handled by matchPrefix
		}
		k := key(normalize(q.Text()))
		if k == "" {
			continue
		}
		if prev, dup := idx.strict[k]; dup {
			if prev != nil && prev.Canonical() != q.Canonical() {
				idx.strict[k] = nil
			}
			continue
		}
		idx.strict[k] = &q
	}
	v, _ := indexes.LoadOrStore(p, idx)
	return v.(*textIndex)
}

// MatchStrict maps text to a question only when its words are those of a
// known question.
func (p *Pack) MatchStrict(text string) (Question, bool) {
	if q, ok := p.matchPrefix(text); ok {
		return q, true
	}
	q := p.index().strict[key(normalize(text))]
	if q == nil {
		return Question{}, false
	}
	return *q, true
}

// matchPrefix reads "does their surname start with K?". With several string
// attributes, a word of the attribute's label must appear.
func (p *Pack) matchPrefix(text string) (Question, bool) {
	m := prefixRe.FindStringSubmatch(text)
	if m == nil {
		return Question{}, false
	}
	var strs []*Attribute
	for i := range p.Attributes {
		if p.Attributes[i].Type == TypeString {
			strs = append(strs, &p.Attributes[i])
		}
	}
	words := " " + strings.Join(normalize(text), " ") + " "
	var pick *Attribute
	for _, a := range strs {
		for _, lw := range normalize(a.Label) {
			if strings.Contains(words, " "+lw+" ") || strings.Contains(words, " "+lw+"s ") {
				if pick != nil && pick != a {
					return Question{}, false
				}
				pick = a
			}
		}
	}
	if pick == nil && len(strs) == 1 {
		pick = strs[0]
	}
	if pick == nil {
		return Question{}, false
	}
	q, err := p.Compile(Predicate{Attr: pick.Key, Op: "starts_with", Value: []byte(strconv.Quote(m[1]))})
	return q, err == nil
}

// MatchLoose tries MatchStrict, then a single-value mention.
func (p *Pack) MatchLoose(text string) (Question, bool) {
	if q, ok := p.MatchStrict(text); ok {
		return q, true
	}
	words := normalize(text)
	for _, w := range words {
		if negations[w] || comparators[w] {
			return Question{}, false // too easy to get the polarity or range wrong
		}
	}
	padded := " " + strings.Join(words, " ") + " "

	type phrase struct {
		text string
		attr *Attribute
		val  string
	}
	var phrases []phrase
	for i := range p.Attributes {
		a := &p.Attributes[i]
		if a.Type != TypeEnum && a.Type != TypeSet {
			continue
		}
		for _, v := range a.Values {
			phrases = append(phrases, phrase{strings.Join(normalize(v), " "), a, v})
			for _, al := range a.Aliases[v] {
				phrases = append(phrases, phrase{strings.Join(normalize(al), " "), a, v})
			}
		}
	}
	// Longest phrases first; a matched span is blanked so "left arm" is not
	// also read as "left".
	sort.SliceStable(phrases, func(i, j int) bool {
		if len(phrases[i].text) != len(phrases[j].text) {
			return len(phrases[i].text) > len(phrases[j].text)
		}
		return phrases[i].text < phrases[j].text
	})
	type hit struct {
		attr *Attribute
		val  string
	}
	hits := map[string]hit{}
	// Phrases with identical text (bare "left" for batting hand and bowling
	// arm) all count before the span is blanked, so the clash is seen.
	for i := 0; i < len(phrases); {
		j := i
		for j < len(phrases) && phrases[j].text == phrases[i].text {
			j++
		}
		text := phrases[i].text
		needle := " " + text + " "
		if text != "" && strings.Contains(padded, needle) {
			for _, ph := range phrases[i:j] {
				hits[ph.attr.Key+"|"+ph.val] = hit{ph.attr, ph.val}
			}
			padded = strings.ReplaceAll(padded, needle, " "+strings.Repeat("_", len(text))+" ")
		}
		i = j
	}

	// A number in an int attribute's range counts only if a word of that
	// attribute's label is present ("debut ... 1990s").
	for i := range p.Attributes {
		a := &p.Attributes[i]
		if a.Type != TypeInt {
			continue
		}
		labelWord := false
		for _, lw := range normalize(a.Label) {
			if len(lw) > 3 && strings.Contains(padded, " "+lw) {
				labelWord = true
			}
		}
		if !labelWord {
			continue
		}
		for _, m := range numberRe.FindAllStringSubmatch(padded, -1) {
			n, _ := strconv.Atoi(m[1])
			if n >= *a.Min && n <= *a.Max && (a.Step <= 0 || (n-*a.Min)%a.Step == 0) {
				hits[a.Key+"|"+m[1]] = hit{a, m[1]}
			}
		}
	}

	if len(hits) != 1 {
		return Question{}, false
	}
	for _, h := range hits {
		var raw []byte
		if h.attr.Type == TypeInt {
			raw = []byte(h.val)
		} else {
			raw = []byte(strconv.Quote(h.val))
		}
		op := OpsFor(h.attr.Type)[0]
		if q, err := p.Compile(Predicate{Attr: h.attr.Key, Op: op, Value: raw}); err == nil {
			return q, true
		}
	}
	return Question{}, false
}
