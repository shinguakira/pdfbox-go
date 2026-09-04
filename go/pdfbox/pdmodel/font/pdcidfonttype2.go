package font

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf"
	fontutil "github.com/shinguakira/pdfbox-go/go/fontbox/util"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// PDCIDFontType2 is a Type 2 CIDFont (TrueType).
//
// Port of org.apache.pdfbox.pdmodel.font.PDCIDFontType2.
type PDCIDFontType2 struct {
	pdCIDFont

	ttf        *ttf.TrueTypeFont
	otf        *ttf.OpenTypeFont
	cid2gid    []int
	cmap       ttf.CmapLookup // may be nil
	fontMatrix *util.Matrix
	fontBBox   *fontutil.BoundingBox
	noMapping  map[int]bool
}

var _ PDCIDFont = (*PDCIDFontType2)(nil)

// NewPDCIDFontType2 returns the Type 2 CIDFont the given dictionary describes.
func NewPDCIDFontType2(fontDictionary *cos.Dictionary,
	resourceCache ResourceCache) (*PDCIDFontType2, error) {
	return NewPDCIDFontType2WithFont(fontDictionary, nil, resourceCache)
}

// NewPDCIDFontType2WithFont returns the Type 2 CIDFont the given dictionary
// describes, trueTypeFont being the font used to create the parent font.
func NewPDCIDFontType2WithFont(fontDictionary *cos.Dictionary,
	trueTypeFont *ttf.TrueTypeFont, resourceCache ResourceCache) (*PDCIDFontType2, error) {
	f := &PDCIDFontType2{
		pdCIDFont: newPDCIDFont(fontDictionary),
		noMapping: map[int]bool{},
	}
	f.pdCIDFont.self = f
	f.initCIDFont(resourceCache)

	if trueTypeFont != nil {
		f.ttf = trueTypeFont
		f.otf = asSupportedOTF(trueTypeFont)
		f.isEmbedded = true
		f.isDamaged = false
	} else {
		fontIsDamaged := false
		var ttfFont *ttf.TrueTypeFont
		var stream *common.PDStream
		fd := f.FontDescriptor()
		if fd != nil {
			stream = fd.FontFile2()
			if stream == nil {
				stream = fd.FontFile3()
			}
			if stream == nil {
				// Acrobat looks in FontFile too, even though it is not in the
				// spec, see PDFBOX-2599
				stream = fd.FontFile()
			}
		}
		if stream != nil {
			parsed, err := readEmbeddedTrueType(stream.Stream())
			if err != nil {
				fontIsDamaged = true
				slog.Warn("Could not read embedded OTF for font", "font", f.BaseFont(), "err", err)
			} else {
				ttfFont = parsed
			}
			if ttfFont != nil && isUnsupportedOTF(ttfFont) {
				// the OpenType font contains CFF2 outlines which are not
				// supported yet
				ttfFont = nil
				fontIsDamaged = true
				slog.Warn("Found an OpenType font using CFF2 outlines which are not supported",
					"font", fd.FontName())
			}
		}
		f.isEmbedded = ttfFont != nil
		f.isDamaged = fontIsDamaged

		if ttfFont == nil {
			var err error
			if ttfFont, err = f.findFontOrSubstitute(); err != nil {
				return nil, err
			}
		}
		f.otf = asSupportedOTF(ttfFont)
		f.ttf = ttfFont
	}
	lookup, err := f.ttf.UnicodeCmapLookup(false)
	if err != nil {
		return nil, err
	}
	f.cmap = lookup
	if f.cid2gid, err = f.readCIDToGIDMap(); err != nil {
		return nil, err
	}
	return f, nil
}

// findFontOrSubstitute returns the system font of this name, or the substitute
// the mapper picked for it.
func (f *PDCIDFontType2) findFontOrSubstitute() (*ttf.TrueTypeFont, error) {
	var ttfFont *ttf.TrueTypeFont

	mapping := FontMappersInstance().GetCIDFont(f.BaseFont(), f.FontDescriptor(),
		f.CIDSystemInfo())
	if mapping.IsCIDFont() {
		ttfFont = mapping.Font().TrueTypeFont
	} else {
		// Java casts the FontBoxFont to TrueTypeFont, which throws
		// ClassCastException where the mapper handed back another kind.
		if trueType := mapping.TrueTypeFont(); trueType != nil {
			ttfFont = trueType.(*ttf.TrueTypeFont)
		}
		if ttfFont == nil {
			// shouldn't happen?!
			return nil, fmt.Errorf(
				"font: mapping.TrueTypeFont() returns nil, please report")
		}
	}
	if mapping.IsFallback() {
		name, err := ttfFont.Name()
		if err != nil {
			return nil, err
		}
		slog.Warn("Using fallback font for CID-keyed TrueType font",
			"fallback", name, "font", f.BaseFont())
	}
	return ttfFont, nil
}

// asSupportedOTF gives the OpenType view of a font that has one and is
// supported, standing for Java's
// `font instanceof OpenTypeFont && ((OpenTypeFont) font).isSupportedOTF()`.
func asSupportedOTF(font *ttf.TrueTypeFont) *ttf.OpenTypeFont {
	otf := font.AsOpenType()
	if otf != nil && otf.IsSupportedOTF() {
		return otf
	}
	return nil
}

// isUnsupportedOTF stands for
// `font instanceof OpenTypeFont && !((OpenTypeFont) font).isSupportedOTF()`.
func isUnsupportedOTF(font *ttf.TrueTypeFont) bool {
	otf := font.AsOpenType()
	return otf != nil && !otf.IsSupportedOTF()
}

// readEmbeddedTrueType parses an embedded OTF or TTF.
func readEmbeddedTrueType(stream *cos.Stream) (*ttf.TrueTypeFont, error) {
	view, err := stream.CreateView()
	if err != nil {
		return nil, err
	}
	isOTF, err := looksLikeOTF(view)
	if err != nil {
		pdfio.CloseQuietly(view)
		return nil, err
	}
	if isOTF {
		otf, err := ttf.NewOTFParserEmbedded(true).Parse(view)
		if err != nil {
			return nil, err
		}
		if err := otf.Close(); err != nil {
			return nil, err
		}
		return otf.TrueTypeFont, nil
	}
	font, err := ttf.NewParserEmbedded(true).Parse(view)
	if err != nil {
		return nil, err
	}
	if err := font.Close(); err != nil {
		return nil, err
	}
	return font, nil
}

// looksLikeOTF reads the four-byte tag at the cursor and puts the cursor back,
// which is Java's getParser.
func looksLikeOTF(randomAccessRead pdfio.RandomAccessRead) (bool, error) {
	startPos, err := randomAccessRead.Position()
	if err != nil {
		return false, err
	}
	tagBytes := make([]byte, 4)
	remainingBytes := len(tagBytes)
	for remainingBytes > 0 {
		amountRead, err := randomAccessRead.Read(tagBytes[len(tagBytes)-remainingBytes:])
		if amountRead <= 0 {
			if err != nil && !errors.Is(err, io.EOF) {
				break
			}
			break
		}
		remainingBytes -= amountRead
	}
	if _, err := randomAccessRead.Seek(startPos, 0); err != nil {
		return false, err
	}
	return string(tagBytes) == "OTTO", nil
}

// FontMatrix returns the transformation from glyph space to text space.
func (f *PDCIDFontType2) FontMatrix() *util.Matrix {
	if f.fontMatrix == nil {
		// 1000 upem, this is not strictly true
		f.fontMatrix = util.NewMatrixOf(0.001, 0, 0, 0.001, 0, 0)
	}
	return f.fontMatrix
}

// BoundingBox returns the font's bounding box.
func (f *PDCIDFontType2) BoundingBox() (*fontutil.BoundingBox, error) {
	if f.fontBBox == nil {
		bbox, err := f.generateBoundingBox()
		if err != nil {
			return nil, err
		}
		f.fontBBox = bbox
	}
	return f.fontBBox, nil
}

func (f *PDCIDFontType2) generateBoundingBox() (*fontutil.BoundingBox, error) {
	if fd := f.FontDescriptor(); fd != nil {
		bbox := fd.FontBoundingBox()
		if isNonZeroBoundingBox(bbox) {
			return fontutil.NewBoundingBoxOf(bbox.LowerLeftX(), bbox.LowerLeftY(),
				bbox.UpperRightX(), bbox.UpperRightY()), nil
		}
	}
	return f.ttf.FontBBox()
}

// CodeToCID returns the CID for the given character code.
func (f *PDCIDFontType2) CodeToCID(code int, parent *PDType0Font) int {
	cMap := parent.CMap()

	// Acrobat allows bad PDFs to use Unicode CMaps here instead of CID CMaps,
	// see PDFBOX-1283
	if !cMap.HasCIDMappings() && cMap.HasUnicodeMappings() {
		if unicode, ok := cMap.ToUnicode(code); ok {
			// actually: code -> CID
			return codePointAt(utf16Units(unicode), 0)
		}
	}

	return cMap.ToCID(code)
}

// CodeToGID returns the GID for the given character code.
func (f *PDCIDFontType2) CodeToGID(code int, parent *PDType0Font) (int, error) {
	if !f.isEmbedded {
		// The conforming reader shall select glyphs by translating characters
		// from the encoding specified by the predefined CMap to one of the
		// encodings in the TrueType font's 'cmap' table. The means by which
		// this is accomplished are implementation-dependent.
		// omit the CID2GID mapping if the embedded font is replaced by an
		// external font
		name := f.BaseFont()
		ttfName := ""
		if f.ttf != nil {
			ttfName, _ = f.ttf.Name()
		}
		if f.cid2gid != nil && !f.isDamaged && name != "" && name == ttfName {
			// Acrobat allows non-embedded GIDs - todo: can we find a test PDF
			// for this?
			// PDFBOX-5612: should happen only if it's really the same font
			// this is not perfect, we may have to improve this because some
			// identical fonts have different names
			slog.Warn("Using non-embedded GIDs in font", "font", f.BaseFont())
			cid := f.CodeToCID(code, parent)
			if cid < len(f.cid2gid) {
				return f.cid2gid[cid], nil
			}
			return 0, nil
		}
		// fallback to the ToUnicode CMap, test with PDFBOX-1422 and PDFBOX-2560
		unicode, err := parent.ToUnicode(code)
		if err != nil {
			return 0, err
		}
		if unicode == "" {
			if !f.noMapping[code] {
				// we keep track of which warnings have been issued, so we don't
				// log multiple times
				f.noMapping[code] = true
				slog.Warn("Failed to find a character mapping", "code", code, "font", f.BaseFont())
			}
			// Acrobat is willing to use the CID as a GID, even when the font
			// isn't embedded, see PDFBOX-2599
			return f.CodeToCID(code, parent), nil
		}
		if len(utf16Units(unicode)) > 1 {
			slog.Warn("Trying to map multi-byte character using 'cmap', result will be poor")
		}

		// a non-embedded font always has a cmap (otherwise FontMapper won't
		// load it)
		return f.cmap.GetGlyphID(codePointAt(utf16Units(unicode), 0)), nil
	}

	// If the TrueType font program is embedded, the Type 2 CIDFont dictionary
	// shall contain a CIDToGIDMap entry that maps CIDs to the glyph indices for
	// the appropriate glyph descriptions in that font program.
	cid := f.CodeToCID(code, parent)
	if f.cid2gid != nil {
		// use CIDToGIDMap
		if cid < len(f.cid2gid) {
			return f.cid2gid[cid], nil
		}
		return 0, nil
	}
	// "Identity" is the default for CFF-based OpenTypeFonts
	if f.otf != nil && f.otf.IsPostScript() {
		return cid, nil
	}
	// "Identity" is the default for TrueTypeFonts if the CID is within the range
	numberOfGlyphs, err := f.ttf.NumberOfGlyphs()
	if err != nil {
		return 0, err
	}
	if cid < numberOfGlyphs {
		return cid, nil
	}
	return 0, nil
}

// Height returns how tall the given glyph is.
func (f *PDCIDFontType2) Height(code int, parent *PDType0Font) (float32, error) {
	// todo: really we want the BBox, (for text extraction:)
	hh, err := f.ttf.HorizontalHeader()
	if err != nil {
		return 0, err
	}
	unitsPerEm, err := f.ttf.UnitsPerEm()
	if err != nil {
		return 0, err
	}
	// todo: shouldn't this be the yMax/yMin?
	return float32(hh.Ascender()+-hh.Descender()) / float32(unitsPerEm), nil
}

// WidthFromFont returns the width the font program gives for the glyph.
func (f *PDCIDFontType2) WidthFromFont(code int, parent *PDType0Font) (float32, error) {
	gid, err := f.CodeToGID(code, parent)
	if err != nil {
		return 0, err
	}
	advance, err := f.ttf.AdvanceWidth(gid)
	if err != nil {
		return 0, err
	}
	width := float32(advance)
	unitsPerEM, err := f.ttf.UnitsPerEm()
	if err != nil {
		return 0, err
	}
	if unitsPerEM != 1000 {
		width *= 1000 / float32(unitsPerEM)
	}
	return width, nil
}

// Encode returns the bytes that draw the given code point.
func (f *PDCIDFontType2) Encode(unicode int, parent *PDType0Font) ([]byte, error) {
	cid := -1
	if f.isEmbedded {
		// embedded fonts always use CIDToGIDMap, with Identity as the default
		if strings.HasPrefix(parent.CMap().Name(), "Identity-") {
			if f.cmap != nil {
				cid = f.cmap.GetGlyphID(unicode)
			}
		} else if parent.CMapUCS2() != nil {
			// if the CMap is predefined then there will be a UCS-2 CMap
			cid = parent.CMapUCS2().ToCID(unicode)
		}

		// otherwise we require an explicit ToUnicode CMap
		if cid == -1 {
			if toUnicodeCMap := parent.ToUnicodeCMap(); toUnicodeCMap != nil {
				codes, ok := toUnicodeCMap.GetCodesFromUnicode(string(rune(uint16(unicode))))
				if ok {
					return codes, nil
				}
			}
			cid = 0
		}
	} else {
		// a non-embedded font always has a cmap (otherwise we wouldn't load it)
		cid = f.cmap.GetGlyphID(unicode)
	}

	if cid == 0 {
		return nil, fmt.Errorf("No glyph for U+%04X (%c) in font %s",
			unicode, rune(unicode), f.BaseFont())
	}

	return f.EncodeGlyphID(cid), nil
}

// EncodeGlyphID returns the encoded value for the given glyph ID.
func (f *PDCIDFontType2) EncodeGlyphID(glyphID int) []byte {
	// CID is always 2-bytes (16-bit) for TrueType
	return []byte{byte(glyphID >> 8 & 0xff), byte(glyphID & 0xff)}
}

// TrueTypeFont returns the embedded or substituted TrueType font. It may be an
// OpenType font if the font is not embedded.
func (f *PDCIDFontType2) TrueTypeFont() *ttf.TrueTypeFont { return f.ttf }

// GetPath returns the glyph path for the given character code.
func (f *PDCIDFontType2) GetPath(code int, parent *PDType0Font) (*geom.Path2D, error) {
	if f.otf != nil && f.otf.IsPostScript() {
		path, err := f.getPathFromOutlines(code, parent)
		if err != nil {
			return nil, err
		}
		if path == nil {
			return geom.NewPathFloat(), nil
		}
		return path, nil
	}
	gid, err := f.CodeToGID(code, parent)
	if err != nil {
		return nil, err
	}
	glyphTable, err := f.ttf.Glyph()
	if err != nil {
		return nil, err
	}
	glyph, err := glyphTable.GetGlyph(gid)
	if err != nil {
		return nil, err
	}
	if glyph != nil {
		return glyph.Path(), nil
	}
	return geom.NewPathFloat(), nil
}

// GetNormalizedPath returns the glyph path scaled to the 1000 unit square.
func (f *PDCIDFontType2) GetNormalizedPath(code int, parent *PDType0Font) (*geom.Path2D, error) {
	var path *geom.Path2D
	var err error
	if f.otf != nil && f.otf.IsPostScript() {
		if path, err = f.getPathFromOutlines(code, parent); err != nil {
			return nil, err
		}
	} else {
		gid, err := f.CodeToGID(code, parent)
		if err != nil {
			return nil, err
		}
		if path, err = f.GetPath(code, parent); err != nil {
			return nil, err
		}
		// Acrobat only draws GID 0 for embedded CIDFonts, see PDFBOX-2372
		if gid == 0 && !f.IsEmbedded() {
			path = nil
		}
	}
	if path == nil {
		// empty glyph (e.g. space, newline)
		return geom.NewPathFloat(), nil
	}

	unitsPerEm, err := f.ttf.UnitsPerEm()
	if err != nil {
		return nil, err
	}
	if unitsPerEm != 1000 {
		scale := 1000 / float64(unitsPerEm)

		// PDFBOX-5567: clone() to avoid repeated modification on cached path
		path = path.Clone()

		path.Transform(geom.ScaleInstance(scale, scale))
	}
	return path, nil
}

func (f *PDCIDFontType2) getPathFromOutlines(code int, parent *PDType0Font) (*geom.Path2D, error) {
	cffTable, err := f.otf.CFF()
	if err != nil {
		return nil, err
	}
	gid, err := f.CodeToGID(code, parent)
	if err != nil {
		return nil, err
	}
	charString, err := cffTable.Font().GetType2CharString(gid)
	if err != nil {
		return nil, err
	}
	if charString == nil {
		return nil, nil
	}
	return charString.Path(), nil
}

// HasGlyph reports whether this font contains a glyph for the given code.
func (f *PDCIDFontType2) HasGlyph(code int, parent *PDType0Font) (bool, error) {
	gid, err := f.CodeToGID(code, parent)
	if err != nil {
		return false, err
	}
	return gid != 0, nil
}
