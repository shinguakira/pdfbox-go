package pdfparser

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream/operator"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// Written from
// pdfbox/src/main/java/org/apache/pdfbox/pdfparser/PDFStreamParser.java. The
// Java suite has no test for this class, so per migration/conventions/tdd.md
// these are written from the source.

func parseTokens(t *testing.T, input string) []any {
	t.Helper()
	p, err := NewStreamTokenParser([]byte(input))
	if err != nil {
		t.Fatalf("NewStreamTokenParser: %v", err)
	}
	tokens, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse(%q): %v", input, err)
	}
	return tokens
}

func TestStreamTokenParserOperands(t *testing.T) {
	tokens := parseTokens(t, "1 0 0 1 100 200 cm")

	if len(tokens) != 7 {
		t.Fatalf("got %d tokens, want 7", len(tokens))
	}
	for i, want := range []int64{1, 0, 0, 1, 100, 200} {
		n, ok := tokens[i].(*cos.Integer)
		if !ok {
			t.Fatalf("token %d = %T, want *cos.Integer", i, tokens[i])
		}
		if n.LongValue() != want {
			t.Errorf("token %d = %d, want %d", i, n.LongValue(), want)
		}
	}
	op, ok := tokens[6].(*operator.Operator)
	if !ok {
		t.Fatalf("last token = %T, want *operator.Operator", tokens[6])
	}
	if op.Name() != operator.Concat {
		t.Errorf("operator = %q, want %q", op.Name(), operator.Concat)
	}
}

func TestStreamTokenParserTypes(t *testing.T) {
	tokens := parseTokens(t, "/F1 12 Tf (Hello) Tj [1 2] TJ <414243> Tj true false null")

	types := make([]string, len(tokens))
	for i, tok := range tokens {
		switch v := tok.(type) {
		case *operator.Operator:
			types[i] = "op:" + v.Name()
		case *cos.Name:
			types[i] = "name:" + v.Name()
		case *cos.Integer:
			types[i] = "int"
		case *cos.StringObj:
			types[i] = "string:" + v.Value()
		case *cos.Array:
			types[i] = "array"
		case *cos.Boolean:
			types[i] = "bool"
		case *cos.Null:
			types[i] = "null"
		default:
			types[i] = "?"
		}
	}

	want := []string{
		"name:F1", "int", "op:Tf",
		"string:Hello", "op:Tj",
		"array", "op:TJ",
		"string:ABC", "op:Tj",
		"bool", "bool", "null",
	}
	if len(types) != len(want) {
		t.Fatalf("got %d tokens %v, want %d %v", len(types), types, len(want), want)
	}
	for i := range want {
		if types[i] != want[i] {
			t.Errorf("token %d = %s, want %s", i, types[i], want[i])
		}
	}
}

func TestStreamTokenParserDictionary(t *testing.T) {
	tokens := parseTokens(t, "<< /Type /Page >> BDC")

	if len(tokens) != 2 {
		t.Fatalf("got %d tokens, want 2", len(tokens))
	}
	d, ok := tokens[0].(*cos.Dictionary)
	if !ok {
		t.Fatalf("token 0 = %T, want *cos.Dictionary", tokens[0])
	}
	if d.GetCOSName(cos.Type) != cos.Page {
		t.Errorf("/Type = %v, want /Page", d.GetCOSName(cos.Type))
	}
}

func TestStreamTokenParserRealNumbers(t *testing.T) {
	tokens := parseTokens(t, "1.5 -2.25 .5 +3")

	if len(tokens) != 4 {
		t.Fatalf("got %d tokens, want 4", len(tokens))
	}
	for i, want := range []float32{1.5, -2.25, 0.5, 3} {
		n, ok := tokens[i].(cos.Number)
		if !ok {
			t.Fatalf("token %d = %T, want a number", i, tokens[i])
		}
		if n.FloatValue() != want {
			t.Errorf("token %d = %v, want %v", i, n.FloatValue(), want)
		}
	}
}

// TestStreamTokenParserDoubleNegative pins the Java comment "Ignore double
// negative (this is consistent with Adobe Reader)".
func TestStreamTokenParserDoubleNegative(t *testing.T) {
	tokens := parseTokens(t, "--5")
	if len(tokens) != 1 {
		t.Fatalf("got %d tokens, want 1", len(tokens))
	}
	n, ok := tokens[0].(cos.Number)
	if !ok {
		t.Fatalf("token = %T, want a number", tokens[0])
	}
	if n.FloatValue() != -5 {
		t.Errorf("value = %v, want -5", n.FloatValue())
	}
}

// TestStreamTokenParserPDFBOX4064 covers a '-' in the middle of a number, which
// the Java drops.
func TestStreamTokenParserPDFBOX4064(t *testing.T) {
	tokens := parseTokens(t, "1-2")
	if len(tokens) != 1 {
		t.Fatalf("got %d tokens, want 1", len(tokens))
	}
	n, ok := tokens[0].(cos.Number)
	if !ok {
		t.Fatalf("token = %T, want a number", tokens[0])
	}
	if n.FloatValue() != 12 {
		t.Errorf("value = %v, want 12 — the inner '-' is dropped", n.FloatValue())
	}
}

// TestStreamTokenParserPDFBOX5906 covers an isolated '+', which is ignored and
// becomes the null object.
func TestStreamTokenParserPDFBOX5906(t *testing.T) {
	tokens := parseTokens(t, "+ Tj")
	if len(tokens) != 2 {
		t.Fatalf("got %d tokens, want 2", len(tokens))
	}
	if tokens[0] != any(cos.NullObject) {
		t.Errorf("token 0 = %v, want the null object", tokens[0])
	}
}

// TestStreamTokenParserStrayCloseBracket covers a ']' with no matching '[',
// which means a corrupt stream. Java keeps going and yields the null object.
func TestStreamTokenParserStrayCloseBracket(t *testing.T) {
	tokens := parseTokens(t, "] Tj")
	if len(tokens) != 2 {
		t.Fatalf("got %d tokens, want 2", len(tokens))
	}
	if tokens[0] != any(cos.NullObject) {
		t.Errorf("token 0 = %v, want the null object", tokens[0])
	}
}

// TestStreamTokenParserOperatorNames covers the operator reader: it stops at
// whitespace and at the delimiters that begin another token.
func TestStreamTokenParserOperatorNames(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"BT", operator.BeginText},
		{"W*", operator.ClipEvenOdd},
		{"T*", operator.NextLine},
		{"'", operator.ShowTextLine},
		{"f*", operator.FillEvenOdd},
	}
	for _, c := range cases {
		tokens := parseTokens(t, c.input)
		if len(tokens) != 1 {
			t.Fatalf("%q gave %d tokens, want 1", c.input, len(tokens))
		}
		op, ok := tokens[0].(*operator.Operator)
		if !ok {
			t.Fatalf("%q gave %T, want *operator.Operator", c.input, tokens[0])
		}
		if op.Name() != c.want {
			t.Errorf("%q = %q, want %q", c.input, op.Name(), c.want)
		}
	}
}

// TestStreamTokenParserType3GlyphOperators covers d0 and d1, which are the only
// operators with a digit in the name. The operator reader stops at a digit, so
// Java special-cases them.
func TestStreamTokenParserType3GlyphOperators(t *testing.T) {
	for _, want := range []string{"d0", "d1"} {
		tokens := parseTokens(t, want)
		if len(tokens) != 1 {
			t.Fatalf("%q gave %d tokens, want 1", want, len(tokens))
		}
		op, ok := tokens[0].(*operator.Operator)
		if !ok {
			t.Fatalf("%q gave %T, want *operator.Operator", want, tokens[0])
		}
		if op.Name() != want {
			t.Errorf("operator = %q, want %q", op.Name(), want)
		}
	}
}

func TestStreamTokenParserComments(t *testing.T) {
	tokens := parseTokens(t, "BT % this is a comment\nET")
	if len(tokens) != 2 {
		t.Fatalf("got %d tokens, want 2", len(tokens))
	}
}

func TestStreamTokenParserEmpty(t *testing.T) {
	if got := parseTokens(t, ""); len(got) != 0 {
		t.Errorf("got %d tokens for empty input, want 0", len(got))
	}
	if got := parseTokens(t, "   \n  "); len(got) != 0 {
		t.Errorf("got %d tokens for whitespace, want 0", len(got))
	}
}

// TestStreamTokenParserInlineImage covers BI ... ID ... EI: the parameters are
// collected into the BI operator and the raw bytes into the ID operator.
func TestStreamTokenParserInlineImage(t *testing.T) {
	tokens := parseTokens(t, "BI /W 2 /H 2 ID \x01\x02\x03\x04 EI Q")

	if len(tokens) < 2 {
		t.Fatalf("got %d tokens, want at least 2", len(tokens))
	}
	bi, ok := tokens[0].(*operator.Operator)
	if !ok || bi.Name() != operator.BeginInlineImage {
		t.Fatalf("token 0 = %v, want the BI operator", tokens[0])
	}
	params := bi.ImageParameters()
	if params == nil {
		t.Fatal("the BI operator carries no image parameters")
	}
	if got := params.GetInt(cos.W); got != 2 {
		t.Errorf("/W = %d, want 2", got)
	}
	if got := params.GetInt(cos.H); got != 2 {
		t.Errorf("/H = %d, want 2", got)
	}
	if len(bi.ImageData()) == 0 {
		t.Error("the BI operator carries no image data")
	}
}

// TestStreamTokenParserNestedInlineImage covers PDFBOX-6038: a BI inside a BI
// is rejected rather than recursed into.
func TestStreamTokenParserNestedInlineImage(t *testing.T) {
	p, err := NewStreamTokenParser([]byte("BI /W 2 BI /H 2 ID x EI"))
	if err != nil {
		t.Fatalf("NewStreamTokenParser: %v", err)
	}
	if _, err := p.Parse(); err == nil {
		t.Error("a nested BI operator was accepted, want an error")
	}
}
