package cos

import (
	"errors"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// Float is a floating point number in a PDF document.
//
// Port of org.apache.pdfbox.cos.COSFloat. The value is a float32 because PDF
// reals are single precision, and because the written form has to match what
// Java produces.
type Float struct {
	object
	value float32
	// valueAsString is the literal this was parsed from, kept so that a number
	// read from a file is written back unchanged. Empty when the value was not
	// parsed from a string, or when parsing had to coerce it.
	valueAsString string
}

var _ Number = (*Float)(nil)

// Predefined float constants.
var (
	FloatZero = &Float{value: 0, valueAsString: "0.0"}
	FloatOne  = &Float{value: 1, valueAsString: "1.0"}
)

// NewFloat returns a Float wrapping the given value.
func NewFloat(value float32) *Float {
	return &Float{value: value}
}

// Patterns for the malformed literals real producers emit. Compiled once; Java
// calls String.matches, which recompiles on every call.
var (
	// PDFBOX-2990 "0.00000-33917698", PDFBOX-3369, PDFBOX-3500 "0.-262"
	misplacedNegativeAfterZero = regexp.MustCompile(`^0\.0*-\d+$`)
	// PDFBOX-5829 "-12.-1"
	misplacedNegativeInFraction = regexp.MustCompile(`^-\d+\.-\d+$`)
)

// ParseFloat reads a PDF real from its literal form.
//
// Port of the COSFloat(String) constructor, including its repairs for the
// malformed numbers that real producers emit. Anything it cannot repair is an
// error, as it is in Java.
func ParseFloat(literal string) (*Float, error) {
	parsed, err := strconv.ParseFloat(literal, 32)
	if err == nil || isRangeError(err) {
		// A range error means the literal was well-formed but outside the
		// float32 range, and parsed already holds ±Inf or 0. Java sees the
		// same thing as a successful parse returning Infinity, and clamps it
		// in coerce; the port takes the same path rather than rejecting.
		f := float32(parsed)
		coerced := coerceFloat(f)
		stringValue := ""
		if f == coerced {
			// Only keep the literal when nothing was coerced, so that a value
			// clamped to the float range is re-rendered rather than written
			// back in a form that does not match it.
			stringValue = literal
		}
		return &Float{value: coerced, valueAsString: stringValue}, nil
	}

	repaired := literal
	switch {
	case strings.HasPrefix(literal, "--"):
		// PDFBOX-4289 has --16.33
		repaired = literal[1:]
	case misplacedNegativeAfterZero.MatchString(literal):
		// 0.00000-33917698 becomes -0.0000033917698
		repaired = "-" + strings.Replace(literal, "-", "", 1)
	case misplacedNegativeInFraction.MatchString(literal):
		// -12.-1 becomes -12.1
		repaired = "-" + strings.ReplaceAll(literal, "-", "")
	default:
		return nil, fmt.Errorf("cos: expected floating point number, actual=%q: %w", literal, err)
	}

	parsed, err = strconv.ParseFloat(repaired, 32)
	if err != nil && !isRangeError(err) {
		return nil, fmt.Errorf("cos: expected floating point number, actual=%q: %w", literal, err)
	}
	// Java discards the original literal on the repair path by leaving
	// valueAsString null, so the repaired value is re-rendered on write.
	return &Float{value: coerceFloat(float32(parsed))}, nil
}

// isRangeError reports whether err is strconv's out-of-range signal, which
// accompanies a usable ±Inf or zero result rather than a parse failure.
func isRangeError(err error) bool {
	var numErr *strconv.NumError
	return errors.As(err, &numErr) && numErr.Err == strconv.ErrRange
}

// coerceFloat clamps a value into the representable range.
//
// Port of the private COSFloat.coerce. Values below the smallest normal float
// become zero, per the PDF spec, Appendix C implementation limits; infinities
// clamp to the largest finite float rather than propagating.
func coerceFloat(value float32) float32 {
	if math.IsInf(float64(value), 1) {
		return math.MaxFloat32
	}
	if math.IsInf(float64(value), -1) {
		return -math.MaxFloat32
	}
	if math.Abs(float64(value)) < math.SmallestNonzeroFloat32*(1<<23) {
		// Java compares against Float.MIN_NORMAL, the smallest normalised
		// value. Go names only SmallestNonzeroFloat32, the smallest
		// subnormal, so the normal minimum is that scaled by 2^23.
		return 0
	}
	return value
}

// FloatValue returns the value.
func (f *Float) FloatValue() float32 { return f.value }

// IntValue returns the value truncated toward zero and clamped to 32 bits.
//
// Port of intValue(), which is `(int) value`. A Java narrowing cast from a
// floating point type saturates: a value too large becomes Integer.MAX_VALUE,
// too small becomes Integer.MIN_VALUE, and NaN becomes zero. Go leaves an
// out-of-range float conversion undefined, so the clamping is written out.
func (f *Float) IntValue() int {
	switch {
	case math.IsNaN(float64(f.value)):
		return 0
	case float64(f.value) >= math.MaxInt32:
		return math.MaxInt32
	case float64(f.value) <= math.MinInt32:
		return math.MinInt32
	}
	return int(int32(f.value))
}

// LongValue returns the value truncated toward zero and clamped to 64 bits.
//
// Port of longValue(), which is `(long) value`, and saturates the same way.
func (f *Float) LongValue() int64 {
	switch {
	case math.IsNaN(float64(f.value)):
		return 0
	case float64(f.value) >= math.MaxInt64:
		return math.MaxInt64
	case float64(f.value) <= math.MinInt64:
		return math.MinInt64
	}
	return int64(f.value)
}

// COSObject returns the receiver.
func (f *Float) COSObject() Base { return f }

// Accept dispatches to the visitor.
func (f *Float) Accept(v Visitor) error { return v.VisitFloat(f) }

// Equals compares by bit pattern, as Java's Float.floatToIntBits comparison
// does, so that -0.0 and 0.0 are distinct and NaN equals itself.
func (f *Float) Equals(other *Float) bool {
	return other != nil &&
		math.Float32bits(other.value) == math.Float32bits(f.value)
}

// String returns the Java toString form.
func (f *Float) String() string { return "COSFloat{" + f.formatString() + "}" }

// formatString builds the written form, caching it as Java does.
//
// Java renders with String.valueOf and, when that produces exponential
// notation, re-renders through BigDecimal to a plain string. strconv with
// 'f' produces the plain form directly, so there is no second pass.
func (f *Float) formatString() string {
	if f.valueAsString == "" {
		f.valueAsString = plainFloatString(f.value)
	}
	return f.valueAsString
}

// plainFloatString renders a float32 in plain decimal notation, never
// exponential, matching what Java's BigDecimal.toPlainString produces from
// String.valueOf(float).
func plainFloatString(value float32) string {
	// 'g' with -1 precision gives the shortest representation that round-trips,
	// which is what Java's String.valueOf(float) also produces.
	shortest := strconv.FormatFloat(float64(value), 'g', -1, 32)
	if !strings.ContainsAny(shortest, "eE") {
		if !strings.Contains(shortest, ".") {
			// Java always renders a float with a fraction part
			shortest += ".0"
		}
		return shortest
	}
	// Re-render without an exponent, keeping every significant digit.
	f64, err := strconv.ParseFloat(shortest, 64)
	if err != nil {
		return shortest
	}
	plain := strconv.FormatFloat(f64, 'f', -1, 64)
	if !strings.Contains(plain, ".") {
		plain += ".0"
	}
	return plain
}

// WritePDF writes the value as a PDF object.
func (f *Float) WritePDF(w io.Writer) error {
	_, err := w.Write([]byte(f.formatString()))
	return err
}
