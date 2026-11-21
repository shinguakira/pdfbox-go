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

// TestEvalOfAProgramWhoseCastFailsIsAnError is the rest of the
// ClassCastException arm, found in review of the commit above.
//
// ExecutionContext's accessors raise type4.ErrWrongType, but five casts in the
// operators were plain Go type assertions -- bitshift's two operands, and the
// procedures and conditions of `if` and `ifelse`. Those panic with
// *runtime.TypeAssertionError, which is an error and wraps nothing, so the
// recover let them past and the page died for a program the Java loses only the
// shading of.
// One program per cast, because a program only reaches the cast its own
// operand order puts in the way: `{ true 1 bitshift }` has the wrong type under
// the shift and `{ 1 true bitshift }` has it in the shift itself.
func TestEvalOfAProgramWhoseCastFailsIsAnError(t *testing.T) {
	for _, program := range []string{
		"{ true 1 bitshift }",  // bitshift's second operand
		"{ 1 true bitshift }",  // bitshift's shift
		"{ 1 {} if }",          // if's condition
		"{ true 1 if }",        // if's procedure
		"{ 1 {} {} ifelse }",   // ifelse's condition
		"{ true {} 1 ifelse }", // ifelse's second procedure
		"{ true 1 {} ifelse }", // ifelse's first procedure
	} {
		f := createFunction(t, program, []float32{0, 1}, []float32{0, 1})

		output, err := f.Eval([]float32{0.5})
		if err == nil {
			t.Errorf("%s was accepted, output %v", program, output)
			continue
		}
		if !errors.Is(err, type4.ErrWrongType) {
			t.Errorf("%s: Eval err = %v, want one wrapping %v",
				program, err, type4.ErrWrongType)
		}
	}
}
