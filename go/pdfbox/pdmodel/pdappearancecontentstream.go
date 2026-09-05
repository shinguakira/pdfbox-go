package pdmodel

import (
	"io"
	"math"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream/operator"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
)

// PDAppearanceContentStream writes the content stream of an annotation
// appearance.
//
// Port of PDAppearanceContentStream, which Java declares final.
type PDAppearanceContentStream struct {
	pdAbstractContentStream
}

// NewPDAppearanceContentStream writes into the given appearance, uncompressed.
func NewPDAppearanceContentStream(
	appearance *annotation.PDAppearanceStream) (*PDAppearanceContentStream, error) {
	outputStream, err := appearance.PDStream().CreateOutputStream()
	if err != nil {
		return nil, err
	}
	return NewPDAppearanceContentStreamOf(appearance, outputStream), nil
}

// NewPDAppearanceContentStreamCompressed writes into the given appearance,
// deflating the content where compress is true.
func NewPDAppearanceContentStreamCompressed(appearance *annotation.PDAppearanceStream,
	compress bool) (*PDAppearanceContentStream, error) {
	var filter *cos.Name
	if compress {
		filter = cos.FlateDecode
	}
	outputStream, err := appearance.PDStream().CreateOutputStreamOfFilter(filter)
	if err != nil {
		return nil, err
	}
	return NewPDAppearanceContentStreamOf(appearance, outputStream), nil
}

// NewPDAppearanceContentStreamOf writes into the given output stream, with the
// resources of the given appearance.
func NewPDAppearanceContentStreamOf(appearance *annotation.PDAppearanceStream,
	outputStream io.WriteCloser) *PDAppearanceContentStream {
	c := &PDAppearanceContentStream{}
	resources, _ := appearance.Resources().(*PDResources)
	c.initAbstractContentStream(nil, outputStream, resources)
	return c
}

// SetStrokingColorOnDemand sets the stroking colour where there is one, and
// reports whether it did.
func (c *PDAppearanceContentStream) SetStrokingColorOnDemand(value *color.PDColor) (bool, error) {
	if value != nil {
		components := value.Components()
		if len(components) > 0 {
			return true, c.SetStrokingColorComponents(components)
		}
	}
	return false, nil
}

// SetStrokingColorComponents sets the stroking colour from its components,
// choosing the operator by how many there are.
//
// Java names this setStrokingColor, overloading the base class method that
// takes a PDColor; Go has no overloading, so the port says what the argument
// is.
func (c *PDAppearanceContentStream) SetStrokingColorComponents(components []float32) error {
	for _, value := range components {
		if err := c.writeOperandFloat(value); err != nil {
			return err
		}
	}
	switch len(components) {
	case 1:
		return c.writeOperator(operator.StrokingColorGray)
	case 3:
		return c.writeOperator(operator.StrokingColorRgb)
	case 4:
		return c.writeOperator(operator.StrokingColorCmyk)
	}
	// TODO shouldn't we set the stack?
	// Or call the appropriate setStrokingColor() method from the base class?
	return nil
}

// SetNonStrokingColorOnDemand sets the non-stroking colour where there is one,
// and reports whether it did.
func (c *PDAppearanceContentStream) SetNonStrokingColorOnDemand(
	value *color.PDColor) (bool, error) {
	if value != nil {
		components := value.Components()
		if len(components) > 0 {
			return true, c.SetNonStrokingColorComponents(components)
		}
	}
	return false, nil
}

// SetNonStrokingColorComponents sets the non-stroking colour from its
// components, choosing the operator by how many there are.
func (c *PDAppearanceContentStream) SetNonStrokingColorComponents(components []float32) error {
	for _, value := range components {
		if err := c.writeOperandFloat(value); err != nil {
			return err
		}
	}
	switch len(components) {
	case 1:
		return c.writeOperator(operator.NonStrokingGray)
	case 3:
		return c.writeOperator(operator.NonStrokingRgb)
	case 4:
		return c.writeOperator(operator.NonStrokingCmyk)
	}
	// TODO shouldn't we set the stack?
	// Or call the appropriate setNonStrokingColor() method from the base class?
	return nil
}

// SetBorderLine sets the dash pattern and the width of a border.
func (c *PDAppearanceContentStream) SetBorderLine(lineWidth float32,
	bs *annotation.PDBorderStyleDictionary, border *cos.Array) error {
	// Can't use PDBorderStyleDictionary.getDashStyle() as
	// this will return a default dash style if non is existing
	if bs != nil && bs.Dictionary().ContainsKey(cos.D) &&
		bs.Style() == annotation.BorderStyleDashed {
		if err := c.SetLineDashPattern(bs.DashStyle().DashArray(), 0); err != nil {
			return err
		}
	} else if bs == nil && border.Size() > 3 {
		if dashArray, isArray := border.GetObject(3).(*cos.Array); isArray {
			if err := c.SetLineDashPattern(dashArray.ToFloatArray(), 0); err != nil {
				return err
			}
		} else {
			// PDFBOX-5266: invalid dash array, be invisible
			if err := c.SetLineDashPattern(make([]float32, 1), 0); err != nil {
				return err
			}
		}
	}
	return c.SetLineWidthOnDemand(lineWidth)
}

// SetLineWidthOnDemand sets the line width unless it is the default of one.
func (c *PDAppearanceContentStream) SetLineWidthOnDemand(lineWidth float32) error {
	// Acrobat doesn't write a line width command
	// for a line width of 1 as this is default.
	// Will do the same.
	if math.Abs(float64(lineWidth-1)) >= 1e-6 {
		return c.SetLineWidth(lineWidth)
	}
	return nil
}

// DrawShape closes the current path the way the given stroke and fill ask.
func (c *PDAppearanceContentStream) DrawShape(lineWidth float32, hasStroke, hasFill bool) error {
	// initial setting if stroking shall be done
	resolvedHasStroke := hasStroke
	// no stroking for very small lines
	if lineWidth < 1e-6 {
		resolvedHasStroke = false
	}
	switch {
	case hasFill && resolvedHasStroke:
		return c.FillAndStroke()
	case resolvedHasStroke:
		return c.Stroke()
	case hasFill:
		return c.Fill()
	}
	return c.writeOperator(operator.Endpath)
}
