package font

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/fontbox"
	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf"
	fontutil "github.com/shinguakira/pdfbox-go/go/fontbox/util"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font/encoding"
)

// The code ranges a symbol cmap may put its glyphs in.
const (
	startRangeF000 = 0xF000
	startRangeF100 = 0xF100
	startRangeF200 = 0xF200
)

// invertedMacOSRoman maps a glyph name onto its Mac OS Roman code, keeping the
// first code each name is seen at.
var invertedMacOSRoman = func() map[string]int {
	inverted := make(map[string]int, 250)
	codeToName := MacOSRomanCodeToName()
	// Java walks a HashMap and keeps the first value it meets for each name;
	// the port walks the codes in order, so that the map is always the same.
	codes := make([]int, 0, len(codeToName))
	for code := range codeToName {
		codes = append(codes, code)
	}
	sort.Ints(codes)
	for _, code := range codes {
		if _, ok := inverted[codeToName[code]]; !ok {
			inverted[codeToName[code]] = code
		}
	}
	return inverted
}()

// MacOSRomanCodeToName returns the code to name mapping of the Mac OS Roman
// encoding.
func MacOSRomanCodeToName() map[int]string {
	return encoding.MacOSRomanEncodingInstance.CodeToNameMap()
}

// PDTrueTypeFont is a TrueType font of a PDF.
//
// Port of org.apache.pdfbox.pdmodel.font.PDTrueTypeFont.
//
// The substitute a system font provides for a font that is not embedded needs
// the font mapper chain, which slice 4 ports; a font that is not embedded
// therefore has no font program here, and the paths that read one report that
// rather than guessing. OpenType fonts with CFF outlines are slice 4 as well.
// See migration/STATUS.md.
type PDTrueTypeFont struct {
	pdSimpleFont

	ttf        *ttf.TrueTypeFont
	isEmbedded bool
	isDamaged  bool

	cmapWinUnicode  *ttf.CmapSubtable
	cmapWinSymbol   *ttf.CmapSubtable
	cmapMacRoman    *ttf.CmapSubtable
	cmapInitialized bool

	gidToCode map[int]int // for embedding
	fontBBox  *fontutil.BoundingBox
}

var (
	_ PDFont       = (*PDTrueTypeFont)(nil)
	_ PDSimpleFont = (*PDTrueTypeFont)(nil)
)

// NewPDTrueTypeFontFromDictionary returns the TrueType font the given
// dictionary describes.
func NewPDTrueTypeFontFromDictionary(fontDictionary *cos.Dictionary, resourceCache ResourceCache) (*PDTrueTypeFont, error) {
	f := &PDTrueTypeFont{
		pdSimpleFont: pdSimpleFont{pdFont: newPDFontFromDictionary(fontDictionary)},
		gidToCode:    map[int]int{},
	}
	f.pdFont.self = f
	f.selfSimple = f
	f.initFromDictionary(resourceCache)

	var ttfFont *ttf.TrueTypeFont
	fontIsDamaged := false
	if fd := f.FontDescriptor(); fd != nil {
		if ff2Stream := fd.FontFile2(); ff2Stream != nil {
			view, err := ff2Stream.Stream().CreateView()
			if err != nil {
				// Could not read embedded TTF for the font
				fontIsDamaged = true
			} else {
				// embedded
				parser, err := trueTypeParser(view, true)
				if err != nil {
					fontIsDamaged = true
					view.Close()
				} else {
					ttfFont, err = parser.Parse(view)
					if err != nil {
						fontIsDamaged = true
						ttfFont = nil
					} else {
						ttfFont.Close()
					}
				}
			}
		}
	}
	f.isEmbedded = ttfFont != nil
	f.isDamaged = fontIsDamaged

	// Java asks the font mapper for a substitute here; that is slice 4, so a
	// font that is not embedded has no font program at all.
	f.ttf = ttfFont

	if err := f.readEncoding(); err != nil {
		return nil, err
	}
	return f, nil
}

// trueTypeParser returns the parser the font at the cursor needs, which for an
// OpenType font with CFF outlines would be the OTF parser.
//
// Java sniffs the "OTTO" tag and returns an OTFParser; slice 4 ports that, so
// an OpenType font is rejected here rather than read as a TrueType one.
func trueTypeParser(randomAccessRead interface {
	Read([]byte) (int, error)
	Seek(int64, int) (int64, error)
	Position() (int64, error)
}, isEmbedded bool) (*ttf.Parser, error) {
	startPos, err := randomAccessRead.Position()
	if err != nil {
		return nil, err
	}
	tagBytes := make([]byte, 4)
	remainingBytes := len(tagBytes)
	for remainingBytes > 0 {
		amountRead, err := randomAccessRead.Read(tagBytes[len(tagBytes)-remainingBytes:])
		if amountRead <= 0 || err != nil {
			break
		}
		remainingBytes -= amountRead
	}
	if _, err := randomAccessRead.Seek(startPos, 0); err != nil {
		return nil, err
	}
	if string(tagBytes) == "OTTO" {
		return nil, fmt.Errorf("font: OpenType fonts with CFF outlines are not ported yet")
	}
	return ttf.NewParserEmbedded(isEmbedded), nil
}

// BaseFont returns the /BaseFont entry of the font dictionary.
func (f *PDTrueTypeFont) BaseFont() string {
	return f.dict.GetNameAsString(cos.BaseFont, "")
}

// Name returns the name of the font as the PDF gives it.
func (f *PDTrueTypeFont) Name() string { return f.BaseFont() }

// readEncodingFromFont returns the encoding built into the font program.
func (f *PDTrueTypeFont) readEncodingFromFont() (encoding.Encoding, error) {
	if !f.IsEmbedded() && f.standard14AFM() != nil {
		// read from AFM
		return encoding.NewType1EncodingFromMetrics(f.standard14AFM()), nil
	}

	// non-symbolic fonts don't have a built-in encoding per se, but there
	// encoding is assumed to be StandardEncoding by the PDF spec unless an
	// explicit Encoding is present which will override this anyway
	if symbolic, ok := f.symbolicFlag(); ok && !symbolic {
		return encoding.StandardEncodingInstance, nil
	}

	// normalise the standard 14 name, e.g "Symbol,Italic" -> "Symbol"
	standard14Name, _ := GetMappedFontName(f.Name())

	// likewise, if the font is standard 14 then we know it's Standard Encoding
	if f.IsStandard14() && standard14Name != SymbolFontName &&
		standard14Name != ZapfDingbatsFontName {
		return encoding.StandardEncodingInstance, nil
	}

	// synthesize an encoding, so that getEncoding() is always usable
	if f.ttf == nil {
		return nil, errNoTrueTypeProgram
	}
	post, err := f.ttf.PostScript()
	if err != nil {
		return nil, err
	}
	codeToName := map[int]string{}
	for code := 0; code <= 256; code++ {
		gid, err := f.CodeToGID(code)
		if err != nil {
			return nil, err
		}
		if gid > 0 {
			name := ""
			if post != nil {
				name = post.GetName(gid)
			}
			if name == "" {
				// GID pseudo-name
				name = strconv.Itoa(gid)
			}
			codeToName[code] = name
		}
	}
	return encoding.NewBuiltInEncoding(codeToName), nil
}

// errNoTrueTypeProgram is what every path that needs the font program returns
// while the font mapper is unported.
var errNoTrueTypeProgram = fmt.Errorf("font: no font program: the font mapper is not ported yet")

// ReadCode reads one character code from the stream.
func (f *PDTrueTypeFont) ReadCode(in *bytes.Reader) (int, error) {
	b, err := in.ReadByte()
	if err != nil {
		// Java's InputStream.read returns -1 at the end rather than throwing.
		return -1, nil
	}
	return int(b), nil
}

// BoundingBox returns the box every glyph of the font fits in.
func (f *PDTrueTypeFont) BoundingBox() (*fontutil.BoundingBox, error) {
	if f.fontBBox == nil {
		bbox, err := f.generateBoundingBox()
		if err != nil {
			return nil, err
		}
		f.fontBBox = bbox
	}
	return f.fontBBox, nil
}

func (f *PDTrueTypeFont) generateBoundingBox() (*fontutil.BoundingBox, error) {
	if fd := f.FontDescriptor(); fd != nil {
		if bbox := fd.FontBoundingBox(); bbox != nil {
			return fontutil.NewBoundingBoxOf(bbox.LowerLeftX(), bbox.LowerLeftY(),
				bbox.UpperRightX(), bbox.UpperRightY()), nil
		}
	}
	if f.ttf == nil {
		return nil, errNoTrueTypeProgram
	}
	return f.ttf.FontBBox()
}

// IsDamaged reports whether the font program could not be read.
func (f *PDTrueTypeFont) IsDamaged() bool { return f.isDamaged }

// TrueTypeFont returns the font program.
func (f *PDTrueTypeFont) TrueTypeFont() *ttf.TrueTypeFont { return f.ttf }

// WidthFromFont returns the width the font program gives for the glyph.
func (f *PDTrueTypeFont) WidthFromFont(code int) (float32, error) {
	if f.ttf == nil {
		return 0, errNoTrueTypeProgram
	}
	gid, err := f.CodeToGID(code)
	if err != nil {
		return 0, err
	}
	advanceWidth, err := f.ttf.AdvanceWidth(gid)
	if err != nil {
		return 0, err
	}
	unitsPerEM, err := f.ttf.UnitsPerEm()
	if err != nil {
		return 0, err
	}
	width := float32(advanceWidth)
	if float32(unitsPerEM) != 1000 {
		width *= 1000 / float32(unitsPerEM)
	}
	return width, nil
}

// Height returns how tall the given glyph is.
func (f *PDTrueTypeFont) Height(code int) (float32, error) {
	if f.ttf == nil {
		return 0, errNoTrueTypeProgram
	}
	gid, err := f.CodeToGID(code)
	if err != nil {
		return 0, err
	}
	glyphTable, err := f.ttf.Glyph()
	if err != nil {
		return 0, err
	}
	glyph, err := glyphTable.GetGlyph(gid)
	if err != nil {
		return 0, err
	}
	if glyph != nil {
		return glyph.BoundingBox().Height(), nil
	}
	return 0, nil
}

// encodeCodePoint returns the bytes that draw the given code point.
func (f *PDTrueTypeFont) encodeCodePoint(unicode int) ([]byte, error) {
	if f.ttf == nil {
		return nil, errNoTrueTypeProgram
	}
	if f.encoding != nil {
		name := f.GlyphList().CodePointToName(unicode)
		if !f.encoding.ContainsName(name) {
			return nil, fmt.Errorf("font: U+%04X is not available in font %s encoding: %s",
				unicode, f.Name(), f.encoding.EncodingName())
		}
		inverted := f.encoding.NameToCodeMap()
		hasGlyph, err := f.ttf.HasGlyph(name)
		if err != nil {
			return nil, err
		}
		if !hasGlyph {
			// try unicode name
			uniName := getUniNameOfCodePoint(unicode)
			hasGlyph, err := f.ttf.HasGlyph(uniName)
			if err != nil {
				return nil, err
			}
			if !hasGlyph {
				return nil, fmt.Errorf("font: No glyph for U+%04X in font %s", unicode, f.Name())
			}
		}
		code := inverted[name]
		return []byte{byte(code)}, nil
	}

	// use TTF font's built-in encoding
	name := f.GlyphList().CodePointToName(unicode)
	hasGlyph, err := f.ttf.HasGlyph(name)
	if err != nil {
		return nil, err
	}
	if !hasGlyph {
		return nil, fmt.Errorf("font: No glyph for U+%04X in font %s", unicode, f.Name())
	}
	gid, err := f.ttf.NameToGID(name)
	if err != nil {
		return nil, err
	}
	gidToCode, err := f.GIDToCode()
	if err != nil {
		return nil, err
	}
	code, ok := gidToCode[gid]
	if !ok {
		return nil, fmt.Errorf("font: U+%04X is not available in font %s encoding", unicode, f.Name())
	}
	return []byte{byte(code)}, nil
}

// GIDToCode returns which character code reaches each glyph, keeping the lowest
// code for a glyph several codes reach.
func (f *PDTrueTypeFont) GIDToCode() (map[int]int, error) {
	if len(f.gidToCode) != 0 {
		return f.gidToCode, nil
	}
	for code := 0; code <= 255; code++ {
		gid, err := f.CodeToGID(code)
		if err != nil {
			return nil, err
		}
		if _, ok := f.gidToCode[gid]; !ok {
			f.gidToCode[gid] = code
		}
	}
	return f.gidToCode, nil
}

// IsEmbedded reports whether the font program is inside the PDF.
func (f *PDTrueTypeFont) IsEmbedded() bool { return f.isEmbedded }

// GetPath returns the outline of the given glyph.
func (f *PDTrueTypeFont) GetPath(code int) (*geom.Path2D, error) {
	if f.ttf == nil {
		return nil, errNoTrueTypeProgram
	}
	gid, err := f.CodeToGID(code)
	if err != nil {
		return nil, err
	}
	glyphTable, err := f.ttf.Glyph()
	if err != nil {
		return nil, err
	}
	if glyphTable == nil {
		// needs to be caught earlier, see PDFBOX-5587 and PDFBOX-3488
		return nil, fmt.Errorf("font: glyf table is missing in font %s, please report this file", f.Name())
	}
	glyph, err := glyphTable.GetGlyph(gid)
	if err != nil {
		return nil, err
	}
	// some glyphs have no outlines (e.g. space, table, newline)
	if glyph == nil {
		return geom.NewPathFloat(), nil
	}
	// Rendering the outline needs GlyphRenderer, which a later slice ports.
	return nil, errGlyphOutlines
}

// errGlyphOutlines is what the paths that render a glyph return while the glyph
// renderers are unported.
var errGlyphOutlines = fmt.Errorf("font: glyph outlines are not ported yet")

// GetPathByName returns the outline of the named glyph.
func (f *PDTrueTypeFont) GetPathByName(name string) (*geom.Path2D, error) {
	if f.ttf == nil {
		return nil, errNoTrueTypeProgram
	}
	// handle glyph names and uniXXXX names
	gid, err := f.ttf.NameToGID(name)
	if err != nil {
		return nil, err
	}
	if gid == 0 {
		// handle GID pseudo-names
		parsed, err := strconv.Atoi(name)
		if err != nil {
			gid = 0
		} else {
			gid = parsed
			numberOfGlyphs, err := f.ttf.NumberOfGlyphs()
			if err != nil {
				return nil, err
			}
			if gid > numberOfGlyphs {
				gid = 0
			}
		}
	}
	// I'm assuming .notdef paths are not drawn, as it PDFBOX-2421
	if gid == 0 {
		return geom.NewPathFloat(), nil
	}
	glyphTable, err := f.ttf.Glyph()
	if err != nil {
		return nil, err
	}
	glyph, err := glyphTable.GetGlyph(gid)
	if err != nil {
		return nil, err
	}
	if glyph == nil {
		return geom.NewPathFloat(), nil
	}
	return nil, errGlyphOutlines
}

// GetNormalizedPath returns the outline of the given glyph, scaled so that the
// font matrix is the default one.
func (f *PDTrueTypeFont) GetNormalizedPath(code int) (*geom.Path2D, error) {
	return nil, errGlyphOutlines
}

// HasGlyphByName reports whether the font has the named glyph.
func (f *PDTrueTypeFont) HasGlyphByName(name string) (bool, error) {
	if f.ttf == nil {
		return false, errNoTrueTypeProgram
	}
	gid, err := f.ttf.NameToGID(name)
	if err != nil {
		return false, err
	}
	maximumProfile, err := f.ttf.MaximumProfile()
	if err != nil {
		return false, err
	}
	return !(gid == 0 || gid >= maximumProfile.NumGlyphs()), nil
}

// FontBoxFont returns the font program the glyphs are drawn from.
func (f *PDTrueTypeFont) FontBoxFont() fontbox.FontBoxFont {
	if f.ttf == nil {
		return nil
	}
	return f.ttf
}

// HasGlyphForCode reports whether the font has an outline for the glyph.
func (f *PDTrueTypeFont) HasGlyphForCode(code int) (bool, error) {
	gid, err := f.CodeToGID(code)
	if err != nil {
		return false, err
	}
	return gid != 0, nil
}

// CodeToGID returns which glyph the given character code reaches.
func (f *PDTrueTypeFont) CodeToGID(code int) (int, error) {
	if f.ttf == nil {
		return 0, errNoTrueTypeProgram
	}
	if err := f.extractCmapTable(); err != nil {
		return 0, err
	}
	gid := 0

	if !f.IsSymbolic() { // non-symbolic
		name := f.encoding.Name(code)
		if name == ".notdef" {
			return 0, nil
		}
		// (3, 1) - (Windows, Unicode)
		if f.cmapWinUnicode != nil {
			unicode := encoding.AdobeGlyphList().ToUnicode(name)
			if unicode != "" {
				uni := int([]rune(unicode)[0])
				gid = f.cmapWinUnicode.GetGlyphID(uni)
			}
		}
		// (1, 0) - (Macintosh, Roman)
		if gid == 0 && f.cmapMacRoman != nil {
			if macCode, ok := invertedMacOSRoman[name]; ok {
				gid = f.cmapMacRoman.GetGlyphID(macCode)
			}
		}
		// 'post' table
		if gid == 0 {
			var err error
			gid, err = f.ttf.NameToGID(name)
			if err != nil {
				return 0, err
			}
		}
		return gid, nil
	}

	// symbolic
	// PDFBOX-4755 / PDF.js #5501
	// PDFBOX-3965: fallback for font has that the symbol flag but isn't
	if f.cmapWinUnicode != nil {
		switch f.encoding.(type) {
		case *encoding.WinAnsiEncoding, *encoding.MacRomanEncoding:
			name := f.encoding.Name(code)
			if name == ".notdef" {
				return 0, nil
			}
			unicode := encoding.AdobeGlyphList().ToUnicode(name)
			if unicode != "" {
				uni := int([]rune(unicode)[0])
				gid = f.cmapWinUnicode.GetGlyphID(uni)
			}
		default:
			gid = f.cmapWinUnicode.GetGlyphID(code)
		}
	}

	// (3, 0) - (Windows, Symbol)
	if gid == 0 && f.cmapWinSymbol != nil {
		gid = f.cmapWinSymbol.GetGlyphID(code)
		if code >= 0 && code <= 0xFF {
			// the CMap may use one of the following code ranges,
			// so that we have to add the high byte to get the
			// mapped value
			if gid == 0 {
				// F000 - F0FF
				gid = f.cmapWinSymbol.GetGlyphID(code + startRangeF000)
			}
			if gid == 0 {
				// F100 - F1FF
				gid = f.cmapWinSymbol.GetGlyphID(code + startRangeF100)
			}
			if gid == 0 {
				// F200 - F2FF
				gid = f.cmapWinSymbol.GetGlyphID(code + startRangeF200)
			}
		}
	}

	// (1, 0) - (Mac, Roman)
	if gid == 0 && f.cmapMacRoman != nil {
		gid = f.cmapMacRoman.GetGlyphID(code)
	}
	return gid, nil
}

// extractCmapTable picks out the cmap subtables the code to glyph mapping goes
// through.
func (f *PDTrueTypeFont) extractCmapTable() error {
	if f.cmapInitialized {
		return nil
	}
	cmapTable, err := f.ttf.Cmap()
	if err != nil {
		return err
	}
	if cmapTable != nil {
		// get all relevant "cmap" subtables
		for _, cmap := range cmapTable.Cmaps() {
			switch {
			case cmap.PlatformID() == ttf.CmapPlatformWindows:
				switch cmap.PlatformEncodingID() {
				case ttf.EncodingWinUnicodeBMP:
					f.cmapWinUnicode = cmap
				case ttf.EncodingWinSymbol:
					f.cmapWinSymbol = cmap
				}
			case cmap.PlatformID() == ttf.CmapPlatformMacintosh &&
				cmap.PlatformEncodingID() == ttf.EncodingMacRoman:
				f.cmapMacRoman = cmap
			case cmap.PlatformID() == ttf.CmapPlatformUnicode &&
				cmap.PlatformEncodingID() == ttf.CmapEncodingUnicode10:
				// PDFBOX-4755 / PDF.js #5501
				f.cmapWinUnicode = cmap
			case cmap.PlatformID() == ttf.CmapPlatformUnicode &&
				cmap.PlatformEncodingID() == ttf.CmapEncodingUnicode20BMP:
				// PDFBOX-5484
				f.cmapWinUnicode = cmap
			}
		}
	}
	f.cmapInitialized = true
	return nil
}
