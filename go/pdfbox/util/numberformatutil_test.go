package util_test

// Port of org.apache.pdfbox.util.TestNumberFormatUtil.
//
// The Java class ends with a property test, testFormattingInRange, that walks a
// range of values and compares against BigDecimal with HALF_UP rounding. It is
// not ported: Go has no arbitrary-precision decimal in its standard library,
// and a re-implementation of BigDecimal to check a formatter would be checking
// the re-implementation. The five example-based cases below are what it is
// built on, and they carry the exact bytes Java asserts.

import (
	"bytes"
	"math"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// wantFormat is `assertEquals(n, formatFloatFast(v, digits, buffer))` followed
// by the array comparison over the first n bytes.
func wantFormat(t *testing.T, value float32, fractionDigits int, want string) {
	t.Helper()
	buffer := make([]byte, 64)
	got := util.FormatFloatFast(value, fractionDigits, buffer)
	if got != len(want) {
		t.Errorf("FormatFloatFast(%v, %d) wrote %d bytes, want %d (%q)",
			value, fractionDigits, got, len(want), want)
		return
	}
	if !bytes.Equal(buffer[:got], []byte(want)) {
		t.Errorf("FormatFloatFast(%v, %d) = %q, want %q",
			value, fractionDigits, buffer[:got], want)
	}
}

// TestFormatOfIntegerValues is testFormatOfIntegerValues.
func TestFormatOfIntegerValues(t *testing.T) {
	wantFormat(t, 51, 5, "51")
	wantFormat(t, -51, 5, "-51")
	wantFormat(t, 0, 5, "0")
	// (float) Long.MAX_VALUE, which is 9223372036854775807 rounded to the
	// nearest float and then printed as a whole number
	wantFormat(t, float32(math.MaxInt64), 5, "9223372036854775807")
	// (float) Integer.MAX_VALUE rounds up to 2147483648
	wantFormat(t, float32(math.MaxInt32), 5, "2147483648")
	wantFormat(t, float32(math.MinInt32), 5, "-2147483648")
}

// TestFormatOfRealValues is testFormatOfRealValues.
func TestFormatOfRealValues(t *testing.T) {
	wantFormat(t, 0.7, 5, "0.7")
	wantFormat(t, -0.7, 5, "-0.7")
	wantFormat(t, 0.003, 5, "0.003")
	wantFormat(t, -0.003, 5, "-0.003")
}

// TestFormatOfRealValuesReturnsMinusOneIfItCannotBeFormatted is
// testFormatOfRealValuesReturnsMinusOneIfItCannotBeFormatted: the fast path
// refuses rather than guessing, and the caller falls back.
func TestFormatOfRealValuesReturnsMinusOneIfItCannotBeFormatted(t *testing.T) {
	buffer := make([]byte, 64)
	for _, row := range []struct {
		name  string
		value float32
	}{
		{"NaN should not be formattable", float32(math.NaN())},
		{"+Infinity should not be formattable", float32(math.Inf(1))},
		{"-Infinity should not be formattable", float32(math.Inf(-1))},
		{"a value above the long range should not be formattable",
			float32(math.MaxInt64) + 1000000000000},
		{"Long.MIN_VALUE should not be formattable", float32(math.MinInt64)},
	} {
		t.Run(row.name, func(t *testing.T) {
			if got := util.FormatFloatFast(row.value, 5, buffer); got != -1 {
				t.Errorf("FormatFloatFast(%v, 5) = %d, want -1", row.value, got)
			}
		})
	}
}

// TestRoundingUp is testRoundingUp.
func TestRoundingUp(t *testing.T) {
	wantFormat(t, 0.999999, 5, "1")
	wantFormat(t, 0.125, 2, "0.13")
	wantFormat(t, -0.999999, 5, "-1")
}

// TestRoundingDown is testRoundingDown.
func TestRoundingDown(t *testing.T) {
	wantFormat(t, 0.994, 2, "0.99")
}
