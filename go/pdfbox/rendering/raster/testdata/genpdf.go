//go:build ignore

// Writes graphics.pdf, the page the end-to-end comparison renders.
//
// The file is checked in rather than built by the test, because both sides
// have to read the same bytes: PDFBox renders it once into graphics-java.png
// -- see RenderDrv.java -- and the Go test renders it on every run and holds
// itself to that image. A page rebuilt at test time would drift out from under
// the reference without saying so.
//
// Regenerate with:
//
//	go run testdata/genpdf.go
//
// and then re-run RenderDrv.java, because the pixels move with the page.
//
// What it draws is what a PageDrawer asks a Backend for: fills under both
// winding rules, strokes with each join, each cap and a dash pattern, a clip,
// a transform, an alpha constant, a blend mode, a transparency group and an
// axial shading. Nothing here is text -- the glyph tests cover that, and a
// font would make the comparison a test of the shaper instead.
package main

import (
	"log"
	"os"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/blend"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/shading"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/state"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

func main() {
	document := pdmodel.NewPDDocument()
	defer document.Close()

	page := pdmodel.NewPDPageOfSize(common.NewPDRectangleOfSize(200, 200))
	document.AddPage(page)

	c, err := pdmodel.NewPDPageContentStream(document, page)
	check(err)

	// 1. a plain fill and an even-odd fill, side by side. The second is a
	// square with a square inside it, so the middle drops out.
	c.SetNonStrokingColorRGB(0.1, 0.2, 0.8)
	check(c.AddRect(10, 160, 30, 30))
	check(c.Fill())

	c.SetNonStrokingColorRGB(0.9, 0.3, 0.1)
	check(c.AddRect(50, 160, 30, 30))
	check(c.AddRect(58, 168, 14, 14))
	check(c.FillEvenOdd())

	// 2. the three joins, on a right angle each
	c.SetStrokingColorRGB(0, 0, 0)
	check(c.SetLineWidth(6))
	for i, join := range []int{0, 1, 2} {
		check(c.SetLineJoinStyle(join))
		x := float32(95 + i*35)
		check(c.MoveTo(x, 160))
		check(c.LineTo(x+18, 160))
		check(c.LineTo(x+18, 190))
		check(c.Stroke())
	}

	// 3. the three caps
	check(c.SetLineJoinStyle(0))
	check(c.SetLineWidth(10))
	for i, cap := range []int{0, 1, 2} {
		check(c.SetLineCapStyle(cap))
		y := float32(140 - i*16)
		check(c.MoveTo(20, y))
		check(c.LineTo(60, y))
		check(c.Stroke())
	}

	// 4. a dashed line, and the same pattern with a phase
	check(c.SetLineCapStyle(0))
	check(c.SetLineWidth(4))
	check(c.SetLineDashPattern([]float32{8, 5}, 0))
	check(c.MoveTo(80, 140))
	check(c.LineTo(190, 140))
	check(c.Stroke())
	check(c.SetLineDashPattern([]float32{8, 5}, 3))
	check(c.MoveTo(80, 128))
	check(c.LineTo(190, 128))
	check(c.Stroke())
	check(c.SetLineDashPattern(nil, 0))

	// 5. a clip and a transform: a rotated square, clipped to a band, so both
	// the clip and the CTM have to be right for the ink to land
	check(c.SaveGraphicsState())
	check(c.AddRect(80, 95, 110, 20))
	check(c.Clip())
	check(c.Transform(util.NewMatrixOf(0.866, 0.5, -0.5, 0.866, 120, 60)))
	c.SetNonStrokingColorRGB(0.2, 0.6, 0.2)
	check(c.AddRect(0, 0, 40, 40))
	check(c.Fill())
	check(c.RestoreGraphicsState())

	// 6. a constant alpha over an opaque ground
	c.SetNonStrokingColorRGB(0.9, 0.7, 0)
	check(c.AddRect(20, 60, 50, 25))
	check(c.Fill())

	half := state.NewPDExtendedGraphicsState()
	halfAlpha := float32(0.5)
	half.SetNonStrokingAlphaConstant(&halfAlpha)
	check(c.SaveGraphicsState())
	check(c.SetGraphicsStateParameters(half))
	c.SetNonStrokingColorRGB(0, 0.2, 0.9)
	check(c.AddRect(35, 68, 50, 25))
	check(c.Fill())
	check(c.RestoreGraphicsState())

	// 7. a blend mode over the same ground
	multiply := state.NewPDExtendedGraphicsState()
	multiply.SetBlendMode(blend.Multiply)
	check(c.SaveGraphicsState())
	check(c.SetGraphicsStateParameters(multiply))
	c.SetNonStrokingColorRGB(0.4, 0.8, 0.4)
	check(c.AddRect(95, 60, 45, 25))
	check(c.Fill())
	check(c.RestoreGraphicsState())

	// 8. an axial shading, from red to blue across a band
	check(c.SaveGraphicsState())
	check(c.AddRect(20, 15, 160, 30))
	check(c.Clip())
	check(c.ShadingFill(axialShading()))
	check(c.RestoreGraphicsState())

	check(c.Close())
	out, err := os.Create("testdata/graphics.pdf")
	check(err)
	check(document.Save(out))
	check(out.Close())
}

// axialShading is a type 2 shading with a type 2 exponential function, red at
// the left edge of the band and blue at the right.
func axialShading() shading.Shading {
	function := cos.NewDictionary()
	function.SetInt(cos.FunctionType, 2)
	function.SetItem(cos.Domain, floats(0, 1))
	function.SetItem(cos.C0, floats(1, 0, 0))
	function.SetItem(cos.C1, floats(0, 0, 1))
	function.SetInt(cos.N, 1)

	dictionary := cos.NewDictionary()
	dictionary.SetInt(cos.ShadingType, 2)
	dictionary.SetItem(cos.ColorSpace, cos.DeviceRGB)
	dictionary.SetItem(cos.Coords, floats(20, 0, 180, 0))
	dictionary.SetItem(cos.Function, function)
	dictionary.SetItem(cos.Extend, booleans(true, true))
	return shading.NewPDShadingType2(dictionary)
}

func floats(values ...float32) *cos.Array {
	array := cos.NewArray()
	for _, v := range values {
		array.Add(cos.NewFloat(v))
	}
	return array
}

func booleans(values ...bool) *cos.Array {
	array := cos.NewArray()
	for _, v := range values {
		array.Add(cos.GetBoolean(v))
	}
	return array
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
