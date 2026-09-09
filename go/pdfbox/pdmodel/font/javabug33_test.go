package font

// JAVA-BUGS 33: `ToUnicodeWriter.allowDestinationRange` checks that `prev` is
// a single code point and does not check `next`, so a CID mapped to one
// character followed by a CID mapped to more extends the bfrange and
// everything after the first character of the longer destination is dropped.
// In-package, because the writer and the predicate are.

import (
	"strings"
	"testing"
)

// TestDestinationRangeChecksBothSides is the defect.
//
// A bfrange is one starting destination that the reader increments, so it is
// only correct when every destination in the range is a single code point. The
// predicate is therefore symmetric in its two arguments, and the expected
// values are that symmetry: what it answers for ("ab", "c") is what it must
// answer for ("a", "bc"). Java answers false and true.
func TestDestinationRangeChecksBothSides(t *testing.T) {
	if allowDestinationRange("a", "bc") {
		t.Error(`allowDestinationRange("a", "bc") = true; a range cannot carry ` +
			`a destination of two code points`)
	}
	// the side Java does check, unchanged
	if allowDestinationRange("ab", "c") {
		t.Error(`allowDestinationRange("ab", "c") = true`)
	}
	// and one code point on each side is still a range
	if !allowDestinationRange("a", "b") {
		t.Error(`allowDestinationRange("a", "b") = false`)
	}
}

// TestCMapKeepsTheTailOfALongerDestination is the consequence, at the writer.
//
// With 0x400 mapped to "a" and 0x401 mapped to "bc", the running Java writes
// one bfrange, `<0400> <0401> <0061>`, and the "c" is nowhere in the CMap: a
// reader decodes 0x401 as "b" and the ligature the mapping existed for is
// gone. Refusing the range writes each destination as its own entry, whole --
// the writer always writes bfranges, so a destination that cannot be joined is
// a range of one.
func TestCMapKeepsTheTailOfALongerDestination(t *testing.T) {
	writer := newToUnicodeWriter()
	writer.add(0x400, "a")
	writer.add(0x401, "bc")

	output := writeToString(t, writer)

	assertContains(t, output, "2 beginbfrange")
	assertContains(t, output, "<0400> <0400> <0061>")
	assertContains(t, output, "<0401> <0401> <00620063>")
	if strings.Contains(output, "<0400> <0401>") {
		t.Errorf("the two destinations were joined into one range:\n%s", output)
	}
}
