package raster

// The three analytic shadings: function-based, axial and radial.
//
// Port of Type1ShadingContext, AxialShadingContext and RadialShadingContext.
// Each answers a colour for a device pixel by mapping it back into the
// shading's own space and evaluating the shading's function there.
//
// Two things they share, and both are Java's rather than obvious:
//
//   - **The colour is quantised through a table.** `factor` is the diagonal of
//     the device bounds rounded up, the table holds `factor + 1` colours
//     evaluated across the domain, and the pixel reads
//     `colorTable[(int)(inputValue * factor)]`. So the same shading over a
//     different sized surface quantises differently.
//   - **`convertToRGB` truncates**, and reaches 255 only at exactly 1.0.

import (
	"errors"
	goimage "image"
	goimagecolor "image/color"
	"math"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common/function"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/shading"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// shadingModel is what every context needs of the shading it draws, which the
// two halves of PDShading give: the colour space to convert through, the
// background to paint outside it, and the function to evaluate.
type shadingModel interface {
	shading.Shading
	ColorSpace() (color.PDColorSpace, error)
	Background() *cos.Array
	EvalFunction(inputValue float32) ([]float32, error)
	EvalFunctionOfInput(input []float32) ([]float32, error)
	Function() (function.PDFunction, error)
}

// baseContext is ShadingContext: the colour space every context converts
// through, and the background it paints where the shading does not reach.
type baseContext struct {
	colorSpace color.PDColorSpace
	background []float32
	rgbBack    goimagecolor.NRGBA
	hasBack    bool
}

func newBaseContext(sh shadingModel) (baseContext, error) {
	colorSpace, err := sh.ColorSpace()
	if err != nil {
		return baseContext{}, err
	}
	base := baseContext{colorSpace: colorSpace}
	if array := sh.Background(); array != nil {
		base.background = array.ToFloatArray()
		base.rgbBack, err = convertToRGB(colorSpace, base.background)
		if err != nil {
			return baseContext{}, err
		}
		base.hasBack = true
	}
	return base, nil
}

// outside is what a context answers where the shading does not reach: the
// background where there is one, and nothing where there is not -- which is
// Java's `continue`, leaving the pixel as it was.
func (b baseContext) outside() (goimagecolor.NRGBA, bool) {
	return b.rgbBack, b.hasBack
}

// deviceToShading is the `rat` every context builds: the transform that takes
// a device pixel back to the shading's own space.
//
//	rat = inverse(matrix); rat.concatenate(inverse(xform))
//
// which is inverse(xform ∘ matrix). Type 1 puts the shading's own /Matrix in
// front of it as well.
func deviceToShading(matrix *util.Matrix, xform *geom.AffineTransform) *geom.AffineTransform {
	rat, err := matrix.CreateAffineTransform().CreateInverse()
	if err != nil {
		// Java logs and carries on with the identity.
		return geom.NewIdentityTransform()
	}
	inverseXform, err := xform.CreateInverse()
	if err != nil {
		return geom.NewIdentityTransform()
	}
	rat.Concatenate(inverseXform)
	return rat
}

// factorOf is `(int) Math.ceil(dist)` over the diagonal of the device bounds,
// which is how many steps the colour table has.
func factorOf(bounds goimage.Rectangle) int {
	dx := float64(bounds.Dx())
	dy := float64(bounds.Dy())
	return int(math.Ceil(math.Sqrt(dx*dx + dy*dy)))
}

// transformedPoint maps a device pixel through a transform.
func transformedPoint(at *geom.AffineTransform, x, y int) (float64, float64) {
	point := at.Transform(geom.NewPointDouble(float64(x), float64(y)), nil)
	return point.X(), point.Y()
}

// colorTableOf is calcColorTable: `factor + 1` colours evaluated across the
// domain, or one colour where there are no steps to take.
func colorTableOf(sh shadingModel, base baseContext, domain [2]float32,
	factor int) ([]goimagecolor.NRGBA, error) {
	d1d0 := domain[1] - domain[0]
	table := make([]goimagecolor.NRGBA, factor+1)
	if factor == 0 || d1d0 == 0 {
		values, err := sh.EvalFunction(domain[0])
		if err != nil {
			return nil, err
		}
		table[0], err = convertToRGB(base.colorSpace, values)
		if err != nil {
			return nil, err
		}
		return table, nil
	}
	for i := 0; i <= factor; i++ {
		t := domain[0] + d1d0*float32(i)/float32(factor)
		values, err := sh.EvalFunction(t)
		if err != nil {
			return nil, err
		}
		table[i], err = convertToRGB(base.colorSpace, values)
		if err != nil {
			return nil, err
		}
	}
	return table, nil
}

// domainOf reads /Domain, which defaults to [0 1].
func domainOf(array *cos.Array) [2]float32 {
	if array != nil && array.Size() >= 2 {
		values := array.ToFloatArray()
		return [2]float32{values[0], values[1]}
	}
	return [2]float32{0, 1}
}

// extendOf reads /Extend, which defaults to [false false].
func extendOf(array *cos.Array) [2]bool {
	if array != nil && array.Size() >= 2 {
		first, _ := array.GetObject(0).(*cos.Boolean)
		second, _ := array.GetObject(1).(*cos.Boolean)
		return [2]bool{first != nil && first.Value(), second != nil && second.Value()}
	}
	return [2]bool{false, false}
}

// The dictionary keys the three read directly, which the model does not
// expose for every type.
var (
	cosCoords = cos.Coords
	cosDomain = cos.Domain
	cosExtend = cos.Extend
)

// errNoCoords is what a shading with no /Coords answers. Java lets the null
// array through and throws NullPointerException on the first read; the port
// refuses, because a shading paint that cannot be built is an error the
// caller of Fill can report.
var errNoCoords = errors.New("raster: the shading has no /Coords")
