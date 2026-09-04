package text

import (
	"math"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream/operator/markedcontent"
	statepr "github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream/operator/state"
	textpr "github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream/operator/text"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font/encoding"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/resources"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// legacyGlyphList is the Adobe Glyph List with the additional mappings this
// package ships, which is what the text extractor reads glyph names through.
var legacyGlyphList = func() *encoding.GlyphList {
	// load additional glyph list for Unicode mapping
	path := "glyphlist/additional.txt"
	input, err := resources.Open(path)
	if err != nil {
		panic(err)
	}
	defer input.Close()
	glyphList, err := encoding.NewGlyphListFrom(encoding.AdobeGlyphList(), input)
	if err != nil {
		panic(err)
	}
	return glyphList
}()

// LegacyPDFStreamEngine turns the glyphs of a page into TextPositions.
//
// Port of org.apache.pdfbox.text.LegacyPDFStreamEngine. Java marks the
// calculations in showGlyph as deliberately incorrect: they are what the text
// stripper has always been built on, and changing them would change the text it
// produces.
//
// The DrawObject processor Java registers -- the text version, which walks into
// a form XObject -- is not here: XObjects are a slice this port has not reached.
// A page whose text is inside a form therefore yields nothing from it. See
// migration/STATUS.md.
type LegacyPDFStreamEngine struct {
	*contentstream.PDFStreamEngine

	pageRotation    int
	pageSize        *common.PDRectangle
	translateMatrix *util.Matrix

	fontHeightMap map[*cos.Dictionary]float32

	// processTextPosition is what the stripper installs to receive each
	// position. Java overrides a protected method; Go embedding does not
	// dispatch, so the receiver is a field.
	processTextPosition func(text *TextPosition) error
}

// NewLegacyPDFStreamEngine returns an engine with the operators the text
// extractor needs registered.
func NewLegacyPDFStreamEngine() *LegacyPDFStreamEngine {
	e := &LegacyPDFStreamEngine{
		PDFStreamEngine: contentstream.NewPDFStreamEngine(),
		fontHeightMap:   map[*cos.Dictionary]float32{},
	}
	e.SetOverrides(e)

	// Java lists each processor it wants; the port registers the three packages
	// it has, which between them cover every operator Java names here and adds
	// the marked content ones the extractor subclasses use.
	statepr.AddAll(e.PDFStreamEngine)
	textpr.AddAll(e.PDFStreamEngine)
	markedcontent.AddAll(e.PDFStreamEngine)
	return e
}

// SetProcessTextPosition installs what receives each text position the engine
// works out.
func (e *LegacyPDFStreamEngine) SetProcessTextPosition(f func(text *TextPosition) error) {
	e.processTextPosition = f
}

// ProcessPage walks the content of the page, working out where each glyph was
// drawn.
func (e *LegacyPDFStreamEngine) ProcessPage(page *pdmodel.PDPage) error {
	e.pageRotation = page.Rotation()
	e.pageSize = page.CropBox()

	if e.pageSize.LowerLeftX() == 0 && e.pageSize.LowerLeftY() == 0 {
		e.translateMatrix = nil
	} else {
		// translation matrix for cropbox
		e.translateMatrix = util.TranslateInstance(-e.pageSize.LowerLeftX(), -e.pageSize.LowerLeftY())
	}
	return e.PDFStreamEngine.ProcessPage(page)
}

// ShowGlyph works out where the glyph was drawn and how big it is, and hands a
// TextPosition to whatever is receiving them.
//
// Port of showGlyph. Java's warning stands: DO NOT USE THIS CODE UNLESS YOU ARE
// WORKING WITH PDFTextStripper. THIS CODE IS DELIBERATELY INCORRECT.
func (e *LegacyPDFStreamEngine) ShowGlyph(textRenderingMatrix *util.Matrix, f font.PDFont, code int, displacement util.Vector) error {
	//
	// legacy calculations which were previously in PDFStreamEngine
	//
	//  DO NOT USE THIS CODE UNLESS YOU ARE WORKING WITH PDFTextStripper.
	//  THIS CODE IS DELIBERATELY INCORRECT
	//
	graphicsState := e.GraphicsState()
	ctm := graphicsState.CurrentTransformationMatrix()
	fontSize := graphicsState.TextState().FontSize()
	horizontalScaling := graphicsState.TextState().HorizontalScaling() / 100
	textMatrix := e.TextMatrix()

	displacementX := displacement.X()
	// the sorting algorithm is based on the width of the character. As the
	// displacement for vertical characters doesn't provide any suitable value
	// for it, we have to calculate our own
	if f.IsVertical() {
		width, err := f.Width(code)
		if err != nil {
			return err
		}
		displacementX = width / 1000
		// there may be an additional scaling factor for true type fonts
		if ttfFont, ok := f.(*font.PDTrueTypeFont); ok {
			if ttf := ttfFont.TrueTypeFont(); ttf != nil {
				unitsPerEm, err := ttf.UnitsPerEm()
				if err != nil {
					return err
				}
				if unitsPerEm != 1000 {
					displacementX *= 1000 / float32(unitsPerEm)
				}
			}
		}
		// the Type 0 branch Java also has needs the CID fonts, which a later
		// slice brings
	}

	//
	// legacy calculations which were previously in PDFStreamEngine
	//
	//  DO NOT USE THIS CODE UNLESS YOU ARE WORKING WITH PDFTextStripper.
	//  THIS CODE IS DELIBERATELY INCORRECT
	//

	// (modified) combined displacement, this is calculated *without* taking the
	// character spacing and word spacing into account, due to legacy code in
	// TextStripper
	tx := displacementX * fontSize * horizontalScaling
	ty := displacement.Y() * fontSize

	// (modified) combined displacement matrix
	td := util.TranslateInstance(tx, ty)

	// (modified) text rendering matrix
	nextTextRenderingMatrix := td.Multiply(textMatrix).Multiply(ctm) // text space -> device space
	nextX := nextTextRenderingMatrix.TranslateX()
	nextY := nextTextRenderingMatrix.TranslateY()

	// (modified) width and height calculations
	dxDisplay := nextX - textRenderingMatrix.TranslateX()
	fontHeight, ok := e.fontHeightMap[f.Dictionary()]
	if !ok {
		var err error
		fontHeight, err = e.ComputeFontHeight(f)
		if err != nil {
			return err
		}
		e.fontHeightMap[f.Dictionary()] = fontHeight
	}
	dyDisplay := fontHeight * textRenderingMatrix.ScalingFactorY()

	//
	// start of the original method
	//

	// Note on variable names. There are three different units being used in
	// this code. Character sizes are given in glyph units, text locations are
	// initially given in text units, and we want to save the data in display
	// units. The variable names should end with Text or Disp to represent if
	// the values are in text or disp units (no glyph units are saved).

	glyphSpaceToTextSpaceFactor := float32(1) / 1000
	if _, ok := f.(*font.PDType3Font); ok {
		glyphSpaceToTextSpaceFactor = f.FontMatrix().ScaleX()
	}

	// to avoid crash as described in PDFBOX-614, see what the space
	// displacement should be. Java catches every exception here.
	spaceWidthText := f.SpaceWidth() * glyphSpaceToTextSpaceFactor

	if spaceWidthText == 0 {
		spaceWidthText = f.AverageFontWidth() * glyphSpaceToTextSpaceFactor
		// the average space width appears to be higher than necessary so make
		// it smaller
		spaceWidthText *= .80
	}
	if spaceWidthText == 0 {
		spaceWidthText = 1.0 // if could not find font, use a generic value
	}

	// the space width has to be transformed into display units
	spaceWidthDisplay := spaceWidthText * textRenderingMatrix.ScalingFactorX()

	// use our additional glyph list for Unicode mapping
	unicodeStr, err := f.ToUnicodeWithGlyphList(code, legacyGlyphList)
	if err != nil {
		return err
	}

	// when there is no Unicode mapping available, Acrobat simply coerces the
	// character code into Unicode, so we do the same. Subclasses of
	// PDFStreamEngine don't necessarily want this, which is why we leave it
	// until this point in PDFTextStreamEngine.
	if unicodeStr == "" {
		if _, ok := f.(font.PDSimpleFont); ok {
			unicodeStr = string(rune(code))
		} else {
			// Acrobat doesn't seem to coerce composite font's character codes,
			// instead it skips them. See the "allah2.pdf" TestTextStripper file.
			return nil
		}
	}

	// adjust for cropbox if needed
	translatedTextRenderingMatrix := textRenderingMatrix
	if e.translateMatrix != nil {
		translatedTextRenderingMatrix = util.Concatenate(e.translateMatrix, textRenderingMatrix)
		nextX -= e.pageSize.LowerLeftX()
		nextY -= e.pageSize.LowerLeftY()
	}

	position := NewTextPosition(e.pageRotation, e.pageSize.Width(), e.pageSize.Height(),
		translatedTextRenderingMatrix, nextX, nextY,
		abs32(dyDisplay), dxDisplay,
		abs32(spaceWidthDisplay), unicodeStr, []int{code}, f, fontSize,
		int(fontSize*textMatrix.ScalingFactorX()))

	if e.processTextPosition != nil {
		return e.processTextPosition(position)
	}
	return nil
}

// ComputeFontHeight works out how tall the glyphs of the font are, in text
// space.
func (e *LegacyPDFStreamEngine) ComputeFontHeight(f font.PDFont) (float32, error) {
	bbox, err := f.BoundingBox()
	if err != nil {
		return 0, err
	}
	if bbox.LowerLeftY() < math.MinInt16 {
		// PDFBOX-2158 and PDFBOX-3130
		// files by Salmat eSolutions / ClibPDF Library
		bbox.SetLowerLeftY(-(bbox.LowerLeftY() + 65536))
	}

	// 1/2 the bbox is used as the height todo: why?
	glyphHeight := bbox.Height() / 2

	// sometimes the bbox has very high values, but CapHeight is OK
	if fontDescriptor := f.FontDescriptor(); fontDescriptor != nil {
		capHeight := fontDescriptor.CapHeight()
		if capHeight != 0 && (capHeight < glyphHeight || glyphHeight == 0) {
			glyphHeight = capHeight
		}
		// PDFBOX-3464, PDFBOX-4480, PDFBOX-4553:
		// sometimes even CapHeight has very high value, but Ascent and Descent
		// are ok
		ascent := fontDescriptor.Ascent()
		descent := fontDescriptor.Descent()
		if capHeight > ascent && ascent > 0 && descent < 0 &&
			((ascent-descent)/2 < glyphHeight || glyphHeight == 0) {
			glyphHeight = (ascent - descent) / 2
		}
	}

	// transformPoint from glyph space -> text space
	if _, ok := f.(*font.PDType3Font); ok {
		return float32(f.FontMatrix().TransformPoint(0, glyphHeight).Y()), nil
	}
	return glyphHeight / 1000, nil
}
