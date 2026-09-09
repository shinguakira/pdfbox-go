package raster

// Port of Type1ShadingContext: shading type 1, the function-based one.
//
// There is no colour table here: the function takes two inputs, so there is
// nothing to tabulate along, and Java evaluates it per pixel.

import (
	goimagecolor "image/color"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

type functionContext struct {
	baseContext
	shading shadingModel
	rat     *geom.AffineTransform
	domain  [4]float32
}

func newFunctionContext(sh shadingModel, matrix *util.Matrix,
	xform *geom.AffineTransform) (shadingContext, error) {
	base, err := newBaseContext(sh)
	if err != nil {
		return nil, err
	}
	c := &functionContext{
		baseContext: base,
		shading:     sh,
		domain:      [4]float32{0, 1, 0, 1},
	}
	if array := sh.Dictionary().GetCOSArray(cosDomain); array != nil && array.Size() >= 4 {
		values := array.ToFloatArray()
		c.domain = [4]float32{values[0], values[1], values[2], values[3]}
	}

	// The shading's own /Matrix goes in front of the two the others use.
	rat := geom.NewIdentityTransform()
	if withMatrix, ok := sh.(interface{ Matrix() *util.Matrix }); ok {
		inverse, err := withMatrix.Matrix().CreateAffineTransform().CreateInverse()
		if err == nil {
			rat = inverse
		}
	}
	rat.Concatenate(deviceToShading(matrix, xform))
	c.rat = rat
	return c, nil
}

func (c *functionContext) colorAt(x, y int) (goimagecolor.NRGBA, bool) {
	px, py := transformedPoint(c.rat, x, y)
	if float32(px) < c.domain[0] || float32(px) > c.domain[1] ||
		float32(py) < c.domain[2] || float32(py) > c.domain[3] {
		return c.outside()
	}
	values, err := c.shading.EvalFunctionOfInput([]float32{float32(px), float32(py)})
	if err != nil {
		// Java logs the IOException and skips the pixel.
		return goimagecolor.NRGBA{}, false
	}
	rgb, err := convertToRGB(c.colorSpace, values)
	if err != nil {
		return goimagecolor.NRGBA{}, false
	}
	return rgb, true
}
