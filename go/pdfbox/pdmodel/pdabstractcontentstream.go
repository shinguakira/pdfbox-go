package pdmodel

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf/gsub"
	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf/model"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream/operator"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdfwriter"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/documentinterchange/markedcontent"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/form"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/image"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/state"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// The bytes written around an operand array.
//
// Port of the three private ASCII_ constants of PDAbstractContentStream.
var (
	asciiSpace                   = []byte{0x20}
	asciiLeftSquareBracket       = []byte{0x5B}
	asciiRightSquareBracketSpace = []byte{0x5D, 0x20}
)

// pdAbstractContentStream writes the operators of a content stream.
//
// Port of PDAbstractContentStream, which Java declares package-private and
// abstract; the page, appearance, form and pattern content streams embed it.
//
// shadingFill is not here: it names PDShading, which belongs to the rendering
// this port has not reached, and PDResources cannot add one either. See
// migration/STATUS.md.
type pdAbstractContentStream struct {
	document     *PDDocument // may be nil
	outputStream io.WriteCloser
	resources    *PDResources

	inTextMode                 bool
	fontStack                  []font.PDFont
	fontSizeStack              []float32
	nonStrokingColorSpaceStack []color.PDColorSpace
	strokingColorSpaceStack    []color.PDColorSpace

	// number format
	maximumFractionDigits int
	formatBuffer          []byte

	gsubWorkerMap        map[*font.PDType0Font]gsub.GsubWorker
	gsubWorkerFactory    *gsub.Factory
	glyphLayoutProcessor GlyphLayoutProcessor
}

var _ ContentStreamForGlyphLayout = (*pdAbstractContentStream)(nil)

// initAbstractContentStream is the package-private
// PDAbstractContentStream(PDDocument, OutputStream, PDResources) constructor.
func (c *pdAbstractContentStream) initAbstractContentStream(document *PDDocument,
	outputStream io.WriteCloser, resources *PDResources) {
	c.document = document
	c.outputStream = outputStream
	c.resources = resources
	c.maximumFractionDigits = 4
	c.formatBuffer = make([]byte, 32)
	c.gsubWorkerMap = map[*font.PDType0Font]gsub.GsubWorker{}
	c.gsubWorkerFactory = gsub.NewFactory()
}

// SetGlyphLayoutProcessor sets the processor that lays text out into glyphs.
func (c *pdAbstractContentStream) SetGlyphLayoutProcessor(processor GlyphLayoutProcessor) {
	c.glyphLayoutProcessor = processor
}

// setMaximumFractionDigits sets how many fraction digits a real operand is
// written with. Java declares it protected.
func (c *pdAbstractContentStream) setMaximumFractionDigits(fractionDigitsNumber int) {
	c.maximumFractionDigits = fractionDigitsNumber
}

// BeginText begins a text object.
//
// Java throws IllegalStateException on a nested call, which is unchecked, so
// the port panics.
func (c *pdAbstractContentStream) BeginText() error {
	if c.inTextMode {
		panic("Error: Nested beginText() calls are not allowed.")
	}
	if err := c.writeOperator(operator.BeginText); err != nil {
		return err
	}
	c.inTextMode = true
	return nil
}

// EndText ends a text object.
func (c *pdAbstractContentStream) EndText() error {
	if !c.inTextMode {
		panic("Error: You must call beginText() before calling endText.")
	}
	if err := c.writeOperator(operator.EndText); err != nil {
		return err
	}
	c.inTextMode = false
	return nil
}

// SetFont sets the font and size to draw text with.
func (c *pdAbstractContentStream) SetFont(f font.PDFont, fontSize float32) error {
	if len(c.fontStack) == 0 {
		c.fontStack = append(c.fontStack, f)
	} else {
		c.fontStack[len(c.fontStack)-1] = f
	}
	if len(c.fontSizeStack) == 0 {
		c.fontSizeStack = append(c.fontSizeStack, fontSize)
	} else {
		c.fontSizeStack[len(c.fontSizeStack)-1] = fontSize
	}

	// keep track of fonts which are configured for subsetting
	if f.WillBeSubset() {
		if c.document != nil {
			c.document.addFontToSubset(f)
		} else {
			slog.Warn("pdmodel: using a subsetted font without a PDDocument context; call subset() before saving",
				slog.String("font", f.Name()))
		}
	} else if !f.IsEmbedded() && !f.IsStandard14() {
		slog.Warn("pdmodel: attempting to use a font that is not embedded",
			slog.String("font", f.Name()))
	}

	// complex text layout
	if type0Font, isType0 := f.(*font.PDType0Font); isType0 {
		gsubData := type0Font.GsubData()
		if gsubData != model.NoDataFound {
			if _, known := c.gsubWorkerMap[type0Font]; !known {
				c.gsubWorkerMap[type0Font] =
					c.gsubWorkerFactory.GetGsubWorker(type0Font.CmapLookup(), gsubData)
			}
		} else {
			slog.Info("pdmodel: no GSUB data found in font", slog.String("font", f.Name()))
		}
	}

	if err := c.writeOperandName(c.resources.AddFont(f)); err != nil {
		return err
	}
	if err := c.writeOperandFloat(fontSize); err != nil {
		return err
	}
	return c.writeOperator(operator.SetFontAndSize)
}

// ShowTextWithPositioning writes the given text and positioning adjustments,
// which are strings and float32s.
//
// Java throws IllegalArgumentException for any other entry, and
// IllegalStateException where the text object or the font is missing; both are
// unchecked, so the port panics.
func (c *pdAbstractContentStream) ShowTextWithPositioning(
	textWithPositioningArray []any) error {
	if !c.inTextMode {
		panic("Must call beginText() before showTextWithPositioning()")
	}
	if len(c.fontStack) == 0 {
		panic("Must call setFont() before showTextWithPositioning()")
	}
	if err := c.writeBytes(asciiLeftSquareBracket); err != nil {
		return err
	}
	for _, obj := range textWithPositioningArray {
		switch value := obj.(type) {
		case string:
			if err := c.showTextInternal(value); err != nil {
				return err
			}
		case float32:
			if err := c.writeOperandFloat(value); err != nil {
				return err
			}
		default:
			panic("Argument must consist of array of Float and String types")
		}
	}
	if err := c.writeBytes(asciiRightSquareBracketSpace); err != nil {
		return err
	}
	return c.writeOperator(operator.ShowTextAdjusted)
}

// ShowGlyphsWithPositioning writes the given glyph runs and positioning
// adjustments.
func (c *pdAbstractContentStream) ShowGlyphsWithPositioning(
	glyphsAndPositions *GlyphsAndPositions) error {
	if err := c.writeBytes(asciiLeftSquareBracket); err != nil {
		return err
	}
	for _, obj := range glyphsAndPositions.Array() {
		switch value := obj.(type) {
		case *GlyphSubList:
			if err := c.writeTextPDType0Font(value.IntArray()); err != nil {
				return err
			}
		case float32:
			if err := c.writeOperandFloat(value); err != nil {
				return err
			}
		default:
			if obj == nil {
				panic("Argument contains null entry")
			}
			panic(fmt.Sprintf("Argument must consist of array of Float and "+
				"GlyphsAndPositions.GlyphSubList types, not %T", obj))
		}
	}
	if err := c.writeBytes(asciiRightSquareBracketSpace); err != nil {
		return err
	}
	return c.writeOperator(operator.ShowTextAdjusted)
}

// ShowText writes the given text.
func (c *pdAbstractContentStream) ShowText(text string) error {
	if !c.inTextMode {
		panic("Must call beginText() before showText()")
	}
	if len(c.fontStack) == 0 {
		panic("Must call setFont() before showText()")
	}
	if len(c.fontSizeStack) == 0 {
		panic("Font is set, but fontSize is not set")
	}
	f := c.fontStack[len(c.fontStack)-1]
	if c.glyphLayoutProcessor != nil && c.glyphLayoutProcessor.SupportsFont(f) {
		fontSize := c.fontSizeStack[len(c.fontSizeStack)-1]
		return c.glyphLayoutProcessor.ShowText(c, f.(*font.PDType0Font), fontSize, text)
	}
	if err := c.showTextInternal(text); err != nil {
		return err
	}
	if err := c.writeBytes(asciiSpace); err != nil {
		return err
	}
	return c.writeOperator(operator.ShowText)
}

// ShowGlyphCodes writes the given glyph ids.
func (c *pdAbstractContentStream) ShowGlyphCodes(glyphCodes []int) error {
	if err := c.writeTextPDType0Font(glyphCodes); err != nil {
		return err
	}
	if err := c.writeBytes(asciiSpace); err != nil {
		return err
	}
	return c.writeOperator(operator.ShowText)
}

// writeTextPDType0Font writes the given glyph ids of the current Type 0 font.
// Java declares it protected.
func (c *pdAbstractContentStream) writeTextPDType0Font(glyphCodes []int) error {
	if !c.inTextMode {
		panic("Must call beginText() before writeTextPDType0Font()")
	}
	if len(c.fontStack) == 0 {
		panic("Must call setFont() before writeTextPDType0Font()")
	}
	pdType0Font, isType0 := c.fontStack[len(c.fontStack)-1].(*font.PDType0Font)
	if !isType0 {
		panic("Must be called with current font instance of PDType0Font")
	}

	// encode glyphs, update set of used glyphs
	out := bytes.Buffer{}
	glyphIds := map[int]bool{}
	for _, glyphCode := range glyphCodes {
		out.Write(pdType0Font.EncodeGlyphID(glyphCode))
		if glyphCode < 0xFFFF {
			glyphIds[glyphCode] = true
		}
	}
	encodedText := out.Bytes()

	// add glyphs to subset
	if pdType0Font.WillBeSubset() {
		pdType0Font.AddGlyphsToSubset(glyphIds)
	}

	// write encoded text and the PDF operator
	return pdfwriter.WriteStringBytes(encodedText, c.outputStream)
}

// showTextInternal writes the given text without an operator after it. Java
// declares it protected.
func (c *pdAbstractContentStream) showTextInternal(text string) error {
	f := c.fontStack[len(c.fontStack)-1]

	// complex text layout
	var encodedText []byte
	if type0Font, isType0 := f.(*font.PDType0Font); isType0 {
		if gsubWorker := c.gsubWorkerMap[type0Font]; gsubWorker != nil {
			glyphIds := map[int]bool{}
			var err error
			encodedText, err = c.encodeForGsub(gsubWorker, glyphIds, type0Font, text)
			if err != nil {
				return err
			}
			if type0Font.WillBeSubset() {
				type0Font.AddGlyphsToSubset(glyphIds)
			}
		}
	}

	if encodedText == nil {
		var err error
		encodedText, err = f.Encode(text)
		if err != nil {
			return err
		}
	}

	// Unicode code points to keep when subsetting
	if f.WillBeSubset() {
		for _, codePoint := range text {
			f.AddToSubset(int(codePoint))
		}
	}

	return pdfwriter.WriteStringBytes(encodedText, c.outputStream)
}

// SetLeading sets the text leading.
func (c *pdAbstractContentStream) SetLeading(leading float32) error {
	if err := c.writeOperandFloat(leading); err != nil {
		return err
	}
	return c.writeOperator(operator.SetTextLeading)
}

// NewLine moves to the start of the next line of text.
func (c *pdAbstractContentStream) NewLine() error {
	if !c.inTextMode {
		panic("Must call beginText() before newLine()")
	}
	return c.writeOperator(operator.NextLine)
}

// NewLineAtOffset moves to the start of the next line of text, offset from the
// start of the current one.
func (c *pdAbstractContentStream) NewLineAtOffset(tx, ty float32) error {
	if !c.inTextMode {
		panic("Error: must call beginText() before newLineAtOffset()")
	}
	if err := c.writeOperandFloat(tx); err != nil {
		return err
	}
	if err := c.writeOperandFloat(ty); err != nil {
		return err
	}
	return c.writeOperator(operator.MoveText)
}

// SetTextMatrix sets the text matrix.
func (c *pdAbstractContentStream) SetTextMatrix(matrix *util.Matrix) error {
	if !c.inTextMode {
		panic("Error: must call beginText() before setTextMatrix")
	}
	if err := c.writeAffineTransform(matrix.CreateAffineTransform()); err != nil {
		return err
	}
	return c.writeOperator(operator.SetMatrix)
}

// DrawImage draws an image at the given position, at its own size.
func (c *pdAbstractContentStream) DrawImage(img *image.PDImageXObject, x, y float32) error {
	return c.DrawImageSized(img, x, y, float32(img.Width()), float32(img.Height()))
}

// DrawImageSized draws an image at the given position and size.
func (c *pdAbstractContentStream) DrawImageSized(img *image.PDImageXObject,
	x, y, width, height float32) error {
	if c.inTextMode {
		panic("Error: drawImage is not allowed within a text block.")
	}
	if err := c.SaveGraphicsState(); err != nil {
		return err
	}
	transform := geom.NewAffineTransform(float64(width), 0, 0, float64(height),
		float64(x), float64(y))
	if err := c.Transform(util.NewMatrixFromAffineTransform(transform)); err != nil {
		return err
	}
	if err := c.writeOperandName(c.resources.AddImageXObject(img)); err != nil {
		return err
	}
	if err := c.writeOperator(operator.DrawObject); err != nil {
		return err
	}
	return c.RestoreGraphicsState()
}

// DrawImageWithMatrix draws an image through the given matrix.
func (c *pdAbstractContentStream) DrawImageWithMatrix(img *image.PDImageXObject,
	matrix *util.Matrix) error {
	if c.inTextMode {
		panic("Error: drawImage is not allowed within a text block.")
	}
	if err := c.SaveGraphicsState(); err != nil {
		return err
	}
	transform := matrix.CreateAffineTransform()
	if err := c.Transform(util.NewMatrixFromAffineTransform(transform)); err != nil {
		return err
	}
	if err := c.writeOperandName(c.resources.AddImageXObject(img)); err != nil {
		return err
	}
	if err := c.writeOperator(operator.DrawObject); err != nil {
		return err
	}
	return c.RestoreGraphicsState()
}

// DrawInlineImage draws an inline image at the given position, at its own size.
func (c *pdAbstractContentStream) DrawInlineImage(inlineImage *image.PDInlineImage,
	x, y float32) error {
	return c.DrawInlineImageSized(inlineImage, x, y,
		float32(inlineImage.Width()), float32(inlineImage.Height()))
}

// DrawInlineImageSized draws an inline image at the given position and size.
func (c *pdAbstractContentStream) DrawInlineImageSized(inlineImage *image.PDInlineImage,
	x, y, width, height float32) error {
	if c.inTextMode {
		panic("Error: drawImage is not allowed within a text block.")
	}
	if err := c.SaveGraphicsState(); err != nil {
		return err
	}
	if err := c.Transform(util.NewMatrixOf(width, 0, 0, height, x, y)); err != nil {
		return err
	}

	// create the image dictionary
	sb := strings.Builder{}
	sb.WriteString(operator.BeginInlineImage)
	sb.WriteString("\n /W ")
	sb.WriteString(strconv.Itoa(inlineImage.Width()))
	sb.WriteString("\n /H ")
	sb.WriteString(strconv.Itoa(inlineImage.Height()))
	sb.WriteString("\n /CS ")
	sb.WriteString("/")
	colorSpace, err := inlineImage.ColorSpace()
	if err != nil {
		return err
	}
	sb.WriteString(colorSpace.Name())
	decodeArray := inlineImage.Decode()
	if decodeArray != nil && !decodeArray.IsEmpty() {
		sb.WriteString("\n /D ")
		sb.WriteString("[")
		for i := 0; i < decodeArray.Size(); i++ {
			sb.WriteString(strconv.Itoa(decodeArray.Get(i).(cos.Number).IntValue()))
			sb.WriteString(" ")
		}
		sb.WriteString("]")
	}
	if inlineImage.IsStencil() {
		sb.WriteString("\n /IM true")
	}
	sb.WriteString("\n /BPC ")
	sb.WriteString(strconv.Itoa(inlineImage.BitsPerComponent()))

	// image dictionary
	if err := c.write(sb.String()); err != nil {
		return err
	}
	if err := c.writeLine(); err != nil {
		return err
	}

	// binary data
	if err := c.writeOperator(operator.BeginInlineImageData); err != nil {
		return err
	}
	if err := c.writeBytes(inlineImage.Data()); err != nil {
		return err
	}
	if err := c.writeLine(); err != nil {
		return err
	}
	if err := c.writeOperator(operator.EndInlineImage); err != nil {
		return err
	}
	return c.RestoreGraphicsState()
}

// DrawForm draws the given form XObject.
func (c *pdAbstractContentStream) DrawForm(formXObject *form.PDFormXObject) error {
	if c.inTextMode {
		panic("Error: drawForm is not allowed within a text block.")
	}
	if err := c.writeOperandName(c.resources.AddFormXObject(formXObject)); err != nil {
		return err
	}
	return c.writeOperator(operator.DrawObject)
}

// Transform concatenates the given matrix onto the current transformation
// matrix.
func (c *pdAbstractContentStream) Transform(matrix *util.Matrix) error {
	if c.inTextMode {
		panic("Error: Modifying the current transformation matrix is not allowed within text objects.")
	}
	if err := c.writeAffineTransform(matrix.CreateAffineTransform()); err != nil {
		return err
	}
	return c.writeOperator(operator.Concat)
}

// SaveGraphicsState pushes the graphics state.
func (c *pdAbstractContentStream) SaveGraphicsState() error {
	if c.inTextMode {
		panic("Error: Saving the graphics state is not allowed within text objects.")
	}
	if len(c.fontStack) != 0 {
		c.fontStack = append(c.fontStack, c.fontStack[len(c.fontStack)-1])
	}
	if len(c.strokingColorSpaceStack) != 0 {
		c.strokingColorSpaceStack = append(c.strokingColorSpaceStack,
			c.strokingColorSpaceStack[len(c.strokingColorSpaceStack)-1])
	}
	if len(c.nonStrokingColorSpaceStack) != 0 {
		c.nonStrokingColorSpaceStack = append(c.nonStrokingColorSpaceStack,
			c.nonStrokingColorSpaceStack[len(c.nonStrokingColorSpaceStack)-1])
	}
	return c.writeOperator(operator.Save)
}

// RestoreGraphicsState pops the graphics state.
func (c *pdAbstractContentStream) RestoreGraphicsState() error {
	if c.inTextMode {
		panic("Error: Restoring the graphics state is not allowed within text objects.")
	}
	if len(c.fontStack) != 0 {
		c.fontStack = c.fontStack[:len(c.fontStack)-1]
	}
	if len(c.strokingColorSpaceStack) != 0 {
		c.strokingColorSpaceStack = c.strokingColorSpaceStack[:len(c.strokingColorSpaceStack)-1]
	}
	if len(c.nonStrokingColorSpaceStack) != 0 {
		c.nonStrokingColorSpaceStack =
			c.nonStrokingColorSpaceStack[:len(c.nonStrokingColorSpaceStack)-1]
	}
	return c.writeOperator(operator.Restore)
}

// getName returns the name to write for the given colour space: the space's own
// name for a device space, and a resource name for anything else. Java declares
// it protected.
func (c *pdAbstractContentStream) getName(colorSpace color.PDColorSpace) *cos.Name {
	switch colorSpace.(type) {
	case *color.PDDeviceGray, *color.PDDeviceRGB, *color.PDDeviceCMYK:
		return cos.GetPDFName(colorSpace.Name())
	}
	return c.resources.AddColorSpace(colorSpace)
}

// needsSCN reports whether the colour of the given space is written with the
// scn form of the operator rather than the sc one.
func needsSCN(colorSpace color.PDColorSpace) bool {
	switch colorSpace.(type) {
	case color.PatternColorSpace, *color.PDSeparation, *color.PDDeviceN, *color.PDICCBased:
		return true
	}
	return false
}

// SetStrokingColor sets the colour to stroke with.
func (c *pdAbstractContentStream) SetStrokingColor(value *color.PDColor) error {
	colorSpace := value.ColorSpace()
	if len(c.strokingColorSpaceStack) == 0 ||
		c.strokingColorSpaceStack[len(c.strokingColorSpaceStack)-1] != colorSpace {
		if err := c.writeOperandName(c.getName(colorSpace)); err != nil {
			return err
		}
		if err := c.writeOperator(operator.StrokingColorspace); err != nil {
			return err
		}
		c.setStrokingColorSpaceStack(colorSpace)
	}
	for _, component := range value.Components() {
		if err := c.writeOperandFloat(component); err != nil {
			return err
		}
	}
	if _, isPattern := colorSpace.(color.PatternColorSpace); isPattern {
		if err := c.writeOperandName(value.PatternName()); err != nil {
			return err
		}
	}
	if needsSCN(colorSpace) {
		return c.writeOperator(operator.StrokingColorN)
	}
	return c.writeOperator(operator.StrokingColor)
}

// SetStrokingColorRGB255 sets the colour to stroke with, from three components
// of zero to 255.
//
// Java takes a java.awt.Color and reads its red, green and blue the same way;
// Go has no such type, so the port takes the three components themselves.
func (c *pdAbstractContentStream) SetStrokingColorRGB255(red, green, blue int) error {
	components := []float32{float32(red) / 255, float32(green) / 255, float32(blue) / 255}
	return c.SetStrokingColor(color.NewPDColorOfComponents(components, color.DeviceRGB))
}

// SetStrokingColorRGB sets the colour to stroke with, in device RGB.
//
// Java throws IllegalArgumentException outside 0..1, which is unchecked, so the
// port panics.
func (c *pdAbstractContentStream) SetStrokingColorRGB(r, g, b float32) error {
	if isOutsideOneInterval(r) || isOutsideOneInterval(g) || isOutsideOneInterval(b) {
		panic(fmt.Sprintf("Parameters must be within 0..1, but are (%.2f,%.2f,%.2f)", r, g, b))
	}
	if err := c.writeOperandFloat(r); err != nil {
		return err
	}
	if err := c.writeOperandFloat(g); err != nil {
		return err
	}
	if err := c.writeOperandFloat(b); err != nil {
		return err
	}
	if err := c.writeOperator(operator.StrokingColorRgb); err != nil {
		return err
	}
	c.setStrokingColorSpaceStack(color.DeviceRGB)
	return nil
}

// SetStrokingColorCMYK sets the colour to stroke with, in device CMYK.
func (c *pdAbstractContentStream) SetStrokingColorCMYK(cyan, magenta, yellow, black float32) error {
	if isOutsideOneInterval(cyan) || isOutsideOneInterval(magenta) ||
		isOutsideOneInterval(yellow) || isOutsideOneInterval(black) {
		panic(fmt.Sprintf("Parameters must be within 0..1, but are (%.2f,%.2f,%.2f,%.2f)",
			cyan, magenta, yellow, black))
	}
	if err := c.writeOperandFloat(cyan); err != nil {
		return err
	}
	if err := c.writeOperandFloat(magenta); err != nil {
		return err
	}
	if err := c.writeOperandFloat(yellow); err != nil {
		return err
	}
	if err := c.writeOperandFloat(black); err != nil {
		return err
	}
	if err := c.writeOperator(operator.StrokingColorCmyk); err != nil {
		return err
	}
	c.setStrokingColorSpaceStack(color.DeviceCMYK)
	return nil
}

// SetStrokingColorGray sets the colour to stroke with, in device gray.
func (c *pdAbstractContentStream) SetStrokingColorGray(g float32) error {
	if isOutsideOneInterval(g) {
		panic(fmt.Sprintf("Parameter must be within 0..1, but is %v", g))
	}
	if err := c.writeOperandFloat(g); err != nil {
		return err
	}
	if err := c.writeOperator(operator.StrokingColorGray); err != nil {
		return err
	}
	c.setStrokingColorSpaceStack(color.DeviceGray)
	return nil
}

// SetNonStrokingColor sets the colour to fill with.
func (c *pdAbstractContentStream) SetNonStrokingColor(value *color.PDColor) error {
	colorSpace := value.ColorSpace()
	if len(c.nonStrokingColorSpaceStack) == 0 ||
		c.nonStrokingColorSpaceStack[len(c.nonStrokingColorSpaceStack)-1] != colorSpace {
		if err := c.writeOperandName(c.getName(colorSpace)); err != nil {
			return err
		}
		if err := c.writeOperator(operator.NonStrokingColorspace); err != nil {
			return err
		}
		c.setNonStrokingColorSpaceStack(colorSpace)
	}
	for _, component := range value.Components() {
		if err := c.writeOperandFloat(component); err != nil {
			return err
		}
	}
	if _, isPattern := colorSpace.(color.PatternColorSpace); isPattern {
		if err := c.writeOperandName(value.PatternName()); err != nil {
			return err
		}
	}
	if needsSCN(colorSpace) {
		return c.writeOperator(operator.NonStrokingColorN)
	}
	return c.writeOperator(operator.NonStrokingColor)
}

// SetNonStrokingColorRGB255 sets the colour to fill with, from three components
// of zero to 255.
func (c *pdAbstractContentStream) SetNonStrokingColorRGB255(red, green, blue int) error {
	components := []float32{float32(red) / 255, float32(green) / 255, float32(blue) / 255}
	return c.SetNonStrokingColor(color.NewPDColorOfComponents(components, color.DeviceRGB))
}

// SetNonStrokingColorRGB sets the colour to fill with, in device RGB.
func (c *pdAbstractContentStream) SetNonStrokingColorRGB(r, g, b float32) error {
	if isOutsideOneInterval(r) || isOutsideOneInterval(g) || isOutsideOneInterval(b) {
		panic(fmt.Sprintf("Parameters must be within 0..1, but are (%.2f,%.2f,%.2f)", r, g, b))
	}
	if err := c.writeOperandFloat(r); err != nil {
		return err
	}
	if err := c.writeOperandFloat(g); err != nil {
		return err
	}
	if err := c.writeOperandFloat(b); err != nil {
		return err
	}
	if err := c.writeOperator(operator.NonStrokingRgb); err != nil {
		return err
	}
	c.setNonStrokingColorSpaceStack(color.DeviceRGB)
	return nil
}

// SetNonStrokingColorCMYK sets the colour to fill with, in device CMYK.
func (c *pdAbstractContentStream) SetNonStrokingColorCMYK(
	cyan, magenta, yellow, black float32) error {
	if isOutsideOneInterval(cyan) || isOutsideOneInterval(magenta) ||
		isOutsideOneInterval(yellow) || isOutsideOneInterval(black) {
		panic(fmt.Sprintf("Parameters must be within 0..1, but are (%.2f,%.2f,%.2f,%.2f)",
			cyan, magenta, yellow, black))
	}
	if err := c.writeOperandFloat(cyan); err != nil {
		return err
	}
	if err := c.writeOperandFloat(magenta); err != nil {
		return err
	}
	if err := c.writeOperandFloat(yellow); err != nil {
		return err
	}
	if err := c.writeOperandFloat(black); err != nil {
		return err
	}
	if err := c.writeOperator(operator.NonStrokingCmyk); err != nil {
		return err
	}
	c.setNonStrokingColorSpaceStack(color.DeviceCMYK)
	return nil
}

// SetNonStrokingColorGray sets the colour to fill with, in device gray.
func (c *pdAbstractContentStream) SetNonStrokingColorGray(g float32) error {
	if isOutsideOneInterval(g) {
		panic(fmt.Sprintf("Parameter must be within 0..1, but is %v", g))
	}
	if err := c.writeOperandFloat(g); err != nil {
		return err
	}
	if err := c.writeOperator(operator.NonStrokingGray); err != nil {
		return err
	}
	c.setNonStrokingColorSpaceStack(color.DeviceGray)
	return nil
}

// AddRect adds a rectangle to the current path.
func (c *pdAbstractContentStream) AddRect(x, y, width, height float32) error {
	if c.inTextMode {
		panic("Error: addRect is not allowed within a text block.")
	}
	for _, value := range []float32{x, y, width, height} {
		if err := c.writeOperandFloat(value); err != nil {
			return err
		}
	}
	return c.writeOperator(operator.AppendRect)
}

// CurveTo appends a cubic Bezier curve to the current path.
func (c *pdAbstractContentStream) CurveTo(x1, y1, x2, y2, x3, y3 float32) error {
	if c.inTextMode {
		panic("Error: curveTo is not allowed within a text block.")
	}
	for _, value := range []float32{x1, y1, x2, y2, x3, y3} {
		if err := c.writeOperandFloat(value); err != nil {
			return err
		}
	}
	return c.writeOperator(operator.CurveTo)
}

// CurveTo2 appends a cubic Bezier curve whose first control point is the
// current point.
func (c *pdAbstractContentStream) CurveTo2(x2, y2, x3, y3 float32) error {
	if c.inTextMode {
		panic("Error: curveTo2 is not allowed within a text block.")
	}
	for _, value := range []float32{x2, y2, x3, y3} {
		if err := c.writeOperandFloat(value); err != nil {
			return err
		}
	}
	return c.writeOperator(operator.CurveToReplicateInitialPoint)
}

// CurveTo1 appends a cubic Bezier curve whose second control point is its end
// point.
func (c *pdAbstractContentStream) CurveTo1(x1, y1, x3, y3 float32) error {
	if c.inTextMode {
		panic("Error: curveTo1 is not allowed within a text block.")
	}
	for _, value := range []float32{x1, y1, x3, y3} {
		if err := c.writeOperandFloat(value); err != nil {
			return err
		}
	}
	return c.writeOperator(operator.CurveToReplicateFinalPoint)
}

// MoveTo begins a new subpath at the given point.
func (c *pdAbstractContentStream) MoveTo(x, y float32) error {
	if c.inTextMode {
		panic("Error: moveTo is not allowed within a text block.")
	}
	if err := c.writeOperandFloat(x); err != nil {
		return err
	}
	if err := c.writeOperandFloat(y); err != nil {
		return err
	}
	return c.writeOperator(operator.MoveTo)
}

// LineTo appends a straight line to the current path.
func (c *pdAbstractContentStream) LineTo(x, y float32) error {
	if c.inTextMode {
		panic("Error: lineTo is not allowed within a text block.")
	}
	if err := c.writeOperandFloat(x); err != nil {
		return err
	}
	if err := c.writeOperandFloat(y); err != nil {
		return err
	}
	return c.writeOperator(operator.LineTo)
}

// Stroke strokes the current path.
func (c *pdAbstractContentStream) Stroke() error {
	if c.inTextMode {
		panic("Error: stroke is not allowed within a text block.")
	}
	return c.writeOperator(operator.StrokePath)
}

// CloseAndStroke closes and strokes the current path.
func (c *pdAbstractContentStream) CloseAndStroke() error {
	if c.inTextMode {
		panic("Error: closeAndStroke is not allowed within a text block.")
	}
	return c.writeOperator(operator.CloseAndStroke)
}

// Fill fills the current path with the nonzero winding rule.
func (c *pdAbstractContentStream) Fill() error {
	if c.inTextMode {
		panic("Error: fill is not allowed within a text block.")
	}
	return c.writeOperator(operator.FillNonZero)
}

// FillEvenOdd fills the current path with the even odd rule.
func (c *pdAbstractContentStream) FillEvenOdd() error {
	if c.inTextMode {
		panic("Error: fillEvenOdd is not allowed within a text block.")
	}
	return c.writeOperator(operator.FillEvenOdd)
}

// FillAndStroke fills and strokes the current path with the nonzero winding
// rule.
func (c *pdAbstractContentStream) FillAndStroke() error {
	if c.inTextMode {
		panic("Error: fillAndStroke is not allowed within a text block.")
	}
	return c.writeOperator(operator.FillNonZeroAndStroke)
}

// FillAndStrokeEvenOdd fills and strokes the current path with the even odd
// rule.
func (c *pdAbstractContentStream) FillAndStrokeEvenOdd() error {
	if c.inTextMode {
		panic("Error: fillAndStrokeEvenOdd is not allowed within a text block.")
	}
	return c.writeOperator(operator.FillEvenOddAndStroke)
}

// CloseAndFillAndStroke closes, fills and strokes the current path with the
// nonzero winding rule.
func (c *pdAbstractContentStream) CloseAndFillAndStroke() error {
	if c.inTextMode {
		panic("Error: closeAndFillAndStroke is not allowed within a text block.")
	}
	return c.writeOperator(operator.CloseFillNonZeroAndStroke)
}

// CloseAndFillAndStrokeEvenOdd closes, fills and strokes the current path with
// the even odd rule.
func (c *pdAbstractContentStream) CloseAndFillAndStrokeEvenOdd() error {
	if c.inTextMode {
		panic("Error: closeAndFillAndStrokeEvenOdd is not allowed within a text block.")
	}
	return c.writeOperator(operator.CloseFillEvenOddAndStroke)
}

// ClosePath closes the current subpath.
func (c *pdAbstractContentStream) ClosePath() error {
	if c.inTextMode {
		panic("Error: closePath is not allowed within a text block.")
	}
	return c.writeOperator(operator.ClosePath)
}

// Clip intersects the clipping path with the current path, with the nonzero
// winding rule.
func (c *pdAbstractContentStream) Clip() error {
	if c.inTextMode {
		panic("Error: clip is not allowed within a text block.")
	}
	if err := c.writeOperator(operator.ClipNonZero); err != nil {
		return err
	}
	// end path without filling or stroking
	return c.writeOperator(operator.Endpath)
}

// ClipEvenOdd intersects the clipping path with the current path, with the even
// odd rule.
func (c *pdAbstractContentStream) ClipEvenOdd() error {
	if c.inTextMode {
		panic("Error: clipEvenOdd is not allowed within a text block.")
	}
	if err := c.writeOperator(operator.ClipEvenOdd); err != nil {
		return err
	}
	// end path without filling or stroking
	return c.writeOperator(operator.Endpath)
}

// SetLineWidth sets the line width.
func (c *pdAbstractContentStream) SetLineWidth(lineWidth float32) error {
	if err := c.writeOperandFloat(lineWidth); err != nil {
		return err
	}
	return c.writeOperator(operator.SetLineWidth)
}

// SetLineJoinStyle sets the line join style.
//
// Java throws IllegalArgumentException for a style outside 0..2, which is
// unchecked, so the port panics.
func (c *pdAbstractContentStream) SetLineJoinStyle(lineJoinStyle int) error {
	if lineJoinStyle < 0 || lineJoinStyle > 2 {
		panic("Error: unknown value for line join style")
	}
	if err := c.writeOperandInt(lineJoinStyle); err != nil {
		return err
	}
	return c.writeOperator(operator.SetLineJoinstyle)
}

// SetLineCapStyle sets the line cap style.
func (c *pdAbstractContentStream) SetLineCapStyle(lineCapStyle int) error {
	if lineCapStyle < 0 || lineCapStyle > 2 {
		panic("Error: unknown value for line cap style")
	}
	if err := c.writeOperandInt(lineCapStyle); err != nil {
		return err
	}
	return c.writeOperator(operator.SetLineCapstyle)
}

// SetLineDashPattern sets the line dash pattern.
func (c *pdAbstractContentStream) SetLineDashPattern(pattern []float32, phase float32) error {
	if err := c.writeBytes(asciiLeftSquareBracket); err != nil {
		return err
	}
	for _, value := range pattern {
		if err := c.writeOperandFloat(value); err != nil {
			return err
		}
	}
	if err := c.writeBytes(asciiRightSquareBracketSpace); err != nil {
		return err
	}
	if err := c.writeOperandFloat(phase); err != nil {
		return err
	}
	return c.writeOperator(operator.SetLineDashpattern)
}

// SetMiterLimit sets the miter limit.
func (c *pdAbstractContentStream) SetMiterLimit(miterLimit float32) error {
	if miterLimit <= 0 {
		panic("A miter limit <= 0 is invalid and will not render in Acrobat Reader")
	}
	if err := c.writeOperandFloat(miterLimit); err != nil {
		return err
	}
	return c.writeOperator(operator.SetLineMiterlimit)
}

// BeginMarkedContent begins a marked content sequence.
func (c *pdAbstractContentStream) BeginMarkedContent(tag *cos.Name) error {
	if err := c.writeOperandName(tag); err != nil {
		return err
	}
	return c.writeOperator(operator.BeginMarkedContent)
}

// BeginMarkedContentWithMCID begins a marked content sequence with the given
// marked content identifier.
func (c *pdAbstractContentStream) BeginMarkedContentWithMCID(tag *cos.Name, mcid int) error {
	if mcid < 0 {
		panic("mcid should not be negative")
	}
	if err := c.writeOperandName(tag); err != nil {
		return err
	}
	if err := c.write("<</MCID " + strconv.Itoa(mcid) + ">> "); err != nil {
		return err
	}
	return c.writeOperator(operator.BeginMarkedContentSeq)
}

// BeginMarkedContentWithProperties begins a marked content sequence with the
// given property list.
func (c *pdAbstractContentStream) BeginMarkedContentWithProperties(tag *cos.Name,
	propertyList markedcontent.PropertyList) error {
	if err := c.writeOperandName(tag); err != nil {
		return err
	}
	dict := propertyList.PropertyDictionary()
	if dict.GetInt(cos.MCID) > -1 && dict.Size() == 1 {
		// PDFBOX-5890: use simplified notation if there's only an MCID
		if err := c.write("<</MCID " + strconv.Itoa(dict.GetInt(cos.MCID)) + ">> "); err != nil {
			return err
		}
	} else {
		if err := c.writeOperandName(c.resources.AddPropertyList(propertyList)); err != nil {
			return err
		}
	}
	return c.writeOperator(operator.BeginMarkedContentSeq)
}

// EndMarkedContent ends a marked content sequence.
func (c *pdAbstractContentStream) EndMarkedContent() error {
	return c.writeOperator(operator.EndMarkedContent)
}

// SetMarkedContentPoint writes a marked content point.
func (c *pdAbstractContentStream) SetMarkedContentPoint(tag *cos.Name) error {
	if err := c.writeOperandName(tag); err != nil {
		return err
	}
	return c.writeOperator(operator.MarkedContentPoint)
}

// SetMarkedContentPointWithProperties writes a marked content point with the
// given property list.
func (c *pdAbstractContentStream) SetMarkedContentPointWithProperties(tag *cos.Name,
	propertyList markedcontent.PropertyList) error {
	if err := c.writeOperandName(tag); err != nil {
		return err
	}
	if err := c.writeOperandName(c.resources.AddPropertyList(propertyList)); err != nil {
		return err
	}
	return c.writeOperator(operator.MarkedContentPointWithProps)
}

// SetGraphicsStateParameters sets the graphics state from the given extended
// graphics state.
func (c *pdAbstractContentStream) SetGraphicsStateParameters(
	extGState *state.PDExtendedGraphicsState) error {
	if err := c.writeOperandName(c.resources.AddExtGState(extGState)); err != nil {
		return err
	}
	return c.writeOperator(operator.SetGraphicsStateParams)
}

// AddComment writes a comment line.
//
// Java throws IllegalArgumentException for a comment holding a newline, which
// is unchecked, so the port panics.
func (c *pdAbstractContentStream) AddComment(comment string) error {
	if strings.ContainsAny(comment, "\n\r") {
		panic("comment should not include a newline")
	}
	if _, err := c.outputStream.Write([]byte{'%'}); err != nil {
		return err
	}
	if _, err := c.outputStream.Write([]byte(comment)); err != nil {
		return err
	}
	_, err := c.outputStream.Write([]byte{'\n'})
	return err
}

// writeOperandFloat writes a real operand. Java declares it protected.
//
// Java throws IllegalArgumentException for a value that is not finite, which is
// unchecked, so the port panics.
func (c *pdAbstractContentStream) writeOperandFloat(real float32) error {
	if isNotFinite(real) {
		panic(fmt.Sprintf("%v is not a finite number", real))
	}
	byteCount := util.FormatFloatFast(real, c.maximumFractionDigits, c.formatBuffer)
	if byteCount == -1 {
		// Fast formatting failed
		if err := c.write(c.formatDecimal(real)); err != nil {
			return err
		}
	} else if _, err := c.outputStream.Write(c.formatBuffer[:byteCount]); err != nil {
		return err
	}
	_, err := c.outputStream.Write(asciiSpace)
	return err
}

// writeOperandInt writes an integer operand. Java declares it protected.
func (c *pdAbstractContentStream) writeOperandInt(integer int) error {
	if err := c.write(strconv.Itoa(integer)); err != nil {
		return err
	}
	_, err := c.outputStream.Write(asciiSpace)
	return err
}

// writeOperandName writes a name operand. Java declares it protected.
func (c *pdAbstractContentStream) writeOperandName(name *cos.Name) error {
	if err := name.WritePDF(c.outputStream); err != nil {
		return err
	}
	_, err := c.outputStream.Write(asciiSpace)
	return err
}

// writeOperator writes an operator and the newline after it. Java declares it
// protected.
func (c *pdAbstractContentStream) writeOperator(operatorName string) error {
	if err := c.writeBytes(operator.GetNameAsBytes(operatorName)); err != nil {
		return err
	}
	return c.writeLine()
}

// write writes the given text as ASCII. Java declares it protected.
func (c *pdAbstractContentStream) write(text string) error {
	return c.writeBytes([]byte(text))
}

// writeLine writes a newline. Java declares it protected.
func (c *pdAbstractContentStream) writeLine() error {
	_, err := c.outputStream.Write([]byte{'\n'})
	return err
}

// writeBytes writes the given bytes. Java declares it protected.
func (c *pdAbstractContentStream) writeBytes(data []byte) error {
	_, err := c.outputStream.Write(data)
	return err
}

// writeAffineTransform writes the six numbers of an affine transform. Java
// declares it private.
func (c *pdAbstractContentStream) writeAffineTransform(transform *geom.AffineTransform) error {
	values := make([]float64, 6)
	transform.GetMatrix(values)
	for _, v := range values {
		if err := c.writeOperandFloat(float32(v)); err != nil {
			return err
		}
	}
	return nil
}

// Close closes the stream.
func (c *pdAbstractContentStream) Close() error {
	if c.inTextMode {
		slog.Warn("pdmodel: you did not call endText(), some viewers won't display your text")
	}
	return c.outputStream.Close()
}

// isOutside255Interval reports whether the given value is outside 0..255. Java
// declares it protected.
func isOutside255Interval(val int) bool {
	return val < 0 || val > 255
}

// isOutsideOneInterval reports whether the given value is outside 0..1. Java
// declares it private and takes a double.
func isOutsideOneInterval(val float32) bool {
	return val < 0 || val > 1
}

// isNotFinite reports whether the given value is not a finite number, which is
// what Float.isFinite answers.
func isNotFinite(value float32) bool {
	return value != value || value > maxFiniteFloat32 || value < -maxFiniteFloat32
}

// maxFiniteFloat32 is the largest finite float32.
const maxFiniteFloat32 = 3.4028234663852886e38

// formatDecimal writes a real the way Java's NumberFormat for Locale.US does
// with grouping off and the maximum fraction digits this stream carries: the
// digits are rounded to that many places and the trailing zeros dropped.
func (c *pdAbstractContentStream) formatDecimal(value float32) string {
	text := strconv.FormatFloat(float64(value), 'f', c.maximumFractionDigits, 32)
	if strings.Contains(text, ".") {
		text = strings.TrimRight(text, "0")
		text = strings.TrimSuffix(text, ".")
	}
	return text
}

// setStrokingColorSpaceStack replaces the top of the stroking colour space
// stack. Java declares it protected.
func (c *pdAbstractContentStream) setStrokingColorSpaceStack(colorSpace color.PDColorSpace) {
	if len(c.strokingColorSpaceStack) == 0 {
		c.strokingColorSpaceStack = append(c.strokingColorSpaceStack, colorSpace)
		return
	}
	c.strokingColorSpaceStack[len(c.strokingColorSpaceStack)-1] = colorSpace
}

// setNonStrokingColorSpaceStack replaces the top of the non-stroking colour
// space stack. Java declares it protected.
func (c *pdAbstractContentStream) setNonStrokingColorSpaceStack(colorSpace color.PDColorSpace) {
	if len(c.nonStrokingColorSpaceStack) == 0 {
		c.nonStrokingColorSpaceStack = append(c.nonStrokingColorSpaceStack, colorSpace)
		return
	}
	c.nonStrokingColorSpaceStack[len(c.nonStrokingColorSpaceStack)-1] = colorSpace
}

// SetCharacterSpacing sets the character spacing.
func (c *pdAbstractContentStream) SetCharacterSpacing(spacing float32) error {
	if err := c.writeOperandFloat(spacing); err != nil {
		return err
	}
	return c.writeOperator(operator.SetCharSpacing)
}

// SetWordSpacing sets the word spacing.
func (c *pdAbstractContentStream) SetWordSpacing(spacing float32) error {
	if err := c.writeOperandFloat(spacing); err != nil {
		return err
	}
	return c.writeOperator(operator.SetWordSpacing)
}

// SetHorizontalScaling sets the horizontal scaling of the text.
func (c *pdAbstractContentStream) SetHorizontalScaling(scale float32) error {
	if err := c.writeOperandFloat(scale); err != nil {
		return err
	}
	return c.writeOperator(operator.SetTextHorizontalScaling)
}

// SetRenderingMode sets the text rendering mode.
func (c *pdAbstractContentStream) SetRenderingMode(rm state.RenderingMode) error {
	if err := c.writeOperandInt(rm.IntValue()); err != nil {
		return err
	}
	return c.writeOperator(operator.SetTextRenderingmode)
}

// SetTextRise sets the text rise.
func (c *pdAbstractContentStream) SetTextRise(rise float32) error {
	if err := c.writeOperandFloat(rise); err != nil {
		return err
	}
	return c.writeOperator(operator.SetTextRise)
}

// encodeForGsub encodes the given text through the substitution rules of its
// script. Java declares it private.
func (c *pdAbstractContentStream) encodeForGsub(gsubWorker gsub.GsubWorker,
	glyphIds map[int]bool, f *font.PDType0Font, text string) ([]byte, error) {
	out := bytes.Buffer{}
	for _, word := range util.TokenizeOnSpace(text) {
		if len([]rune(word)) == 1 && strings.TrimSpace(word) == "" {
			encoded, err := f.Encode(word)
			if err != nil {
				return nil, err
			}
			out.Write(encoded)
			continue
		}
		applied, err := c.applyGSUBRules(gsubWorker, &out, f, word)
		if err != nil {
			return nil, err
		}
		for _, glyphID := range applied {
			glyphIds[glyphID] = true
		}
	}
	return out.Bytes(), nil
}

// applyGSUBRules writes one word through the substitution rules of its script,
// and returns the glyph ids it wrote. Java declares it private.
//
// Java throws IllegalStateException for a character the font has no glyph for,
// which is unchecked, so the port panics.
func (c *pdAbstractContentStream) applyGSUBRules(gsubWorker gsub.GsubWorker,
	out *bytes.Buffer, f *font.PDType0Font, word string) ([]int, error) {
	codePoints := []rune(word)
	originalGlyphIds := make([]int, 0, len(codePoints))
	cmapLookup := f.CmapLookup()

	// convert characters into glyph IDs
	for _, codePoint := range codePoints {
		glyphID := cmapLookup.GetGlyphID(int(codePoint))
		if glyphID <= 0 {
			panic(fmt.Sprintf("could not find the glyphId for the character: %s, codePoint: %d (0x%X)",
				string(codePoint), codePoint, codePoint))
		}
		originalGlyphIds = append(originalGlyphIds, glyphID)
	}

	// transform glyph IDs, write them to the output stream
	glyphIdsAfterGsub := gsubWorker.ApplyTransforms(originalGlyphIds)
	for _, glyphID := range glyphIdsAfterGsub {
		out.Write(f.EncodeGlyphID(glyphID))
	}
	return glyphIdsAfterGsub, nil
}
