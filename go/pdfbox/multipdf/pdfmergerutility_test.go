package multipdf_test

// Port of org.apache.pdfbox.multipdf.PDFMergerUtilityTest.
//
// Thirty `@Test` methods, of which eighteen are ported. What is left out and
// why:
//
//   - Twelve read a PDF out of `target/pdfs`, a directory the Maven build fills
//     by downloading files from the issue tracker. The port fetches nothing in
//     a test, which is the reason every other slice gives for the same
//     omission; see migration/STATUS.md. They are
//     testStructureTreeMerge, 2, 3, 6, 7, testMissingParentTreeNextKey,
//     testMergeBogusStructParents1 and 2, testParentTree,
//     testSplitWithPgEntryAtTheTop, testOutlinesSelfParent and testPDFBox515.
//   - The pixel half of `checkMergeIdentical`, which renders both sources and
//     the merged file with `PDFRenderer` and compares them. Nothing implements
//     `rendering.Backend`; `track/raster` has it. The page-count half is here,
//     and so is a check the rendering was standing in for -- see
//     TestJpegCcitt.
//
// The assertion values -- 104 elements, 192 IDTree entries, page index 4 -- are
// the Java's.

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/multipdf"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/documentinterchange/logicalstructure"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/action"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/documentnavigation/destination"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/form"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/text"
)

// srcDir is the Java's SRCDIR.
const srcDir = "../../../pdfbox/src/test/resources/input/merge/"

// TestPDFMergerUtility merges "two PDF files with identically named but
// different global resources ... two fonts each named /TT1 and /TT0 that are
// Arial and Courier and vice versa in the second file. Revisions before 1613017
// fail this test because global resources were merged which made trouble when
// resources of the same kind had the same name."
func TestPDFMergerUtility(t *testing.T) {
	checkMergeIdentical(t, "PDFBox.GlobalResourceMergeTest.Doc01.decoded.pdf",
		"PDFBox.GlobalResourceMergeTest.Doc02.decoded.pdf",
		"GlobalResourceMergeTestResult1.pdf")
}

// TestPDFMergerUtility2 is the same two documents undecoded. See PDFBOX-2893.
func TestPDFMergerUtility2(t *testing.T) {
	checkMergeIdentical(t, "PDFBox.GlobalResourceMergeTest.Doc01.pdf",
		"PDFBox.GlobalResourceMergeTest.Doc02.pdf",
		"GlobalResourceMergeTestResult3.pdf")
}

// TestJpegCcitt merges "two PDF files with JPEG and CCITT ... A few revisions
// before 1704911 this test failed because the clone utility attempted to decode
// and re-encode the streams, see PDFBOX-2893".
//
// Java says that by rendering all three files and comparing pixels. That needs
// `track/raster`, and the sentence does not: if a stream were decoded and
// re-encoded on the way through, its bytes would change. So the merged file's
// image streams are compared with the sources' raw bytes, which is the claim
// the rendering was standing in for and is a stronger statement of it.
func TestJpegCcitt(t *testing.T) {
	merged := checkMergeIdentical(t, "jpegrgb.pdf", "multitiff.pdf",
		"JpegMultiMergeTestResult.pdf")

	want := append(imageStreamsOf(t, filepath.Join(srcDir, "jpegrgb.pdf")),
		imageStreamsOf(t, filepath.Join(srcDir, "multitiff.pdf"))...)
	got := imageStreamsOf(t, merged)
	if len(got) != len(want) {
		t.Fatalf("the merged file holds %d image streams, want %d", len(got), len(want))
	}
	for i := range want {
		if !bytes.Equal(got[i], want[i]) {
			t.Errorf("image stream %d is %d raw bytes in the merged file and %d in "+
				"the source; the clone utility decoded and re-encoded it",
				i, len(got[i]), len(want[i]))
		}
	}
}

// TestPDFMergerOpenAction is PDFBOX-3972: "Test that OpenAction page
// destination isn't lost after merge."
func TestPDFMergerOpenAction(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "MergerOpenActionTest1.pdf")
	second := filepath.Join(dir, "MergerOpenActionTest2.pdf")
	result := filepath.Join(dir, "MergerOpenActionTestResult.pdf")

	doc1 := pdmodel.NewPDDocument()
	doc1.AddPage(pdmodel.NewPDPage())
	doc1.AddPage(pdmodel.NewPDPage())
	doc1.AddPage(pdmodel.NewPDPage())
	if err := doc1.SaveToFile(first); err != nil {
		t.Fatal(err)
	}
	doc1.Close()

	doc2 := pdmodel.NewPDDocument()
	doc2.AddPage(pdmodel.NewPDPage())
	doc2.AddPage(pdmodel.NewPDPage())
	doc2.AddPage(pdmodel.NewPDPage())
	fitDestination := destination.NewPDPageFitDestination()
	fitDestination.SetPage(doc2.Page(1))
	doc2.DocumentCatalog().SetOpenAction(fitDestination)
	if err := doc2.SaveToFile(second); err != nil {
		t.Fatal(err)
	}
	doc2.Close()

	merger := multipdf.NewPDFMergerUtility()
	if err := merger.AddSourceFile(first); err != nil {
		t.Fatal(err)
	}
	if err := merger.AddSourceFile(second); err != nil {
		t.Fatal(err)
	}
	merger.SetDestinationFileName(result)
	if err := merger.MergeDocuments(); err != nil {
		t.Fatalf("mergeDocuments: %v", err)
	}

	mergedDoc, err := pdfbox.LoadPDF(result)
	if err != nil {
		t.Fatal(err)
	}
	defer mergedDoc.Close()
	documentCatalog := mergedDoc.DocumentCatalog()
	openAction, err := documentCatalog.OpenAction()
	if err != nil {
		t.Fatalf("getOpenAction: %v", err)
	}
	pageDestination, isPageDestination := openAction.(destination.PageDestination)
	if !isPageDestination {
		t.Fatalf("the open action is %T, want a page destination", openAction)
	}
	page, isPage := pageDestination.Page().(*pdmodel.PDPage)
	if !isPage {
		t.Fatalf("the destination names %T, not a page", pageDestination.Page())
	}
	if got := documentCatalog.Pages().IndexOf(page); got != 4 {
		t.Errorf("the open action points at page index %d, want 4", got)
	}
}

// TestStructureTreeMerge4 is PDFBOX-4417: "Same as the previous tests, but this
// one failed when the previous tests succeeded because of more bugs with
// cloning."
func TestStructureTreeMerge4(t *testing.T) {
	dir := t.TempDir()
	merged := filepath.Join(dir, "PDFBOX-4417-001031-merged.pdf")

	src := loadFixture(t, "PDFBOX-4417-001031.pdf")
	counter := &elementCounter{}
	counter.walk(structureTreeRootOf(t, src).K())
	singleCnt := counter.cnt
	singleSetSize := len(counter.set)
	if singleCnt != 104 {
		t.Errorf("the source has %d elements, want 104", singleCnt)
	}
	if singleSetSize != 104 {
		t.Errorf("the source has %d distinct elements, want 104", singleSetSize)
	}

	dst := loadFixture(t, "PDFBOX-4417-001031.pdf")
	if err := multipdf.NewPDFMergerUtility().AppendDocument(dst, src); err != nil {
		t.Fatalf("appendDocument: %v", err)
	}
	src.Close()
	if err := dst.SaveToFile(merged); err != nil {
		t.Fatal(err)
	}
	dst.Close()

	dst = loadFile(t, merged)
	// Assume that the merged tree has double element count
	counter = &elementCounter{}
	counter.walk(structureTreeRootOf(t, dst).K())
	if counter.cnt != singleCnt*2 {
		t.Errorf("the merged tree has %d elements, want %d", counter.cnt, singleCnt*2)
	}
	if len(counter.set) != singleSetSize*2 {
		t.Errorf("the merged tree has %d distinct elements, want %d",
			len(counter.set), singleSetSize*2)
	}
	checkWithNumberTree(t, dst)
	checkForPageOrphans(t, dst)
	dst.Close()
	checkStructTreeRootCount(t, merged)
}

// TestStructureTreeMerge5 is PDFBOX-4417 again, on the file whose "/K tree
// started with two dictionaries and not with an array".
func TestStructureTreeMerge5(t *testing.T) {
	dir := t.TempDir()
	merged := filepath.Join(dir, "PDFBOX-4417-054080-merged.pdf")

	src := loadFixture(t, "PDFBOX-4417-054080.pdf")
	counter := &elementCounter{}
	counter.walk(structureTreeRootOf(t, src).K())
	singleCnt := counter.cnt
	singleSetSize := len(counter.set)

	dst := loadFixture(t, "PDFBOX-4417-054080.pdf")
	if err := multipdf.NewPDFMergerUtility().AppendDocument(dst, src); err != nil {
		t.Fatalf("appendDocument: %v", err)
	}
	src.Close()
	if err := dst.SaveToFile(merged); err != nil {
		t.Fatal(err)
	}
	dst.Close()

	dst = loadFile(t, merged)
	checkWithNumberTree(t, dst)
	checkForPageOrphans(t, dst)

	counter = &elementCounter{}
	counter.walk(structureTreeRootOf(t, dst).K())
	if counter.cnt != singleCnt*2 {
		t.Errorf("the merged tree has %d elements, want %d", counter.cnt, singleCnt*2)
	}
	if len(counter.set) != singleSetSize*2 {
		t.Errorf("the merged tree has %d distinct elements, want %d",
			len(counter.set), singleSetSize*2)
	}
	dst.Close()
	checkStructTreeRootCount(t, merged)
}

// TestStructureTreeMergeIDTree is PDFBOX-4416, "Test merging of /IDTree", and
// PDFBOX-4009, "test merging to empty destination".
func TestStructureTreeMergeIDTree(t *testing.T) {
	dir := t.TempDir()
	merged := filepath.Join(dir, "PDFBOX-4416-IDTree-merged.pdf")

	merger := multipdf.NewPDFMergerUtility()
	src := loadFixture(t, "PDFBOX-4417-001031.pdf")
	dst := loadFixture(t, "PDFBOX-4417-054080.pdf")

	srcIDTreeMap := idTreeAsMap(t, structureTreeRootOf(t, src).IDTree())
	dstIDTreeMap := idTreeAsMap(t, structureTreeRootOf(t, dst).IDTree())
	expectedTotal := len(srcIDTreeMap) + len(dstIDTreeMap)
	if expectedTotal != 192 {
		t.Errorf("the two IDTrees hold %d entries between them, want 192", expectedTotal)
	}

	// PDFBOX-4009, test that empty dest doc still merges structure tree
	// (empty dest doc is used in command line app)
	emptyDest := pdmodel.NewPDDocument()
	if err := merger.AppendDocument(emptyDest, src); err != nil {
		t.Fatalf("appendDocument into an empty document: %v", err)
	}
	src.Close()
	src = emptyDest
	if got := structureTreeRootOf(t, src).ParentTreeNextKey(); got != 4 {
		t.Errorf("the empty destination's ParentTreeNextKey is %d, want 4", got)
	}

	if err := merger.AppendDocument(dst, src); err != nil {
		t.Fatalf("appendDocument: %v", err)
	}
	src.Close()
	if err := dst.SaveToFile(merged); err != nil {
		t.Fatal(err)
	}
	dst.Close()

	dst = loadFile(t, merged)
	checkWithNumberTree(t, dst)
	checkForPageOrphans(t, dst)
	if got := len(idTreeAsMap(t, structureTreeRootOf(t, dst).IDTree())); got != expectedTotal {
		t.Errorf("the merged IDTree holds %d entries, want %d", got, expectedTotal)
	}
	dst.Close()
	checkStructTreeRootCount(t, merged)
}

// TestFileDeletion is PDFBOX-4383, "Test that file can be deleted after merge".
//
// Java holds the sources open as RandomAccessReads and deletes all three files
// at the end; Go's equivalent is that the merge closed what it opened, and the
// deletion is the assertion.
func TestFileDeletion(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "PDFBOX-4383-result.pdf")
	inFile1 := filepath.Join(dir, "PDFBOX-4383-src1.pdf")
	inFile2 := filepath.Join(dir, "PDFBOX-4383-src2.pdf")

	createSimpleFile(t, inFile1)
	createSimpleFile(t, inFile2)

	out, err := os.Create(outFile)
	if err != nil {
		t.Fatal(err)
	}
	merger := multipdf.NewPDFMergerUtility()
	merger.SetDestinationStream(out)
	if merger.DestinationStream() != out {
		t.Error("getDestinationStream() is not the stream that was set")
	}
	if err := merger.AddSourceFile(inFile1); err != nil {
		t.Fatal(err)
	}
	if err := merger.AddSourceFile(inFile2); err != nil {
		t.Fatal(err)
	}
	if err := merger.MergeDocuments(); err != nil {
		t.Fatalf("mergeDocuments: %v", err)
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}

	doc := loadFile(t, outFile)
	if got := doc.NumberOfPages(); got != 2 {
		t.Errorf("the merged document has %d pages, want 2", got)
	}
	doc.Close()

	for _, path := range []string{inFile1, inFile2, outFile} {
		if err := os.Remove(path); err != nil {
			t.Errorf("the merge left %s open: %v", path, err)
		}
	}
}

// TestPDFBox5198_2 checks "that there is a top level Document and Parts below
// in a merge of 2 documents".
func TestPDFBox5198_2(t *testing.T) {
	checkMergedParts(t, 2, "PDFA3A-merged2.pdf")
}

// TestPDFBox5198_3 is the same for three documents.
func TestPDFBox5198_3(t *testing.T) {
	checkMergedParts(t, 3, "PDFA3A-merged3.pdf")
}

// checkMergedParts merges PDFA3A.pdf with itself the given number of times and
// checks the /Document and /Part structure of the result.
func checkMergedParts(t *testing.T, copies int, name string) {
	t.Helper()
	result := filepath.Join(t.TempDir(), name)
	merger := multipdf.NewPDFMergerUtility()
	for i := 0; i < copies; i++ {
		if err := merger.AddSourceFile(filepath.Join(srcDir, "PDFA3A.pdf")); err != nil {
			t.Fatal(err)
		}
	}
	merger.SetDestinationFileName(result)
	if err := merger.MergeDocuments(); err != nil {
		t.Fatalf("mergeDocuments: %v", err)
	}
	checkParts(t, result)
}

// checkParts is the Java's: "Check that there is a top level Document and Parts
// below."
func checkParts(t *testing.T, path string) {
	t.Helper()
	doc := loadFile(t, path)
	defer doc.Close()

	structureTreeRoot := structureTreeRootOf(t, doc)
	topDict, isDictionary := structureTreeRoot.K().(*cos.Dictionary)
	if !isDictionary {
		t.Fatalf("the /K entry is %T, want a dictionary", structureTreeRoot.K())
	}
	if got := topDict.GetItem(cos.S); got != cos.DocumentName {
		t.Errorf("the top element is /%v, want /Document", got)
	}
	if got := topDict.GetCOSDictionary(cos.P); got != structureTreeRoot.COSObject() {
		t.Error("the top element's /P is not the structure tree root")
	}
	kArray := topDict.GetCOSArray(cos.K)
	if kArray == nil {
		t.Fatal("the top element has no /K array")
	}
	if got := kArray.Size(); got != doc.NumberOfPages() {
		t.Fatalf("the top element has %d kids and the document has %d pages",
			got, doc.NumberOfPages())
	}
	for i := 0; i < kArray.Size(); i++ {
		dict, isDictionary := kArray.GetObject(i).(*cos.Dictionary)
		if !isDictionary {
			t.Fatalf("kid %d is %T, want a dictionary", i, kArray.GetObject(i))
		}
		if got := dict.GetItem(cos.S); got != cos.Part {
			t.Errorf("kid %d is /%v, want /Part", i, got)
		}
		if got := dict.GetCOSDictionary(cos.P); got != topDict {
			t.Errorf("kid %d's /P is not the top element", i)
		}
	}
}

// checkMergeIdentical merges two files of the corpus and answers the path of
// the result.
//
// Java renders every page of both sources and of the result and compares them
// pixel for pixel; that is `track/raster`'s. What is left is the page count,
// which is the Java's own first assertion.
func checkMergeIdentical(t *testing.T, filename1, filename2, mergeFilename string) string {
	t.Helper()
	src1 := loadFixture(t, filename1)
	src1PageCount := src1.NumberOfPages()
	src1.Close()

	src2 := loadFixture(t, filename2)
	src2PageCount := src2.NumberOfPages()
	src2.Close()

	result := filepath.Join(t.TempDir(), mergeFilename)
	merger := multipdf.NewPDFMergerUtility()
	if err := merger.AddSourceFile(filepath.Join(srcDir, filename1)); err != nil {
		t.Fatal(err)
	}
	if err := merger.AddSourceFile(filepath.Join(srcDir, filename2)); err != nil {
		t.Fatal(err)
	}
	merger.SetDestinationFileName(result)
	if err := merger.MergeDocuments(); err != nil {
		t.Fatalf("mergeDocuments: %v", err)
	}

	mergedDoc := loadFile(t, result)
	defer mergedDoc.Close()
	if got := mergedDoc.NumberOfPages(); got != src1PageCount+src2PageCount {
		t.Errorf("the merged document has %d pages and the two sources have %d and %d",
			got, src1PageCount, src2PageCount)
	}
	return result
}

// imageStreamsOf answers the raw, still-encoded bytes of every image XObject on
// every page of a document, in page order.
func imageStreamsOf(t *testing.T, path string) [][]byte {
	t.Helper()
	doc := loadFile(t, path)
	defer doc.Close()

	var streams [][]byte
	for page := range doc.Pages().All {
		resources := page.Resources()
		if resources == nil {
			continue
		}
		for _, name := range resources.XObjectNames() {
			xobject, err := resources.GetXObject(name)
			if err != nil {
				t.Fatalf("reading %s of %s: %v", name.Name(), path, err)
			}
			stream, isStream := xobject.COSObject().(*cos.Stream)
			if !isStream || stream.GetCOSName(cos.Subtype) != cos.Image {
				continue
			}
			reader, err := stream.CreateRawReader()
			if err != nil {
				t.Fatalf("reading %s of %s: %v", name.Name(), path, err)
			}
			var raw bytes.Buffer
			if _, err := raw.ReadFrom(reader); err != nil {
				t.Fatal(err)
			}
			streams = append(streams, raw.Bytes())
		}
	}
	return streams
}

// createSimpleFile is the Java's one-page document.
func createSimpleFile(t *testing.T, path string) {
	t.Helper()
	doc := pdmodel.NewPDDocument()
	defer doc.Close()
	doc.AddPage(pdmodel.NewPDPage())
	if err := doc.SaveToFile(path); err != nil {
		t.Fatal(err)
	}
}

// loadFixture loads a document out of the Java's input/merge corpus.
func loadFixture(t *testing.T, name string) *pdmodel.PDDocument {
	t.Helper()
	return loadFile(t, filepath.Join(srcDir, name))
}

func loadFile(t *testing.T, path string) *pdmodel.PDDocument {
	t.Helper()
	doc, err := pdfbox.LoadPDF(path)
	if err != nil {
		t.Fatalf("loading %s: %v", path, err)
	}
	return doc
}

// structureTreeRootOf is `doc.getDocumentCatalog().getStructureTreeRoot()`.
func structureTreeRootOf(t *testing.T,
	doc *pdmodel.PDDocument) *logicalstructure.PDStructureTreeRoot {
	t.Helper()
	root := doc.DocumentCatalog().StructureTreeRoot()
	if root == nil {
		t.Fatal("the document has no structure tree root")
	}
	return root
}

// idTreeAsMap is PDFMergerUtility.getIDTreeAsMap, which the Java test calls
// directly.
func idTreeAsMap(t *testing.T,
	idTree common.NameTreeNode[*logicalstructure.PDStructureElement],
) map[string]*logicalstructure.PDStructureElement {
	t.Helper()
	names, err := multipdf.GetIDTreeAsMap(idTree)
	if err != nil {
		t.Fatalf("getIDTreeAsMap: %v", err)
	}
	return names
}

// checkStructTreeRootCount asserts a merged file has exactly one structure tree
// root object.
func checkStructTreeRootCount(t *testing.T, path string) {
	t.Helper()
	doc := loadFile(t, path)
	defer doc.Close()
	objects := doc.Document().ObjectsByType(cos.StructTreeRoot)
	if len(objects) != 1 {
		t.Errorf("%s holds %d structure tree root objects, want 1", path, len(objects))
	}
}

// elementCounter is the Java's inner class of the same name: it counts the
// structure elements that carry a /Pg, and the distinct ones.
type elementCounter struct {
	cnt int
	set map[*cos.Dictionary]bool
}

func (c *elementCounter) add(dictionary *cos.Dictionary) {
	c.cnt++
	if c.set == nil {
		c.set = map[*cos.Dictionary]bool{}
	}
	c.set[dictionary] = true
}

func (c *elementCounter) walk(base cos.Base) {
	switch value := base.(type) {
	case *cos.Array:
		for i := 0; i < value.Size(); i++ {
			c.walk(value.GetObject(i))
		}
	case *cos.Dictionary:
		if value.ContainsKey(cos.Pg) {
			c.add(value)
		} else if value.ContainsKey(cos.K) {
			// at least 1 kid with dict with /Pg, /MCID and type /MCR
			// happens with confidential file from PDFBOX-6009
			if kidArray := value.GetCOSArray(cos.K); kidArray != nil {
				for i := 0; i < kidArray.Size(); i++ {
					kid, isDictionary := kidArray.GetObject(i).(*cos.Dictionary)
					if isDictionary && kid.ContainsKey(cos.Pg) && kid.ContainsKey(cos.MCID) {
						c.add(value)
						break
					}
				}
			}
		}
		if value.ContainsKey(cos.K) {
			c.walk(value.GetDictionaryObject(cos.K))
		}
	}
}

// checkForPageOrphans is the Java's: "check for orphan pages in the
// StructTreeRoot/K, StructTreeRoot/ParentTree and StructTreeRoot/IDTree trees."
func checkForPageOrphans(t *testing.T, doc *pdmodel.PDDocument) {
	t.Helper()
	pageTree := doc.Pages()
	structureTreeRoot := structureTreeRootOf(t, doc)
	checkElement(t, pageTree, structureTreeRoot.ParentTree().COSObject(),
		structureTreeRoot.COSObject().(*cos.Dictionary))
	if structureTreeRoot.K() == nil {
		t.Fatal("the structure tree root has no /K")
	}
	checkElement(t, pageTree, structureTreeRoot.K(), structureTreeRoot.COSObject().(*cos.Dictionary))
	checkForIDTreeOrphans(t, pageTree, structureTreeRoot)
	checkParentTreeAgainstK(t, structureTreeRoot)
}

// checkParentTreeAgainstK checks "that elements in the /ParentTree are in the
// /K tree".
func checkParentTreeAgainstK(t *testing.T,
	structureTreeRoot *logicalstructure.PDStructureTreeRoot) {
	t.Helper()
	counter := &elementCounter{}
	counter.walk(structureTreeRoot.K())
	numberTreeAsMap, err := multipdf.GetNumberTreeAsMap(structureTreeRoot.ParentTree())
	if err != nil {
		t.Fatalf("getNumberTreeAsMap: %v", err)
	}
	for key, value := range numberTreeAsMap {
		if value == nil {
			continue
		}
		array, isArray := value.COSObject().(*cos.Array)
		if !isArray {
			// can't check this COSDictionary; ElementsCounter only counts those
			// with a /Pg entry
			continue
		}
		for i := 0; i < array.Size(); i++ {
			element, isDictionary := array.GetObject(i).(*cos.Dictionary)
			if isDictionary && !counter.set[element] {
				t.Errorf("Element %d:%d from /ParentTree missing in /K ", key, i)
			}
		}
	}
}

func checkForIDTreeOrphans(t *testing.T, pageTree *pdmodel.PDPageTree,
	structureTreeRoot *logicalstructure.PDStructureTreeRoot) {
	t.Helper()
	idTree := structureTreeRoot.IDTree()
	if idTree == nil {
		return
	}
	for _, element := range idTreeAsMap(t, idTree) {
		if element.Page() != nil {
			checkForPage(t, pageTree, element)
		}
		if len(element.Kids()) > 0 {
			checkElement(t, pageTree, element.COSObject().(*cos.Dictionary).GetDictionaryObject(cos.K),
				element.COSObject().(*cos.Dictionary))
		}
	}
}

// checkForPage asserts a structure element's page is in the page tree.
func checkForPage(t *testing.T, pageTree *pdmodel.PDPageTree,
	structureElement *logicalstructure.PDStructureElement) {
	t.Helper()
	page, isPage := structureElement.Page().(*pdmodel.PDPage)
	if isPage && pageTree.IndexOf(page) == -1 {
		t.Error("Page is not in the page tree")
	}
}

// checkElement walks a structure or number tree, asserting that every page it
// names is in the page tree.
//
// "Each element can be an array, a dictionary or a number." The comment naming
// the four specification tables is in the Java above this method.
func checkElement(t *testing.T, pageTree *pdmodel.PDPageTree, base cos.Base,
	parentDict *cos.Dictionary) {
	t.Helper()
	switch value := base.(type) {
	case *cos.Array:
		for i := 0; i < value.Size(); i++ {
			checkElement(t, pageTree, value.GetObject(i), parentDict)
		}
	case *cos.Dictionary:
		checkElementDictionary(t, pageTree, value, parentDict)
	}
}

func checkElementDictionary(t *testing.T, pageTree *pdmodel.PDPageTree,
	kdict *cos.Dictionary, parentDict *cos.Dictionary) {
	t.Helper()
	if kdict.ContainsKey(cos.Pg) {
		checkForPage(t, pageTree, logicalstructure.NewPDStructureElementOf(kdict))
	}
	if kdict.ContainsKey(cos.K) {
		checkElement(t, pageTree, kdict.GetDictionaryObject(cos.K), kdict)

		// Check that the /P entry points to the correct object
		// The port's StructureNode interface does not carry Kids, which both
		// of its implementations have; Java's abstract class does.
		node, hasKids := logicalstructure.Create(kdict).(interface{ Kids() []any })
		if hasKids {
			for _, kid := range node.Kids() {
				element, isElement := kid.(*logicalstructure.PDStructureElement)
				if !isElement {
					continue
				}
				if parent := element.Parent(); parent == nil || parent.COSObject() != cos.Base(kdict) {
					t.Error("a kid's /P does not point back at its parent")
				}
			}
		}
		return
	}

	// if we're in a number tree, check /Nums and /Kids
	if kdict.ContainsKey(cos.Kids) {
		checkElement(t, pageTree, kdict.GetDictionaryObject(cos.Kids), kdict)
	} else if kdict.ContainsKey(cos.Nums) {
		checkElement(t, pageTree, kdict.GetDictionaryObject(cos.Nums), kdict)
	}

	if kind := kdict.GetDictionaryObject(cos.Type); kind == cos.OBJR || kind == cos.MCR {
		if kdict.GetCOSDictionary(cos.Pg) == nil && parentDict.GetCOSDictionary(cos.Pg) == nil {
			t.Error("an /OBJR or /MCR has no /Pg and neither has its parent")
		}
	}

	// if we're an object reference dictionary (/OBJR), check the obj
	if kdict.ContainsKey(cos.Obj) {
		checkObjectReference(t, pageTree, kdict)
	}
}

func checkObjectReference(t *testing.T, pageTree *pdmodel.PDPageTree, kdict *cos.Dictionary) {
	t.Helper()
	obj, isDictionary := kdict.GetDictionaryObject(cos.Obj).(*cos.Dictionary)
	if !isDictionary {
		t.Fatalf("the /Obj entry is %T, want a dictionary", kdict.GetDictionaryObject(cos.Obj))
	}
	kind := obj.GetDictionaryObject(cos.Type)
	subtype := obj.GetDictionaryObject(cos.Subtype)
	if kind != cos.Annot && subtype != cos.Link {
		// TODO needs to be investigated. Specification mentions
		// "such as an XObject or an annotation"
		t.Fatalf("Other type: %v, obj: %v", kind, obj)
	}
	ann, err := annotation.CreateAnnotation(obj)
	if err != nil {
		t.Fatalf("createAnnotation: %v", err)
	}
	if link, isLink := ann.(*annotation.PDAnnotationLink); isLink {
		// PDFBOX-5928: check whether the destination of a link annotation is an orphan
		checkLinkDestination(t, pageTree, link)
	}
	// Page is on PDAnnotationBase rather than on the interface, the way
	// StructParent is.
	pager, hasPage := ann.(interface{ Page() annotation.PageLike })
	if !hasPage {
		return
	}
	page, isPage := pager.Page().(*pdmodel.PDPage)
	if isPage && pageTree.IndexOf(page) == -1 {
		t.Error("Annotation page is not in the page tree")
	}
}

func checkLinkDestination(t *testing.T, pageTree *pdmodel.PDPageTree,
	link *annotation.PDAnnotationLink) {
	t.Helper()
	dest, err := link.Destination()
	if err != nil {
		t.Fatalf("getDestination: %v", err)
	}
	if dest == nil {
		if goTo, isGoTo := link.Action().(*action.PDActionGoTo); isGoTo {
			dest, err = goTo.Destination()
			if err != nil {
				t.Fatalf("getDestination: %v", err)
			}
		}
	}
	pageDestination, isPageDestination := dest.(destination.PageDestination)
	if !isPageDestination {
		return
	}
	destPage, isPage := pageDestination.Page().(*pdmodel.PDPage)
	if isPage && pageTree.IndexOf(destPage) == -1 {
		t.Error("Annotation destination page is not in the page tree")
	}
}

// checkWithNumberTree is PDFBOX-4408: "Check that /StructParents values from
// pages and /StructParent values from annotations are found in the
// /ParentTree", expanded in 2025 to check "that all MCIDs of a page content
// stream have an entry in the ParentTree".
func checkWithNumberTree(t *testing.T, document *pdmodel.PDDocument) {
	t.Helper()
	documentCatalog := document.DocumentCatalog()
	structureTreeRoot := structureTreeRootOf(t, document)
	parentTree := structureTreeRoot.ParentTree()
	if structureTreeRoot.ParentTreeNextKey() == -1 {
		t.Error("the structure tree root has no /ParentTreeNextKey")
	}
	numberTreeAsMap, err := multipdf.GetNumberTreeAsMap(parentTree)
	if err != nil {
		t.Fatalf("getNumberTreeAsMap: %v", err)
	}

	acroForm := form.AcroFormOfCatalog(documentCatalog)
	if acroForm != nil {
		for field := range acroForm.FieldTree().All() {
			for _, widget := range field.Widgets() {
				if key := widget.StructParent(); key >= 0 {
					if _, found := numberTreeAsMap[key]; !found {
						t.Errorf("field '%s' /StructParent %d missing in /ParentTree",
							field.FullyQualifiedName(), key)
					}
				}
			}
		}
	}

	pageTree := document.Pages()
	for page := range pageTree.All {
		pageNum := pageTree.IndexOf(page) + 1
		if structParents := page.StructParents(); structParents >= 0 {
			value, found := numberTreeAsMap[structParents]
			if !found {
				t.Errorf("/StructParents %d from page %d not found in /ParentTree",
					structParents, pageNum)
				continue
			}
			array, isArray := value.COSObject().(*cos.Array)
			if !isArray {
				t.Errorf("Expected array in page %d, got %T", pageNum, value.COSObject())
				continue
			}
			checkMarkedContent(t, page, pageNum, array)
		}
		for ann := range page.Annotations().All {
			if key := ann.COSObject().(*cos.Dictionary).GetInt(cos.StructParent); key >= 0 {
				if _, found := numberTreeAsMap[key]; !found {
					t.Errorf("/StructParent %d missing in /ParentTree", key)
				}
			}
		}
	}

	// might also test image and form dictionaries...
}

// checkMarkedContent is the inner half of checkWithNumberTree: every marked
// content sequence of a page names an element of that page's /ParentTree entry,
// and that element claims the sequence back.
func checkMarkedContent(t *testing.T, page *pdmodel.PDPage, pageNum int, array *cos.Array) {
	t.Helper()
	extractor := text.NewPDFMarkedContentExtractor()
	if err := extractor.ProcessPage(page); err != nil {
		t.Fatalf("processPage: %v", err)
	}
	last := -1
	empty := true
	for _, pdMarkedContent := range extractor.MarkedContents() {
		if pdMarkedContent.Properties() == nil {
			continue
		}
		mcid := pdMarkedContent.MCID()
		if mcid < 0 {
			continue
		}
		// "For a page object (...), the value shall be an array of references
		// to the parent elements of those marked-content sequences."
		// this means that the /Pg entry doesn't have to match the page
		dict, isDictionary := array.GetObject(mcid).(*cos.Dictionary)
		if !isDictionary {
			t.Fatalf("page %d, mcid %d: the /ParentTree entry is %T",
				pageNum, mcid, array.GetObject(mcid))
		}
		empty = false
		if mcid > last {
			last = mcid
		}
		structureElement := logicalstructure.NewPDStructureElementOf(dict)
		if !kidsClaimMCID(t, page, structureElement, structureElement.Kids(), mcid) {
			t.Errorf("page: %d, mcid: %d not found", pageNum, mcid)
		}
	}
	// actual count may be larger if last element is null, e.g. PDFBOX-4408
	// set can be empty, see last page of pdf_32000_2008.pdf
	if !empty && last > array.Size()-1 {
		t.Errorf("page %d names mcid %d and its /ParentTree entry holds %d",
			pageNum, last, array.Size())
	}
}

// kidsClaimMCID reports whether one of a structure element's kids is the given
// marked content identifier.
func kidsClaimMCID(t *testing.T, page *pdmodel.PDPage,
	structureElement *logicalstructure.PDStructureElement, kids []any, mcid int) bool {
	t.Helper()
	for _, kid := range kids {
		if number, isNumber := kid.(int); isNumber && number == mcid {
			return true
		}
		mcr, isReference := kid.(*logicalstructure.PDMarkedContentReference)
		if !isReference || mcr.MCID() != mcid {
			continue
		}
		// Java compares with assertEquals, and PDPage.equals is
		// `other.getCOSObject() == this.getCOSObject()` -- the dictionary, not
		// the wrapper. mcr.getPage() builds a fresh PDPage over the same
		// dictionary every call, so comparing the wrappers reports every page
		// as a different one.
		if mcrPage := mcr.Page(); mcrPage != nil {
			if !samePage(mcrPage, page) {
				t.Error("the marked content reference names another page")
			}
			return true
		}
		if elementPage := structureElement.Page(); !samePage(elementPage, page) {
			t.Error("the structure element names another page")
		}
		return true
	}
	return false
}

// samePage is PDPage.equals: two wrappers are the same page when they hold
// the same dictionary.
func samePage(left common.COSObjectable, right *pdmodel.PDPage) bool {
	if left == nil || right == nil {
		return false
	}
	return left.COSObject() == right.COSObject()
}
