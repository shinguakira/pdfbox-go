package text_test

// The hooks a subclass of PDFTextStripper overrides, and that the base calls
// them rather than its own.
//
// Java gets this from dynamic dispatch and needs no test for it. Go does not:
// a struct embedding *PDFTextStripper cannot override a method the base calls
// on itself, so the base holds an interface and calls that. This checks it
// does -- without it every override would compile, do nothing, and leave the
// suite green.

import (
	"strings"
	"testing"

	pdfbox "github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/text"
)

// shoutingStripper overrides five of the seven hooks, in the shape
// PDFText2HTML overrides them.
type shoutingStripper struct {
	*text.PDFTextStripper

	started, ended int
	words          []string
	articles       int
	paragraphEnds  int
}

func newShoutingStripper() *shoutingStripper {
	s := &shoutingStripper{PDFTextStripper: text.NewPDFTextStripper()}
	s.SetTextOverrides(s)
	return s
}

func (s *shoutingStripper) StartDocument(document *pdmodel.PDDocument) error {
	s.started++
	return s.WriteStringChars("<<")
}

func (s *shoutingStripper) EndDocument(document *pdmodel.PDDocument) error {
	s.ended++
	return s.WriteStringChars(">>")
}

func (s *shoutingStripper) StartArticleLTR(isLTR bool) error {
	s.articles++
	return s.PDFTextStripper.StartArticleLTR(isLTR)
}

func (s *shoutingStripper) WriteString(word string, positions []*text.TextPosition) error {
	s.words = append(s.words, word)
	return s.WriteStringChars(strings.ToUpper(word))
}

func (s *shoutingStripper) WriteParagraphEnd() error {
	s.paragraphEnds++
	return s.PDFTextStripper.WriteParagraphEnd()
}

// TestStripperCallsItsOverrides runs a document through a stripper that
// overrides five hooks and checks each was reached.
func TestStripperCallsItsOverrides(t *testing.T) {
	const fixture = "../../../tools/src/test/resources/org/apache/pdfbox/hello3.pdf"
	document, err := pdfbox.LoadPDF(fixture)
	if err != nil {
		t.Fatalf("loading %s: %v", fixture, err)
	}
	defer document.Close()

	stripper := newShoutingStripper()
	got, err := stripper.GetText(document)
	if err != nil {
		t.Fatalf("GetText: %v", err)
	}

	if stripper.started != 1 || stripper.ended != 1 {
		t.Errorf("StartDocument ran %d times and EndDocument %d; want 1 each",
			stripper.started, stripper.ended)
	}
	if !strings.HasPrefix(got, "<<") || !strings.HasSuffix(got, ">>") {
		t.Errorf("the text is %q; the two document hooks should have wrapped it", got)
	}
	if stripper.articles == 0 {
		t.Error("StartArticleLTR was never called")
	}
	if stripper.paragraphEnds == 0 {
		t.Error("WriteParagraphEnd was never called")
	}
	if len(stripper.words) == 0 {
		t.Fatal("WriteString was never called")
	}
	// The override upper-cased every word, so the base cannot have written the
	// originals.
	for _, word := range stripper.words {
		if word != strings.ToUpper(word) && strings.Contains(got, word) {
			t.Errorf("the text contains %q as written; the override upper-cases it", word)
		}
	}
	if !strings.Contains(got, strings.ToUpper(stripper.words[0])) {
		t.Errorf("the text is %q, want it to contain %q",
			got, strings.ToUpper(stripper.words[0]))
	}
}

// TestStripperWithNoOverridesIsUnchanged is the other half: a stripper nobody
// installed anything on calls its own hooks, which is every existing caller.
func TestStripperWithNoOverridesIsUnchanged(t *testing.T) {
	const fixture = "../../../tools/src/test/resources/org/apache/pdfbox/hello3.pdf"
	document, err := pdfbox.LoadPDF(fixture)
	if err != nil {
		t.Fatalf("loading %s: %v", fixture, err)
	}
	defer document.Close()

	fromWriteText, err := text.NewPDFTextStripper().GetText(document)
	if err != nil {
		t.Fatalf("GetText: %v", err)
	}
	fromPages, err := text.NewPDFTextStripper().GetTextOfPages(document.Pages())
	if err != nil {
		t.Fatalf("GetTextOfPages: %v", err)
	}
	if fromWriteText != fromPages {
		t.Errorf("writeText gave %q and the page walk %q; with no hooks overridden "+
			"they are the same walk", fromWriteText, fromPages)
	}
	if !strings.Contains(fromWriteText, "Hello") {
		t.Errorf("the text is %q, want it to contain the fixture's text", fromWriteText)
	}
}
