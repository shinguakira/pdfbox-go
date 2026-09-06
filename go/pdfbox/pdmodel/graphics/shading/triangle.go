package shading

// The geometry a triangle mesh shading decomposes into.
//
// Port of Vertex, CoordinateColorPair, Line and ShadedTriangle. Java gives them
// a file each and declares all four package-private; they are a few dozen lines
// apiece and only the mesh shadings use them, so the port keeps them together
// and unexported.

import (
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
)

// vertex is one point of a mesh with the colour recorded there.
//
// Port of Vertex.
type vertex struct {
	point geom.Point2D
	color []float32
}

// newVertex returns a vertex at the given point, copying the colour the way
// Java's constructor clones it.
func newVertex(p geom.Point2D, c []float32) *vertex {
	return &vertex{point: p, color: append([]float32(nil), c...)}
}

// String is Vertex.toString.
func (v *vertex) String() string {
	var sb strings.Builder
	for _, f := range v.color {
		if sb.Len() > 0 {
			sb.WriteByte(' ')
		}
		fmt.Fprintf(&sb, "%3.2f", f)
	}
	return fmt.Sprintf("Vertex{ %v, colors=[%s] }", v.point, sb.String())
}

// coordinateColorPair is a point of a patch with the colour there.
//
// Port of CoordinateColorPair.
type coordinateColorPair struct {
	coordinate geom.Point2D
	color      []float32
}

// newCoordinateColorPair returns a pair, copying the colour as Java clones it.
func newCoordinateColorPair(p geom.Point2D, c []float32) *coordinateColorPair {
	return &coordinateColorPair{coordinate: p, color: append([]float32(nil), c...)}
}

// integerPoint is java.awt.Point, the integer point Line rasterises onto. It is
// a map key here, which is what Java's HashSet<Point> makes of it.
type integerPoint struct{ x, y int }

// line is a triangle that has collapsed onto a line, rasterised to the points
// it covers.
//
// Port of Line.
type line struct {
	point0     integerPoint
	point1     integerPoint
	color0     []float32
	color1     []float32
	linePoints map[integerPoint]bool
}

// newLine returns the rasterised line between two points.
func newLine(p0, p1 integerPoint, c0, c1 []float32) *line {
	l := &line{
		point0: p0,
		point1: p1,
		color0: append([]float32(nil), c0...),
		color1: append([]float32(nil), c1...),
	}
	l.linePoints = calcLine(p0.x, p0.y, p1.x, p1.y)
	return l
}

// calcLine walks the line with Bresenham's algorithm and collects every point
// it covers.
//
// Port of the private Line.calcLine.
func calcLine(x0, y0, x1, y1 int) map[integerPoint]bool {
	points := map[integerPoint]bool{}
	dx := absInt(x1 - x0)
	dy := absInt(y1 - y0)
	sx, sy := -1, -1
	if x0 < x1 {
		sx = 1
	}
	if y0 < y1 {
		sy = 1
	}
	err := dx - dy
	for {
		points[integerPoint{x0, y0}] = true
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
	return points
}

// calcColor interpolates the colour at a point of the line.
//
// Port of Line.calcColor.
func (l *line) calcColor(p integerPoint) []float32 {
	if l.point0.x == l.point1.x && l.point0.y == l.point1.y {
		return l.color0
	}
	numberOfColorComponents := len(l.color0)
	pc := make([]float32, numberOfColorComponents)
	if l.point0.x == l.point1.x {
		length := float32(l.point1.y - l.point0.y)
		for i := 0; i < numberOfColorComponents; i++ {
			pc[i] = l.color0[i]*float32(l.point1.y-p.y)/length +
				l.color1[i]*float32(p.y-l.point0.y)/length
		}
		return pc
	}
	length := float32(l.point1.x - l.point0.x)
	for i := 0; i < numberOfColorComponents; i++ {
		pc[i] = l.color0[i]*float32(l.point1.x-p.x)/length +
			l.color1[i]*float32(p.x-l.point0.x)/length
	}
	return pc
}

// shadedTriangle is one triangle of a mesh, with a colour at each corner.
//
// Port of ShadedTriangle.
type shadedTriangle struct {
	// corner and color are protected fields in Java, which the shading types
	// read when they chain one triangle onto the last.
	corner [3]geom.Point2D
	color  [3][]float32

	area float64
	// degree is 3 for a normal triangle, 2 where it has collapsed to a line
	// and 1 where it has collapsed to a point.
	degree int
	// line describes the rasterised line where the triangle collapsed to one,
	// and is nil otherwise.
	line *line
	// v0, v1 and v2 are each corner's opposite edge equation value.
	v0, v1, v2 float64
}

// newShadedTriangle returns a triangle over three corners and their colours.
func newShadedTriangle(p [3]geom.Point2D, c [3][]float32) *shadedTriangle {
	t := &shadedTriangle{corner: p, color: c}
	t.area = triangleArea(p[0], p[1], p[2])
	t.degree = calcDegree(p)
	if t.degree == 2 {
		corner0, corner1, corner2 := t.corner[0], t.corner[1], t.corner[2]
		if pointsOverlap(corner1, corner2) && !pointsOverlap(corner0, corner2) {
			t.line = newLine(roundPoint(corner0), roundPoint(corner2), t.color[0], t.color[2])
		} else {
			t.line = newLine(roundPoint(corner1), roundPoint(corner2), t.color[1], t.color[2])
		}
	}
	t.v0 = edgeEquationValue(p[0], p[1], p[2])
	t.v1 = edgeEquationValue(p[1], p[2], p[0])
	t.v2 = edgeEquationValue(p[2], p[0], p[1])
	return t
}

// calcDegree counts how many distinct corners the triangle has, to a thousandth
// of a unit, which is what Java's set of rounded points does.
//
// Port of the private ShadedTriangle.calcDeg.
func calcDegree(p [3]geom.Point2D) int {
	set := map[integerPoint]bool{}
	for _, itp := range p {
		set[integerPoint{
			x: int(math.Round(itp.X() * 1000)),
			y: int(math.Round(itp.Y() * 1000)),
		}] = true
	}
	return len(set)
}

// Deg returns the degree of the triangle.
func (t *shadedTriangle) Deg() int { return t.degree }

// Boundary returns the integer bounding box as the four values minX, maxX,
// minY, maxY, which is the order Java fills its array in.
func (t *shadedTriangle) Boundary() [4]int {
	x0 := int(math.Round(t.corner[0].X()))
	x1 := int(math.Round(t.corner[1].X()))
	x2 := int(math.Round(t.corner[2].X()))
	y0 := int(math.Round(t.corner[0].Y()))
	y1 := int(math.Round(t.corner[1].Y()))
	y2 := int(math.Round(t.corner[2].Y()))
	return [4]int{
		minInt(minInt(x0, x1), x2),
		maxInt(maxInt(x0, x1), x2),
		minInt(minInt(y0, y1), y2),
		maxInt(maxInt(y0, y1), y2),
	}
}

// Line returns the rasterised line of a collapsed triangle, or nil.
func (t *shadedTriangle) Line() *line { return t.line }

// Contains reports whether the point is in the triangle, counting a point on an
// edge as contained.
func (t *shadedTriangle) Contains(p geom.Point2D) bool {
	if t.degree == 1 {
		return pointsOverlap(t.corner[0], p) || pointsOverlap(t.corner[1], p) ||
			pointsOverlap(t.corner[2], p)
	}
	if t.degree == 2 {
		return t.line.linePoints[roundPoint(p)]
	}
	// If corner[0] and point p are on different sides of the line from
	// corner[1] to corner[2], p is outside of the triangle.
	if edgeEquationValue(p, t.corner[1], t.corner[2])*t.v0 < 0 {
		return false
	}
	// If vertex corner[1] and point p are on different sides of the line from
	// corner[2] to corner[0], p is outside of the triangle.
	if edgeEquationValue(p, t.corner[2], t.corner[0])*t.v1 < 0 {
		return false
	}
	// Only one case left: if corner[1] and point p are on different sides of
	// the line from corner[2] to corner[0], p is outside of the triangle,
	// otherwise p is contained in it.
	return edgeEquationValue(p, t.corner[0], t.corner[1])*t.v2 >= 0
}

// pointsOverlap reports whether two points are the same to within a
// thousandth, which is the accuracy Java uses since the coordinates are
// doubles.
func pointsOverlap(p0, p1 geom.Point2D) bool {
	return math.Abs(p0.X()-p1.X()) < 0.001 && math.Abs(p0.Y()-p1.Y()) < 0.001
}

// edgeEquationValue plugs p into the line equation through p1 and p2, arranged
// so that the right hand side is zero.
func edgeEquationValue(p, p1, p2 geom.Point2D) float64 {
	return (p2.Y()-p1.Y())*(p.X()-p1.X()) - (p2.X()-p1.X())*(p.Y()-p1.Y())
}

// triangleArea is the area of the triangle through the three points.
func triangleArea(a, b, c geom.Point2D) float64 {
	return math.Abs((c.X()-b.X())*(c.Y()-a.Y())-(c.X()-a.X())*(c.Y()-b.Y())) / 2.0
}

// roundPoint is `new Point((int) Math.round(p.getX()), (int) Math.round(p.getY()))`.
func roundPoint(p geom.Point2D) integerPoint {
	return integerPoint{x: int(math.Round(p.X())), y: int(math.Round(p.Y()))}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// bitReader reads a stream one run of bits at a time.
//
// Java reads a mesh through javax.imageio.stream.ImageInputStream, for
// readBits and getBitOffset. Go has no such reader; pdmodel/common/function and
// graphics/image each keep their own for the same reason, and this is the
// third, with the bit offset the vertex padding needs.
type bitReader struct {
	source    io.Reader
	current   byte
	bitOffset int
	loaded    bool
}

// newBitReader returns a reader over the given stream.
func newBitReader(source io.Reader) *bitReader { return &bitReader{source: source} }

// readBits reads the given number of bits, most significant first, and reports
// io.EOF where the stream ends part way through -- which is how every mesh
// reader here learns that it has run out of vertices.
func (b *bitReader) readBits(numBits int) (int64, error) {
	var value int64
	for i := 0; i < numBits; i++ {
		if !b.loaded || b.bitOffset == 8 {
			buf := make([]byte, 1)
			if _, err := io.ReadFull(b.source, buf); err != nil {
				if err == io.ErrUnexpectedEOF {
					err = io.EOF
				}
				return 0, err
			}
			b.current = buf[0]
			b.bitOffset = 0
			b.loaded = true
		}
		bit := (b.current >> (7 - b.bitOffset)) & 1
		b.bitOffset++
		value = value<<1 | int64(bit)
	}
	return value, nil
}

// getBitOffset returns how far into the current byte the reader is, which is
// what ImageInputStream.getBitOffset answers: zero when it sits on a boundary.
func (b *bitReader) getBitOffset() int {
	if !b.loaded || b.bitOffset == 8 {
		return 0
	}
	return b.bitOffset
}
