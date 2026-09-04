package encryption

import (
	"fmt"
	"unicode"
	"unicode/utf16"

	"golang.org/x/text/unicode/bidi"
	"golang.org/x/text/unicode/norm"
)

// SaslPrepQuery returns the SASLprep of the given string, allowing unassigned
// code points as a query does.
//
// Port of the package-private SaslPrep.saslPrepQuery. The port exports the two
// entry points because a Go test in another package has no other way in.
func SaslPrepQuery(str string) (string, error) { return saslPrep(str, true) }

// SaslPrepStored returns the SASLprep of the given string, rejecting unassigned
// code points as a stored string does.
//
// Port of SaslPrep.saslPrepStored.
func SaslPrepStored(str string) (string, error) { return saslPrep(str, false) }

// saslPrep is RFC 4013's SASLprep profile of RFC 3454.
//
// Java walks the string as UTF-16 units, which is what charAt and length give
// it; the port does the same, since the mapping and the prohibited tests are
// written against units and not against code points.
func saslPrep(str string, allowUnassigned bool) (string, error) {
	chars := utf16.Encode([]rune(str))

	// 1. Map

	// non-ASCII space chars mapped to space
	for i := 0; i < len(chars); i++ {
		ch := chars[i]
		if nonAsciiSpace(ch) {
			chars[i] = ' '
		}
	}

	length := 0
	for i := 0; i < len(chars); i++ {
		ch := chars[i]
		if !mappedToNothing(ch) {
			chars[length] = ch
			length++
		}
	}

	// 2. Normalize
	normalized := norm.NFKC.String(string(utf16.Decode(chars[:length])))

	containsRandALCat := false
	containsLCat := false
	initialRandALCat := false

	normalizedUnits := utf16.Encode([]rune(normalized))
	i := 0
	for i < len(normalizedUnits) {
		codepoint := codePointAtUnits(normalizedUnits, i)

		// 3. Prohibit
		if prohibited(codepoint) {
			// Java names the character with Character.getName, which Go has no
			// table for; the port writes the code point instead.
			return "", fmt.Errorf("Prohibited character 'U+%04X' at position %d", codepoint, i)
		}

		// 4. Check bidi
		isRandALcat := isRandALCat(rune(codepoint))
		containsRandALCat = containsRandALCat || isRandALcat
		containsLCat = containsLCat || isLCat(rune(codepoint))

		initialRandALCat = initialRandALCat || (i == 0 && isRandALcat)

		if !allowUnassigned && !isDefined(rune(codepoint)) {
			return "", fmt.Errorf("Character at position %d is unassigned", i)
		}

		i += charCount(codepoint)

		if initialRandALCat && i >= len(normalizedUnits) && !isRandALcat {
			return "", fmt.Errorf("First character is RandALCat, but last character is not")
		}
	}

	if containsRandALCat && containsLCat {
		return "", fmt.Errorf("Contains both RandALCat characters and LCat characters")
	}

	return normalized, nil
}

// codePointAtUnits is Java's String.codePointAt over UTF-16 units.
func codePointAtUnits(units []uint16, index int) int {
	unit := units[index]
	if utf16.IsSurrogate(rune(unit)) && index+1 < len(units) {
		if decoded := utf16.DecodeRune(rune(unit), rune(units[index+1])); decoded != 0xFFFD {
			return int(decoded)
		}
	}
	return int(unit)
}

// charCount is Java's Character.charCount: two units for a supplementary code
// point, one otherwise.
func charCount(codepoint int) int {
	if codepoint >= 0x10000 {
		return 2
	}
	return 1
}

// isRandALCat is Java's directionality test for RIGHT_TO_LEFT and
// RIGHT_TO_LEFT_ARABIC.
func isRandALCat(r rune) bool {
	properties, _ := bidi.LookupRune(r)
	class := properties.Class()
	return class == bidi.R || class == bidi.AL
}

// isLCat is Java's directionality test for LEFT_TO_RIGHT.
func isLCat(r rune) bool {
	properties, _ := bidi.LookupRune(r)
	return properties.Class() == bidi.L
}

// definedCategories are the top-level Unicode general categories; a code point
// is defined where it falls in one of them, which is Character.isDefined.
var definedCategories = []*unicode.RangeTable{
	unicode.L, unicode.M, unicode.N, unicode.P, unicode.S, unicode.Z, unicode.C,
}

func isDefined(r rune) bool {
	for _, table := range definedCategories {
		if unicode.Is(table, r) {
			return true
		}
	}
	return false
}

func prohibited(codepoint int) bool {
	return nonAsciiSpace(uint16(codepoint)) ||
		asciiControl(uint16(codepoint)) ||
		nonAsciiControl(codepoint) ||
		privateUse(codepoint) ||
		nonCharacterCodePoint(codepoint) ||
		surrogateCodePoint(codepoint) ||
		inappropriateForPlainText(codepoint) ||
		inappropriateForCanonical(codepoint) ||
		changeDisplayProperties(codepoint) ||
		tagging(codepoint)
}

func tagging(codepoint int) bool {
	return codepoint == 0xE0001 ||
		0xE0020 <= codepoint && codepoint <= 0xE007F
}

func changeDisplayProperties(codepoint int) bool {
	return codepoint == 0x0340 ||
		codepoint == 0x0341 ||
		codepoint == 0x200E ||
		codepoint == 0x200F ||
		codepoint == 0x202A ||
		codepoint == 0x202B ||
		codepoint == 0x202C ||
		codepoint == 0x202D ||
		codepoint == 0x202E ||
		codepoint == 0x206A ||
		codepoint == 0x206B ||
		codepoint == 0x206C ||
		codepoint == 0x206D ||
		codepoint == 0x206E ||
		codepoint == 0x206F
}

func inappropriateForCanonical(codepoint int) bool {
	return 0x2FF0 <= codepoint && codepoint <= 0x2FFB
}

func inappropriateForPlainText(codepoint int) bool {
	return codepoint == 0xFFF9 ||
		codepoint == 0xFFFA ||
		codepoint == 0xFFFB ||
		codepoint == 0xFFFC ||
		codepoint == 0xFFFD
}

func surrogateCodePoint(codepoint int) bool {
	return 0xD800 <= codepoint && codepoint <= 0xDFFF
}

func nonCharacterCodePoint(codepoint int) bool {
	return 0xFDD0 <= codepoint && codepoint <= 0xFDEF ||
		0xFFFE <= codepoint && codepoint <= 0xFFFF ||
		0x1FFFE <= codepoint && codepoint <= 0x1FFFF ||
		0x2FFFE <= codepoint && codepoint <= 0x2FFFF ||
		0x3FFFE <= codepoint && codepoint <= 0x3FFFF ||
		0x4FFFE <= codepoint && codepoint <= 0x4FFFF ||
		0x5FFFE <= codepoint && codepoint <= 0x5FFFF ||
		0x6FFFE <= codepoint && codepoint <= 0x6FFFF ||
		0x7FFFE <= codepoint && codepoint <= 0x7FFFF ||
		0x8FFFE <= codepoint && codepoint <= 0x8FFFF ||
		0x9FFFE <= codepoint && codepoint <= 0x9FFFF ||
		0xAFFFE <= codepoint && codepoint <= 0xAFFFF ||
		0xBFFFE <= codepoint && codepoint <= 0xBFFFF ||
		0xCFFFE <= codepoint && codepoint <= 0xCFFFF ||
		0xDFFFE <= codepoint && codepoint <= 0xDFFFF ||
		0xEFFFE <= codepoint && codepoint <= 0xEFFFF ||
		0xFFFFE <= codepoint && codepoint <= 0xFFFFF ||
		0x10FFFE <= codepoint && codepoint <= 0x10FFFF
}

func privateUse(codepoint int) bool {
	return 0xE000 <= codepoint && codepoint <= 0xF8FF ||
		0xF0000 <= codepoint && codepoint <= 0xFFFFD ||
		0x100000 <= codepoint && codepoint <= 0x10FFFD
}

func nonAsciiControl(codepoint int) bool {
	return 0x0080 <= codepoint && codepoint <= 0x009F ||
		codepoint == 0x06DD ||
		codepoint == 0x070F ||
		codepoint == 0x180E ||
		codepoint == 0x200C ||
		codepoint == 0x200D ||
		codepoint == 0x2028 ||
		codepoint == 0x2029 ||
		codepoint == 0x2060 ||
		codepoint == 0x2061 ||
		codepoint == 0x2062 ||
		codepoint == 0x2063 ||
		0x206A <= codepoint && codepoint <= 0x206F ||
		codepoint == 0xFEFF ||
		0xFFF9 <= codepoint && codepoint <= 0xFFFC ||
		0x1D173 <= codepoint && codepoint <= 0x1D17A
}

func asciiControl(ch uint16) bool {
	return ch <= 0x001F || ch == 0x007F
}

func nonAsciiSpace(ch uint16) bool {
	return ch == 0x00A0 ||
		ch == 0x1680 ||
		0x2000 <= ch && ch <= 0x200B ||
		ch == 0x202F ||
		ch == 0x205F ||
		ch == 0x3000
}

func mappedToNothing(ch uint16) bool {
	return ch == 0x00AD ||
		ch == 0x034F ||
		ch == 0x1806 ||
		ch == 0x180B ||
		ch == 0x180C ||
		ch == 0x180D ||
		ch == 0x200B ||
		ch == 0x200C ||
		ch == 0x200D ||
		ch == 0x2060 ||
		0xFE00 <= ch && ch <= 0xFE0F ||
		ch == 0xFEFF
}
