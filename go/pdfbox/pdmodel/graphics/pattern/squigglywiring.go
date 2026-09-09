package pattern

// The squiggly annotation's tiling pattern.
//
// PDSquigglyAppearanceHandler.generateNormalAppearance fills the squiggle with
// an uncoloured tiling pattern, and Java writes it inline. The port cannot:
// the handler is in interactive/annotation/handlers, and PDTilingPattern,
// PDPatternContentStream and the PDPattern colour space are all in packages
// that import it, directly or through pdmodel. So the handler names what it
// wants -- annotation.NewSquigglyPatternColor -- and this sets it, in the
// package that can see all three, the way NewPatternOfDictionary already does.
//
// What is inside is Java's, statement for statement.

import (
	"errors"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/form"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
)

func init() {
	annotation.NewSquigglyPatternColor = squigglyPatternColor
}

// squigglyPatternColor builds the pattern, adds it to the form's resources and
// answers the colour that names it.
func squigglyPatternColor(resources form.ResourcesLike,
	components []float32) (*color.PDColor, error) {
	tiling := NewPDTilingPattern(nil)
	tiling.SetBBox(common.NewPDRectangleOf(0, 0, 10, 12))
	tiling.SetXStep(10)
	tiling.SetYStep(13)
	tiling.SetTilingType(TilingConstantSpacingFasterTiling)
	tiling.SetPaintType(PaintUncolored)

	patternCS, err := pdmodel.NewPDPatternContentStream(tiling)
	if err != nil {
		return nil, err
	}
	// from Adobe
	for _, write := range []func() error{
		func() error { return patternCS.SetLineCapStyle(1) },
		func() error { return patternCS.SetLineJoinStyle(1) },
		func() error { return patternCS.SetLineWidth(1) },
		func() error { return patternCS.SetMiterLimit(10) },
		func() error { return patternCS.MoveTo(0, 1) },
		func() error { return patternCS.LineTo(5, 11) },
		func() error { return patternCS.LineTo(10, 1) },
		func() error { return patternCS.Stroke() },
		patternCS.Close,
	} {
		if err := write(); err != nil {
			return nil, err
		}
	}

	formResources, _ := resources.(*pdmodel.PDResources)
	if formResources == nil {
		return nil, errNoFormResources
	}
	patternName := formResources.AddPattern(tiling)
	// new PDPattern(null, PDDeviceRGB.INSTANCE)
	patternColorSpace := NewPDPatternOfColorSpace(nil, color.DeviceRGB)
	return color.NewPDColorOfPatternComponents(components, patternName,
		patternColorSpace), nil
}

// errNoFormResources is a form whose resources are not a *pdmodel.PDResources,
// which nothing in the port builds: PDFormXObject.setResources takes one and
// form.NewEmptyResources makes one.
var errNoFormResources = errors.New("pattern: the form has no resources to add a pattern to")
