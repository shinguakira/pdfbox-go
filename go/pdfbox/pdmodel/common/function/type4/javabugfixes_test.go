package type4

// JAVA-BUGS 29: the type 4 `not` operator negates an integer instead of
// complementing it.

import "testing"

// TestNotOfAnIntegerIsTheComplement is the defect.
//
// PostScript's `not` is a logical negation for a boolean and a **bitwise
// complement** for an integer — PDF 32000-1:2008 table 42, and the PostScript
// Language Reference before it. Java writes `-int1`, which is the arithmetic
// negation: the two agree for no value at all except that both leave 0 and -1
// swapped in a way that looks right.
//
// The expected values are the complement: ~0 is -1, ~1 is -2, ~-1 is 0.
func TestNotOfAnIntegerIsTheComplement(t *testing.T) {
	for _, c := range []struct{ in, want int32 }{
		{0, -1},
		{1, -2},
		{-1, 0},
		{5, -6},
	} {
		context := NewExecutionContext(NewOperators())
		context.Push(c.in)
		notOperator{}.Execute(context)
		got, isInt := context.Pop().(int32)
		if !isInt {
			t.Fatalf("not %d answered something that is not an integer", c.in)
		}
		if got != c.want {
			t.Errorf("not %d is %d, want %d (the bitwise complement)", c.in, got, c.want)
		}
	}

	// A boolean is still a logical negation.
	context := NewExecutionContext(NewOperators())
	context.Push(true)
	notOperator{}.Execute(context)
	if got := context.Pop(); got != false {
		t.Errorf("not true is %v, want false", got)
	}
}
