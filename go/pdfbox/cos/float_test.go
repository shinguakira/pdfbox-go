package cos

import (
	"bytes"
	"math"
	"testing"
)

// Ported from pdfbox/src/test/java/org/apache/pdfbox/cos/TestCOSFloat.java.
//
// The Java file drives its sweeps through nested BaseTester subclasses; the
// port uses plain loops over the same ranges.

func floatSweep(yield func(f float32)) {
	for i := -1000; i < 3000; i += 200 {
		yield(float32(i))
	}
}

func TestFloatBaseContract(t *testing.T) {
	assertBaseContract(t, NewFloat(0))
}

func TestFloatEquals(t *testing.T) {
	floatSweep(func(f float32) {
		test1, test2, test3 := NewFloat(f), NewFloat(f), NewFloat(f)

		if !test1.Equals(test1) {
			t.Errorf("%v: not reflexive", f)
		}
		if !test2.Equals(test3) || !test3.Equals(test2) {
			t.Errorf("%v: not symmetric", f)
		}
		if !test1.Equals(test2) || !test2.Equals(test3) || !test1.Equals(test3) {
			t.Errorf("%v: not transitive", f)
		}
	})
}

func TestFloatFloatValue(t *testing.T) {
	floatSweep(func(f float32) {
		if got := NewFloat(f).FloatValue(); got != f {
			t.Errorf("NewFloat(%v).FloatValue() = %v, want %v", f, got, f)
		}
	})
}

func TestFloatIntValue(t *testing.T) {
	floatSweep(func(f float32) {
		if got := NewFloat(f).IntValue(); got != int(f) {
			t.Errorf("NewFloat(%v).IntValue() = %d, want %d", f, got, int(f))
		}
	})
}

func TestFloatLongValue(t *testing.T) {
	floatSweep(func(f float32) {
		if got := NewFloat(f).LongValue(); got != int64(f) {
			t.Errorf("NewFloat(%v).LongValue() = %d, want %d", f, got, int64(f))
		}
	})
}

func TestFloatAccept(t *testing.T) {
	// The emitted bytes are asserted in accept_external_test.go.
	assertVisits(t, NewFloat(1.5), "float")
}

func TestFloatWritePDF(t *testing.T) {
	check := func(f float32) {
		t.Helper()
		cf := NewFloat(f)
		var buf bytes.Buffer
		if err := cf.WritePDF(&buf); err != nil {
			t.Fatalf("WritePDF(%v): %v", f, err)
		}
		want := formatFloatForTest(f)
		if got := buf.String(); got != want {
			t.Errorf("WritePDF(%v) = %q, want %q", f, got, want)
		}
		if got, wantStr := cf.String(), "COSFloat{"+want+"}"; got != wantStr {
			t.Errorf("String() = %q, want %q", got, wantStr)
		}
	}
	floatSweep(check)
	// corner case from PDFBOX-1778
	check(0.000000000000000000000000000000001)
}

// formatFloatForTest mirrors the floatToString helper in the Java test: a plain
// (non-exponential) decimal with trailing fraction zeros removed, but never
// trimmed past a single ".0".
func formatFloatForTest(f float32) string {
	s := plainFloatString(f)
	if i := indexByteStr(s, '.'); i >= 0 && !hasSuffixStr(s, ".0") {
		for hasSuffixStr(s, "0") && !hasSuffixStr(s, ".0") {
			s = s[:len(s)-1]
		}
	}
	return s
}

func indexByteStr(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func hasSuffixStr(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

// TestFloatDoubleNegative covers PDFBOX-4289, which has --16.33.
func TestFloatDoubleNegative(t *testing.T) {
	cf, err := ParseFloat("--16.33")
	if err != nil {
		t.Fatalf("ParseFloat: %v", err)
	}
	if got := cf.FloatValue(); got != -16.33 {
		t.Errorf("FloatValue() = %v, want -16.33", got)
	}
}

// TestFloatVerySmallValues checks that values below the smallest normal float
// become zero, per the PDF spec, Appendix C implementation limits.
func TestFloatVerySmallValues(t *testing.T) {
	for _, s := range []string{
		"1.4012984643248171E-46",
		"0.00000000000000000000000000000000000000000000014012984643248171",
		"-1.4012984643248171E-46",
		"-0.00000000000000000000000000000000000000000000014012984643248171",
	} {
		cf, err := ParseFloat(s)
		if err != nil {
			t.Fatalf("ParseFloat(%q): %v", s, err)
		}
		if got := cf.FloatValue(); got != 0 {
			t.Errorf("ParseFloat(%q).FloatValue() = %v, want 0", s, got)
		}
	}
}

// TestFloatVeryLargeValues checks that values beyond the float range are
// coerced to the largest representable float rather than becoming infinity.
func TestFloatVeryLargeValues(t *testing.T) {
	const maxFloat32 = math.MaxFloat32

	for _, c := range []struct {
		in   string
		want float32
	}{
		{"3.4028234663852886E39", maxFloat32},
		{"340282346638528860000000000000000000000000", maxFloat32},
		{"-3.4028234663852886E39", -maxFloat32},
		{"-340282346638528860000000000000000000000000", -maxFloat32},
	} {
		cf, err := ParseFloat(c.in)
		if err != nil {
			t.Fatalf("ParseFloat(%q): %v", c.in, err)
		}
		if got := cf.FloatValue(); got != c.want {
			t.Errorf("ParseFloat(%q).FloatValue() = %v, want %v", c.in, got, c.want)
		}
	}
}

// TestFloatMisplacedNegative covers PDFBOX-2990, PDFBOX-3369 (0.00000-33917698),
// PDFBOX-3500 (0.-262) and PDFBOX-5829 (-12.-1). Producers really emit these.
func TestFloatMisplacedNegative(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"0.00000-33917698", "-0.0000033917698"},
		{"0.-262", "-0.262"},
		{"-0.-262", "-0.262"},
		{"-12.-1", "-12.1"},
	}
	for _, c := range cases {
		got, err := ParseFloat(c.in)
		if err != nil {
			t.Fatalf("ParseFloat(%q): %v", c.in, err)
		}
		want, err := ParseFloat(c.want)
		if err != nil {
			t.Fatalf("ParseFloat(%q): %v", c.want, err)
		}
		if !got.Equals(want) {
			t.Errorf("ParseFloat(%q) = %v, want %v", c.in, got.FloatValue(), want.FloatValue())
		}
	}
}

func TestFloatDuplicateMisplacedNegative(t *testing.T) {
	for _, in := range []string{"0.-26-2", "---0.262", "--0.2-62"} {
		if _, err := ParseFloat(in); err == nil {
			t.Errorf("ParseFloat(%q) succeeded, want an error", in)
		}
	}
}

func TestFloatStubOperatorMinMaxValues(t *testing.T) {
	if got := NewFloat(32768).FloatValue(); got != 32768 {
		t.Errorf("FloatValue() = %v, want 32768", got)
	}
	if got := NewFloat(-32768).FloatValue(); got != -32768 {
		t.Errorf("FloatValue() = %v, want -32768", got)
	}
}

// TestFloatIntValueSaturates pins Java's (int) and (long) narrowing casts on a
// float, which clamp rather than wrap: a value above the target range becomes
// MAX_VALUE and one below becomes MIN_VALUE. Go leaves an out-of-range float
// conversion undefined, so the clamping has to be written explicitly.
func TestFloatIntValueSaturates(t *testing.T) {
	cases := []struct {
		in       float32
		wantInt  int
		wantLong int64
	}{
		{0, 0, 0},
		{1.9, 1, 1}, // truncates toward zero
		{-1.9, -1, -1},
		{math.MaxFloat32, math.MaxInt32, math.MaxInt64},
		{-math.MaxFloat32, math.MinInt32, math.MinInt64},
	}
	for _, c := range cases {
		f := NewFloat(c.in)
		if got := f.IntValue(); got != c.wantInt {
			t.Errorf("NewFloat(%v).IntValue() = %d, want %d", c.in, got, c.wantInt)
		}
		if got := f.LongValue(); got != c.wantLong {
			t.Errorf("NewFloat(%v).LongValue() = %d, want %d", c.in, got, c.wantLong)
		}
	}
}
