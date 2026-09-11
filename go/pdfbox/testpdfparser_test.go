package pdfbox

// Port of org.apache.pdfbox.pdfparser.TestPDFParser, the seventeen cases that
// read from `target/pdfs`.
//
// That directory is filled by the Maven build downloading the reduced
// reproducer attached to each JIRA issue, and it was empty in this repository
// until 2026-09-11. The eighteenth case, testPDFParserMissingCatalog, reads a
// checked-in fixture and has been in `loader_test.go` all along.
//
// This is the recovery suite: every file here is broken in a different way --
// a trailer that has to be rebuilt, an object stream with newlines in the
// wrong places, a file that stops in the middle -- and what is asserted is
// that the parser gets something sensible out of it anyway. The expected
// values are the Java's, copied, not recomputed.

import (
	"testing"
	"time"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// targetPDFDir is Java's TARGETPDFDIR.
const targetPDFDir = "../../pdfbox/target/pdfs/"

// openTarget loads one of them, failing the test where the fetch has not been
// run rather than where the parse goes wrong, because the two look the same
// from a bare error.
func openTarget(t *testing.T, name string) *pdmodel.PDDocument {
	t.Helper()
	document, err := LoadPDF(targetPDFDir + name)
	if err != nil {
		t.Fatalf("LoadPDF(%s): %v\n"+
			"if the file is not there, run migration/scripts/fetch-testdata.ps1",
			name, err)
	}
	t.Cleanup(func() { document.Close() })
	return document
}

// loadsWithoutError is Java's assertDoesNotThrow round a load and a close.
//
// Java asserts nothing else in six of these cases: the file used to bring down
// the parser and now it does not, and there is no other answer to check. The
// port asserts the page count is not negative as well, which is not in the
// Java -- an error return is easier to swallow by accident than an exception
// is, and "it loaded" should mean the document is usable.
func loadsWithoutError(t *testing.T, name string) {
	t.Helper()
	document := openTarget(t, name)
	if got := document.NumberOfPages(); got < 0 {
		t.Errorf("NumberOfPages() = %d after loading %s", got, name)
	}
}

// wantDate parses one of the Java's expected date strings through the same
// DateConverter the Java asserts with.
func wantDate(t *testing.T, text string) time.Time {
	t.Helper()
	when, ok := util.ToCalendar(text)
	if !ok {
		t.Fatalf("the test's own expected date %q does not parse", text)
	}
	return when
}

// checkDate compares one of the two dates of an /Info dictionary.
func checkDate(t *testing.T, what string, got time.Time, ok bool, want time.Time) {
	t.Helper()
	if !ok {
		t.Errorf("%s is absent, want %v", what, want)
		return
	}
	if !got.Equal(want) {
		t.Errorf("%s = %v, want %v", what, got, want)
	}
}

// TestPDFBox3208 is testPDFBox3208: the /Info of a file whose trailer has to be
// rebuilt. An incorrect rebuild picks up the outline dictionary and calls it
// the /Info, so every field here is the assertion.
func TestPDFBox3208(t *testing.T) {
	document := openTarget(t, "PDFBOX-3208-L33MUTT2SVCWGCS6UIYL5TH3PNPXHIS6.pdf")
	information := document.DocumentInformation()

	for _, c := range []struct{ what, got, want string }{
		{"Author", information.Author(), "Liquent Enterprise Services"},
		{"Creator", information.Creator(), "Liquent services server"},
		{"Producer", information.Producer(), "Amyuni PDF Converter version 4.0.0.9"},
		{"Keywords", information.Keywords(), ""},
		{"Subject", information.Subject(), ""},
		{"Title", information.Title(),
			"892B77DE781B4E71A1BEFB81A51A5ABC_20140326022424.docx"},
	} {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.what, c.got, c.want)
		}
	}

	created, ok := information.CreationDate()
	checkDate(t, "CreationDate", created, ok, wantDate(t, "D:20140326142505-02'00'"))
	modified, ok := information.ModificationDate()
	checkDate(t, "ModificationDate", modified, ok, wantDate(t, "20140326172513Z"))
}

// TestPDFBox3940 is testPDFBox3940: the same rebuild, on a file whose /Info has
// no modification date. Java asserts the creation date only, and so does this.
func TestPDFBox3940(t *testing.T) {
	document := openTarget(t, "PDFBOX-3940-079977.pdf")
	information := document.DocumentInformation()

	for _, c := range []struct{ what, got, want string }{
		{"Author", information.Author(), "Unknown"},
		{"Creator", information.Creator(), "C:REGULA~1IREGSFR_EQ_EM.WP"},
		{"Producer", information.Producer(), "Acrobat PDFWriter 3.02 for Windows"},
		{"Keywords", information.Keywords(), ""},
		{"Subject", information.Subject(), ""},
		{"Title", information.Title(), "C:REGULA~1IREGSFR_EQ_EM.PDF"},
	} {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.what, c.got, c.want)
		}
	}

	created, ok := information.CreationDate()
	checkDate(t, "CreationDate", created, ok,
		wantDate(t, "Tuesday, July 28, 1998 4:00:09 PM"))
}

// TestPDFBox3783 is testPDFBox3783: trash after %%EOF.
func TestPDFBox3783(t *testing.T) {
	loadsWithoutError(t, "PDFBOX-3783-72GLBIGUC6LB46ELZFBARRJTLN4RBSQM.pdf")
}

// TestPDFBox3785 is testPDFBox3785, and PDFBOX-3957: a truncated file with
// several revisions still counts its pages.
func TestPDFBox3785(t *testing.T) {
	document := openTarget(t, "PDFBOX-3785-202097.pdf")
	if got := document.NumberOfPages(); got != 11 {
		t.Errorf("NumberOfPages() = %d, want 11", got)
	}
}

// TestPDFBox3947 is testPDFBox3947: a broken object stream.
func TestPDFBox3947(t *testing.T) {
	loadsWithoutError(t, "PDFBOX-3947-670064.pdf")
}

// TestPDFBox3948 is testPDFBox3948: an object stream with unexpected newlines.
func TestPDFBox3948(t *testing.T) {
	loadsWithoutError(t, "PDFBOX-3948-EUWO6SQS5TM4VGOMRD3FLXZHU35V2CP2.pdf")
}

// TestPDFBox3949 is testPDFBox3949: an incomplete object stream.
func TestPDFBox3949(t *testing.T) {
	loadsWithoutError(t, "PDFBOX-3949-MKFYUGZWS3OPXLLVU2Z4LWCTVA5WNOGF.pdf")
}

// TestPDFBox3951 is testPDFBox3951: a truncated file, 143 pages of it.
func TestPDFBox3951(t *testing.T) {
	document := openTarget(t, "PDFBOX-3951-FIHUZWDDL2VGPOE34N6YHWSIGSH5LVGZ.pdf")
	if got := document.NumberOfPages(); got != 143 {
		t.Errorf("NumberOfPages() = %d, want 143", got)
	}
}

// TestPDFBox3964 is testPDFBox3964: a broken file, 10 pages of it.
func TestPDFBox3964(t *testing.T) {
	document := openTarget(t, "PDFBOX-3964-c687766d68ac766be3f02aaec5e0d713_2.pdf")
	if got := document.NumberOfPages(); got != 10 {
		t.Errorf("NumberOfPages() = %d, want 10", got)
	}
}

// TestPDFBox3977 is testPDFBox3977: the /Info found by the brute force search
// for the Info and Catalog dictionaries. Java asserts five of the seven fields.
func TestPDFBox3977(t *testing.T) {
	document := openTarget(t, "PDFBOX-3977-63NGFQRI44HQNPIPEJH5W2TBM6DJZWMI.pdf")
	information := document.DocumentInformation()

	for _, c := range []struct{ what, got, want string }{
		{"Creator", information.Creator(), "QuarkXPress(tm) 6.52"},
		{"Producer", information.Producer(), "Acrobat Distiller 7.0 pour Macintosh"},
		{"Title", information.Title(), "Fich sal Fabr corr1 (Page 6)"},
	} {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.what, c.got, c.want)
		}
	}

	created, ok := information.CreationDate()
	checkDate(t, "CreationDate", created, ok, wantDate(t, "D:20070608151915+02'00'"))
	modified, ok := information.ModificationDate()
	checkDate(t, "ModificationDate", modified, ok, wantDate(t, "D:20080604152122+02'00'"))
}

// TestParseGenko is testParseGenko: a file the Java calls susceptible to
// regression, with no assertion beyond loading.
func TestParseGenko(t *testing.T) {
	loadsWithoutError(t, "genko_oc_shiryo1.pdf")
}

// TestPDFBox4338 is testPDFBox4338: this one raised
// ArrayIndexOutOfBoundsException before the fix.
func TestPDFBox4338(t *testing.T) {
	loadsWithoutError(t, "PDFBOX-4338.pdf")
}

// TestPDFBox4339 is testPDFBox4339: this one raised NullPointerException.
func TestPDFBox4339(t *testing.T) {
	loadsWithoutError(t, "PDFBOX-4339.pdf")
}

// TestPDFBox4153 is testPDFBox4153: the outline of a file susceptible to
// regression. The bare name in the Java's javadoc is the attachment's; the
// pom saves it under the name below.
func TestPDFBox4153(t *testing.T) {
	document := openTarget(t, "PDFBOX-4153-WXMDXCYRWFDCMOSFQJ5OAJIAFXYRZ5OA.pdf")
	documentOutline := document.DocumentCatalog().DocumentOutline()
	if documentOutline == nil {
		t.Fatal("DocumentOutline() = nil, want the outline")
	}
	firstChild := documentOutline.FirstChild()
	if firstChild == nil {
		t.Fatal("FirstChild() = nil, want the first outline item")
	}
	if got := firstChild.Title(); got != "Main Menu" {
		t.Errorf("the first outline item is %q, want %q", got, "Main Menu")
	}
}

// TestPDFBox4490 is testPDFBox4490: three pages.
func TestPDFBox4490(t *testing.T) {
	document := openTarget(t, "PDFBOX-4490.pdf")
	if got := document.NumberOfPages(); got != 3 {
		t.Errorf("NumberOfPages() = %d, want 3", got)
	}
}

// TestPDFBox5025IsTheWholeFont is testPDFBox5025: "74191endobj", a length
// written with no space before the keyword. The number is what the parser has
// to get right, so it is read back off the embedded font program.
//
// The name differs from the Java's testPDFBox5025 because `pdfparser` already
// has a TestParseCOSNumberPDFBOX5025 for the same issue at the token level.
func TestPDFBox5025IsTheWholeFont(t *testing.T) {
	document := openTarget(t, "PDFBOX-5025.pdf")
	if got := document.NumberOfPages(); got != 1 {
		t.Fatalf("NumberOfPages() = %d, want 1", got)
	}

	resources := document.Page(0).Resources()
	if resources == nil {
		t.Fatal("Resources() = nil, want the page's resources")
	}
	embedded, err := resources.GetFont(cos.GetPDFName("F1"))
	if err != nil {
		t.Fatalf("GetFont(F1): %v", err)
	}
	if embedded == nil {
		t.Fatal("GetFont(F1) = nil, want the embedded font")
	}
	descriptor := embedded.FontDescriptor()
	if descriptor == nil {
		t.Fatal("FontDescriptor() = nil")
	}
	fontFile := descriptor.FontFile2()
	if fontFile == nil {
		t.Fatal("FontFile2() = nil, want the embedded TrueType program")
	}
	stream, isStream := fontFile.COSObject().(*cos.Stream)
	if !isStream {
		t.Fatalf("FontFile2 is %T, want a stream", fontFile.COSObject())
	}
	if got := stream.GetInt(cos.Length1); got != 74191 {
		t.Errorf("/Length1 = %d, want 74191 -- the length written as "+
			"\"74191endobj\", with no space before the keyword", got)
	}
}

// TestPDFBox3950 is testPDFBox3950, minus the rendering.
//
// Java loads the file, asserts four pages, and then renders every one of them,
// tolerating exactly one failure: page 3 may answer "Missing descendant font
// array". The port asserts the page count here and renders in
// rendering/raster, which is where the backend is; a test in this package
// cannot reach it without importing the renderer into the loader's own tests.
// See TestPDFBox3950Renders there.
func TestPDFBox3950(t *testing.T) {
	document := openTarget(t, "PDFBOX-3950-23EGDHXSBBYQLKYOKGZUOVYVNE675PRD.pdf")
	if got := document.NumberOfPages(); got != 4 {
		t.Errorf("NumberOfPages() = %d, want 4", got)
	}
}
