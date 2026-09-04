package font

import (
	"bytes"
	"fmt"
	"log/slog"
	"unicode/utf16"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/fontbox"
	"github.com/shinguakira/pdfbox-go/go/fontbox/cff"
	fontutil "github.com/shinguakira/pdfbox-go/go/fontbox/util"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font/encoding"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// PDType1CFont is a Type 1-equivalent CFF font.
//
// Port of org.apache.pdfbox.pdmodel.font.PDType1CFont.
//
// The substitute the system supplies for a font that is not embedded needs the
// font mapper chain (B6); until that lands genericFont is nil for such a font
// and every path that reads the font program fails rather than guessing. See
// migration/STATUS.md.
type PDType1CFont struct {
	pdSimpleFont

	glyphHeights        map[string]float32
	fontMatrixTransform *geom.AffineTransform
	cffFont             *cff.CFFType1Font   // embedded font
	genericFont         fontbox.FontBoxFont // embedded or system font for rendering
	isEmbedded          bool
	isDamaged           bool
	avgWidth            float32
	avgWidthSet         bool
	fontMatrix          *util.Matrix
	fontBBox            *fontutil.BoundingBox
}

var (
	_ PDFont       = (*PDType1CFont)(nil)
	_ PDSimpleFont = (*PDType1CFont)(nil)
	_ PDVectorFont = (*PDType1CFont)(nil)
)

// NewPDType1CFontFromDictionary returns the Type 1C font the given dictionary
// describes.
func NewPDType1CFontFromDictionary(fontDictionary *cos.Dictionary,
	resourceCache ResourceCache) (*PDType1CFont, error) {
	f := &PDType1CFont{
		pdSimpleFont: pdSimpleFont{pdFont: newPDFontFromDictionary(fontDictionary)},
		glyphHeights: map[string]float32{},
	}
	f.pdFont.self = f
	f.selfSimple = f
	f.initFromDictionary(resourceCache)

	fontIsDamaged := false
	var cffEmbedded *cff.CFFType1Font
	fd := f.FontDescriptor()
	if fd != nil {
		if ff3Stream := fd.FontFile3(); ff3Stream != nil {
			parsed, damaged, err := readEmbeddedCFF(ff3Stream.Stream(), f.Name())
			if err != nil {
				slog.Error("Can't read the embedded Type1C font", "font", f.Name(), "err", err)
				fontIsDamaged = true
			} else {
				cffEmbedded = parsed
				fontIsDamaged = damaged
			}
		}
	}
	f.isDamaged = fontIsDamaged
	f.cffFont = cffEmbedded
	if f.cffFont != nil {
		f.genericFont = f.cffFont
		f.isEmbedded = true
	} else {
		// Java asks the font mapper for a substitute here; that is B6, so an
		// unembedded font has none.
		f.genericFont = nil
		f.isEmbedded = false
	}
	if err := f.readEncoding(); err != nil {
		return nil, err
	}
	f.fontMatrixTransform = f.FontMatrix().CreateAffineTransform()
	f.fontMatrixTransform.Scale(1000, 1000)
	return f, nil
}

// readEmbeddedCFF parses the /FontFile3 stream. It gives the font where it
// holds a Type 1-equivalent CFF one, and otherwise says whether that leaves the
// font damaged: an empty stream is logged and left unembedded, anything that is
// not a CFFType1Font is damaged.
func readEmbeddedCFF(stream *cos.Stream, name string) (*cff.CFFType1Font, bool, error) {
	randomAccessRead, err := stream.CreateView()
	if err != nil {
		return nil, false, err
	}
	defer pdfio.CloseQuietly(randomAccessRead)
	length, err := randomAccessRead.Length()
	if err != nil {
		return nil, false, err
	}
	if length == 0 {
		slog.Error("Invalid data for embedded Type1C font", "font", name)
		return nil, false, nil
	}
	// note: this could be an OpenType file, fortunately CFFParser can handle
	// that
	fonts, err := cff.NewCFFParser().Parse(randomAccessRead)
	if err != nil {
		return nil, false, err
	}
	if len(fonts) == 0 {
		// Java indexes the list and throws IndexOutOfBounds on an empty parse,
		// which the catch below turns into a damaged font.
		return nil, true, nil
	}
	if type1, ok := fonts[0].(*cff.CFFType1Font); ok {
		return type1, false, nil
	}
	slog.Error("Expected CFFType1Font", "got", fmt.Sprintf("%T", fonts[0]))
	return nil, true, nil
}

// FontBoxFont returns the font program the glyphs are drawn from.
func (f *PDType1CFont) FontBoxFont() fontbox.FontBoxFont { return f.genericFont }

// BaseFont returns the PostScript name of the font.
func (f *PDType1CFont) BaseFont() string {
	return f.dict.GetNameAsString(cos.BaseFont, "")
}

// Name returns the name of the font as the PDF gives it.
func (f *PDType1CFont) Name() string { return f.BaseFont() }

// GetPathByName returns the outline of the named glyph.
func (f *PDType1CFont) GetPathByName(name string) (*geom.Path2D, error) {
	// Acrobat only draws .notdef for embedded or "Standard 14" fonts, see
	// PDFBOX-2372
	if name == ".notdef" && !f.IsEmbedded() && !f.IsStandard14() {
		return geom.NewPathFloat(), nil
	}
	if f.genericFont == nil {
		return nil, errNoType1CProgram
	}
	if name == "sfthyphen" {
		return f.genericFont.GetPath("hyphen")
	}
	if name == "nbspace" {
		hasSpace, err := f.HasGlyphByName("space")
		if err != nil {
			return nil, err
		}
		if !hasSpace {
			return geom.NewPathFloat(), nil
		}
		return f.genericFont.GetPath("space")
	}
	return f.genericFont.GetPath(name)
}

// errNoType1CProgram is what every path that needs the font program returns for
// a font that is not embedded, while the font mapper is unported.
var errNoType1CProgram = fmt.Errorf(
	"font: no font program: the font is not embedded and the font mapper is not ported yet")

// HasGlyphForCode reports whether the font has an outline for the glyph.
func (f *PDType1CFont) HasGlyphForCode(code int) (bool, error) {
	name := f.Encoding().Name(code)
	name, err := f.getNameInFont(name)
	if err != nil {
		return false, err
	}
	if name == "sfthyphen" {
		return f.HasGlyphByName("hyphen")
	}
	if name == "nbspace" {
		return f.HasGlyphByName("space")
	}
	return f.HasGlyphByName(name)
}

// GetPath returns the outline of the given glyph, in glyph space.
func (f *PDType1CFont) GetPath(code int) (*geom.Path2D, error) {
	name := f.Encoding().Name(code)
	name, err := f.getNameInFont(name)
	if err != nil {
		return nil, err
	}
	if name == "sfthyphen" {
		return f.GetPathByName("hyphen")
	}
	if name == "nbspace" {
		hasSpace, err := f.HasGlyphByName("space")
		if err != nil {
			return nil, err
		}
		if !hasSpace {
			return geom.NewPathFloat(), nil
		}
		return f.GetPathByName("space")
	}
	return f.GetPathByName(name)
}

// GetNormalizedPath returns the outline of the given glyph, scaled so that the
// font matrix is the default one.
func (f *PDType1CFont) GetNormalizedPath(code int) (*geom.Path2D, error) {
	name := f.Encoding().Name(code)
	name, err := f.getNameInFont(name)
	if err != nil {
		return nil, err
	}
	switch name {
	case "nbspace":
		hasSpace, err := f.HasGlyphByName("space")
		if err != nil {
			return nil, err
		}
		if !hasSpace {
			return geom.NewPathFloat(), nil
		}
		name = "space"
	case "sfthyphen":
		name = "hyphen"
	}
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
func (f *PDType1CFont) HasGlyphByName(name string) (bool, error) {
	if f.genericFont == nil {
		return false, errNoType1CProgram
	}
	return f.genericFont.HasGlyph(name)
}

// BoundingBox returns the box every glyph of the font fits in.
func (f *PDType1CFont) BoundingBox() (*fontutil.BoundingBox, error) {
	if f.fontBBox == nil {
		bbox, err := f.generateBoundingBox()
		if err != nil {
			return nil, err
		}
		f.fontBBox = bbox
	}
	return f.fontBBox, nil
}

func (f *PDType1CFont) generateBoundingBox() (*fontutil.BoundingBox, error) {
	if fd := f.FontDescriptor(); fd != nil {
		bbox := fd.FontBoundingBox()
		if isNonZeroBoundingBox(bbox) {
			return fontutil.NewBoundingBoxOf(bbox.LowerLeftX(), bbox.LowerLeftY(),
				bbox.UpperRightX(), bbox.UpperRightY()), nil
		}
	}
	if f.genericFont == nil {
		return nil, errNoType1CProgram
	}
	return f.genericFont.FontBBox()
}

// CodeToName returns the glyph name the given code stands for.
func (f *PDType1CFont) CodeToName(code int) (string, error) {
	return f.Encoding().Name(code), nil
}

// readEncodingFromFont works out the encoding from the font program.
func (f *PDType1CFont) readEncodingFromFont() (encoding.Encoding, error) {
	if !f.IsEmbedded() && f.standard14AFM() != nil {
		// read from AFM
		return encoding.NewType1EncodingFromMetrics(f.standard14AFM()), nil
	}
	// extract from Type1 font/substitute
	if encoded, ok := f.genericFont.(fontbox.EncodedFont); ok {
		return encoding.Type1EncodingFromFontBox(encoded.Encoding()), nil
	}
	// default (only happens with TTFs)
	return encoding.StandardEncodingInstance, nil
}

// ReadCode reads one character code from the stream.
func (f *PDType1CFont) ReadCode(in *bytes.Reader) (int, error) {
	b, err := in.ReadByte()
	if err != nil {
		return -1, err
	}
	return int(b), nil
}

// FontMatrix returns the transform from glyph space to text space.
func (f *PDType1CFont) FontMatrix() *util.Matrix {
	if f.fontMatrix != nil {
		return f.fontMatrix
	}
	if f.genericFont == nil {
		f.fontMatrix = defaultFontMatrix
		return f.fontMatrix
	}
	numbers, err := f.genericFont.FontMatrix()
	if err != nil {
		slog.Debug("Couldn't get font matrix - returning default value", "err", err)
		f.fontMatrix = defaultFontMatrix
		return f.fontMatrix
	}
	if len(numbers) != 6 {
		return f.pdSimpleFont.FontMatrix()
	}
	f.fontMatrix = util.NewMatrixOf(numbers[0], numbers[1], numbers[2], numbers[3],
		numbers[4], numbers[5])
	return f.fontMatrix
}

// IsDamaged reports whether the font program could not be read.
func (f *PDType1CFont) IsDamaged() bool { return f.isDamaged }

// WidthFromFont returns the width the font program gives for the glyph.
func (f *PDType1CFont) WidthFromFont(code int) (float32, error) {
	name, err := f.CodeToName(code)
	if err != nil {
		return 0, err
	}
	if name, err = f.getNameInFont(name); err != nil {
		return 0, err
	}
	if f.genericFont == nil {
		return 0, errNoType1CProgram
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
func (f *PDType1CFont) IsEmbedded() bool { return f.isEmbedded }

// Height returns how tall the given glyph is.
func (f *PDType1CFont) Height(code int) (float32, error) {
	name, err := f.CodeToName(code)
	if err != nil {
		return 0, err
	}
	if height, ok := f.glyphHeights[name]; ok {
		return height, nil
	}
	if f.cffFont == nil {
		slog.Warn("No embedded CFF font, returning 0")
		return 0, nil
	}
	charString, err := f.cffFont.Type1CharString(name)
	if err != nil {
		return 0, err
	}
	height := float32(charString.Bounds().Height)
	f.glyphHeights[name] = height
	return height, nil
}

// encodeCodePoint returns the bytes that draw the given code point.
func (f *PDType1CFont) encodeCodePoint(unicode int) ([]byte, error) {
	name := f.GlyphList().CodePointToName(unicode)
	if !f.encoding.ContainsName(name) {
		return nil, fmt.Errorf("font: U+%04X ('%s') is not available in font %s encoding: %s",
			unicode, name, f.Name(), f.encoding.EncodingName())
	}

	nameInFont, err := f.getNameInFont(name)
	if err != nil {
		return nil, err
	}

	inverted := f.encoding.NameToCodeMap()

	hasGlyph := false
	if f.genericFont != nil {
		if hasGlyph, err = f.genericFont.HasGlyph(nameInFont); err != nil {
			return nil, err
		}
	}
	if nameInFont == ".notdef" || !hasGlyph {
		return nil, fmt.Errorf("font: No glyph for U+%04X in font %s", unicode, f.Name())
	}

	code := inverted[name]
	return []byte{byte(code)}, nil
}

// StringWidth returns how wide the given text is in this font.
func (f *PDType1CFont) StringWidth(text string) (float32, error) {
	if f.cffFont == nil {
		slog.Warn("No embedded CFF font, returning 0")
		return 0, nil
	}
	var width float32
	// Java walks the String by index and reads codePointAt at every one, so a
	// character outside the basic plane is measured twice: once whole and once
	// as its trailing surrogate. See migration/JAVA-BUGS.md entry 18.
	units := utf16Units(text)
	for i := 0; i < len(units); i++ {
		codePoint := codePointAt(units, i)
		name := f.GlyphList().CodePointToName(codePoint)
		hasGlyph, err := f.cffFont.HasGlyph(name)
		if err != nil {
			return 0, err
		}
		if !hasGlyph {
			return 0, fmt.Errorf("font: U+%04X ('%s') is not available in font %s",
				codePoint, name, f.Name())
		}
		charString, err := f.cffFont.Type1CharString(name)
		if err != nil {
			return 0, err
		}
		width += float32(charString.Width())
	}
	return width, nil
}

// AverageFontWidth returns the average width of the glyphs.
func (f *PDType1CFont) AverageFontWidth() float32 {
	if !f.avgWidthSet {
		f.avgWidth = f.averageCharacterWidth()
		f.avgWidthSet = true
	}
	return f.avgWidth
}

// CFFType1Font returns the embedded Type 1-equivalent CFF font.
func (f *PDType1CFont) CFFType1Font() *cff.CFFType1Font { return f.cffFont }

// averageCharacterWidth is a replacement for a FontMetrics method.
//
// todo: not implemented, highly suspect
func (f *PDType1CFont) averageCharacterWidth() float32 { return 500 }

// getNameInFont maps a PostScript glyph name to the name in the underlying
// font, for example when using a TTF font we might map "W" to "uni0057".
func (f *PDType1CFont) getNameInFont(name string) (string, error) {
	if f.IsEmbedded() {
		return name, nil
	}
	if f.genericFont != nil {
		hasGlyph, err := f.genericFont.HasGlyph(name)
		if err != nil {
			return "", err
		}
		if hasGlyph {
			return name, nil
		}
		// try unicode name
		unicodes := f.GlyphList().ToUnicode(name)
		if runes := []rune(unicodes); len(runes) == 1 {
			uniName := getUniNameOfCodePoint(int(runes[0]))
			hasUni, err := f.genericFont.HasGlyph(uniName)
			if err != nil {
				return "", err
			}
			if hasUni {
				return uniName, nil
			}
		}
	}
	return ".notdef", nil
}

// utf16Units returns the UTF-16 code units of a string, which is what Java's
// String.length and String.charAt walk.
func utf16Units(s string) []uint16 {
	return utf16.Encode([]rune(s))
}

// codePointAt is Java's String.codePointAt over those units: the whole
// character where a well-formed surrogate pair begins at the index, and the
// bare unit otherwise.
func codePointAt(units []uint16, index int) int {
	unit := units[index]
	if unit >= 0xD800 && unit <= 0xDBFF && index+1 < len(units) {
		if next := units[index+1]; next >= 0xDC00 && next <= 0xDFFF {
			return int(utf16.DecodeRune(rune(unit), rune(next)))
		}
	}
	return int(unit)
}
