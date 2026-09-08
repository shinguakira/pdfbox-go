package annotation_test

// PDAnnotationPopup.getParent() casts to PDAnnotationMarkup, which every markup
// annotation is: a text annotation, a highlight, a square. The port narrows
// with a Go type assertion, which is not a cast -- `*PDAnnotationText` is not
// `*PDAnnotationMarkup`, it embeds one -- so the assertion has to be on what
// the embedding promotes.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
)

// TestPopupParentIsASubclassOfMarkup is the cast: a popup whose /Parent is a
// text annotation answers that annotation.
func TestPopupParentIsASubclassOfMarkup(t *testing.T) {
	text := annotation.NewPDAnnotationText()
	popup := annotation.NewPDAnnotationPopup()
	popup.AnnotationDictionary().SetItem(cos.Parent, text.COSObject())

	parent := popup.Parent()
	if parent == nil {
		t.Fatal("the popup answered no parent; its /Parent is a text annotation, " +
			"which in Java is a PDAnnotationMarkup")
	}
	if parent.COSObject() != text.COSObject() {
		t.Error("the popup's parent is not the annotation its /Parent names")
	}
}
