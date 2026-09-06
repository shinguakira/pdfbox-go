package pdfparser

import (
	"strings"
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

// --- Port of org.apache.pdfbox.pdfparser.PDFStreamParserTest -------------
//
// The Java class is PDFStreamParser, which is this file's StreamTokenParser.
// Both of its cases are about finding where an inline image ends, which is the
// hardest thing this parser does: EI is not a delimiter, it is two bytes that
// can appear inside the image data.

// inlineImageTokens is the Java helper parseTokenString.
func inlineImageTokens(t *testing.T, s string) []any {
	t.Helper()
	parser, err := NewStreamTokenParser([]byte(s))
	if err != nil {
		t.Fatalf("NewStreamTokenParser(%q): %v", s, err)
	}
	tokens, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse(%q): %v", s, err)
	}
	return tokens
}

// wantInlineImage is testInlineImage2ops and testInlineImage1op together: the
// first token is always the image, and opName is empty where Java expects the
// one-operator form.
func wantInlineImage(t *testing.T, s, imageData, opName string) {
	t.Helper()
	tokens := inlineImageTokens(t, s)

	want := 1
	if opName != "" {
		want = 2
	}
	if len(tokens) != want {
		t.Fatalf("%q gave %d tokens, want %d", s, len(tokens), want)
	}

	image, ok := tokens[0].(*operator.Operator)
	if !ok {
		t.Fatalf("%q: first token is %T, want an operator", s, tokens[0])
	}
	if image.Name() != operator.BeginInlineImageData {
		t.Errorf("%q: first operator is %q, want %q", s, image.Name(),
			operator.BeginInlineImageData)
	}
	if got := string(image.ImageData()); got != imageData {
		t.Errorf("%q: image data = %q, want %q", s, got, imageData)
	}

	if opName == "" {
		return
	}
	next, ok := tokens[1].(*operator.Operator)
	if !ok {
		t.Fatalf("%q: second token is %T, want an operator", s, tokens[1])
	}
	if next.Name() != opName {
		t.Errorf("%q: second operator is %q, want %q", s, next.Name(), opName)
	}
}

// TestInlineImages is testInlineImages.
//
// Its comment says the point of the long runs of spaces is
// hasNoFollowingBinData: the parser looks ahead maxBinCharTestLength bytes
// after an EI to decide whether what follows is an operator or more image data.
func TestInlineImages(t *testing.T) {
	for _, row := range []struct {
		input     string
		imageData string
		op        string
	}{
		{"ID\n12345EI Q", "12345", "Q"},
		{"ID\n12345EI EMC", "12345", "EMC"},
		{"ID\n12345EI Q ", "12345", "Q"},
		{"ID\n12345EI EMC ", "12345", "EMC"},
		{"ID\n12345EI  Q", "12345", "Q"},
		{"ID\n12345EI  EMC", "12345", "EMC"},
		{"ID\n12345EI  Q ", "12345", "Q"},
		{"ID\n12345EI  EMC ", "12345", "EMC"},

		{"ID\n12345EI \000Q", "12345", "Q"},

		{"ID\n12345EI Q                             ", "12345", "Q"},
		{"ID\n12345EI EMC                           ", "12345", "EMC"},

		{"ID\n12345EI", "12345", ""},
		{"ID\n12345EI                               ", "12345", ""},

		{"ID\n12345EI                               Q ", "12345", "Q"},
		{"ID\n12345EI                               EMC ", "12345", "EMC"},
		{"ID\n12345EI                               Q", "12345", "Q"},
		{"ID\n12345EI                               EMC", "12345", "EMC"},

		// an EI inside the data is not the end of it
		{"ID\n12EI5EI", "12EI5", ""},
		{"ID\n12EI5EI ", "12EI5", ""},
		{"ID\n12EI5EIQEI", "12EI5EIQ", ""},
		{"ID\n12EI5EIQEI Q", "12EI5EIQ", "Q"},
		{"ID\n12EI5EI Q", "12EI5", "Q"},
		{"ID\n12EI5EI Q ", "12EI5", "Q"},
		{"ID\n12EI5EI EMC", "12EI5", "EMC"},
		{"ID\n12EI5EI EMC ", "12EI5", "EMC"},
		{"ID\n12EI5EI                                Q", "12EI5", "Q"},
		{"ID\n12EI5EI                                Q ", "12EI5", "Q"},
		{"ID\n12EI5EI                                EMC", "12EI5", "EMC"},
		{"ID\n12EI5EI                                EMC ", "12EI5", "EMC"},

		// maxBinCharTestLength is 10; these walk its boundary
		//                    1234567890
		{"ID\n12EI5EI       EMC ", "12EI5", "EMC"},
		{"ID\n12EI5EI        EMC ", "12EI5", "EMC"},
		{"ID\n12EI5EI         EMC ", "12EI5", "EMC"},
		{"ID\n12EI5EI          EMC ", "12EI5", "EMC"},
		{"ID\n12EI5EI       Q   ", "12EI5", "Q"},
		{"ID\n12EI5EI        Q   ", "12EI5", "Q"},
		{"ID\n12EI5EI         Q   ", "12EI5", "Q"},
		{"ID\n12EI5EI          Q   ", "12EI5", "Q"},
	} {
		t.Run(row.input, func(t *testing.T) {
			wantInlineImage(t, row.input, row.imageData, row.op)
		})
	}
}

// TestNestedBI is testNestedBI, PDFBOX-6038: a BI inside an inline image is
// refused, and the failure names where both of them are.
//
// Java asserts the whole message:
//
//	Nested 'BI' operator not allowed at offset 11, first: 2
//
// The port's reads `pdfparser: nested "BI" operator not allowed at offset 11,
// first: 2` — the same two offsets, in the lower-case package-prefixed form
// every error in this package uses. The offsets are the substance and are
// asserted; the prose is a house convention applied throughout, and asserting
// it here would only pin the convention.
func TestNestedBI(t *testing.T) {
	parser, err := NewStreamTokenParser([]byte("BI/IB/IB BI/ BI"))
	if err != nil {
		t.Fatalf("NewStreamTokenParser: %v", err)
	}
	_, err = parser.Parse()
	if err == nil {
		t.Fatal("Parse reported nothing, want the nested-BI error")
	}
	got := err.Error()
	if !strings.Contains(got, "at offset 11") || !strings.Contains(got, "first: 2") {
		t.Errorf("Parse = %q, want it to name offset 11 and first 2", got)
	}
	if !strings.Contains(got, "BI") {
		t.Errorf("Parse = %q, want it to name the BI operator", got)
	}
}
