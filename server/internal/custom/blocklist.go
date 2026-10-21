package custom

import "strings"

// Words that are also names or places (Dick, Lund, Randi) are deliberately
// left out to avoid blocking factual packs.
//
// blockedWords is a short first line of defence against abusive packs; the
// report button and the admin review do the rest. Matching is on whole words
// after folding common character swaps (0->o, 1->i, 3->e, 4->a, 5->s, @->a, $->s).
// Extend with BLOCKED_WORDS (comma separated) at deploy time.
var blockedWords = map[string]bool{}

func init() {
	for _, w := range strings.Fields(`
		fuck fucking fucker motherfucker shit bitch cunt pussy whore slut
		nigger nigga faggot retard retarded tranny chink spic kike paki
		chutiya chutiye madarchod behenchod bhenchod bhosdi bhosdike gaandu gandu harami`) {
		blockedWords[w] = true
	}
}

// AddBlockedWords extends the list (from configuration).
func AddBlockedWords(csv string) {
	for _, w := range strings.Split(csv, ",") {
		if w = strings.ToLower(strings.TrimSpace(w)); w != "" {
			blockedWords[w] = true
		}
	}
}

var fold = strings.NewReplacer("0", "o", "1", "i", "3", "e", "4", "a", "5", "s", "@", "a", "$", "s", "!", "i")

// blocked returns the first blocked word in s, or "".
func blocked(s string) string {
	s = fold.Replace(strings.ToLower(s))
	words := strings.FieldsFunc(s, func(r rune) bool { return !(r >= 'a' && r <= 'z') })
	for _, w := range words {
		if blockedWords[w] {
			return w
		}
	}
	// Also catch words split by spaces or dots ("f u c k").
	joined := strings.Join(words, "")
	for w := range blockedWords {
		if len(w) >= 5 && strings.Contains(joined, w) && !strings.Contains(strings.Join(words, " "), w) {
			for _, part := range words {
				if len(part) <= 2 {
					return w
				}
			}
		}
	}
	return ""
}
