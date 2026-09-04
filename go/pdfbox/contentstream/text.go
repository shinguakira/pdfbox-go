package contentstream

import (
	"bytes"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// ShowTextString draws the given string with the current font.
//
// Port of org.apache.pdfbox.contentstream.PDFStreamEngine.showTextString.
func (e *PDFStreamEngine) ShowTextString(str []byte) error {
	return e.showText(str)
}

// ShowTextStrings draws the strings of a TJ array, moving the pen by the
// numbers between them.
//
// Port of showTextStrings.
func (e *PDFStreamEngine) ShowTextStrings(array *cos.Array) error {
	textState := e.GraphicsState().TextState()
	fontSize := textState.FontSize()
	horizontalScaling := textState.HorizontalScaling() / 100
	currentFont := textState.Font()
	isVertical := false
	if currentFont != nil {
		isVertical = currentFont.IsVertical()
	}

	for i := 0; i < array.Size(); i++ {
		obj := array.GetObject(i)
		switch value := obj.(type) {
		case cos.Number:
			tj := value.FloatValue()
			// calculate the combined displacements
			var tx, ty float32
			if isVertical {
				tx = 0
				ty = -tj / 1000 * fontSize
			} else {
				tx = -tj / 1000 * fontSize * horizontalScaling
				ty = 0
			}
			e.overrides.ApplyTextAdjustment(tx, ty)
		case *cos.StringObj:
			if err := e.showText(value.Bytes()); err != nil {
				return err
			}
		default:
			// a nested array, or anything else, is an error Java logs and skips
		}
	}
	return nil
}

// ApplyTextAdjustment moves the pen by the given amount, which is what the
// numbers of a TJ array ask for.
//
// Port of applyTextAdjustment.
func (e *PDFStreamEngine) ApplyTextAdjustment(tx, ty float32) {
	// update the text matrix
	e.GraphicsState().TextMatrix().Translate(tx, ty)
}

// showText decodes the string one code at a time and draws each glyph.
//
// Port of showText.
func (e *PDFStreamEngine) showText(str []byte) error {
	graphicsState := e.GraphicsState()
	textState := graphicsState.TextState()

	// get the current font
	currentFont := textState.Font()
	if currentFont == nil {
		// No current font, will use default
		var err error
		currentFont, err = e.DefaultFont()
		if err != nil {
			return err
		}
	}
	fontSize := textState.FontSize()
	horizontalScaling := textState.HorizontalScaling() / 100
	charSpacing := textState.CharacterSpacing()

	// put the text state parameters into matrix form
	parameters := util.NewMatrixOf(
		fontSize*horizontalScaling, 0, // 0
		0, fontSize, // 0
		0, textState.Rise()) // 1

	textMatrix := graphicsState.TextMatrix()

	// read the stream until it is empty
	bais := bytes.NewReader(str)
	for before := bais.Len(); before > 0; before = bais.Len() {
		// decode a character
		code, err := currentFont.ReadCode(bais)
		if err != nil {
			return err
		}
		codeLength := before - bais.Len()

		// Word spacing shall be applied to every occurrence of the single-byte
		// character code 32 in a string when using a simple font or a composite
		// font that defines code 32 as a single-byte code.
		var wordSpacing float32
		if codeLength == 1 && code == 32 {
			wordSpacing += textState.WordSpacing()
		}

		// text rendering matrix (text space -> device space)
		ctm := graphicsState.CurrentTransformationMatrix()
		textRenderingMatrix := parameters.Multiply(textMatrix).Multiply(ctm)

		// get glyph's position vector if this is vertical text
		// changes to vertical text should be tested with PDFBOX-2294 and
		// PDFBOX-1422
		if currentFont.IsVertical() {
			// position vector, in text space
			v := currentFont.PositionVector(code)
			// apply the position vector to the horizontal origin to get the
			// vertical origin
			textRenderingMatrix.TranslateVector(v)
		}

		// get glyph's horizontal and vertical displacements, in text space
		w, err := currentFont.Displacement(code)
		if err != nil {
			return err
		}

		// process the decoded glyph
		if err := e.overrides.ShowGlyph(textRenderingMatrix, currentFont, code, w); err != nil {
			return err
		}

		// calculate the combined displacements
		var tx, ty float32
		if currentFont.IsVertical() {
			tx = 0
			ty = w.Y()*fontSize + charSpacing + wordSpacing
		} else {
			tx = (w.X()*fontSize + charSpacing + wordSpacing) * horizontalScaling
			ty = 0
		}

		// update the text matrix
		textMatrix.Translate(tx, ty)
	}
	return nil
}

// ShowGlyph is called for each glyph the engine decodes.
//
// Port of showGlyph.
func (e *PDFStreamEngine) ShowGlyph(textRenderingMatrix *util.Matrix, f font.PDFont, code int, displacement util.Vector) error {
	if type3, ok := f.(*font.PDType3Font); ok {
		return e.overrides.ShowType3Glyph(textRenderingMatrix, type3, code, displacement)
	}
	return e.overrides.ShowFontGlyph(textRenderingMatrix, f, code, displacement)
}

// ShowFontGlyph is called for each glyph of a font that is not Type 3. The
// engine itself does nothing with it; an embedder overrides it.
//
// Port of showFontGlyph.
func (e *PDFStreamEngine) ShowFontGlyph(textRenderingMatrix *util.Matrix, f font.PDFont, code int, displacement util.Vector) error {
	// overridden in subclasses
	return nil
}

// ShowType3Glyph runs the content stream that draws one glyph of a Type 3 font.
//
// Port of showType3Glyph.
func (e *PDFStreamEngine) ShowType3Glyph(textRenderingMatrix *util.Matrix, f *font.PDType3Font, code int, displacement util.Vector) error {
	charProc := f.CharProc(code)
	if charProc != nil {
		return e.ProcessType3Stream(charProc, textRenderingMatrix)
	}
	return nil
}

// ProcessType3Stream runs the content stream of one Type 3 glyph.
//
// Port of processType3Stream.
func (e *PDFStreamEngine) ProcessType3Stream(charProc *font.PDType3CharProc, textRenderingMatrix *util.Matrix) error {
	if e.currentPage == nil {
		panic("No current page, call #processChildStream(PDContentStream, PDPage) instead")
	}
	contentStream := newType3CharProcStream(charProc)
	parent := e.pushResources(contentStream)
	savedStack := e.SaveGraphicsStack()
	graphicsState := e.GraphicsState()

	// replace the CTM with the TRM
	graphicsState.SetCurrentTransformationMatrix(textRenderingMatrix)

	// transform the CTM using the stream's matrix (this is the FontMatrix)
	textRenderingMatrix.Concatenate(charProc.Matrix())

	// note: we don't clip to the BBox as it is often wrong, see PDFBOX-1917

	graphicsState.SetTextMatrix(util.NewMatrix())
	graphicsState.SetTextLineMatrix(util.NewMatrix())

	err := e.processStreamOperators(contentStream)
	e.RestoreGraphicsStack(savedStack)
	e.popResources(parent)
	return err
}

// DefaultFont returns the font the engine draws with where the content stream
// set none.
//
// Port of getDefaultFont.
func (e *PDFStreamEngine) DefaultFont() (font.PDFont, error) {
	if e.defaultFont == nil {
		defaultFont, err := font.NewPDType1FontStandard14(font.Helvetica)
		if err != nil {
			return nil, err
		}
		e.defaultFont = defaultFont
	}
	return e.defaultFont, nil
}

// type3CharProcStream is a Type 3 char proc seen as a content stream.
//
// font.PDType3CharProc cannot implement PDContentStream itself: the interface
// names pdmodel.PDResources, and pdmodel imports the font package. The adapter
// lives here, where both are in view. See migration/STATUS.md.
type type3CharProcStream struct {
	charProc *font.PDType3CharProc
}

var _ PDContentStream = (*type3CharProcStream)(nil)

func newType3CharProcStream(charProc *font.PDType3CharProc) *type3CharProcStream {
	return &type3CharProcStream{charProc: charProc}
}

// ContentsForRandomAccess returns the content of the char proc.
func (s *type3CharProcStream) ContentsForRandomAccess() (pdfio.RandomAccessRead, error) {
	return s.charProc.ContentsForRandomAccess()
}

// Resources returns the resources the glyph is drawn against.
func (s *type3CharProcStream) Resources() *pdmodel.PDResources {
	dict := s.charProc.ResourcesDictionary()
	if dict == nil {
		return nil
	}
	return pdmodel.NewPDResourcesOfCache(dict, s.charProc.Font().ResourceCache())
}

// BBox returns the box the glyph is drawn in.
func (s *type3CharProcStream) BBox() *common.PDRectangle { return s.charProc.BBox() }

// Matrix returns the transform from glyph space to text space.
func (s *type3CharProcStream) Matrix() *util.Matrix { return s.charProc.Matrix() }
