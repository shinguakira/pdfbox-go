package type4

// PDFBOX-6331, Apache 47d20cb71, in the sync of 2026-11-21, and JAVA-BUGS.md
// 92.
//
// `roll` rotates the top n elements of the stack by j, and PostScript takes j
// modulo n: `3 7 roll` is `3 1 roll`. The Java guarded a negative n and a zero
// j and nothing else, so it popped j elements -- or |j| for a negative roll --
// where only n were promised. With other values under them it rotated
// elements the operator was never given; with nothing under them it ran off
// the bottom of the stack, which is EmptyStackException in Java and a panic
// here.
//
// Upstream returns for n == 0, takes j modulo n, and returns again where that
// leaves zero. Every expected value below is the Java's, read off the existing
// TestRoll above: a roll by the remainder has to leave the stack exactly as
// the roll by that remainder does.

import "testing"

// TestRollByMoreThanNTakesTheRemainder is the fix. `5 7 roll` is `5 2 roll`,
// whose expected stack is TestRoll's second case. Before the fix the operator
// popped seven values off a stack of five and panicked.
func TestRollByMoreThanNTakesTheRemainder(t *testing.T) {
	newType4Tester(t, "1 2 3 4 5 5 7 roll").
		popInt(3).popInt(2).popInt(1).popInt(5).popInt(4).isEmpty()
}

// TestNegativeRollByMoreThanNTakesTheRemainder is the other arm: `5 -7 roll` is
// `5 -2 roll`, TestRoll's first case.
func TestNegativeRollByMoreThanNTakesTheRemainder(t *testing.T) {
	newType4Tester(t, "1 2 3 4 5 5 -7 roll").
		popInt(2).popInt(1).popInt(5).popInt(4).popInt(3).isEmpty()
}

// TestRollByAMultipleOfNChangesNothing is the case the modulo leaves at zero.
// `3 3 roll` and `3 -3 roll` are whole turns.
func TestRollByAMultipleOfNChangesNothing(t *testing.T) {
	newType4Tester(t, "1 2 3 3 3 roll").
		popInt(3).popInt(2).popInt(1).isEmpty()
	newType4Tester(t, "1 2 3 3 -3 roll").
		popInt(3).popInt(2).popInt(1).isEmpty()
}

// TestRollOfNothingIsNothing is the n == 0 arm, which the modulo needs taken
// before it: a roll of no elements leaves the stack alone whatever j is. The
// Java reached the division with n at zero without the early return, and the
// port popped j elements it had not been given.
func TestRollOfNothingIsNothing(t *testing.T) {
	newType4Tester(t, "1 2 3 0 5 roll").
		popInt(3).popInt(2).popInt(1).isEmpty()
}

// TestRollStillRefusesANegativeCount is the guard beside the fix, and it passes
// either way: a negative n is a rangecheck in both trees, and the early return
// must not swallow it.
func TestRollStillRefusesANegativeCount(t *testing.T) {
	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("a negative roll count was accepted")
		}
		if got, want := recovered, "rangecheck: -1"; got != want {
			t.Errorf("panicked with %v, want %q", got, want)
		}
	}()
	newType4Tester(t, "1 2 3 -1 1 roll")
}
