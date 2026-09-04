package cmap

import "unicode/utf16"

// The cached mappings, built when the package loads.
//
// Port of org.apache.fontbox.cmap.CMapStrings, whose static initialiser says
// it creates all mappings when loading the class to avoid concurrency issues.
var (
	twoByteMappings [256 * 256]string
	oneByteMappings [256]string
	indexValues     [256 * 256]int
	oneByteValues   [256][]byte
	twoByteValues   [256 * 256][]byte
)

func init() {
	fillMappings()
}

func fillMappings() {
	for i := 0; i < 256; i++ {
		for j := 0; j < 256; j++ {
			bytes := []byte{byte(i), byte(j)}
			index := i*256 + j
			twoByteMappings[index] = decodeUTF16BE(bytes)
			twoByteValues[index] = bytes
			indexValues[index] = index
		}
	}
	for i := 0; i < 256; i++ {
		bytes := []byte{byte(i)}
		oneByteMappings[i] = decodeLatin1(bytes)
		oneByteValues[i] = bytes
	}
}

// decodeUTF16BE decodes big-endian UTF-16, as Java's UTF_16BE charset does.
func decodeUTF16BE(data []byte) string {
	units := make([]uint16, 0, len(data)/2)
	for i := 0; i+1 < len(data); i += 2 {
		units = append(units, uint16(data[i])<<8|uint16(data[i+1]))
	}
	return string(utf16.Decode(units))
}

// decodeLatin1 decodes ISO-8859-1, where every byte is the code point of the
// same value.
func decodeLatin1(data []byte) string {
	runes := make([]rune, len(data))
	for i, b := range data {
		runes[i] = rune(b)
	}
	return string(runes)
}

// GetMapping returns the cached string for a one or two byte code, or the empty
// string where the code is longer than two bytes and nothing is cached.
//
// Java returns null there, and its callers null-check.
//
// JAVA-BUGS entry 23: Java's ternary has two arms for three cases, so a
// zero-length code falls into the two-byte arm and comes back as U+0000 rather
// than as the empty string its caller's other arm would give it. Ported as
// written; do not special-case the empty code here.
func GetMapping(bytes []byte) (string, bool) {
	if len(bytes) > 2 {
		return "", false
	}
	if len(bytes) == 1 {
		return oneByteMappings[ToInt(bytes)], true
	}
	return twoByteMappings[ToInt(bytes)], true
}

// GetIndexValue returns the cached index for a one or two byte code. The second
// result is false for a longer code, which Java reports with null.
func GetIndexValue(bytes []byte) (int, bool) {
	if len(bytes) > 2 {
		return 0, false
	}
	return indexValues[ToInt(bytes)], true
}

// GetByteValue returns the cached copy of a one or two byte code, or nil for a
// longer one.
//
// The copy is shared: every call for the same code hands back the same slice,
// which is the point of the cache. It must not be written to.
func GetByteValue(bytes []byte) []byte {
	if len(bytes) > 2 {
		return nil
	}
	if len(bytes) == 1 {
		return oneByteValues[ToInt(bytes)]
	}
	return twoByteValues[ToInt(bytes)]
}

// ToInt reads a code as a big-endian integer.
//
// Port of the package-private CMap.toInt.
func ToInt(data []byte) int {
	return ToIntLen(data, len(data))
}

// ToIntLen reads the first dataLen bytes of a code as a big-endian integer.
//
// Java accumulates in an int, which is 32 bits, so a four-byte code whose first
// byte is 0x80 or more comes out negative; a Go int is 64 bits and would keep
// it positive. The width is observable: CMap.toUnicode(int) tests the code
// against 256, 0xFFFF and 0xFFFFFF to decide how many bytes it had, and a
// negative code takes the two-byte branch. The port narrows to match.
func ToIntLen(data []byte, dataLen int) int {
	var code int32
	for i := 0; i < dataLen; i++ {
		code <<= 8
		code |= int32(data[i]) & 0xFF
	}
	return int(code)
}
