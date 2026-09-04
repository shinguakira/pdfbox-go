package font

import (
	"bytes"
	"fmt"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/fontbox"
	fontutil "github.com/shinguakira/pdfbox-go/go/fontbox/util"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font/encoding"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// altNames are alternative names for glyphs which are commonly encountered.
var altNames = map[string]string{
	"ff":       "f_f",
	"ffi":      "f_f_i",
	"ffl":      "f_f_l",
	"fi":       "f_i",
	"fl":       "f_l",
	"st":       "s_t",
	"IJ":       "I_J",
	"ij":       "i_j",
	"ellipsis": "elipsis", // misspelled in ArialMT
}

// PDType1Font is a Type 1 font, which is what the fourteen standard fonts are.
//
// Port of org.apache.pdfbox.pdmodel.font.PDType1Font.
//
// Two things this slice does not carry: the embedded PFB font program, which
// needs fontbox/type1, and the substitute the system supplies for a font that
// is not embedded, which needs the font mapper chain. Both are slice 4, so
// genericFont is nil here and every path that reads a glyph outline or a width
// out of the font program fails rather than guessing. The widths of a standard
// 14 font come from its AFM, which is what this slice is built round. See
// migration/STATUS.md.
type PDType1Font struct {
	pdSimpleFont

	genericFont         fontbox.FontBoxFont
	isEmbedded          bool
	isDamaged           bool
	fontMatrixTransform *geom.AffineTransform
	codeToBytesMap      map[int][]byte
	fontMatrix          *util.Matrix
	fontBBox            *fontutil.BoundingBox
}

var (
	_ PDFont       = (*PDType1Font)(nil)
	_ PDSimpleFont = (*PDType1Font)(nil)
)

// NewPDType1FontStandard14 returns one of the fourteen standard fonts.
func NewPDType1FontStandard14(baseFont FontName) (*PDType1Font, error) {
	base, err := newPDFontStandard14(baseFont)
	if err != nil {
		return nil, err
	}
	f := &PDType1Font{
		pdSimpleFont:   pdSimpleFont{pdFont: base},
		codeToBytesMap: map[int][]byte{},
	}
	f.pdFont.self = f
	f.selfSimple = f
	f.assignGlyphList(baseFont)

	f.dict.SetItem(cos.Subtype, cos.Type1)
	f.dict.SetName(cos.BaseFont, baseFont.Name())
	switch baseFont {
	case ZapfDingbatsFontName:
		f.encoding = encoding.ZapfDingbatsEncodingInstance
	case SymbolFontName:
		f.encoding = encoding.SymbolEncodingInstance
	default:
		f.encoding = encoding.WinAnsiEncodingInstance
		f.dict.SetItem(cos.Encoding, cos.WinAnsiEncoding)
	}

	// todo: could load the PFB font here if we wanted to support Standard 14
	// embedding
	//
	// Java asks the font mapper for a substitute to draw the glyphs with; that
	// is slice 4, so there is none here.
	f.genericFont = nil
	f.isEmbedded = false
	f.isDamaged = false
	f.fontMatrixTransform = geom.NewIdentityTransform()
	return f, nil
}

// NewPDType1FontFromDictionary returns the Type 1 font the given dictionary
// describes.
func NewPDType1FontFromDictionary(fontDictionary *cos.Dictionary, resourceCache ResourceCache) (*PDType1Font, error) {
	f := &PDType1Font{
		pdSimpleFont:   pdSimpleFont{pdFont: newPDFontFromDictionary(fontDictionary)},
		codeToBytesMap: map[int][]byte{},
	}
	f.pdFont.self = f
	f.selfSimple = f
	f.initFromDictionary(resourceCache)

	fd := f.FontDescriptor()
	fontIsDamaged := false
	if fd != nil {
		// a Type1 font may contain a Type1C font
		if fontFile3 := fd.FontFile3(); fontFile3 != nil {
			// /FontFile3 for Type1 font not supported
			_ = fontFile3
		}
		// or it may contain a PFB
		//
		// Reading it needs fontbox/type1, which slice 4 ports; until then an
		// embedded Type 1 font is read as damaged, which is how Java treats one
		// it cannot parse. See migration/STATUS.md.
		if fontFile := fd.FontFile(); fontFile != nil {
			fontIsDamaged = true
		}
	}
	f.isEmbedded = false
	f.isDamaged = fontIsDamaged

	// Java finds a generic font to render with here, through the font mapper;
	// that is slice 4.
	f.genericFont = nil

	if err := f.readEncoding(); err != nil {
		return nil, err
	}
	f.fontMatrixTransform = f.FontMatrix().CreateAffineTransform()
	f.fontMatrixTransform.Scale(1000, 1000)
	return f, nil
}

// BaseFont returns the /BaseFont entry of the font dictionary.
func (f *PDType1Font) BaseFont() string {
	return f.dict.GetNameAsString(cos.BaseFont, "")
}

// Name returns the name of the font as the PDF gives it.
func (f *PDType1Font) Name() string { return f.BaseFont() }

// Height returns how tall the given glyph is.
func (f *PDType1Font) Height(code int) (float32, error) {
	if afmStandard14 := f.standard14AFM(); afmStandard14 != nil {
		afmName := f.Encoding().Name(code)
		// todo: isn't this the y-advance, not the height?
		return afmStandard14.CharacterHeight(afmName), nil
	}
	// Java measures the glyph outline of the substitute font, which this slice
	// does not have.
	return 0, errNoFontProgram
}

// errNoFontProgram is what every path that needs the font program returns while
// the font mapper and the Type 1 parser are unported.
var errNoFontProgram = fmt.Errorf("font: no font program: the Type 1 parser and the font mapper are not ported yet")

// encodeCodePoint returns the bytes that draw the given code point.
func (f *PDType1Font) encodeCodePoint(unicode int) ([]byte, error) {
	if b, ok := f.codeToBytesMap[unicode]; ok {
		return b, nil
	}

	name := f.GlyphList().CodePointToName(unicode)
	if f.IsStandard14() {
		// genericFont not needed, thus simplified code
		// this is important on systems with no installed fonts
		if !f.encoding.ContainsName(name) {
			return nil, fmt.Errorf("font: U+%04X ('%s') is not available in the font %s, encoding: %s",
				unicode, name, f.Name(), f.encoding.EncodingName())
		}
		if name == ".notdef" {
			return nil, fmt.Errorf("font: No glyph for U+%04X in the font %s", unicode, f.Name())
		}
	} else {
		if !f.encoding.ContainsName(name) {
			return nil, fmt.Errorf("font: U+%04X ('%s') is not available in the font %s, encoding: %s",
				unicode, name, f.Name(), f.encoding.EncodingName())
		}
		nameInFont, err := f.getNameInFont(name)
		if err != nil {
			return nil, err
		}
		hasGlyph := false
		if f.genericFont != nil {
			hasGlyph, err = f.genericFont.HasGlyph(nameInFont)
			if err != nil {
				return nil, err
			}
		}
		if nameInFont == ".notdef" || !hasGlyph {
			return nil, fmt.Errorf("font: No glyph for U+%04X in the font %s", unicode, f.Name())
		}
	}

	code, ok := f.encoding.NameToCodeMap()[name]
	if !ok || code < 0 {
		return nil, fmt.Errorf("font: U+%04X ('%s') is not available in the font %s, encoding: %s",
			unicode, name, f.Name(), f.encoding.EncodingName())
	}
	b := []byte{byte(code)}
	f.codeToBytesMap[unicode] = b
	return b, nil
}

// WidthFromFont returns the width the font program gives for the glyph.
func (f *PDType1Font) WidthFromFont(code int) (float32, error) {
	name, err := f.CodeToName(code)
	if err != nil {
		return 0, err
	}
	// width of .notdef is ignored for substitutes, see PDFBOX-1900
	if !f.isEmbedded && name == ".notdef" {
		return 250, nil
	}
	if f.genericFont == nil {
		return 0, errNoFontProgram
	}
	width, err := f.genericFont.GetWidth(name)
	if err != nil {
		return 0, err
	}
	p := geom.NewPointFloat(width, 0)
	f.fontMatrixTransform.Transform(p, p)
	return float32(p.X()), nil
}

// IsEmbedded reports whether the font program is inside the PDF.
func (f *PDType1Font) IsEmbedded() bool { return f.isEmbedded }

// AverageFontWidth returns the average width of the glyphs.
func (f *PDType1Font) AverageFontWidth() float32 {
	if afmStandard14 := f.standard14AFM(); afmStandard14 != nil {
		return afmStandard14.AverageCharacterWidth()
	}
	return f.pdFont.AverageFontWidth()
}

// ReadCode reads one character code from the stream.
func (f *PDType1Font) ReadCode(in *bytes.Reader) (int, error) {
	b, err := in.ReadByte()
	if err != nil {
		// Java's InputStream.read returns -1 at the end rather than throwing.
		return -1, nil
	}
	return int(b), nil
}

// readEncodingFromFont returns the encoding built into the font program.
func (f *PDType1Font) readEncodingFromFont() (encoding.Encoding, error) {
	if !f.IsEmbedded() && f.standard14AFM() != nil {
		// read from AFM
		return encoding.NewType1EncodingFromMetrics(f.standard14AFM()), nil
	}
	// extract from Type1 font/substitute
	if encodedFont, ok := f.genericFont.(fontbox.EncodedFont); ok {
		return encoding.Type1EncodingFromFontBox(encodedFont.Encoding()), nil
	}
	// default (only happens with TTFs)
	return encoding.StandardEncodingInstance, nil
}

// FontBoxFont returns the font program the glyphs are drawn from.
func (f *PDType1Font) FontBoxFont() fontbox.FontBoxFont { return f.genericFont }

// BoundingBox returns the box every glyph of the font fits in.
func (f *PDType1Font) BoundingBox() (*fontutil.BoundingBox, error) {
	if f.fontBBox == nil {
		bbox, err := f.generateBoundingBox()
		if err != nil {
			return nil, err
		}
		f.fontBBox = bbox
	}
	return f.fontBBox, nil
}

func (f *PDType1Font) generateBoundingBox() (*fontutil.BoundingBox, error) {
	if fd := f.FontDescriptor(); fd != nil {
		bbox := fd.FontBoundingBox()
		if isNonZeroBoundingBox(bbox) {
			return fontutil.NewBoundingBoxOf(bbox.LowerLeftX(), bbox.LowerLeftY(),
				bbox.UpperRightX(), bbox.UpperRightY()), nil
		}
	}
	if f.genericFont == nil {
		return nil, errNoFontProgram
	}
	return f.genericFont.FontBBox()
}

// CodeToName returns the name the font program knows the given glyph by.
func (f *PDType1Font) CodeToName(code int) (string, error) {
	name := ".notdef"
	if f.Encoding() != nil {
		name = f.Encoding().Name(code)
	}
	return f.getNameInFont(name)
}

// getNameInFont returns the name the font program knows the glyph by, which is
// not always the name the encoding gives.
func (f *PDType1Font) getNameInFont(name string) (string, error) {
	if f.IsEmbedded() {
		return name, nil
	}
	if f.genericFont == nil {
		// Java asks the substitute font here; with none, nothing else in this
		// method can answer either.
		return name, nil
	}
	hasGlyph, err := f.genericFont.HasGlyph(name)
	if err != nil {
		return "", err
	}
	if hasGlyph {
		return name, nil
	}

	// try alternative name
	if altName, ok := altNames[name]; ok && name != ".notdef" {
		hasGlyph, err := f.genericFont.HasGlyph(altName)
		if err != nil {
			return "", err
		}
		if hasGlyph {
			return altName, nil
		}
	}

	// try unicode name
	unicodes := f.GlyphList().ToUnicode(name)
	if runes := []rune(unicodes); len(runes) == 1 {
		uniName := getUniNameOfCodePoint(int(runes[0]))
		hasGlyph, err := f.genericFont.HasGlyph(uniName)
		if err != nil {
			return "", err
		}
		if hasGlyph {
			return uniName, nil
		}
		// PDFBOX-4017: no postscript table on Windows 10, and the low uni00NN
		// names are not found in Symbol font. What works is using the PDF code
		// plus 0xF000 while disregarding encoding from the PDF (because of file
		// from PDFBOX-1606, makes sense because this segment is about finding
		// the name in a standard font)
		//TODO bring up better solution than this
		genericName, err := f.genericFont.Name()
		if err != nil {
			return "", err
		}
		if genericName == "SymbolMT" {
			if code, ok := encoding.SymbolEncodingInstance.NameToCodeMap()[name]; ok {
				uniName = getUniNameOfCodePoint(code + 0xF000)
				hasGlyph, err := f.genericFont.HasGlyph(uniName)
				if err != nil {
					return "", err
				}
				if hasGlyph {
					return uniName, nil
				}
			}
		}
	}
	return ".notdef", nil
}

// GetPathByName returns the outline of the named glyph.
func (f *PDType1Font) GetPathByName(name string) (*geom.Path2D, error) {
	// Acrobat does not draw .notdef for Type 1 fonts, see PDFBOX-2421
	// I suspect that it does do this for embedded fonts though, but this is
	// untested
	if name == ".notdef" && !f.isEmbedded {
		return geom.NewPathFloat(), nil
	}
	if f.genericFont == nil {
		return nil, errNoFontProgram
	}
	nameInFont, err := f.getNameInFont(name)
	if err != nil {
		return nil, err
	}
	return f.genericFont.GetPath(nameInFont)
}

// GetPath returns the outline of the given glyph.
func (f *PDType1Font) GetPath(code int) (*geom.Path2D, error) {
	name := f.Encoding().Name(code)
	return f.GetPathByName(name)
}

// GetNormalizedPath returns the outline of the given glyph, scaled so that the
// font matrix is the default one.
func (f *PDType1Font) GetNormalizedPath(code int) (*geom.Path2D, error) {
	name := f.Encoding().Name(code)
	path, err := f.GetPathByName(name)
	if err != nil {
		return nil, err
	}
	if path == nil {
		return f.GetPathByName(".notdef")
	}
	return path, nil
}

// HasGlyphByName reports whether the font has the named glyph.
func (f *PDType1Font) HasGlyphByName(name string) (bool, error) {
	if f.genericFont == nil {
		return false, errNoFontProgram
	}
	nameInFont, err := f.getNameInFont(name)
	if err != nil {
		return false, err
	}
	return f.genericFont.HasGlyph(nameInFont)
}

// HasGlyphForCode reports whether the font has an outline for the glyph.
func (f *PDType1Font) HasGlyphForCode(code int) (bool, error) {
	return f.Encoding().Name(code) != ".notdef", nil
}

// FontMatrix returns the transform from glyph space to text space.
func (f *PDType1Font) FontMatrix() *util.Matrix {
	if f.fontMatrix == nil {
		// PDF specified that Type 1 fonts use a 1000upem matrix, but some fonts
		// specify their own custom matrix anyway, for example PDFBOX-2298
		var numbers []float32
		if f.genericFont != nil {
			if got, err := f.genericFont.FontMatrix(); err == nil {
				numbers = got
			} else {
				f.fontMatrix = defaultFontMatrix
			}
		}
		if len(numbers) == 6 {
			f.fontMatrix = util.NewMatrixOf(numbers[0], numbers[1], numbers[2],
				numbers[3], numbers[4], numbers[5])
		} else {
			return f.pdFont.FontMatrix()
		}
	}
	return f.fontMatrix
}

// IsDamaged reports whether the font program could not be read.
func (f *PDType1Font) IsDamaged() bool { return f.isDamaged }
