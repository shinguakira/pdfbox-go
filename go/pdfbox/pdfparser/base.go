// Package pdfparser reads the byte-level syntax of a PDF file.
//
// Port of org.apache.pdfbox.pdfparser. AGENTS.md flags parsing and
// cross-reference recovery as historically bug-prone, so this package is ported
// line for line rather than rewritten; the recovery paths in particular exist
// for files that real producers emit and nothing in the specification predicts.
package pdfparser

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// maxLengthLong is the number of digits in the largest int64, used to stop
// reading a number long before it can become a denial of service.
var maxLengthLong = len(strconv.FormatInt(1<<63-1, 10))

// ASCII values the lexer tests for, named as they are in BaseParser.
const (
	asciiNull  = 0
	asciiTab   = 9
	asciiLF    = 10
	asciiFF    = 12
	asciiCR    = 13
	asciiSpace = 32
)

// eof is what a read past the end reports, standing in for the -1 Java returns.
const eof = -1

// BaseParser reads the lexical elements of a PDF: whitespace, comments, names,
// numbers and literal strings.
//
// Port of the abstract class org.apache.pdfbox.pdfparser.BaseParser. Java uses
// inheritance to give the parsers above it these methods; the port embeds this
// struct instead, per migration/conventions/java-to-go.md.
type BaseParser struct {
	source pdfio.RandomAccessRead
}

// NewBaseParser returns a parser reading from source.
func NewBaseParser(source pdfio.RandomAccessRead) *BaseParser {
	return &BaseParser{source: source}
}

// Source returns the underlying source, for the parsers built on this one.
func (p *BaseParser) Source() pdfio.RandomAccessRead { return p.source }

// readByte reads one byte, reporting eof rather than an error at the end of the
// source. Java's read() returns -1 there, and the whole lexer is written
// against that.
func (p *BaseParser) readByte() (int, error) {
	b, err := p.source.ReadByte()
	if errors.Is(err, io.EOF) {
		return eof, nil
	}
	if err != nil {
		return eof, err
	}
	return int(b), nil
}

// rewind steps the source back n bytes.
func (p *BaseParser) rewind(n int64) error {
	return pdfio.Rewind(p.source, n)
}

// IsEOF reports whether the source is exhausted.
func (p *BaseParser) IsEOF() (bool, error) {
	return p.source.IsEOF()
}

// IsEOL reports whether c is a carriage return or a line feed.
func IsEOL(c int) bool { return IsLF(c) || IsCR(c) }

// IsLF reports whether c is a line feed.
func IsLF(c int) bool { return c == asciiLF }

// IsCR reports whether c is a carriage return.
func IsCR(c int) bool { return c == asciiCR }

// IsWhitespace reports whether c is PDF whitespace.
//
// Port of the static isWhitespace(int). Note that NUL counts, which is not
// obvious and matters for files padded with it.
func IsWhitespace(c int) bool {
	switch c {
	case asciiNull, asciiTab, asciiLF, asciiFF, asciiCR, asciiSpace:
		return true
	}
	return false
}

// isSpace reports whether c is an ordinary space.
func isSpace(c int) bool { return c == asciiSpace }

// IsDigit reports whether c is an ASCII digit.
func IsDigit(c int) bool { return c >= '0' && c <= '9' }

// IsEndOfName reports whether c terminates a name token.
//
// Port of the static isEndOfName(int).
func IsEndOfName(c int) bool {
	switch c {
	case asciiSpace, asciiCR, asciiLF, asciiTab,
		'>', '<', '[', '/', ']', ')', '(',
		asciiNull, '\f', '%', eof:
		return true
	}
	return false
}

// SkipSpaces advances past whitespace and comments.
//
// Port of skipSpaces. A '%' begins a comment that runs to the end of the line.
func (p *BaseParser) SkipSpaces() error {
	c, err := p.readByte()
	if err != nil {
		return err
	}
	for IsWhitespace(c) || c == '%' {
		if c == '%' {
			// skip past the comment
			for {
				c, err = p.readByte()
				if err != nil {
					return err
				}
				if IsEOL(c) || c == eof {
					break
				}
			}
		} else {
			c, err = p.readByte()
			if err != nil {
				return err
			}
		}
	}
	if c != eof {
		return p.rewind(1)
	}
	return nil
}

// SkipWhiteSpaces advances past the line break that follows the "stream"
// keyword.
//
// Port of skipWhiteSpaces. The specification allows only CRLF or LF there, but
// some producers insert spaces first — the Java comment names
// brother_scan_cover.pdf — so spaces are consumed before looking for the break.
func (p *BaseParser) SkipWhiteSpaces() error {
	c, err := p.readByte()
	if err != nil {
		return err
	}
	for isSpace(c) {
		c, err = p.readByte()
		if err != nil {
			return err
		}
	}
	skipped, err := p.skipLinebreakByte(c)
	if err != nil {
		return err
	}
	if !skipped {
		return p.rewind(1)
	}
	return nil
}

// SkipLinebreak consumes a CR, LF or CRLF, reporting whether one was there.
func (p *BaseParser) SkipLinebreak() (bool, error) {
	c, err := p.readByte()
	if err != nil {
		return false, err
	}
	skipped, err := p.skipLinebreakByte(c)
	if err != nil {
		return false, err
	}
	if !skipped {
		return false, p.rewind(1)
	}
	return true, nil
}

// skipLinebreakByte handles an already-read byte, consuming the LF of a CRLF.
func (p *BaseParser) skipLinebreakByte(c int) (bool, error) {
	if IsCR(c) {
		next, err := p.readByte()
		if err != nil {
			return false, err
		}
		if !IsLF(next) {
			if next != eof {
				if err := p.rewind(1); err != nil {
					return false, err
				}
			}
		}
		return true, nil
	}
	return IsLF(c), nil
}

// ReadString reads a token up to the next name delimiter.
//
// Port of readString. The delimiter is left unread.
func (p *BaseParser) ReadString() (string, error) {
	if err := p.SkipSpaces(); err != nil {
		return "", err
	}
	var buf bytes.Buffer
	c, err := p.readByte()
	if err != nil {
		return "", err
	}
	for !IsEndOfName(c) {
		buf.WriteByte(byte(c))
		c, err = p.readByte()
		if err != nil {
			return "", err
		}
	}
	if c != eof {
		if err := p.rewind(1); err != nil {
			return "", err
		}
	}
	return buf.String(), nil
}

// ReadStringNumber reads a run of digits.
//
// Port of readStringNumber. It refuses a number with more digits than the
// largest int64 has, rather than reading until memory runs out — the guard
// exists because the length is attacker-controlled.
func (p *BaseParser) ReadStringNumber() (string, error) {
	var buf bytes.Buffer
	for {
		c, err := p.readByte()
		if err != nil {
			return "", err
		}
		if !IsDigit(c) {
			if c != eof {
				if err := p.rewind(1); err != nil {
					return "", err
				}
			}
			return buf.String(), nil
		}
		buf.WriteByte(byte(c))
		if buf.Len() > maxLengthLong {
			pos, _ := p.source.Position()
			return "", fmt.Errorf("pdfparser: number %q is getting too long, stopped at offset %d",
				buf.String(), pos)
		}
	}
}

// ReadInt reads a decimal integer.
//
// Port of readInt. On failure Java rewinds by the length of what it read before
// throwing, so a caller can try something else at the same offset; the port
// does the same.
func (p *BaseParser) ReadInt() (int, error) {
	if err := p.SkipSpaces(); err != nil {
		return 0, err
	}
	digits, err := p.ReadStringNumber()
	if err != nil {
		return 0, err
	}
	value, convErr := strconv.Atoi(digits)
	if convErr != nil {
		if err := p.rewind(int64(len(digits))); err != nil {
			return 0, err
		}
		pos, _ := p.source.Position()
		return 0, fmt.Errorf("pdfparser: expected an integer at offset %d, got %q: %w",
			pos, digits, convErr)
	}
	return value, nil
}

// ReadLong reads a decimal long integer.
//
// Port of readLong, with the same rewind-before-failing behaviour as ReadInt.
func (p *BaseParser) ReadLong() (int64, error) {
	if err := p.SkipSpaces(); err != nil {
		return 0, err
	}
	digits, err := p.ReadStringNumber()
	if err != nil {
		return 0, err
	}
	value, convErr := strconv.ParseInt(digits, 10, 64)
	if convErr != nil {
		if err := p.rewind(int64(len(digits))); err != nil {
			return 0, err
		}
		pos, _ := p.source.Position()
		return 0, fmt.Errorf("pdfparser: expected a long at offset %d, got %q: %w",
			pos, digits, convErr)
	}
	return value, nil
}

// ReadExpectedString consumes an exact keyword, failing if it is not there.
func (p *BaseParser) ReadExpectedString(expected string, skipSpaces bool) error {
	if skipSpaces {
		if err := p.SkipSpaces(); err != nil {
			return err
		}
	}
	for i := 0; i < len(expected); i++ {
		c, err := p.readByte()
		if err != nil {
			return err
		}
		if c != int(expected[i]) {
			pos, _ := p.source.Position()
			return fmt.Errorf("pdfparser: expected %q but missed at character %q at offset %d",
				expected, rune(expected[i]), pos)
		}
	}
	if skipSpaces {
		return p.SkipSpaces()
	}
	return nil
}

// ReadExpectedChar consumes one exact character, failing if it is not there.
func (p *BaseParser) ReadExpectedChar(expected byte) error {
	c, err := p.readByte()
	if err != nil {
		return err
	}
	if c != int(expected) {
		pos, _ := p.source.Position()
		return fmt.Errorf("pdfparser: expected %q but got %q at offset %d",
			rune(expected), rune(c), pos)
	}
	return nil
}

// checkForEndOfString decides whether an unbalanced parenthesis really ends the
// string.
//
// Port of the private checkForEndOfString. A closing parenthesis that leaves
// the brace count above zero still ends the string when what follows looks like
// the start of the next object or the end of the dictionary: a line break then
// '/' or '>'. Producers emit strings whose parentheses do not balance, and
// without this the parser swallows the rest of the file.
func (p *BaseParser) checkForEndOfString(braces int) (int, error) {
	if braces == 0 {
		return 0, nil
	}

	next := make([]byte, 3)
	n, err := p.source.Read(next)
	if err != nil && !errors.Is(err, io.EOF) {
		return braces, err
	}
	if n > 0 {
		if err := p.rewind(int64(n)); err != nil {
			return braces, err
		}
	}
	if n < 3 {
		return braces, nil
	}

	first, second, third := int(next[0]), int(next[1]), int(next[2])
	// Six shapes end the string, per the Java comment:
	//   CR LF '/'   CR LF '>'   LF '/'   LF '>'   CR '/'   CR '>'
	if (IsEOL(first) && (second == '/' || second == '>')) ||
		(IsCR(first) && IsLF(second) && (third == '/' || third == '>')) {
		return 0, nil
	}
	return braces, nil
}

// ReadLiteralString reads a parenthesised string, resolving its escapes.
//
// Port of readLiteralString.
func (p *BaseParser) ReadLiteralString() ([]byte, error) {
	if err := p.ReadExpectedChar('('); err != nil {
		return nil, err
	}

	var out bytes.Buffer
	braces := 1
	c, err := p.readByte()
	if err != nil {
		return nil, err
	}

	for braces > 0 && c != eof {
		// nextc holds a byte already read by an escape, so that the loop does
		// not read past it. Java uses -2 for "not yet read".
		nextc := -2

		switch {
		case c == ')':
			braces--
			braces, err = p.checkForEndOfString(braces)
			if err != nil {
				return nil, err
			}
			if braces != 0 {
				out.WriteByte(byte(c))
			}

		case c == '(':
			braces++
			out.WriteByte(byte(c))

		case c == '\\':
			next, err := p.readByte()
			if err != nil {
				return nil, err
			}
			nextc, err = p.readEscape(&out, next, &braces)
			if err != nil {
				return nil, err
			}

		default:
			out.WriteByte(byte(c))
		}

		if nextc != -2 {
			c = nextc
		} else {
			c, err = p.readByte()
			if err != nil {
				return nil, err
			}
		}
	}

	if c != eof {
		if err := p.rewind(1); err != nil {
			return nil, err
		}
	}
	return out.Bytes(), nil
}

// readEscape handles the character after a backslash, returning a byte the
// caller should treat as already read, or -2 for none.
func (p *BaseParser) readEscape(out *bytes.Buffer, next int, braces *int) (int, error) {
	switch next {
	case 'n':
		out.WriteByte('\n')
	case 'r':
		out.WriteByte('\r')
	case 't':
		out.WriteByte('\t')
	case 'b':
		out.WriteByte('\b')
	case 'f':
		out.WriteByte('\f')

	case ')':
		// PDFBOX-276: /Title (c:\) — the parenthesis may really close the
		// string, in which case the backslash was literal.
		updated, err := p.checkForEndOfString(*braces)
		if err != nil {
			return -2, err
		}
		*braces = updated
		if updated != 0 {
			out.WriteByte(byte(next))
		} else {
			out.WriteByte('\\')
		}

	case '(', '\\':
		out.WriteByte(byte(next))

	case asciiLF, asciiCR:
		// a break in the line: skip it and everything that follows it
		c, err := p.readByte()
		if err != nil {
			return -2, err
		}
		for IsEOL(c) && c != eof {
			c, err = p.readByte()
			if err != nil {
				return -2, err
			}
		}
		return c, nil

	case '0', '1', '2', '3', '4', '5', '6', '7':
		return p.readOctalEscape(out, next)

	default:
		// The backslash is dropped and the character kept; see 7.3.4.2,
		// Literal Strings.
		out.WriteByte(byte(next))
	}
	return -2, nil
}

// readOctalEscape reads up to three octal digits after a backslash.
func (p *BaseParser) readOctalEscape(out *bytes.Buffer, first int) (int, error) {
	octal := []byte{byte(first)}
	nextc := -2

	c, err := p.readByte()
	if err != nil {
		return -2, err
	}
	if c >= '0' && c <= '7' {
		octal = append(octal, byte(c))
		c, err = p.readByte()
		if err != nil {
			return -2, err
		}
		if c >= '0' && c <= '7' {
			octal = append(octal, byte(c))
		} else {
			nextc = c
		}
	} else {
		nextc = c
	}

	value, convErr := strconv.ParseInt(string(octal), 8, 32)
	if convErr != nil {
		return -2, fmt.Errorf("pdfparser: expected an octal character, got %q: %w",
			string(octal), convErr)
	}
	out.WriteByte(byte(value))
	return nextc, nil
}
