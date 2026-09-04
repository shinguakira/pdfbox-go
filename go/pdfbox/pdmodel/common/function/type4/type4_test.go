package type4

import (
	"math"
	"testing"
)

// Port of
// pdfbox/src/test/java/org/apache/pdfbox/pdmodel/common/function/type4/TestOperators.java
// and TestParser.java, with Type4Tester.java as the helper below. Every
// expected value is the Java's.
//
// Java's Type4Tester overloads pop for a boolean, an int and a float, and has
// popReal beside them; Go has no overloading, so each becomes a method whose
// name says which type it asserts. The distinction matters: popReal and popInt
// check that the value on the stack really is a real or an integer, which half
// these tests are about, and popNumber accepts either.

// type4Tester runs a program and checks what it left on the stack.
type type4Tester struct {
	t       *testing.T
	context *ExecutionContext
}

func newType4Tester(t *testing.T, text string) *type4Tester {
	t.Helper()
	instructions := Parse(text)
	context := NewExecutionContext(NewOperators())
	instructions.Execute(context)
	return &type4Tester{t: t, context: context}
}

func (x *type4Tester) popBool(expected bool) *type4Tester {
	x.t.Helper()
	value, ok := x.context.Pop().(bool)
	if !ok {
		x.t.Fatal("expected a boolean on the stack")
	}
	if value != expected {
		x.t.Errorf("popped %v, want %v", value, expected)
	}
	return x
}

func (x *type4Tester) popReal(expected float32) *type4Tester {
	x.t.Helper()
	return x.popRealDelta(expected, 0.0000001)
}

func (x *type4Tester) popRealDelta(expected float32, delta float64) *type4Tester {
	x.t.Helper()
	value, ok := x.context.Pop().(float32)
	if !ok {
		x.t.Fatal("expected a real on the stack")
	}
	if math.Abs(float64(value)-float64(expected)) > delta {
		x.t.Errorf("popped %v, want %v", value, expected)
	}
	return x
}

func (x *type4Tester) popInt(expected int32) *type4Tester {
	x.t.Helper()
	value := x.context.PopInt()
	if value != expected {
		x.t.Errorf("popped %v, want %v", value, expected)
	}
	return x
}

// popNumber is Java's pop(float), which takes either an integer or a real off
// the stack and compares as a double.
func (x *type4Tester) popNumber(expected float64) *type4Tester {
	x.t.Helper()
	return x.popNumberDelta(expected, 0.0000001)
}

func (x *type4Tester) popNumberDelta(expected, delta float64) *type4Tester {
	x.t.Helper()
	value := numberAsDouble(x.context.PopNumber())
	if math.Abs(value-expected) > delta {
		x.t.Errorf("popped %v, want %v", value, expected)
	}
	return x
}

func (x *type4Tester) isEmpty() *type4Tester {
	x.t.Helper()
	if !x.context.IsEmpty() {
		x.t.Errorf("the stack is not empty: %v", x.context.Stack())
	}
	return x
}

// mustPanic runs f and fails unless it panics, which is what the port does
// where Java throws an unchecked exception.
func mustPanic(t *testing.T, what string, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("%s", what)
		}
	}()
	f()
}

func TestAdd(t *testing.T) {
	newType4Tester(t, "5 6 add").popInt(11).isEmpty()
	newType4Tester(t, "5 0.23 add").popNumber(5.23).isEmpty()
	bigValue := int64(math.MaxInt32) - 2
	context := newType4Tester(t, itoa(bigValue)+" "+itoa(bigValue)+" add").context
	floatResult, ok := context.Pop().(float32)
	if !ok {
		t.Fatal("an integer overflow should give a real")
	}
	// Java writes assertEquals(2 * (long) Integer.MAX_VALUE - 4, floatResult, 1),
	// and its overload resolution picks assertEquals(float, float, float) over
	// the double one, so the expected long is narrowed to a float before the
	// comparison. That matters: 4294967290 has no float32 of its own and both
	// sides land on 4294967296, where comparing in double would be 6 out and
	// fail the delta of 1.
	want := float32(2*int64(math.MaxInt32) - 4)
	if math.Abs(float64(floatResult)-float64(want)) > 1 {
		t.Errorf("popped %v, want %v", floatResult, want)
	}
	if !context.IsEmpty() {
		t.Error("the stack is not empty")
	}
}

func itoa(v int64) string {
	if v == 0 {
		return "0"
	}
	negative := v < 0
	if negative {
		v = -v
	}
	var digits []byte
	for v > 0 {
		digits = append([]byte{byte('0' + v%10)}, digits...)
		v /= 10
	}
	if negative {
		return "-" + string(digits)
	}
	return string(digits)
}

func TestAbs(t *testing.T) {
	newType4Tester(t, "-3 abs 2.1 abs -2.1 abs -7.5 abs").
		popNumber(7.5).popNumber(2.1).popNumber(2.1).popInt(3).isEmpty()
}

func TestAnd(t *testing.T) {
	newType4Tester(t, "true true and true false and").
		popBool(false).popBool(true).isEmpty()
	newType4Tester(t, "99 1 and 52 7 and").
		popInt(4).popInt(1).isEmpty()
}

func TestAtan(t *testing.T) {
	newType4Tester(t, "0 1 atan").popNumber(0).isEmpty()
	newType4Tester(t, "1 0 atan").popNumber(90).isEmpty()
	newType4Tester(t, "-100 0 atan").popNumber(270).isEmpty()
	newType4Tester(t, "4 4 atan").popNumber(45).isEmpty()
}

func TestCeiling(t *testing.T) {
	newType4Tester(t, "3.2 ceiling -4.8 ceiling 99 ceiling").
		popInt(99).popNumber(-4).popNumber(4).isEmpty()
}

func TestCos(t *testing.T) {
	newType4Tester(t, "0 cos").popReal(1).isEmpty()
	newType4Tester(t, "90 cos").popReal(0).isEmpty()
}

func TestCvi(t *testing.T) {
	newType4Tester(t, "-47.8 cvi").popInt(-47).isEmpty()
	newType4Tester(t, "520.9 cvi").popInt(520).isEmpty()
}

func TestCvr(t *testing.T) {
	newType4Tester(t, "-47.8 cvr").popReal(-47.8).isEmpty()
	newType4Tester(t, "520.9 cvr").popReal(520.9).isEmpty()
	newType4Tester(t, "77 cvr").popReal(77).isEmpty()
	// Check that the data types are really right
	context := newType4Tester(t, "77 77 cvr").context
	if _, ok := context.Pop().(float32); !ok {
		t.Error("Expected a real as the result of 'cvr'")
	}
	if _, ok := context.Pop().(int32); !ok {
		t.Error("Expected an int from an integer literal")
	}
}

func TestDiv(t *testing.T) {
	newType4Tester(t, "3 2 div").popReal(1.5).isEmpty()
	newType4Tester(t, "4 2 div").popReal(2.0).isEmpty()
}

func TestExp(t *testing.T) {
	newType4Tester(t, "9 0.5 exp").popReal(3.0).isEmpty()
	newType4Tester(t, "-9 -1 exp").popRealDelta(-0.111111, 0.000001).isEmpty()
}

func TestFloor(t *testing.T) {
	newType4Tester(t, "3.2 floor -4.8 floor 99 floor").
		popInt(99).popNumber(-5).popNumber(3).isEmpty()
}

func TestIDiv(t *testing.T) {
	newType4Tester(t, "3 2 idiv").popInt(1).isEmpty()
	newType4Tester(t, "4 2 idiv").popInt(2).isEmpty()
	newType4Tester(t, "-5 2 idiv").popInt(-2).isEmpty()
	mustPanic(t, "Expected typecheck", func() { newType4Tester(t, "4.4 2 idiv") })
}

func TestLn(t *testing.T) {
	newType4Tester(t, "10 ln").popRealDelta(2.30259, 0.00001).isEmpty()
	newType4Tester(t, "100 ln").popRealDelta(4.60517, 0.00001).isEmpty()
}

func TestLog(t *testing.T) {
	newType4Tester(t, "10 log").popReal(1.0).isEmpty()
	newType4Tester(t, "100 log").popReal(2.0).isEmpty()
}

func TestMod(t *testing.T) {
	newType4Tester(t, "5 3 mod").popInt(2).isEmpty()
	newType4Tester(t, "5 2 mod").popInt(1).isEmpty()
	newType4Tester(t, "-5 3 mod").popInt(-2).isEmpty()
	mustPanic(t, "Expected typecheck", func() { newType4Tester(t, "4.4 2 mod") })
}

func TestMul(t *testing.T) {
	newType4Tester(t, "1 2 mul").popInt(2).isEmpty()
	newType4Tester(t, "1.5 2 mul").popReal(3.0).isEmpty()
	newType4Tester(t, "1.5 2.1 mul").popRealDelta(3.15, 0.001).isEmpty()
	newType4Tester(t, itoa(int64(math.MaxInt32)-3)+" 2 mul"). // integer overflow -> real
									popRealDelta(float32(2*(int64(math.MaxInt32)-3)), 0.001).isEmpty()
}

func TestNeg(t *testing.T) {
	newType4Tester(t, "4.5 neg").popReal(-4.5).isEmpty()
	newType4Tester(t, "-3 neg").popInt(3).isEmpty()
	// Border cases
	newType4Tester(t, itoa(int64(math.MinInt32)+1)+" neg").
		popInt(math.MaxInt32).isEmpty()
	newType4Tester(t, itoa(int64(math.MinInt32))+" neg").
		popReal(-float32(math.MinInt32)).isEmpty()
}

func TestRound(t *testing.T) {
	newType4Tester(t, "3.2 round").popReal(3.0).isEmpty()
	newType4Tester(t, "6.5 round").popReal(7.0).isEmpty()
	newType4Tester(t, "-4.8 round").popReal(-5.0).isEmpty()
	newType4Tester(t, "-6.5 round").popReal(-6.0).isEmpty()
	newType4Tester(t, "99 round").popInt(99).isEmpty()
}

func TestSin(t *testing.T) {
	newType4Tester(t, "0 sin").popReal(0).isEmpty()
	newType4Tester(t, "90 sin").popReal(1).isEmpty()
	newType4Tester(t, "-90.0 sin").popReal(-1).isEmpty()
}

func TestSqrt(t *testing.T) {
	newType4Tester(t, "0 sqrt").popReal(0).isEmpty()
	newType4Tester(t, "1 sqrt").popReal(1).isEmpty()
	newType4Tester(t, "4 sqrt").popReal(2).isEmpty()
	newType4Tester(t, "4.4 sqrt").popRealDelta(2.097617, 0.000001).isEmpty()
	mustPanic(t, "Expected rangecheck", func() { newType4Tester(t, "-4.1 sqrt") })
}

func TestSub(t *testing.T) {
	newType4Tester(t, "5 2 sub -7.5 1 sub").popNumber(-8.5).popInt(3).isEmpty()
}

func TestTruncate(t *testing.T) {
	newType4Tester(t, "3.2 truncate").popReal(3.0).isEmpty()
	newType4Tester(t, "-4.8 truncate").popReal(-4.0).isEmpty()
	newType4Tester(t, "99 truncate").popInt(99).isEmpty()
}

func TestBitshift(t *testing.T) {
	newType4Tester(t, "7 3 bitshift 142 -3 bitshift").
		popInt(17).popInt(56).isEmpty()
}

func TestEq(t *testing.T) {
	newType4Tester(t, "7 7 eq 7 6 eq 7 -7 eq true true eq false true eq 7.7 7.7 eq").
		popBool(true).popBool(false).popBool(true).popBool(false).
		popBool(false).popBool(true).isEmpty()
}

func TestGe(t *testing.T) {
	newType4Tester(t, "5 7 ge 7 5 ge 7 7 ge -1 2 ge").
		popBool(false).popBool(true).popBool(true).popBool(false).isEmpty()
}

func TestGt(t *testing.T) {
	newType4Tester(t, "5 7 gt 7 5 gt 7 7 gt -1 2 gt").
		popBool(false).popBool(false).popBool(true).popBool(false).isEmpty()
}

func TestLe(t *testing.T) {
	newType4Tester(t, "5 7 le 7 5 le 7 7 le -1 2 le").
		popBool(true).popBool(true).popBool(false).popBool(true).isEmpty()
}

func TestLt(t *testing.T) {
	newType4Tester(t, "5 7 lt 7 5 lt 7 7 lt -1 2 lt").
		popBool(true).popBool(false).popBool(false).popBool(true).isEmpty()
}

func TestNe(t *testing.T) {
	newType4Tester(t, "7 7 ne 7 6 ne 7 -7 ne true true ne false true ne 7.7 7.7 ne").
		popBool(false).popBool(true).popBool(false).popBool(true).
		popBool(true).popBool(false).isEmpty()
}

// TestNot carries JAVA-BUGS: the integer arm of `not` is the arithmetic
// negation in PDFBox, where PostScript defines it as the bitwise complement.
// The values below are the Java test's, so 52 not is -52 and not -53.
func TestNot(t *testing.T) {
	newType4Tester(t, "true not false not").
		popBool(true).popBool(false).isEmpty()
	newType4Tester(t, "52 not -37 not").
		popInt(37).popInt(-52).isEmpty()
}

func TestOr(t *testing.T) {
	newType4Tester(t, "true true or true false or false false or").
		popBool(false).popBool(true).popBool(true).isEmpty()
	newType4Tester(t, "17 5 or 1 1 or").
		popInt(1).popInt(21).isEmpty()
}

func TestXor(t *testing.T) {
	newType4Tester(t, "true true xor true false xor false false xor").
		popBool(false).popBool(true).popBool(false).isEmpty()
	newType4Tester(t, "7 3 xor 12 3 or").
		popInt(15).popInt(4)
}

func TestIf(t *testing.T) {
	newType4Tester(t, "true { 2 1 add } if").popInt(3).isEmpty()
	newType4Tester(t, "false { 2 1 add } if").isEmpty()
	mustPanic(t, "Need typecheck error for the '0'",
		func() { newType4Tester(t, "0 { 2 1 add } if") })
}

func TestIfElse(t *testing.T) {
	newType4Tester(t, "true { 2 1 add } { 2 1 sub } ifelse").popInt(3).isEmpty()
	newType4Tester(t, "false { 2 1 add } { 2 1 sub } ifelse").popInt(1).isEmpty()
}

func TestCopy(t *testing.T) {
	newType4Tester(t, "true 1 2 3 3 copy").
		popInt(3).popInt(2).popInt(1).
		popInt(3).popInt(2).popInt(1).
		popBool(true).
		isEmpty()
}

func TestDup(t *testing.T) {
	newType4Tester(t, "true 1 2 dup").
		popInt(2).popInt(2).popInt(1).
		popBool(true).
		isEmpty()
	newType4Tester(t, "true dup").popBool(true).popBool(true).isEmpty()
}

func TestExch(t *testing.T) {
	newType4Tester(t, "true 1 exch").popBool(true).popInt(1).isEmpty()
	newType4Tester(t, "1 2.5 exch").popInt(1).popNumber(2.5).isEmpty()
}

func TestIndex(t *testing.T) {
	newType4Tester(t, "1 2 3 4 0 index").
		popInt(4).popInt(4).popInt(3).popInt(2).popInt(1).isEmpty()
	newType4Tester(t, "1 2 3 4 3 index").
		popInt(1).popInt(4).popInt(3).popInt(2).popInt(1).isEmpty()
}

func TestPop(t *testing.T) {
	newType4Tester(t, "1 pop 7 2 pop").popInt(7).isEmpty()
	newType4Tester(t, "1 2 3 pop pop").popInt(1).isEmpty()
}

func TestRoll(t *testing.T) {
	newType4Tester(t, "1 2 3 4 5 5 -2 roll").
		popInt(2).popInt(1).popInt(5).popInt(4).popInt(3).isEmpty()
	newType4Tester(t, "1 2 3 4 5 5 2 roll").
		popInt(3).popInt(2).popInt(1).popInt(5).popInt(4).isEmpty()
	newType4Tester(t, "1 2 3 3 0 roll").
		popInt(3).popInt(2).popInt(1).isEmpty()
}

// Port of TestParser.java.

func TestParserBasics(t *testing.T) {
	newType4Tester(t, "3 4 add 2 sub").popInt(5).isEmpty()
}

func TestNested(t *testing.T) {
	newType4Tester(t, "true { 2 1 add } { 2 1 sub } ifelse").popInt(3).isEmpty()
	newType4Tester(t, "{ true }").popBool(true).isEmpty()
}

func TestParseFloat(t *testing.T) {
	cases := []struct {
		token string
		want  float64
	}{
		{"0", 0},
		{"1", 1},
		{"+1", 1},
		{"-1", -1},
		{"3.14157", 3.14157},
		{"-1.2", -1.2},
		{"1.0E-5", 1.0e-5},
	}
	for _, c := range cases {
		if got := float64(ParseReal(c.token)); math.Abs(got-c.want) > 0.00001 {
			t.Errorf("ParseReal(%q) = %v, want %v", c.token, got, c.want)
		}
	}
}

// TestJira804 is an example of a tint to CMYK function. Problems here were:
//  1. no whitespace between "mul" and "}" (token was detected as "mul}")
//  2. line breaks cause endless loops
func TestJira804(t *testing.T) {
	newType4Tester(t, "1 {dup dup .72 mul exch 0 exch .38 mul}\n").
		popNumber(0.38).popNumber(0).popNumber(0.72).popNumber(1.0).isEmpty()
}
