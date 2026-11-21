package function_test

// PDFBOX-6320, Apache 7611f9329, in the sync of 2026-11-21, and JAVA-BUGS.md 91
// beside it.
//
// eval declares `throws IOException` and every caller on the paint path handles
// one, but the program it runs could raise two unchecked exceptions that went
// straight past them all: EmptyStackException, where an operator asks for an
// operand that is not there, and ClassCastException, where the operand has the
// wrong type. Upstream wraps both in an IOException, so a malformed function
// loses its shading rather than the page. The port panicked for the same two.
//
// The defect JAVA-BUGS.md 91 is about -- the IllegalStateException for too few
// output values, two statements below this catch -- upstream did not touch, and
// it is still a panic here. TestEvalWithTooFewOutputValuesStillPanics pins
// that, and passes either way.

import (
	"errors"
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common/function/type4"
)

// TestEvalOfAProgramThatRunsOutOfOperandsIsAnError is the EmptyStackException
// arm: `add` wants two operands and the function is given one input.
func TestEvalOfAProgramThatRunsOutOfOperandsIsAnError(t *testing.T) {
	f := createFunction(t, "{ add }", []float32{0, 1}, []float32{0, 1})

	output, err := f.Eval([]float32{0.5})
	if err == nil {
		t.Fatalf("a program that runs out of operands was accepted, output %v", output)
	}
	if !errors.Is(err, type4.ErrEmptyStack) {
		t.Errorf("Eval err = %v, want one wrapping %v", err, type4.ErrEmptyStack)
	}
}

// TestEvalOfAProgramWithAnOperandOfTheWrongTypeIsAnError is the
// ClassCastException arm: `add` is handed a boolean.
func TestEvalOfAProgramWithAnOperandOfTheWrongTypeIsAnError(t *testing.T) {
	f := createFunction(t, "{ true add }", []float32{0, 1}, []float32{0, 1})

	output, err := f.Eval([]float32{0.5})
	if err == nil {
		t.Fatalf("a program that adds a boolean was accepted, output %v", output)
	}
	if !errors.Is(err, type4.ErrWrongType) {
		t.Errorf("Eval err = %v, want one wrapping %v", err, type4.ErrWrongType)
	}
}

// TestEvalWithTooFewOutputValuesStillPanics is JAVA-BUGS.md 91, which this
// commit deliberately leaves where it is. The program leaves two values and
// /Range asks for three; the Java raises IllegalStateException, which is
// unchecked and which 7611f9329 did not catch, so the port keeps the panic.
//
// A guard, not a test of the change: it passes with the rest of this commit
// reverted.
func TestEvalWithTooFewOutputValuesStillPanics(t *testing.T) {
	f := createFunction(t, "{ dup }", []float32{0, 1}, []float32{0, 1, 0, 1, 0, 1})

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("a program that returned too few values did not panic")
		}
		message, isString := recovered.(string)
		if !isString || !strings.Contains(message, "returned 2 values") {
			t.Errorf("panicked with %v, want the Java's message about 2 values", recovered)
		}
	}()
	//nolint:errcheck // the panic is the point
	f.Eval([]float32{0.5})
}

// TestEvalOfAWholeProgramIsUnaffected is the guard on the recover: it must take
// the two failures it is there for and leave an ordinary program alone.
func TestEvalOfAWholeProgramIsUnaffected(t *testing.T) {
	f := createFunction(t, "{ add }", []float32{0, 1, 0, 1}, []float32{0, 2})

	output := mustEval(t, f, []float32{0.25, 0.5})
	if len(output) != 1 || output[0] != 0.75 {
		t.Errorf("output = %v, want [0.75]", output)
	}
}
