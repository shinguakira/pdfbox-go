package type1

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
)

// DamagedFontException is thrown when a font is damaged and cannot be read.
//
// Port of org.apache.fontbox.type1.DamagedFontException, which extends
// IOException; errors.As tells the two apart the way a catch clause does.
type DamagedFontException struct {
	Message string
}

func (e *DamagedFontException) Error() string { return e.Message }

// errPrematureEnd is what a read past the end of the buffer gives, standing in
// for the BufferUnderflowException Java catches and rethrows.
var errPrematureEnd = errors.New("Premature end of buffer reached")

// byteBuffer is the read cursor over the font's bytes.
//
// Java uses a java.nio.ByteBuffer, whose position, mark and reset the lexer
// leans on heavily; Go's readers have no mark, so the cursor is written out.
type byteBuffer struct {
	data     []byte
	position int
	mark     int
}

func (b *byteBuffer) hasRemaining() bool { return b.position < len(b.data) }

// get reads one byte, reporting the underflow Java throws at the end.
func (b *byteBuffer) get() (byte, error) {
	if b.position >= len(b.data) {
		return 0, errPrematureEnd
	}
	value := b.data[b.position]
	b.position++
	return value, nil
}

func (b *byteBuffer) setMark()      { b.mark = b.position }
func (b *byteBuffer) reset()        { b.position = b.mark }
func (b *byteBuffer) backUp()       { b.position-- }
func (b *byteBuffer) array() []byte { return b.data }

// type1Lexer is a lexer for the ASCII portions of an Adobe Type 1 font.
//
// See type1Parser.
//
// The PostScript language, of which Type 1 fonts are a subset, has a
// somewhat awkward lexical structure. It is neither regular nor
// context-free, and the execution of the program can modify the
// the behaviour of the lexer/parser.
//
// Nevertheless, this class represents an attempt to artificially separate
// the PostScript parsing process into separate lexing and parsing phases
// in order to reduce the complexity of the parsing phase.
//
// See "PostScript Language Reference 3rd ed, Adobe Systems (1999)".
//
// Port of org.apache.fontbox.type1.Type1Lexer.
type type1Lexer struct {
	buffer     byteBuffer
	aheadToken *token
	openParens int
}

// newType1Lexer constructs a new type1Lexer given a header-less .pfb segment.
func newType1Lexer(bytes []byte) (*type1Lexer, error) {
	l := &type1Lexer{buffer: byteBuffer{data: bytes}}
	aheadToken, err := l.readToken(nil)
	if err != nil {
		return nil, err
	}
	l.aheadToken = aheadToken
	return l, nil
}

// NextToken returns the next token and consumes it.
func (l *type1Lexer) NextToken() (*token, error) {
	curToken := l.aheadToken
	aheadToken, err := l.readToken(curToken)
	if err != nil {
		return nil, err
	}
	l.aheadToken = aheadToken
	return curToken, nil
}

// PeekToken returns the next token without consuming it.
func (l *type1Lexer) PeekToken() *token { return l.aheadToken }

// PeekKind checks if the kind of the next token equals the given one without
// consuming it.
func (l *type1Lexer) PeekKind(kind tokenKind) bool {
	return l.aheadToken != nil && l.aheadToken.Kind() == kind
}

// getChar reads an ASCII char from the buffer.
func (l *type1Lexer) getChar() (rune, error) {
	b, err := l.buffer.get()
	if err != nil {
		return 0, err
	}
	return rune(b), nil
}

// readToken reads a single token, prevToken being the previous token.
func (l *type1Lexer) readToken(prevToken *token) (*token, error) {
	skip := true
	for skip {
		skip = false
		for l.buffer.hasRemaining() {
			c, err := l.getChar()
			if err != nil {
				return nil, err
			}

			// delimiters
			switch {
			case c == '%':
				// comment
				if _, err := l.readComment(); err != nil {
					return nil, err
				}
			case c == '(':
				return l.readString()
			case c == ')':
				// not allowed outside a string context
				return nil, errors.New("unexpected closing parenthesis")
			case c == '[':
				return newCharToken(c, kindStartArray), nil
			case c == '{':
				return newCharToken(c, kindStartProc), nil
			case c == ']':
				return newCharToken(c, kindEndArray), nil
			case c == '}':
				return newCharToken(c, kindEndProc), nil
			case c == '/':
				regular, err := l.readRegular()
				if err != nil {
					return nil, err
				}
				if regular == "" {
					// the stream is corrupt
					return nil, &DamagedFontException{fmt.Sprintf(
						"Could not read token at position %d", l.buffer.position)}
				}
				return newToken(regular, kindLiteral), nil
			case c == '<':
				c2, err := l.getChar()
				if err != nil {
					return nil, err
				}
				if c2 == c {
					return newToken("<<", kindStartDict), nil
				}
				// code may have to be changed in something better, maybe new
				// token type
				l.buffer.backUp()
				return newCharToken(c, kindName), nil
			case c == '>':
				c2, err := l.getChar()
				if err != nil {
					return nil, err
				}
				if c2 == c {
					return newToken(">>", kindEndDict), nil
				}
				// code may have to be changed in something better, maybe new
				// token type
				l.buffer.backUp()
				return newCharToken(c, kindName), nil
			case isWhitespace(c):
				skip = true
			case c == 0:
				slog.Warn("NULL byte in font, skipped")
				skip = true
			default:
				l.buffer.backUp()

				// regular character: try parse as number
				number, err := l.tryReadNumber()
				if err != nil {
					return nil, err
				}
				if number != nil {
					return number, nil
				}
				// otherwise this must be a name
				name, err := l.readRegular()
				if err != nil {
					return nil, err
				}
				if name == "" {
					// the stream is corrupt
					return nil, &DamagedFontException{fmt.Sprintf(
						"Could not read token at position %d", l.buffer.position)}
				}

				if name == "RD" || name == "-|" {
					// return the next CharString instead
					if prevToken != nil && prevToken.Kind() == kindInteger {
						return l.readCharString(prevToken.IntValue())
					}
					return nil, errors.New("expected INTEGER before -| or RD")
				}
				return newToken(name, kindName), nil
			}
		}
	}
	return nil, nil
}

// tryReadNumber reads a number or returns nil.
func (l *type1Lexer) tryReadNumber() (*token, error) {
	l.buffer.setMark()

	var sb strings.Builder
	var radix *strings.Builder
	c, err := l.getChar()
	if err != nil {
		return nil, err
	}
	hasDigit := false

	// optional + or -
	if c == '+' || c == '-' {
		sb.WriteRune(c)
		if c, err = l.getChar(); err != nil {
			return nil, err
		}
	}

	// optional digits
	for isDigit(c) {
		sb.WriteRune(c)
		if c, err = l.getChar(); err != nil {
			return nil, err
		}
		hasDigit = true
	}

	// optional .
	switch {
	case c == '.':
		sb.WriteRune(c)
		if c, err = l.getChar(); err != nil {
			return nil, err
		}
	case c == '#':
		// PostScript radix number takes the form base#number
		radix = &strings.Builder{}
		radix.WriteString(sb.String())
		sb.Reset()
		if c, err = l.getChar(); err != nil {
			return nil, err
		}
	case sb.Len() == 0 || !hasDigit:
		// failure
		l.buffer.reset()
		return nil, nil
	case c != 'e' && c != 'E':
		// integer
		l.buffer.backUp()
		return newToken(sb.String(), kindInteger), nil
	}

	// required digit
	if isDigit(c) {
		sb.WriteRune(c)
		if c, err = l.getChar(); err != nil {
			return nil, err
		}
	} else if c != 'e' && c != 'E' {
		// failure
		l.buffer.reset()
		return nil, nil
	}

	// optional digits
	for isDigit(c) {
		sb.WriteRune(c)
		if c, err = l.getChar(); err != nil {
			return nil, err
		}
	}

	// optional E
	if c == 'E' || c == 'e' {
		sb.WriteRune(c)
		if c, err = l.getChar(); err != nil {
			return nil, err
		}

		// optional minus
		if c == '-' {
			sb.WriteRune(c)
			if c, err = l.getChar(); err != nil {
				return nil, err
			}
		}

		// required digit
		if isDigit(c) {
			sb.WriteRune(c)
			if c, err = l.getChar(); err != nil {
				return nil, err
			}
		} else {
			// failure
			l.buffer.reset()
			return nil, nil
		}

		// optional digits
		for isDigit(c) {
			sb.WriteRune(c)
			if c, err = l.getChar(); err != nil {
				return nil, err
			}
		}
	}

	l.buffer.backUp()
	if radix != nil {
		val, err := parseRadix(sb.String(), radix.String())
		if err != nil {
			return nil, fmt.Errorf("Invalid number '%s': %w", sb.String(), err)
		}
		return newToken(strconv.Itoa(val), kindInteger), nil
	}
	return newToken(sb.String(), kindReal), nil
}

// parseRadix is Integer.parseInt(text, Integer.parseInt(radix)), which rejects
// a radix outside 2 to 36 and a value outside the 32-bit range.
func parseRadix(text, radix string) (int, error) {
	base, err := strconv.ParseInt(radix, 10, 32)
	if err != nil {
		return 0, err
	}
	if base < 2 || base > 36 {
		return 0, fmt.Errorf("radix %d out of range", base)
	}
	value, err := strconv.ParseInt(text, int(base), 32)
	if err != nil {
		return 0, err
	}
	return int(value), nil
}

// readRegular reads a sequence of regular characters, i.e. not delimiters or
// whitespace. It gives the empty string where Java gives null.
func (l *type1Lexer) readRegular() (string, error) {
	var sb strings.Builder
	for l.buffer.hasRemaining() {
		l.buffer.setMark()
		c, err := l.getChar()
		if err != nil {
			return "", err
		}
		if isWhitespace(c) ||
			c == '(' || c == ')' ||
			c == '<' || c == '>' ||
			c == '[' || c == ']' ||
			c == '{' || c == '}' ||
			c == '/' || c == '%' {
			l.buffer.reset()
			break
		}
		sb.WriteRune(c)
	}
	return sb.String(), nil
}

// readComment reads a line comment.
func (l *type1Lexer) readComment() (string, error) {
	var sb strings.Builder
	for l.buffer.hasRemaining() {
		c, err := l.getChar()
		if err != nil {
			return "", err
		}
		if c == '\r' || c == '\n' {
			break
		}
		sb.WriteRune(c)
	}
	return sb.String(), nil
}

// readString reads a (string).
func (l *type1Lexer) readString() (*token, error) {
	var sb strings.Builder

	for l.buffer.hasRemaining() {
		c, err := l.getChar()
		if err != nil {
			return nil, err
		}

		// string context
		switch c {
		case '(':
			l.openParens++
			sb.WriteRune('(')
		case ')':
			if l.openParens == 0 {
				// end of string
				return newToken(sb.String(), kindString), nil
			}
			sb.WriteRune(')')
			l.openParens--
		case '\\':
			// escapes: \n \r \t \b \f \\ \( \)
			c1, err := l.getChar()
			if err != nil {
				return nil, err
			}
			switch c1 {
			case 'n', 'r':
				sb.WriteString("\n")
			case 't':
				sb.WriteRune('\t')
			case 'b':
				sb.WriteRune('\b')
			case 'f':
				sb.WriteRune('\f')
			case '\\':
				sb.WriteRune('\\')
			case '(':
				sb.WriteRune('(')
			case ')':
				sb.WriteRune(')')
			}
			// octal \ddd
			if isDigit(c1) {
				c2, err := l.getChar()
				if err != nil {
					return nil, err
				}
				c3, err := l.getChar()
				if err != nil {
					return nil, err
				}
				num := string([]rune{c1, c2, c3})
				code, err := strconv.ParseInt(num, 8, 32)
				if err != nil {
					return nil, fmt.Errorf("type1: %w", err)
				}
				sb.WriteRune(rune(code))
			}
		case '\r', '\n':
			sb.WriteString("\n")
		default:
			sb.WriteRune(c)
		}
	}
	return nil, nil
}

// readCharString reads a binary CharString.
func (l *type1Lexer) readCharString(length int) (*token, error) {
	if length > len(l.buffer.array()) {
		return nil, fmt.Errorf("String length %d is larger than input", length)
	}
	if _, err := l.buffer.get(); err != nil { // space
		return nil, err
	}
	if l.buffer.position+length > len(l.buffer.data) {
		return nil, errPrematureEnd
	}
	data := make([]byte, length)
	copy(data, l.buffer.data[l.buffer.position:l.buffer.position+length])
	l.buffer.position += length
	return newDataToken(data, kindCharstring), nil
}

// isWhitespace is Java's Character.isWhitespace for the byte values the lexer
// reads. Of the values below 256 that is the ASCII whitespace plus the four
// file, group, record and unit separators; the non-breaking space at 0xA0 is
// deliberately not one of them.
func isWhitespace(c rune) bool {
	switch c {
	case '\t', '\n', 0x0B, '\f', '\r', 0x1C, 0x1D, 0x1E, 0x1F, ' ':
		return true
	}
	return false
}

// isDigit is Java's Character.isDigit for the byte values the lexer reads,
// which below 256 is the ASCII digits alone.
func isDigit(c rune) bool { return c >= '0' && c <= '9' }
