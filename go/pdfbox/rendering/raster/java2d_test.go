package raster

// What Java2D draws, and what this backend draws beside it.
//
// **PDFBox has no rendering test.** What this package is a substitution for is
// `java.awt.Graphics2D`, so that is the reference, and `testdata/java2d.txt`
// is what a JDK 17 Graphics2D produced for each case. The driver that wrote it
// is checked in beside it, `testdata/Java2DDrv.java`, so the numbers can be
// regenerated rather than believed:
//
//	javac -d . Java2DDrv.java
//	java -Djava.awt.headless=true -cp . Java2DDrv > java2d.txt
//
// Nothing here is derived by hand, and nothing is read off this port.
//
// Each case is in the file twice, because `KEY_STROKE_CONTROL` changes what
// Java2D draws and PDFBox never sets it:
//
//	name       VALUE_STROKE_PURE, the geometry as given
//	nameNorm   VALUE_STROKE_NORMALIZE, the JDK default, and so PDFBox's
//
// The backend is held to both, under the hint each was drawn with: it takes
// the hint for a stroke, where it normalizes the path as Marlin does, and for
// a fill with antialiasing off, where the hint decides whether a pixel is
// taken at its centre or its corner (fillrule.go). It is only a stroke drawn
// with antialiasing on that the backend leaves as given whatever the hint
// says, and what that costs is each case's normDiffering and normWorst, which
// TestAgainstJava2DNormalized holds it to.

import (
	goimage "image"
	goimagecolor "image/color"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/rendering"
)

// java2dCase is one case of testdata/java2d.txt and what draws it here.
//
// differing and worst are how far this backend is from Java2D's pixels, and
// they are measurements, not targets: zero for every shape whose edges are
// axis-aligned, and a few units of one channel where an edge is diagonal or
// curved, because Java2D's rasteriser and this one quantise partial coverage
// differently. See TestAgainstJava2D for what that difference is made of.
type java2dCase struct {
	name          string
	width, height int
	antiAliasing  bool
	differing     int
	worst         int

	// normDiffering and normWorst are the same, under
	// VALUE_STROKE_NORMALIZE, which is what PDFBox renders under.
	normDiffering int
	normWorst     int

	draw func(i *Image)
}

var java2dCases = []java2dCase{
	{name: "fillRect", width: 20, height: 20, draw: func(i *Image) {
		must(i.Fill(geom.NewRectangle2D(5, 5, 10, 10)))
	}},
	{name: "fillTranslated", width: 20, height: 20, draw: func(i *Image) {
		i.SetTransform(geom.NewAffineTransform(1, 0, 0, 1, 5, 5))
		must(i.Fill(geom.NewRectangle2D(0, 0, 5, 5)))
	}},
	{name: "fillClipped", width: 20, height: 20, draw: func(i *Image) {
		i.SetClip(geom.NewAreaOfShape(geom.NewRectangle2D(0, 0, 10, 20)))
		must(i.Fill(geom.NewRectangle2D(5, 5, 10, 10)))
	}},
	// A fill whose edges fall halfway across a pixel, so every edge pixel is
	// exactly half covered. Java2D writes 0x80 there and this backend 0x7f:
	// the whole difference is one unit on 36 edge pixels.
	{name: "fillHalfAA", width: 20, height: 20, antiAliasing: true,
		differing: 36, worst: 1, normDiffering: 36, normWorst: 1, draw: func(i *Image) {
			must(i.Fill(geom.NewRectangle2D(5.5, 5.5, 9, 9)))
		}},
	{name: "strokeLine", width: 20, height: 20, draw: func(i *Image) {
		i.SetStroke(&rendering.Stroke{LineWidth: 4, MiterLimit: 10})
		must(i.Draw(lineShape(2, 10, 18, 10)))
	}},
	{name: "windNonZero", width: 20, height: 20, draw: func(i *Image) {
		must(i.Fill(squareInASquare(geom.WindNonZero)))
	}},
	{name: "windEvenOdd", width: 20, height: 20, draw: func(i *Image) {
		must(i.Fill(squareInASquare(geom.WindEvenOdd)))
	}},
	{name: "joinMiter", width: 32, height: 32, antiAliasing: true,
		normDiffering: 91, normWorst: 64, draw: func(i *Image) {
			i.SetStroke(&rendering.Stroke{LineWidth: 8, LineJoin: 0, MiterLimit: 10})
			must(i.Draw(rightAngle()))
		}},
	{name: "joinRound", width: 32, height: 32, antiAliasing: true,
		differing: 7, worst: 11, normDiffering: 90, normWorst: 64, draw: func(i *Image) {
			i.SetStroke(&rendering.Stroke{LineWidth: 8, LineJoin: 1, MiterLimit: 10})
			must(i.Draw(rightAngle()))
		}},
	{name: "joinBevel", width: 32, height: 32, antiAliasing: true,
		differing: 4, worst: 1, normDiffering: 88, normWorst: 64, draw: func(i *Image) {
			i.SetStroke(&rendering.Stroke{LineWidth: 8, LineJoin: 2, MiterLimit: 10})
			must(i.Draw(rightAngle()))
		}},
	{name: "capButt", width: 24, height: 24, antiAliasing: true,
		normDiffering: 28, normWorst: 1, draw: func(i *Image) {
			i.SetStroke(&rendering.Stroke{LineWidth: 8, LineCap: 0, MiterLimit: 10})
			must(i.Draw(lineShape(8, 12, 16, 12)))
		}},
	{name: "capRound", width: 24, height: 24, antiAliasing: true,
		differing: 28, worst: 15, normDiffering: 47, normWorst: 17, draw: func(i *Image) {
			i.SetStroke(&rendering.Stroke{LineWidth: 8, LineCap: 1, MiterLimit: 10})
			must(i.Draw(lineShape(8, 12, 16, 12)))
		}},
	{name: "capSquare", width: 24, height: 24, antiAliasing: true,
		normDiffering: 44, normWorst: 1, draw: func(i *Image) {
			i.SetStroke(&rendering.Stroke{LineWidth: 8, LineCap: 2, MiterLimit: 10})
			must(i.Draw(lineShape(8, 12, 16, 12)))
		}},
	{name: "dashed", width: 24, height: 12, draw: func(i *Image) {
		i.SetStroke(&rendering.Stroke{LineWidth: 4, MiterLimit: 10,
			DashArray: []float32{4, 4}})
		must(i.Draw(lineShape(0, 6, 24, 6)))
	}},
	{name: "dashedPhase", width: 24, height: 12, draw: func(i *Image) {
		i.SetStroke(&rendering.Stroke{LineWidth: 4, MiterLimit: 10,
			DashArray: []float32{4, 4}, DashPhase: 2})
		must(i.Draw(lineShape(0, 6, 24, 6)))
	}},
	// A join whose miter is longer than the limit allows, which both cut back
	// to a bevel in the same place. What differs is the coverage along four
	// long diagonal edges.
	{name: "miterLimited", width: 40, height: 24, antiAliasing: true,
		differing: 129, worst: 7, normDiffering: 118, normWorst: 61, draw: func(i *Image) {
			i.SetStroke(&rendering.Stroke{LineWidth: 6, LineJoin: 0, MiterLimit: 2})
			path := geom.NewPathDouble()
			path.MoveTo(2, 20)
			path.LineTo(20, 4)
			path.LineTo(38, 20)
			must(i.Draw(path))
		}},
	{name: "strokeCurve", width: 32, height: 32, antiAliasing: true,
		differing: 102, worst: 25, normDiffering: 109, normWorst: 25, draw: func(i *Image) {
			i.SetStroke(&rendering.Stroke{LineWidth: 5, LineJoin: 1, MiterLimit: 10})
			path := geom.NewPathDouble()
			path.MoveTo(4, 26)
			path.CurveTo(4, 4, 28, 4, 28, 26)
			must(i.Draw(path))
		}},
	// A stroke through a transform is widened by it, as everything else is.
	// PageDrawer.getStroke has put the width through the CTM, and what is in
	// force here is the page's own transform, the scale it is rendered at, so
	// a page at 144 dpi strokes twice as wide as one at 72.
	{name: "strokeScaled", width: 24, height: 20, antiAliasing: true,
		normDiffering: 44, normWorst: 1, draw: func(i *Image) {
			i.SetTransform(geom.NewAffineTransform(2, 0, 0, 2, 0, 0))
			i.SetStroke(&rendering.Stroke{LineWidth: 4, MiterLimit: 10})
			must(i.Draw(lineShape(2, 5, 10, 5)))
		}},
	{name: "strokeShrunk", width: 24, height: 20, antiAliasing: true,
		normDiffering: 44, normWorst: 1, draw: func(i *Image) {
			i.SetTransform(geom.NewAffineTransform(0.5, 0, 0, 0.5, 0, 0))
			i.SetStroke(&rendering.Stroke{LineWidth: 8, MiterLimit: 10})
			must(i.Draw(lineShape(4, 20, 44, 20)))
		}},
	// The dashes and the phase are scaled with the width: this draws the
	// pixels of dashedPhase.
	{name: "dashedScaled", width: 24, height: 12, draw: func(i *Image) {
		i.SetTransform(geom.NewAffineTransform(2, 0, 0, 2, 0, 0))
		i.SetStroke(&rendering.Stroke{LineWidth: 2, MiterLimit: 10,
			DashArray: []float32{2, 2}, DashPhase: 1})
		must(i.Draw(lineShape(0, 3, 12, 3)))
	}},
	// A scale that is not the same both ways, through which the pen is an
	// ellipse: the horizontal arm is 4 pixels thick and the vertical one 8.
	{name: "strokeNonUniform", width: 32, height: 28, antiAliasing: true,
		normDiffering: 83, normWorst: 64, draw: func(i *Image) {
			i.SetTransform(geom.NewAffineTransform(2, 0, 0, 1, 0, 0))
			i.SetStroke(&rendering.Stroke{LineWidth: 4, MiterLimit: 10})
			path := geom.NewPathDouble()
			path.MoveTo(2, 6)
			path.LineTo(12, 6)
			path.LineTo(12, 24)
			must(i.Draw(path))
		}},
	// A transform that flattens everything onto a line, through which Marlin
	// strokes nothing at all.
	{name: "strokeSingular", width: 20, height: 20, antiAliasing: true, draw: func(i *Image) {
		i.SetTransform(geom.NewAffineTransform(1, 0, 0, 0, 0, 0))
		i.SetStroke(&rendering.Stroke{LineWidth: 4, MiterLimit: 10})
		must(i.Draw(lineShape(2, 10, 18, 10)))
	}},
	// Fills whose edges fall between pixel centres, with antialiasing off,
	// which is how PDFBox fills every rectangular path. Java2D puts them on
	// different pixels depending on the stroke that happens to be set, and
	// under VALUE_STROKE_PURE on different pixels again; see fillrule.go.
	{name: "fillFractional", width: 16, height: 16, draw: func(i *Image) {
		must(i.Fill(pdfRect(0, 0, 9.59, 4.11)))
	}},
	{name: "fillFractionalOffset", width: 16, height: 16, draw: func(i *Image) {
		must(i.Fill(pdfRect(0.3, 0.3, 4.41, 4.41)))
	}},
	{name: "fillFractionalWide", width: 16, height: 16, draw: func(i *Image) {
		i.SetStroke(&rendering.Stroke{LineWidth: 3, MiterLimit: 10})
		must(i.Fill(pdfRect(0.3, 0.3, 4.41, 4.41)))
	}},
	{name: "fillFractionalDashed", width: 16, height: 16, draw: func(i *Image) {
		i.SetStroke(&rendering.Stroke{LineWidth: 1, MiterLimit: 10,
			DashArray: []float32{4, 4}})
		must(i.Fill(pdfRect(0.3, 0.3, 4.41, 4.41)))
	}},
	{name: "fillFractionalScaled", width: 16, height: 16, draw: func(i *Image) {
		i.SetTransform(geom.NewAffineTransform(1.37, 0, 0, 1.37, 0, 0))
		must(i.Fill(pdfRect(0, 0, 7, 3)))
	}},
	{name: "fillFractionalScaledThin", width: 16, height: 16, draw: func(i *Image) {
		i.SetStroke(&rendering.Stroke{LineWidth: 0.5, MiterLimit: 10})
		i.SetTransform(geom.NewAffineTransform(1.37, 0, 0, 1.37, 0, 0))
		must(i.Fill(pdfRect(0, 0, 7, 3)))
	}},
	{name: "fillFractionalTriangle", width: 16, height: 16, draw: func(i *Image) {
		path := geom.NewPathDouble()
		path.MoveTo(1.3, 1.2)
		path.LineTo(13.8, 4.4)
		path.LineTo(4.6, 14.1)
		path.ClosePath()
		must(i.Fill(path))
	}},
	{name: "fillFractionalCurve", width: 16, height: 16, draw: func(i *Image) {
		path := geom.NewPathDouble()
		path.MoveTo(2.4, 13.6)
		path.CurveTo(2.4, 2.2, 13.7, 2.2, 13.7, 13.6)
		path.ClosePath()
		must(i.Fill(path))
	}},
	// Images through drawImage(image, transform, null), under the bicubic hint.
	// Java2D copies one that lands one to one on whole pixels and sends every
	// other through TransformHelper; see transformhelper.go.
	{name: "imageCopy", width: 20, height: 20, antiAliasing: true, draw: func(i *Image) {
		drawImageThrough(i, patternImage(8, 6, "rgb"), geom.NewAffineTransform(1, 0, 0, 1, 3, 4))
	}},
	{name: "imageOffset", width: 20, height: 20, antiAliasing: true, draw: func(i *Image) {
		drawImageThrough(i, patternImage(8, 6, "rgb"), geom.NewAffineTransform(1, 0, 0, 1, 3.4, 4.6))
	}},
	{name: "imageScaled", width: 24, height: 20, antiAliasing: true, draw: func(i *Image) {
		drawImageThrough(i, patternImage(8, 6, "rgb"), geom.NewAffineTransform(1.7, 0, 0, 1.7, 2.3, 3.1))
	}},
	{name: "imageShrunk", width: 20, height: 20, antiAliasing: true, draw: func(i *Image) {
		drawImageThrough(i, patternImage(16, 12, "rgb"), geom.NewAffineTransform(0.7, 0, 0, 0.7, 1.5, 2.25))
	}},
	{name: "imageRotated", width: 24, height: 24, antiAliasing: true, draw: func(i *Image) {
		// AffineTransform.getRotateInstance(toRadians(30), 12, 12), written the
		// way setToRotation writes it.
		sin, cos := math.Sincos(30 * (math.Pi / 180))
		at := geom.NewAffineTransform(cos, sin, -sin, cos, 12*(1-cos)+12*sin, 12*(1-cos)-12*sin)
		at.Translate(6.5, 7.25)
		at.Scale(1.2, 1.2)
		drawImageThrough(i, patternImage(8, 6, "rgb"), at)
	}},
	// The one case that differs, and only where the image is partly
	// transparent: the sampling is exact, and the two composite a premultiplied
	// colour onto the page with different arithmetic -- Java2D's SrcOver
	// MaskBlit in MUL8 bytes, this backend in floats.
	{name: "imageAlpha", width: 20, height: 20, antiAliasing: true,
		differing: 30, worst: 2, normDiffering: 30, normWorst: 2, draw: func(i *Image) {
			drawImageThrough(i, patternImage(6, 5, "argb"), geom.NewAffineTransform(1.6, 0, 0, 1.6, 2.2, 2.7))
		}},
	{name: "imageGray", width: 20, height: 20, antiAliasing: true, draw: func(i *Image) {
		drawImageThrough(i, patternImage(12, 10, "gray"), geom.NewAffineTransform(0.75, 0, 0, 0.75, 3.3, 2.1))
	}},
	// The way PageDrawer draws one: the page's transform flips y, and the
	// image's own transform flips it back.
	{name: "imageFlipped", width: 20, height: 20, antiAliasing: true, draw: func(i *Image) {
		i.SetTransform(geom.NewAffineTransform(1, 0, 0, -1, 0, 20))
		drawImageThrough(i, patternImage(8, 6, "rgb"), geom.NewAffineTransform(1.5, 0, 0, -1.5, 2.4, 16.3))
	}},
}

// must fails a case that cannot draw at all, which is a bug in the backend and
// not something a comparison should try to describe.
func must(err error) {
	if err != nil {
		panic(err)
	}
}

func lineShape(x0, y0, x1, y1 float64) geom.Shape {
	path := geom.NewPathDouble()
	path.MoveTo(x0, y0)
	path.LineTo(x1, y1)
	return path
}

// squareInASquare is what the winding cases fill: two squares wound the same
// way, so the middle stays in under nonzero and drops out under even-odd.
func squareInASquare(rule int) geom.Shape {
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

// rightAngle is what the joins are drawn on.
func rightAngle() geom.Shape {
	path := geom.NewPathDouble()
	path.MoveTo(0, 20)
	path.LineTo(20, 20)
	path.LineTo(20, 0)
	return path
}

// render draws the case onto a backend the size the driver used, under the
// given KEY_STROKE_CONTROL.
func (c java2dCase) render(normalize bool) *Image {
	i := NewImage(c.width, c.height, rendering.RGB)
	i.SetPaint(black)
	i.SetAntiAliasing(c.antiAliasing)
	i.SetStrokeNormalization(normalize)
	c.draw(i)
	return i
}

// java2dGrids reads the driver's output: one named grid of red-channel bytes
// per case.
func java2dGrids(t *testing.T) map[string][][]uint8 {
	t.Helper()
	contents, err := os.ReadFile("testdata/java2d.txt")
	if err != nil {
		t.Fatalf("reading the Java2D reference: %v", err)
	}
	grids := map[string][][]uint8{}
	var name string
	for _, line := range strings.Split(strings.ReplaceAll(string(contents), "\r\n", "\n"), "\n") {
		if strings.HasPrefix(line, "### ") {
			name = strings.Fields(line)[1]
			grids[name] = nil
			continue
		}
		if line == "" || name == "" {
			continue
		}
		row := make([]uint8, len(line)/2)
		for x := range row {
			value, err := strconv.ParseUint(line[x*2:x*2+2], 16, 8)
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			row[x] = uint8(value)
		}
		grids[name] = append(grids[name], row)
	}
	return grids
}

// gridFor answers one named grid, and fails if the driver did not write it.
func gridFor(t *testing.T, grids map[string][][]uint8, name string) [][]uint8 {
	t.Helper()
	grid, found := grids[name]
	if !found {
		t.Fatalf("the Java2D reference has no case named %q", name)
	}
	return grid
}

// differenceFrom counts the pixels of the backend's red channel that are not
// the reference's, and answers the worst one.
func differenceFrom(t *testing.T, grid [][]uint8, i *Image) (differing, worst int) {
	t.Helper()
	bounds := i.dst.Bounds()
	if len(grid) != bounds.Dy() || len(grid[0]) != bounds.Dx() {
		t.Fatalf("the reference is %dx%d and the backend drew %dx%d",
			len(grid[0]), len(grid), bounds.Dx(), bounds.Dy())
	}
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			delta := int(i.dst.NRGBAAt(x, y).R) - int(grid[y][x])
			if delta == 0 {
				continue
			}
			differing++
			if delta < 0 {
				delta = -delta
			}
			if delta > worst {
				worst = delta
			}
		}
	}
	return differing, worst
}

// TestAgainstJava2D holds every case to the pixels Java2D produced for it.
//
// Sixteen of the twenty-two match exactly, and they are the ones that decide
// whether the port is right: the fills, the clip, the transform, both winding
// rules, the butt and square caps, the miter join, the dashes with and without
// a phase, and the strokes drawn through a transform -- a scale up, a scale
// down, a scale that differs each way, and one that flattens everything.
//
// The other six differ, and only along edges that are neither horizontal nor
// vertical. Two rasterisers are at work -- Java2D's is Marlin, which samples a
// pixel on an 8x8 subpixel grid and truncates the count to a byte, and this
// backend's is freetype's, which integrates the area exactly. On an
// exactly-half-covered pixel Marlin gives 0x80 and freetype 0x7f, and that one
// unit is all of fillHalfAA and all of joinBevel. Where an edge is curved the
// flatteners also disagree about where to place their line segments, which is
// the rest: joinRound, capRound and strokeCurve. In every one of the six the
// ink is in the same place -- what differs is how dark its edge is.
//
// The counts are pinned rather than bounded so that either direction fails.
// A wrong join or a wrong limit could not hide inside them: the bevel this
// backend would draw for a miter covers a whole corner, some 200 pixels at
// full strength.
func TestAgainstJava2D(t *testing.T) {
	grids := java2dGrids(t)
	for _, c := range java2dCases {
		t.Run(c.name, func(t *testing.T) {
			differing, worst := differenceFrom(t, gridFor(t, grids, c.name), c.render(false))
			if differing != c.differing || worst != c.worst {
				t.Errorf("%d pixels differ from Java2D by up to %d, and it was %d by up to %d",
					differing, worst, c.differing, c.worst)
			}
		})
	}
}

// TestAgainstJava2DNormalized is the same twenty-two under the other value of
// KEY_STROKE_CONTROL, which is the one PDFBox actually renders under.
//
// `createDefaultRenderingHints` sets three hints and not this one, so a page
// goes through the Graphics2D default, and the default normalizes: before
// stroking, Marlin moves each segment endpoint to the nearest pixel centre --
// or pixel quarter, with anti-aliasing off -- so that a thin line lands on
// whole pixels instead of straddling two. `normalize.go` is that, ported, and
// `SetStrokeNormalization` is the hint.
//
// Holding the port to both grids is what says the port is right rather than
// merely close: the same twenty-two shapes, drawn twice, against a Java2D told
// to do each thing.
func TestAgainstJava2DNormalized(t *testing.T) {
	grids := java2dGrids(t)
	for _, c := range java2dCases {
		t.Run(c.name, func(t *testing.T) {
			differing, worst := differenceFrom(t,
				gridFor(t, grids, c.name+"Norm"), c.render(true))
			if differing != c.normDiffering || worst != c.normWorst {
				t.Errorf("%d pixels differ from a normalizing Java2D by up to %d, "+
					"and it was %d by up to %d",
					differing, worst, c.normDiffering, c.normWorst)
			}
		})
	}
}

// drawImageThrough draws an image as Graphics2D.drawImage(image, m, null)
// does under the bicubic hint, with the backend's own transform standing for
// the Graphics2D's.
//
// The backend is handed images as PageDrawer hands them, with the transform
// that takes the unit square to where the image goes; this is m written that
// way.
func drawImageThrough(i *Image, img goimage.Image, m *geom.AffineTransform) {
	// PDFRenderer's hint, which Java2DDrv sets.
	i.SetInterpolation(rendering.Bicubic)
	w, h := float64(img.Bounds().Dx()), float64(img.Bounds().Dy())
	at := m.Clone()
	at.Translate(0, h)
	at.Scale(w, -h)
	must(i.drawSampled(img, at, nil))
}

// patternImage is Java2DDrv.pattern: every pixel differs from its neighbours.
// "rgb" is TYPE_INT_RGB, which the port holds as an opaque image.RGBA;
// "argb" is TYPE_INT_ARGB, an image.NRGBA; "gray" is TYPE_BYTE_GRAY, an
// image.Gray.
func patternImage(w, h int, kind string) goimage.Image {
	bounds := goimage.Rect(0, 0, w, h)
	rgba := goimage.NewRGBA(bounds)
	nrgba := goimage.NewNRGBA(bounds)
	gray := goimage.NewGray(bounds)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b := uint8(x*37+y*11), uint8(x*x+y*51), uint8((x^y)*29)
			rgba.SetRGBA(x, y, goimagecolor.RGBA{R: r, G: g, B: b, A: 0xFF})
			nrgba.SetNRGBA(x, y, goimagecolor.NRGBA{R: r, G: g, B: b, A: uint8(64 + (x*40+y*13)&0xbf)})
			gray.SetGray(x, y, goimagecolor.Gray{Y: uint8(x*31 + y*47)})
		}
	}
	switch kind {
	case "argb":
		return nrgba
	case "gray":
		return gray
	}
	return rgba
}

// pdfRect is a rectangle as PDFBox builds one, four lines and a close, in a
// float path: Java2D fills a Rectangle2D through a third filler again, so the
// shape's class is part of the case.
func pdfRect(x0, y0, x1, y1 float32) geom.Shape {
	path := geom.NewPathFloat()
	path.MoveTo(float64(x0), float64(y0))
	path.LineTo(float64(x1), float64(y0))
	path.LineTo(float64(x1), float64(y1))
	path.LineTo(float64(x0), float64(y1))
	path.ClosePath()
	return path
}

// tablesAtLoad is whether the tables this package keeps between renders were
// built before any test ran. A test file's variables are initialised after the
// package's own, so this is the state at load and not after something in a
// test has touched them.
var tablesAtLoad = bicubicCoefficients != nil && mul8Table != nil

// TestThePackageTablesAreBuiltAtLoad holds the tables to being built at load
// rather than on first use.
//
// They are read by every image drawn and every pixel composited, and this
// package is a library: two pages rendered at once in one process would race
// on a table that the first of them built. The race is not hypothetical --
// building the bicubic coefficients on first use was reported by the race
// detector, which is what put this here -- and it cannot be caught by a test
// that renders anything first, because the table is built by then.
func TestThePackageTablesAreBuiltAtLoad(t *testing.T) {
	if !tablesAtLoad {
		t.Error("a table this package keeps is built on first use, which two " +
			"renders at once would race on; build it at load")
	}
}
