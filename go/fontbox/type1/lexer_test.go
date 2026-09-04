package type1

import (
	"errors"
	"testing"
)

// Port of org.apache.fontbox.type1.Type1LexerTest.

// readTokens drains the lexer, as the Java test's private helper does.
func readTokens(t *testing.T, l *type1Lexer) ([]*token, error) {
	t.Helper()
	var tokens []*token
	for {
		nextToken, err := l.NextToken()
		if err != nil {
			return tokens, err
		}
		if nextToken == nil {
			return tokens, nil
		}
		tokens = append(tokens, nextToken)
	}
}

// newLexer builds a lexer over the ASCII of s.
func newLexer(t *testing.T, s string) *type1Lexer {
	t.Helper()
	l, err := newType1Lexer([]byte(s))
	if err != nil {
		t.Fatalf("newType1Lexer(%q): %v", s, err)
	}
	return l
}

// TestRealNumbers covers PDFBOX-5155: test real numbers.
func TestRealNumbers(t *testing.T) {
	s := "/FontMatrix [1e-3 0e-3 0e-3 -1E-03 0 0 1.23 -1.23 ] readonly def"
	tokens, err := readTokens(t, newLexer(t, s))
	if err != nil {
		t.Fatalf("readTokens: %v", err)
	}

	kinds := []tokenKind{
		kindLiteral, kindStartArray,
		kindReal, kindReal, kindReal, kindReal,
		kindInteger, kindInteger,
		kindReal, kindReal,
		kindEndArray, kindName, kindName,
	}
	if len(tokens) < len(kinds) {
		t.Fatalf("got %d tokens, want at least %d", len(tokens), len(kinds))
	}
	for i, want := range kinds {
		if got := tokens[i].Kind(); got != want {
			t.Errorf("token %d kind = %v, want %v", i, got, want)
		}
	}

	texts := map[int]string{
		0: "FontMatrix",
		2: "1e-3", 3: "0e-3", 4: "0e-3", 5: "-1E-03",
		6: "0", 7: "0", 8: "1.23", 9: "-1.23",
	}
	for i, want := range texts {
		if got := tokens[i].Text(); got != want {
			t.Errorf("token %d text = %q, want %q", i, got, want)
		}
	}
	if got := tokens[5].FloatValue(); got != -1e-3 {
		t.Errorf("token 5 float = %v, want %v", got, float32(-1e-3))
	}
}

func TestEmptyName(t *testing.T) {
	s := "dup 127 / put"
	_, err := readTokens(t, newLexer(t, s))
	if err == nil {
		t.Fatal("an empty name was accepted")
	}
	var damaged *DamagedFontException
	if !errors.As(err, &damaged) {
		t.Fatalf("error is %T, want a DamagedFontException", err)
	}
	if got, want := damaged.Error(), "Could not read token at position 9"; got != want {
		t.Errorf("error = %q, want %q", got, want)
	}
}

func TestProcAndNameAndDictAndString(t *testing.T) {
	s := "/ND {noaccess def} executeonly def \n 8#173 +2#110 \n%comment \n" +
		"<< (string \\n \\r \\t \\b \\f \\\\ \\( \\) \\123) >>"
	tokens, err := readTokens(t, newLexer(t, s))
	if err != nil {
		t.Fatalf("readTokens: %v", err)
	}

	want := []struct {
		kind tokenKind
		text string // "" where the Java test does not assert one
	}{
		{kindLiteral, "ND"},
		{kindStartProc, ""},
		{kindName, "noaccess"},
		{kindName, "def"},
		{kindEndProc, ""},
		{kindName, "executeonly"},
		{kindName, "def"},
		{kindInteger, "123"},
		{kindInteger, "6"},
		{kindStartDict, ""},
		// the Java expectation is a source literal whose \123 is the octal
		// escape for S
		{kindString, "string \n \n \t \b \f \\ ( ) S"},
		{kindEndDict, ""},
	}
	if len(tokens) < len(want) {
		t.Fatalf("got %d tokens, want at least %d", len(tokens), len(want))
	}
	for i, w := range want {
		if got := tokens[i].Kind(); got != w.kind {
			t.Errorf("token %d kind = %v, want %v", i, got, w.kind)
		}
		if w.text != "" {
			if got := tokens[i].Text(); got != w.text {
				t.Errorf("token %d text = %q, want %q", i, got, w.text)
			}
		}
	}
}

func TestData(t *testing.T) {
	s := "3 RD 123 ND"
	tokens, err := readTokens(t, newLexer(t, s))
	if err != nil {
		t.Fatalf("readTokens: %v", err)
	}
	if len(tokens) < 3 {
		t.Fatalf("got %d tokens, want at least 3", len(tokens))
	}
	if got := tokens[0].Kind(); got != kindInteger {
		t.Errorf("token 0 kind = %v, want %v", got, kindInteger)
	}
	if got := tokens[0].IntValue(); got != 3 {
		t.Errorf("token 0 int = %d, want 3", got)
	}
	if got := tokens[1].Kind(); got != kindCharstring {
		t.Errorf("token 1 kind = %v, want %v", got, kindCharstring)
	}
	if got := string(tokens[1].Data()); got != "123" {
		t.Errorf("token 1 data = %q, want %q", got, "123")
	}
	if got := tokens[2].Kind(); got != kindName {
		t.Errorf("token 2 kind = %v, want %v", got, kindName)
	}
	if got := tokens[2].Text(); got != "ND" {
		t.Errorf("token 2 text = %q, want %q", got, "ND")
	}
}

// TestPDFBOX6043 covers detection of illegal string length.
func TestPDFBOX6043(t *testing.T) {
	s := "999 RD"
	_, err := readTokens(t, newLexer(t, s))
	if err == nil {
		t.Fatal("a charstring longer than the input was accepted")
	}
	if got, want := err.Error(), "String length 999 is larger than input"; got != want {
		t.Errorf("error = %q, want %q", got, want)
	}
}
