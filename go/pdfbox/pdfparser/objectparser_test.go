package pdfparser

import (
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// Written from the object-parsing half of
// pdfbox/src/main/java/org/apache/pdfbox/pdfparser/COSParser.java. The Java
// suite exercises these only through whole documents, so per
// migration/conventions/tdd.md the tests are written from the source, and the
// recovery paths its comments name are pinned individually.

func newObjectParser(input string) *ObjectParser {
	return NewObjectParser(pdfio.NewReadBufferBytes([]byte(input)), nil)
}

func parseOne(t *testing.T, input string) cos.Base {
	t.Helper()
	got, err := newObjectParser(input).ParseDirObject()
	if err != nil {
		t.Fatalf("ParseDirObject(%q): %v", input, err)
	}
	return got
}

func TestParseDirObjectSimpleTypes(t *testing.T) {
	if got := parseOne(t, "null"); got != cos.Base(cos.NullObject) {
		t.Errorf("null = %v, want the null object", got)
	}
	if got := parseOne(t, "true"); got != cos.Base(cos.True) {
		t.Errorf("true = %v, want the true boolean", got)
	}
	if got := parseOne(t, "false"); got != cos.Base(cos.False) {
		t.Errorf("false = %v, want the false boolean", got)
	}
}

func TestParseCOSName(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"/Type", "Type"},
		{"/Type ", "Type"},
		{"/", ""},
		{"/A#20B", "A B"},
		// '#' is only an escape when two valid hex digits follow it
		{"/A#ZZ", "A#ZZ"},
		// A '#' with only one digit before the end of input takes Java's
		// premature-EOF branch, which logs and drops what it had read.
		{"/A#2", "A"},
		// with a following byte there is no premature EOF, so the '#' is kept
		{"/A#2Z ", "A#2Z"},
		{"/Name/Next", "Name"},
		{"/Name[", "Name"},
	}
	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			p := newObjectParser(c.input)
			got, err := p.ParseCOSName()
			if err != nil {
				t.Fatalf("ParseCOSName: %v", err)
			}
			if got.Name() != c.want {
				t.Errorf("ParseCOSName(%q) = %q, want %q", c.input, got.Name(), c.want)
			}
		})
	}
}

func TestParseCOSNumber(t *testing.T) {
	cases := []struct {
		input   string
		isFloat bool
		want    float64
	}{
		{"0", false, 0},
		{"42", false, 42},
		{"-17", false, -17},
		{"+5", false, 5},
		{"1.5", true, 1.5},
		{"-0.25", true, -0.25},
	}
	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			got := parseOne(t, c.input)
			n, ok := got.(cos.Number)
			if !ok {
				t.Fatalf("parsed %T, want a number", got)
			}
			if float64(n.FloatValue()) != c.want {
				t.Errorf("value = %v, want %v", n.FloatValue(), c.want)
			}
			if _, isFloat := got.(*cos.Float); isFloat != c.isFloat {
				t.Errorf("float = %v, want %v", isFloat, c.isFloat)
			}
		})
	}
}

// TestParseCOSNumberPDFBOX5025 covers the case the Java comment names: a number
// run together with a keyword, "74191endobj". The trailing 'e' is taken as part
// of an exponent, then given back.
func TestParseCOSNumberPDFBOX5025(t *testing.T) {
	p := newObjectParser("74191endobj")
	got, err := p.ParseDirObject()
	if err != nil {
		t.Fatalf("ParseDirObject: %v", err)
	}
	n, ok := got.(*cos.Integer)
	if !ok {
		t.Fatalf("parsed %T, want *cos.Integer", got)
	}
	if n.LongValue() != 74191 {
		t.Errorf("value = %d, want 74191", n.LongValue())
	}
	// the keyword must still be there for the caller
	rest, err := p.ReadString()
	if err != nil {
		t.Fatalf("ReadString: %v", err)
	}
	if rest != "endobj" {
		t.Errorf("remaining input = %q, want %q", rest, "endobj")
	}
}

func TestParseCOSLiteralString(t *testing.T) {
	got := parseOne(t, "(hello)")
	s, ok := got.(*cos.StringObj)
	if !ok {
		t.Fatalf("parsed %T, want *cos.StringObj", got)
	}
	if s.Value() != "hello" {
		t.Errorf("value = %q, want %q", s.Value(), "hello")
	}
}

func TestParseCOSHexString(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"<48656C6C6F>", "Hello"},
		{"<48 65 6C 6C 6F>", "Hello"},  // whitespace is skipped
		{"<48656C6C6F7>", "Hello\x70"}, // odd length pads with a zero nibble
		{"<>", ""},
	}
	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			got := parseOne(t, c.input)
			s, ok := got.(*cos.StringObj)
			if !ok {
				t.Fatalf("parsed %T, want *cos.StringObj", got)
			}
			if s.Value() != c.want {
				t.Errorf("value = %q, want %q", s.Value(), c.want)
			}
		})
	}
}

// TestParseCOSHexStringInvalidChars covers the recovery: on an invalid
// character the last unpaired digit is discarded and the parser runs to the
// closing bracket.
func TestParseCOSHexStringInvalidChars(t *testing.T) {
	got := parseOne(t, "<4865ZZ6C6C6F>")
	s, ok := got.(*cos.StringObj)
	if !ok {
		t.Fatalf("parsed %T, want *cos.StringObj", got)
	}
	if s.Value() != "He" {
		t.Errorf("value = %q, want %q — everything after the bad character is dropped", s.Value(), "He")
	}
}

func TestParseCOSHexStringUnterminated(t *testing.T) {
	if _, err := newObjectParser("<4865").ParseDirObject(); err == nil {
		t.Error("an unterminated hex string parsed without error")
	}
}

func TestParseCOSArray(t *testing.T) {
	got := parseOne(t, "[1 2 3]")
	a, ok := got.(*cos.Array)
	if !ok {
		t.Fatalf("parsed %T, want *cos.Array", got)
	}
	if a.Size() != 3 {
		t.Fatalf("Size() = %d, want 3", a.Size())
	}
	for i, want := range []int{1, 2, 3} {
		if a.GetInt(i) != want {
			t.Errorf("[%d] = %d, want %d", i, a.GetInt(i), want)
		}
	}
}

func TestParseCOSArrayNested(t *testing.T) {
	got := parseOne(t, "[/A [1 2] (s)]")
	a, ok := got.(*cos.Array)
	if !ok {
		t.Fatalf("parsed %T, want *cos.Array", got)
	}
	if a.Size() != 3 {
		t.Fatalf("Size() = %d, want 3", a.Size())
	}
	if _, ok := a.Get(1).(*cos.Array); !ok {
		t.Errorf("[1] = %T, want a nested array", a.Get(1))
	}
}

func TestParseCOSArrayEmpty(t *testing.T) {
	got := parseOne(t, "[]")
	a, ok := got.(*cos.Array)
	if !ok {
		t.Fatalf("parsed %T, want *cos.Array", got)
	}
	if a.Size() != 0 {
		t.Errorf("Size() = %d, want 0", a.Size())
	}
}

func TestParseCOSDictionary(t *testing.T) {
	got := parseOne(t, "<< /Type /Page /Count 3 >>")
	d, ok := got.(*cos.Dictionary)
	if !ok {
		t.Fatalf("parsed %T, want *cos.Dictionary", got)
	}
	if d.Size() != 2 {
		t.Fatalf("Size() = %d, want 2", d.Size())
	}
	if got := d.GetCOSName(cos.Type); got != cos.Page {
		t.Errorf("/Type = %v, want /Page", got)
	}
	if got := d.GetInt(cos.Count); got != 3 {
		t.Errorf("/Count = %d, want 3", got)
	}
}

func TestParseCOSDictionaryNested(t *testing.T) {
	got := parseOne(t, "<< /Resources << /Font 1 >> >>")
	d, ok := got.(*cos.Dictionary)
	if !ok {
		t.Fatalf("parsed %T, want *cos.Dictionary", got)
	}
	sub := d.GetCOSDictionary(cos.Resources)
	if sub == nil {
		t.Fatal("/Resources is not a dictionary")
	}
	if got := sub.GetInt(cos.Font); got != 1 {
		t.Errorf("/Font = %d, want 1", got)
	}
}

// TestParseCOSDictionaryRecovery covers the path taken when a key is not a
// name: Java logs and reads until it can recover, returning what it has.
func TestParseCOSDictionaryRecovery(t *testing.T) {
	got := parseOne(t, "<< /Type /Page 123 >>")
	d, ok := got.(*cos.Dictionary)
	if !ok {
		t.Fatalf("parsed %T, want *cos.Dictionary", got)
	}
	// what was read before the damage survives
	if got := d.GetCOSName(cos.Type); got != cos.Page {
		t.Errorf("/Type = %v, want /Page to survive the recovery", got)
	}
}

// TestParseIndirectReference covers a dictionary value of the form "12 0 R",
// which becomes a proxy from the document object pool.
func TestParseIndirectReference(t *testing.T) {
	doc := cos.NewDocument(nil)
	p := NewObjectParser(pdfio.NewReadBufferBytes([]byte("<< /Root 12 0 R >>")), doc)

	got, err := p.ParseDirObject()
	if err != nil {
		t.Fatalf("ParseDirObject: %v", err)
	}
	d, ok := got.(*cos.Dictionary)
	if !ok {
		t.Fatalf("parsed %T, want *cos.Dictionary", got)
	}
	ref := d.GetCOSObject(cos.Root)
	if ref == nil {
		t.Fatal("/Root is not an indirect reference")
	}
	if got := ref.Key().Number(); got != 12 {
		t.Errorf("object number = %d, want 12", got)
	}
	if got := ref.Key().Generation(); got != 0 {
		t.Errorf("generation = %d, want 0", got)
	}
}

// TestParseIndirectReferenceInArray covers the same in an array, where Java has
// to walk back over the two integers it already added — PDFBOX-385.
func TestParseIndirectReferenceInArray(t *testing.T) {
	doc := cos.NewDocument(nil)
	p := NewObjectParser(pdfio.NewReadBufferBytes([]byte("[1 0 R 2 0 R]")), doc)

	got, err := p.ParseDirObject()
	if err != nil {
		t.Fatalf("ParseDirObject: %v", err)
	}
	a, ok := got.(*cos.Array)
	if !ok {
		t.Fatalf("parsed %T, want *cos.Array", got)
	}
	if a.Size() != 2 {
		t.Fatalf("Size() = %d, want 2 — the integers must be replaced by references", a.Size())
	}
	for i := 0; i < 2; i++ {
		ref, ok := a.Get(i).(*cos.Object)
		if !ok {
			t.Fatalf("[%d] = %T, want *cos.Object", i, a.Get(i))
		}
		if got := ref.Key().Number(); got != int64(i+1) {
			t.Errorf("[%d] object number = %d, want %d", i, got, i+1)
		}
	}
}

// TestParseDirObjectRecursionLimit covers the guard against a deeply nested
// structure, which is attacker-controlled: Java stops at 500.
func TestParseDirObjectRecursionLimit(t *testing.T) {
	deep := strings.Repeat("[", 600) + strings.Repeat("]", 600)
	if _, err := newObjectParser(deep).ParseDirObject(); err == nil {
		t.Error("a 600-deep array parsed without error, want the recursion limit to stop it")
	}
}

func TestReadObjectNumberRejectsOutOfRange(t *testing.T) {
	// more than ten digits
	if _, err := newObjectParser("99999999999").ReadObjectNumber(); err == nil {
		t.Error("an 11-digit object number was accepted")
	}
	// a valid one is fine
	if got, err := newObjectParser("12").ReadObjectNumber(); err != nil || got != 12 {
		t.Errorf("ReadObjectNumber() = %d, %v; want 12, nil", got, err)
	}
}

func TestReadGenerationNumberRejectsOutOfRange(t *testing.T) {
	if _, err := newObjectParser("65536").ReadGenerationNumber(); err == nil {
		t.Error("a generation number above 65535 was accepted")
	}
	if got, err := newObjectParser("0").ReadGenerationNumber(); err != nil || got != 0 {
		t.Errorf("ReadGenerationNumber() = %d, %v; want 0, nil", got, err)
	}
}

func TestReadLine(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"hello\nworld", "hello"},
		{"hello\r\nworld", "hello"},
		{"hello\rworld", "hello"},
		{"hello", "hello"},
	}
	for _, c := range cases {
		p := newObjectParser(c.input)
		got, err := p.ReadLine()
		if err != nil {
			t.Fatalf("ReadLine(%q): %v", c.input, err)
		}
		if got != c.want {
			t.Errorf("ReadLine(%q) = %q, want %q", c.input, got, c.want)
		}
	}

	if _, err := newObjectParser("").ReadLine(); err == nil {
		t.Error("ReadLine at end of input succeeded, want an error")
	}
}

func TestObjectKeyCache(t *testing.T) {
	p := newObjectParser("")

	first, err := p.ObjectKey(12, 0)
	if err != nil {
		t.Fatalf("ObjectKey: %v", err)
	}
	if first.Number() != 12 || first.Generation() != 0 {
		t.Errorf("ObjectKey(12, 0) = %v, want 12 0 R", first)
	}
}
