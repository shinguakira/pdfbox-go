package form

// PDFBOX-5660, Apache ced684bba, in the sync of 2026-09-07.
//
// computeBBox read the widget's /Rect and used it without a check:
//
//	PDRectangle rect = widget.getRectangle();
//	Matrix matrix = Matrix.getRotateInstance(...);
//	Point2D.Float point2D = matrix.transformPoint(rect.getWidth(), rect.getHeight());
//
// getRectangle answers null when /Rect is absent or is not an array of four,
// so a widget without one threw NullPointerException. Apache gave the method
// `throws IOException` and a check.
//
// It is not reachable through the one caller: setAppearanceValue reads
// getRectangle itself twenty lines earlier and skips the widget when it is
// null, and nothing between the two reads touches /Rect. The guard is there
// for the next caller, and the port takes it for the same reason -- the
// alternative in Go is a nil dereference in a package whose whole surface
// reports errors.

import (
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
)

// TestComputeBBoxOfAWidgetWithNoRectangle is the site.
func TestComputeBBoxOfAWidgetWithNoRectangle(t *testing.T) {
	widget := annotation.NewPDAnnotationWidget()
	widget.AnnotationDictionary().RemoveItem(cos.Rect)
	if widget.Rectangle() != nil {
		t.Fatal("the widget still has a rectangle; the case needs one with none")
	}

	box, err := computeBBox(widget, 0)
	if err == nil {
		t.Fatalf("computeBBox of a widget with no rectangle answered %v, want an error", box)
	}
	// Java's message.
	if !strings.Contains(err.Error(), "Missing rectangle") {
		t.Errorf("err = %v, want one saying Missing rectangle", err)
	}
}

// TestComputeBBoxTurnsTheRectangle is the guard beside it: the check must not
// change what the method answers for a widget that has a rectangle. A quarter
// turn swaps the two sides, which is the whole point of the method.
func TestComputeBBoxTurnsTheRectangle(t *testing.T) {
	widget := annotation.NewPDAnnotationWidget()
	widget.SetRectangle(common.NewPDRectangleOf(10, 20, 30, 40))

	for _, c := range []struct {
		rotation              int
		wantWidth, wantHeight float32
	}{
		{0, 30, 40},
		{90, 40, 30},
		{180, 30, 40},
		{270, 40, 30},
	} {
		box, err := computeBBox(widget, c.rotation)
		if err != nil {
			t.Fatalf("computeBBox at %d: %v", c.rotation, err)
		}
		if box.Width() != c.wantWidth || box.Height() != c.wantHeight {
			t.Errorf("computeBBox at %d = %vx%v, want %vx%v",
				c.rotation, box.Width(), box.Height(), c.wantWidth, c.wantHeight)
		}
	}
}
