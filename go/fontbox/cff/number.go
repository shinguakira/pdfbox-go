package cff

import (
	"strconv"
	"strings"
)

// A charstring sequence is a Java List<Object> holding Numbers and
// CharStringCommands, and the number list beside it a List<Number>. Both are
// []any here, and a number in one of them is an int for Java's Integer, a
// float32 for its Float, or a float64 for its Double -- the three the two
// parsers and the two charstrings put there. The helpers below stand for the
// instanceof tests and the Number methods the Java calls on them.

// isNumber reports whether the entry is one of the three number types, which is
// Java's `obj instanceof Number`.
func isNumber(entry any) bool {
	switch entry.(type) {
	case int, float32, float64:
		return true
	}
	return false
}

// isInteger reports whether the entry is Java's Integer.
func isInteger(entry any) bool {
	_, ok := entry.(int)
	return ok
}

// numberFloat is Java's Number.floatValue.
func numberFloat(entry any) float32 {
	switch v := entry.(type) {
	case int:
		return float32(v)
	case float32:
		return v
	case float64:
		return float32(v)
	}
	panic("cff: not a number")
}

// numberInt is Java's Number.intValue, which truncates towards zero.
func numberInt(entry any) int {
	switch v := entry.(type) {
	case int:
		return v
	case float32:
		return floatToInt(v)
	case float64:
		return doubleToInt(v)
	}
	panic("cff: not a number")
}

// floatToInt narrows a float the way a Java (int) cast does: towards zero, with
// anything past the range clamped to it and NaN going to zero.
func floatToInt(value float32) int {
	switch {
	case value != value: // NaN
		return 0
	case value >= 2147483647:
		return 2147483647
	case value <= -2147483648:
		return -2147483648
	}
	return int(value)
}

// doubleToInt narrows a double the same way.
func doubleToInt(value float64) int {
	switch {
	case value != value: // NaN
		return 0
	case value >= 2147483647:
		return 2147483647
	case value <= -2147483648:
		return -2147483648
	}
	return int(value)
}

// entryString renders one sequence entry the way Java's String.valueOf would,
// for the toString of a charstring.
func entryString(entry any) string {
	switch v := entry.(type) {
	case int:
		return strconv.Itoa(v)
	case float32:
		return javaFloatString(v)
	case float64:
		return javaDoubleString(v)
	case *CharStringCommand:
		return v.String()
	}
	return "null"
}

// javaFloatString renders a float32 the way Java's Float.toString does, which
// always leaves a decimal point in.
func javaFloatString(value float32) string {
	s := strconv.FormatFloat(float64(value), 'g', -1, 32)
	if !strings.ContainsAny(s, ".eE") {
		s += ".0"
	}
	return s
}

// javaDoubleString renders a float64 the way Java's Double.toString does.
func javaDoubleString(value float64) string {
	s := strconv.FormatFloat(value, 'g', -1, 64)
	if !strings.ContainsAny(s, ".eE") {
		s += ".0"
	}
	return s
}

// sequenceString renders a whole sequence the way Java's List.toString does.
func sequenceString(sequence []any) string {
	var sb strings.Builder
	sb.WriteByte('[')
	for i, entry := range sequence {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(entryString(entry))
	}
	sb.WriteByte(']')
	return sb.String()
}
