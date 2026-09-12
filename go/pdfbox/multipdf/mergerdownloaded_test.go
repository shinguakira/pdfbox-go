package multipdf_test

// The cases of org.apache.pdfbox.multipdf.PDFMergerUtilityTest that read their
// documents out of `target/pdfs` rather than out of the checked-in corpus.
//
// They were deferred while that directory was empty;
// `migration/scripts/fetch-testdata.ps1` fills it. The helpers they use --
// `elementCounter`, `checkWithNumberTree`, `checkForPageOrphans`,
// `structureTreeRootOf`, `checkStructTreeRootCount` -- are in
// `pdfmergerutility_test.go`, ported with the rest of the class.
//
// Every expected value is the Java's.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/multipdf"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/image"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/form"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// targetPDFDir is Java's TARGETPDFDIR.
const targetPDFDir = "../../../pdfbox/target/pdfs/"

// loadDownloaded loads one of them, skipping where the fetch has not run.
func loadDownloaded(t *testing.T, name string) *pdmodel.PDDocument {
	t.Helper()
	path := targetPDFDir + name
	if _, err := os.Stat(path); err != nil {
		t.Skipf("%s is not there; run migration/scripts/fetch-testdata.ps1", name)
	}
	return loadFile(t, path)
}

// mergeWithItself is the shape most of these share: load the same document
// twice, append one to the other, save, read back. It answers the element count
// of the source tree and the path the merge was written to.
func mergeWithItself(t *testing.T, name, outName string) (singleCount, singleSet int, merged string) {
	t.Helper()
	merged = filepath.Join(t.TempDir(), outName)

	src := loadDownloaded(t, name)
	counter := &elementCounter{}
	counter.walk(structureTreeRootOf(t, src).K())
	singleCount, singleSet = counter.cnt, len(counter.set)

	dst := loadDownloaded(t, name)
	if err := multipdf.NewPDFMergerUtility().AppendDocument(dst, src); err != nil {
		t.Fatalf("AppendDocument: %v", err)
	}
	src.Close()
	if err := dst.SaveToFile(merged); err != nil {
		t.Fatalf("SaveToFile: %v", err)
	}
	dst.Close()
	return singleCount, singleSet, merged
}

// assertDoubled is the assertion every merge-with-itself case ends on: the
// merged tree holds twice the elements of the source, and twice the distinct
// ones.
func assertDoubled(t *testing.T, merged string, singleCount, singleSet int) *pdmodel.PDDocument {
	t.Helper()
	document := loadFile(t, merged)
	counter := &elementCounter{}
	counter.walk(structureTreeRootOf(t, document).K())
	if counter.cnt != singleCount*2 {
		t.Errorf("the merged tree holds %d elements, want %d",
			counter.cnt, singleCount*2)
	}
	if len(counter.set) != singleSet*2 {
		t.Errorf("the merged tree holds %d distinct elements, want %d",
			len(counter.set), singleSet*2)
	}
	return document
}

// TestStructureTreeMerge is testStructureTreeMerge, PDFBOX-3999.
func TestStructureTreeMerge(t *testing.T) {
	singleCount, singleSet, merged := mergeWithItself(t,
		"PDFBOX-3999-GeneralForbearance.pdf", "PDFBOX-3999-GeneralForbearance-merged.pdf")
	if singleCount != 134 || singleSet != 134 {
		t.Errorf("the source holds %d elements and %d distinct, want 134 and 134",
			singleCount, singleSet)
	}
	document := assertDoubled(t, merged, singleCount, singleSet)
	checkForPageOrphans(t, document)
	document.Close()
	checkStructTreeRootCount(t, merged)
}

// TestStructureTreeMerge2 is testStructureTreeMerge2: the same document with
// its form flattened first, which is a different tree to merge.
func TestStructureTreeMerge2(t *testing.T) {
	dir := t.TempDir()
	flattened := filepath.Join(dir, "PDFBOX-3999-GeneralForbearance-flattened.pdf")
	merged := filepath.Join(dir, "PDFBOX-3999-GeneralForbearance-flattened-merged.pdf")

	original := loadDownloaded(t, "PDFBOX-3999-GeneralForbearance.pdf")
	acroForm := form.AcroFormOfCatalog(original.DocumentCatalog())
	if acroForm == nil {
		original.Close()
		t.Fatal("the document has no AcroForm")
	}
	if err := acroForm.Flatten(); err != nil {
		original.Close()
		t.Fatalf("Flatten: %v", err)
	}
	if err := original.SaveToFile(flattened); err != nil {
		original.Close()
		t.Fatalf("SaveToFile: %v", err)
	}
	original.Close()

	src := loadFile(t, flattened)
	counter := &elementCounter{}
	counter.walk(structureTreeRootOf(t, src).K())
	singleCount, singleSet := counter.cnt, len(counter.set)
	if singleCount != 134 || singleSet != 134 {
		t.Errorf("the flattened source holds %d elements and %d distinct, "+
			"want 134 and 134", singleCount, singleSet)
	}

	dst := loadFile(t, flattened)
	if err := multipdf.NewPDFMergerUtility().AppendDocument(dst, src); err != nil {
		t.Fatalf("AppendDocument: %v", err)
	}
	src.Close()
	if err := dst.SaveToFile(merged); err != nil {
		t.Fatalf("SaveToFile: %v", err)
	}
	dst.Close()

	document := assertDoubled(t, merged, singleCount, singleSet)
	checkForPageOrphans(t, document)
	document.Close()
	checkStructTreeRootCount(t, merged)
}

// TestStructureTreeMerge3 is testStructureTreeMerge3, PDFBOX-4408.
func TestStructureTreeMerge3(t *testing.T) {
	singleCount, singleSet, merged := mergeWithItself(t,
		"PDFBOX-4408.pdf", "PDFBOX-4408-merged.pdf")
	if singleCount != 25 || singleSet != 25 {
		t.Errorf("the source holds %d elements and %d distinct, want 25 and 25",
			singleCount, singleSet)
	}
	document := assertDoubled(t, merged, singleCount, singleSet)
	checkWithNumberTree(t, document)
	checkForPageOrphans(t, document)
	document.Close()
	checkStructTreeRootCount(t, merged)
}

// breakStructParents is what both bogus cases do to one of the two documents:
// throw the structure tree root away and put numbers in /StructParents and
// /StructParent that the parent tree has no entry for.
func breakStructParents(t *testing.T, document *pdmodel.PDDocument) {
	t.Helper()
	document.DocumentCatalog().SetStructureTreeRoot(nil)
	page := document.Page(0)
	page.SetStructParents(9999)
	annotations := page.Annotations().ToSlice()
	if len(annotations) == 0 {
		t.Fatal("the first page has no annotation to break")
	}
	// SetStructParent is on *PDAnnotationBase, which the interface does not
	// expose; the entry is what the merge reads, so it is written directly.
	dictionary, isDictionary := annotations[0].COSObject().(*cos.Dictionary)
	if !isDictionary {
		t.Fatalf("the annotation is %T, want a dictionary", annotations[0].COSObject())
	}
	dictionary.SetInt(cos.StructParent, 9998)
}

// TestMergeBogusStructParents1 is testMergeBogusStructParents1: the
// **destination** has no structure tree root and carries /StructParents the
// parent tree cannot answer. The merge must still leave a walkable tree with
// no orphan page.
func TestMergeBogusStructParents1(t *testing.T) {
	src := loadDownloaded(t, "PDFBOX-4408.pdf")
	dst := loadDownloaded(t, "PDFBOX-4408.pdf")
	breakStructParents(t, dst)

	if err := multipdf.NewPDFMergerUtility().AppendDocument(dst, src); err != nil {
		t.Fatalf("AppendDocument: %v", err)
	}
	src.Close()
	checkWithNumberTree(t, dst)
	checkForPageOrphans(t, dst)
	dst.Close()
}

// TestMergeBogusStructParents2 is testMergeBogusStructParents2: the same
// damage, on the **source** instead.
func TestMergeBogusStructParents2(t *testing.T) {
	src := loadDownloaded(t, "PDFBOX-4408.pdf")
	dst := loadDownloaded(t, "PDFBOX-4408.pdf")
	breakStructParents(t, src)

	if err := multipdf.NewPDFMergerUtility().AppendDocument(dst, src); err != nil {
		t.Fatalf("AppendDocument: %v", err)
	}
	src.Close()
	checkWithNumberTree(t, dst)
	checkForPageOrphans(t, dst)
	dst.Close()
}

// parentTreeOf reads a document's parent tree as the map Java's
// PDFMergerUtility.getNumberTreeAsMap answers, with its smallest and largest
// key, which is what every case below asserts on.
func parentTreeOf(t *testing.T, document *pdmodel.PDDocument) (
	size, min, max, nextKey int) {
	t.Helper()
	root := structureTreeRootOf(t, document)
	tree, err := multipdf.GetNumberTreeAsMap(root.ParentTree())
	if err != nil {
		t.Fatalf("GetNumberTreeAsMap: %v", err)
	}
	min, max = -1, -1
	for key := range tree {
		if min == -1 || key < min {
			min = key
		}
		if key > max {
			max = key
		}
	}
	return len(tree), min, max, root.ParentTreeNextKey()
}

// assertParentTree is the four assertions together. Java writes max as
// `Collections.max(...) + 1`, so the wanted value here is that too.
func assertParentTree(t *testing.T, what string, document *pdmodel.PDDocument,
	wantSize, wantMin, wantMaxPlusOne, wantNextKey int) {
	t.Helper()
	size, min, max, nextKey := parentTreeOf(t, document)
	if size != wantSize {
		t.Errorf("%s: the parent tree holds %d entries, want %d", what, size, wantSize)
	}
	if min != wantMin {
		t.Errorf("%s: its smallest key is %d, want %d", what, min, wantMin)
	}
	if max+1 != wantMaxPlusOne {
		t.Errorf("%s: its largest key plus one is %d, want %d",
			what, max+1, wantMaxPlusOne)
	}
	if nextKey != wantNextKey {
		t.Errorf("%s: /ParentTreeNextKey is %d, want %d", what, nextKey, wantNextKey)
	}
}

// TestParentTree is testParentTree: the numbers of one document's parent tree,
// read without merging anything.
func TestParentTree(t *testing.T) {
	document := loadDownloaded(t, "PDFBOX-3999-GeneralForbearance.pdf")
	defer document.Close()
	assertParentTree(t, "PDFBOX-3999", document, 31, 0, 31, 31)
}

// TestStructureTreeMerge6 is testStructureTreeMerge6, PDFBOX-4418: two
// different documents, whose parent trees must add up after the merge and
// whose keys must not collide.
func TestStructureTreeMerge6(t *testing.T) {
	merged := filepath.Join(t.TempDir(), "PDFBOX-4418-merged.pdf")

	src := loadDownloaded(t, "PDFBOX-4418-000671.pdf")
	assertParentTree(t, "the source", src, 381, 0, 743, 743)

	dst := loadDownloaded(t, "PDFBOX-4418-000314.pdf")
	assertParentTree(t, "the destination", dst, 7, 321, 328, 408)

	if err := multipdf.NewPDFMergerUtility().AppendDocument(dst, src); err != nil {
		t.Fatalf("AppendDocument: %v", err)
	}
	src.Close()
	if err := dst.SaveToFile(merged); err != nil {
		t.Fatalf("SaveToFile: %v", err)
	}
	dst.Close()

	result := loadFile(t, merged)
	checkWithNumberTree(t, result)
	checkForPageOrphans(t, result)
	assertParentTree(t, "the merge", result, 381+7, 321, 408+743, 408+743)
	result.Close()
	checkStructTreeRootCount(t, merged)
}

// TestStructureTreeMerge7 is testStructureTreeMerge7, PDFBOX-4423: merged into
// an empty document, so the tree arrives unchanged and only
// /ParentTreeNextKey moves.
func TestStructureTreeMerge7(t *testing.T) {
	merged := filepath.Join(t.TempDir(), "PDFBOX-4423-merged.pdf")

	src := loadDownloaded(t, "PDFBOX-4423-000746.pdf")
	assertParentTree(t, "the source", src, 33, 31, 64, 126)

	dst := pdmodel.NewPDDocument()
	if err := multipdf.NewPDFMergerUtility().AppendDocument(dst, src); err != nil {
		t.Fatalf("AppendDocument: %v", err)
	}
	src.Close()
	if err := dst.SaveToFile(merged); err != nil {
		t.Fatalf("SaveToFile: %v", err)
	}
	dst.Close()

	result := loadFile(t, merged)
	checkWithNumberTree(t, result)
	checkForPageOrphans(t, result)
	// The tree is the source's, and the next key has come down to match it.
	assertParentTree(t, "the merge", result, 33, 31, 64, 64)
	result.Close()
	checkStructTreeRootCount(t, merged)
}

// TestMissingParentTreeNextKey is testMissingParentTreeNextKey: the
// destination has no /ParentTreeNextKey at all, so the merge has to work one
// out from the tree rather than read it.
func TestMissingParentTreeNextKey(t *testing.T) {
	merged := filepath.Join(t.TempDir(), "PDFBOX-4418-000314-merged.pdf")

	src := loadDownloaded(t, "PDFBOX-4418-000314.pdf")
	dst := loadDownloaded(t, "PDFBOX-4418-000314.pdf")
	structureTreeRootOf(t, dst).NodeDictionary().RemoveItem(cos.ParentTreeNextKey)

	if err := multipdf.NewPDFMergerUtility().AppendDocument(dst, src); err != nil {
		t.Fatalf("AppendDocument: %v", err)
	}
	src.Close()
	if err := dst.SaveToFile(merged); err != nil {
		t.Fatalf("SaveToFile: %v", err)
	}
	dst.Close()

	result := loadFile(t, merged)
	defer result.Close()
	if got := structureTreeRootOf(t, result).ParentTreeNextKey(); got != 656 {
		t.Errorf("/ParentTreeNextKey is %d, want 656", got)
	}
}

// TestSplitWithPgEntryAtTheTop is testSplitWithPgEntryAtTheTop, PDFBOX-6009: a
// document whose structure tree carries /Pg at the top splits into three
// one-page documents, each of which must still have a walkable tree.
func TestSplitWithPgEntryAtTheTop(t *testing.T) {
	document := loadDownloaded(t, "PDFBOX-6009.pdf")
	defer document.Close()

	splitter := multipdf.NewSplitter()
	splitter.SetSplitAtPage(1)
	parts, err := splitter.Split(document)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}
	if len(parts) != 3 {
		t.Fatalf("the split gave %d documents, want 3", len(parts))
	}
	for i, part := range parts {
		if got := part.NumberOfPages(); got != 1 {
			t.Errorf("part %d holds %d pages, want 1", i+1, got)
		}
		checkWithNumberTree(t, part)
		checkForPageOrphans(t, part)
		part.Close()
	}
}

// TestOutlinesSelfParent is testOutlinesSelfParent, PDFBOX-5939: a document
// whose outline has a node that is its own parent, merged with itself. What is
// asserted is that the merge terminates and gives two pages.
func TestOutlinesSelfParent(t *testing.T) {
	merged := filepath.Join(t.TempDir(), "PDFBOX-5939-google-docs-result.pdf")

	if _, err := os.Stat(targetPDFDir + "PDFBOX-5939-google-docs-1.pdf"); err != nil {
		t.Skip("PDFBOX-5939-google-docs-1.pdf is not there; " +
			"run migration/scripts/fetch-testdata.ps1")
	}
	merger := multipdf.NewPDFMergerUtility()
	merger.SetDestinationFileName(merged)
	for i := 0; i < 2; i++ {
		source, err := pdfio.OpenBufferedFile(targetPDFDir + "PDFBOX-5939-google-docs-1.pdf")
		if err != nil {
			t.Fatalf("opening the source: %v", err)
		}
		merger.AddSource(source)
	}
	if err := merger.MergeDocuments(); err != nil {
		t.Fatalf("MergeDocuments: %v", err)
	}

	result := loadFile(t, merged)
	defer result.Close()
	if got := result.NumberOfPages(); got != 2 {
		t.Errorf("NumberOfPages() = %d, want 2", got)
	}
}

// TestPDFBox515 is testPDFBox515, PDFBOX-515 and PDFBOX-5950: two documents
// where one has a stream buried in its /Info dictionary, at
// Info/ImPDF/Images/Kids/[0]. It passes only if the source is not closed too
// early, or if the clone is deep enough.
func TestPDFBox515(t *testing.T) {
	merged := filepath.Join(t.TempDir(), "PDFBOX-515-result.pdf")

	for _, name := range []string{"ComSquare1.pdf", "Ghostscript1.pdf"} {
		if _, err := os.Stat(targetPDFDir + name); err != nil {
			t.Skipf("%s is not there; run migration/scripts/fetch-testdata.ps1", name)
		}
	}
	merger := multipdf.NewPDFMergerUtility()
	merger.SetDestinationFileName(merged)
	for _, name := range []string{"ComSquare1.pdf", "Ghostscript1.pdf"} {
		source, err := pdfio.OpenBufferedFile(targetPDFDir + name)
		if err != nil {
			t.Fatalf("opening %s: %v", name, err)
		}
		merger.AddSource(source)
	}
	if err := merger.MergeDocuments(); err != nil {
		t.Fatalf("MergeDocuments: %v", err)
	}

	result := loadFile(t, merged)
	defer result.Close()
	if got := result.NumberOfPages(); got != 2 {
		t.Fatalf("NumberOfPages() = %d, want 2", got)
	}

	information := result.DocumentInformation().Dictionary()
	images := information.GetCOSDictionary(cos.GetPDFName("ImPDF"))
	if images == nil {
		t.Fatal("the merged /Info has no /ImPDF")
	}
	inner := images.GetCOSDictionary(cos.GetPDFName("Images"))
	if inner == nil {
		t.Fatal("/ImPDF has no /Images")
	}
	kids := inner.GetCOSArray(cos.Kids)
	if kids == nil || kids.Size() == 0 {
		t.Fatal("/Images has no /Kids")
	}
	// Java calls PDImageXObject.createXObject(imageDict, new PDResources());
	// the entry is a stream, and the port's constructor takes the PDStream.
	imageStream, isStream := kids.GetObject(0).(*cos.Stream)
	if !isStream {
		t.Fatalf("the first kid is %T, want a stream", kids.GetObject(0))
	}
	imageXObject := image.NewPDImageXObject(
		common.NewPDStream(imageStream), pdmodel.NewPDResources())
	rendered, err := imageXObject.Image()
	if err != nil {
		t.Fatalf("Image: %v", err)
	}
	if got := rendered.Bounds(); got.Dx() != 909 || got.Dy() != 233 {
		t.Errorf("the buried image is %dx%d, want 909x233", got.Dx(), got.Dy())
	}
}
