package raster

// What the backend does that Graphics2D does not.
//
// The shapes it draws are held to Java2D's own pixels in java2d_test.go and
// the compositing to PDFBox's own in blend_test.go, which is where a
// comparison belongs when there is something to compare against. What is left
// here is the two places where there is not: a rule PDFBox adds on top of
// Graphics2D, and a property of transparency groups that no single Java call
// answers.

import (
	goimagecolor "image/color"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/rendering"
)

// black is what a page of text is made of; white is image.go's, the ground
// under a blit.
var black = rendering.ColorPaint{Red: 0, Green: 0, Blue: 0, Alpha: 1}

// at is the colour of one pixel of a backend.
func at(i *Image, x, y int) goimagecolor.NRGBA {
	return i.dst.NRGBAAt(x, y)
}

// wantColor fails unless the pixel is the colour asked for.
func wantColor(t *testing.T, i *Image, x, y int, want goimagecolor.NRGBA, what string) {
	t.Helper()
	if got := at(i, x, y); got != want {
		t.Errorf("%s: (%d,%d) is %v, want %v", what, x, y, got, want)
	}
}

// rectangle is a Shape of the given box.
func rectangle(x, y, w, h float64) geom.Shape {
	return geom.NewRectangle2D(x, y, w, h)
}

// TestAnInvisibleStrokeDrawsNothing is PDFBOX-5168: an all-zero dash array is
// drawn as nothing at all, and PageDrawer says so by the Invisible flag.
//
// There is nothing to compare this against, because Java2D has no such rule.
// A BasicStroke cannot even hold an all-zero dash array -- its constructor
// throws -- so PageDrawer catches the case before it builds one, and what it
// does instead is skip the draw. This is that skip.
func TestAnInvisibleStrokeDrawsNothing(t *testing.T) {
	i := NewImage(20, 20, rendering.RGB)
	i.SetPaint(black)
	i.SetStroke(&rendering.Stroke{Invisible: true})

	path := geom.NewPathDouble()
	path.MoveTo(2, 10)
	path.LineTo(18, 10)
	if err := i.Draw(path); err != nil {
		t.Fatalf("Draw: %v", err)
	}

	wantColor(t, i, 10, 10, white, "where an invisible stroke would have gone")
}

// TestATransparencyGroupIsCompositedAsAWhole is what a group is for: the alpha
// constant applies to the group and not to each thing inside it, so two
// overlapping opaque fills inside a half-transparent group do not show through
// each other.
//
// This is a property and not a pixel, deliberately. The alpha of a group is
// applied by PageDrawer.TransparencyGroup and GroupGraphics together, over a
// content stream, and there is no Java call that composites a group the way
// one call composites a blend mode -- so there is nothing to run and read a
// number off. What the property catches is the mistake that matters: applying
// the group's alpha to each fill as it is made, which would leave the overlap
// darker than the rest.
func TestATransparencyGroupIsCompositedAsAWhole(t *testing.T) {
	i := NewImage(20, 20, rendering.RGB)
	i.SetAntiAliasing(false)
	i.SetComposite(nil, 0.5)

	if err := i.PushGroup(nil, false, false, nil); err != nil {
		t.Fatalf("PushGroup: %v", err)
	}
	i.SetPaint(black)
	if err := i.Fill(rectangle(2, 2, 10, 10)); err != nil {
		t.Fatalf("Fill: %v", err)
	}
	if err := i.Fill(rectangle(6, 6, 10, 10)); err != nil {
		t.Fatalf("Fill: %v", err)
	}
	if err := i.PopGroup(); err != nil {
		t.Fatalf("PopGroup: %v", err)
	}

	// Where the two fills overlap the group is still one layer of black at
	// half alpha, the same as where only one of them drew.
	single := at(i, 4, 4)
	overlap := at(i, 8, 8)
	if single != overlap {
		t.Errorf("the overlap is %v and a single fill is %v; the group's alpha "+
			"was applied twice", overlap, single)
	}
	if single.R == 0 || single.R == 0xFF {
		t.Errorf("the group came out %v, want a half-way grey", single)
	}
}
