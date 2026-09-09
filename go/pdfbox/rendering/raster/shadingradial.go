package raster

// Port of RadialShadingContext: shading type 3.
//
// The parameter at a point is the solution of Adobe's Technical Note #5600
// quadratic -- the two circles the shading interpolates between, and the value
// of s at which the circle through the point lies.

import (
	goimage "image"
	goimagecolor "image/color"
	"math"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

type radialContext struct {
	baseContext
	rat              *geom.AffineTransform
	coords           []float32
	x1x0, y1y0, r1r0 float64
	r0pow2, denom    float64
	domain           [2]float32
	extend           [2]bool
	factor           int
	colorTable       []goimagecolor.NRGBA
}

func newRadialContext(sh shadingModel, matrix *util.Matrix, xform *geom.AffineTransform,
	bounds goimage.Rectangle) (shadingContext, error) {
	base, err := newBaseContext(sh)
	if err != nil {
		return nil, err
	}
	coordsArray := sh.Dictionary().GetCOSArray(cosCoords)
	if coordsArray == nil || coordsArray.Size() < 6 {
		return nil, errNoCoords
	}
	coords := coordsArray.ToFloatArray()

	c := &radialContext{
		baseContext: base,
		rat:         deviceToShading(matrix, xform),
		coords:      coords,
		x1x0:        float64(coords[3] - coords[0]),
		y1y0:        float64(coords[4] - coords[1]),
		r1r0:        float64(coords[5] - coords[2]),
		r0pow2:      float64(coords[2]) * float64(coords[2]),
		domain:      domainOf(sh.Dictionary().GetCOSArray(cosDomain)),
		extend:      extendOf(sh.Dictionary().GetCOSArray(cosExtend)),
		factor:      factorOf(bounds),
	}
	c.denom = c.x1x0*c.x1x0 + c.y1y0*c.y1y0 - c.r1r0*c.r1r0

	if c.colorTable, err = colorTableOf(sh, base, c.domain, c.factor); err != nil {
		return nil, err
	}
	return c, nil
}

// inputValues is calculateInputValues: the two roots of the quadratic, in the
// order Java answers them, which depends on the sign of the denominator.
func (c *radialContext) inputValues(x, y float64) (float32, float32) {
	p := -(x-float64(c.coords[0]))*c.x1x0 - (y-float64(c.coords[1]))*c.y1y0 -
		float64(c.coords[2])*c.r1r0
	q := (x-float64(c.coords[0]))*(x-float64(c.coords[0])) +
		(y-float64(c.coords[1]))*(y-float64(c.coords[1])) - c.r0pow2
	root := math.Sqrt(p*p - c.denom*q)
	root1 := float32((-p + root) / c.denom)
	root2 := float32((-p - root) / c.denom)
	if c.denom < 0 {
		return root1, root2
	}
	return root2, root1
}

func (c *radialContext) colorAt(x, y int) (goimagecolor.NRGBA, bool) {
	px, py := transformedPoint(c.rat, x, y)
	first, second := c.inputValues(px, py)

	if isNaN(first) && isNaN(second) {
		return c.outside()
	}

	var inputValue float32
	inFirst := first >= 0 && first <= 1
	inSecond := second >= 0 && second <= 1
	switch {
	case inFirst && inSecond:
		// both values are in the range -> choose the larger one
		inputValue = maxFloat32(first, second)
	case inFirst:
		inputValue = first
	case inSecond:
		inputValue = second
	case c.extend[0] && c.extend[1]:
		inputValue = maxFloat32(first, second)
	case c.extend[0]:
		inputValue = first
	case c.extend[1]:
		inputValue = second
	default:
		return c.outside()
	}

	switch {
	case inputValue > 1:
		// extend shading if extend[1] is true and nonzero radius
		if !c.extend[1] || c.coords[5] <= 0 {
			return c.outside()
		}
		inputValue = 1
	case inputValue < 0:
		if !c.extend[0] || c.coords[2] <= 0 {
			return c.outside()
		}
		inputValue = 0
	}
	return c.colorTable[int(inputValue*float32(c.factor))], true
}

func isNaN(v float32) bool { return v != v }

func maxFloat32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
