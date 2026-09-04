package font

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/fontbox"
	"github.com/shinguakira/pdfbox-go/go/fontbox/type1"
	fontutil "github.com/shinguakira/pdfbox-go/go/fontbox/util"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
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
type PDType1Font struct {
	pdSimpleFont

	genericFont         fontbox.FontBoxFont
	type1font           *type1.Type1Font
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
	f.type1font = nil
	mapping := FontMappersInstance().GetFontBoxFont(f.BaseFont(), f.FontDescriptor())
	f.genericFont = mapping.Font()

	if mapping.IsFallback() {
		// Java catches the IOException getName may throw and logs "?" instead.
		fontName, err := f.genericFont.Name()
		if err != nil {
			slog.Debug("Couldn't get font name - setting to '?'", "err", err)
			fontName = "?"
		}
		slog.Warn("Using fallback font", "fallback", fontName, "baseFont", f.BaseFont())
	}
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
	var t1 *type1.Type1Font
	fontIsDamaged := false
	if fd != nil {
		// a Type1 font may contain a Type1C font
		if fontFile3 := fd.FontFile3(); fontFile3 != nil {
			slog.Warn("/FontFile3 for Type1 font not supported")
		}

		// or it may contain a PFB
		if fontFile := fd.FontFile(); fontFile != nil {
			var err error
			t1, err = f.readType1FontProgram(fontFile)
			if err != nil {
				var damaged *type1.DamagedFontException
				if errors.As(err, &damaged) {
					slog.Warn("Can't read damaged embedded Type1 font",
						"font", fd.FontName(), "err", err)
				} else {
					slog.Error("Can't read the embedded Type1 font",
						"font", fd.FontName(), "err", err)
				}
				fontIsDamaged = true
			}
		}
	}
	f.isEmbedded = t1 != nil
	f.isDamaged = fontIsDamaged
	f.type1font = t1

	// find a generic font to use for rendering, could be a .pfb, but might be
	// a .ttf
	if t1 != nil {
		f.genericFont = t1
	} else {
		mapping := FontMappersInstance().GetFontBoxFont(f.BaseFont(), fd)
		f.genericFont = mapping.Font()

		if mapping.IsFallback() {
			fontName, err := f.genericFont.Name()
			if err != nil {
				return nil, err
			}
			slog.Warn("Using fallback font", "fallback", fontName, "font", f.BaseFont())
		}
	}

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
	name, err := f.CodeToName(code)
	if err != nil {
		return 0, err
	}
	// todo: should be scaled by font matrix
	path, err := f.genericFont.GetPath(name)
	if err != nil {
		return 0, err
	}
	return float32(path.Bounds().Height), nil
}

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
			genericName, err := f.genericFont.Name()
			if err != nil {
				return nil, err
			}
			return nil, fmt.Errorf(
				"font: U+%04X ('%s') is not available in the font %s (generic: %s), encoding: %s",
				unicode, name, f.Name(), genericName, f.encoding.EncodingName())
		}

		nameInFont, err := f.getNameInFont(name)
		if err != nil {
			return nil, err
		}

		hasGlyph := false
		if nameInFont != ".notdef" {
			// Java's || short-circuits, so a .notdef never reaches hasGlyph.
			if hasGlyph, err = f.genericFont.HasGlyph(nameInFont); err != nil {
				return nil, err
			}
		}
		if nameInFont == ".notdef" || !hasGlyph {
			genericName, err := f.genericFont.Name()
			if err != nil {
				return nil, err
			}
			return nil, fmt.Errorf("font: No glyph for U+%04X in the font %s (generic: %s)",
				unicode, f.Name(), genericName)
		}
	}

	code, ok := f.encoding.NameToCodeMap()[name]
	if !ok || code < 0 {
		genericName, err := f.genericFont.Name()
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf(
			"font: U+%04X ('%s') is not available in the font %s (generic: %s), encoding: %s",
			unicode, name, f.Name(), genericName, f.encoding.EncodingName())
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
		if got, err := f.genericFont.FontMatrix(); err == nil {
			numbers = got
		} else {
			slog.Debug("Couldn't get font matrix box - returning default value", "err", err)
			f.fontMatrix = defaultFontMatrix
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

// pfbStartMarker is the byte a whole PFB file begins with.
const pfbStartMarker = 0x80

// readType1FontProgram reads the embedded PFB out of the /FontFile stream.
//
// Port of the /FontFile branch of the Java constructor, which is long enough to
// stand on its own here.
func (f *PDType1Font) readType1FontProgram(fontFile *common.PDStream) (*type1.Type1Font, error) {
	stream := fontFile.Stream()
	length1 := stream.GetInt(cos.Length1)
	length2 := stream.GetInt(cos.Length2)

	// repair Length1 and Length2 if necessary
	bytes, err := fontFile.ToByteArray()
	if err != nil {
		return nil, err
	}
	if len(bytes) == 0 {
		return nil, errors.New("Font data unavailable")
	}
	length1 = f.repairLength1(bytes, length1)
	length2 = f.repairLength2(bytes, length1, length2)

	if int(bytes[0])&0xff == pfbStartMarker {
		// some bad files embed the entire PFB, see PDFBOX-2607
		return type1.CreateWithPFBBytes(bytes)
	}
	// the PFB embedded as two segments back-to-back
	if length1 < 0 || length1 > length1+length2 {
		return nil, fmt.Errorf(
			"Invalid length data, actual length: %d, /Length1: %d, /Length2: %d",
			len(bytes), length1, length2)
	}
	if length1 > len(bytes) || length1+length2 > len(bytes) {
		// Java's Arrays.copyOfRange pads past the end rather than failing; the
		// repair above keeps both lengths inside the data, so this is only the
		// belt to Go's slice bounds.
		return nil, fmt.Errorf(
			"Invalid length data, actual length: %d, /Length1: %d, /Length2: %d",
			len(bytes), length1, length2)
	}
	segment1 := bytes[0:length1]
	segment2 := bytes[length1 : length1+length2]

	// empty streams are simply ignored
	if length1 > 0 && length2 > 0 {
		return type1.CreateWithSegments(segment1, segment2)
	}
	return nil, nil
}

// repairLength1 repairs an invalid Length1, which causes the binary segment of
// the font to be truncated, see PDFBOX-2350, PDFBOX-3677.
func (f *PDType1Font) repairLength1(bytes []byte, length1 int) int {
	// scan backwards from the end of the first segment to find 'exec'
	offset := max(0, length1-4)
	if offset <= 0 || offset > len(bytes)-4 {
		offset = len(bytes) - 4
	}

	offset = findBinaryOffsetAfterExec(bytes, offset)
	if offset == 0 && length1 > 0 {
		// 2nd try with brute force
		offset = findBinaryOffsetAfterExec(bytes, len(bytes)-4)
	}

	if length1-offset != 0 && offset > 0 {
		slog.Warn("Ignored invalid Length1 for Type 1 font",
			"Length1", length1, "font", f.Name())
		return offset
	}

	return length1
}

func findBinaryOffsetAfterExec(bytes []byte, startOffset int) int {
	offset := startOffset
	for offset > 0 {
		if bytes[offset+0] == 'e' && bytes[offset+1] == 'x' &&
			bytes[offset+2] == 'e' && bytes[offset+3] == 'c' {
			offset += 4
			// skip additional CR LF space characters
			for offset < len(bytes) &&
				(bytes[offset] == '\r' || bytes[offset] == '\n' ||
					bytes[offset] == ' ' || bytes[offset] == '\t') {
				offset++
			}
			break
		}
		offset--
	}
	return offset
}

// repairLength2 repairs an invalid Length2, see PDFBOX-3475. A negative
// /Length2 brings an IllegalArgumentException in Arrays.copyOfRange(), a huge
// value eats up memory because of padding.
func (f *PDType1Font) repairLength2(bytes []byte, length1, length2 int) int {
	// repair Length2 if necessary
	if length2 < 0 || length2 > len(bytes)-length1 {
		slog.Warn("Ignored invalid Length2 for Type 1 font",
			"Length2", length2, "font", f.Name())
		return len(bytes) - length1
	}
	return length2
}
