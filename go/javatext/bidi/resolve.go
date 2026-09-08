package bidi

import (
	xbidi "golang.org/x/text/unicode/bidi"
)

// The resolution rules of UAX#9, in the order the annex gives them.
//
// The names below are the rule numbers, so that each block can be read against
// https://www.unicode.org/reports/tr9/ rather than against prose.

// resolve fills in levels, paragraphLevel and runs.
func (p *Paragraph) resolve(flags int) {
	// The class of each unit. A surrogate pair takes its class on the first
	// unit and BN on the second, so that indices stay UTF-16 throughout: the
	// rules of the annex are about characters, and the second unit of a pair
	// is not one, so it has to be invisible to them the way an X9-removed
	// character is.
	//
	// `trailing` remembers which units those are, because being invisible to
	// the rules is not the same as taking whatever level they leave behind.
	// See restoreSurrogatePairs.
	classes := make([]xbidi.Class, len(p.units))
	trailing := make([]bool, len(p.units))
	for i := 0; i < len(p.units); {
		r, size := decodeUnits(p.units[i:])
		classes[i] = classOf(r)
		for j := 1; j < size; j++ {
			classes[i+j] = xbidi.BN
			trailing[i+j] = true
		}
		i += size
	}
	original := make([]xbidi.Class, len(classes))
	copy(original, classes)

	p.levels = make([]int, len(classes))

	// P1: split the text into paragraphs at each paragraph separator, keeping
	// the separator with the paragraph it ends, and run the rest of the
	// algorithm over each. `new Bidi(String, int)` does this -- the corpus has
	// Hebrew, a newline and then Latin, and the running Java answers a level 1
	// run ending in the newline followed by a level 0 run, which is two
	// paragraphs each with its own P2 and P3.
	first := true
	for start := 0; start < len(classes); {
		end := start
		for end < len(classes) && original[end] != xbidi.B {
			end++
		}
		if end < len(classes) {
			end++ // the separator belongs to the paragraph it ends
		}
		level := p.resolveParagraph(classes, original, start, end, flags)
		if first {
			// getBaseLevel answers the first paragraph's level.
			p.paragraphLevel = level
			first = false
		}
		start = end
	}
	if len(classes) == 0 {
		p.paragraphLevel = paragraphLevel(nil, flags)
	}

	restoreSurrogatePairs(p.levels, trailing)
	p.buildRuns()
}

// restoreSurrogatePairs gives the second unit of a surrogate pair the level of
// the first.
//
// The rules of the annex ran over it as an X9-removed character, which is what
// keeps a pair from counting as two characters, and X9-removed characters end
// up with the level of the character that *follows* them. That is right for a
// formatting control and wrong for half a character: it puts a run boundary
// between the two units, `buildRuns` splits them, and each half on its own
// decodes to a replacement character. An emoji beside a Hebrew word came out
// as two of those.
//
// Java has nothing corresponding, because `java.text.Bidi` works in code
// points and never had the two halves apart. Measured against it: over eight
// texts with a supplementary character next to a right-to-left run, the JDK
// puts no run boundary inside a pair in any of them.
func restoreSurrogatePairs(levels []int, trailing []bool) {
	for i := 1; i < len(levels); i++ {
		if trailing[i] {
			levels[i] = levels[i-1]
		}
	}
}

// resolveParagraph runs X, W, N, I and L1 over one paragraph of the text and
// answers the level P2 and P3 gave it.
func (p *Paragraph) resolveParagraph(classes, original []xbidi.Class,
	start, end, flags int) int {
	paragraph := paragraphLevel(original[start:end], flags)

	explicitLevels(classes[start:end], p.levels[start:end], paragraph)

	for _, sequence := range isolatingRunSequences(classes[start:end], original[start:end],
		p.levels[start:end], paragraph) {
		sequence.resolveWeak()
		sequence.resolveNeutral(p.units[start:end])
		sequence.resolveImplicit()
		sequence.writeBack(classes[start:end], p.levels[start:end])
	}

	resetSeparators(p.levels[start:end], original[start:end], paragraph)
	return paragraph
}

// paragraphLevel is rules P2 and P3: the level of the paragraph comes from its
// first strong character, skipping anything inside an isolate.
func paragraphLevel(classes []xbidi.Class, flags int) int {
	switch flags {
	case DirectionLeftToRight:
		return 0
	case DirectionRightToLeft:
		return 1
	}
	isolate := 0
	for _, class := range classes {
		switch class {
		case xbidi.LRI, xbidi.RLI, xbidi.FSI:
			isolate++
		case xbidi.PDI:
			if isolate > 0 {
				isolate--
			}
		case xbidi.L:
			if isolate == 0 {
				return 0
			}
		case xbidi.R, xbidi.AL:
			if isolate == 0 {
				return 1
			}
		}
	}
	if flags == DirectionDefaultRightToLeft {
		return 1
	}
	return 0
}

// directionalStatus is one entry of the stack rules X1 to X8 keep.
type directionalStatus struct {
	level              int
	override           xbidi.Class // L, R, or unknown for neutral
	directionalIsolate bool
}

// neutralOverride is "neutral" in the override slot, which UAX#9 writes as a
// status of neither L nor R.
const neutralOverride = xbidi.Class(0xFF)

// explicitLevels is rules X1 to X8: walk the text keeping a stack of embedding
// levels and overrides, and give every character a level.
func explicitLevels(classes []xbidi.Class, levels []int, paragraph int) {
	stack := []directionalStatus{{level: paragraph, override: neutralOverride}}
	overflowIsolates := 0
	overflowEmbeddings := 0
	validIsolates := 0

	top := func() directionalStatus { return stack[len(stack)-1] }

	// nextOdd and nextEven are UAX#9's "least odd/even level greater than".
	nextOdd := func(level int) int { return level + 1 + (level & 1) ^ 0 }
	_ = nextOdd

	leastGreaterOdd := func(level int) int {
		if level%2 == 0 {
			return level + 1
		}
		return level + 2
	}
	leastGreaterEven := func(level int) int {
		if level%2 == 0 {
			return level + 2
		}
		return level + 1
	}

	for i, class := range classes {
		switch class {
		case xbidi.RLE, xbidi.LRE, xbidi.RLO, xbidi.LRO:
			// X2 to X5: an embedding or override initiator is itself removed by
			// X9, and takes the level of the entry it pushed onto.
			levels[i] = top().level
			var newLevel int
			var override xbidi.Class = neutralOverride
			switch class {
			case xbidi.RLE:
				newLevel = leastGreaterOdd(top().level)
			case xbidi.LRE:
				newLevel = leastGreaterEven(top().level)
			case xbidi.RLO:
				newLevel = leastGreaterOdd(top().level)
				override = xbidi.R
			case xbidi.LRO:
				newLevel = leastGreaterEven(top().level)
				override = xbidi.L
			}
			if newLevel <= maxDepth && overflowIsolates == 0 && overflowEmbeddings == 0 {
				stack = append(stack, directionalStatus{level: newLevel, override: override})
			} else if overflowIsolates == 0 {
				overflowEmbeddings++
			}

		case xbidi.RLI, xbidi.LRI, xbidi.FSI:
			// X5a to X5c.
			effective := class
			if class == xbidi.FSI {
				if firstStrongIsRTL(classes[i+1:]) {
					effective = xbidi.RLI
				} else {
					effective = xbidi.LRI
				}
			}
			levels[i] = top().level
			if top().override != neutralOverride {
				classes[i] = top().override
			}
			var newLevel int
			if effective == xbidi.RLI {
				newLevel = leastGreaterOdd(top().level)
			} else {
				newLevel = leastGreaterEven(top().level)
			}
			if newLevel <= maxDepth && overflowIsolates == 0 && overflowEmbeddings == 0 {
				validIsolates++
				stack = append(stack, directionalStatus{
					level: newLevel, override: neutralOverride, directionalIsolate: true})
			} else {
				overflowIsolates++
			}

		case xbidi.PDI:
			// X6a.
			if overflowIsolates > 0 {
				overflowIsolates--
			} else if validIsolates > 0 {
				overflowEmbeddings = 0
				for !top().directionalIsolate {
					stack = stack[:len(stack)-1]
				}
				stack = stack[:len(stack)-1]
				validIsolates--
			}
			levels[i] = top().level
			if top().override != neutralOverride {
				classes[i] = top().override
			}

		case xbidi.PDF:
			// X7. The PDF itself is removed by X9 and takes the level it ends
			// up at.
			if overflowIsolates > 0 {
				// nothing
			} else if overflowEmbeddings > 0 {
				overflowEmbeddings--
			} else if !top().directionalIsolate && len(stack) >= 2 {
				stack = stack[:len(stack)-1]
			}
			levels[i] = top().level

		case xbidi.B:
			// X8: a paragraph separator takes the paragraph level and resets
			// everything.
			stack = stack[:1]
			overflowIsolates, overflowEmbeddings, validIsolates = 0, 0, 0
			levels[i] = paragraph

		default:
			// X6.
			levels[i] = top().level
			if top().override != neutralOverride {
				classes[i] = top().override
			}
		}
	}
}

// firstStrongIsRTL is the lookahead rule X5c performs for an FSI: scan to the
// matching PDI for the first strong character.
func firstStrongIsRTL(rest []xbidi.Class) bool {
	depth := 0
	for _, class := range rest {
		switch class {
		case xbidi.LRI, xbidi.RLI, xbidi.FSI:
			depth++
		case xbidi.PDI:
			if depth == 0 {
				return false
			}
			depth--
		case xbidi.L:
			if depth == 0 {
				return false
			}
		case xbidi.R, xbidi.AL:
			if depth == 0 {
				return true
			}
		}
	}
	return false
}

// resetSeparators is rule L1: a segment or paragraph separator, and any run of
// whitespace or isolate formatting before one or at the end of the line, goes
// back to the paragraph level.
//
// It then gives every character rule X9 removed the level of the one after it,
// which is a choice the annex leaves open and the JDK makes this way: the
// corpus has an RLE, "abc" and a PDF, and the running Java answers the RLE at
// the level of the "abc" it opened rather than at the level it was pushed from.
func resetSeparators(levels []int, original []xbidi.Class, paragraph int) {
	resetting := true
	for i := len(levels) - 1; i >= 0; i-- {
		switch original[i] {
		case xbidi.B, xbidi.S:
			levels[i] = paragraph
			resetting = true
		case xbidi.WS, xbidi.RLI, xbidi.LRI, xbidi.FSI, xbidi.PDI:
			if resetting {
				levels[i] = paragraph
			}
		case xbidi.RLE, xbidi.LRE, xbidi.RLO, xbidi.LRO, xbidi.PDF, xbidi.BN:
			// Removed by X9, so they do not end a run of whitespace.
			if resetting {
				levels[i] = paragraph
			}
		default:
			resetting = false
		}
	}

	// The X9-removed characters take the level of the next character that was
	// not removed, or the paragraph level at the end.
	following := paragraph
	for i := len(levels) - 1; i >= 0; i-- {
		if removedByX9(original[i]) {
			levels[i] = following
		} else {
			following = levels[i]
		}
	}
}

// buildRuns groups the units into maximal runs of one level.
func (p *Paragraph) buildRuns() {
	if len(p.levels) == 0 {
		p.runs = nil
		return
	}
	start := 0
	for i := 1; i <= len(p.levels); i++ {
		if i == len(p.levels) || p.levels[i] != p.levels[start] {
			p.runs = append(p.runs, Run{Start: start, Limit: i, Level: p.levels[start]})
			start = i
		}
	}
}
