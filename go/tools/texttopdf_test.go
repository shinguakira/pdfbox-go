package tools_test

// Port of org.apache.pdfbox.tools.TestTextToPdf, all four cases.
//
// Every wanted value is the Java's, including the two full pages of laid-out
// Lorem ipsum in testOverflow -- which is the case that says the line breaking
// and the page breaking are right, and nothing else in the suite does.

import (
	"bytes"
	"strings"
	"testing"

	pdfbox "github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/text"
	"github.com/shinguakira/pdfbox-go/go/tools"
)

// createPDFFromText is the Java helper createPDFFromText(Reader), which builds
// its own document.
func createPDFFromText(t *testing.T, creator *tools.TextToPDF, source string) *pdmodel.PDDocument {
	t.Helper()
	doc := pdmodel.NewPDDocument()
	if err := creator.CreatePDFFromText(doc, strings.NewReader(source)); err != nil {
		t.Fatalf("CreatePDFFromText: %v", err)
	}
	return doc
}

// saveAndReload writes the document out and reads it back, which every case
// but the first does.
func saveAndReload(t *testing.T, doc *pdmodel.PDDocument) *pdmodel.PDDocument {
	t.Helper()
	var baos bytes.Buffer
	if err := doc.Save(&baos); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := doc.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	reloaded, err := pdfbox.LoadPDFBytes(baos.Bytes())
	if err != nil {
		t.Fatalf("LoadPDFBytes: %v", err)
	}
	return reloaded
}

// pageText is the stripper the last three cases read one page with.
func pageText(t *testing.T, doc *pdmodel.PDDocument, startPage, endPage int,
	plainSeparators bool) string {
	t.Helper()
	stripper := text.NewPDFTextStripper()
	if plainSeparators {
		stripper.SetLineSeparator("\n")
		stripper.SetParagraphStart("\n")
	}
	stripper.SetStartPage(startPage)
	stripper.SetEndPage(endPage)
	got, err := stripper.GetText(doc)
	if err != nil {
		t.Fatalf("GetText: %v", err)
	}
	return strings.TrimSpace(got)
}

// TestCreateEmptyPdf is testCreateEmptyPdf: a PDF made from an empty string is
// still readable by Adobe Reader, which needs it to have a page.
func TestCreateEmptyPdf(t *testing.T) {
	pdfDoc := createPDFFromText(t, tools.NewTextToPDF(), "")
	defer pdfDoc.Close()

	pageCount := pdfDoc.NumberOfPages()
	if pageCount <= 0 {
		t.Error("All Pages was unexpectedly zero.")
	}
	if pageCount != 1 {
		t.Errorf("Wrong number of pages: %d", pageCount)
	}
}

// TestFormFeed is testFormFeed: a form feed starts a new page.
func TestFormFeed(t *testing.T) {
	doc := createPDFFromText(t, tools.NewTextToPDF(), "First page\fSecond page\f\nThird page")
	reloaded := saveAndReload(t, doc)
	defer reloaded.Close()

	if got := reloaded.NumberOfPages(); got != 3 {
		t.Fatalf("the document has %d pages, want 3", got)
	}
	for page, want := range map[int]string{1: "First page", 2: "Second page", 3: "Third page"} {
		if got := pageText(t, reloaded, page, page, false); got != want {
			t.Errorf("page %d is %q, want %q", page, got, want)
		}
	}
}

// TestOverflow is testOverflow: x overflow so that a new line is used, and
// overflow on the y axis so a new page must be created.
func TestOverflow(t *testing.T) {
	creator := tools.NewTextToPDF()
	creator.SetMediaBox(common.A6)

	const source = "Lorem ipsum dolor sit amet, consetetur sadipscing " +
		"elitr, sed diam nonumy eirmod tempor invidunt ut labore et dolore magna aliquyam " +
		"erat, sed diam voluptua. At vero eos et accusam et justo duo dolores et ea rebum. " +
		"Stet clita kasd gubergren, no sea takimata sanctus est Lorem ipsum dolor sit amet. " +
		"Lorem ipsum dolor sit amet, consetetur sadipscing elitr, sed diam nonumy eirmod " +
		"tempor invidunt ut labore et dolore magna aliquyam erat, sed diam voluptua. " +
		"At vero eos et accusam et justo duo dolores et ea rebum. Stet clita kasd " +
		"gubergren, no sea takimata sanctus est Lorem ipsum dolor sit amet. Lorem " +
		"ipsum dolor sit amet, consetetur sadipscing elitr, sed diam nonumy eirmod " +
		"tempor invidunt ut labore et dolore magna aliquyam erat, sed diam voluptua. " +
		"At vero eos et accusam et justo duo dolores et ea rebum. Stet clita kasd " +
		"gubergren, no sea takimata sanctus est Lorem ipsum dolor sit amet.\n" +
		"\n" +
		"Duis autem vel eum iriure dolor in hendrerit in vulputate velit esse molestie " +
		"consequat, vel illum dolore eu feugiat nulla facilisis at vero eros et accumsan " +
		"et iusto odio dignissim qui blandit praesent luptatum zzril delenit augue " +
		"duis dolore te feugait nulla facilisi. Lorem ipsum dolor sit amet, " +
		"consectetuer adipiscing elit, sed diam nonummy nibh euismod tincidunt ut " +
		"laoreet dolore magna aliquam erat volutpat.\n" +
		"\n" +
		"Ut wisi enim ad minim veniam, quis nostrud exerci tation ullamcorper " +
		"suscipit lobortis nisl ut aliquip ex ea commodo consequat. Duis autem vel eum " +
		"iriure dolor in hendrerit in vulputate velit esse molestie consequat, vel illum " +
		"dolore eu feugiat nulla facilisis at vero eros et accumsan et iusto odio " +
		"dignissim qui blandit praesent luptatum zzril delenit augue duis dolore te " +
		"feugait nulla facilisi.\n" +
		"\n" +
		"Nam liber tempor cum soluta nobis eleifend option congue nihil imperdiet " +
		"doming id quod mazim placerat facer."

	const expectedPage1Text = "Lorem ipsum dolor sit amet, consetetur\n" +
		"sadipscing elitr, sed diam nonumy eirmod\n" +
		"tempor invidunt ut labore et dolore magna\n" +
		"aliquyam erat, sed diam voluptua. At vero eos et\n" +
		"accusam et justo duo dolores et ea rebum. Stet\n" +
		"clita kasd gubergren, no sea takimata sanctus\n" +
		"est Lorem ipsum dolor sit amet. Lorem ipsum\n" +
		"dolor sit amet, consetetur sadipscing elitr, sed\n" +
		"diam nonumy eirmod tempor invidunt ut labore et\n" +
		"dolore magna aliquyam erat, sed diam voluptua.\n" +
		"At vero eos et accusam et justo duo dolores et\n" +
		"ea rebum. Stet clita kasd gubergren, no sea\n" +
		"takimata sanctus est Lorem ipsum dolor sit amet.\n" +
		"Lorem ipsum dolor sit amet, consetetur\n" +
		"sadipscing elitr, sed diam nonumy eirmod\n" +
		"tempor invidunt ut labore et dolore magna\n" +
		"aliquyam erat, sed diam voluptua. At vero eos et\n" +
		"accusam et justo duo dolores et ea rebum. Stet\n" +
		"clita kasd gubergren, no sea takimata sanctus\n" +
		"est Lorem ipsum dolor sit amet.\n" +
		"\n" +
		"Duis autem vel eum iriure dolor in hendrerit in\n" +
		"vulputate velit esse molestie consequat, vel illum\n" +
		"dolore eu feugiat nulla facilisis at vero eros et\n" +
		"accumsan et iusto odio dignissim qui blandit\n" +
		"praesent luptatum zzril delenit augue duis dolore\n" +
		"te feugait nulla facilisi. Lorem ipsum dolor sit\n" +
		"amet, consectetuer adipiscing elit, sed diam"

	const expectedPage2Text = "nonummy nibh euismod tincidunt ut laoreet\n" +
		"dolore magna aliquam erat volutpat.\n" +
		"\n" +
		"Ut wisi enim ad minim veniam, quis nostrud\n" +
		"exerci tation ullamcorper suscipit lobortis nisl ut\n" +
		"aliquip ex ea commodo consequat. Duis autem\n" +
		"vel eum iriure dolor in hendrerit in vulputate velit\n" +
		"esse molestie consequat, vel illum dolore eu\n" +
		"feugiat nulla facilisis at vero eros et accumsan et\n" +
		"iusto odio dignissim qui blandit praesent\n" +
		"luptatum zzril delenit augue duis dolore te feugait\n" +
		"nulla facilisi.\n" +
		"\n" +
		"Nam liber tempor cum soluta nobis eleifend\n" +
		"option congue nihil imperdiet doming id quod\n" +
		"mazim placerat facer."

	doc := createPDFFromText(t, creator, source)
	reloaded := saveAndReload(t, doc)
	defer reloaded.Close()

	if got := reloaded.NumberOfPages(); got != 2 {
		t.Fatalf("the document has %d pages, want 2", got)
	}
	if got := pageText(t, reloaded, 1, 1, true); got != expectedPage1Text {
		t.Errorf("page 1 is\n%q\nwant\n%q", got, expectedPage1Text)
	}
	if got := pageText(t, reloaded, 2, 2, true); got != expectedPage2Text {
		t.Errorf("page 2 is\n%q\nwant\n%q", got, expectedPage2Text)
	}
}

// TestLeadingTrailingSpaces is testLeadingTrailingSpaces: leading and trailing
// spaces and newlines are preserved.
func TestLeadingTrailingSpaces(t *testing.T) {
	const source = "Lorem ipsum dolor sit amet,\n" +
		"    consectetur adipiscing \n" +
		"\n" +
		"elit. sed do eiusmod"

	doc := createPDFFromText(t, tools.NewTextToPDF(), source)
	reloaded := saveAndReload(t, doc)
	defer reloaded.Close()

	if got := reloaded.NumberOfPages(); got != 1 {
		t.Fatalf("the document has %d pages, want 1", got)
	}
	if got := pageText(t, reloaded, 1, reloaded.NumberOfPages(), true); got != source {
		t.Errorf("the text is\n%q\nwant\n%q", got, source)
	}
}
