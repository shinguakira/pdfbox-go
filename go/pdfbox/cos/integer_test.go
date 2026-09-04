package cos

import (
	"bytes"
	"math"
	"strconv"
	"testing"
)

// Ported from pdfbox/src/test/java/org/apache/pdfbox/cos/TestCOSInteger.java.
//
// Every test there walks i from -1000 to 3000 in steps of 200, which spans the
// cached range (-100 to 256) on both sides. The port keeps that sweep.

func integerSweep(yield func(i int64)) {
	for i := int64(-1000); i < 3000; i += 200 {
		yield(i)
	}
}

func TestIntegerBaseContract(t *testing.T) {
	// Java: setUp() assigns testCOSBase = COSNumber.get("0")
	n, err := GetNumber("0")
	if err != nil {
		t.Fatalf("GetNumber: %v", err)
	}
	assertBaseContract(t, n)
}

func TestIntegerEquals(t *testing.T) {
	integerSweep(func(i int64) {
		test1, test2, test3 := GetInteger(i), GetInteger(i), GetInteger(i)

		if !test1.Equals(test1) {
			t.Errorf("%d: not reflexive", i)
		}
		if !test2.Equals(test1) || !test1.Equals(test2) {
			t.Errorf("%d: not symmetric", i)
		}
		if !test2.Equals(test3) || !test1.Equals(test3) {
			t.Errorf("%d: not transitive", i)
		}

		if test1.Equals(GetInteger(i + 1)) {
			t.Errorf("%d equals %d", i, i+1)
		}
	})
}

func TestIntegerFloatValue(t *testing.T) {
	integerSweep(func(i int64) {
		if got := GetInteger(i).FloatValue(); got != float32(i) {
			t.Errorf("GetInteger(%d).FloatValue() = %v, want %v", i, got, float32(i))
		}
	})
}

func TestIntegerIntValue(t *testing.T) {
	integerSweep(func(i int64) {
		if got := GetInteger(i).IntValue(); got != int(i) {
			t.Errorf("GetInteger(%d).IntValue() = %d, want %d", i, got, i)
		}
	})
}

func TestIntegerLongValue(t *testing.T) {
	integerSweep(func(i int64) {
		if got := GetInteger(i).LongValue(); got != i {
			t.Errorf("GetInteger(%d).LongValue() = %d, want %d", i, got, i)
		}
	})
}

func TestIntegerAccept(t *testing.T) {
	// The emitted bytes are asserted in accept_external_test.go.
	assertVisits(t, GetInteger(0), "integer")
}

func TestIntegerWritePDF(t *testing.T) {
	integerSweep(func(i int64) {
		var buf bytes.Buffer
		if err := GetInteger(i).WritePDF(&buf); err != nil {
			t.Fatalf("WritePDF(%d): %v", i, err)
		}
		assertBytesEqual(t, []byte(strconv.FormatInt(i, 10)), buf.Bytes())
	})
}

func TestIntegerString(t *testing.T) {
	// Java: toString returns "COSInt{" + value + "}"
	if got := GetInteger(42).String(); got != "COSInt{42}" {
		t.Errorf("String() = %q, want %q", got, "COSInt{42}")
	}
	if got := GetInteger(-7).String(); got != "COSInt{-7}" {
		t.Errorf("String() = %q, want %q", got, "COSInt{-7}")
	}
}

// TestIntegerCaching pins the shared-instance range. Java keeps -100..256 in a
// static array and allocates outside it; the values are equal either way, so
// this checks the caching itself rather than any behaviour that depends on it.
func TestIntegerCaching(t *testing.T) {
	for _, i := range []int64{-100, 0, 1, 255, 256} {
		if GetInteger(i) != GetInteger(i) {
			t.Errorf("GetInteger(%d) returned distinct instances inside the cached range", i)
		}
	}
	// outside the range the values still compare equal
	for _, i := range []int64{-101, 257, 100000} {
		if !GetInteger(i).Equals(GetInteger(i)) {
			t.Errorf("GetInteger(%d) values compare unequal outside the cached range", i)
		}
	}
}

func TestIntegerIsValid(t *testing.T) {
	if !GetInteger(0).IsValid() {
		t.Error("a normal integer reports IsValid() = false")
	}
	// the out-of-range sentinels are reachable only through GetNumber
	n, err := GetNumber("18446744073307448448")
	if err != nil {
		t.Fatalf("GetNumber: %v", err)
	}
	if n.(*Integer).IsValid() {
		t.Error("an out-of-range integer reports IsValid() = true")
	}
}

// TestIntegerIntValueTruncatesTo32Bits pins Java's (int) narrowing cast, which
// drops everything above bit 31. Go's int(int64) is a no-op on a 64-bit
// platform, so the truncation has to be written explicitly.
func TestIntegerIntValueTruncatesTo32Bits(t *testing.T) {
	cases := []struct {
		in   int64
		want int
	}{
		{0, 0},
		{42, 42},
		{-1, -1},
		{math.MaxInt32, math.MaxInt32},
		{math.MinInt32, math.MinInt32},
		// above 32 bits the high word is dropped
		{1 << 32, 0},
		{1<<32 + 5, 5},
		{math.MaxInt32 + 1, math.MinInt32},
		{math.MaxInt64, -1},
	}
	for _, c := range cases {
		if got := GetInteger(c.in).IntValue(); got != c.want {
			t.Errorf("GetInteger(%d).IntValue() = %d, want %d", c.in, got, c.want)
		}
	}
}

// TestIntegerEqualsIsTruncating pins JAVA-BUGS entry 1: because equals compares
// intValue(), two values differing only above bit 31 compare equal. The port
// reproduces the defect rather than correcting it, so this asserts the wrong
// answer on purpose.
func TestIntegerEqualsIsTruncating(t *testing.T) {
	if !GetInteger(0).Equals(GetInteger(1 << 32)) {
		t.Error("0 and 1<<32 compare unequal; Java's equals truncates to 32 bits and finds them equal")
	}
	if !GetInteger(5).Equals(GetInteger(1<<32 + 5)) {
		t.Error("5 and 1<<32+5 compare unequal; the truncation must make them equal")
	}
	// values that differ inside 32 bits are still distinct
	if GetInteger(5).Equals(GetInteger(6)) {
		t.Error("5 and 6 compare equal")
	}
}

// TestIntegerLongValueIsNotTruncated checks that only IntValue narrows.
func TestIntegerLongValueIsNotTruncated(t *testing.T) {
	if got := GetInteger(1 << 32).LongValue(); got != 1<<32 {
		t.Errorf("LongValue() = %d, want %d", got, int64(1)<<32)
	}
}
