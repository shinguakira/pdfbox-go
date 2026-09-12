package multipdf_test

// The three cases of MergeAcroFormsTest and MergeAnnotationsTest that read
// their two documents out of `target/pdfs`.
//
// They were deferred while that directory was empty;
// `migration/scripts/fetch-testdata.ps1` fills it. Every expected value is the
// Java's.

import (
	"bytes"
	"os"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/multipdf"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/form"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// mergePDFDir is Java's TARGET_PDF_DIR.
const mergePDFDir = "../../../pdfbox/target/pdfs/"

// mergeTwo merges the two named documents and reads the result back. Java
// writes to a file and loads it; the port keeps the bytes, which is the same
// merge without the detour through the disk.
func mergeTwo(t *testing.T, first, second string) *pdmodel.PDDocument {
	t.Helper()
	for _, name := range []string{first, second} {
		if _, err := os.Stat(mergePDFDir + name); err != nil {
			t.Skipf("%s is not there; run migration/scripts/fetch-testdata.ps1", name)
		}
	}

	merger := multipdf.NewPDFMergerUtility()
	var merged bytes.Buffer
	merger.SetDestinationStream(&merged)
	for _, name := range []string{first, second} {
		source, err := pdfio.OpenBufferedFile(mergePDFDir + name)
		if err != nil {
			t.Fatalf("opening %s: %v", name, err)
		}
		merger.AddSource(source)
	}
	if err := merger.MergeDocuments(); err != nil {
		t.Fatalf("MergeDocuments: %v", err)
	}

	document, err := pdfbox.LoadPDFBytes(merged.Bytes())
	if err != nil {
		t.Fatalf("reading the merged document: %v", err)
	}
	t.Cleanup(func() { document.Close() })
	return document
}

// TestAnnotsEntry is MergeAcroFormsTest.testAnnotsEntry, PDFBOX-1031: both
// pages of the merge must keep their /Annots, and one annotation each.
func TestAnnotsEntry(t *testing.T) {
	merged := mergeTwo(t, "PDFBOX-1031-1.pdf", "PDFBOX-1031-2.pdf")

	if got := merged.NumberOfPages(); got != 2 {
		t.Fatalf("NumberOfPages() = %d, want 2", got)
	}
	for i := 0; i < 2; i++ {
		page := merged.Page(i)
		if got := page.Dictionary().GetDictionaryObject(cos.Annots); got == nil {
			t.Errorf("page %d has no /Annots entry", i+1)
		}
		annotations := page.Annotations()
		if annotations.Size() != 1 {
			t.Errorf("page %d holds %d annotations, want 1", i+1, annotations.Size())
		}
	}
}

// TestAPEntry is MergeAcroFormsTest.testAPEntry, PDFBOX-1100: the two fields
// of the merged form must keep their appearance and their value.
func TestAPEntry(t *testing.T) {
	merged := mergeTwo(t, "PDFBOX-1100-1.pdf", "PDFBOX-1100-2.pdf")

	if got := merged.NumberOfPages(); got != 2 {
		t.Fatalf("NumberOfPages() = %d, want 2", got)
	}
	acroForm := form.AcroFormOfCatalog(merged.DocumentCatalog())
	if acroForm == nil {
		t.Fatal("the merged document has no AcroForm")
	}
	for _, name := range []string{"Testfeld", "Testfeld2"} {
		field := acroForm.Field(name)
		if field == nil {
			t.Errorf("the merged form has no field %q", name)
			continue
		}
		dictionary, isDictionary := field.COSObject().(*cos.Dictionary)
		if !isDictionary {
			t.Errorf("%s is %T, want a dictionary", name, field.COSObject())
			continue
		}
		if dictionary.GetDictionaryObject(cos.AP) == nil {
			t.Errorf("%s has no /AP entry", name)
		}
		if dictionary.GetDictionaryObject(cos.V) == nil {
			t.Errorf("%s has no /V entry", name)
		}
	}
}

// TestLinkAnnotations is MergeAnnotationsTest.testLinkAnnotations,
// PDFBOX-1065: six pages, twelve named destinations, and every source
// annotation on page 1 must find its target on page 3 by the destination name
// the merge rewrote.
func TestLinkAnnotations(t *testing.T) {
	merged := mergeTwo(t, "PDFBOX-1065-1.pdf", "PDFBOX-1065-2.pdf")

	if got := merged.NumberOfPages(); got != 6 {
		t.Fatalf("NumberOfPages() = %d, want 6", got)
	}

	destinations := merged.DocumentCatalog().Dests()
	if destinations == nil {
		t.Fatal("the merged document has no /Dests")
	}
	dictionary, isDictionary := destinations.COSObject().(*cos.Dictionary)
	if !isDictionary {
		t.Fatalf("/Dests is %T, want a dictionary", destinations.COSObject())
	}
	if got := len(dictionary.KeySet()); got != 12 {
		t.Errorf("/Dests holds %d entries, want 12", got)
	}

	for _, pair := range [][2]int{{0, 2}, {3, 5}} {
		source := annotationsOfPage(t, merged, pair[0])
		target := annotationsOfPage(t, merged, pair[1])
		if len(source) != 3 {
			t.Errorf("page %d holds %d annotations, want 3", pair[0]+1, len(source))
		}
		if len(target) != 3 {
			t.Errorf("page %d holds %d annotations, want 3", pair[1]+1, len(target))
		}
		assertAnnotationsMatch(t, source, target, pair[0]+1, pair[1]+1)
	}
}

// annotationsOfPage reads one page's annotations as their dictionaries.
func annotationsOfPage(t *testing.T, document *pdmodel.PDDocument, index int) []*cos.Dictionary {
	t.Helper()
	annotations := document.Page(index).Annotations().ToSlice()
	out := make([]*cos.Dictionary, 0, len(annotations))
	for _, annotation := range annotations {
		dictionary, isDictionary := annotation.COSObject().(*cos.Dictionary)
		if !isDictionary {
			t.Fatalf("an annotation of page %d is %T", index+1, annotation.COSObject())
		}
		out = append(out, dictionary)
	}
	return out
}

// assertAnnotationsMatch is Java's private testAnnotationsMatch: every source
// annotation's /Dest must appear among the targets' with "annoRef_" in front,
// which is the rename the merge does to keep the two sets apart.
func assertAnnotationsMatch(t *testing.T, source, target []*cos.Dictionary,
	sourcePage, targetPage int) {
	t.Helper()
	byName := map[string]bool{}
	for _, annotation := range target {
		name, isName := annotation.GetDictionaryObject(cos.Dest).(*cos.Name)
		if !isName {
			t.Fatalf("a /Dest on page %d is %T, want a name",
				targetPage, annotation.GetDictionaryObject(cos.Dest))
		}
		byName[name.Name()] = true
	}
	for _, annotation := range source {
		name, isName := annotation.GetDictionaryObject(cos.Dest).(*cos.Name)
		if !isName {
			t.Fatalf("a /Dest on page %d is %T, want a name",
				sourcePage, annotation.GetDictionaryObject(cos.Dest))
		}
		if !byName["annoRef_"+name.Name()] {
			t.Errorf("page %d has a link to %q and page %d has no annoRef_%s "+
				"to answer it", sourcePage, name.Name(), targetPage, name.Name())
		}
	}
}
