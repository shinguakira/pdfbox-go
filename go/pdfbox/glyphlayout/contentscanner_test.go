package glyphlayout_test

// A content stream reader, just enough of one to read back what the layout
// wrote.
//
// The port has a full one in pdfbox/contentstream, but it is a rendering engine
// that resolves graphics state, and what these tests need is the operators as
// they were written -- including the string operands byte for byte, which is
// where the glyph codes are.

import (
	"strings"
	"unicode/utf16"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// operatorScanner walks a content stream, answering one operator at a time
// with the operands in front of it.
type operatorScanner struct {
	contents string
	at       int
}

// next answers the next operator and its operands, and false at the end.
func (s *operatorScanner) next() (string, []string, bool) {
	var operands []string
	for {
		token, ok := s.token()
		if !ok {
			return "", nil, false
		}
		if isOperand(token) {
			operands = append(operands, token)
			continue
		}
		return token, operands, true
	}
}

// operands answers every operand of the rest of the stream, which is what the
// inside of an array is.
func (s *operatorScanner) operands() []string {
	var all []string
	for {
		token, ok := s.token()
		if !ok {
			return all
		}
		all = append(all, token)
	}
}

// isOperand reports whether a token is a value rather than an operator.
func isOperand(token string) bool {
	if token == "" {
		return false
	}
	switch token[0] {
	case '(', '<', '[', '/', '+', '-', '.':
		return true
	}
	return token[0] >= '0' && token[0] <= '9'
}

// token answers the next token, and false at the end of the stream.
func (s *operatorScanner) token() (string, bool) {
	for s.at < len(s.contents) && isWhitespace(s.contents[s.at]) {
		s.at++
	}
	if s.at >= len(s.contents) {
		return "", false
	}
	start := s.at
	switch s.contents[s.at] {
	case '(':
		s.at = endOfLiteralString(s.contents, s.at)
	case '[':
		s.at = endOfArray(s.contents, s.at)
	case '<':
		if end := strings.IndexByte(s.contents[s.at:], '>'); end >= 0 {
			s.at += end + 1
		} else {
			s.at = len(s.contents)
		}
	default:
		for s.at < len(s.contents) && !isWhitespace(s.contents[s.at]) &&
			!strings.ContainsRune("()[]<>/", rune(s.contents[s.at])) {
			s.at++
		}
		if s.at == start {
			// A delimiter this reader does not take apart -- a dictionary, or
			// the name after a slash. Step over it so the walk terminates.
			s.at++
			if s.contents[start] == '/' {
				for s.at < len(s.contents) && !isWhitespace(s.contents[s.at]) &&
					!strings.ContainsRune("()[]<>/", rune(s.contents[s.at])) {
					s.at++
				}
			}
		}
	}
	return s.contents[start:s.at], true
}

// endOfArray answers the index just past the array that starts at open.
func endOfArray(contents string, open int) int {
	depth := 0
	for i := open; i < len(contents); i++ {
		switch contents[i] {
		case '(':
			i = endOfLiteralString(contents, i) - 1
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}
	return len(contents)
}

// isWhitespace reports whether a byte separates tokens, as PDF defines it.
func isWhitespace(b byte) bool {
	switch b {
	case 0, '\t', '\n', '\f', '\r', ' ':
		return true
	}
	return false
}

// stringBytesOf answers the bytes a string operand carries, undoing the
// escapes a literal string is written with.
func stringBytesOf(operand string) []byte {
	operand = strings.TrimSpace(operand)
	if strings.HasPrefix(operand, "<") {
		return hexBytesOf(strings.TrimSuffix(strings.TrimPrefix(operand, "<"), ">"))
	}
	inner := strings.TrimSuffix(strings.TrimPrefix(operand, "("), ")")
	var out []byte
	for i := 0; i < len(inner); i++ {
		if inner[i] != '\\' {
			out = append(out, inner[i])
			continue
		}
		i++
		if i >= len(inner) {
			break
		}
		switch inner[i] {
		case 'n':
			out = append(out, '\n')
		case 'r':
			out = append(out, '\r')
		case 't':
			out = append(out, '\t')
		case 'b':
			out = append(out, '\b')
		case 'f':
			out = append(out, '\f')
		case '0', '1', '2', '3', '4', '5', '6', '7':
			value := 0
			digits := 0
			for digits < 3 && i < len(inner) && inner[i] >= '0' && inner[i] <= '7' {
				value = value*8 + int(inner[i]-'0')
				i++
				digits++
			}
			i--
			out = append(out, byte(value))
		default:
			out = append(out, inner[i])
		}
	}
	return out
}

// hexBytesOf answers the bytes of a hexadecimal string operand.
func hexBytesOf(digits string) []byte {
	var out []byte
	value, half := 0, false
	for _, digit := range digits {
		nibble := -1
		switch {
		case digit >= '0' && digit <= '9':
			nibble = int(digit - '0')
		case digit >= 'a' && digit <= 'f':
			nibble = int(digit-'a') + 10
		case digit >= 'A' && digit <= 'F':
			nibble = int(digit-'A') + 10
		default:
			continue
		}
		value = value<<4 | nibble
		if half {
			out = append(out, byte(value))
			value, half = 0, false
			continue
		}
		half = true
	}
	if half {
		// An odd number of digits is padded with a zero, which is what the
		// specification says to do.
		out = append(out, byte(value<<4))
	}
	return out
}

// utf16UnitsOf answers the UTF-16 code units of a string, which is what a
// ToUnicode entry is made of and what the recorded dumps print.
func utf16UnitsOf(text string) []uint16 { return utf16.Encode([]rune(text)) }

// trimName strips the leading slash of a name operand.
func trimName(operand string) string { return strings.TrimPrefix(operand, "/") }

// cosNameOf answers the COS name of a resource.
func cosNameOf(name string) *cos.Name { return cos.GetPDFName(name) }
