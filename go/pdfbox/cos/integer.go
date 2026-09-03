package cos

import (
	"io"
	"math"
	"strconv"
)

// Integer is an integer number in a PDF document.
//
// Port of org.apache.pdfbox.cos.COSInteger.
type Integer struct {
	object
	value   int64
	isValid bool
}

var _ Number = (*Integer)(nil)

const (
	// integerCacheLow and integerCacheHigh bound the shared instances Java
	// keeps in a static array.
	integerCacheLow  = -100
	integerCacheHigh = 256
)

// The cache is built by a function called from the variable initializer, not
// from init(). Package-level variables are initialized before init() runs, so
// filling it in init() would leave IntegerZero and friends holding nil — Go
// orders variable initialization by dependency, and only this form makes
// GetInteger's dependency on the cache visible to it.
var integerCache = newIntegerCache()

func newIntegerCache() [integerCacheHigh - integerCacheLow + 1]*Integer {
	var cache [integerCacheHigh - integerCacheLow + 1]*Integer
	for i := range cache {
		cache[i] = &Integer{value: int64(i + integerCacheLow), isValid: true}
	}
	return cache
}

// Predefined small integers.
var (
	IntegerZero  = GetInteger(0)
	IntegerOne   = GetInteger(1)
	IntegerTwo   = GetInteger(2)
	IntegerThree = GetInteger(3)
)

// Values returned by GetNumber for input that is a well-formed integer literal
// but too large to represent. They carry a clamped value and report IsValid as
// false; see PDFBOX-5176.
var (
	integerOutOfRangeMax = &Integer{value: math.MaxInt64, isValid: false}
	integerOutOfRangeMin = &Integer{value: math.MinInt64, isValid: false}
)

// GetInteger returns an Integer with the given value, sharing an instance for
// small values as Java does.
//
// Port of COSInteger.get(long). Java populates its cache lazily and notes that
// no synchronization is needed because a duplicate allocation would be
// harmless; the port fills the cache in init instead, which removes the data
// race that reasoning depends on.
func GetInteger(value int64) *Integer {
	if value >= integerCacheLow && value <= integerCacheHigh {
		return integerCache[value-integerCacheLow]
	}
	return &Integer{value: value, isValid: true}
}

// FloatValue returns the value as a float32.
func (i *Integer) FloatValue() float32 { return float32(i.value) }

// IntValue returns the value narrowed to 32 bits.
//
// Port of intValue(), which is `(int) value`. Java's narrowing cast discards
// everything above bit 31; Go's int(int64) does not, because int is 64 bits on
// every platform this builds for, so the truncation has to be written out.
func (i *Integer) IntValue() int { return int(int32(i.value)) }

// LongValue returns the value.
func (i *Integer) LongValue() int64 { return i.value }

// IsValid reports whether the value is within the representable range. It is
// false only for a number that parsed as an integer literal but was too large;
// the value is then clamped.
func (i *Integer) IsValid() bool { return i.isValid }

// COSObject returns the receiver.
func (i *Integer) COSObject() Base { return i }

// Accept dispatches to the visitor.
func (i *Integer) Accept(v Visitor) error { return v.VisitInteger(i) }

// Equals reports whether two integers hold the same value.
//
// This looks wrong and is ported as written. Java compares intValue(), which
// truncates to 32 bits, so two values differing only above bit 31 compare
// equal — 1<<32 equals 0 here. Comparing the full int64 would be correct, but
// the Java is the reference and object identity in a PDF can already depend on
// this.
func (i *Integer) Equals(other *Integer) bool {
	return other != nil && other.IntValue() == i.IntValue()
}

// String returns the Java toString form.
func (i *Integer) String() string {
	return "COSInt{" + strconv.FormatInt(i.value, 10) + "}"
}

// WritePDF writes the value as a PDF object.
func (i *Integer) WritePDF(w io.Writer) error {
	_, err := w.Write([]byte(strconv.FormatInt(i.value, 10)))
	return err
}
