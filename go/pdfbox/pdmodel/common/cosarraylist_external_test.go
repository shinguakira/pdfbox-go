package common_test

// Ported from
// pdfbox/src/test/java/org/apache/pdfbox/pdmodel/common/COSArrayListTest.java,
// which slice 2 could not port because it needs annotations.
//
// The package is common_test rather than common: the whole test is built out of
// annotations on a page, and both sit above this one.
//
// addToList is not here: Java has its @Test commented out.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
)

// arrayListFixture is the setUp of COSArrayListTest.
type arrayListFixture struct {
	// tbcAnnotationsList and tbcAnnotationsArray are to be used for comparison
	// with COSArrayList behaviour in order to ensure that the intended object is
	// now at the correct position. They will also be used for Collection/Array
	// based setting and comparison.
	tbcAnnotationsList  []annotation.PDAnnotation
	tbcAnnotationsArray []cos.Base

	// annotationsList and annotationsArray are to be used within COSArrayList.
	annotationsList  []annotation.PDAnnotation
	annotationsArray *cos.Array

	// pdPage is to be used when testing retrieving filtered items as can be
	// done with PDPage.getAnnotations(AnnotationFilter annotationFilter).
	pdPage *pdmodel.PDPage
}

// newArrayListFixture is COSArrayListTest.setUp.
func newArrayListFixture(t *testing.T) *arrayListFixture {
	t.Helper()
	f := &arrayListFixture{}
	txtMark := annotation.NewPDAnnotationHighlight()
	txtLink := annotation.NewPDAnnotationLink()
	aCircle := annotation.NewPDAnnotationCircle()

	f.annotationsList = []annotation.PDAnnotation{txtMark, txtLink, aCircle, txtLink}
	assertSize(t, "annotationsList", len(f.annotationsList), 4)

	f.tbcAnnotationsList = []annotation.PDAnnotation{txtMark, txtLink, aCircle, txtLink}
	assertSize(t, "tbcAnnotationsList", len(f.tbcAnnotationsList), 4)

	f.annotationsArray = cos.NewArray()
	for _, annot := range []annotation.PDAnnotation{txtMark, txtLink, aCircle, txtLink} {
		f.annotationsArray.Add(annot.COSObject())
	}
	assertSize(t, "annotationsArray", f.annotationsArray.Size(), 4)

	f.tbcAnnotationsArray = []cos.Base{
		txtMark.COSObject(), txtLink.COSObject(), aCircle.COSObject(), txtLink.COSObject(),
	}
	assertSize(t, "tbcAnnotationsArray", len(f.tbcAnnotationsArray), 4)

	// add the annotations to the page
	f.pdPage = pdmodel.NewPDPage()
	f.pdPage.SetAnnotations(f.annotationsList)
	return f
}

// list returns the COSArrayList the tests operate on.
func (f *arrayListFixture) list() *common.COSArrayList[annotation.PDAnnotation] {
	return common.NewCOSArrayListOf(f.annotationsList, f.annotationsArray)
}

// assertSize is the assertEquals of Java over a collection size.
func assertSize(t *testing.T, what string, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("%s size = %d, want %d", what, got, want)
	}
}

// TestGetFromList is COSArrayListTest.getFromList.
func TestGetFromList(t *testing.T) {
	f := newArrayListFixture(t)
	cosArrayList := f.list()
	for i := 0; i < cosArrayList.Size(); i++ {
		annot := cosArrayList.Get(i)
		if got, want := annot.COSObject(), f.annotationsArray.Get(i); got != want {
			t.Errorf("PDAnnotations cosObject at %d shall be equal to index %d of COSArray", i, i)
		}
		// compare with Java List/Array
		if got, want := annot, f.tbcAnnotationsList[i]; got != want {
			t.Errorf("PDAnnotations at %d shall be at index %d of List", i, i)
		}
		if got, want := annot.COSObject(), f.tbcAnnotationsArray[i]; got != want {
			t.Errorf("PDAnnotations cosObject at %d shall be at position %d of Array", i, i)
		}
	}
}

// TestRemoveFromListByIndex is COSArrayListTest.removeFromListByIndex.
func TestRemoveFromListByIndex(t *testing.T) {
	f := newArrayListFixture(t)
	cosArrayList := f.list()
	const positionToRemove = 2
	toBeRemoved := cosArrayList.Get(positionToRemove)
	if got := cosArrayList.RemoveAt(positionToRemove); got != toBeRemoved {
		t.Error("Remove operation shall return the removed object")
	}
	assertSize(t, "List", cosArrayList.Size(), 3)
	assertSize(t, "COSArray", f.annotationsArray.Size(), 3)
	if got := cosArrayList.IndexOf(f.tbcAnnotationsList[positionToRemove]); got != -1 {
		t.Errorf("PDAnnotation shall no longer exist in List: %d", got)
	}
	if got := f.annotationsArray.IndexOf(f.tbcAnnotationsArray[positionToRemove]); got != -1 {
		t.Errorf("COSObject shall no longer exist in COSArray: %d", got)
	}
}

// TestRemoveUniqueFromListByObject is
// COSArrayListTest.removeUniqueFromListByObject.
func TestRemoveUniqueFromListByObject(t *testing.T) {
	f := newArrayListFixture(t)
	cosArrayList := f.list()
	const positionToRemove = 2
	toBeRemoved := f.annotationsList[positionToRemove]
	if !cosArrayList.Remove(toBeRemoved) {
		t.Error("Remove operation shall return true")
	}
	assertSize(t, "List", cosArrayList.Size(), 3)
	assertSize(t, "COSArray", f.annotationsArray.Size(), 3)

	// compare with Java List/Array to ensure correct object at position
	if got, want := cosArrayList.Get(2), f.tbcAnnotationsList[3]; got != want {
		t.Error("List object at 3 is at position 2 in COSArrayList now")
	}
	if got, want := f.annotationsArray.Get(2), f.tbcAnnotationsList[3].COSObject(); got != want {
		t.Error("COSObject of List object at 3 is at position 2 in COSArray now")
	}
	if got, want := f.annotationsArray.Get(2), f.tbcAnnotationsArray[3]; got != want {
		t.Error("Array object at 3 is at position 2 in underlying COSArray now")
	}
}

// TestRemoveAllUniqueFromListByObject is
// COSArrayListTest.removeAllUniqueFromListByObject.
func TestRemoveAllUniqueFromListByObject(t *testing.T) {
	f := newArrayListFixture(t)
	cosArrayList := f.list()
	const positionToRemove = 2
	toBeRemovedInstances := []annotation.PDAnnotation{f.annotationsList[positionToRemove]}
	if !cosArrayList.RemoveAll(toBeRemovedInstances) {
		t.Error("Remove operation shall return true")
	}
	assertSize(t, "List", cosArrayList.Size(), 3)
	assertSize(t, "COSArray", f.annotationsArray.Size(), 3)
	if cosArrayList.RemoveAll(toBeRemovedInstances) {
		t.Error("Remove shall not remove any object")
	}
}

// TestRemoveMultipleFromListByObject is
// COSArrayListTest.removeMultipleFromListByObject.
func TestRemoveMultipleFromListByObject(t *testing.T) {
	f := newArrayListFixture(t)
	cosArrayList := f.list()
	const positionToRemove = 1
	toBeRemoved := f.tbcAnnotationsList[positionToRemove]
	if !cosArrayList.Remove(toBeRemoved) {
		t.Error("Remove operation shall return true")
	}
	assertSize(t, "List", cosArrayList.Size(), 3)
	assertSize(t, "COSArray", f.annotationsArray.Size(), 3)
	if !cosArrayList.Remove(toBeRemoved) {
		t.Error("Remove operation shall return true")
	}
	assertSize(t, "List", cosArrayList.Size(), 2)
	assertSize(t, "COSArray", f.annotationsArray.Size(), 2)
}

// TestRemoveAllMultipleFromListByObject is
// COSArrayListTest.removeAllMultipleFromListByObject.
func TestRemoveAllMultipleFromListByObject(t *testing.T) {
	f := newArrayListFixture(t)
	cosArrayList := f.list()
	const positionToRemove = 1
	toBeRemovedInstances := []annotation.PDAnnotation{f.annotationsList[positionToRemove]}
	if !cosArrayList.RemoveAll(toBeRemovedInstances) {
		t.Error("Remove operation shall return true")
	}
	assertSize(t, "List", cosArrayList.Size(), 2)
	assertSize(t, "COSArray", f.annotationsArray.Size(), 2)
	if cosArrayList.RemoveAll(toBeRemovedInstances) {
		t.Error("Remove shall not remove any object")
	}
}

// notLink is the AnnotationFilter the two filtered tests use: retrieve all
// annotations from page but the link annotation, which is 2nd in the list.
func notLink(annot annotation.PDAnnotation) bool {
	_, isLink := annot.(*annotation.PDAnnotationLink)
	return !isLink
}

// TestRemoveFromFilteredListByIndex is
// COSArrayListTest.removeFromFilteredListByIndex. Java asserts
// UnsupportedOperationException, which is unchecked, so the port panics.
func TestRemoveFromFilteredListByIndex(t *testing.T) {
	f := newArrayListFixture(t)
	cosArrayList := f.pdPage.AnnotationsOfFilter(notLink)
	defer func() {
		if recover() == nil {
			t.Error("RemoveAt() did not panic")
		}
	}()
	cosArrayList.RemoveAt(1) // this call should fail
}

// TestRemoveFromFilteredListByObject is
// COSArrayListTest.removeFromFilteredListByObject.
func TestRemoveFromFilteredListByObject(t *testing.T) {
	f := newArrayListFixture(t)
	cosArrayList := f.pdPage.AnnotationsOfFilter(notLink)
	// remove object
	toBeRemoved := cosArrayList.Get(1)
	defer func() {
		if recover() == nil {
			t.Error("Remove() did not panic")
		}
	}()
	cosArrayList.Remove(toBeRemoved) // this call should fail
}

// writeFourAnnotations is the "generate test file" half the last three tests
// share, and answers the path it wrote to.
func writeFourAnnotations(t *testing.T, name string, direct bool) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	pdf := pdmodel.NewPDDocument()
	defer pdf.Close()
	page := pdmodel.NewPDPage()
	pdf.AddPage(page)
	txtMark := annotation.NewPDAnnotationHighlight()
	txtLink := annotation.NewPDAnnotationLink()
	if direct {
		// enforce the COSDictionaries to be written directly into the COSArray
		txtMark.AnnotationDictionary().SetDirect(true)
		txtLink.AnnotationDictionary().SetDirect(true)
	}
	pageAnnots := []annotation.PDAnnotation{txtMark, txtMark, txtMark, txtLink}
	assertSize(t, "There shall be 4 annotations generated", len(pageAnnots), 4)
	page.SetAnnotations(pageAnnots)

	out, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	if err := pdf.Save(out); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestRemoveSingleDirectObject is COSArrayListTest.removeSingleDirectObject.
func TestRemoveSingleDirectObject(t *testing.T) {
	path := writeFourAnnotations(t, "removeSingleDirectObjectTest.pdf", true)
	assertRemovesOne(t, path)
}

// TestRemoveSingleIndirectObject is COSArrayListTest.removeSingleIndirectObject.
func TestRemoveSingleIndirectObject(t *testing.T) {
	path := writeFourAnnotations(t, "removeSingleIndirectObjectTest.pdf", false)
	assertRemovesOne(t, path)
}

// assertRemovesOne is the reading half the two removal tests share.
func assertRemovesOne(t *testing.T, path string) {
	t.Helper()
	pdf, err := pdfbox.LoadPDF(path)
	if err != nil {
		t.Fatal(err)
	}
	defer pdf.Close()
	page := pdf.Page(0)
	annotations := page.Annotations()
	assertSize(t, "There shall be 4 annotations retrieved", annotations.Size(), 4)
	assertSize(t, "The size of the internal COSArray shall be 4",
		annotations.ToList().Size(), 4)
	toBeRemoved := annotations.Get(0)
	annotations.Remove(toBeRemoved)
	assertSize(t, "There shall be 3 annotations left", annotations.Size(), 3)
	assertSize(t, "The size of the internal COSArray shall be 3",
		annotations.ToList().Size(), 3)
}

// TestRetainIndirectObject is COSArrayListTest.retainIndirectObject.
func TestRetainIndirectObject(t *testing.T) {
	path := writeFourAnnotations(t, "removeIndirectObjectTest.pdf", false)
	pdf, err := pdfbox.LoadPDF(path)
	if err != nil {
		t.Fatal(err)
	}
	defer pdf.Close()
	page := pdf.Page(0)
	annotations := page.Annotations()
	assertSize(t, "There shall be 4 annotations retrieved", annotations.Size(), 4)
	assertSize(t, "The size of the internal COSArray shall be 4",
		annotations.ToList().Size(), 4)
	toBeRetained := []annotation.PDAnnotation{annotations.Get(0)}
	annotations.RetainAll(toBeRetained)
	assertSize(t, "There shall be 3 annotations left", annotations.Size(), 3)
	assertSize(t, "The size of the internal COSArray shall be 3",
		annotations.ToList().Size(), 3)
}
