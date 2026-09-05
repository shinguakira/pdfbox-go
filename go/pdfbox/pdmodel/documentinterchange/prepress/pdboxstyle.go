// Package prepress holds the box style dictionary, which says how a page's
// boundary guidelines are drawn when the page is printed for prepress.
//
// Port of org.apache.pdfbox.pdmodel.documentinterchange.prepress.
package prepress

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
)

// The guideline styles a box style can have.
const (
	GuidelineStyleSolid  = "S"
	GuidelineStyleDashed = "D"
)

// PDBoxStyle is the style of the guidelines that show a page boundary.
//
// Port of PDBoxStyle.
type PDBoxStyle struct {
	dictionary *cos.Dictionary
}

var _ common.COSObjectable = (*PDBoxStyle)(nil)

// NewPDBoxStyle builds an empty box style.
func NewPDBoxStyle() *PDBoxStyle {
	return &PDBoxStyle{dictionary: cos.NewDictionary()}
}

// NewPDBoxStyleOf builds one over the given dictionary.
func NewPDBoxStyleOf(dic *cos.Dictionary) *PDBoxStyle {
	return &PDBoxStyle{dictionary: dic}
}

// COSObject returns the dictionary.
func (s *PDBoxStyle) COSObject() cos.Base { return s.dictionary }

// Dictionary returns the dictionary, typed.
func (s *PDBoxStyle) Dictionary() *cos.Dictionary { return s.dictionary }

// GuidelineColor returns the /C colour of the guidelines, writing the default
// of black into the dictionary where there is none.
func (s *PDBoxStyle) GuidelineColor() *color.PDColor {
	colorValues := s.dictionary.GetCOSArray(cos.C)
	if colorValues == nil {
		colorValues = cos.NewArrayOf([]cos.Base{
			cos.GetInteger(0),
			cos.GetInteger(0),
			cos.GetInteger(0),
		})
		s.dictionary.SetItem(cos.C, colorValues)
	}
	return color.NewPDColorOfComponents(colorValues.ToFloatArray(), color.DeviceRGB)
}

// SetGuideLineColor sets the /C colour of the guidelines.
func (s *PDBoxStyle) SetGuideLineColor(guidelineColor *color.PDColor) {
	var values *cos.Array
	if guidelineColor != nil {
		values = guidelineColor.ToCOSArray()
	}
	if values == nil {
		s.dictionary.SetItem(cos.C, nil)
		return
	}
	s.dictionary.SetItem(cos.C, values)
}

// GuidelineWidth returns the /W width of the guidelines, which defaults to 1.
func (s *PDBoxStyle) GuidelineWidth() float32 {
	return s.dictionary.GetFloat(cos.W, 1)
}

// SetGuidelineWidth sets the /W width of the guidelines.
func (s *PDBoxStyle) SetGuidelineWidth(width float32) {
	s.dictionary.SetFloat(cos.W, width)
}

// GuidelineStyle returns the /S style of the guidelines, which defaults to
// solid.
func (s *PDBoxStyle) GuidelineStyle() string {
	return s.dictionary.GetNameAsString(cos.S, GuidelineStyleSolid)
}

// SetGuidelineStyle sets the /S style of the guidelines.
func (s *PDBoxStyle) SetGuidelineStyle(style string) {
	s.dictionary.SetName(cos.S, style)
}

// LineDashPattern returns the /D dash pattern of the guidelines, writing the
// default of a three unit dash into the dictionary where there is none.
func (s *PDBoxStyle) LineDashPattern() *graphics.PDLineDashPattern {
	d := s.dictionary.GetCOSArray(cos.D)
	if d == nil {
		d = cos.NewArrayOf([]cos.Base{cos.GetInteger(3)})
		s.dictionary.SetItem(cos.D, d)
	}
	lineArray := cos.NewArrayOf([]cos.Base{d})
	// dash phase is not specified and assumed to be zero.
	return graphics.NewPDLineDashPatternOf(lineArray, 0)
}

// SetLineDashPattern sets the /D dash pattern of the guidelines.
func (s *PDBoxStyle) SetLineDashPattern(dashArray *cos.Array) {
	if dashArray == nil {
		s.dictionary.SetItem(cos.D, nil)
		return
	}
	s.dictionary.SetItem(cos.D, dashArray)
}
