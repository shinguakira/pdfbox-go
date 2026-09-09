package shading

// The pixel table of a mesh shading.
//
// Port of TriangleBasedShadingContext.calcPixelTable, which lives in this
// package in the Java too: the contexts are `graphics/shading` classes there,
// and the triangles they walk are package-private. The port keeps the raster
// half in `rendering/raster` and this half here, because this is the mesh
// arithmetic rather than the drawing -- what comes out is the shading's own
// colour components for each device pixel, and turning those into RGB is the
// backend's.

import (
	goimage "image"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// PixelTable answers the colour components of every device pixel the mesh
// covers, keyed by pixel.
//
// The colours are what the mesh carries: where the shading has a /Function
// they are the one parameter it is evaluated on, and where it does not they
// are the colour space's components. `EvalFunction` decides which, and the
// caller applies it, exactly as Java's evalFunctionAndConvertToRGB does.
func (s *pdTriangleBasedShadingType) PixelTable(xform *geom.AffineTransform,
	matrix *util.Matrix, bounds goimage.Rectangle) (map[goimage.Point][]float32, error) {
	triangles, err := s.self.collectTriangles(xform, matrix)
	if err != nil {
		return nil, err
	}

	table := map[goimage.Point][]float32{}
	for _, triangle := range triangles {
		if triangle.Deg() == 2 {
			addLinePoints(triangle.Line(), table)
			continue
		}

		boundary := triangle.Boundary()
		boundary[0] = max(boundary[0], bounds.Min.X)
		boundary[1] = min(boundary[1], bounds.Min.X+bounds.Dx())
		boundary[2] = max(boundary[2], bounds.Min.Y)
		boundary[3] = min(boundary[3], bounds.Min.Y+bounds.Dy())

		for x := boundary[0]; x <= boundary[1]; x++ {
			for y := boundary[2]; y <= boundary[3]; y++ {
				p := geom.NewPointDouble(float64(x), float64(y))
				if triangle.Contains(p) {
					table[goimage.Point{X: x, Y: y}] = triangle.CalcColor(p)
				}
			}
		}

		// "fatten" the triangle by drawing its borders with Bresenham's line
		// algorithm. Inspiration: Raph Levien in
		// http://bugs.ghostscript.com/show_bug.cgi?id=219588
		corners := [3]integerPoint{
			roundPoint(triangle.corner[0]),
			roundPoint(triangle.corner[1]),
			roundPoint(triangle.corner[2]),
		}
		addLinePoints(newLine(corners[0], corners[1], triangle.color[0], triangle.color[1]), table)
		addLinePoints(newLine(corners[1], corners[2], triangle.color[1], triangle.color[2]), table)
		addLinePoints(newLine(corners[2], corners[0], triangle.color[2], triangle.color[0]), table)
	}
	return table, nil
}

// addLinePoints is the private method of the same name: every pixel the line
// covers takes the colour interpolated along it.
func addLinePoints(l *line, table map[goimage.Point][]float32) {
	for p := range l.linePoints {
		table[goimage.Point{X: p.x, Y: p.y}] = l.calcColor(p)
	}
}
