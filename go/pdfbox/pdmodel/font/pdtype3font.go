package font

import (
	"bytes"
	"fmt"
	"math"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/fontbox"
	fontutil "github.com/shinguakira/pdfbox-go/go/fontbox/util"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font/encoding"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// PDType3Font is a font whose glyphs are content streams rather than outlines.
//
// Port of org.apache.pdfbox.pdmodel.font.PDType3Font.
//
// Java's getResources returns a PDResources, which lives in pdmodel; pdmodel
// imports this package for PDFont, so this one cannot import it back. The port
// hands out the resource dictionary and the cache instead, and pdmodel wraps
// them. See migration/STATUS.md.
type PDType3Font struct {
	pdSimpleFont

	charProcs     *cos.Dictionary
	fontMatrix    *util.Matrix
	fontBBox      *fontutil.BoundingBox
	resourceCache ResourceCache
}

var (
	_ PDFont       = (*PDType3Font)(nil)
	_ PDSimpleFont = (*PDType3Font)(nil)
)

// NewPDType3Font returns the Type 3 font the given dictionary describes.
func NewPDType3Font(fontDictionary *cos.Dictionary, resourceCache ResourceCache) (*PDType3Font, error) {
	f := &PDType3Font{
		pdSimpleFont: pdSimpleFont{pdFont: newPDFontFromDictionary(fontDictionary)},
	}
	f.pdFont.self = f
	f.selfSimple = f
	f.initFromDictionary(resourceCache)
	f.resourceCache = resourceCache
	if err := f.readEncoding(); err != nil {
		return nil, err
	}
	return f, nil
}

// Name returns the name of the font as the PDF gives it.
func (f *PDType3Font) Name() string { return f.dict.GetNameAsString(cos.NameKey, "") }

// readEncoding works out how the character codes of the font map onto glyph
// names. A Type 3 font names every glyph in its own encoding.
func (f *PDType3Font) readEncoding() error {
	encodingBase := f.dict.GetDictionaryObject(cos.Encoding)
	switch value := encodingBase.(type) {
	case *cos.Name:
		f.encoding = encoding.GetInstance(value)
		// an unknown encoding is left nil, as it is in Java
	case *cos.Dictionary:
		f.encoding = encoding.NewDictionaryEncodingForType3(value)
	}
	f.glyphList = encoding.AdobeGlyphList()
	return nil
}

// readEncodingFromFont panics: Type 3 fonts do not have a built-in encoding.
func (f *PDType3Font) readEncodingFromFont() (encoding.Encoding, error) {
	panic("not supported for Type 3 fonts")
}

// isFontSymbolic reports that a Type 3 font is not symbolic.
func (f *PDType3Font) isFontSymbolic() (bool, bool) { return false, true }

// GetPathByName panics: Type 3 fonts do not use vector paths.
func (f *PDType3Font) GetPathByName(name string) (*geom.Path2D, error) {
	panic("not supported for Type 3 fonts")
}

// HasGlyphByName reports whether the font has a char proc for the named glyph.
func (f *PDType3Font) HasGlyphByName(name string) (bool, error) {
	cp := f.CharProcs()
	return cp != nil && cosStream(cp, cos.GetPDFName(name)) != nil, nil
}

// FontBoxFont panics: Type 3 fonts do not use FontBox fonts.
func (f *PDType3Font) FontBoxFont() fontbox.FontBoxFont {
	panic("not supported for Type 3 fonts")
}

// Displacement returns how far the pen moves after the given glyph, in text
// space.
func (f *PDType3Font) Displacement(code int) (util.Vector, error) {
	width, err := f.Width(code)
	if err != nil {
		return util.Vector{}, err
	}
	return f.FontMatrix().TransformVector(util.NewVector(width, 0)), nil
}

// Width returns how far the pen moves after the given glyph.
func (f *PDType3Font) Width(code int) (float32, error) {
	firstChar := f.dict.GetIntDefault(cos.FirstChar, -1)
	lastChar := f.dict.GetIntDefault(cos.LastChar, -1)
	widths := f.getWidths()
	if len(widths) != 0 && code >= firstChar && code <= lastChar {
		if code-firstChar >= len(widths) {
			return 0, nil
		}
		w := widths[code-firstChar]
		if w == nil {
			return 0, nil
		}
		return *w, nil
	}
	if fd := f.FontDescriptor(); fd != nil {
		return fd.MissingWidth(), nil
	}
	return f.WidthFromFont(code)
}

// WidthFromFont returns the width the glyph's own content stream declares.
func (f *PDType3Font) WidthFromFont(code int) (float32, error) {
	charProc := f.CharProc(code)
	if charProc == nil {
		return 0, nil
	}
	length, err := charProc.Stream().Length()
	if err != nil {
		return 0, err
	}
	if length == 0 {
		return 0, nil
	}
	return charProc.Width()
}

// IsEmbedded reports that a Type 3 font is always inside the PDF.
func (f *PDType3Font) IsEmbedded() bool { return true }

// Height returns how tall the given glyph is.
func (f *PDType3Font) Height(code int) (float32, error) {
	desc := f.FontDescriptor()
	if desc == nil {
		return 0, nil
	}
	// the following values are all more or less accurate at least all are
	// average values. Maybe we'll find another way to get those value for every
	// single glyph in the future if needed
	var retval float32
	if bbox := desc.FontBoundingBox(); bbox != nil {
		retval = bbox.Height() / 2
	}
	if retval == 0 {
		retval = desc.CapHeight()
	}
	if retval == 0 {
		retval = desc.Ascent()
	}
	if retval == 0 {
		retval = desc.XHeight()
		if retval > 0 {
			retval -= desc.Descent()
		}
	}
	return retval, nil
}

// encodeCodePoint panics: a Type 3 font cannot be written to.
func (f *PDType3Font) encodeCodePoint(unicode int) ([]byte, error) {
	panic("Not implemented: Type3")
}

// ReadCode reads one character code from the stream.
func (f *PDType3Font) ReadCode(in *bytes.Reader) (int, error) {
	b, err := in.ReadByte()
	if err != nil {
		// Java's InputStream.read returns -1 at the end rather than throwing.
		return -1, nil
	}
	return int(b), nil
}

// FontMatrix returns the transform from glyph space to text space, which a
// Type 3 font always gives itself.
func (f *PDType3Font) FontMatrix() *util.Matrix {
	if f.fontMatrix == nil {
		matrix := f.dict.GetCOSArray(cos.FontMatrix)
		if checkFontMatrixValues(matrix) {
			f.fontMatrix = util.CreateMatrix(matrix)
		} else {
			f.fontMatrix = f.pdFont.FontMatrix()
		}
	}
	return f.fontMatrix
}

// checkFontMatrixValues reports whether the array is six numbers.
func checkFontMatrixValues(matrix *cos.Array) bool {
	if matrix == nil || matrix.Size() != 6 {
		return false
	}
	for _, value := range matrix.ToNumberFloatList() {
		if value == nil {
			return false
		}
	}
	return true
}

// IsDamaged reports that a Type 3 font is never damaged: there's no font file
// to load.
func (f *PDType3Font) IsDamaged() bool { return false }

// IsStandard14 reports that a Type 3 font is never one of the fourteen.
func (f *PDType3Font) IsStandard14() bool { return false }

// ResourcesDictionary returns the resource dictionary of the font, or nil where
// it has none.
func (f *PDType3Font) ResourcesDictionary() *cos.Dictionary {
	return f.dict.GetCOSDictionary(cos.Resources)
}

// ResourceCache returns the cache the font's resources are read through.
func (f *PDType3Font) ResourceCache() ResourceCache { return f.resourceCache }

// FontBBox returns the /FontBBox entry of the font, or nil where it has none.
func (f *PDType3Font) FontBBox() *common.PDRectangle {
	bBox := f.dict.GetCOSArray(cos.FontBBox)
	if bBox == nil {
		return nil
	}
	return common.NewPDRectangleOfCOSArray(bBox)
}

// BoundingBox returns the box every glyph of the font fits in.
func (f *PDType3Font) BoundingBox() (*fontutil.BoundingBox, error) {
	if f.fontBBox == nil {
		f.fontBBox = f.generateBoundingBox()
	}
	return f.fontBBox, nil
}

func (f *PDType3Font) generateBoundingBox() *fontutil.BoundingBox {
	rect := f.FontBBox()
	if rect == nil {
		// FontBBox missing, returning empty rectangle
		return fontutil.NewBoundingBox()
	}
	if !isNonZeroBoundingBox(rect) {
		// Plan B: get the max bounding box of the glyphs
		if cp := f.CharProcs(); cp != nil {
			for _, name := range cp.KeySet() {
				typ3CharProcStream := cosStream(cp, name)
				if typ3CharProcStream == nil {
					continue
				}
				charProc := NewPDType3CharProc(f, typ3CharProcStream)
				glyphBBox, err := charProc.GlyphBBox()
				if err != nil || glyphBBox == nil {
					// error getting the glyph bounding box - font bounding box
					// will be used
					continue
				}
				rect.SetLowerLeftX(minFloat32(rect.LowerLeftX(), glyphBBox.LowerLeftX()))
				rect.SetLowerLeftY(minFloat32(rect.LowerLeftY(), glyphBBox.LowerLeftY()))
				rect.SetUpperRightX(maxFloat32(rect.UpperRightX(), glyphBBox.UpperRightX()))
				rect.SetUpperRightY(maxFloat32(rect.UpperRightY(), glyphBBox.UpperRightY()))
			}
		}
	}
	return fontutil.NewBoundingBoxOf(rect.LowerLeftX(), rect.LowerLeftY(),
		rect.UpperRightX(), rect.UpperRightY())
}

func minFloat32(a, b float32) float32 { return float32(math.Min(float64(a), float64(b))) }
func maxFloat32(a, b float32) float32 { return float32(math.Max(float64(a), float64(b))) }

// CharProcs returns the dictionary of the glyph content streams, or nil where
// the font has none.
func (f *PDType3Font) CharProcs() *cos.Dictionary {
	if f.charProcs == nil {
		f.charProcs = f.dict.GetCOSDictionary(cos.CharProcs)
	}
	return f.charProcs
}

// CharProc returns the content stream that draws the given glyph, or nil where
// the font has none for it.
func (f *PDType3Font) CharProc(code int) *PDType3CharProc {
	if f.Encoding() == nil || f.CharProcs() == nil {
		return nil
	}
	name := f.Encoding().Name(code)
	stream := cosStream(f.CharProcs(), cos.GetPDFName(name))
	if stream == nil {
		return nil
	}
	return NewPDType3CharProc(f, stream)
}

// cosStream returns the stream under the given key, or nil where there is none.
//
// Java's COSDictionary.getCOSStream; the port's cos.Dictionary does not carry
// it, so the assertion is written out here.
func cosStream(d *cos.Dictionary, key *cos.Name) *cos.Stream {
	stream, _ := d.GetDictionaryObject(key).(*cos.Stream)
	return stream
}

// unused keeps fmt in play for the errors this file's neighbours return.
var _ = fmt.Errorf
