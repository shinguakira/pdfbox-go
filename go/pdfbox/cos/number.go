package cos

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ErrNotANumber is returned when a string cannot be read as a PDF number.
// Java throws IOException("Not a number: ...").
var ErrNotANumber = errors.New("cos: not a number")

// Number is a numeric COS object, either an Integer or a Float.
//
// Port of the abstract class org.apache.pdfbox.cos.COSNumber. Java uses it as a
// common superclass so callers can take either; the port makes it an interface
// over Base for the same purpose.
type Number interface {
	Base

	// FloatValue returns the value as a float32.
	FloatValue() float32

	// IntValue returns the value as an int.
	IntValue() int

	// LongValue returns the value as an int64.
	LongValue() int64
}

// GetNumber parses a PDF number, returning an Integer or a Float.
//
// Port of the static COSNumber.get(String).
func GetNumber(number string) (Number, error) {
	if number == "" {
		// Java takes a null here and throws NullPointerException. Go has no
		// null string, so the empty string is the degenerate input and is
		// rejected the same way a non-number is.
		return nil, fmt.Errorf("%w: %q", ErrNotANumber, number)
	}

	if len(number) == 1 {
		digit := number[0]
		if digit >= '0' && digit <= '9' {
			return GetInteger(int64(digit - '0')), nil
		}
		if digit == '-' || digit == '.' {
			// A lone sign or point is read as zero; see PDFBOX-592.
			return IntegerZero, nil
		}
		return nil, fmt.Errorf("%w: %q", ErrNotANumber, number)
	}

	if isFloatLiteral(number) {
		return ParseFloat(number)
	}

	if v, err := strconv.ParseInt(number, 10, 64); err == nil {
		return GetInteger(v), nil
	}

	// Not representable as an int64. Java distinguishes "too big" from "not a
	// number at all" by re-checking that the remaining characters are digits.
	digits := strings.TrimPrefix(strings.TrimPrefix(number, "+"), "-")
	if !isAllDigits(digits) {
		return nil, fmt.Errorf("%w: %q", ErrNotANumber, number)
	}
	// Return a clamped Integer marked invalid; see PDFBOX-5176.
	if strings.HasPrefix(number, "-") {
		return integerOutOfRangeMin, nil
	}
	return integerOutOfRangeMax, nil
}

// isFloatLiteral reports whether the string should be read as a float.
//
// Port of the private COSNumber.isFloat. Note that it looks for a lower-case
// 'e' only, so "1E5" is routed to the integer branch and then rejected as a
// non-number. The port keeps that: producers emit lower-case, and widening it
// here would change which inputs are accepted.
func isFloatLiteral(number string) bool {
	for i := 0; i < len(number); i++ {
		if number[i] == '.' || number[i] == 'e' {
			return true
		}
	}
	return false
}

func isAllDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
