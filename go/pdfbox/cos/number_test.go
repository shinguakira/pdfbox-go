package cos

import (
	"errors"
	"math"
	"testing"
)

// Ported from pdfbox/src/test/java/org/apache/pdfbox/cos/TestCOSNumber.java.
//
// TestCOSNumber is abstract in Java: it declares testFloatValue, testIntValue
// and testLongValue for its subclasses and carries the shared testGet,
// testLargeNumber and testInvalidNumber. The shared tests are ported here; the
// per-type ones live in integer_test.go and float_test.go.

func TestNumberGet(t *testing.T) {
	// The basic static numbers are recognised.
	// "-" and "." both yield zero; see PDFBOX-592.
	statics := []struct {
		in   string
		want *Integer
	}{
		{"0", IntegerZero},
		{"-", IntegerZero},
		{".", IntegerZero},
		{"1", IntegerOne},
		{"2", IntegerTwo},
		{"3", IntegerThree},
	}
	for _, c := range statics {
		got, err := GetNumber(c.in)
		if err != nil {
			t.Fatalf("GetNumber(%q): %v", c.in, err)
		}
		if got != Base(c.want) {
			t.Errorf("GetNumber(%q) = %v, want %v", c.in, got, c.want)
		}
	}

	// Arbitrary integers.
	ints := []struct {
		in   string
		want int64
	}{
		{"100", 100},
		{"256", 256},
		{"-1000", -1000},
		{"+2000", 2000},
	}
	for _, c := range ints {
		got, err := GetNumber(c.in)
		if err != nil {
			t.Fatalf("GetNumber(%q): %v", c.in, err)
		}
		n, ok := got.(*Integer)
		if !ok {
			t.Fatalf("GetNumber(%q) = %T, want *Integer", c.in, got)
		}
		if n.LongValue() != c.want {
			t.Errorf("GetNumber(%q) = %d, want %d", c.in, n.LongValue(), c.want)
		}
	}

	// Arbitrary floats.
	floats := []struct {
		in   string
		want float32
	}{
		{"1.1", 1.1},
		{"100.0", 100},
		{"-100.001", -100.001},
	}
	for _, c := range floats {
		got, err := GetNumber(c.in)
		if err != nil {
			t.Fatalf("GetNumber(%q): %v", c.in, err)
		}
		f, ok := got.(*Float)
		if !ok {
			t.Fatalf("GetNumber(%q) = %T, want *Float", c.in, got)
		}
		if f.FloatValue() != c.want {
			t.Errorf("GetNumber(%q) = %v, want %v", c.in, f.FloatValue(), c.want)
		}
	}

	// The spec says exponential notation shall not be used, but it occurs.
	for _, in := range []string{"-2e-006", "-8e+05"} {
		got, err := GetNumber(in)
		if err != nil {
			t.Errorf("GetNumber(%q): %v", in, err)
		}
		if got == nil {
			t.Errorf("GetNumber(%q) = nil, want a number", in)
		}
	}

	// Java asserts NullPointerException for null; Go has no null string, so
	// the empty string stands in as the degenerate input.
	if _, err := GetNumber(""); err == nil {
		t.Error("GetNumber(\"\") succeeded, want an error")
	}
	if _, err := GetNumber("a"); err == nil {
		t.Error("GetNumber(\"a\") succeeded, want an error")
	}
}

// TestNumberLargeNumber covers PDFBOX-5176: a number too big for an int64
// yields an Integer marked invalid rather than an error.
func TestNumberLargeNumber(t *testing.T) {
	cases := []struct {
		in    string
		valid bool
	}{
		{"9223372036854775807", true},   // math.MaxInt64
		{"-9223372036854775808", true},  // math.MinInt64
		{"18446744073307448448", false}, // out of range, max
		{"-18446744073307448448", false},
	}
	for _, c := range cases {
		got, err := GetNumber(c.in)
		if err != nil {
			t.Fatalf("GetNumber(%q): %v", c.in, err)
		}
		n, ok := got.(*Integer)
		if !ok {
			t.Fatalf("GetNumber(%q) = %T, want *Integer", c.in, got)
		}
		if n.IsValid() != c.valid {
			t.Errorf("GetNumber(%q).IsValid() = %v, want %v", c.in, n.IsValid(), c.valid)
		}
	}

	// the in-range extremes round-trip
	got, _ := GetNumber("9223372036854775807")
	if v := got.(*Integer).LongValue(); v != math.MaxInt64 {
		t.Errorf("max = %d, want %d", v, int64(math.MaxInt64))
	}
	got, _ = GetNumber("-9223372036854775808")
	if v := got.(*Integer).LongValue(); v != math.MinInt64 {
		t.Errorf("min = %d, want %d", v, int64(math.MinInt64))
	}
}

func TestNumberInvalidNumber(t *testing.T) {
	if _, err := GetNumber("18446744073307F448448"); err == nil {
		t.Error("GetNumber of a non-number succeeded, want an error")
	} else if !errors.Is(err, ErrNotANumber) {
		t.Errorf("error = %v, want ErrNotANumber", err)
	}
}
