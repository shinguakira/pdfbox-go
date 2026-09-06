package fdf_test

// Ported from pdfbox/src/test/java/org/apache/pdfbox/pdmodel/TestFDF.java.
//
// The package is fdf_test rather than fdf: the test loads both an FDF and a PDF
// through the Loader and imports the one into the other's AcroForm, and both
// the Loader and interactive/form sit above this package.
//
// testPDFBox5894 is not here: it reads target/pdfs/PDFBOX-5894.fdf, which the
// Maven build downloads from the issue tracker. See migration/STATUS.md.

import (
	"io"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/form"
)

// parserFixture is the directory the two FDF files and the form live in.
const parserFixture = "../../../../pdfbox/src/test/resources/org/apache/pdfbox/pdfparser/"

// TestLoad2 is TestFDF.testLoad2: load two simple fdf files with two fields.
// One of the files does not have a /Type/Catalog entry, which isn't required
// anyway (PDFBOX-3639).
func TestLoad2(t *testing.T) {
	checkFields(t, parserFixture+"withcatalog.fdf")
	checkFields(t, parserFixture+"nocatalog.fdf")
}

// checkFields is the private checkFields of the Java test.
func checkFields(t *testing.T, name string) {
	t.Helper()
	fdfDocument, err := pdfbox.LoadFDF(name)
	if err != nil {
		t.Fatal(err)
	}
	defer fdfDocument.Close()

	if err := fdfDocument.SaveXFDF(io.Discard); err != nil {
		t.Fatal(err)
	}

	fields := fdfDocument.Catalog().FDF().Fields()
	if got := fields.Size(); got != 2 {
		t.Fatalf("Fields() = %d, want 2", got)
	}
	for i, want := range []struct{ name, value string }{
		{"Field1", "Test1"},
		{"Field2", "Test2"},
	} {
		if got := fields.Get(i).PartialFieldName(); got != want.name {
			t.Errorf("fields[%d].PartialFieldName() = %q, want %q", i, got, want.name)
		}
		value, err := fields.Get(i).Value()
		if err != nil {
			t.Fatal(err)
		}
		if got, isString := value.(string); !isString || got != want.value {
			t.Errorf("fields[%d].Value() = %v, want %q", i, value, want.value)
		}
	}

	pdf, err := pdfbox.LoadPDF(parserFixture + "SimpleForm2Fields.pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer pdf.Close()
	acroForm := form.AcroFormOfCatalog(pdf.DocumentCatalog())
	if acroForm == nil {
		t.Fatal("AcroFormOfCatalog() = nil, want the form of SimpleForm2Fields.pdf")
	}
	if err := form.ImportFDFDocument(acroForm, fdfDocument); err != nil {
		t.Fatal(err)
	}
	for _, want := range []struct{ name, value string }{
		{"Field1", "Test1"},
		{"Field2", "Test2"},
	} {
		field := acroForm.Field(want.name)
		if field == nil {
			t.Fatalf("Field(%q) = nil, want the field", want.name)
		}
		if got := field.ValueAsString(); got != want.value {
			t.Errorf("Field(%q).ValueAsString() = %q, want %q", want.name, got, want.value)
		}
	}
}
