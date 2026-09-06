package pdmodel_test

// Ported from
// pdfbox/src/test/java/org/apache/pdfbox/pdmodel/TestPDPageAnnotationsFiltering.java
// and TestPDPageTransitions.java. Both name types from pdmodel/interactive, and
// the transitions test loads a PDF through the Loader, so the tests are in
// pdmodel_test.

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/pagenavigation"
)

// mockedPageWithAnnotations is the @BeforeEach initMock of
// TestPDPageAnnotationsFiltering: a page carrying a rubber stamp, a square and
// a link, in that order.
func mockedPageWithAnnotations() *pdmodel.PDPage {
	page := cos.NewDictionary()
	annotsDictionary := cos.NewArray()
	annotsDictionary.Add(annotation.NewPDAnnotationRubberStamp().COSObject())
	annotsDictionary.Add(annotation.NewPDAnnotationSquare().COSObject())
	annotsDictionary.Add(annotation.NewPDAnnotationLink().COSObject())
	page.SetItem(cos.Annots, annotsDictionary)
	return pdmodel.NewPDPageOf(page)
}

// TestValidateNoFiltering is TestPDPageAnnotationsFiltering.validateNoFiltering.
func TestValidateNoFiltering(t *testing.T) {
	annotations := mockedPageWithAnnotations().Annotations()
	if got := annotations.Size(); got != 3 {
		t.Fatalf("Annotations() = %d, want 3", got)
	}
	if _, ok := annotations.Get(0).(*annotation.PDAnnotationRubberStamp); !ok {
		t.Errorf("Annotations()[0] = %T, want *PDAnnotationRubberStamp", annotations.Get(0))
	}
	if _, ok := annotations.Get(1).(*annotation.PDAnnotationSquare); !ok {
		t.Errorf("Annotations()[1] = %T, want *PDAnnotationSquare", annotations.Get(1))
	}
	if _, ok := annotations.Get(2).(*annotation.PDAnnotationLink); !ok {
		t.Errorf("Annotations()[2] = %T, want *PDAnnotationLink", annotations.Get(2))
	}
}

// TestValidateAllFiltered is TestPDPageAnnotationsFiltering.validateAllFiltered.
func TestValidateAllFiltered(t *testing.T) {
	annotations := mockedPageWithAnnotations().AnnotationsOfFilter(
		func(annotation.PDAnnotation) bool { return false })
	if got := annotations.Size(); got != 0 {
		t.Errorf("AnnotationsOfFilter(false) = %d, want 0", got)
	}
}

// TestValidateSelectedFew is TestPDPageAnnotationsFiltering.validateSelectedFew.
func TestValidateSelectedFew(t *testing.T) {
	annotations := mockedPageWithAnnotations().AnnotationsOfFilter(
		func(a annotation.PDAnnotation) bool {
			switch a.(type) {
			case *annotation.PDAnnotationLink, *annotation.PDAnnotationSquare:
				return true
			}
			return false
		})
	if got := annotations.Size(); got != 2 {
		t.Fatalf("AnnotationsOfFilter(link or square) = %d, want 2", got)
	}
	if _, ok := annotations.Get(0).(*annotation.PDAnnotationSquare); !ok {
		t.Errorf("filtered[0] = %T, want *PDAnnotationSquare", annotations.Get(0))
	}
	if _, ok := annotations.Get(1).(*annotation.PDAnnotationLink); !ok {
		t.Errorf("filtered[1] = %T, want *PDAnnotationLink", annotations.Get(1))
	}
}

// TestReadTransitions is TestPDPageTransitions.readTransitions.
func TestReadTransitions(t *testing.T) {
	doc, err := pdfbox.LoadPDF(filepath.Join(catalogFixture,
		"interactive/pagenavigation/transitions_test.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	defer doc.Close()
	firstTransition := doc.Pages().Get(0).Transition()
	if firstTransition == nil {
		t.Fatal("Transition() = nil, want the first transition")
	}
	if got, want := firstTransition.Style(),
		string(pagenavigation.TransitionStyleGlitter); got != want {
		t.Errorf("Style() = %q, want %q", got, want)
	}
	if got := firstTransition.Duration(); got != 2 {
		t.Errorf("Duration() = %v, want 2", got)
	}
	if got, want := firstTransition.Direction(),
		pagenavigation.TransitionDirectionTopLeftToBottomRight.COSBase(); !cosEquals(got, want) {
		t.Errorf("Direction() = %v, want %v", got, want)
	}
}

// TestSaveAndReadTransitions is TestPDPageTransitions.saveAndReadTransitions.
func TestSaveAndReadTransitions(t *testing.T) {
	baos := &bytes.Buffer{}

	// save
	func() {
		document := pdmodel.NewPDDocument()
		defer document.Close()
		page := pdmodel.NewPDPage()
		document.AddPage(page)
		transition := pagenavigation.NewPDTransitionOfStyle(pagenavigation.TransitionStyleFly)
		transition.SetDirection(pagenavigation.TransitionDirectionNone)
		transition.SetFlyScale(0.5)
		page.SetTransitionOfDuration(transition, 2)
		if err := document.Save(baos); err != nil {
			t.Fatal(err)
		}
	}()

	// read
	doc, err := pdfbox.LoadPDFBytes(baos.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	defer doc.Close()
	page := doc.Pages().Get(0)
	loadedTransition := page.Transition()
	if loadedTransition == nil {
		t.Fatal("Transition() = nil, want the transition that was saved")
	}
	if got, want := loadedTransition.Style(),
		string(pagenavigation.TransitionStyleFly); got != want {
		t.Errorf("Style() = %q, want %q", got, want)
	}
	if got := page.Dictionary().GetFloat(cos.Dur, 0); got != 2 {
		t.Errorf("/Dur = %v, want 2", got)
	}
	if got, want := loadedTransition.Direction(),
		pagenavigation.TransitionDirectionNone.COSBase(); !cosEquals(got, want) {
		t.Errorf("Direction() = %v, want %v", got, want)
	}
}

// cosEquals is Java's assertEquals over two COSBase, which calls equals; the
// port's Equals is typed on the receiver's own class, so this dispatches on
// what the two values are. A name is interned, so pointer equality is
// COSName.equals; an integer is not.
func cosEquals(got, want cos.Base) bool {
	switch value := got.(type) {
	case *cos.Integer:
		other, isInteger := want.(*cos.Integer)
		return isInteger && value.Equals(other)
	case *cos.Name:
		other, isName := want.(*cos.Name)
		return isName && value.Equals(other)
	}
	return got == want
}
