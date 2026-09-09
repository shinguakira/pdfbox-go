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
// This backend renders the geometry as given, so `name` is what it is held to.
// What `nameNorm` would cost is measured in TestTheStrokeNormalizationCost,
// which is the only place that difference lives.

import (
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
			delta := int(i.dst.RGBAAt(x, y).R) - int(grid[y][x])
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
// Eleven of the seventeen match exactly, and they are the ones that decide
// whether the port is right: the fills, the clip, the transform, both winding
// rules, the butt and square caps, the miter join, and the dashes with and
// without a phase.
//
// The other six differ, and only along edges that are neither horizontal nor
// vertical. Two rasterisers are at work -- Java2D's is Marlin, which samples a
// pixel on an 8x8 subpixel grid and truncates the count to a byte, and this
// backend's is freetype's, which integrates the area exactly. On an
// exactly-half-covered pixel Marlin gives 0x7f and freetype 0x80, and that one
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

// TestAgainstJava2DNormalized is the same seventeen under the other value of
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
// merely close: the same seventeen shapes, drawn twice, against a Java2D told
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
