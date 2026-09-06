package form

import (
	"fmt"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream/operator"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdfparser"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
	graphicsform "github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/form"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
)

// defaultFontSize is the size a default appearance string starts at.
//
// Port of the private PDDefaultAppearanceString.DEFAULT_FONT_SIZE.
const defaultFontSize = 12

// pdDefaultAppearanceString is a /DA string, read into the font, size and
// colour it sets.
//
// Port of PDDefaultAppearanceString, which Java declares package-private.
type pdDefaultAppearanceString struct {
	defaultResources *pdmodel.PDResources

	fontName  *cos.Name
	font      font.PDFont
	fontSize  float32
	fontColor *color.PDColor
}

// newPDDefaultAppearanceString reads the given /DA string.
//
// Java throws IllegalArgumentException for a missing string or missing
// resources, which is unchecked, so the port panics.
func newPDDefaultAppearanceString(defaultAppearance *cos.StringObj,
	defaultResources *pdmodel.PDResources) (*pdDefaultAppearanceString, error) {
	if defaultAppearance == nil {
		panic("/DA is a required entry. Please set a default appearance first.")
	}
	if defaultResources == nil {
		panic("/DR is a required entry")
	}
	s := &pdDefaultAppearanceString{
		defaultResources: defaultResources,
		fontSize:         defaultFontSize,
	}
	if err := s.processAppearanceStringOperators(defaultAppearance.Bytes()); err != nil {
		return nil, err
	}
	return s, nil
}

// processAppearanceStringOperators walks the operators of the string. Java
// declares it private.
func (s *pdDefaultAppearanceString) processAppearanceStringOperators(content []byte) error {
	arguments := []cos.Base{}
	parser, err := pdfparser.NewStreamTokenParser(content)
	if err != nil {
		return err
	}
	token, err := parser.ParseNextToken()
	if err != nil {
		return err
	}
	for token != nil {
		if op, isOperator := token.(*operator.Operator); isOperator {
			if err := s.processOperator(op, arguments); err != nil {
				return err
			}
			arguments = []cos.Base{}
		} else {
			arguments = append(arguments, token.(cos.Base))
		}
		token, err = parser.ParseNextToken()
		if err != nil {
			return err
		}
	}
	return nil
}

// processOperator handles one operator of the string. Java declares it private.
func (s *pdDefaultAppearanceString) processOperator(op *operator.Operator,
	operands []cos.Base) error {
	switch op.Name() {
	case operator.SetFontAndSize:
		return s.processSetFont(operands)
	case operator.NonStrokingGray, operator.NonStrokingRgb, operator.NonStrokingCmyk:
		return s.processSetFontColor(operands)
	}
	return nil
}

// processSetFont reads the font and size of a Tf operator. Java declares it
// private.
func (s *pdDefaultAppearanceString) processSetFont(operands []cos.Base) error {
	if len(operands) < 2 {
		return fmt.Errorf("Missing operands for set font operator %v", operands)
	}
	fontName, isName := operands[0].(*cos.Name)
	if !isName {
		return nil
	}
	size, isNumber := operands[1].(cos.Number)
	if !isNumber {
		return nil
	}
	appearanceFont, err := s.defaultResources.GetFont(fontName)
	if err != nil {
		return err
	}
	fontSize := size.FloatValue()

	// todo: handle cases where font == null with special mapping logic (see PDFBOX-2661)
	if appearanceFont == nil {
		return fmt.Errorf("Could not find font: /%s", fontName.Name())
	}
	s.setFontName(fontName)
	s.setFont(appearanceFont)
	s.setFontSize(fontSize)
	return nil
}

// processSetFontColor reads the colour of a non-stroking colour operator. Java
// declares it private.
func (s *pdDefaultAppearanceString) processSetFontColor(operands []cos.Base) error {
	var colorSpace color.PDColorSpace
	switch len(operands) {
	case 1:
		colorSpace = color.DeviceGray
	case 3:
		colorSpace = color.DeviceRGB
	case 4:
		colorSpace = color.DeviceCMYK
	default:
		return fmt.Errorf("Missing operands for set non stroking color operator %v", operands)
	}
	array := cos.NewArray()
	array.AddAll(operands)
	s.setFontColor(color.NewPDColorOfCOSArray(array, colorSpace))
	return nil
}

// FontName returns the name of the font the string names. Java declares it
// package-private.
func (s *pdDefaultAppearanceString) FontName() *cos.Name { return s.fontName }

// setFontName sets the name of the font.
func (s *pdDefaultAppearanceString) setFontName(fontName *cos.Name) { s.fontName = fontName }

// Font returns the font the string names.
func (s *pdDefaultAppearanceString) Font() font.PDFont { return s.font }

// setFont sets the font.
func (s *pdDefaultAppearanceString) setFont(f font.PDFont) { s.font = f }

// FontSize returns the size the string sets.
func (s *pdDefaultAppearanceString) FontSize() float32 { return s.fontSize }

// setFontSize sets the size.
func (s *pdDefaultAppearanceString) setFontSize(fontSize float32) { s.fontSize = fontSize }

// FontColor returns the colour the string sets, or nil.
func (s *pdDefaultAppearanceString) FontColor() *color.PDColor { return s.fontColor }

// setFontColor sets the colour.
func (s *pdDefaultAppearanceString) setFontColor(fontColor *color.PDColor) {
	s.fontColor = fontColor
}

// writeTo writes the font and colour into the given content stream, taking
// zeroFontSize where the string asks for a size of zero.
func (s *pdDefaultAppearanceString) writeTo(contents *pdmodel.PDAppearanceContentStream,
	zeroFontSize float32) error {
	fontSize := s.FontSize()
	if fontSize == 0 {
		fontSize = zeroFontSize
	}
	if err := contents.SetFont(s.Font(), fontSize); err != nil {
		return err
	}
	if s.FontColor() != nil {
		return contents.SetNonStrokingColor(s.FontColor())
	}
	return nil
}

// copyNeededResourcesTo gives the appearance stream the font this string names.
func (s *pdDefaultAppearanceString) copyNeededResourcesTo(
	appearanceStream *annotation.PDAppearanceStream) error {
	// make sure we have resources
	streamResources, _ := appearanceStream.Resources().(*pdmodel.PDResources)
	if streamResources == nil {
		streamResources = pdmodel.NewPDResources()
		appearanceStream.SetResources(graphicsform.ResourcesLike(streamResources))
	}

	existing, err := streamResources.GetFont(s.fontName)
	if err != nil {
		return err
	}
	if existing == nil {
		streamResources.PutFont(s.fontName, s.Font())
	}
	// todo: other kinds of resource...
	return nil
}
