package tools_test

// Port of org.apache.pdfbox.tools.TestPDFText2HTML, both of its cases.

import (
	"regexp"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
	"github.com/shinguakira/pdfbox-go/go/tools"
)

// createDocument is the Java helper of the same name: one page, one line of
// text in the given font, and a title.
func createDocument(t *testing.T, title string, embedded font.PDFont, text string) *pdmodel.PDDocument {
	t.Helper()
	doc := pdmodel.NewPDDocument()
	doc.DocumentInformation().SetTitle(title)
	page := pdmodel.NewPDPage()
	doc.AddPage(page)

	stream, err := pdmodel.NewPDPageContentStream(doc, page)
	if err != nil {
		t.Fatalf("NewPDPageContentStream: %v", err)
	}
	if err := stream.BeginText(); err != nil {
		t.Fatalf("BeginText: %v", err)
	}
	if err := stream.SetFont(embedded, 12); err != nil {
		t.Fatalf("SetFont: %v", err)
	}
	if err := stream.NewLineAtOffset(100, 700); err != nil {
		t.Fatalf("NewLineAtOffset: %v", err)
	}
	if err := stream.ShowText(text); err != nil {
		t.Fatalf("ShowText: %v", err)
	}
	if err := stream.EndText(); err != nil {
		t.Fatalf("EndText: %v", err)
	}
	if err := stream.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	return doc
}

// standard14 builds one of the standard 14 fonts.
func standard14(t *testing.T, name font.FontName) *font.PDType1Font {
	t.Helper()
	f, err := font.NewPDType1FontStandard14(name)
	if err != nil {
		t.Fatalf("NewPDType1FontStandard14: %v", err)
	}
	return f
}

// TestEscapeTitle is testEscapeTitle: the title and the body are both escaped,
// and a character outside ASCII becomes a numeric entity.
func TestEscapeTitle(t *testing.T) {
	stripper := tools.NewPDFText2HTML()
	// "<script>" followed by U+3042 HIRAGANA LETTER A.
	doc := createDocument(t, "<script>\u3042", standard14(t, font.Helvetica), "<foo>")
	defer doc.Close()

	text, err := stripper.GetText(doc)
	if err != nil {
		t.Fatalf("GetText: %v", err)
	}

	matcher := regexp.MustCompile(`(?s)<title>(.*?)</title>`).FindStringSubmatch(text)
	if matcher == nil {
		t.Fatalf("no <title> in the output:\n%s", text)
	}
	if want := "&lt;script&gt;&#12354;"; matcher[1] != want {
		t.Errorf("the title is %q, want %q", matcher[1], want)
	}
	if want := "&lt;foo&gt;"; !contains(text, want) {
		t.Errorf("the output does not contain %q:\n%s", want, text)
	}
}

// TestStyle is testStyle: a bold font wraps the word in <b>.
func TestStyle(t *testing.T) {
	stripper := tools.NewPDFText2HTML()
	doc := createDocument(t, "t", standard14(t, font.HelveticaBold), "<bold>")
	defer doc.Close()

	text, err := stripper.GetText(doc)
	if err != nil {
		t.Fatalf("GetText: %v", err)
	}

	matcher := regexp.MustCompile(`(?s)<p>(.*?)</p>`).FindStringSubmatch(text)
	if matcher == nil {
		t.Fatalf("body p exists: no <p> in the output:\n%s", text)
	}
	if want := "<b>&lt;bold&gt;</b>"; matcher[1] != want {
		t.Errorf("body p is %q, want %q", matcher[1], want)
	}
}

func contains(haystack, needle string) bool {
	return regexp.MustCompile(regexp.QuoteMeta(needle)).MatchString(haystack)
}

// TestMarkdownEscapesAndBolds has no Java test: PDFText2Markdown is newer than
// the module's test class and nothing covers it. A2 asks for a decision either
// way, and it is worth one -- the escaping table and the bold/italic tests are
// the whole class, and both differ from the HTML ones in ways that would
// otherwise go unchecked.
//
// The wanted values are read off the Java source: the escape switch lists the
// twelve characters that take a backslash, three that become HTML entities, and
// the two superscripts.
func TestMarkdownEscapesAndBolds(t *testing.T) {
	stripper := tools.NewPDFText2Markdown()
	doc := createDocument(t, "t", standard14(t, font.HelveticaBold), "a*b")
	defer doc.Close()

	got, err := stripper.GetText(doc)
	if err != nil {
		t.Fatalf("GetText: %v", err)
	}
	// Helvetica-Bold is bold by name, so the word is wrapped in ** and the
	// asterisk in it is escaped.
	if want := `**a\*b**`; !contains(got, want) {
		t.Errorf("the output does not contain %q:\n%q", want, got)
	}
	// The HTML stripper would have written <b>...</b> instead.
	if contains(got, "<b>") {
		t.Errorf("the Markdown output carries an HTML tag:\n%q", got)
	}
}

// TestMarkdownIsItalicTakesOblique is the one difference from the HTML class
// that a document can show: Helvetica-Oblique is italic to the Markdown
// stripper -- which lower-cases the name and looks for "oblique" as well as
// "italic" -- and is not to the HTML one, which matches "Italic" exactly.
func TestMarkdownIsItalicTakesOblique(t *testing.T) {
	markdown := tools.NewPDFText2Markdown()
	doc := createDocument(t, "t", standard14(t, font.HelveticaOblique), "x")
	defer doc.Close()
	gotMarkdown, err := markdown.GetText(doc)
	if err != nil {
		t.Fatalf("GetText: %v", err)
	}
	if want := "*x*"; !contains(gotMarkdown, want) {
		t.Errorf("the Markdown output does not contain %q:\n%q", want, gotMarkdown)
	}

	html := tools.NewPDFText2HTML()
	doc2 := createDocument(t, "t", standard14(t, font.HelveticaOblique), "x")
	defer doc2.Close()
	gotHTML, err := html.GetText(doc2)
	if err != nil {
		t.Fatalf("GetText: %v", err)
	}
	if contains(gotHTML, "<i>") {
		t.Errorf("the HTML output italicised Helvetica-Oblique; it matches "+
			"\"Italic\" exactly:\n%q", gotHTML)
	}
}
