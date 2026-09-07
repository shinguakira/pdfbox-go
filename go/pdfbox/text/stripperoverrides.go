package text

// What a subclass of PDFTextStripper overrides, and the entry point that calls
// the two document hooks.
//
// Java's PDFTextStripper is designed to be extended: writeText calls
// startDocument and endDocument, writePage calls startArticle, endArticle,
// writeString and writeParagraphEnd, and a subclass overrides whichever of
// those it needs. Go has no dynamic dispatch from a base into an embedder, so
// the base calls an interface it holds instead -- itself until an embedder
// installs itself, which is what PDFStreamEngine already does with
// StreamEngineOverrides.
//
// Slice 3 did not need this: nothing in the port extended the stripper, and
// GetTextOfPages stood in for writeText while Loader was unported. The two
// classes that do extend it, PDFText2HTML and PDFText2Markdown, are in the
// `tools` module and arrive with track/tools, which is what added this.

import (
	"io"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
)

// TextStripperOverrides is the set of hooks a subclass of PDFTextStripper
// overrides. Every one of them is `protected` in Java and called by the base.
type TextStripperOverrides interface {
	// StartDocument is called before the pages are walked.
	StartDocument(document *pdmodel.PDDocument) error

	// EndDocument is called after the pages have been walked.
	EndDocument(document *pdmodel.PDDocument) error

	// StartArticleLTR writes whatever opens an article.
	StartArticleLTR(isLTR bool) error

	// EndArticle writes whatever closes an article.
	EndArticle() error

	// WriteString writes one word, with the positions it was built from.
	WriteString(text string, textPositions []*TextPosition) error

	// WriteStringChars writes a run of characters. Java overloads writeString;
	// Go cannot, so the one-argument form is named.
	WriteStringChars(chars string) error

	// WriteParagraphEnd writes whatever closes a paragraph.
	WriteParagraphEnd() error
}

var _ TextStripperOverrides = (*PDFTextStripper)(nil)

// SetTextOverrides installs the implementation whose hooks the stripper should
// call. An embedder passes itself here, which is what stands in for Java's
// dynamic dispatch from the superclass into the subclass.
func (s *PDFTextStripper) SetTextOverrides(overrides TextStripperOverrides) {
	s.textOverrides = overrides
}

// textHooks answers what the base should call, which is itself where nothing
// was installed.
func (s *PDFTextStripper) textHooks() TextStripperOverrides {
	if s.textOverrides == nil {
		return s
	}
	return s.textOverrides
}

// StartDocument is called before the pages are walked. It does nothing; Java
// says "no default implementation, but available for subclasses".
func (s *PDFTextStripper) StartDocument(document *pdmodel.PDDocument) error { return nil }

// EndDocument is called after the pages have been walked. It does nothing.
func (s *PDFTextStripper) EndDocument(document *pdmodel.PDDocument) error { return nil }

// WriteStringChars writes a run of characters.
//
// Port of the one-argument writeString(String), which Java overloads against
// writeString(String, List<TextPosition>). Go has no overloading, so the port
// names them apart; the base's own body is the same write either way.
func (s *PDFTextStripper) WriteStringChars(chars string) error { return s.write(chars) }

// WriteText writes the text of the given document to the given writer.
//
// Port of writeText(PDDocument, Writer). GetTextOfPages stood in for it while
// Loader was unported and the stripper had no document to be given; it is the
// same walk with the two document hooks around it.
func (s *PDFTextStripper) WriteText(doc *pdmodel.PDDocument, output io.Writer) error {
	s.resetEngine()
	s.document = doc
	s.output = output
	if s.AddMoreFormatting() {
		s.paragraphEnd = s.lineSeparator
		s.pageStart = s.lineSeparator
		s.articleStart = s.lineSeparator
		s.articleEnd = s.lineSeparator
	}
	if err := s.textHooks().StartDocument(doc); err != nil {
		return err
	}
	if err := s.ProcessPages(doc.Pages()); err != nil {
		return err
	}
	return s.textHooks().EndDocument(doc)
}

// GetText returns the text of the given document.
//
// Port of getText(PDDocument), which is writeText into a StringWriter.
func (s *PDFTextStripper) GetText(doc *pdmodel.PDDocument) (string, error) {
	var out stringWriter
	if err := s.WriteText(doc, &out); err != nil {
		return "", err
	}
	return out.String(), nil
}

// stringWriter is Java's StringWriter.
type stringWriter struct{ builder []byte }

func (w *stringWriter) Write(p []byte) (int, error) {
	w.builder = append(w.builder, p...)
	return len(p), nil
}

func (w *stringWriter) String() string { return string(w.builder) }

// LineSeparatorDefault is Java's static PDFTextStripper.LINE_SEPARATOR, which
// the subclasses in the `tools` module build their separators out of.
//
// Java reads System.lineSeparator; the port uses "\n" outright for the reason
// given where the constant is declared.
const LineSeparatorDefault = lineSeparatorDefault

// Document returns the document writeText was given.
//
// Port of the protected field `document`, which PDFText2HTML.getTitle reads.
func (s *PDFTextStripper) Document() *pdmodel.PDDocument { return s.document }
