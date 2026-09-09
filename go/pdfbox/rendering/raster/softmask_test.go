package raster

// The parts of SoftMask and of the backend the rendered pages cannot reach.
//
// `page_test.go` renders four soft masks against PDFBox and two of them are
// exact, which covers the group, its box, its origin, the backdrop fill and
// both channel readings. What it does not reach is the /TR transfer function,
// which no page there carries, and DrawSurface, which only PDFPrintable calls.

import (
	goimage "image"
	goimagecolor "image/color"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common/function"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/rendering"
)

// invertingFunction is a type 2 exponential from 1 to 0, so that a grey of g
// comes back as 1 - g.
func invertingFunction(t *testing.T) function.PDFunction {
	t.Helper()
	dictionary := cos.NewDictionary()
	dictionary.SetInt(cos.FunctionType, 2)
	dictionary.SetItem(cos.Domain, floatArray([]float32{0, 1}))
	dictionary.SetItem(cos.C0, floatArray([]float32{1}))
	dictionary.SetItem(cos.C1, floatArray([]float32{0}))
	dictionary.SetInt(cos.N, 1)

	f, err := function.NewPDFunction(dictionary)
	if err != nil {
		t.Fatalf("building the transfer function: %v", err)
	}
	return f
}

// maskOfGreys is a soft mask over a strip of the given grey values, at the
// origin, with no backdrop.
func maskOfGreys(under paintSource, greys []uint8, transfer function.PDFunction) *softMaskSource {
	mask := goimage.NewAlpha(goimage.Rect(0, 0, len(greys), 1))
	for x, grey := range greys {
		mask.SetAlpha(x, 0, goimagecolor.Alpha{A: grey})
	}
	return &softMaskSource{under: under, mask: mask, transfer: transfer}
}

// TestASoftMaskMultipliesByTheGrey is SoftPaintContext.getRaster with no
// transfer function: `pixelOutput[3] = Math.round(pixelOutput[3] * (g / 255f))`.
func TestASoftMaskMultipliesByTheGrey(t *testing.T) {
	source := maskOfGreys(solidSource{color: goimagecolor.RGBA{R: 0x40, A: 0xFF}},
		[]uint8{0, 0x40, 0x80, 0xFF}, nil)

	for x, want := range []uint8{0, 0x40, 0x80, 0xFF} {
		c, painted := source.colorAt(x, 0)
		if !painted {
			t.Fatalf("(%d,0) painted nothing", x)
		}
		// 0xFF * g/255, rounded, which for these greys is g itself.
		if c.A != want {
			t.Errorf("(%d,0) has alpha %#02x, want %#02x", x, c.A, want)
		}
		if c.R != 0x40 {
			t.Errorf("(%d,0) is %#02x, want the paint underneath unchanged", x, c.R)
		}
	}
}

// TestASoftMaskAppliesTheTransferFunction is the /TR arm, which maps the grey
// before it multiplies.
func TestASoftMaskAppliesTheTransferFunction(t *testing.T) {
	source := maskOfGreys(solidSource{color: goimagecolor.RGBA{A: 0xFF}},
		[]uint8{0, 0x40, 0x80, 0xFF}, invertingFunction(t))

	// The function is 1 - g/255, so the alpha runs the other way. 0x40/255 is
	// 0.25098, one minus that is 0.74902, and 255 times that rounds to 191.
	for x, want := range []uint8{0xFF, 191, 127, 0} {
		c, _ := source.colorAt(x, 0)
		if c.A != want {
			t.Errorf("(%d,0) has alpha %d, want %d", x, c.A, want)
		}
	}
}

// TestASoftMaskIsTheBackdropOutsideItself is the else of the bounds test, which
// is what PDFBOX's `bc` is for: a pixel the mask does not cover is worth the
// backdrop colour's grey and not zero.
func TestASoftMaskIsTheBackdropOutsideItself(t *testing.T) {
	source := maskOfGreys(solidSource{color: goimagecolor.RGBA{A: 0xFF}},
		[]uint8{0xFF}, nil)
	source.backdrop = 0x80

	if c, _ := source.colorAt(0, 0); c.A != 0xFF {
		t.Errorf("inside the mask the alpha is %#02x, want 0xff", c.A)
	}
	// 0xFF * 0x80/255 rounds to 128.
	if c, _ := source.colorAt(5, 0); c.A != 0x80 {
		t.Errorf("outside the mask the alpha is %#02x, want the backdrop 0x80", c.A)
	}
	if c, _ := source.colorAt(0, 3); c.A != 0x80 {
		t.Errorf("below the mask the alpha is %#02x, want the backdrop 0x80", c.A)
	}
}

// TestASoftMaskSitsAtItsOrigin is `x1 = x1 - (int) origin.getX()`: the mask is
// read at the pixel offset by its origin, not at the device pixel.
func TestASoftMaskSitsAtItsOrigin(t *testing.T) {
	source := maskOfGreys(solidSource{color: goimagecolor.RGBA{A: 0xFF}},
		[]uint8{0xFF, 0}, nil)
	source.originX = 10

	if c, _ := source.colorAt(10, 0); c.A != 0xFF {
		t.Errorf("(10,0) is the mask's first pixel and has alpha %#02x, want 0xff", c.A)
	}
	if c, _ := source.colorAt(11, 0); c.A != 0 {
		t.Errorf("(11,0) is the mask's second pixel and has alpha %#02x, want 0", c.A)
	}
	if c, _ := source.colorAt(0, 0); c.A != 0 {
		t.Errorf("(0,0) is outside the mask and has alpha %#02x, want the backdrop 0", c.A)
	}
}

// TestDrawSurfacePutsAnOffscreenDown is the two calls PDFPrintable rasterizes
// through: NewOffscreen makes a transparent surface of the size asked for, and
// DrawSurface puts it onto a white ground.
func TestDrawSurfacePutsAnOffscreenDown(t *testing.T) {
	page := NewImage(8, 8, rendering.RGB)
	page.SetAntiAliasing(false)
	// Something under the blit, which clearRect's white ground covers.
	page.SetPaint(rendering.ColorPaint{Red: 0, Green: 0, Blue: 1, Alpha: 1})
	if err := page.Fill(rectangle(0, 0, 8, 8)); err != nil {
		t.Fatalf("Fill: %v", err)
	}

	offscreen, isImage := page.NewOffscreen(8, 8).(*Image)
	if !isImage {
		t.Fatal("NewOffscreen answered something other than an Image")
	}
	if got := offscreen.dst.Bounds(); got.Dx() != 8 || got.Dy() != 8 {
		t.Errorf("the offscreen is %v, want 8x8", got)
	}
	if c := offscreen.dst.RGBAAt(0, 0); c.A != 0 {
		t.Errorf("the offscreen starts as %v, want transparent", c)
	}

	offscreen.SetAntiAliasing(false)
	offscreen.SetPaint(black)
	if err := offscreen.Fill(rectangle(2, 2, 4, 4)); err != nil {
		t.Fatalf("Fill: %v", err)
	}
	if err := page.DrawSurface(offscreen); err != nil {
		t.Fatalf("DrawSurface: %v", err)
	}

	wantColor(t, page, 4, 4, goimagecolor.RGBA{A: 0xFF}, "where the offscreen drew")
	// clearRect puts white down under the whole blit, so the blue is gone even
	// where the offscreen drew nothing.
	wantColor(t, page, 0, 0, white, "where the offscreen drew nothing")
}

// TestDrawSurfaceHonoursTheClip is `clearRect` and `drawImage` both going
// through the Graphics2D's clip.
func TestDrawSurfaceHonoursTheClip(t *testing.T) {
	page := NewImage(8, 8, rendering.RGB)
	page.SetAntiAliasing(false)
	page.SetPaint(rendering.ColorPaint{Red: 0, Green: 0, Blue: 1, Alpha: 1})
	if err := page.Fill(rectangle(0, 0, 8, 8)); err != nil {
		t.Fatalf("Fill: %v", err)
	}

	offscreen := page.NewOffscreen(8, 8).(*Image)
	offscreen.SetAntiAliasing(false)
	offscreen.SetPaint(black)
	if err := offscreen.Fill(rectangle(0, 0, 8, 8)); err != nil {
		t.Fatalf("Fill: %v", err)
	}

	page.SetClip(areaOf(0, 0, 4, 8))
	if err := page.DrawSurface(offscreen); err != nil {
		t.Fatalf("DrawSurface: %v", err)
	}

	wantColor(t, page, 1, 4, goimagecolor.RGBA{A: 0xFF}, "inside the clip")
	wantColor(t, page, 6, 4, goimagecolor.RGBA{B: 0xFF, A: 0xFF}, "outside the clip")
}

// areaOf is a clipping area of the given box.
func areaOf(x, y, w, h float64) *geom.Area {
	return geom.NewAreaOfShape(geom.NewRectangle2D(x, y, w, h))
}
