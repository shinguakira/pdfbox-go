package type4

import "strings"

// The tokenizer of the calculator language.
//
// Port of org.apache.pdfbox.pdmodel.common.function.type4.Parser, whose
// Tokenizer is a private inner class and whose SyntaxHandler is an interface
// with an abstract implementation that ignores everything but tokens; the port
// keeps the interface and lets a handler embed nothing, since Go has no
// abstract class and the three ignored callbacks are one line each.

// parserState is Java's Parser.State.
type parserState int

const (
	stateNewline parserState = iota
	stateWhitespace
	stateComment
	stateToken
)

// syntaxHandler is told what the tokenizer found.
//
// Port of Parser.SyntaxHandler.
type syntaxHandler interface {
	newLine(text string)
	whitespace(text string)
	token(text string)
	comment(text string)
}

// parse runs the tokenizer over the input.
//
// Port of the static Parser.parse.
func parse(input string, handler syntaxHandler) {
	t := &tokenizer{input: input, handler: handler, state: stateWhitespace}
	t.tokenize()
}

// The characters the tokenizer singles out.
const (
	charNUL   = 0x00 // NUL
	charEOT   = 0x04 // END OF TRANSMISSION
	charTAB   = 0x09 // TAB CHARACTER
	charFF    = 0x0C // FORM FEED
	charCR    = '\r' // CARRIAGE RETURN
	charLF    = '\n' // LINE FEED
	charSPACE = ' '  // SPACE
)

// tokenizer is Java's Parser.Tokenizer.
//
// Java walks a CharSequence by char, which is a UTF-16 code unit; a type 4
// function is read as ISO-8859-1, so every char is one byte and the port walks
// bytes.
type tokenizer struct {
	input   string
	index   int
	handler syntaxHandler
	state   parserState
	buffer  strings.Builder
}

func (t *tokenizer) hasMore() bool { return t.index < len(t.input) }

func (t *tokenizer) currentChar() byte { return t.input[t.index] }

func (t *tokenizer) nextChar() byte {
	t.index++
	if !t.hasMore() {
		return charEOT
	}
	return t.currentChar()
}

func (t *tokenizer) peek() byte {
	if t.index < len(t.input)-1 {
		return t.input[t.index+1]
	}
	return charEOT
}

func (t *tokenizer) nextState() parserState {
	switch t.currentChar() {
	case charCR, charLF, charFF: // FF
		t.state = stateNewline
	case charNUL, charTAB, charSPACE:
		t.state = stateWhitespace
	case '%':
		t.state = stateComment
	default:
		t.state = stateToken
	}
	return t.state
}

func (t *tokenizer) tokenize() {
	for t.hasMore() {
		t.buffer.Reset()
		t.nextState()
		switch t.state {
		case stateNewline:
			t.scanNewLine()
		case stateWhitespace:
			t.scanWhitespace()
		case stateComment:
			t.scanComment()
		default:
			t.scanToken()
		}
	}
}

func (t *tokenizer) scanNewLine() {
	ch := t.currentChar()
	t.buffer.WriteByte(ch)
	if ch == charCR && t.peek() == charLF {
		// CRLF is treated as one newline
		t.buffer.WriteByte(t.nextChar())
	}
	t.handler.newLine(t.buffer.String())
	t.nextChar()
}

func (t *tokenizer) scanWhitespace() {
	t.buffer.WriteByte(t.currentChar())
	for t.hasMore() {
		ch := t.nextChar()
		switch ch {
		case charNUL, charTAB, charSPACE:
			t.buffer.WriteByte(ch)
		default:
			t.handler.whitespace(t.buffer.String())
			return
		}
	}
	t.handler.whitespace(t.buffer.String())
}

func (t *tokenizer) scanComment() {
	t.buffer.WriteByte(t.currentChar())
	for t.hasMore() {
		ch := t.nextChar()
		switch ch {
		case charCR, charLF, charFF:
			t.handler.comment(t.buffer.String())
			return
		default:
			t.buffer.WriteByte(ch)
		}
	}
	// EOF reached
	t.handler.comment(t.buffer.String())
}

func (t *tokenizer) scanToken() {
	ch := t.currentChar()
	t.buffer.WriteByte(ch)
	switch ch {
	case '{', '}':
		t.handler.token(t.buffer.String())
		t.nextChar()
		return
	}
	for t.hasMore() {
		ch = t.nextChar()
		switch ch {
		case charNUL, charTAB, charSPACE, charCR, charLF, charFF, charEOT, '{', '}':
			t.handler.token(t.buffer.String())
			return
		default:
			t.buffer.WriteByte(ch)
		}
	}
	// EOF reached
	t.handler.token(t.buffer.String())
}
