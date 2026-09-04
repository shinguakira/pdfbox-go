package font

import (
	"bytes"
	"fmt"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/fontbox/afm"
	fontutil "github.com/shinguakira/pdfbox-go/go/fontbox/util"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font/encoding"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// PDFontLike is what everything that behaves like a font offers, whether it is
// a font in its own right or one CIDFont inside a Type 0 font.
//
// Port of org.apache.pdfbox.pdmodel.font.PDFontLike.
type PDFontLike interface {
	// Name returns the name of the font as the PDF gives it.
	Name() string

	// FontDescriptor returns what the PDF says about the font, or nil where it
	// says nothing.
	FontDescriptor() *PDFontDescriptor

	// FontMatrix returns the transform from glyph space to text space.
	FontMatrix() *util.Matrix

	// BoundingBox returns the box every glyph of the font fits in.
	BoundingBox() (*fontutil.BoundingBox, error)

	// PositionVector returns where a vertically set glyph sits relative to its
	// origin.
	PositionVector(code int) util.Vector

	// Height returns how tall the given glyph is.
	//
	// Deprecated: Java marks this deprecated without saying what to use
	// instead.
	Height(code int) (float32, error)

	// Width returns how far the pen moves after the given glyph.
	Width(code int) (float32, error)

	// HasExplicitWidth reports whether the PDF gives a width for the glyph
	// itself, rather than leaving it to the font program.
	HasExplicitWidth(code int) (bool, error)

	// WidthFromFont returns the width the font program gives for the glyph.
	WidthFromFont(code int) (float32, error)

	// IsEmbedded reports whether the font program is inside the PDF.
	IsEmbedded() bool

	// IsDamaged reports whether the font program could not be read.
	IsDamaged() bool

	// AverageFontWidth returns the average width of the glyphs.
	//
	// todo: this method is highly suspicious, the average glyph width is not
	// usually a good metric
	AverageFontWidth() float32
}

// PDVectorFont is a font whose glyphs are outlines.
//
// Port of org.apache.pdfbox.pdmodel.font.PDVectorFont. Rendering an outline
// needs the glyph renderers, which a later slice ports; the interface is here
// because the font classes name it. See migration/STATUS.md.
type PDVectorFont interface {
	// GetPath returns the outline of the given glyph, in glyph space.
	GetPath(code int) (*geom.Path2D, error)

	// GetNormalizedPath returns the outline of the given glyph, scaled so that
	// the font matrix is the default one.
	GetNormalizedPath(code int) (*geom.Path2D, error)

	// HasGlyphForCode reports whether the font has an outline for the glyph.
	HasGlyphForCode(code int) (bool, error)
}

// PDFont is a font of a PDF document.
//
// Port of the abstract org.apache.pdfbox.pdmodel.font.PDFont. Java has the
// concrete fonts extend it; the port keeps the shared state in pdFont, which
// each of them embeds, and puts what varies behind this interface.
type PDFont interface {
	PDFontLike

	// COSObject returns the font dictionary.
	COSObject() cos.Base

	// Dictionary returns the font dictionary, typed.
	Dictionary() *cos.Dictionary

	// Displacement returns how far the pen moves after the given glyph, in
	// text space.
	Displacement(code int) (util.Vector, error)

	// Encode returns the bytes that draw the given text with this font.
	Encode(text string) ([]byte, error)

	// StringWidth returns how wide the given text is in this font.
	StringWidth(text string) (float32, error)

	// ReadCode reads one character code from the stream.
	ReadCode(in *bytes.Reader) (int, error)

	// ToUnicode returns what the given character code stands for, or the empty
	// string where the font cannot say. Java returns null there.
	ToUnicode(code int) (string, error)

	// ToUnicodeWithGlyphList returns what the given character code stands for,
	// reading unknown glyph names through the given list.
	ToUnicodeWithGlyphList(code int, customGlyphList *encoding.GlyphList) (string, error)

	// Type returns the /Type entry of the font dictionary.
	Type() string

	// SubType returns the /Subtype entry of the font dictionary.
	SubType() string

	// SpaceWidth returns how wide a space is in this font.
	SpaceWidth() float32

	// IsVertical reports whether the font is set vertically.
	IsVertical() bool

	// IsStandard14 reports whether the font is one of the fourteen every
	// reader has.
	IsStandard14() bool

	// standard14Width returns the width the metrics of a standard 14 font give
	// for the glyph. Java's protected abstract getStandard14Width.
	standard14Width(code int) float32

	// encodeCodePoint returns the bytes that draw the given code point. Java's
	// protected abstract encode(int).
	encodeCodePoint(unicode int) ([]byte, error)

	// base returns the shared part, which is how the concrete fonts reach it.
	base() *pdFont
}

// defaultFontMatrix is the transform a font uses unless it says otherwise.
var defaultFontMatrix = util.NewMatrixOf(0.001, 0, 0, 0.001, 0, 0)

// pdFont is the state every font carries.
type pdFont struct {
	dict *cos.Dictionary

	// self is the font this state belongs to, which is how the shared methods
	// reach the ones each font implements. Java gets this from virtual
	// dispatch; Go embedding does not, so the font hands itself over.
	self PDFont

	afmStandard14    *afm.FontMetrics
	fontDescriptor   *PDFontDescriptor
	widths           []*float32
	widthsRead       bool
	avgFontWidth     float32
	fontWidthOfSpace float32
	codeToWidthMap   map[int]float32
}

// newPDFont returns the state of a font built from nothing, which is what a
// font being written from scratch starts with.
func newPDFont() pdFont {
	dict := cos.NewDictionary()
	dict.SetItem(cos.Type, cos.Font)
	return pdFont{
		dict:             dict,
		fontWidthOfSpace: -1,
		codeToWidthMap:   map[int]float32{},
	}
}

// newPDFontStandard14 returns the state of one of the fourteen standard fonts.
func newPDFontStandard14(baseFont FontName) (pdFont, error) {
	dict := cos.NewDictionary()
	dict.SetItem(cos.Type, cos.Font)
	afmStandard14 := GetAFM(baseFont.Name())
	if afmStandard14 == nil {
		return pdFont{}, fmt.Errorf("font: No AFM for font %s", baseFont)
	}
	return pdFont{
		dict:           dict,
		afmStandard14:  afmStandard14,
		fontDescriptor: buildFontDescriptor(afmStandard14),
		// standard 14 fonts may be accessed concurrently, as they are
		// singletons
		fontWidthOfSpace: -1,
		codeToWidthMap:   map[int]float32{},
	}, nil
}

// newPDFontFromDictionary returns the state of a font read out of a PDF.
//
// The concrete font sets self and then calls initFromDictionary: Java does the
// whole of this in one constructor, which can call the abstract getName because
// Java dispatches virtually from a constructor and Go does not.
func newPDFontFromDictionary(fontDictionary *cos.Dictionary) pdFont {
	return pdFont{
		dict:             fontDictionary,
		fontWidthOfSpace: -1,
		codeToWidthMap:   map[int]float32{},
	}
}

// initFromDictionary finishes what newPDFontFromDictionary starts, once self is
// set.
func (f *pdFont) initFromDictionary(resourceCache ResourceCache) {
	// standard 14 fonts use an AFM
	f.afmStandard14 = GetAFM(f.self.Name()) // may be nil (it usually is)
	f.fontDescriptor = f.loadFontDescriptor(resourceCache)
	// The ToUnicode CMap is read by a later slice, which ports fontbox/cmap;
	// see migration/STATUS.md. Until then ToUnicode falls through to whatever
	// each font works out from its encoding.
}

// loadFontDescriptor reads the font descriptor, through the cache where there
// is one.
func (f *pdFont) loadFontDescriptor(resourceCache ResourceCache) *PDFontDescriptor {
	fdIndirectObject := f.dict.GetCOSObject(cos.FontDescriptor)
	if fdIndirectObject != nil && resourceCache != nil {
		if pdFontDescriptor := resourceCache.GetFontDescriptor(fdIndirectObject); pdFontDescriptor != nil {
			return pdFontDescriptor
		}
	}
	fd := f.dict.GetCOSDictionary(cos.FontDescriptor)
	if fd != nil {
		pdFontDescriptor := NewPDFontDescriptorFromDictionary(fd)
		if resourceCache != nil && fdIndirectObject != nil {
			resourceCache.PutFontDescriptor(fdIndirectObject, pdFontDescriptor)
		}
		return pdFontDescriptor
	} else if f.afmStandard14 != nil {
		// build font descriptor from the AFM
		return buildFontDescriptor(f.afmStandard14)
	}
	return nil
}

// standard14AFM returns the metrics of the standard 14 font this font is, or
// nil where it is not one.
func (f *pdFont) standard14AFM() *afm.FontMetrics { return f.afmStandard14 }

// FontDescriptor returns what the PDF says about the font.
func (f *pdFont) FontDescriptor() *PDFontDescriptor { return f.fontDescriptor }

// setFontDescriptor sets what the PDF says about the font.
func (f *pdFont) setFontDescriptor(fontDescriptor *PDFontDescriptor) {
	f.fontDescriptor = fontDescriptor
}

// COSObject returns the font dictionary.
func (f *pdFont) COSObject() cos.Base { return f.dict }

// Dictionary returns the font dictionary, typed.
func (f *pdFont) Dictionary() *cos.Dictionary { return f.dict }

// base returns the shared part of the font.
func (f *pdFont) base() *pdFont { return f }

// PositionVector panics: a horizontally set font has no position vector.
func (f *pdFont) PositionVector(code int) util.Vector {
	panic("Horizontal fonts have no position vector")
}

// Displacement returns how far the pen moves after the given glyph.
func (f *pdFont) Displacement(code int) (util.Vector, error) {
	width, err := f.Width(code)
	if err != nil {
		return util.Vector{}, err
	}
	return util.NewVector(width/1000, 0), nil
}

// Width returns how far the pen moves after the given glyph.
func (f *pdFont) Width(code int) (float32, error) {
	if width, ok := f.codeToWidthMap[code]; ok {
		return width, nil
	}

	// Acrobat overrides the widths in the font program on the conforming
	// reader's system with the widths specified in the font dictionary."
	// (Adobe Supplement to the ISO 32000)
	//
	// Note: The Adobe Supplement says that the override happens "If the font
	// program is not embedded", however PDFBOX-427 shows that it also applies
	// to embedded fonts.

	// Type1, Type1C, Type3
	if f.dict.GetDictionaryObject(cos.Widths) != nil || f.dict.ContainsKey(cos.MissingWidth) {
		firstChar := f.dict.GetIntDefault(cos.FirstChar, -1)
		lastChar := f.dict.GetIntDefault(cos.LastChar, -1)
		siz := len(f.getWidths())
		idx := code - firstChar
		if siz > 0 && code >= firstChar && code <= lastChar && idx < siz {
			var width float32
			if w := f.getWidths()[idx]; w != nil {
				width = *w
			}
			f.codeToWidthMap[code] = width
			return width, nil
		}
		if fd := f.self.FontDescriptor(); fd != nil {
			// get entry from /MissingWidth entry
			width := fd.MissingWidth()
			f.codeToWidthMap[code] = width
			return width, nil
		}
	}

	// standard 14 font widths are specified by an AFM
	if f.self.IsStandard14() {
		width := f.self.standard14Width(code)
		f.codeToWidthMap[code] = width
		return width, nil
	}

	// if there's nothing to override with, then obviously we fall back to the
	// font
	width, err := f.self.WidthFromFont(code)
	if err != nil {
		return 0, err
	}
	f.codeToWidthMap[code] = width
	return width, nil
}

// Encode returns the bytes that draw the given text with this font.
func (f *pdFont) Encode(text string) ([]byte, error) {
	var out bytes.Buffer
	out.Grow(max(32, len(text)))
	for _, codePoint := range text {
		// multi-byte encoding with 1 to 4 bytes
		b, err := f.self.encodeCodePoint(int(codePoint))
		if err != nil {
			return nil, err
		}
		out.Write(b)
	}
	return out.Bytes(), nil
}

// StringWidth returns how wide the given text is in this font.
func (f *pdFont) StringWidth(text string) (float32, error) {
	b, err := f.Encode(text)
	if err != nil {
		return 0, err
	}
	in := bytes.NewReader(b)
	var width float32
	for in.Len() > 0 {
		code, err := f.self.ReadCode(in)
		if err != nil {
			return 0, err
		}
		w, err := f.self.Width(code)
		if err != nil {
			return 0, err
		}
		width += w
	}
	return width, nil
}

// AverageFontWidth returns the average width of the glyphs.
func (f *pdFont) AverageFontWidth() float32 {
	// todo: this method is highly suspicious, the average glyph width is not
	// usually a good metric
	if f.avgFontWidth != 0 {
		return f.avgFontWidth
	}
	var totalWidth float32
	var characterCount float32
	widths := f.dict.GetCOSArray(cos.Widths)
	if widths != nil {
		for i := 0; i < widths.Size(); i++ {
			if fontWidth, ok := widths.GetObject(i).(cos.Number); ok {
				floatValue := fontWidth.FloatValue()
				if floatValue > 0 {
					totalWidth += floatValue
					characterCount++
				}
			}
		}
	}
	var average float32
	if totalWidth > 0 {
		average = totalWidth / characterCount
	}
	f.avgFontWidth = average
	return average
}

// ToUnicodeWithGlyphList returns what the given character code stands for.
func (f *pdFont) ToUnicodeWithGlyphList(code int, customGlyphList *encoding.GlyphList) (string, error) {
	return f.self.ToUnicode(code)
}

// ToUnicode returns what the given character code stands for, or the empty
// string where the font cannot say.
//
// The ToUnicode CMap path is not ported: it needs fontbox/cmap, which a later
// slice brings. Without it this always falls through, which is the same as a
// font that carries no /ToUnicode entry; see migration/STATUS.md.
func (f *pdFont) ToUnicode(code int) (string, error) {
	// if no value has been produced, there is no way to obtain Unicode for the
	// character. this behaviour can be overridden in the concrete fonts, but
	// this method *must* return nothing here
	return "", nil
}

// Type returns the /Type entry of the font dictionary.
func (f *pdFont) Type() string { return f.dict.GetNameAsString(cos.Type, "") }

// SubType returns the /Subtype entry of the font dictionary.
func (f *pdFont) SubType() string { return f.dict.GetNameAsString(cos.Subtype, "") }

// getWidths returns the /Widths array, read once.
func (f *pdFont) getWidths() []*float32 {
	if !f.widthsRead {
		if array := f.dict.GetCOSArray(cos.Widths); array != nil {
			f.widths = array.ToNumberFloatList()
		} else {
			f.widths = nil
		}
		f.widthsRead = true
	}
	return f.widths
}

// FontMatrix returns the transform from glyph space to text space.
func (f *pdFont) FontMatrix() *util.Matrix { return defaultFontMatrix }

// SpaceWidth returns how wide a space is in this font.
func (f *pdFont) SpaceWidth() float32 {
	if f.fontWidthOfSpace != -1 {
		return f.fontWidthOfSpace
	}
	// Java catches every exception here and falls back to 250; the port does
	// the same with the errors that stand in for them.
	//
	// The /ToUnicode branch needs fontbox/cmap and is not ported; a font that
	// carries one therefore takes the encoding branch, which is what Java does
	// for a font that carries none.
	//
	// Java catches IllegalArgumentException and UnsupportedOperationException
	// round this one call -- "Happens if space is not available in the font or
	// if encoding isn't implemented". A Type 3 font's encode throws the second
	// outright, so the recover is not an edge case: it is the ordinary path for
	// every Type 3 font.
	if width, err := f.stringWidthOfSpace(); err == nil {
		// PDFBOX-5920: try with encoding, which gets the correct code
		f.fontWidthOfSpace = width
	}
	if f.fontWidthOfSpace <= 0 {
		width, err := f.self.Width(32)
		if err != nil {
			f.fontWidthOfSpace = 250
			return f.fontWidthOfSpace
		}
		f.fontWidthOfSpace = width
	}
	// try to get it from the font itself
	if f.fontWidthOfSpace <= 0 {
		width, err := f.self.WidthFromFont(32)
		if err != nil {
			f.fontWidthOfSpace = 250
			return f.fontWidthOfSpace
		}
		f.fontWidthOfSpace = width
		// use the average font width as fall back
		if f.fontWidthOfSpace <= 0 {
			f.fontWidthOfSpace = f.self.AverageFontWidth()
		}
	}
	return f.fontWidthOfSpace
}

// IsStandard14 reports whether the font is one of the fourteen every reader
// has.
func (f *pdFont) IsStandard14() bool {
	// this logic is based on Acrobat's behaviour, see PDFBOX-2372
	// embedded fonts never get special treatment
	if f.self.IsEmbedded() {
		return false
	}
	// if the name matches, this is a Standard 14 font
	return Standard14ContainsName(f.self.Name())
}

// String returns the font written out, as Java's toString does.
func (f *pdFont) String() string {
	return fmt.Sprintf("%T %s", f.self, f.self.Name())
}

// readCMap would read a CMap from the given object.
//
// Not ported: it needs fontbox/cmap, which a later slice brings. See
// migration/STATUS.md.
func (f *pdFont) readCMap(base cos.Base) error {
	return fmt.Errorf("font: CMaps are not ported yet")
}

// buildFontDescriptor builds a font descriptor out of AFM metrics, which is how
// a standard 14 font gets one.
//
// Port of org.apache.pdfbox.pdmodel.font.PDType1FontEmbedder.buildFontDescriptor.
// The rest of that class writes a font into a document, which a later slice
// ports; see migration/STATUS.md.
func buildFontDescriptor(metrics *afm.FontMetrics) *PDFontDescriptor {
	isSymbolic := metrics.EncodingScheme() == "FontSpecific"

	fd := NewPDFontDescriptor()
	fd.SetFontName(metrics.FontName())
	fd.SetFontFamily(metrics.FamilyName())
	fd.SetNonSymbolic(!isSymbolic)
	fd.SetSymbolic(isSymbolic)
	fd.SetFontBoundingBox(common.NewPDRectangleOfBoundingBox(metrics.FontBBox()))
	fd.SetItalicAngle(metrics.ItalicAngle())
	fd.SetAscent(metrics.Ascender())
	fd.SetDescent(metrics.Descender())
	fd.SetCapHeight(metrics.CapHeight())
	fd.SetXHeight(metrics.XHeight())
	fd.SetAverageWidth(metrics.AverageCharacterWidth())
	fd.SetCharacterSet(metrics.CharacterSet())
	fd.SetStemV(0) // for PDF/A
	return fd
}

// stringWidthOfSpace measures a single space, turning the panic a font that
// cannot encode one raises into an error.
//
// Java writes this as a try/catch round getStringWidth(" ") inside
// getSpaceWidth, catching IllegalArgumentException and
// UnsupportedOperationException. Both are unchecked, which this port maps to a
// panic, so the catch maps to a recover.
func (f *pdFont) stringWidthOfSpace() (width float32, err error) {
	defer func() {
		if r := recover(); r != nil {
			width = 0
			err = fmt.Errorf("font: %v", r)
		}
	}()
	return f.self.StringWidth(" ")
}
