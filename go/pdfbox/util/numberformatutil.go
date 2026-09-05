package util

import "math"

// maxFractionDigits is the most fraction digits FormatFloatFast can write.
//
// Port of the private NumberFormatUtil.MAX_FRACTION_DIGITS.
const maxFractionDigits = 5

// powerOfTens and powerOfTensInt are the tables the formatting divides by.
var (
	powerOfTens    [19]int64
	powerOfTensInt [10]int32
)

func init() {
	powerOfTens[0] = 1
	for exp := 1; exp < len(powerOfTens); exp++ {
		powerOfTens[exp] = powerOfTens[exp-1] * 10
	}
	powerOfTensInt[0] = 1
	for exp := 1; exp < len(powerOfTensInt); exp++ {
		powerOfTensInt[exp] = powerOfTensInt[exp-1] * 10
	}
}

// FormatFloatFast writes value into asciiBuffer with at most maxFractionDigits
// fraction digits, and returns how many bytes it wrote, or -1 where it cannot
// write the value at all.
//
// Port of the static NumberFormatUtil.formatFloatFast. It is there because
// java.text.NumberFormat is far slower, and the writer formats a great many
// numbers.
func FormatFloatFast(value float32, fractionDigits int, asciiBuffer []byte) int {
	if math.IsNaN(float64(value)) ||
		math.IsInf(float64(value), 0) ||
		value > float32(math.MaxInt64) ||
		value <= float32(math.MinInt64) ||
		fractionDigits > maxFractionDigits {
		return -1
	}

	offset := 0
	integerPart := int64(value)

	// handle sign
	if value < 0 {
		asciiBuffer[offset] = '-'
		offset++
		integerPart = -integerPart
	}

	// extract fraction part
	fractionPart := int64((math.Abs(float64(value))-float64(integerPart))*
		float64(powerOfTens[fractionDigits]) + 0.5)

	// Check for rounding to next integer
	if fractionPart >= powerOfTens[fractionDigits] {
		integerPart++
		fractionPart -= powerOfTens[fractionDigits]
	}

	// format integer part
	offset = formatPositiveNumber(integerPart, exponentOf(integerPart), false, asciiBuffer, offset)

	if fractionPart > 0 && fractionDigits > 0 {
		asciiBuffer[offset] = '.'
		offset++
		offset = formatPositiveNumber(fractionPart, fractionDigits-1, true, asciiBuffer, offset)
	}
	return offset
}

// formatPositiveNumber writes one number digit by digit, from the given power
// of ten down.
func formatPositiveNumber(number int64, exp int, omitTrailingZeros bool,
	asciiBuffer []byte, startOffset int) int {
	offset := startOffset
	remaining := number
	for remaining > math.MaxInt32 {
		digit := remaining / powerOfTens[exp]
		remaining -= digit * powerOfTens[exp]
		asciiBuffer[offset] = byte('0' + digit)
		offset++
		exp--
	}
	// If the remaining fits into an integer, use int arithmetic as it is faster
	remainingInt := int32(remaining)
	for exp >= 0 && (!omitTrailingZeros || remainingInt > 0) {
		digit := remainingInt / powerOfTensInt[exp]
		remainingInt -= digit * powerOfTensInt[exp]
		asciiBuffer[offset] = byte('0' + digit)
		offset++
		exp--
	}
	return offset
}

// exponentOf returns the power of ten of the highest digit of number.
func exponentOf(number int64) int {
	for exp := 0; exp < len(powerOfTens)-1; exp++ {
		if number < powerOfTens[exp+1] {
			return exp
		}
	}
	return len(powerOfTens) - 1
}
