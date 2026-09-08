package multipdf_test

// The Splitter half of org.apache.pdfbox.multipdf.PDFMergerUtilityTest.
//
// Eight of its thirty cases test `Splitter` rather than `PDFMergerUtility`, and
// they live in this class because they share its structure-tree helpers --
// `checkForPageOrphans` and the two static tree flatteners, which are methods
// of `PDFMergerUtility`. Slice 7 ported `Splitter` and could not port these;
// this branch brings the flatteners, so they go in here beside the helpers.
//
// Seven are here. `testSplitWithPgEntryAtTheTop` reads `target/pdfs`, which
// the Maven build downloads.
//
// "These tests just verify the status quo. Changes should be checked visually
// with a PDF viewer that can display structural information." Every number is
// the Java's.

import (
	"bytes"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/multipdf"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/documentinterchange/logicalstructure"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/action"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/documentnavigation/destination"
)

// TestSplitWithStructureTree splits two pages out of a tagged document and
// checks what came with them.
func TestSplitWithStructureTree(t *testing.T) {
	doc := loadFixture(t, "PDFBOX-4417-001031.pdf")
	defer doc.Close()

	splitter := multipdf.NewSplitter()
	splitter.SetStartPage(1)
	splitter.SetEndPage(2)
	splitter.SetSplitAtPage(2)
	splitResult := split(t, splitter, doc, 1)
	dstDoc := splitResult[0]
	defer dstDoc.Close()

	if got := dstDoc.NumberOfPages(); got != 2 {
		t.Fatalf("the split document has %d pages, want 2", got)
	}
	checkForPageOrphans(t, dstDoc)
	// these tests just verify the status quo. Changes should be checked
	// visually with a PDF viewer that can display structural information.
	structureTreeRoot := structureTreeRootOf(t, dstDoc)
	if got := len(idTreeAsMap(t, structureTreeRoot.IDTree())); got != 126 {
		t.Errorf("the IDTree holds %d entries, want 126", got)
	}
	if got := len(numberTreeAsMap(t, structureTreeRoot)); got != 2 {
		t.Errorf("the ParentTree holds %d entries, want 2", got)
	}
	if got := structureTreeRoot.RoleMap().Size(); got != 6 {
		t.Errorf("the RoleMap holds %d entries, want 6", got)
	}
}

// TestSplitWithStructureTreeAndDestinations also checks that the link
// destinations were fixed: only the two that point into the split document
// still name a page.
func TestSplitWithStructureTreeAndDestinations(t *testing.T) {
	doc := loadFixture(t, "PDFBOX-5762-722238.pdf")
	defer doc.Close()

	splitter := multipdf.NewSplitter()
	splitter.SetStartPage(1)
	splitter.SetEndPage(2)
	splitter.SetSplitAtPage(2)
	splitResult := split(t, splitter, doc, 1)
	dstDoc := splitResult[0]
	defer dstDoc.Close()

	if got := dstDoc.NumberOfPages(); got != 2 {
		t.Fatalf("the split document has %d pages, want 2", got)
	}
	checkForPageOrphans(t, dstDoc)
	structureTreeRoot := structureTreeRootOf(t, dstDoc)
	if got := len(numberTreeAsMap(t, structureTreeRoot)); got != 7 {
		t.Errorf("the ParentTree holds %d entries, want 7", got)
	}
	if got := structureTreeRoot.RoleMap().Size(); got != 4 {
		t.Errorf("the RoleMap holds %d entries, want 4", got)
	}

	// check that destinations are fixed (only the two first point to the split doc)
	pageTree := dstDoc.Pages()
	wantIndexes := []int{0, 1, -1, -1, -1}
	pages := linkDestinationPages(t, dstDoc.Page(0), len(wantIndexes))
	for i, want := range wantIndexes {
		got := -1
		if pages[i] != nil {
			got = pageTree.IndexOf(pages[i])
		}
		if got != want {
			t.Errorf("link %d points at page index %d, want %d", i, got, want)
		}
	}
}

// TestSplitWithStructureTreeAndDestinationsAndRemovedAnnotations is
// PDFBOX-5929: "Check that orphan annotations are removed from the structure
// tree if annotations were removed from the pages (don't do that!)."
func TestSplitWithStructureTreeAndDestinationsAndRemovedAnnotations(t *testing.T) {
	doc := loadFixture(t, "PDFBOX-5762-722238.pdf")
	defer doc.Close()

	splitter := multipdf.NewSplitter()
	for page := range doc.Pages().All {
		page.SetAnnotations(nil)
	}
	splitter.SetStartPage(1)
	splitter.SetEndPage(2)
	splitter.SetSplitAtPage(2)
	splitResult := split(t, splitter, doc, 1)
	dstDoc := splitResult[0]
	defer dstDoc.Close()

	if got := dstDoc.NumberOfPages(); got != 2 {
		t.Fatalf("the split document has %d pages, want 2", got)
	}
	checkForPageOrphans(t, dstDoc)
}

// TestSinglePageSplit is PDFBOX-5792, "where a destination was outside a target
// document and hit an NPE in the next call of Splitter.fixDestinations()".
func TestSinglePageSplit(t *testing.T) {
	doc := loadFixture(t, "PDFBOX-5792-240045.pdf")
	defer doc.Close()

	splitter := multipdf.NewSplitter()
	splitter.SetSplitAtPage(1)
	splitResult := split(t, splitter, doc, 6)
	defer closeAll(splitResult)

	for i, dstDoc := range splitResult {
		if got := dstDoc.NumberOfPages(); got != 1 {
			t.Fatalf("split document %d has %d pages, want 1", i, got)
		}
		checkForPageOrphans(t, dstDoc)
		for ann := range dstDoc.Page(0).Annotations().All {
			link, isLink := ann.(*annotation.PDAnnotationLink)
			if !isLink {
				t.Fatalf("split document %d carries a %T, want a link", i, ann)
			}
			if page := linkDestinationPage(t, link); page != nil {
				t.Errorf("split document %d has a link that still names a page", i)
			}
		}
	}

	// The parent tree and role map of each of the six, in order.
	for i, want := range []struct{ parentTree, roleMap int }{
		{6, 3}, {6, 3}, {6, 4}, {5, 4}, {1, 6}, {1, 7},
	} {
		structureTreeRoot := structureTreeRootOf(t, splitResult[i])
		if got := len(numberTreeAsMap(t, structureTreeRoot)); got != want.parentTree {
			t.Errorf("split document %d has %d ParentTree entries, want %d",
				i, got, want.parentTree)
		}
		if got := structureTreeRoot.RoleMap().Size(); got != want.roleMap {
			t.Errorf("split document %d has %d RoleMap entries, want %d",
				i, got, want.roleMap)
		}
	}
}

// TestSplitWithPopupAnnotations checks that a text annotation and its popup
// come across still pointing at each other, and that the source is unchanged.
func TestSplitWithPopupAnnotations(t *testing.T) {
	doc := loadFixture(t, "PDFBOX-5809-509329.pdf")
	defer doc.Close()

	splitter := multipdf.NewSplitter()
	splitter.SetStartPage(3)
	splitter.SetEndPage(3)
	splitter.SetSplitAtPage(1)
	splitResult := split(t, splitter, doc, 1)

	dstDoc := splitResult[0]
	checkForPageOrphans(t, dstDoc)
	if got := dstDoc.NumberOfPages(); got != 1 {
		t.Fatalf("the split document has %d pages, want 1", got)
	}
	checkTextAndPopup(t, dstDoc.Page(0))
	dstDoc.Close()

	// Check that source document is ok
	checkTextAndPopup(t, doc.Page(2))
}

// checkTextAndPopup is the five lines the popup case makes twice: the fourth
// annotation is a text one, the fifth is its popup, and each names the other.
func checkTextAndPopup(t *testing.T, page *pdmodel.PDPage) {
	t.Helper()
	annotations := page.Annotations().ToSlice()
	if len(annotations) != 5 {
		t.Fatalf("the page carries %d annotations, want 5", len(annotations))
	}
	annotationText3, isText := annotations[3].(*annotation.PDAnnotationText)
	if !isText {
		t.Fatalf("annotation 3 is %T, want a text annotation", annotations[3])
	}
	annotationPopup4, isPopup := annotations[4].(*annotation.PDAnnotationPopup)
	if !isPopup {
		t.Fatalf("annotation 4 is %T, want a popup", annotations[4])
	}
	if got := annotationText3.Popup(); got == nil ||
		got.COSObject() != annotationPopup4.COSObject() {
		t.Error("the text annotation's /Popup is not the popup beside it")
	}
	if got := annotationPopup4.Parent(); got == nil ||
		got.COSObject() != annotationText3.COSObject() {
		t.Error("the popup's /Parent is not the text annotation beside it")
	}
	if got := annotationText3.Page(); got == nil || got.COSObject() != page.COSObject() {
		t.Error("the text annotation's /P is not the page it is on")
	}
}

// TestSplitWithBrokenDestination is PDFBOX-5811: the split document's link has
// no destination at all, and the source's still refuses to answer one.
func TestSplitWithBrokenDestination(t *testing.T) {
	doc := loadFixture(t, "PDFBOX-5811-362972.pdf")
	defer doc.Close()

	splitter := multipdf.NewSplitter()
	splitter.SetStartPage(2)
	splitter.SetEndPage(2)
	splitResult := split(t, splitter, doc, 1)

	dstDoc := splitResult[0]
	checkForPageOrphans(t, dstDoc)
	if got := dstDoc.NumberOfPages(); got != 1 {
		t.Fatalf("the split document has %d pages, want 1", got)
	}
	annotations := dstDoc.Page(0).Annotations().ToSlice()
	if len(annotations) != 1 {
		t.Fatalf("the split page carries %d annotations, want 1", len(annotations))
	}
	link, isLink := annotations[0].(*annotation.PDAnnotationLink)
	if !isLink {
		t.Fatalf("the annotation is %T, want a link", annotations[0])
	}
	dest, err := link.Destination()
	if err != nil {
		t.Fatalf("getDestination: %v", err)
	}
	if dest != nil {
		t.Errorf("the split link has the destination %v, want none", dest)
	}
	dstDoc.Close()

	// Check source document
	annotations = doc.Page(1).Annotations().ToSlice()
	if len(annotations) != 1 {
		t.Fatalf("the source page carries %d annotations, want 1", len(annotations))
	}
	sourceLink, isLink := annotations[0].(*annotation.PDAnnotationLink)
	if !isLink {
		t.Fatalf("the annotation is %T, want a link", annotations[0])
	}
	if _, err := sourceLink.Destination(); err == nil {
		t.Error("the source link answered a destination; the Java throws an IOException")
	}
}

// TestSplitWithNamedDestinations is PDFBOX-5840: a named destination is turned
// into a page one, and the source keeps its name.
func TestSplitWithNamedDestinations(t *testing.T) {
	doc := loadFixture(t, "PDFBOX-5840-410609.pdf")
	defer doc.Close()

	splitter := multipdf.NewSplitter()
	splitter.SetSplitAtPage(6)
	splitResult := split(t, splitter, doc, 1)

	dstDoc := splitResult[0]
	checkForPageOrphans(t, dstDoc)
	if got := dstDoc.NumberOfPages(); got != 6 {
		t.Fatalf("the split document has %d pages, want 6", got)
	}

	pageTree := dstDoc.Pages()
	wantIndexes := []int{0, 1, 3, 3, 5}
	pages := linkDestinationPages(t, dstDoc.Page(0), len(wantIndexes))
	for i, want := range wantIndexes {
		got := -1
		if pages[i] != nil {
			got = pageTree.IndexOf(pages[i])
		}
		if got != want {
			t.Errorf("link %d points at page index %d, want %d", i, got, want)
		}
	}

	if dstDoc.DocumentCatalog().Metadata() == nil {
		t.Error("the split document lost its /Metadata")
	}
	var saved bytes.Buffer
	if err := dstDoc.Save(&saved); err != nil {
		t.Fatal(err)
	}
	reloadedDoc, err := pdfbox.LoadPDFBytes(saved.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if reloadedDoc.DocumentCatalog().Metadata() == nil {
		t.Error("the reloaded document lost its /Metadata")
	}
	reloadedDoc.Close()
	dstDoc.Close()

	// Check that source document is unchanged
	annotations := doc.Page(0).Annotations().ToSlice()
	if len(annotations) != 5 {
		t.Fatalf("the source page carries %d annotations, want 5", len(annotations))
	}
	link, isLink := annotations[0].(*annotation.PDAnnotationLink)
	if !isLink {
		t.Fatalf("the annotation is %T, want a link", annotations[0])
	}
	goTo, isGoTo := link.Action().(*action.PDActionGoTo)
	if !isGoTo {
		t.Fatalf("the link's action is %T, want a GoTo", link.Action())
	}
	dest, err := goTo.Destination()
	if err != nil {
		t.Fatalf("getDestination: %v", err)
	}
	if _, isNamed := dest.(*destination.PDNamedDestination); !isNamed {
		t.Errorf("the source link's destination is %T, want a named one", dest)
	}
}

// TestSplitWithOrphanPopupAnnotation is PDFBOX-6018: two text annotations whose
// popups are not on any page still come across pointing at each other.
func TestSplitWithOrphanPopupAnnotation(t *testing.T) {
	doc := loadFixture(t, "PDFBOX-6018-099267-p9-OrphanPopups.pdf")
	defer doc.Close()

	splitResult := split(t, multipdf.NewSplitter(), doc, 1)
	dstDoc := splitResult[0]
	defer dstDoc.Close()

	if got := dstDoc.NumberOfPages(); got != 1 {
		t.Fatalf("the split document has %d pages, want 1", got)
	}
	page := dstDoc.Page(0)
	annotations := page.Annotations().ToSlice()
	if len(annotations) != 2 {
		t.Fatalf("the page carries %d annotations, want 2", len(annotations))
	}
	for i, ann := range annotations {
		text, isText := ann.(*annotation.PDAnnotationText)
		if !isText {
			t.Fatalf("annotation %d is %T, want a text annotation", i, ann)
		}
		if got := text.Page(); got == nil || got.COSObject() != page.COSObject() {
			t.Errorf("annotation %d's /P is not the page it is on", i)
		}
		popup := text.Popup()
		if popup == nil {
			t.Fatalf("annotation %d has no popup", i)
		}
		if parent := popup.Parent(); parent == nil ||
			parent.COSObject() != text.COSObject() {
			t.Errorf("annotation %d's popup does not point back at it", i)
		}
	}
}

// split runs a splitter and asserts how many documents came out.
func split(t *testing.T, splitter *multipdf.Splitter, doc *pdmodel.PDDocument,
	want int) []*pdmodel.PDDocument {
	t.Helper()
	splitResult, err := splitter.Split(doc)
	if err != nil {
		t.Fatalf("split: %v", err)
	}
	if len(splitResult) != want {
		t.Fatalf("the document split into %d, want %d", len(splitResult), want)
	}
	return splitResult
}

func closeAll(documents []*pdmodel.PDDocument) {
	for _, doc := range documents {
		doc.Close()
	}
}

// numberTreeAsMap is PDFMergerUtility.getNumberTreeAsMap over a structure
// tree's /ParentTree, which the Java test calls directly.
func numberTreeAsMap(t *testing.T,
	root *logicalstructure.PDStructureTreeRoot) map[int]common.COSObjectable {
	t.Helper()
	numbers, err := multipdf.GetNumberTreeAsMap(root.ParentTree())
	if err != nil {
		t.Fatalf("getNumberTreeAsMap: %v", err)
	}
	return numbers
}

// linkDestinationPages answers the page each of a page's first `count` link
// annotations names, or nil where it names none.
func linkDestinationPages(t *testing.T, page *pdmodel.PDPage, count int) []*pdmodel.PDPage {
	t.Helper()
	annotations := page.Annotations().ToSlice()
	if len(annotations) != count {
		t.Fatalf("the page carries %d annotations, want %d", len(annotations), count)
	}
	pages := make([]*pdmodel.PDPage, count)
	for i, ann := range annotations {
		link, isLink := ann.(*annotation.PDAnnotationLink)
		if !isLink {
			t.Fatalf("annotation %d is %T, want a link", i, ann)
		}
		pages[i] = linkDestinationPage(t, link)
	}
	return pages
}

// linkDestinationPage answers the page a link's GoTo action names, or nil.
func linkDestinationPage(t *testing.T, link *annotation.PDAnnotationLink) *pdmodel.PDPage {
	t.Helper()
	goTo, isGoTo := link.Action().(*action.PDActionGoTo)
	if !isGoTo {
		t.Fatalf("the link's action is %T, want a GoTo", link.Action())
	}
	dest, err := goTo.Destination()
	if err != nil {
		t.Fatalf("getDestination: %v", err)
	}
	pageDestination, isPageDestination := dest.(destination.PageDestination)
	if !isPageDestination {
		t.Fatalf("the destination is %T, want a page destination", dest)
	}
	page, isPage := pageDestination.Page().(*pdmodel.PDPage)
	if !isPage {
		return nil
	}
	return page
}
