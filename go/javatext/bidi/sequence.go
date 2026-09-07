package bidi

import (
	"unicode/utf16"

	xbidi "golang.org/x/text/unicode/bidi"
)

// The isolating run sequences of rule X10, and the weak, neutral and implicit
// rules that run over each of them.

// runSequence is one isolating run sequence: the indices it covers, the classes
// at those indices, and the two directions on either side of it.
type runSequence struct {
	indices  []int
	classes  []xbidi.Class
	original []xbidi.Class
	levels   []int

	level    int
	sos, eos xbidi.Class
}

// isolatingRunSequences is rule X10 and BD13: chain each level run to the one
// its isolate initiator points at, and work out sos and eos for each chain.
func isolatingRunSequences(classes, original []xbidi.Class, levels []int,
	paragraph int) []*runSequence {
	// The level runs, over the characters X9 did not remove.
	type levelRun struct{ indices []int }
	var runs []levelRun
	var current []int
	currentLevel := -1
	for i, class := range original {
		if removedByX9(class) {
			continue
		}
		if currentLevel != levels[i] {
			if len(current) != 0 {
				runs = append(runs, levelRun{current})
			}
			current = nil
			currentLevel = levels[i]
		}
		current = append(current, i)
	}
	if len(current) != 0 {
		runs = append(runs, levelRun{current})
	}
	if len(runs) == 0 {
		return nil
	}

	// runOf answers which level run a character index belongs to.
	runOf := map[int]int{}
	for r, run := range runs {
		for _, i := range run.indices {
			runOf[i] = r
		}
	}

	used := make([]bool, len(runs))
	var sequences []*runSequence
	for r := range runs {
		if used[r] {
			continue
		}
		// BD13: a sequence starts at a run whose first character is not a PDI
		// matching an isolate initiator.
		first := runs[r].indices[0]
		if original[first] == xbidi.PDI && matchingInitiator(original, first) >= 0 {
			continue
		}
		var indices []int
		next := r
		for {
			used[next] = true
			indices = append(indices, runs[next].indices...)
			last := runs[next].indices[len(runs[next].indices)-1]
			if !isIsolateInitiator(original[last]) {
				break
			}
			pdi := matchingPDI(original, last)
			if pdi < 0 {
				break
			}
			following, ok := runOf[pdi]
			if !ok || used[following] {
				break
			}
			next = following
		}
		sequences = append(sequences, newRunSequence(indices, classes, levels,
			original, paragraph))
	}
	return sequences
}

// newRunSequence works out sos and eos for a sequence, which is the rest of
// X10.
func newRunSequence(indices []int, classes []xbidi.Class, levels []int,
	original []xbidi.Class, paragraph int) *runSequence {
	s := &runSequence{indices: indices, level: levels[indices[0]]}
	for _, i := range indices {
		s.classes = append(s.classes, classes[i])
		s.original = append(s.original, original[i])
		s.levels = append(s.levels, levels[i])
	}

	// sos: compare the sequence's level with the level of the character before
	// its first, skipping what X9 removed; the paragraph level stands in where
	// there is none.
	before := paragraph
	for i := indices[0] - 1; i >= 0; i-- {
		if removedByX9(original[i]) {
			continue
		}
		before = levels[i]
		break
	}
	s.sos = directionOf(max(s.level, before))

	// eos: the same after the last, except that an unmatched isolate initiator
	// takes the paragraph level.
	last := indices[len(indices)-1]
	after := paragraph
	if isIsolateInitiator(original[last]) && matchingPDI(original, last) < 0 {
		after = paragraph
	} else {
		for i := last + 1; i < len(original); i++ {
			if removedByX9(original[i]) {
				continue
			}
			after = levels[i]
			break
		}
	}
	s.eos = directionOf(max(s.level, after))
	return s
}

// resolveWeak is rules W1 to W7.
func (s *runSequence) resolveWeak() {
	// W1: a non-spacing mark takes the class of what precedes it, or sos.
	previous := s.sos
	for i, class := range s.classes {
		if class == xbidi.NSM {
			if isIsolateInitiator(previous) || previous == xbidi.PDI {
				s.classes[i] = xbidi.ON
			} else {
				s.classes[i] = previous
			}
		}
		previous = s.classes[i]
	}

	// W2: a European number after an Arabic letter is an Arabic number.
	strong := s.sos
	for i, class := range s.classes {
		switch class {
		case xbidi.L, xbidi.R, xbidi.AL:
			strong = class
		case xbidi.EN:
			if strong == xbidi.AL {
				s.classes[i] = xbidi.AN
			}
		}
	}

	// W3: an Arabic letter is right to left.
	for i, class := range s.classes {
		if class == xbidi.AL {
			s.classes[i] = xbidi.R
		}
	}

	// W4: a single separator between two numbers of the same kind joins them.
	for i := 1; i < len(s.classes)-1; i++ {
		switch s.classes[i] {
		case xbidi.ES:
			if s.classes[i-1] == xbidi.EN && s.classes[i+1] == xbidi.EN {
				s.classes[i] = xbidi.EN
			}
		case xbidi.CS:
			if s.classes[i-1] == xbidi.EN && s.classes[i+1] == xbidi.EN {
				s.classes[i] = xbidi.EN
			} else if s.classes[i-1] == xbidi.AN && s.classes[i+1] == xbidi.AN {
				s.classes[i] = xbidi.AN
			}
		}
	}

	// W5: a run of European terminators next to a European number joins it.
	for i := 0; i < len(s.classes); i++ {
		if s.classes[i] != xbidi.ET {
			continue
		}
		end := i
		for end < len(s.classes) && s.classes[end] == xbidi.ET {
			end++
		}
		before := s.sos
		if i > 0 {
			before = s.classes[i-1]
		}
		after := s.eos
		if end < len(s.classes) {
			after = s.classes[end]
		}
		if before == xbidi.EN || after == xbidi.EN {
			for j := i; j < end; j++ {
				s.classes[j] = xbidi.EN
			}
		}
		i = end - 1
	}

	// W6: whatever separators and terminators are left become neutral.
	for i, class := range s.classes {
		switch class {
		case xbidi.ES, xbidi.ET, xbidi.CS:
			s.classes[i] = xbidi.ON
		}
	}

	// W7: a European number after a left-to-right character is left to right.
	strong = s.sos
	for i, class := range s.classes {
		switch class {
		case xbidi.L, xbidi.R:
			strong = class
		case xbidi.EN:
			if strong == xbidi.L {
				s.classes[i] = xbidi.L
			}
		}
	}
}

// resolveNeutral is rules N0 to N2.
func (s *runSequence) resolveNeutral(units []uint16) {
	s.resolvePairedBrackets(units)

	embedding := directionOf(s.level)
	for i := 0; i < len(s.classes); i++ {
		if !isNeutralOrIsolate(s.classes[i]) {
			continue
		}
		end := i
		for end < len(s.classes) && isNeutralOrIsolate(s.classes[end]) {
			end++
		}
		before := s.sos
		if i > 0 {
			before = strongDirection(s.classes[i-1])
		}
		after := s.eos
		if end < len(s.classes) {
			after = strongDirection(s.classes[end])
		}
		// N1: neutrals between two of the same direction take it. N2:
		// otherwise they take the embedding direction.
		resolved := embedding
		if before == after && (before == xbidi.L || before == xbidi.R) {
			resolved = before
		}
		for j := i; j < end; j++ {
			s.classes[j] = resolved
		}
		i = end - 1
	}
}

// resolveImplicit is rules I1 and I2.
func (s *runSequence) resolveImplicit() {
	for i, class := range s.classes {
		level := s.level
		if level%2 == 0 {
			// I1
			switch class {
			case xbidi.R:
				level++
			case xbidi.AN, xbidi.EN:
				level += 2
			}
		} else {
			// I2
			switch class {
			case xbidi.L, xbidi.EN, xbidi.AN:
				level++
			}
		}
		s.levels[i] = level
	}
}

// writeBack copies the sequence's resolved classes and levels back over the
// paragraph.
func (s *runSequence) writeBack(classes []xbidi.Class, levels []int) {
	for j, i := range s.indices {
		classes[i] = s.classes[j]
		levels[i] = s.levels[j]
	}
}

// strongDirection maps a resolved class to the direction rules N1 and N2 pair
// on: a number counts as right to left there.
func strongDirection(class xbidi.Class) xbidi.Class {
	switch class {
	case xbidi.L:
		return xbidi.L
	case xbidi.R, xbidi.EN, xbidi.AN:
		return xbidi.R
	}
	return class
}

// isNeutralOrIsolate is UAX#9's NI: a neutral or an isolate formatting
// character.
func isNeutralOrIsolate(class xbidi.Class) bool {
	switch class {
	case xbidi.B, xbidi.S, xbidi.WS, xbidi.ON, xbidi.FSI, xbidi.LRI, xbidi.RLI, xbidi.PDI:
		return true
	}
	return false
}

// directionOf answers the direction a level runs in.
func directionOf(level int) xbidi.Class {
	if level%2 == 0 {
		return xbidi.L
	}
	return xbidi.R
}

// removedByX9 reports whether rule X9 takes the character out of the
// resolution.
func removedByX9(class xbidi.Class) bool {
	switch class {
	case xbidi.RLE, xbidi.LRE, xbidi.RLO, xbidi.LRO, xbidi.PDF, xbidi.BN:
		return true
	}
	return false
}

// isIsolateInitiator reports whether the class opens an isolate.
func isIsolateInitiator(class xbidi.Class) bool {
	switch class {
	case xbidi.LRI, xbidi.RLI, xbidi.FSI:
		return true
	}
	return false
}

// matchingPDI answers the index of the PDI that closes the isolate opened at
// the given index, or -1 where the text ends first. BD9.
func matchingPDI(classes []xbidi.Class, initiator int) int {
	depth := 1
	for i := initiator + 1; i < len(classes); i++ {
		switch classes[i] {
		case xbidi.LRI, xbidi.RLI, xbidi.FSI:
			depth++
		case xbidi.PDI:
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// matchingInitiator answers the index of the isolate initiator the PDI at the
// given index closes, or -1. BD10.
func matchingInitiator(classes []xbidi.Class, pdi int) int {
	depth := 1
	for i := pdi - 1; i >= 0; i-- {
		switch classes[i] {
		case xbidi.PDI:
			depth++
		case xbidi.LRI, xbidi.RLI, xbidi.FSI:
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// decodeAt reads the rune at a UTF-16 index of the paragraph's units.
func decodeAt(units []uint16, i int) rune {
	r, _ := decodeUnits(units[i:])
	return r
}

var _ = utf16.Encode
