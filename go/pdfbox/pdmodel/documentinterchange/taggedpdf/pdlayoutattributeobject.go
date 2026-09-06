package taggedpdf

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/documentinterchange/logicalstructure"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
)

// OwnerLayout is the /O owner of a layout attribute object.
const OwnerLayout = "Layout"

// The attributes of a layout attribute object. Java declares them private.
const (
	placement                = "Placement"
	writingMode              = "WritingMode"
	backgroundColor          = "BackgroundColor"
	borderColor              = "BorderColor"
	borderStyle              = "BorderStyle"
	borderThickness          = "BorderThickness"
	padding                  = "Padding"
	colorAttribute           = "Color"
	spaceBefore              = "SpaceBefore"
	spaceAfter               = "SpaceAfter"
	startIndent              = "StartIndent"
	endIndent                = "EndIndent"
	textIndent               = "TextIndent"
	textAlign                = "TextAlign"
	bbox                     = "BBox"
	width                    = "Width"
	height                   = "Height"
	blockAlign               = "BlockAlign"
	inlineAlign              = "InlineAlign"
	tBorderStyle             = "TBorderStyle"
	tPadding                 = "TPadding"
	baselineShift            = "BaselineShift"
	lineHeight               = "LineHeight"
	textDecorationColor      = "TextDecorationColor"
	textDecorationThickness  = "TextDecorationThickness"
	textDecorationType       = "TextDecorationType"
	rubyAlign                = "RubyAlign"
	rubyPosition             = "RubyPosition"
	glyphOrientationVertical = "GlyphOrientationVertical"
	columnCount              = "ColumnCount"
	columnGap                = "ColumnGap"
	columnWidths             = "ColumnWidths"
)

// The placements a layout can have.
const (
	PlacementBlock  = "Block"
	PlacementInline = "Inline"
	PlacementBefore = "Before"
	PlacementStart  = "Start"
	PlacementEnd    = "End"
)

// The writing modes a layout can have.
const (
	WritingModeLrTb = "LrTb"
	WritingModeRlTb = "RlTb"
	WritingModeTbRl = "TbRl"
)

// The border styles a layout can have.
const (
	BorderStyleNone   = "None"
	BorderStyleHidden = "Hidden"
	BorderStyleDotted = "Dotted"
	BorderStyleDashed = "Dashed"
	BorderStyleSolid  = "Solid"
	BorderStyleDouble = "Double"
	BorderStyleGroove = "Groove"
	BorderStyleRidge  = "Ridge"
	BorderStyleInset  = "Inset"
	BorderStyleOutset = "Outset"
)

// The text alignments a layout can have.
const (
	TextAlignStart   = "Start"
	TextAlignCenter  = "Center"
	TextAlignEnd     = "End"
	TextAlignJustify = "Justify"
)

// WidthAuto and HeightAuto say the width or height is worked out from the
// content.
const (
	WidthAuto  = "Auto"
	HeightAuto = "Auto"
)

// The block alignments a layout can have.
const (
	BlockAlignBefore  = "Before"
	BlockAlignMiddle  = "Middle"
	BlockAlignAfter   = "After"
	BlockAlignJustify = "Justify"
)

// The inline alignments a layout can have.
const (
	InlineAlignStart  = "Start"
	InlineAlignCenter = "Center"
	InlineAlignEnd    = "End"
)

// The line heights a layout can have beside a number.
const (
	LineHeightNormal = "Normal"
	LineHeightAuto   = "Auto"
)

// The text decorations a layout can have.
const (
	TextDecorationTypeNone        = "None"
	TextDecorationTypeUnderline   = "Underline"
	TextDecorationTypeOverline    = "Overline"
	TextDecorationTypeLineThrough = "LineThrough"
)

// The ruby alignments a layout can have.
const (
	RubyAlignStart      = "Start"
	RubyAlignCenter     = "Center"
	RubyAlignEnd        = "End"
	RubyAlignJustify    = "Justify"
	RubyAlignDistribute = "Distribute"
)

// The ruby positions a layout can have.
const (
	RubyPositionBefore  = "Before"
	RubyPositionAfter   = "After"
	RubyPositionWarichu = "Warichu"
	RubyPositionInline  = "Inline"
)

// The vertical glyph orientations a layout can have.
const (
	GlyphOrientationVerticalAuto            = "Auto"
	GlyphOrientationVerticalMinus180Degrees = "-180"
	GlyphOrientationVerticalMinus90Degrees  = "-90"
	GlyphOrientationVerticalZeroDegrees     = "0"
	GlyphOrientationVertical90Degrees       = "90"
	GlyphOrientationVertical180Degrees      = "180"
	GlyphOrientationVertical270Degrees      = "270"
	GlyphOrientationVertical360Degrees      = "360"
)

func init() {
	logicalstructure.RegisterAttributeObject(OwnerLayout,
		func(dictionary *cos.Dictionary) logicalstructure.PDAttributeObject {
			return NewPDLayoutAttributeObjectOf(dictionary)
		})
}

// PDLayoutAttributeObject is the layout attribute object of a structure
// element.
//
// Port of PDLayoutAttributeObject.
type PDLayoutAttributeObject struct {
	PDStandardAttributeObject
}

var _ logicalstructure.PDAttributeObject = (*PDLayoutAttributeObject)(nil)

// NewPDLayoutAttributeObject builds an empty layout attribute object.
func NewPDLayoutAttributeObject() *PDLayoutAttributeObject {
	o := &PDLayoutAttributeObject{}
	o.InitStandardAttributeObject(o)
	o.SetOwner(OwnerLayout)
	return o
}

// NewPDLayoutAttributeObjectOf builds one over the given dictionary.
func NewPDLayoutAttributeObjectOf(dictionary *cos.Dictionary) *PDLayoutAttributeObject {
	o := &PDLayoutAttributeObject{}
	o.InitStandardAttributeObjectOf(o, dictionary)
	return o
}

// Placement returns the /Placement, which defaults to inline.
func (o *PDLayoutAttributeObject) Placement() string {
	return o.GetNameDefault(placement, PlacementInline)
}

// SetPlacement sets the /Placement.
func (o *PDLayoutAttributeObject) SetPlacement(value string) {
	o.SetName(placement, value)
}

// WritingMode returns the /WritingMode, which defaults to left to right, top to
// bottom.
func (o *PDLayoutAttributeObject) WritingMode() string {
	return o.GetNameDefault(writingMode, WritingModeLrTb)
}

// SetWritingMode sets the /WritingMode.
func (o *PDLayoutAttributeObject) SetWritingMode(value string) {
	o.SetName(writingMode, value)
}

// BackgroundColor returns the /BackgroundColor, or nil.
func (o *PDLayoutAttributeObject) BackgroundColor() *color.PDGamma {
	return o.GetColor(backgroundColor)
}

// SetBackgroundColor sets the /BackgroundColor.
func (o *PDLayoutAttributeObject) SetBackgroundColor(value *color.PDGamma) {
	o.PDStandardAttributeObject.SetColor(backgroundColor, value)
}

// BorderColors returns the /BorderColor: one colour, four of them, or nil.
func (o *PDLayoutAttributeObject) BorderColors() any {
	return o.GetColorOrFourColors(borderColor)
}

// SetAllBorderColors sets one /BorderColor for every edge.
func (o *PDLayoutAttributeObject) SetAllBorderColors(value *color.PDGamma) {
	o.PDStandardAttributeObject.SetColor(borderColor, value)
}

// SetBorderColors sets the /BorderColor of the four edges.
func (o *PDLayoutAttributeObject) SetBorderColors(borderColors *PDFourColours) {
	o.SetFourColors(borderColor, borderColors)
}

// BorderStyle returns the /BorderStyle: one name, an array of them, or the
// default of none.
func (o *PDLayoutAttributeObject) BorderStyle() any {
	return o.GetNameOrArrayOfName(borderStyle, BorderStyleNone)
}

// SetAllBorderStyles sets one /BorderStyle for every edge.
func (o *PDLayoutAttributeObject) SetAllBorderStyles(value string) {
	o.SetName(borderStyle, value)
}

// SetBorderStyles sets the /BorderStyle of the four edges.
func (o *PDLayoutAttributeObject) SetBorderStyles(borderStyles []string) {
	o.SetArrayOfName(borderStyle, borderStyles)
}

// BorderThickness returns the /BorderThickness: one number, an array of them,
// or nil.
func (o *PDLayoutAttributeObject) BorderThickness() any {
	return o.GetNumberOrArrayOfNumber(borderThickness, Unspecified)
}

// SetAllBorderThicknesses sets one /BorderThickness for every edge.
func (o *PDLayoutAttributeObject) SetAllBorderThicknesses(value float32) {
	o.SetNumber(borderThickness, value)
}

// SetAllBorderThicknessesInt sets one /BorderThickness for every edge as an
// integer. Java tells the two apart by the argument type.
func (o *PDLayoutAttributeObject) SetAllBorderThicknessesInt(value int) {
	o.SetNumberInt(borderThickness, value)
}

// SetBorderThicknesses sets the /BorderThickness of the four edges.
func (o *PDLayoutAttributeObject) SetBorderThicknesses(borderThicknesses []float32) {
	o.SetArrayOfNumber(borderThickness, borderThicknesses)
}

// Padding returns the /Padding: one number, an array of them, or the default of
// zero.
func (o *PDLayoutAttributeObject) Padding() any {
	return o.GetNumberOrArrayOfNumber(padding, 0)
}

// SetAllPaddings sets one /Padding for every edge.
func (o *PDLayoutAttributeObject) SetAllPaddings(value float32) {
	o.SetNumber(padding, value)
}

// SetAllPaddingsInt sets one /Padding for every edge as an integer.
func (o *PDLayoutAttributeObject) SetAllPaddingsInt(value int) {
	o.SetNumberInt(padding, value)
}

// SetPaddings sets the /Padding of the four edges.
func (o *PDLayoutAttributeObject) SetPaddings(paddings []float32) {
	o.SetArrayOfNumber(padding, paddings)
}

// Color returns the /Color of the text and the decorations, or nil.
func (o *PDLayoutAttributeObject) Color() *color.PDGamma {
	return o.GetColor(colorAttribute)
}

// SetColor sets the /Color of the text and the decorations.
//
// It shadows the two-argument SetColor of PDStandardAttributeObject, the way
// Java's public setColor(PDGamma) overloads the protected setColor(String,
// PDGamma); this package reaches the other one through the embedded field.
func (o *PDLayoutAttributeObject) SetColor(value *color.PDGamma) {
	o.PDStandardAttributeObject.SetColor(colorAttribute, value)
}

// SpaceBefore returns the /SpaceBefore, which defaults to zero.
func (o *PDLayoutAttributeObject) SpaceBefore() float32 {
	return o.GetNumberDefault(spaceBefore, 0)
}

// SetSpaceBefore sets the /SpaceBefore.
func (o *PDLayoutAttributeObject) SetSpaceBefore(value float32) {
	o.SetNumber(spaceBefore, value)
}

// SetSpaceBeforeInt sets the /SpaceBefore as an integer.
func (o *PDLayoutAttributeObject) SetSpaceBeforeInt(value int) {
	o.SetNumberInt(spaceBefore, value)
}

// SpaceAfter returns the /SpaceAfter, which defaults to zero.
func (o *PDLayoutAttributeObject) SpaceAfter() float32 {
	return o.GetNumberDefault(spaceAfter, 0)
}

// SetSpaceAfter sets the /SpaceAfter.
func (o *PDLayoutAttributeObject) SetSpaceAfter(value float32) {
	o.SetNumber(spaceAfter, value)
}

// SetSpaceAfterInt sets the /SpaceAfter as an integer.
func (o *PDLayoutAttributeObject) SetSpaceAfterInt(value int) {
	o.SetNumberInt(spaceAfter, value)
}

// StartIndent returns the /StartIndent, which defaults to zero.
func (o *PDLayoutAttributeObject) StartIndent() float32 {
	return o.GetNumberDefault(startIndent, 0)
}

// SetStartIndent sets the /StartIndent.
func (o *PDLayoutAttributeObject) SetStartIndent(value float32) {
	o.SetNumber(startIndent, value)
}

// SetStartIndentInt sets the /StartIndent as an integer.
func (o *PDLayoutAttributeObject) SetStartIndentInt(value int) {
	o.SetNumberInt(startIndent, value)
}

// EndIndent returns the /EndIndent, which defaults to zero.
func (o *PDLayoutAttributeObject) EndIndent() float32 {
	return o.GetNumberDefault(endIndent, 0)
}

// SetEndIndent sets the /EndIndent.
func (o *PDLayoutAttributeObject) SetEndIndent(value float32) {
	o.SetNumber(endIndent, value)
}

// SetEndIndentInt sets the /EndIndent as an integer.
func (o *PDLayoutAttributeObject) SetEndIndentInt(value int) {
	o.SetNumberInt(endIndent, value)
}

// TextIndent returns the /TextIndent, which defaults to zero.
func (o *PDLayoutAttributeObject) TextIndent() float32 {
	return o.GetNumberDefault(textIndent, 0)
}

// SetTextIndent sets the /TextIndent.
func (o *PDLayoutAttributeObject) SetTextIndent(value float32) {
	o.SetNumber(textIndent, value)
}

// SetTextIndentInt sets the /TextIndent as an integer.
func (o *PDLayoutAttributeObject) SetTextIndentInt(value int) {
	o.SetNumberInt(textIndent, value)
}

// TextAlign returns the /TextAlign, which defaults to start.
func (o *PDLayoutAttributeObject) TextAlign() string {
	return o.GetNameDefault(textAlign, TextAlignStart)
}

// SetTextAlign sets the /TextAlign.
func (o *PDLayoutAttributeObject) SetTextAlign(value string) {
	o.SetName(textAlign, value)
}

// BBox returns the /BBox, or nil.
func (o *PDLayoutAttributeObject) BBox() *common.PDRectangle {
	if array := o.Dictionary().GetCOSArray(cos.BBox); array != nil {
		return common.NewPDRectangleOfCOSArray(array)
	}
	return nil
}

// SetBBox sets the /BBox.
func (o *PDLayoutAttributeObject) SetBBox(box *common.PDRectangle) {
	key := cos.GetPDFName(bbox)
	oldValue := o.Dictionary().GetDictionaryObject(key)
	var newValue cos.Base
	if box != nil {
		newValue = box.COSObject()
	}
	o.Dictionary().SetItem(key, newValue)
	o.PotentiallyNotifyChanged(oldValue, newValue)
}

// Width returns the /Width: a number, or the name Auto.
func (o *PDLayoutAttributeObject) Width() any {
	return o.GetNumberOrName(width, WidthAuto)
}

// SetWidthAuto sets the /Width to Auto.
func (o *PDLayoutAttributeObject) SetWidthAuto() {
	o.SetName(width, WidthAuto)
}

// SetWidth sets the /Width.
func (o *PDLayoutAttributeObject) SetWidth(value float32) {
	o.SetNumber(width, value)
}

// SetWidthInt sets the /Width as an integer.
func (o *PDLayoutAttributeObject) SetWidthInt(value int) {
	o.SetNumberInt(width, value)
}

// Height returns the /Height: a number, or the name Auto.
func (o *PDLayoutAttributeObject) Height() any {
	return o.GetNumberOrName(height, HeightAuto)
}

// SetHeightAuto sets the /Height to Auto.
func (o *PDLayoutAttributeObject) SetHeightAuto() {
	o.SetName(height, HeightAuto)
}

// SetHeight sets the /Height.
func (o *PDLayoutAttributeObject) SetHeight(value float32) {
	o.SetNumber(height, value)
}

// SetHeightInt sets the /Height as an integer.
func (o *PDLayoutAttributeObject) SetHeightInt(value int) {
	o.SetNumberInt(height, value)
}

// BlockAlign returns the /BlockAlign, which defaults to before.
func (o *PDLayoutAttributeObject) BlockAlign() string {
	return o.GetNameDefault(blockAlign, BlockAlignBefore)
}

// SetBlockAlign sets the /BlockAlign.
func (o *PDLayoutAttributeObject) SetBlockAlign(value string) {
	o.SetName(blockAlign, value)
}

// InlineAlign returns the /InlineAlign, which defaults to start.
func (o *PDLayoutAttributeObject) InlineAlign() string {
	return o.GetNameDefault(inlineAlign, InlineAlignStart)
}

// SetInlineAlign sets the /InlineAlign.
func (o *PDLayoutAttributeObject) SetInlineAlign(value string) {
	o.SetName(inlineAlign, value)
}

// TBorderStyle returns the /TBorderStyle of a table: one name, an array of
// them, or the default of none.
func (o *PDLayoutAttributeObject) TBorderStyle() any {
	return o.GetNameOrArrayOfName(tBorderStyle, BorderStyleNone)
}

// SetAllTBorderStyles sets one /TBorderStyle for every edge.
func (o *PDLayoutAttributeObject) SetAllTBorderStyles(value string) {
	o.SetName(tBorderStyle, value)
}

// SetTBorderStyles sets the /TBorderStyle of the four edges.
func (o *PDLayoutAttributeObject) SetTBorderStyles(tBorderStyles []string) {
	o.SetArrayOfName(tBorderStyle, tBorderStyles)
}

// TPadding returns the /TPadding of a table: one number, an array of them, or
// the default of zero.
func (o *PDLayoutAttributeObject) TPadding() any {
	return o.GetNumberOrArrayOfNumber(tPadding, 0)
}

// SetAllTPaddings sets one /TPadding for every edge.
func (o *PDLayoutAttributeObject) SetAllTPaddings(value float32) {
	o.SetNumber(tPadding, value)
}

// SetAllTPaddingsInt sets one /TPadding for every edge as an integer.
func (o *PDLayoutAttributeObject) SetAllTPaddingsInt(value int) {
	o.SetNumberInt(tPadding, value)
}

// SetTPaddings sets the /TPadding of the four edges.
func (o *PDLayoutAttributeObject) SetTPaddings(tPaddings []float32) {
	o.SetArrayOfNumber(tPadding, tPaddings)
}

// BaselineShift returns the /BaselineShift, which defaults to zero.
func (o *PDLayoutAttributeObject) BaselineShift() float32 {
	return o.GetNumberDefault(baselineShift, 0)
}

// SetBaselineShift sets the /BaselineShift.
func (o *PDLayoutAttributeObject) SetBaselineShift(value float32) {
	o.SetNumber(baselineShift, value)
}

// SetBaselineShiftInt sets the /BaselineShift as an integer.
func (o *PDLayoutAttributeObject) SetBaselineShiftInt(value int) {
	o.SetNumberInt(baselineShift, value)
}

// LineHeight returns the /LineHeight: a number, or the name Normal or Auto.
func (o *PDLayoutAttributeObject) LineHeight() any {
	return o.GetNumberOrName(lineHeight, LineHeightNormal)
}

// SetLineHeightNormal sets the /LineHeight to Normal.
func (o *PDLayoutAttributeObject) SetLineHeightNormal() {
	o.SetName(lineHeight, LineHeightNormal)
}

// SetLineHeightAuto sets the /LineHeight to Auto.
func (o *PDLayoutAttributeObject) SetLineHeightAuto() {
	o.SetName(lineHeight, LineHeightAuto)
}

// SetLineHeight sets the /LineHeight.
func (o *PDLayoutAttributeObject) SetLineHeight(value float32) {
	o.SetNumber(lineHeight, value)
}

// SetLineHeightInt sets the /LineHeight as an integer.
func (o *PDLayoutAttributeObject) SetLineHeightInt(value int) {
	o.SetNumberInt(lineHeight, value)
}

// TextDecorationColor returns the /TextDecorationColor, or nil.
func (o *PDLayoutAttributeObject) TextDecorationColor() *color.PDGamma {
	return o.GetColor(textDecorationColor)
}

// SetTextDecorationColor sets the /TextDecorationColor.
func (o *PDLayoutAttributeObject) SetTextDecorationColor(value *color.PDGamma) {
	o.PDStandardAttributeObject.SetColor(textDecorationColor, value)
}

// TextDecorationThickness returns the /TextDecorationThickness, or -1.
func (o *PDLayoutAttributeObject) TextDecorationThickness() float32 {
	return o.GetNumber(textDecorationThickness)
}

// SetTextDecorationThickness sets the /TextDecorationThickness.
func (o *PDLayoutAttributeObject) SetTextDecorationThickness(value float32) {
	o.SetNumber(textDecorationThickness, value)
}

// SetTextDecorationThicknessInt sets the /TextDecorationThickness as an
// integer.
func (o *PDLayoutAttributeObject) SetTextDecorationThicknessInt(value int) {
	o.SetNumberInt(textDecorationThickness, value)
}

// TextDecorationType returns the /TextDecorationType, which defaults to none.
func (o *PDLayoutAttributeObject) TextDecorationType() string {
	return o.GetNameDefault(textDecorationType, TextDecorationTypeNone)
}

// SetTextDecorationType sets the /TextDecorationType.
func (o *PDLayoutAttributeObject) SetTextDecorationType(value string) {
	o.SetName(textDecorationType, value)
}

// RubyAlign returns the /RubyAlign, which defaults to distribute.
func (o *PDLayoutAttributeObject) RubyAlign() string {
	return o.GetNameDefault(rubyAlign, RubyAlignDistribute)
}

// SetRubyAlign sets the /RubyAlign.
func (o *PDLayoutAttributeObject) SetRubyAlign(value string) {
	o.SetName(rubyAlign, value)
}

// RubyPosition returns the /RubyPosition, which defaults to before.
func (o *PDLayoutAttributeObject) RubyPosition() string {
	return o.GetNameDefault(rubyPosition, RubyPositionBefore)
}

// SetRubyPosition sets the /RubyPosition.
func (o *PDLayoutAttributeObject) SetRubyPosition(value string) {
	o.SetName(rubyPosition, value)
}

// GlyphOrientationVertical returns the /GlyphOrientationVertical, which
// defaults to Auto.
func (o *PDLayoutAttributeObject) GlyphOrientationVertical() string {
	return o.GetNameDefault(glyphOrientationVertical, GlyphOrientationVerticalAuto)
}

// SetGlyphOrientationVertical sets the /GlyphOrientationVertical.
func (o *PDLayoutAttributeObject) SetGlyphOrientationVertical(value string) {
	o.SetName(glyphOrientationVertical, value)
}

// ColumnCount returns the /ColumnCount, which defaults to 1.
func (o *PDLayoutAttributeObject) ColumnCount() int {
	return o.GetInteger(columnCount, 1)
}

// SetColumnCount sets the /ColumnCount.
func (o *PDLayoutAttributeObject) SetColumnCount(value int) {
	o.SetInteger(columnCount, value)
}

// ColumnGap returns the /ColumnGap: one number, an array of them, or nil.
func (o *PDLayoutAttributeObject) ColumnGap() any {
	return o.GetNumberOrArrayOfNumber(columnGap, Unspecified)
}

// SetColumnGap sets one /ColumnGap for every gap.
func (o *PDLayoutAttributeObject) SetColumnGap(value float32) {
	o.SetNumber(columnGap, value)
}

// SetColumnGapInt sets one /ColumnGap for every gap as an integer.
func (o *PDLayoutAttributeObject) SetColumnGapInt(value int) {
	o.SetNumberInt(columnGap, value)
}

// SetColumnGaps sets the /ColumnGap of each gap.
func (o *PDLayoutAttributeObject) SetColumnGaps(columnGaps []float32) {
	o.SetArrayOfNumber(columnGap, columnGaps)
}

// ColumnWidths returns the /ColumnWidths: one number, an array of them, or nil.
func (o *PDLayoutAttributeObject) ColumnWidths() any {
	return o.GetNumberOrArrayOfNumber(columnWidths, Unspecified)
}

// SetAllColumnWidths sets one /ColumnWidths for every column.
func (o *PDLayoutAttributeObject) SetAllColumnWidths(value float32) {
	o.SetNumber(columnWidths, value)
}

// SetAllColumnWidthsInt sets one /ColumnWidths for every column as an integer.
func (o *PDLayoutAttributeObject) SetAllColumnWidthsInt(value int) {
	o.SetNumberInt(columnWidths, value)
}

// SetColumnWidths sets the /ColumnWidths of each column.
func (o *PDLayoutAttributeObject) SetColumnWidths(widths []float32) {
	o.SetArrayOfNumber(columnWidths, widths)
}

// String renders the attribute object the way Java's toString does.
func (o *PDLayoutAttributeObject) String() string {
	sb := strings.Builder{}
	sb.WriteString(o.PDAttributeObjectBase.String())
	if o.IsSpecified(placement) {
		sb.WriteString(", Placement=")
		sb.WriteString(o.Placement())
	}
	if o.IsSpecified(writingMode) {
		sb.WriteString(", WritingMode=")
		sb.WriteString(o.WritingMode())
	}
	if o.IsSpecified(backgroundColor) {
		sb.WriteString(", BackgroundColor=")
		sb.WriteString(fmt.Sprintf("%v", o.BackgroundColor()))
	}
	if o.IsSpecified(borderColor) {
		sb.WriteString(", BorderColor=")
		sb.WriteString(fmt.Sprintf("%v", o.BorderColors()))
	}
	if o.IsSpecified(borderStyle) {
		sb.WriteString(", BorderStyle=")
		sb.WriteString(nameOrArrayToString(o.BorderStyle()))
	}
	if o.IsSpecified(borderThickness) {
		sb.WriteString(", BorderThickness=")
		sb.WriteString(numberOrArrayToString(o.BorderThickness()))
	}
	if o.IsSpecified(padding) {
		sb.WriteString(", Padding=")
		sb.WriteString(numberOrArrayToString(o.Padding()))
	}
	if o.IsSpecified(colorAttribute) {
		sb.WriteString(", Color=")
		sb.WriteString(fmt.Sprintf("%v", o.Color()))
	}
	if o.IsSpecified(spaceBefore) {
		sb.WriteString(", SpaceBefore=")
		sb.WriteString(floatToString(o.SpaceBefore()))
	}
	if o.IsSpecified(spaceAfter) {
		sb.WriteString(", SpaceAfter=")
		sb.WriteString(floatToString(o.SpaceAfter()))
	}
	if o.IsSpecified(startIndent) {
		sb.WriteString(", StartIndent=")
		sb.WriteString(floatToString(o.StartIndent()))
	}
	if o.IsSpecified(endIndent) {
		sb.WriteString(", EndIndent=")
		sb.WriteString(floatToString(o.EndIndent()))
	}
	if o.IsSpecified(textIndent) {
		sb.WriteString(", TextIndent=")
		sb.WriteString(floatToString(o.TextIndent()))
	}
	if o.IsSpecified(textAlign) {
		sb.WriteString(", TextAlign=")
		sb.WriteString(o.TextAlign())
	}
	if o.IsSpecified(bbox) {
		sb.WriteString(", BBox=")
		sb.WriteString(fmt.Sprintf("%v", o.BBox()))
	}
	if o.IsSpecified(width) {
		sb.WriteString(", Width=")
		sb.WriteString(fmt.Sprintf("%v", o.Width()))
	}
	if o.IsSpecified(height) {
		sb.WriteString(", Height=")
		sb.WriteString(fmt.Sprintf("%v", o.Height()))
	}
	if o.IsSpecified(blockAlign) {
		sb.WriteString(", BlockAlign=")
		sb.WriteString(o.BlockAlign())
	}
	if o.IsSpecified(inlineAlign) {
		sb.WriteString(", InlineAlign=")
		sb.WriteString(o.InlineAlign())
	}
	if o.IsSpecified(tBorderStyle) {
		sb.WriteString(", TBorderStyle=")
		sb.WriteString(nameOrArrayToString(o.TBorderStyle()))
	}
	if o.IsSpecified(tPadding) {
		sb.WriteString(", TPadding=")
		sb.WriteString(numberOrArrayToString(o.TPadding()))
	}
	if o.IsSpecified(baselineShift) {
		sb.WriteString(", BaselineShift=")
		sb.WriteString(floatToString(o.BaselineShift()))
	}
	if o.IsSpecified(lineHeight) {
		sb.WriteString(", LineHeight=")
		sb.WriteString(fmt.Sprintf("%v", o.LineHeight()))
	}
	if o.IsSpecified(textDecorationColor) {
		sb.WriteString(", TextDecorationColor=")
		sb.WriteString(fmt.Sprintf("%v", o.TextDecorationColor()))
	}
	if o.IsSpecified(textDecorationThickness) {
		sb.WriteString(", TextDecorationThickness=")
		sb.WriteString(floatToString(o.TextDecorationThickness()))
	}
	if o.IsSpecified(textDecorationType) {
		sb.WriteString(", TextDecorationType=")
		sb.WriteString(o.TextDecorationType())
	}
	if o.IsSpecified(rubyAlign) {
		sb.WriteString(", RubyAlign=")
		sb.WriteString(o.RubyAlign())
	}
	if o.IsSpecified(rubyPosition) {
		sb.WriteString(", RubyPosition=")
		sb.WriteString(o.RubyPosition())
	}
	if o.IsSpecified(glyphOrientationVertical) {
		sb.WriteString(", GlyphOrientationVertical=")
		sb.WriteString(o.GlyphOrientationVertical())
	}
	if o.IsSpecified(columnCount) {
		sb.WriteString(", ColumnCount=")
		sb.WriteString(strconv.Itoa(o.ColumnCount()))
	}
	if o.IsSpecified(columnGap) {
		sb.WriteString(", ColumnGap=")
		sb.WriteString(numberOrArrayToString(o.ColumnGap()))
	}
	if o.IsSpecified(columnWidths) {
		sb.WriteString(", ColumnWidths=")
		sb.WriteString(numberOrArrayToString(o.ColumnWidths()))
	}
	return sb.String()
}

// nameOrArrayToString renders what getNameOrArrayOfName answered, which Java
// tells apart with an instanceof String[].
func nameOrArrayToString(value any) string {
	if names, isArray := value.([]string); isArray {
		return logicalstructure.ArrayToString(stringsAsAny(names))
	}
	return fmt.Sprintf("%v", value)
}

// numberOrArrayToString renders what getNumberOrArrayOfNumber answered, which
// Java tells apart with an instanceof float[].
func numberOrArrayToString(value any) string {
	if numbers, isArray := value.([]float32); isArray {
		return logicalstructure.ArrayToStringFloats(numbers)
	}
	if number, isFloat := value.(float32); isFloat {
		return floatToString(number)
	}
	return fmt.Sprintf("%v", value)
}

// floatToString renders a float the way Java's Float.toString does for a
// StringBuilder.
func floatToString(value float32) string {
	return strconv.FormatFloat(float64(value), 'g', -1, 32)
}
