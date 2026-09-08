package multipdf_test

// Port of org.apache.pdfbox.multipdf.MergeAcroFormsTest and
// MergeAnnotationsTest.
//
// One of the four cases is here. `testAnnotsEntry` (PDFBOX-1031),
// `testAPEntry` (PDFBOX-1100) and `MergeAnnotationsTest.testLinkAnnotations`
// (PDFBOX-1065) all read their two documents out of `target/pdfs`, which the
// Maven build downloads; see migration/STATUS.md.
//
// The one that is left is the strongest of the four anyway: it merges a form
// with itself and compares every field of the result against
// `PDFBoxLegacyMerge-SameMerged.pdf`, which is the same merge done by PDFBox
// and checked in.

import (
	"path/filepath"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/multipdf"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/form"
)

// multipdfDir is the Java's IN_DIR.
const multipdfDir = "../../../pdfbox/src/test/resources/org/apache/pdfbox/multipdf/"

// TestLegacyModeMerge merges AcroFormForMerge.pdf with itself and compares the
// result field by field with the merge PDFBox produced.
func TestLegacyModeMerge(t *testing.T) {
	toBeMerged := filepath.Join(multipdfDir, "AcroFormForMerge.pdf")
	pdfOutput := filepath.Join(t.TempDir(), "PDFBoxLegacyMerge-SameMerged.pdf")

	merger := multipdf.NewPDFMergerUtility()
	merger.SetDestinationFileName(pdfOutput)
	if got := merger.DestinationFileName(); got != pdfOutput {
		t.Errorf("getDestinationFileName() is %q, want %q", got, pdfOutput)
	}
	// Java adds the same document twice, once as a File and once by name.
	if err := merger.AddSourceFile(toBeMerged); err != nil {
		t.Fatal(err)
	}
	if err := merger.AddSourceFile(toBeMerged); err != nil {
		t.Fatal(err)
	}
	if err := merger.MergeDocuments(); err != nil {
		t.Fatalf("mergeDocuments: %v", err)
	}
	merger.SetAcroFormMergeMode(multipdf.PDFBoxLegacyAcroFormMode)
	if got := merger.AcroFormMergeMode(); got != multipdf.PDFBoxLegacyAcroFormMode {
		t.Errorf("getAcroFormMergeMode() is %v, want the legacy mode", got)
	}

	compliantDocument := loadFile(t, filepath.Join(multipdfDir, "PDFBoxLegacyMerge-SameMerged.pdf"))
	defer compliantDocument.Close()
	toBeCompared := loadFile(t, pdfOutput)
	defer toBeCompared.Close()

	compliantAcroForm := form.AcroFormOfCatalog(compliantDocument.DocumentCatalog())
	toBeComparedAcroForm := form.AcroFormOfCatalog(toBeCompared.DocumentCatalog())
	if compliantAcroForm == nil || toBeComparedAcroForm == nil {
		t.Fatal("one of the two documents has no AcroForm")
	}

	if len(compliantAcroForm.Fields()) != len(toBeComparedAcroForm.Fields()) {
		t.Errorf("There shall be the same number of root fields: %d against %d",
			len(toBeComparedAcroForm.Fields()), len(compliantAcroForm.Fields()))
	}

	for compliantField := range compliantAcroForm.FieldTree().All() {
		name := compliantField.FullyQualifiedName()
		toBeComparedField := toBeComparedAcroForm.Field(name)
		if toBeComparedField == nil {
			t.Errorf("There shall be a field with the same FQN: %q", name)
			continue
		}
		compareFieldProperties(t, compliantField, toBeComparedField)
	}

	for toBeComparedField := range toBeComparedAcroForm.FieldTree().All() {
		name := toBeComparedField.FullyQualifiedName()
		compliantField := compliantAcroForm.Field(name)
		if compliantField == nil {
			t.Errorf("There shall be a field with the same FQN: %q", name)
			continue
		}
		compareFieldProperties(t, toBeComparedField, compliantField)
	}
}

// fieldPropertyKeys is the Java's list, with its reason: "Don't include too
// complex properties such as AP as this will fail the test because of a stack
// overflow".
var fieldPropertyKeys = []string{
	"FT", "T", "TU", "TM", "Ff", "V", "DV", "Opts", "TI", "I", "Rect", "DA",
}

func compareFieldProperties(t *testing.T, sourceField, toBeComparedField form.PDField) {
	t.Helper()
	sourceFieldCos := sourceField.COSObject().(*cos.Dictionary)
	toBeComparedCos := toBeComparedField.COSObject().(*cos.Dictionary)

	for _, key := range fieldPropertyKeys {
		name := cos.GetPDFName(key)
		sourceBase := sourceFieldCos.GetDictionaryObject(name)
		toBeComparedBase := toBeComparedCos.GetDictionaryObject(name)

		if sourceBase == nil {
			if toBeComparedBase != nil {
				t.Errorf("If the source property is null the compared property "+
					"shall be null too: /%s is %v", key, toBeComparedBase)
			}
			continue
		}
		// Java compares toString(), which for a COSBase is its rendering; the
		// port compares the same way rather than by identity, because the two
		// documents hold different objects.
		if toBeComparedBase == nil {
			t.Errorf("The content of the field properties shall be the same: "+
				"/%s is %v against nothing", key, sourceBase)
			continue
		}
		if got, want := stringOf(toBeComparedBase), stringOf(sourceBase); got != want {
			t.Errorf("The content of the field properties shall be the same: "+
				"/%s is %s, want %s", key, got, want)
		}
	}
}

// stringOf is Java's COSBase.toString().
func stringOf(base cos.Base) string {
	if stringer, isStringer := base.(interface{ String() string }); isStringer {
		return stringer.String()
	}
	return ""
}
