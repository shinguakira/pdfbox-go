package raster

// A2 of `track/raster`, and a correction to what the task file asked for.
//
// It says "Port the shading tests -- `PDShadingTest` and the type-specific
// cases". **There is no such test.** PDFBox has no test for the shading
// package at all: `grep -rli shading` over `pdfbox/src/test` finds two files,
// and neither tests a shading -- one lists operator names and the other checks
// that `shadingFill` refuses a shading with an empty dictionary. Slice 9's A3
// had already found this and wrote its own tests from the Java source and the
// specification, which is what these do too.
//
// What a `ShadingContext` answers is a colour at a point, and that is what is
// asserted here: no rasteriser, no page, one pixel at a time. Every expected
// value below is worked out by hand from `AxialShadingContext`'s own
// arithmetic, and the working is written out, because a value read off the
// port would only say the port agrees with itself.

import (
	goimage "image"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/shading"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// greyRampShading is `[/Pattern]`'s simplest useful case: an axial shading
// from black at x=0 to white at x=100, in DeviceRGB, with no extension.
//
//	/ShadingType 2
//	/ColorSpace /DeviceRGB
//	/Coords [0 0 100 0]
//	/Function << /FunctionType 2 /Domain [0 1] /C0 [0 0 0] /C1 [1 1 1] /N 1 >>
//	/Extend [false false]
func greyRampShading(t *testing.T) shading.Shading {
	t.Helper()
	fn := cos.NewDictionary()
	fn.SetInt(cos.FunctionType, 2)
	fn.SetItem(cos.Domain, floatArray([]float32{0, 1}))
	fn.SetItem(cos.C0, floatArray([]float32{0, 0, 0}))
	fn.SetItem(cos.C1, floatArray([]float32{1, 1, 1}))
	fn.SetInt(cos.N, 1)

	dictionary := cos.NewDictionary()
	dictionary.SetInt(cos.ShadingType, 2)
	dictionary.SetItem(cos.ColorSpace, cos.DeviceRGB)
	dictionary.SetItem(cos.Coords, floatArray([]float32{0, 0, 100, 0}))
	dictionary.SetItem(cos.Function, fn)
	extend := cos.NewArray()
	extend.Add(cos.GetBoolean(false))
	extend.Add(cos.GetBoolean(false))
	dictionary.SetItem(cos.Extend, extend)

	return shading.NewPDShadingType2(dictionary)
}

// TestAxialShadingColourAtAPoint is `AxialShadingContext.getRaster`, one pixel
// at a time.
//
// The arithmetic, from the Java:
//
//	x1x0 = 100, y1y0 = 0, denom = 100² + 0² = 10000
//	inputValue = (x1x0*(px - x0) + y1y0*(py - y0)) / denom = px / 100
//
// The colour is not evaluated at that parameter directly. The context builds a
// table of `factor + 1` colours, where `factor = ceil(the diagonal of the
// device bounds)`, and the raster loop reads `colorTable[(int)(inputValue *
// factor)]`, whose entry was evaluated at `domain[0] + d1d0 * i / factor`.
//
// The bounds here are 300 by 400, whose diagonal is exactly 500, so
// `factor = 500` and every quarter falls on a table entry:
//
//	px=25  -> u=0.25 -> key=125 -> t=0.25 -> C=0.25 -> (int)(0.25*255) = 63
//	px=50  -> u=0.50 -> key=250 -> t=0.50 -> C=0.50 -> (int)(0.50*255) = 127
//	px=75  -> u=0.75 -> key=375 -> t=0.75 -> C=0.75 -> (int)(0.75*255) = 191
//	px=100 -> u=1.00 -> key=500 -> t=1.00 -> C=1.00 -> (int)(1.00*255) = 255
//
// The last is in range: the loop refuses `inputValue > 1`, not `>= 1`. The
// truncation is `convertToRGB`'s, and it is why 0.5 gives 127 and not 128.
func TestAxialShadingColourAtAPoint(t *testing.T) {
	context, err := newShadingContext(greyRampShading(t), util.NewMatrix(),
		geom.NewAffineTransform(1, 0, 0, 1, 0, 0), goimage.Rect(0, 0, 300, 400))
	if err != nil {
		t.Fatalf("newShadingContext: %v", err)
	}

	for _, c := range []struct {
		x, want int
	}{{0, 0}, {25, 63}, {50, 127}, {75, 191}, {100, 255}} {
		got, painted := context.colorAt(c.x, 10)
		if !painted {
			t.Errorf("x=%d paints nothing, and it is inside the axis", c.x)
			continue
		}
		want := uint8(c.want)
		if got.R != want || got.G != want || got.B != want {
			t.Errorf("x=%d is (%d,%d,%d), want (%d,%d,%d)",
				c.x, got.R, got.G, got.B, want, want, want)
		}
	}
}

// TestAxialShadingPaintsNothingOutsideItsAxis is the two `continue` arms.
//
// With `/Extend [false false]` and no `/Background`, a point before the start
// of the axis or after its end is left as it was -- Java's raster loop skips
// the pixel and never writes it.
func TestAxialShadingPaintsNothingOutsideItsAxis(t *testing.T) {
	context, err := newShadingContext(greyRampShading(t), util.NewMatrix(),
		geom.NewAffineTransform(1, 0, 0, 1, 0, 0), goimage.Rect(0, 0, 300, 400))
	if err != nil {
		t.Fatalf("newShadingContext: %v", err)
	}

	for _, x := range []int{-1, -50, 101, 250} {
		if _, painted := context.colorAt(x, 10); painted {
			t.Errorf("x=%d is painted, and it is off the end of an unextended axis", x)
		}
	}
}

// floatArray is a COSArray of the given numbers.
func floatArray(values []float32) *cos.Array {
	array := cos.NewArray()
	array.SetFloatArray(values)
	return array
}

// radialShading is a type 3 shading: two concentric circles at the origin,
// the inner of radius 0 and the outer of radius 100, black at the centre and
// white at the rim.
//
//	/ShadingType 3
//	/Coords [0 0 0  0 0 100]
func radialShading(t *testing.T) shading.Shading {
	t.Helper()
	fn := cos.NewDictionary()
	fn.SetInt(cos.FunctionType, 2)
	fn.SetItem(cos.Domain, floatArray([]float32{0, 1}))
	fn.SetItem(cos.C0, floatArray([]float32{0, 0, 0}))
	fn.SetItem(cos.C1, floatArray([]float32{1, 1, 1}))
	fn.SetInt(cos.N, 1)

	dictionary := cos.NewDictionary()
	dictionary.SetInt(cos.ShadingType, 3)
	dictionary.SetItem(cos.ColorSpace, cos.DeviceRGB)
	dictionary.SetItem(cos.Coords, floatArray([]float32{0, 0, 0, 0, 0, 100}))
	dictionary.SetItem(cos.Function, fn)
	extend := cos.NewArray()
	extend.Add(cos.GetBoolean(false))
	extend.Add(cos.GetBoolean(false))
	dictionary.SetItem(cos.Extend, extend)

	return shading.NewPDShadingType3(dictionary)
}

// TestRadialShadingColourAtAPoint is RadialShadingContext.getRaster.
//
// The circles share a centre, so Adobe's quadratic reduces to the distance
// from it over the radius: s = √(x² + y²) / 100. The colour table is the
// axial one's -- the same 300 by 400 surface, so factor is 500 -- and the
// same truncation applies.
//
//	(0,0)   -> s=0    -> key=0   -> 0
//	(25,0)  -> s=0.25 -> key=125 -> (int)(0.25*255) = 63
//	(60,80) -> s=1.00 -> key=500 -> 255, since √(3600+6400) = 100
func TestRadialShadingColourAtAPoint(t *testing.T) {
	context, err := newShadingContext(radialShading(t), util.NewMatrix(),
		geom.NewAffineTransform(1, 0, 0, 1, 0, 0), goimage.Rect(0, 0, 300, 400))
	if err != nil {
		t.Fatalf("newShadingContext: %v", err)
	}

	for _, c := range []struct {
		x, y, want int
	}{{0, 0, 0}, {25, 0, 63}, {50, 0, 127}, {60, 80, 255}} {
		got, painted := context.colorAt(c.x, c.y)
		if !painted {
			t.Errorf("(%d,%d) paints nothing, and it is inside the outer circle",
				c.x, c.y)
			continue
		}
		want := uint8(c.want)
		if got.R != want || got.G != want || got.B != want {
			t.Errorf("(%d,%d) is (%d,%d,%d), want (%d,%d,%d)",
				c.x, c.y, got.R, got.G, got.B, want, want, want)
		}
	}
}

// TestRadialShadingPaintsNothingOutsideItsCircles is the unextended case: a
// point beyond the outer circle has no parameter in range and nothing to
// extend to.
func TestRadialShadingPaintsNothingOutsideItsCircles(t *testing.T) {
	context, err := newShadingContext(radialShading(t), util.NewMatrix(),
		geom.NewAffineTransform(1, 0, 0, 1, 0, 0), goimage.Rect(0, 0, 300, 400))
	if err != nil {
		t.Fatalf("newShadingContext: %v", err)
	}
	if _, painted := context.colorAt(150, 0); painted {
		t.Error("(150,0) is painted, and it is outside an unextended circle")
	}
}
