package pdmodel_test

// Ported from
// pdfbox/src/test/java/org/apache/pdfbox/pdmodel/TestPDDocumentCatalog.java.
//
// The package is pdmodel_test rather than pdmodel: three of the cases load a
// PDF through the Loader, which lives above this package.
//
// handleOutputIntents is not here. It builds a PDOutputIntent from sRGB.icc
// through the PDDocument constructor, which reads the profile with
// java.awt.color.ICC_Profile; Go has no ICC engine, so that constructor is not
// ported. See migration/STATUS.md. The three accessors it then sets are covered
// by the assertions below on a hand-built intent.

import (
	"path/filepath"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/pagenavigation"
)

// catalogFixture is the directory the catalogue test PDFs live in.
const catalogFixture = "../../../pdfbox/src/test/resources/org/apache/pdfbox/pdmodel/"

// TestRetrievePageLabels is TestPDDocumentCatalog.retrievePageLabels, the test
// case for PDFBOX-90 - Support explicit retrieval of page labels.
func TestRetrievePageLabels(t *testing.T) {
	doc, err := pdfbox.LoadPDF(filepath.Join(catalogFixture, "test_pagelabels.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	defer doc.Close()
	cat := doc.DocumentCatalog()
	pageLabels, err := cat.PageLabels()
	if err != nil {
		t.Fatal(err)
	}
	labels := pageLabels.LabelsByPageIndices()
	want := []string{
		"A1", "A2", "A3", "i", "ii", "iii", "iv", "v", "vi", "vii",
		"Appendix I", "Appendix II",
	}
	if got := len(labels); got != 12 {
		t.Fatalf("LabelsByPageIndices() = %d labels, want 12", got)
	}
	for i, w := range want {
		if labels[i] != w {
			t.Errorf("labels[%d] = %q, want %q", i, labels[i], w)
		}
	}
}

// TestRetrievePageLabelsOnMalformedPdf is
// TestPDDocumentCatalog.retrievePageLabelsOnMalformedPdf, the test case for
// PDFBOX-900 - Handle malformed PDFs.
func TestRetrievePageLabelsOnMalformedPdf(t *testing.T) {
	doc, err := pdfbox.LoadPDF(filepath.Join(catalogFixture, "badpagelabels.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	defer doc.Close()
	cat := doc.DocumentCatalog()
	// LabelsByPageIndices() should not throw an exception
	pageLabels, err := cat.PageLabels()
	if err != nil {
		t.Fatal(err)
	}
	pageLabels.LabelsByPageIndices()
}

// TestRetrieveNumberOfPages is TestPDDocumentCatalog.retrieveNumberOfPages, the
// test case for PDFBOX-911 - Method PDDocument.getNumberOfPages() returns wrong
// number of pages.
func TestRetrieveNumberOfPages(t *testing.T) {
	doc, err := pdfbox.LoadPDF(filepath.Join(catalogFixture, "test.unc.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	defer doc.Close()
	if got := doc.NumberOfPages(); got != 4 {
		t.Errorf("NumberOfPages() = %d, want 4", got)
	}
}

// TestHandleOutputIntents is the half of TestPDDocumentCatalog.handleOutputIntents
// that does not build the intent from an ICC profile: PDFBOX-2687,
// ClassCastException when trying to get OutputIntents or add to it.
func TestHandleOutputIntents(t *testing.T) {
	doc, err := pdfbox.LoadPDF(filepath.Join(catalogFixture, "test.unc.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	defer doc.Close()
	catalog := doc.DocumentCatalog()

	// retrieve OutputIntents
	outputIntents := catalog.OutputIntents()
	if len(outputIntents) != 0 {
		t.Fatalf("OutputIntents() = %d, want none", len(outputIntents))
	}

	// create and add output intent
	oi := color.NewPDOutputIntent(cos.NewDictionary())
	oi.SetInfo("sRGB IEC61966-2.1")
	oi.SetOutputCondition("sRGB IEC61966-2.1")
	oi.SetOutputConditionIdentifier("sRGB IEC61966-2.1")
	oi.SetRegistryName("http://www.color.org")
	doc.DocumentCatalog().AddOutputIntent(oi)

	// retrieve OutputIntents
	outputIntents = catalog.OutputIntents()
	if len(outputIntents) != 1 {
		t.Fatalf("OutputIntents() = %d, want 1", len(outputIntents))
	}

	// set OutputIntents
	catalog.SetOutputIntents(outputIntents)
	outputIntents = catalog.OutputIntents()
	if len(outputIntents) != 1 {
		t.Fatalf("OutputIntents() = %d, want 1", len(outputIntents))
	}

	// The four accessors the Java sets, read back off the entry that survived
	// the round trip.
	got := outputIntents[0]
	for _, want := range []struct {
		what, got, want string
	}{
		{"Info", got.Info(), "sRGB IEC61966-2.1"},
		{"OutputCondition", got.OutputCondition(), "sRGB IEC61966-2.1"},
		{"OutputConditionIdentifier", got.OutputConditionIdentifier(), "sRGB IEC61966-2.1"},
		{"RegistryName", got.RegistryName(), "http://www.color.org"},
	} {
		if want.got != want.want {
			t.Errorf("%s() = %q, want %q", want.what, want.got, want.want)
		}
	}
}

// TestHandleBooleanInOpenAction is
// TestPDDocumentCatalog.handleBooleanInOpenAction, PDFBOX-3772 -- allow for
// COSBoolean.
func TestHandleBooleanInOpenAction(t *testing.T) {
	doc := pdmodel.NewPDDocument()
	defer doc.Close()
	doc.DocumentCatalog().Dictionary().SetBoolean(cos.OpenAction, false)
	openAction, err := doc.DocumentCatalog().OpenAction()
	if err != nil {
		t.Fatal(err)
	}
	if openAction != nil {
		t.Errorf("OpenAction() = %v, want nil", openAction)
	}
}

// TestNullThreads is TestPDDocumentCatalog.testNullThreads, PDFBOX-6186.
func TestNullThreads(t *testing.T) {
	doc := pdmodel.NewPDDocument()
	defer doc.Close()
	documentCatalog := doc.DocumentCatalog()
	if got := documentCatalog.Threads().Size(); got != 0 {
		t.Errorf("Threads() = %d, want 0", got)
	}
	documentCatalog.SetThreads([]*pagenavigation.PDThread{})
	if got := documentCatalog.Threads().Size(); got != 0 {
		t.Errorf("Threads() = %d, want 0", got)
	}
	documentCatalog.SetThreads(nil)
	if got := documentCatalog.Threads().Size(); got != 0 {
		t.Errorf("Threads() = %d, want 0", got)
	}
}
