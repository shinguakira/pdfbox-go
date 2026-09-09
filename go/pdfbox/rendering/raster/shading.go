package raster

// The shading contexts: what turns a shading into pixels.
//
// Port of the eleven `*ShadingContext` classes of
// pdmodel/graphics/shading, each of which implements java.awt.PaintContext.
// Java's PaintContext hands back a Raster of a whole rectangle at a time; the
// port asks for one pixel, because the compositor walks pixels anyway and a
// shading that is asked for a rectangle has to be told which one.
//
// The model half -- what colour the shading gives for a parameter -- is
// `shading.Shading` and was ported by slice 9. What is here is the geometry
// that turns a device pixel into that parameter, and the colour table that
// quantises it.

import (
	"fmt"
	goimage "image"
	goimagecolor "image/color"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/shading"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// shadingContext answers the colour a shading paints a device pixel.
//
// The second result is false where the shading paints nothing there, which is
// Java's `continue` in the raster loop: the pixel is left as it was, and with
// no background there is nothing to put in it.
type shadingContext interface {
	colorAt(x, y int) (goimagecolor.RGBA, bool)
}

// convertToRGB is ShadingContext.convertToRGB: the shading's colour space
// converts the components to RGB, and each channel is **truncated** to a byte
// rather than rounded. `(int) (rgbValues[0] * 255)` gives 254 for 0.999 and
// reaches 255 only at exactly 1.
func convertToRGB(colorSpace color.PDColorSpace, values []float32) (goimagecolor.RGBA, error) {
	rgb, err := colorSpace.ToRGB(values)
	if err != nil {
		return goimagecolor.RGBA{}, err
	}
	return goimagecolor.RGBA{
		R: uint8(rgb[0] * 255),
		G: uint8(rgb[1] * 255),
		B: uint8(rgb[2] * 255),
		A: 0xFF,
	}, nil
}

// newShadingContext returns the context for a shading, which is
// PDShading.toPaint(Matrix) followed by createContext.
//
// matrix is the pattern matrix, xform the transform in force when the paint
// was installed, and deviceBounds the pixels that will be asked for -- Java
// sizes the colour table from its diagonal.
func newShadingContext(sh shading.Shading, matrix *util.Matrix,
	xform *geom.AffineTransform, deviceBounds goimage.Rectangle) (shadingContext, error) {
	model, ok := sh.(shadingModel)
	if !ok {
		return nil, fmt.Errorf("raster: %T is not a shading this backend can draw", sh)
	}
	switch sh.ShadingType() {
	case shading.ShadingType1:
		return newFunctionContext(model, matrix, xform)
	case shading.ShadingType2:
		return newAxialContext(model, matrix, xform, deviceBounds)
	case shading.ShadingType3:
		return newRadialContext(model, matrix, xform, deviceBounds)
	}
	return nil, ErrNotDrawn
}
