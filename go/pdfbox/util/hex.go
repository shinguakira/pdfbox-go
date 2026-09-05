package util

import (
	"bytes"
	"encoding/base64"
	"io"
	"log/slog"
	"regexp"
	"strings"
	"unicode/utf16"
)

// hexChars are the sixteen hexadecimal digits, upper case.
//
// Port of the private HEX_BYTES and HEX_CHARS of Hex, which hold the same
// digits as bytes and as chars; Go writes both from one string.
const hexChars = "0123456789ABCDEF"

// patternSpace matches one whitespace character.
//
// Port of StringUtil.PATTERN_SPACE. Java's \s is [ \t\n\x0B\f\r]; the \s of Go
// is the same set.
var patternSpace = regexp.MustCompile(`\s`)

// HexString returns the two hexadecimal digits of the given byte.
//
// Port of the static Hex.getString(byte).
func HexString(b byte) string {
	return string([]byte{hexChars[highNibble(b)], hexChars[lowNibble(b)]})
}

// HexStringOfBytes returns the hexadecimal digits of the given bytes.
//
// Port of the static Hex.getString(byte[]).
func HexStringOfBytes(bs []byte) string {
	str := &strings.Builder{}
	str.Grow(len(bs) * 2)
	for _, b := range bs {
		str.WriteByte(hexChars[highNibble(b)])
		str.WriteByte(hexChars[lowNibble(b)])
	}
	return str.String()
}

// HexBytes returns the two hexadecimal digits of the given byte, as ASCII.
//
// Port of the static Hex.getBytes(byte).
func HexBytes(b byte) []byte {
	return []byte{hexChars[highNibble(b)], hexChars[lowNibble(b)]}
}

// HexBytesOfBytes returns the hexadecimal digits of the given bytes, as ASCII.
//
// Port of the static Hex.getBytes(byte[]).
func HexBytesOfBytes(bs []byte) []byte {
	asciiBytes := make([]byte, len(bs)*2)
	for i, b := range bs {
		asciiBytes[i*2] = hexChars[highNibble(b)]
		asciiBytes[i*2+1] = hexChars[lowNibble(b)]
	}
	return asciiBytes
}

// HexChars returns the four hexadecimal digits of the given 16-bit number.
//
// Port of the static Hex.getChars(short).
func HexChars(num int16) []byte {
	hex := make([]byte, 4)
	hex[0] = hexChars[(num>>12)&0x0F]
	hex[1] = hexChars[(num>>8)&0x0F]
	hex[2] = hexChars[(num>>4)&0x0F]
	hex[3] = hexChars[num&0x0F]
	return hex
}

// HexCharsUTF16BE returns the hexadecimal digits of the given text, encoded as
// UTF-16BE.
//
// Port of the static Hex.getCharsUTF16BE. Java walks the string by char, which
// is already a UTF-16 code unit; Go strings are UTF-8, so the port encodes to
// UTF-16 first.
func HexCharsUTF16BE(text string) []byte {
	units := utf16Units(text)
	hex := make([]byte, 0, len(units)*4)
	for _, c := range units {
		hex = append(hex,
			hexChars[(c>>12)&0x0F],
			hexChars[(c>>8)&0x0F],
			hexChars[(c>>4)&0x0F],
			hexChars[c&0x0F])
	}
	return hex
}

// WriteHexByte writes the two hexadecimal digits of the given byte.
//
// Port of the static Hex.writeHexByte.
func WriteHexByte(b byte, output io.Writer) error {
	_, err := output.Write([]byte{hexChars[highNibble(b)], hexChars[lowNibble(b)]})
	return err
}

// WriteHexBytes writes the hexadecimal digits of the given bytes.
//
// Port of the static Hex.writeHexBytes.
func WriteHexBytes(bs []byte, output io.Writer) error {
	for _, b := range bs {
		if err := WriteHexByte(b, output); err != nil {
			return err
		}
	}
	return nil
}

// utf16Units returns the UTF-16 code units of the given text, which is what the
// chars of a Java string already are.
func utf16Units(s string) []uint16 { return utf16.Encode([]rune(s)) }

// highNibble returns the top four bits of the byte. Java declares it private.
func highNibble(b byte) int { return int(b&0xF0) >> 4 }

// lowNibble returns the bottom four bits of the byte. Java declares it private.
func lowNibble(b byte) int { return int(b & 0x0F) }

// DecodeBase64 decodes the given base64 text, with every whitespace character
// dropped first.
//
// Port of the static Hex.decodeBase64. Java throws IllegalArgumentException for
// text that is not base64, which is unchecked, so the port returns the error and
// the one caller logs it -- Java catches it there too.
func DecodeBase64(base64Value string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(patternSpace.ReplaceAllString(base64Value, ""))
}

// DecodeHex decodes the given ASCII hexadecimal text, skipping newlines and
// carriage returns and logging any pair it cannot read.
//
// Port of the static Hex.decodeHex.
func DecodeHex(s string) []byte {
	baos := &bytes.Buffer{}
	baos.Grow((len(s) + 1) / 2)
	i := 0
	for i < len(s)-1 {
		if s[i] == '\n' || s[i] == '\r' {
			i++
			continue
		}
		value := 16*HexValue(rune(s[i])) + HexValue(rune(s[i+1]))
		if value >= 0 {
			baos.WriteByte(byte(value))
		} else {
			slog.Error("util: can't parse, aborting decode", slog.String("byte", s[i:i+2]))
		}
		i += 2
	}
	return baos.Bytes()
}

// HexValue returns the value of the given hexadecimal digit, and -256 for a
// character that is not one.
//
// Port of the static Hex.getHexValue. The value of -256 is chosen so that two
// hex digits can be combined before checking for an invalid hex string.
func HexValue(c rune) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	}
	return -256
}
