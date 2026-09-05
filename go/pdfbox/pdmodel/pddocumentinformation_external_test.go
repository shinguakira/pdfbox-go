package pdmodel_test

// Ported from
// pdfbox/src/test/java/org/apache/pdfbox/pdmodel/TestPDDocumentInformation.java,
// which tests the extraction of document-level metadata. Since 1.3.0.
//
// Java asserts null for the entries the document does not carry; the port's
// string accessors answer "", and its date accessors report false, so the
// assertions read those instead.

import (
	"path/filepath"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
)

// inputFixture is pdfbox's src/test/resources/input.
const inputFixture = "../../../pdfbox/src/test/resources/input/"

// TestMetadataExtraction is TestPDDocumentInformation.testMetadataExtraction.
func TestMetadataExtraction(t *testing.T) {
	// This document has been selected for this test as it contains custom metadata.
	doc, err := pdfbox.LoadPDF(filepath.Join(inputFixture, "hello3.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	defer doc.Close()
	info := doc.DocumentInformation()

	if got, want := info.Author(), "Brian Carrier"; got != want {
		t.Errorf("Wrong author: %q, want %q", got, want)
	}
	if _, ok := info.CreationDate(); !ok {
		t.Error("Wrong creationDate: there is none")
	}
	if got, want := info.Creator(), "Acrobat PDFMaker 8.1 for Word"; got != want {
		t.Errorf("Wrong creator: %q, want %q", got, want)
	}
	if got := info.Keywords(); got != "" {
		t.Errorf("Wrong keywords: %q, want the empty string, which is Java's null", got)
	}
	if _, ok := info.ModificationDate(); !ok {
		t.Error("Wrong modificationDate: there is none")
	}
	if got, want := info.Producer(), "Acrobat Distiller 8.1.0 (Windows)"; got != want {
		t.Errorf("Wrong producer: %q, want %q", got, want)
	}
	if got := info.Subject(); got != "" {
		t.Errorf("Wrong subject: %q, want the empty string", got)
	}
	if got := info.Trapped(); got != "" {
		t.Errorf("Wrong trapped: %q, want the empty string", got)
	}

	expectedMetadataKeys := []string{"CreationDate", "Author", "Creator",
		"Producer", "ModDate", "Company", "SourceModified", "Title"}
	metadataKeys := map[string]bool{}
	for _, key := range info.MetadataKeys() {
		metadataKeys[key] = true
	}
	if got, want := len(info.MetadataKeys()), len(expectedMetadataKeys); got != want {
		t.Errorf("Wrong metadata key count: %d, want %d", got, want)
	}
	for _, key := range expectedMetadataKeys {
		if !metadataKeys[key] {
			t.Errorf("Missing metadata key: %s", key)
		}
	}

	// Custom metadata fields.
	if got, want := info.CustomMetadataValue("Company"), "Basis Technology Corp."; got != want {
		t.Errorf("Wrong company: %q, want %q", got, want)
	}
	if got, want := info.CustomMetadataValue("SourceModified"), "D:20080819181502"; got != want {
		t.Errorf("Wrong sourceModified: %q, want %q", got, want)
	}
}

// TestPDFBox3068 is TestPDDocumentInformation.testPDFBox3068: an indirect
// /Title element of the /Info entry can be found.
func TestPDFBox3068(t *testing.T) {
	doc, err := pdfbox.LoadPDF(filepath.Join(catalogFixture, "PDFBOX-3068.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	defer doc.Close()
	documentInformation := doc.DocumentInformation()
	if got, want := documentInformation.Title(), "Title"; got != want {
		t.Errorf("Title() = %q, want %q", got, want)
	}
}
