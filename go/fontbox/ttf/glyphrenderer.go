package ttf

import (
	"fmt"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
)

// glyphRenderer provides a glyph to path conversion for true type fonts.
//
// Based on code from Apache Batik, a subproject of Apache XMLGraphics; see
// http://xmlgraphics.apache.org/batik. Contour rendering ported from PDF.js,
// viewed on 14.2.2015, rev 2e97c0d; see
// https://github.com/mozilla/pdf.js/blob/c0d17013a28ee7aa048831560b6494a26c52360c/src/core/font_renderer.js
//
// Port of org.apache.fontbox.ttf.GlyphRenderer.
type glyphRenderer struct {
	glyphDescription GlyphDescription
}

func newGlyphRenderer(glyphDescription GlyphDescription) *glyphRenderer {
	return &glyphRenderer{glyphDescription: glyphDescription}
}

// Path returns the path of the glyph.
func (r *glyphRenderer) Path() *geom.Path2D {
	points := r.describe(r.glyphDescription)
	return calculatePath(points)
}

// describe sets the points of a glyph from the GlyphDescription.
func (r *glyphRenderer) describe(gd GlyphDescription) []glyphPoint {
	endPtIndex := 0
	endPtOfContourIndex := -1
	points := make([]glyphPoint, gd.PointCount())
	for i := range points {
		if endPtOfContourIndex == -1 {
			endPtOfContourIndex = gd.EndPtOfContours(endPtIndex)
		}
		endPt := endPtOfContourIndex == i
		if endPt {
			endPtIndex++
			endPtOfContourIndex = -1
		}
		points[i] = glyphPoint{
			x:            int(gd.XCoordinate(i)),
			y:            int(gd.YCoordinate(i)),
			onCurve:      gd.Flags(i)&OnCurve != 0,
			endOfContour: endPt,
		}
	}
	return points
}

// calculatePath uses the given points to calculate a path.
func calculatePath(points []glyphPoint) *geom.Path2D {
	path := geom.NewPathFloat()
	start := 0
	for p := 0; p < len(points); p++ {
		if !points[p].endOfContour {
			continue
		}
		firstPoint := points[start]
		lastPoint := points[p]
		contour := make([]glyphPoint, 0, (p-start)+3)
		contour = append(contour, points[start:p+1]...)
		switch {
		case points[start].onCurve:
			// using start point at the contour end
			contour = append(contour, firstPoint)
		case points[p].onCurve:
			// first is off-curve point, trying to use one from the end
			contour = append([]glyphPoint{lastPoint}, contour...)
		default:
			// start and end are off-curve points, creating implicit one
			pmid := midPoint(firstPoint, lastPoint)
			contour = append([]glyphPoint{pmid}, contour...)
			contour = append(contour, pmid)
		}
		moveTo(path, contour[0])
		for j := 1; j < len(contour); j++ {
			pnow := contour[j]
			switch {
			case pnow.onCurve:
				lineTo(path, pnow)
			case contour[j+1].onCurve:
				quadTo(path, pnow, contour[j+1])
				j++
			default:
				quadTo(path, pnow, midPoint(pnow, contour[j+1]))
			}
		}
		path.ClosePath()
		start = p + 1
	}
	return path
}

func moveTo(path *geom.Path2D, point glyphPoint) {
	path.MoveTo(float64(point.x), float64(point.y))
}

func lineTo(path *geom.Path2D, point glyphPoint) {
	path.LineTo(float64(point.x), float64(point.y))
}

func quadTo(path *geom.Path2D, ctrlPoint, point glyphPoint) {
	path.QuadTo(float64(ctrlPoint.x), float64(ctrlPoint.y),
		float64(point.x), float64(point.y))
}

func midValue(a, b int) int { return a + (b-a)/2 }

// midPoint creates an onCurve point that is between point1 and point2.
func midPoint(point1, point2 glyphPoint) glyphPoint {
	// this constructs an on-curve, non-endofcontour point
	return glyphPoint{
		x:       midValue(point1.x, point2.x),
		y:       midValue(point1.y, point2.y),
		onCurve: true,
	}
}

// glyphPoint represents one point of a glyph.
type glyphPoint struct {
	x            int
	y            int
	onCurve      bool
	endOfContour bool
}

func (p glyphPoint) String() string {
	onCurve := ""
	if p.onCurve {
		onCurve = "onCurve"
	}
	endOfContour := ""
	if p.endOfContour {
		endOfContour = "endOfContour"
	}
	return fmt.Sprintf("Point(%d,%d,%s,%s)", p.x, p.y, onCurve, endOfContour)
}
