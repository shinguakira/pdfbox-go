package raster

// The four mesh shadings: types 4 to 7.
//
// Port of TriangleBasedShadingContext and the four contexts over it. The mesh
// arithmetic -- which triangles, which pixels, which colour at each -- is
// `shading.PixelTable`, in the package the triangles live in, as it is in the
// Java. What is here is the half a backend owns: evaluating the shading's
// function where it has one, and converting to RGB.

import (
	goimage "image"
	goimagecolor "image/color"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// meshShading is the half of a mesh shading this context needs beyond the
// model every shading has.
type meshShading interface {
	shadingModel
	PixelTable(xform *geom.AffineTransform, matrix *util.Matrix,
		bounds goimage.Rectangle) (map[goimage.Point][]float32, error)
}

// meshContext looks a pixel up in the table the mesh built.
type meshContext struct {
	baseContext
	table map[goimage.Point]goimagecolor.NRGBA
}

func newMeshContext(sh meshShading, matrix *util.Matrix, xform *geom.AffineTransform,
	bounds goimage.Rectangle) (shadingContext, error) {
	base, err := newBaseContext(sh)
	if err != nil {
		return nil, err
	}
	components, err := sh.PixelTable(xform, matrix, bounds)
	if err != nil {
		return nil, err
	}

	// Whether the mesh's numbers are a parameter or the colour itself is
	// decided once, by whether the shading has a /Function, which is
	// evalFunctionAndConvertToRGB's own test.
	function, err := sh.Function()
	if err != nil {
		return nil, err
	}

	c := &meshContext{
		baseContext: base,
		table:       make(map[goimage.Point]goimagecolor.NRGBA, len(components)),
	}
	for point, values := range components {
		if function != nil {
			if values, err = sh.EvalFunctionOfInput(values[:1]); err != nil {
				return nil, err
			}
		}
		rgb, err := convertToRGB(base.colorSpace, values)
		if err != nil {
			return nil, err
		}
		c.table[point] = rgb
	}
	return c, nil
}

func (c *meshContext) colorAt(x, y int) (goimagecolor.NRGBA, bool) {
	if rgb, covered := c.table[goimage.Point{X: x, Y: y}]; covered {
		return rgb, true
	}
	// Java leaves the pixel alone where the mesh does not reach it, and paints
	// the background where there is one.
	return c.outside()
}
