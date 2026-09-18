package cos

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// ErrInvalidHexString is returned by ParseHexString for a string containing a
// character that is not a hex digit. Java throws IOException("Invalid hex
// string: ...").
var ErrInvalidHexString = errors.New("cos: invalid hex string")

// StringObj is a string in a PDF document.
//
// Port of org.apache.pdfbox.cos.COSString. The name keeps a suffix because
// cos.String(x) would read as a conversion; see the package doc.
//
// A string holds bytes, not a Go string: PDF strings are byte sequences in
// either PDFDocEncoding or UTF-16, and the writer has to reproduce them.
type StringObj struct {
	object
	bytes        []byte
	forceHexForm bool
}

var _ Base = (*StringObj)(nil)

// NewStringObjBytes wraps the given bytes. The slice is copied.
//
// Port of COSString(byte[]).
func NewStringObjBytes(b []byte) *StringObj {
	return NewStringObjBytesHex(b, false)
}

// NewStringObjBytesHex wraps the given bytes, optionally forcing the hex form
// when written.
//
// Port of COSString(byte[], boolean).
func NewStringObjBytesHex(b []byte, forceHex bool) *StringObj {
	stored := make([]byte, len(b))
	copy(stored, b)
	return &StringObj{bytes: stored, forceHexForm: forceHex}
}

// NewStringObj encodes text as a PDF string.
//
// Port of COSString(String). Text that PDFDocEncoding can represent is stored
// in that single-byte encoding; anything else is stored as UTF-16BE with a
// leading byte order mark.
func NewStringObj(text string) *StringObj {
	return NewStringObjHex(text, false)
}

// NewStringObjHex encodes text as a PDF string, optionally forcing the hex form
// when written.
//
// Port of COSString(String, boolean).
func NewStringObjHex(text string, forceHex bool) *StringObj {
	onlyPDFDocEncoding := true
	for _, r := range text {
		if !pdfDocEncodingContainsRune(r) {
			onlyPDFDocEncoding = false
			break
		}
	}

	if onlyPDFDocEncoding {
		return &StringObj{bytes: pdfDocEncodingGetBytes(text), forceHexForm: forceHex}
	}

	// UTF-16BE with a byte order mark
	units := utf16.Encode([]rune(text))
	b := make([]byte, 0, len(units)*2+2)
	b = append(b, 0xFE, 0xFF)
	for _, u := range units {
		b = append(b, byte(u>>8), byte(u))
	}
	return &StringObj{bytes: b, forceHexForm: forceHex}
}

// ParseHexString reads a PDF string from its hex form, the bytes between the
// angle brackets.
//
// Port of COSString.parseHex. Leading and trailing whitespace is skipped, and
// an odd number of digits is padded with a trailing zero nibble.
//
// ForceParsing replaces a malformed hex digit with '?' instead of failing.
//
// Port of the COSString.FORCE_PARSING flag, which Java reads once from the
// org.apache.pdfbox.forceParsing system property. Go has no system properties,
// so it is a package variable with the same default (false). The Java source
// marks the substitution with a todo asking what Acrobat actually does; that is
// carried over as-is rather than resolved here.
var ForceParsing = false

func ParseHexString(hex string) (*StringObj, error) {
	end := len(hex)
	for end > 0 && isPDFWhitespaceByte(hex[end-1]) {
		end--
	}
	start := 0
	for start < end && isPDFWhitespaceByte(hex[start]) {
		start++
	}

	// JAVA-BUGS entry 11: start is computed and then never used. Java indexes
	// the original string from zero, so only the *length* of the leading
	// whitespace is honoured and the spaces themselves are read as hex digits.
	// Ported as written; do not slice hex[start:end] here.
	_ = start
	length := end - start
	uneven := length%2 != 0
	if uneven {
		length--
	}

	out := make([]byte, 0, (length+1)/2)
	for i := 0; i < length; i += 2 {
		hi, lo := hexValue(hex[i]), hexValue(hex[i+1])
		value := 16*hi + lo
		switch {
		case hi >= 0 && lo >= 0:
			out = append(out, byte(value))
		case ForceParsing:
			slog.Warn("cos: encountered a malformed hex string")
			out = append(out, '?')
		default:
			return nil, fmt.Errorf("%w: %q", ErrInvalidHexString, hex)
		}
	}
	if uneven {
		hi := hexValue(hex[length])
		switch {
		case hi >= 0:
			out = append(out, byte(16*hi))
		case ForceParsing:
			slog.Warn("cos: encountered a malformed hex string")
			out = append(out, '?')
		default:
			return nil, fmt.Errorf("%w: %q", ErrInvalidHexString, hex)
		}
	}

	return &StringObj{bytes: out}, nil
}

// isPDFWhitespaceByte reports whether b is whitespace for the purposes of
// trimming a hex string.
//
// Java uses Character.isWhitespace on a char. The port checks the ASCII set,
// which is what appears between angle brackets in a PDF.
func isPDFWhitespaceByte(b byte) bool {
	switch b {
	case ' ', '\t', '\n', '\r', '\f', '\v':
		return true
	}
	return false
}

// hexValue returns the value of a hex digit, or -1 if it is not one.
//
// Port of org.apache.pdfbox.util.Hex.getHexValue. That class will move to
// pdfbox/util when more of it is needed; only this and the digit table are used
// so far.
func hexValue(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return -1
}

// ForceHexForm reports whether the string is written in hex regardless of its
// content.
func (s *StringObj) ForceHexForm() bool { return s.forceHexForm }

// Value decodes the string.
//
// Port of getString(). A leading byte order mark selects UTF-16, big or little
// endian; anything else is PDFDocEncoding. UTF-16LE is not in the PDF spec but
// occurs in the wild, and Java accepts it.
func (s *StringObj) Value() string {
	if len(s.bytes) >= 2 {
		if s.bytes[0] == 0xFE && s.bytes[1] == 0xFF {
			return decodeUTF16(s.bytes[2:], true)
		}
		if s.bytes[0] == 0xFF && s.bytes[1] == 0xFE {
			return decodeUTF16(s.bytes[2:], false)
		}
	}
	return pdfDocEncodingToString(s.bytes)
}

// decodeUTF16 is Java's new String(bytes, UTF_16BE) or UTF_16LE, which decodes
// through a CharsetDecoder that replaces what it cannot read with U+FFFD.
//
// Port of sun.nio.cs.UnicodeDecoder.decodeLoop and the end of input
// CharsetDecoder.decode reports. A high surrogate and the unit after it are one
// malformed sequence of four bytes when that unit is not a low surrogate, so the
// unit is lost with it; a low surrogate on its own is two malformed bytes; and
// whatever is left when the input ends, an odd byte or a high surrogate with
// nothing after it, is one malformed sequence. utf16.Decode, which the port used
// before, keeps the unit after an unpaired high surrogate, and the odd last byte
// never reached it.
func decodeUTF16(b []byte, bigEndian bool) string {
	unit := func(at int) rune {
		if bigEndian {
			return rune(b[at])<<8 | rune(b[at+1])
		}
		return rune(b[at+1])<<8 | rune(b[at])
	}
	var out strings.Builder
	at := 0
	for len(b)-at > 1 {
		c := unit(at)
		if !utf16.IsSurrogate(c) {
			out.WriteRune(c)
			at += 2
			continue
		}
		if c <= 0xDBFF {
			if len(b)-at < 4 {
				break
			}
			if c2 := unit(at + 2); c2 >= 0xDC00 && c2 <= 0xDFFF {
				out.WriteRune(utf16.DecodeRune(c, c2))
			} else {
				out.WriteRune(utf8.RuneError)
			}
			at += 4
			continue
		}
		// Unpaired low surrogate
		out.WriteRune(utf8.RuneError)
		at += 2
	}
	if at < len(b) {
		out.WriteRune(utf8.RuneError)
	}
	return out.String()
}

// ASCII decodes the string as US-ASCII.
//
// Port of getASCII().
func (s *StringObj) ASCII() string {
	runes := make([]rune, len(s.bytes))
	for i, b := range s.bytes {
		if b > 0x7F {
			// Java's US_ASCII decoder substitutes for bytes above 0x7F
			runes[i] = '�'
			continue
		}
		runes[i] = rune(b)
	}
	return string(runes)
}

// Bytes returns a copy of the raw bytes.
func (s *StringObj) Bytes() []byte {
	out := make([]byte, len(s.bytes))
	copy(out, s.bytes)
	return out
}

// ToHexString returns the bytes as upper-case hex, two digits per byte.
func (s *StringObj) ToHexString() string {
	out := make([]byte, 0, len(s.bytes)*2)
	for _, b := range s.bytes {
		out = append(out, hexDigits[b>>4], hexDigits[b&0x0F])
	}
	return string(out)
}

// COSObject returns the receiver.
func (s *StringObj) COSObject() Base { return s }

// Accept dispatches to the visitor.
func (s *StringObj) Accept(v Visitor) error { return v.VisitStringObj(s) }

// Equals compares the decoded values and the hex-form flag, as Java does.
//
// Note that this is not the same as comparing the bytes: two strings encoding
// the same text differently are equal here.
func (s *StringObj) Equals(other *StringObj) bool {
	return other != nil &&
		s.Value() == other.Value() &&
		s.forceHexForm == other.forceHexForm
}

// String returns the Java toString form.
func (s *StringObj) String() string { return "COSString{" + s.Value() + "}" }
