package shading

// The patch geometry a Coons or tensor-product mesh decomposes into.
//
// Port of Patch, CoonsPatch, TensorPatch and CubicBezierCurve. Java gives them
// a file each and declares all four package-private; only the two patch mesh
// shadings use them, so the port keeps them together and unexported.

import (
	"fmt"
	"math"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
)

// cubicBezierCurve is one edge of a patch, sampled at a level of subdivision.
//
// Port of CubicBezierCurve.
type cubicBezierCurve struct {
	controlPoints []geom.Point2D
	level         int
	curve         []geom.Point2D
}

// newCubicBezierCurve samples the curve through the four control points at the
// given level.
func newCubicBezierCurve(ctrlPnts []geom.Point2D, l int) *cubicBezierCurve {
	c := &cubicBezierCurve{
		controlPoints: append([]geom.Point2D(nil), ctrlPnts...),
		level:         l,
	}
	c.curve = c.getPoints(c.level)
	return c
}

// Level returns the level of subdivision.
func (c *cubicBezierCurve) Level() int { return c.level }

// getPoints calculates the sampled points on the cubic Bezier curve defined by
// the four control points.
func (c *cubicBezierCurve) getPoints(l int) []geom.Point2D {
	if l < 0 {
		l = 0
	}
	sz := (1 << l) + 1
	res := make([]geom.Point2D, sz)
	step := 1.0 / float64(sz-1)
	t := -step
	for i := 0; i < sz; i++ {
		t += step
		tmpX := (1-t)*(1-t)*(1-t)*c.controlPoints[0].X() +
			3*t*(1-t)*(1-t)*c.controlPoints[1].X() +
			3*t*t*(1-t)*c.controlPoints[2].X() +
			t*t*t*c.controlPoints[3].X()
		tmpY := (1-t)*(1-t)*(1-t)*c.controlPoints[0].Y() +
			3*t*(1-t)*(1-t)*c.controlPoints[1].Y() +
			3*t*t*(1-t)*c.controlPoints[2].Y() +
			t*t*t*c.controlPoints[3].Y()
		res[i] = geom.NewPointDouble(tmpX, tmpY)
	}
	return res
}

// CubicBezierCurve returns the sampled points.
func (c *cubicBezierCurve) CubicBezierCurve() []geom.Point2D { return c.curve }

// String is CubicBezierCurve.toString.
func (c *cubicBezierCurve) String() string {
	var sb strings.Builder
	for _, p := range c.controlPoints {
		if sb.Len() > 0 {
			sb.WriteByte(' ')
		}
		fmt.Fprintf(&sb, "%v", p)
	}
	return "Cubic Bezier curve{control points p0, p1, p2, p3: " + sb.String() + "}"
}

// patchShape is what a concrete patch supplies.
//
// Patch is an abstract class in Java with three abstract methods; the port
// splits it into this interface and the embedded struct below.
type patchShape interface {
	// getFlag1Edge, getFlag2Edge and getFlag3Edge are the edge a following
	// patch inherits for each of the three joining flags.
	getFlag1Edge() []geom.Point2D
	getFlag2Edge() []geom.Point2D
	getFlag3Edge() []geom.Point2D
}

// patch carries the state and the concrete methods of a patch.
//
// Port of the non-abstract half of Patch.
type patch struct {
	controlPoints [][]geom.Point2D
	cornerColor   [][]float32
	// level is {levelU, levelV}: levelU says the patch's u direction edges are
	// divided into 2^levelU parts, levelV the same for v.
	level           [2]int
	listOfTriangles []*shadedTriangle
}

// initPatch is the Patch(float[][]) constructor, which clones the colours.
func (p *patch) initPatch(color [][]float32) {
	p.cornerColor = append([][]float32(nil), color...)
}

// getFlag1Color is the colour pair a patch joined with flag 1 inherits.
func (p *patch) getFlag1Color() [][]float32 {
	return p.implicitCornerColor(1, 2)
}

// getFlag2Color is the colour pair a patch joined with flag 2 inherits.
func (p *patch) getFlag2Color() [][]float32 {
	return p.implicitCornerColor(2, 3)
}

// getFlag3Color is the colour pair a patch joined with flag 3 inherits.
func (p *patch) getFlag3Color() [][]float32 {
	return p.implicitCornerColor(3, 0)
}

// implicitCornerColor is the body the three getFlagNColor share, which Java
// writes out three times over a different pair of corners.
func (p *patch) implicitCornerColor(first, second int) [][]float32 {
	numberOfColorComponents := len(p.cornerColor[0])
	implicitCornerColor := [][]float32{
		make([]float32, numberOfColorComponents),
		make([]float32, numberOfColorComponents),
	}
	for i := 0; i < numberOfColorComponents; i++ {
		implicitCornerColor[0][i] = p.cornerColor[first][i]
		implicitCornerColor[1][i] = p.cornerColor[second][i]
	}
	return implicitCornerColor
}

// getLen is the distance between two points.
func getLen(ps, pe geom.Point2D) float64 {
	x := pe.X() - ps.X()
	y := pe.Y() - ps.Y()
	return math.Sqrt(x*x + y*y)
}

// isEdgeALine reports whether the four control points of an edge are close
// enough to straight to be treated as a line.
func isEdgeALine(ctl []geom.Point2D) bool {
	ctl1 := math.Abs(edgeEquationValue(ctl[1], ctl[0], ctl[3]))
	ctl2 := math.Abs(edgeEquationValue(ctl[2], ctl[0], ctl[3]))
	x := math.Abs(ctl[0].X() - ctl[3].X())
	y := math.Abs(ctl[0].Y() - ctl[3].Y())
	return (ctl1 <= x && ctl2 <= x) || (ctl1 <= y && ctl2 <= y)
}

// getShadedTriangles cuts the grid of coordinates and colours into triangles.
func (p *patch) getShadedTriangles(patchCC [][]*coordinateColorPair) []*shadedTriangle {
	var list []*shadedTriangle
	szV := len(patchCC)
	szU := len(patchCC[0])
	for i := 1; i < szV; i++ {
		for j := 1; j < szU; j++ {
			p0 := patchCC[i-1][j-1].coordinate
			p1 := patchCC[i-1][j].coordinate
			p2 := patchCC[i][j].coordinate
			p3 := patchCC[i][j-1].coordinate
			ll := true
			if pointsOverlap(p0, p1) || pointsOverlap(p0, p3) {
				ll = false
			} else {
				// p0, p1 and p3 are in counter clock wise order, p1 has
				// priority over p0, p3 has priority over p1
				list = append(list, newShadedTriangle(
					[3]geom.Point2D{p0, p1, p3},
					[3][]float32{
						patchCC[i-1][j-1].color,
						patchCC[i-1][j].color,
						patchCC[i][j-1].color,
					}))
			}
			if ll && (pointsOverlap(p2, p1) || pointsOverlap(p2, p3)) {
				// Java's if body is empty here: the upper right triangle is
				// dropped only when the lower left one was kept and this one
				// has collapsed.
				continue
			}
			// p3, p1 and p2 are in counter clock wise order, p1 has priority
			// over p3, p2 has priority over p1
			list = append(list, newShadedTriangle(
				[3]geom.Point2D{p3, p1, p2},
				[3][]float32{
					patchCC[i][j-1].color,
					patchCC[i-1][j].color,
					patchCC[i][j].color,
				}))
		}
	}
	return list
}

// clonedPointSlice is the static Patch.clonedPoint2DArray.
func clonedPointSlice(input []geom.Point2D) []geom.Point2D {
	cloned := make([]geom.Point2D, len(input))
	for i, p := range input {
		cloned[i] = geom.NewPointDouble(p.X(), p.Y())
	}
	return cloned
}

// dividingLevelOf is the run of length tests the two patches share: an edge
// pair short enough divides into fewer parts.
//
// Java writes this ladder out four times, twice in each patch, with the same
// thresholds each time.
func dividingLevelOf(l1, l2 float64) int {
	switch {
	case l1 > 800 || l2 > 800:
		return 4 // keeps init value 4
	case l1 > 400 || l2 > 400:
		return 3
	case l1 > 200 || l2 > 200:
		return 2
	}
	return 1
}

// coonsPatch is a Coons patch: four cubic Bezier edges and four corner colours.
//
// Port of CoonsPatch.
type coonsPatch struct {
	patch
}

var _ patchShape = (*coonsPatch)(nil)

// newCoonsPatch builds a Coons patch from its twelve control points and four
// corner colours.
func newCoonsPatch(points []geom.Point2D, color [][]float32) *coonsPatch {
	p := &coonsPatch{}
	p.initPatch(color)
	p.controlPoints = reshapeCoonsControlPoints(points)
	p.level = p.calcLevel()
	p.listOfTriangles = p.getTriangles()
	return p
}

// reshapeCoonsControlPoints adjusts the twelve control points to four groups,
// each group defining one edge of the patch.
func reshapeCoonsControlPoints(points []geom.Point2D) [][]geom.Point2D {
	fourRows := make([][]geom.Point2D, 4)
	fourRows[2] = []geom.Point2D{points[0], points[1], points[2], points[3]}   // d1
	fourRows[1] = []geom.Point2D{points[3], points[4], points[5], points[6]}   // c2
	fourRows[3] = []geom.Point2D{points[9], points[8], points[7], points[6]}   // d2
	fourRows[0] = []geom.Point2D{points[0], points[11], points[10], points[9]} // c1
	return fourRows
}

// calcLevel calculates the dividing level from the control points.
func (p *coonsPatch) calcLevel() [2]int {
	l := [2]int{4, 4}
	// if two opposite edges are both lines, there is a possibility to reduce
	// the dividing level
	if isEdgeALine(p.controlPoints[0]) && isEdgeALine(p.controlPoints[1]) {
		lc1 := getLen(p.controlPoints[0][0], p.controlPoints[0][3])
		lc2 := getLen(p.controlPoints[1][0], p.controlPoints[1][3])
		// determine the dividing level by the lengths of edges
		l[0] = dividingLevelOf(lc1, lc2)
	}
	// the other two opposite edges
	if isEdgeALine(p.controlPoints[2]) && isEdgeALine(p.controlPoints[3]) {
		ld1 := getLen(p.controlPoints[2][0], p.controlPoints[2][3])
		ld2 := getLen(p.controlPoints[3][0], p.controlPoints[3][3])
		l[1] = dividingLevelOf(ld1, ld2)
	}
	return l
}

// getTriangles returns the triangles which compose this Coons patch.
func (p *coonsPatch) getTriangles() []*shadedTriangle {
	// 4 edges are 4 cubic Bezier curves
	eC1 := newCubicBezierCurve(p.controlPoints[0], p.level[0])
	eC2 := newCubicBezierCurve(p.controlPoints[1], p.level[0])
	eD1 := newCubicBezierCurve(p.controlPoints[2], p.level[1])
	eD2 := newCubicBezierCurve(p.controlPoints[3], p.level[1])
	return p.getShadedTriangles(p.getPatchCoordinatesColor(eC1, eC2, eD1, eD2))
}

func (p *coonsPatch) getFlag1Edge() []geom.Point2D {
	return clonedPointSlice(p.controlPoints[1])
}

func (p *coonsPatch) getFlag2Edge() []geom.Point2D {
	return []geom.Point2D{
		p.controlPoints[3][3], p.controlPoints[3][2],
		p.controlPoints[3][1], p.controlPoints[3][0],
	}
}

func (p *coonsPatch) getFlag3Edge() []geom.Point2D {
	return []geom.Point2D{
		p.controlPoints[0][3], p.controlPoints[0][2],
		p.controlPoints[0][1], p.controlPoints[0][0],
	}
}

// getPatchCoordinatesColor divides the patch into a grid and returns the
// coordinate and colour at each crossing point.
//
// The rule for the coordinate is on page 195 of PDF32000_2008.pdf; the colour
// is bilinear interpolation.
func (p *coonsPatch) getPatchCoordinatesColor(c1, c2, d1, d2 *cubicBezierCurve) [][]*coordinateColorPair {
	curveC1 := c1.CubicBezierCurve()
	curveC2 := c2.CubicBezierCurve()
	curveD1 := d1.CubicBezierCurve()
	curveD2 := d2.CubicBezierCurve()

	numberOfColorComponents := len(p.cornerColor[0])
	szV := len(curveD1)
	szU := len(curveC1)
	patchCC := make([][]*coordinateColorPair, szV)
	stepV := 1.0 / float64(szV-1)
	stepU := 1.0 / float64(szU-1)
	v := -stepV
	for i := 0; i < szV; i++ {
		// v and u are the assistant parameters
		v += stepV
		patchCC[i] = make([]*coordinateColorPair, szU)
		u := -stepU
		for j := 0; j < szU; j++ {
			u += stepU
			scx := (1-v)*curveC1[j].X() + v*curveC2[j].X()
			scy := (1-v)*curveC1[j].Y() + v*curveC2[j].Y()
			sdx := (1-u)*curveD1[i].X() + u*curveD2[i].X()
			sdy := (1-u)*curveD1[i].Y() + u*curveD2[i].Y()
			sbx := (1-v)*((1-u)*p.controlPoints[0][0].X()+u*p.controlPoints[0][3].X()) +
				v*((1-u)*p.controlPoints[1][0].X()+u*p.controlPoints[1][3].X())
			sby := (1-v)*((1-u)*p.controlPoints[0][0].Y()+u*p.controlPoints[0][3].Y()) +
				v*((1-u)*p.controlPoints[1][0].Y()+u*p.controlPoints[1][3].Y())
			sx := scx + sdx - sbx
			sy := scy + sdy - sby
			// the above defines the patch surface (coordinates)
			paramSC := make([]float32, numberOfColorComponents)
			for ci := 0; ci < numberOfColorComponents; ci++ {
				// bilinear interpolation
				paramSC[ci] = float32((1-v)*((1-u)*float64(p.cornerColor[0][ci])+
					u*float64(p.cornerColor[3][ci])) +
					v*((1-u)*float64(p.cornerColor[1][ci])+u*float64(p.cornerColor[2][ci])))
			}
			patchCC[i][j] = newCoordinateColorPair(geom.NewPointDouble(sx, sy), paramSC)
		}
	}
	return patchCC
}

// tensorPatch is a tensor-product patch: sixteen control points and four corner
// colours.
//
// Port of TensorPatch.
type tensorPatch struct {
	patch
}

var _ patchShape = (*tensorPatch)(nil)

// newTensorPatch builds a tensor-product patch from its sixteen control points
// and four corner colours.
func newTensorPatch(tcp []geom.Point2D, color [][]float32) *tensorPatch {
	p := &tensorPatch{}
	p.initPatch(color)
	p.controlPoints = reshapeTensorControlPoints(tcp)
	p.level = p.calcLevel()
	p.listOfTriangles = p.getTriangles()
	return p
}

// reshapeTensorControlPoints orders the sixteen points into the square matrix
// described on page 199 of PDF32000_2008.pdf, rotated 90 degrees clockwise.
func reshapeTensorControlPoints(tcp []geom.Point2D) [][]geom.Point2D {
	square := make([][]geom.Point2D, 4)
	for i := range square {
		square[i] = make([]geom.Point2D, 4)
	}
	for i := 0; i <= 3; i++ {
		square[0][i] = tcp[i]
		square[3][i] = tcp[9-i]
	}
	for i := 1; i <= 2; i++ {
		square[i][0] = tcp[12-i]
		square[i][2] = tcp[12+i]
		square[i][3] = tcp[3+i]
	}
	square[1][1] = tcp[12]
	square[2][1] = tcp[15]
	return square
}

// calcLevel calculates the dividing level from the control points.
func (p *tensorPatch) calcLevel() [2]int {
	l := [2]int{4, 4}
	ctlC1 := make([]geom.Point2D, 4)
	ctlC2 := make([]geom.Point2D, 4)
	for j := 0; j < 4; j++ {
		ctlC1[j] = p.controlPoints[j][0]
		ctlC2[j] = p.controlPoints[j][3]
	}
	// if two opposite edges are both lines, there is a possibility to reduce
	// the dividing level
	if isEdgeALine(ctlC1) && isEdgeALine(ctlC2) {
		// if any of the 4 inner control points is out of the patch formed by
		// the 4 edges, keep the high dividing level, otherwise determine it by
		// the lengths of the edges
		if !(p.isOnSameSideCC(p.controlPoints[1][1]) || p.isOnSameSideCC(p.controlPoints[1][2]) ||
			p.isOnSameSideCC(p.controlPoints[2][1]) || p.isOnSameSideCC(p.controlPoints[2][2])) {
			// length's unit is one pixel in device space
			l[0] = dividingLevelOf(getLen(ctlC1[0], ctlC1[3]), getLen(ctlC2[0], ctlC2[3]))
		}
	}
	// the other two opposite edges
	if isEdgeALine(p.controlPoints[0]) && isEdgeALine(p.controlPoints[3]) {
		if !(p.isOnSameSideDD(p.controlPoints[1][1]) || p.isOnSameSideDD(p.controlPoints[1][2]) ||
			p.isOnSameSideDD(p.controlPoints[2][1]) || p.isOnSameSideDD(p.controlPoints[2][2])) {
			ld1 := getLen(p.controlPoints[0][0], p.controlPoints[0][3])
			ld2 := getLen(p.controlPoints[3][0], p.controlPoints[3][3])
			l[1] = dividingLevelOf(ld1, ld2)
		}
	}
	return l
}

// isOnSameSideCC reports whether a point is on the same side of edge C1 and
// edge C2.
func (p *tensorPatch) isOnSameSideCC(pt geom.Point2D) bool {
	cc := edgeEquationValue(pt, p.controlPoints[0][0], p.controlPoints[3][0]) *
		edgeEquationValue(pt, p.controlPoints[0][3], p.controlPoints[3][3])
	return cc > 0
}

// isOnSameSideDD reports whether a point is on the same side of edge D1 and
// edge D2.
func (p *tensorPatch) isOnSameSideDD(pt geom.Point2D) bool {
	dd := edgeEquationValue(pt, p.controlPoints[0][0], p.controlPoints[0][3]) *
		edgeEquationValue(pt, p.controlPoints[3][0], p.controlPoints[3][3])
	return dd > 0
}

// getTriangles returns the triangles which compose this tensor patch.
func (p *tensorPatch) getTriangles() []*shadedTriangle {
	return p.getShadedTriangles(p.getPatchCoordinatesColor())
}

func (p *tensorPatch) getFlag1Edge() []geom.Point2D {
	implicitEdge := make([]geom.Point2D, 4)
	for i := 0; i < 4; i++ {
		implicitEdge[i] = p.controlPoints[i][3]
	}
	return implicitEdge
}

func (p *tensorPatch) getFlag2Edge() []geom.Point2D {
	implicitEdge := make([]geom.Point2D, 4)
	for i := 0; i < 4; i++ {
		implicitEdge[i] = p.controlPoints[3][3-i]
	}
	return implicitEdge
}

func (p *tensorPatch) getFlag3Edge() []geom.Point2D {
	implicitEdge := make([]geom.Point2D, 4)
	for i := 0; i < 4; i++ {
		implicitEdge[i] = p.controlPoints[3-i][0]
	}
	return implicitEdge
}

// getPatchCoordinatesColor divides the patch into a grid according to the level
// and calculates the coordinate and colour at each crossing point.
//
// The coordinate is the tensor product defined on page 119 of
// PDF32000_2008.pdf; the colour is bilinear interpolation.
func (p *tensorPatch) getPatchCoordinatesColor() [][]*coordinateColorPair {
	numberOfColorComponents := len(p.cornerColor[0])
	bernsteinPolyU := getBernsteinPolynomials(p.level[0])
	szU := len(bernsteinPolyU[0])
	bernsteinPolyV := getBernsteinPolynomials(p.level[1])
	szV := len(bernsteinPolyV[0])
	patchCC := make([][]*coordinateColorPair, szV)
	stepU := 1.0 / float64(szU-1)
	stepV := 1.0 / float64(szV-1)
	v := -stepV
	for k := 0; k < szV; k++ {
		// v and u are the assistant parameters
		v += stepV
		patchCC[k] = make([]*coordinateColorPair, szU)
		u := -stepU
		for l := 0; l < szU; l++ {
			tmpx := 0.0
			tmpy := 0.0
			// these two loops are the equation defining the patch surface
			for i := 0; i < 4; i++ {
				for j := 0; j < 4; j++ {
					tmpx += p.controlPoints[i][j].X() * bernsteinPolyU[i][l] * bernsteinPolyV[j][k]
					tmpy += p.controlPoints[i][j].Y() * bernsteinPolyU[i][l] * bernsteinPolyV[j][k]
				}
			}
			// Java advances u after the coordinate and before the colour, so
			// the two are a step apart; the port keeps that.
			u += stepU
			paramSC := make([]float32, numberOfColorComponents)
			for ci := 0; ci < numberOfColorComponents; ci++ {
				// bilinear interpolation
				paramSC[ci] = float32((1-v)*((1-u)*float64(p.cornerColor[0][ci])+
					u*float64(p.cornerColor[3][ci])) +
					v*((1-u)*float64(p.cornerColor[1][ci])+u*float64(p.cornerColor[2][ci])))
			}
			patchCC[k][l] = newCoordinateColorPair(geom.NewPointDouble(tmpx, tmpy), paramSC)
		}
	}
	return patchCC
}

// getBernsteinPolynomials returns the Bernstein polynomials defined on page 119
// of PDF32000_2008.pdf, sampled at the given level.
func getBernsteinPolynomials(lvl int) [][]float64 {
	sz := (1 << lvl) + 1
	poly := make([][]float64, 4)
	for i := range poly {
		poly[i] = make([]float64, sz)
	}
	step := 1.0 / float64(sz-1)
	t := -step
	for i := 0; i < sz; i++ {
		t += step
		poly[0][i] = (1 - t) * (1 - t) * (1 - t)
		poly[1][i] = 3 * t * (1 - t) * (1 - t)
		poly[2][i] = 3 * t * t * (1 - t)
		poly[3][i] = t * t * t
	}
	return poly
}
