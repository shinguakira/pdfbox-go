package pdfparser

import (
	"bytes"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// Written from pdfbox/src/main/java/org/apache/pdfbox/pdfparser/BaseParser.java.
//
// The Java suite has no BaseParserTest — the lexer is exercised only through
// the parsers above it. Per migration/conventions/tdd.md, untested Java still
// gets a test written first from the source. Every expectation here comes from
// reading BaseParser, and the PDFBOX-numbered cases are the ones its comments
// name.

func newParser(input string) *BaseParser {
	return NewBaseParser(pdfio.NewReadBufferBytes([]byte(input)))
}

func TestIsEndOfName(t *testing.T) {
	// The delimiters that terminate a name, from BaseParser.isEndOfName.
	for _, c := range []int{' ', '\r', '\n', '\t', '>', '<', '[', '/', ']', ')', '(', 0, '\f', '%', -1} {
		if !IsEndOfName(c) {
			t.Errorf("IsEndOfName(%d) = false, want true", c)
		}
	}
	for _, c := range []int{'a', 'Z', '0', '9', '+', '-', '#', '.'} {
		if IsEndOfName(c) {
			t.Errorf("IsEndOfName(%q) = true, want false", rune(c))
		}
	}
}

func TestIsWhitespace(t *testing.T) {
	// Java treats NUL, tab, LF, FF, CR and space as whitespace.
	for _, c := range []int{0, 9, 10, 12, 13, 32} {
		if !IsWhitespace(c) {
			t.Errorf("IsWhitespace(%d) = false, want true", c)
		}
	}
	for _, c := range []int{'a', '/', 11, -1} {
		if IsWhitespace(c) {
			t.Errorf("IsWhitespace(%d) = true, want false", c)
		}
	}
}

func TestIsDigitAndEOL(t *testing.T) {
	for c := '0'; c <= '9'; c++ {
		if !IsDigit(int(c)) {
			t.Errorf("IsDigit(%q) = false, want true", c)
		}
	}
	if IsDigit('a') || IsDigit(-1) {
		t.Error("IsDigit accepted a non-digit")
	}

	if !IsEOL('\r') || !IsEOL('\n') {
		t.Error("IsEOL rejected CR or LF")
	}
	if IsEOL(' ') {
		t.Error("IsEOL(' ') = true")
	}
}

func TestSkipSpaces(t *testing.T) {
	t.Run("plain whitespace", func(t *testing.T) {
		p := newParser("   \t\r\n/Name")
		if err := p.SkipSpaces(); err != nil {
			t.Fatalf("SkipSpaces: %v", err)
		}
		b, err := p.source.ReadByte()
		if err != nil || b != '/' {
			t.Errorf("stopped at %q, %v; want '/'", b, err)
		}
	})

	t.Run("comments are skipped", func(t *testing.T) {
		// A % begins a comment that runs to the end of the line.
		p := newParser("  % this is a comment\r\n/Name")
		if err := p.SkipSpaces(); err != nil {
			t.Fatalf("SkipSpaces: %v", err)
		}
		b, err := p.source.ReadByte()
		if err != nil || b != '/' {
			t.Errorf("stopped at %q, %v; want '/'", b, err)
		}
	})

	t.Run("at end of input", func(t *testing.T) {
		p := newParser("   ")
		if err := p.SkipSpaces(); err != nil {
			t.Fatalf("SkipSpaces: %v", err)
		}
		if eof, _ := p.IsEOF(); !eof {
			t.Error("IsEOF() = false after skipping trailing spaces")
		}
	})
}

func TestSkipWhiteSpaces(t *testing.T) {
	// After the "stream" keyword the specification allows only CRLF or LF, but
	// some producers add spaces first; see the brother_scan_cover.pdf comment
	// in BaseParser.skipWhiteSpaces.
	cases := []struct {
		name  string
		input string
		want  byte
	}{
		{"LF", "\nDATA", 'D'},
		{"CRLF", "\r\nDATA", 'D'},
		{"CR alone", "\rDATA", 'D'},
		{"spaces then LF", "   \nDATA", 'D'},
		{"no line break at all", "DATA", 'D'},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := newParser(c.input)
			if err := p.SkipWhiteSpaces(); err != nil {
				t.Fatalf("SkipWhiteSpaces: %v", err)
			}
			b, err := p.source.ReadByte()
			if err != nil || b != c.want {
				t.Errorf("stopped at %q, %v; want %q", b, err, c.want)
			}
		})
	}
}

func TestReadString(t *testing.T) {
	p := newParser("  Hello/World")
	got, err := p.ReadString()
	if err != nil {
		t.Fatalf("ReadString: %v", err)
	}
	if got != "Hello" {
		t.Errorf("ReadString() = %q, want %q", got, "Hello")
	}
	// the delimiter is left unread
	b, _ := p.source.ReadByte()
	if b != '/' {
		t.Errorf("next byte = %q, want '/'", b)
	}
}

func TestReadInt(t *testing.T) {
	p := newParser("  1234 ")
	got, err := p.ReadInt()
	if err != nil {
		t.Fatalf("ReadInt: %v", err)
	}
	if got != 1234 {
		t.Errorf("ReadInt() = %d, want 1234", got)
	}
}

func TestReadIntRejectsNonNumber(t *testing.T) {
	p := newParser("  abc")
	before, _ := p.source.Position()
	if _, err := p.ReadInt(); err == nil {
		t.Fatal("ReadInt over a non-number succeeded, want an error")
	}
	// Java rewinds by the length of what it read before throwing, so the
	// caller can try something else at the same place.
	after, _ := p.source.Position()
	if after < before {
		t.Errorf("position moved backwards past the start: %d, was %d", after, before)
	}
}

func TestReadLong(t *testing.T) {
	p := newParser(" 9223372036854775807 ")
	got, err := p.ReadLong()
	if err != nil {
		t.Fatalf("ReadLong: %v", err)
	}
	if got != 9223372036854775807 {
		t.Errorf("ReadLong() = %d, want the maximum int64", got)
	}
}

// TestReadStringNumberTooLong covers the guard against a number long enough to
// be an attack rather than a document: BaseParser stops once the digits exceed
// the length of the largest int64.
func TestReadStringNumberTooLong(t *testing.T) {
	p := newParser(string(bytes.Repeat([]byte("9"), 40)))
	if _, err := p.ReadStringNumber(); err == nil {
		t.Fatal("ReadStringNumber accepted an over-long number, want an error")
	}
}

func TestReadExpectedString(t *testing.T) {
	p := newParser("  stream  ")
	if err := p.ReadExpectedString("stream", true); err != nil {
		t.Fatalf("ReadExpectedString: %v", err)
	}

	p = newParser("stresm")
	if err := p.ReadExpectedString("stream", false); err == nil {
		t.Fatal("ReadExpectedString accepted a mismatch, want an error")
	}
}

func TestReadExpectedChar(t *testing.T) {
	p := newParser("(")
	if err := p.ReadExpectedChar('('); err != nil {
		t.Fatalf("ReadExpectedChar: %v", err)
	}
	p = newParser("x")
	if err := p.ReadExpectedChar('('); err == nil {
		t.Fatal("ReadExpectedChar accepted a mismatch, want an error")
	}
}

func TestReadLiteralString(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"plain", "(hello)", "hello"},
		{"empty", "()", ""},
		{"nested parentheses are kept", "(a(b)c)", "a(b)c"},
		{"escaped newline", `(a\nb)`, "a\nb"},
		{"escaped carriage return", `(a\rb)`, "a\rb"},
		{"escaped tab", `(a\tb)`, "a\tb"},
		{"escaped backspace", `(a\bb)`, "a\bb"},
		{"escaped form feed", `(a\fb)`, "a\fb"},
		{"escaped parenthesis", `(a\(b\)c)`, "a(b)c"},
		{"escaped backslash", `(a\\b)`, `a\b`},
		{"unknown escape drops the backslash", `(a\qb)`, "aqb"},
		{"octal escape", `(\101\102)`, "AB"},
		{"short octal escape", `(\7)`, "\a"},
		{"two digit octal escape", `(\40)`, " "},
		// a backslash before a real line break continues the string
		{"line continuation", "(a\\\nb)", "ab"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := newParser(c.input)
			got, err := p.ReadLiteralString()
			if err != nil {
				t.Fatalf("ReadLiteralString: %v", err)
			}
			if string(got) != c.want {
				t.Errorf("ReadLiteralString() = %q, want %q", got, c.want)
			}
		})
	}
}

// TestReadLiteralStringUnbalanced covers the recovery BaseParser does for
// strings whose parentheses do not balance, which producers really emit. The
// comment in checkForEndOfString names the six line shapes it looks for; a
// closing parenthesis followed by a line break and then '/' or '>' ends the
// string even though the brace count says otherwise.
func TestReadLiteralStringUnbalanced(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		// An extra '(' pushes the brace count to 2, so the ')' would normally
		// leave the string open; the look-ahead closes it anyway.
		{"LF then slash", "(a(b)\n/Next", "a(b"},
		{"CRLF then angle", "(a(b)\r\n>>", "a(b"},
		{"CR then slash", "(a(b)\r/Next", "a(b"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := newParser(c.input)
			got, err := p.ReadLiteralString()
			if err != nil {
				t.Fatalf("ReadLiteralString: %v", err)
			}
			if string(got) != c.want {
				t.Errorf("ReadLiteralString() = %q, want %q", got, c.want)
			}
		})
	}
}

// TestReadLiteralStringPDFBOX276 covers the case the Java comment marks as
// "PDFBox 276 /Title (c:\)": a backslash before the closing parenthesis, where
// the parenthesis really does close the string.
func TestReadLiteralStringPDFBOX276(t *testing.T) {
	p := newParser("(c:\\)\n/Next")
	got, err := p.ReadLiteralString()
	if err != nil {
		t.Fatalf("ReadLiteralString: %v", err)
	}
	if string(got) != `c:\` {
		t.Errorf("ReadLiteralString() = %q, want %q", got, `c:\`)
	}
}
