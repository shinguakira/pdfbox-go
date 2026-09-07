package bidi

import (
	"unicode"

	xbidi "golang.org/x/text/unicode/bidi"
)

// Rule N0 and BD16: matching pairs of brackets take the direction of what is
// inside them, or of what surrounds them.
//
// The corpus measures that this rule is needed rather than optional: the
// running Java answers `["(" and the Hebrew at level 1, "abc" at level 2, ")"
// at level 1]` for `אבג (abc)`, which is N0 pairing the two parentheses and
// giving them the surrounding right-to-left direction.

// bracketPair is one matched pair, by position within the run sequence.
type bracketPair struct{ opener, closer int }

// resolvePairedBrackets is N0.
func (s *runSequence) resolvePairedBrackets(units []uint16) {
	pairs := s.findBracketPairs(units)
	if len(pairs) == 0 {
		return
	}
	embedding := directionOf(s.level)
	opposite := xbidi.L
	if embedding == xbidi.L {
		opposite = xbidi.R
	}

	for _, pair := range pairs {
		// a. Inspect the bidirectional types of the characters enclosed within
		// the bracket pair.
		foundEmbedding := false
		foundOpposite := false
		for i := pair.opener + 1; i < pair.closer; i++ {
			direction := strongDirection(s.classes[i])
			if direction != xbidi.L && direction != xbidi.R {
				continue
			}
			if direction == embedding {
				foundEmbedding = true
			} else {
				foundOpposite = true
			}
		}

		switch {
		case foundEmbedding:
			// b. Set the type for both brackets to the embedding direction.
			s.setBracketPair(pair, embedding)
		case foundOpposite:
			// c. Look backwards for a preceding strong type.
			previous := s.sos
			for i := pair.opener - 1; i >= 0; i-- {
				direction := strongDirection(s.classes[i])
				if direction == xbidi.L || direction == xbidi.R {
					previous = direction
					break
				}
			}
			if previous == opposite {
				// c.1: an established context of the opposite direction.
				s.setBracketPair(pair, opposite)
			} else {
				// c.2: otherwise the embedding direction.
				s.setBracketPair(pair, embedding)
			}
		default:
			// d. No strong type inside: leave both brackets alone.
		}
	}
}

// setBracketPair gives both brackets the direction, and with them any
// non-spacing mark that followed either, which is N0's closing note.
func (s *runSequence) setBracketPair(pair bracketPair, direction xbidi.Class) {
	s.classes[pair.opener] = direction
	s.classes[pair.closer] = direction
	for _, at := range []int{pair.opener, pair.closer} {
		for i := at + 1; i < len(s.classes); i++ {
			if s.originalClassAt(i) != xbidi.NSM {
				break
			}
			s.classes[i] = direction
		}
	}
}

// findBracketPairs is BD16: a stack of at most 63 openers, matched by the
// Bidi_Paired_Bracket property, answered in order of opener position.
func (s *runSequence) findBracketPairs(units []uint16) []bracketPair {
	const maxPairingDepth = 63
	type openBracket struct {
		position int
		closer   rune
	}
	var stack []openBracket
	var pairs []bracketPair

	for i, class := range s.classes {
		// BD14/BD15: only a character that is still ON can open or close.
		if class != xbidi.ON {
			continue
		}
		r := canonical(decodeAt(units, s.indices[i]))
		if closer, isOpener := pairedBracketOpeners[r]; isOpener {
			if len(stack) == maxPairingDepth {
				// BD16 stops altogether once the stack overflows.
				break
			}
			stack = append(stack, openBracket{position: i, closer: closer})
			continue
		}
		if _, isCloser := pairedBracketClosers[r]; !isCloser {
			continue
		}
		for depth := len(stack) - 1; depth >= 0; depth-- {
			if stack[depth].closer != r {
				continue
			}
			pairs = append(pairs, bracketPair{opener: stack[depth].position, closer: i})
			stack = stack[:depth]
			break
		}
	}

	// BD16 answers the pairs sorted by the position of the opener.
	for i := 1; i < len(pairs); i++ {
		for j := i; j > 0 && pairs[j-1].opener > pairs[j].opener; j-- {
			pairs[j-1], pairs[j] = pairs[j], pairs[j-1]
		}
	}
	return pairs
}

// originalClassAt answers the class a position had before the rules changed
// it, which N0's note about non-spacing marks needs.
func (s *runSequence) originalClassAt(i int) xbidi.Class {
	return s.original[i]
}

// canonical maps the two brackets whose canonical equivalents differ, which is
// BD16's note: U+2329 and U+3008, and their closers, pair with each other.
func canonical(r rune) rune {
	switch r {
	case 0x3008:
		return 0x2329
	case 0x3009:
		return 0x232A
	}
	return r
}

// pairedBracketOpeners maps each opening bracket to the closer it pairs with,
// and pairedBracketClosers is the set of closers.
//
// Unicode's BidiBrackets.txt is the source. The port carries the pairs whose
// General_Category is Ps or Pe and which have a Bidi_Paired_Bracket, taken from
// the Unicode tables Go already has: every rune Go reports as an open or close
// punctuation with a mapped pair.
var (
	pairedBracketOpeners = map[rune]rune{}
	pairedBracketClosers = map[rune]bool{}
)

func init() {
	// The pairs of BidiBrackets.txt. Go's unicode tables carry Ps/Pe but not
	// the pairing, so the pairing is written out.
	pairs := [][2]rune{
		{0x0028, 0x0029}, {0x005B, 0x005D}, {0x007B, 0x007D},
		{0x0F3A, 0x0F3B}, {0x0F3C, 0x0F3D},
		{0x169B, 0x169C},
		{0x2045, 0x2046}, {0x207D, 0x207E}, {0x208D, 0x208E},
		{0x2308, 0x2309}, {0x230A, 0x230B},
		{0x2329, 0x232A},
		{0x2768, 0x2769}, {0x276A, 0x276B}, {0x276C, 0x276D}, {0x276E, 0x276F},
		{0x2770, 0x2771}, {0x2772, 0x2773}, {0x2774, 0x2775},
		{0x27C5, 0x27C6},
		{0x27E6, 0x27E7}, {0x27E8, 0x27E9}, {0x27EA, 0x27EB}, {0x27EC, 0x27ED},
		{0x27EE, 0x27EF},
		{0x2983, 0x2984}, {0x2985, 0x2986}, {0x2987, 0x2988}, {0x2989, 0x298A},
		{0x298B, 0x298C}, {0x298D, 0x2990}, {0x298F, 0x298E}, {0x2991, 0x2992},
		{0x2993, 0x2994}, {0x2995, 0x2996}, {0x2997, 0x2998},
		{0x29D8, 0x29D9}, {0x29DA, 0x29DB}, {0x29FC, 0x29FD},
		{0x2E22, 0x2E23}, {0x2E24, 0x2E25}, {0x2E26, 0x2E27}, {0x2E28, 0x2E29},
		{0x2E55, 0x2E56}, {0x2E57, 0x2E58}, {0x2E59, 0x2E5A}, {0x2E5B, 0x2E5C},
		{0x3008, 0x3009}, {0x300A, 0x300B}, {0x300C, 0x300D}, {0x300E, 0x300F},
		{0x3010, 0x3011}, {0x3014, 0x3015}, {0x3016, 0x3017}, {0x3018, 0x3019},
		{0x301A, 0x301B},
		{0xFE59, 0xFE5A}, {0xFE5B, 0xFE5C}, {0xFE5D, 0xFE5E},
		{0xFF08, 0xFF09}, {0xFF3B, 0xFF3D}, {0xFF5B, 0xFF5D}, {0xFF5F, 0xFF60},
		{0xFF62, 0xFF63},
	}
	for _, pair := range pairs {
		pairedBracketOpeners[canonical(pair[0])] = canonical(pair[1])
		pairedBracketClosers[canonical(pair[1])] = true
	}
}

var _ = unicode.Ps
