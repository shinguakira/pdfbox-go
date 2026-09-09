package raster

// Port of AxialShadingContext: shading type 2.

import (
	goimage "image"
	goimagecolor "image/color"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// axialContext paints a colour that varies along a line.
type axialContext struct {
	baseContext
	rat        *geom.AffineTransform
	coords     []float32
	x1x0, y1y0 float64
	denom      float64
	domain     [2]float32
	extend     [2]bool
	factor     int
	colorTable []goimagecolor.NRGBA
}

func newAxialContext(sh shadingModel, matrix *util.Matrix, xform *geom.AffineTransform,
	bounds goimage.Rectangle) (shadingContext, error) {
	base, err := newBaseContext(sh)
	if err != nil {
		return nil, err
	}
	coordsArray := sh.Dictionary().GetCOSArray(cosCoords)
	if coordsArray == nil || coordsArray.Size() < 4 {
		return nil, errNoCoords
	}
	coords := coordsArray.ToFloatArray()

	c := &axialContext{
		baseContext: base,
		rat:         deviceToShading(matrix, xform),
		coords:      coords,
		x1x0:        float64(coords[2] - coords[0]),
		y1y0:        float64(coords[3] - coords[1]),
		domain:      domainOf(sh.Dictionary().GetCOSArray(cosDomain)),
		extend:      extendOf(sh.Dictionary().GetCOSArray(cosExtend)),
		factor:      factorOf(bounds),
	}
	c.denom = c.x1x0*c.x1x0 + c.y1y0*c.y1y0

	if c.colorTable, err = colorTableOf(sh, base, c.domain, c.factor); err != nil {
		return nil, err
	}
	return c, nil
}

// colorAt is the body of getRaster, for one pixel.
func (c *axialContext) colorAt(x, y int) (goimagecolor.NRGBA, bool) {
	px, py := transformedPoint(c.rat, x, y)
	inputValue := c.x1x0*(px-float64(c.coords[0])) + c.y1y0*(py-float64(c.coords[1]))

	if c.denom == 0 {
		// TODO this happens if start == end, see PDFBOX-1442
		return c.outside()
	}
	inputValue /= c.denom

	switch {
	case inputValue < 0:
		// the shading has to be extended if extend[0] == true
		if !c.extend[0] {
			return c.outside()
		}
		inputValue = float64(c.domain[0])
	case inputValue > 1:
		if !c.extend[1] {
			return c.outside()
		}
		inputValue = float64(c.domain[1])
	}
	return c.colorTable[int(inputValue*float64(c.factor))], true
}
