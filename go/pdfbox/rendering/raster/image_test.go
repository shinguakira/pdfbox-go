package raster

// B1: fill, stroke, clip and transform over a solid paint.
//
// There is no Java to compare against here -- Graphics2D is the reference and
// it is not in the tree -- so what is asserted is what the operations mean:
// a filled rectangle covers exactly its own pixels, a stroke covers the band
// its width describes, a clip removes what falls outside it, and the transform
// moves all three.

import (
	goimagecolor "image/color"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/rendering"
)

// black and white are what a page of text is made of.
var (
	black = rendering.ColorPaint{Red: 0, Green: 0, Blue: 0, Alpha: 1}
	white = goimagecolor.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	inked = goimagecolor.RGBA{R: 0, G: 0, B: 0, A: 0xFF}
)

// at is the colour of one pixel of a backend.
func at(i *Image, x, y int) goimagecolor.RGBA {
	return i.dst.RGBAAt(x, y)
}

// wantColor fails unless the pixel is the colour asked for.
func wantColor(t *testing.T, i *Image, x, y int, want goimagecolor.RGBA, what string) {
	t.Helper()
	if got := at(i, x, y); got != want {
		t.Errorf("%s: (%d,%d) is %v, want %v", what, x, y, got, want)
	}
}

// rectangle is a Shape of the given box.
func rectangle(x, y, w, h float64) geom.Shape {
	return geom.NewRectangle2D(x, y, w, h)
}

// TestFillsARectangle is the first thing that should come out: a black box on
// a white page, with hard edges where the box ends.
func TestFillsARectangle(t *testing.T) {
	i := NewImage(20, 20, rendering.RGB)
	i.SetPaint(black)
	i.SetAntiAliasing(false)

	if err := i.Fill(rectangle(5, 5, 10, 10)); err != nil {
		t.Fatalf("Fill: %v", err)
	}

	wantColor(t, i, 10, 10, inked, "the middle of the box")
	wantColor(t, i, 5, 5, inked, "the first pixel of the box")
	wantColor(t, i, 14, 14, inked, "the last pixel of the box")
	wantColor(t, i, 4, 10, white, "the column before the box")
	wantColor(t, i, 15, 10, white, "the column after the box")
	wantColor(t, i, 10, 4, white, "the row above the box")
	wantColor(t, i, 10, 15, white, "the row below the box")
}

// TestFillHonoursTheTransform moves the same rectangle by the transform, which
// is how a page's flip and every `cm` reaches the paper.
func TestFillHonoursTheTransform(t *testing.T) {
	i := NewImage(20, 20, rendering.RGB)
	i.SetPaint(black)
	i.SetAntiAliasing(false)
	i.SetTransform(geom.NewAffineTransform(1, 0, 0, 1, 5, 5))

	if err := i.Fill(rectangle(0, 0, 5, 5)); err != nil {
		t.Fatalf("Fill: %v", err)
	}

	wantColor(t, i, 7, 7, inked, "inside the translated box")
	wantColor(t, i, 2, 2, white, "where the box would have been untranslated")
}

// TestFillIsClipped is `W n` followed by a fill: only the intersection is
// painted.
func TestFillIsClipped(t *testing.T) {
	i := NewImage(20, 20, rendering.RGB)
	i.SetPaint(black)
	i.SetAntiAliasing(false)
	i.SetClip(geom.NewAreaOfShape(rectangle(0, 0, 10, 20)))

	if err := i.Fill(rectangle(5, 5, 10, 10)); err != nil {
		t.Fatalf("Fill: %v", err)
	}

	wantColor(t, i, 7, 7, inked, "inside both the box and the clip")
	wantColor(t, i, 12, 7, white, "inside the box and outside the clip")
}

// TestStrokesALine is `S`: a horizontal line four units wide covers two rows
// either side of its centre and nothing else.
func TestStrokesALine(t *testing.T) {
	i := NewImage(20, 20, rendering.RGB)
	i.SetPaint(black)
	i.SetAntiAliasing(false)
	i.SetStroke(&rendering.Stroke{LineWidth: 4, MiterLimit: 10})

	path := geom.NewPathDouble()
	path.MoveTo(2, 10)
	path.LineTo(18, 10)
	if err := i.Draw(path); err != nil {
		t.Fatalf("Draw: %v", err)
	}

	wantColor(t, i, 10, 9, inked, "just above the centre of the line")
	wantColor(t, i, 10, 10, inked, "the centre of the line")
	wantColor(t, i, 10, 7, white, "above the line")
	wantColor(t, i, 10, 13, white, "below the line")
}

// TestAnInvisibleStrokeDrawsNothing is PDFBOX-5168: an all-zero dash array is
// drawn as nothing at all, and PageDrawer says so by the Invisible flag.
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

// TestFillHonoursTheWindingRule is `f` against `f*`: a square with a smaller
// square inside it, both wound the same way, is solid under the nonzero rule
// and has a hole under even-odd.
//
// rasterx's own scanner cannot answer this -- its SetWinding is a no-op -- and
// it is why the fills go through freetype's rasteriser instead.
func TestFillHonoursTheWindingRule(t *testing.T) {
	square := func(rule int) *geom.Path2D {
		path := geom.NewPathDouble()
		path.SetWindingRule(rule)
		path.MoveTo(2, 2)
		path.LineTo(18, 2)
		path.LineTo(18, 18)
		path.LineTo(2, 18)
		path.ClosePath()
		path.MoveTo(6, 6)
		path.LineTo(14, 6)
		path.LineTo(14, 14)
		path.LineTo(6, 14)
		path.ClosePath()
		return path
	}

	nonZero := NewImage(20, 20, rendering.RGB)
	nonZero.SetPaint(black)
	nonZero.SetAntiAliasing(false)
	if err := nonZero.Fill(square(geom.WindNonZero)); err != nil {
		t.Fatalf("Fill: %v", err)
	}
	wantColor(t, nonZero, 10, 10, inked, "the middle, wound nonzero")

	evenOdd := NewImage(20, 20, rendering.RGB)
	evenOdd.SetPaint(black)
	evenOdd.SetAntiAliasing(false)
	if err := evenOdd.Fill(square(geom.WindEvenOdd)); err != nil {
		t.Fatalf("Fill: %v", err)
	}
	wantColor(t, evenOdd, 10, 10, white, "the middle, wound even-odd")
	wantColor(t, evenOdd, 4, 10, inked, "the ring, wound even-odd")
}
