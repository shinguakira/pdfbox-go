// Package javafmt renders numbers the way Java does.
//
// Ported code carries Java's toString output in log messages, error text and a
// few string-valued APIs, and Java's float formatting differs from Go's in one
// visible way: it always shows a fraction part, so 1f prints as "1.0" rather
// than "1". Keeping the difference in one place stops each package that needs
// it from growing its own copy.
package javafmt

import (
	"strconv"
	"strings"
)

// Float32 renders a float the way Java's String.valueOf(float) does.
func Float32(value float32) string {
	// 'g' with -1 precision gives the shortest representation that round-trips,
	// which is what Java produces as well.
	return withFractionPart(strconv.FormatFloat(float64(value), 'g', -1, 32))
}

// Float64 renders a double the way Java's String.valueOf(double) does.
func Float64(value float64) string {
	return withFractionPart(strconv.FormatFloat(value, 'g', -1, 64))
}

// withFractionPart appends ".0" to a plain integer rendering. A value already
// carrying a point, an exponent, or a name — NaN, +Inf, -Inf — is left alone.
func withFractionPart(s string) string {
	if strings.ContainsAny(s, ".eEnI") {
		return s
	}
	return s + ".0"
}
