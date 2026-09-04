// Package type1 reads Adobe Type 1 fonts.
//
// Port of org.apache.fontbox.type1.
package type1

import (
	"fmt"
	"strconv"
)

// tokenKind is one of all the different types of tokens.
//
// Port of org.apache.fontbox.type1.Token.Kind. Java also declares a static
// alias for each constant on Token itself, purely for convenience at the call
// site; Go needs no such thing.
type tokenKind int

const (
	kindNone tokenKind = iota
	kindString
	kindName
	kindLiteral
	kindReal
	kindInteger
	kindStartArray
	kindEndArray
	kindStartProc
	kindEndProc
	kindStartDict
	kindEndDict
	kindCharstring
)

// String names the kind, for the messages a token appears in.
func (k tokenKind) String() string {
	switch k {
	case kindNone:
		return "NONE"
	case kindString:
		return "STRING"
	case kindName:
		return "NAME"
	case kindLiteral:
		return "LITERAL"
	case kindReal:
		return "REAL"
	case kindInteger:
		return "INTEGER"
	case kindStartArray:
		return "START_ARRAY"
	case kindEndArray:
		return "END_ARRAY"
	case kindStartProc:
		return "START_PROC"
	case kindEndProc:
		return "END_PROC"
	case kindStartDict:
		return "START_DICT"
	case kindEndDict:
		return "END_DICT"
	case kindCharstring:
		return "CHARSTRING"
	}
	return strconv.Itoa(int(k))
}

// token is a lexical token in an Adobe Type 1 font.
//
// Port of org.apache.fontbox.type1.Token. See type1Lexer.
type token struct {
	text string
	data []byte
	kind tokenKind
}

// newToken constructs a new token given its text and kind.
func newToken(text string, kind tokenKind) *token {
	return &token{text: text, kind: kind}
}

// newCharToken constructs a new token given its single-character text and kind.
func newCharToken(character rune, kind tokenKind) *token {
	return &token{text: string(character), kind: kind}
}

// newDataToken constructs a new token given its raw data and kind. This is for
// CHARSTRING tokens only.
func newDataToken(data []byte, kind tokenKind) *token {
	return &token{data: data, kind: kind}
}

// Text returns the token's text.
func (t *token) Text() string { return t.text }

// Kind returns the token's kind.
func (t *token) Kind() tokenKind { return t.kind }

// IntValue returns the token's text as an int.
func (t *token) IntValue() int {
	// some fonts have reals where integers should be, so we tolerate it.
	//
	// Java parses at float precision and casts, which truncates towards zero
	// and saturates at the int range rather than wrapping.
	return floatToInt(t.FloatValue())
}

// FloatValue returns the token's text as a float.
func (t *token) FloatValue() float32 {
	value, err := strconv.ParseFloat(t.text, 32)
	if err != nil {
		// Java's Float.parseFloat throws NumberFormatException, which is
		// unchecked and so travels straight out of the caller.
		panic(fmt.Sprintf("type1: For input string: %q", t.text))
	}
	return float32(value)
}

// BooleanValue returns whether the token's text is "true".
func (t *token) BooleanValue() bool { return t.text == "true" }

// Data returns the token's raw data.
func (t *token) Data() []byte { return t.data }

// String describes the token.
func (t *token) String() string {
	if t.kind == kindCharstring {
		return fmt.Sprintf("Token[kind=CHARSTRING, data=%d bytes]", len(t.data))
	}
	return fmt.Sprintf("Token[kind=%v, text=%s]", t.kind, t.text)
}

// floatToInt narrows a float to an int the way a Java (int) cast does: towards
// zero, with anything past the range clamped to it and NaN going to zero.
func floatToInt(value float32) int {
	switch {
	case value != value: // NaN
		return 0
	case value >= 2147483647:
		return 2147483647
	case value <= -2147483648:
		return -2147483648
	}
	return int(value)
}
