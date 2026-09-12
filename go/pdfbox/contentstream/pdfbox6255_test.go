package contentstream_test

// PDFBOX-6255, Apache 4a42d294e, in the sync of 2026-09-07.
//
// `Matrix.concatenate` raises the unchecked IllegalArgumentException when the
// product runs off the end of the float range, and until this commit nothing
// in PDFBox caught it: a `cm` whose operands overflow left the exception to
// come out of the parse as a RuntimeException, past every caller written to
// handle IOException. `cm` now catches it and rethrows it as an IOException,
// which is a failure the callers already know how to take.
//
// The port raised the same thing as a panic, and for the same reason: Java's
// exception is unchecked, so putting an error return on every arithmetic
// method of Matrix would have been a deviation with no caller to justify it.
// This commit gives it the one caller.

import (
	"errors"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// bigScale is 3e20, written out in full because a content stream is not a COS
// object: the tokenizer takes digits, one dot and a minus, and stops at the
// `e`. It is an ordinary real that any file may carry, being well inside
// float32; two of them are not, since 9e40 is past MaxFloat32.
const bigScale = "300000000000000000000.0"

// TestConcatenateOverflowIsAnErrorNotAPanic is the operator's half. Before this
// commit the second `cm` panicked out of ProcessPage.
func TestConcatenateOverflowIsAnErrorNotAPanic(t *testing.T) {
	r := newRecorder()
	err := r.ProcessPage(pageWith(t,
		bigScale+" 0 0 1 0 0 cm "+bigScale+" 0 0 1 0 0 cm"))
	if err == nil {
		t.Fatal("a cm that overflows the float range was accepted")
	}
	if !errors.Is(err, util.ErrIllegalMatrixValues) {
		t.Errorf("ProcessPage err = %v, want one wrapping %v",
			err, util.ErrIllegalMatrixValues)
	}
}

// TestConcatenateOverflowEndsTheWalk pins which side of operatorException the
// new error lands on. Java throws it out of Concatenate.process as an
// IOException, and PDFStreamEngine.operatorException rethrows anything that is
// not one of the four failures it names, so the rest of the stream is not
// walked.
func TestConcatenateOverflowEndsTheWalk(t *testing.T) {
	r := newRecorder()
	err := r.ProcessPage(pageWith(t,
		bigScale+" 0 0 1 0 0 cm "+bigScale+" 0 0 1 0 0 cm BT ET"))
	if err == nil {
		t.Fatal("a cm that overflows the float range was accepted")
	}
	for _, seen := range r.seen {
		if seen == "BT" || seen == "ET" {
			t.Errorf("the walk went on to %s after the cm that overflowed", seen)
		}
	}
}

// TestConcatenateThatFitsIsUnaffected is the guard on the recover: it must take
// the one panic it is there for and leave the ordinary path alone.
//
// It is a guard rather than a test of the change: it passes with the fix
// reverted, which is the point of it.
func TestConcatenateThatFitsIsUnaffected(t *testing.T) {
	r := newRecorder()
	r.run(t, bigScale+" 0 0 1 0 0 cm")
	if got := r.probed.CurrentTransformationMatrix().ScaleX(); got != 3e20 {
		t.Errorf("the CTM scales x by %v, want 3e20", got)
	}
}

// TestMatrixArithmeticStillPanics is the other half: only `cm` catches this in
// Java, so every other caller of Matrix.concatenate keeps the unchecked
// failure. A port that turned the panic into an error return everywhere would
// have deviated from the Java in every one of them.
//
// A guard too: it passes with the operator change reverted, and would stop
// passing if some later branch made Matrix report rather than panic.
func TestMatrixArithmeticStillPanics(t *testing.T) {
	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("Matrix.Concatenate did not panic on an overflow")
		}
		failure, isError := recovered.(error)
		if !isError || !errors.Is(failure, util.ErrIllegalMatrixValues) {
			t.Errorf("panicked with %v, want %v", recovered, util.ErrIllegalMatrixValues)
		}
	}()
	m := util.NewMatrixOf(3e20, 0, 0, 1, 0, 0)
	m.Concatenate(util.NewMatrixOf(3e20, 0, 0, 1, 0, 0))
}
