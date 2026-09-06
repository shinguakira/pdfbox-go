package font

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/fontbox/cmap"
	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf"
	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf/model"
	fontutil "github.com/shinguakira/pdfbox-go/go/fontbox/util"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font/encoding"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// PDType0Font is a composite (Type 0) font.
//
// Port of org.apache.pdfbox.pdmodel.font.PDType0Font.
//
// The embedding half of the Java class -- the load and loadVertical factories,
// addToSubset, subset and the PDCIDFontType2Embedder behind them -- writes a
// font into a document, which a later slice ports. GsubData and the cmap lookup
// belong to that half and keep the values Java gives them for a font read out
// of a PDF: GsubData.NO_DATA_FOUND and null.
type PDType0Font struct {
	pdFont

	descendantFont   PDCIDFont
	noUnicode        map[int]bool
	cMap             *cmap.CMap
	cMapUCS2         *cmap.CMap
	gsubData         model.GsubData
	cmapLookup       ttf.CmapLookup
	isCMapPredefined bool
	isDescendantCJK  bool
}

var (
	_ PDFont       = (*PDType0Font)(nil)
	_ PDVectorFont = (*PDType0Font)(nil)
	_ type0Font    = (*PDType0Font)(nil)
)

// isType0 marks this font as the one Java's `instanceof PDType0Font` finds.
func (f *PDType0Font) isType0() {}

// NewPDType0Font reads a Type0 font from a PDF file, reporting an error if the
// descendant font is missing.
func NewPDType0Font(fontDictionary *cos.Dictionary, resourceCache ResourceCache) (*PDType0Font, error) {
	f := &PDType0Font{
		pdFont:     newPDFontFromDictionary(fontDictionary),
		noUnicode:  map[int]bool{},
		gsubData:   model.NoDataFound,
		cmapLookup: nil,
	}
	f.pdFont.self = f
	f.initFromDictionary(resourceCache)

	descendantFonts := f.dict.GetCOSArray(cos.DescendantFonts)
	if descendantFonts == nil {
		return nil, errors.New("Missing descendant font array")
	}
	if descendantFonts.Size() == 0 {
		return nil, errors.New("Descendant font array is empty")
	}
	descendantFontDict, ok := descendantFonts.GetObject(0).(*cos.Dictionary)
	if !ok {
		return nil, errors.New("Missing descendant font dictionary")
	}
	if !cos.Font.Equals(descendantFontDict.GetCOSNameDefault(cos.Type, cos.Font)) {
		return nil, errors.New("Missing or wrong type in descendant font dictionary")
	}
	descendantFontBaseObject := descendantFonts.Get(0)
	var cachedCIDFont PDCIDFont
	indirect, isIndirect := descendantFontBaseObject.(*cos.Object)
	if resourceCache != nil && isIndirect {
		cachedCIDFont = resourceCache.GetCIDFont(indirect)
	}
	if cachedCIDFont == nil {
		var err error
		cachedCIDFont, err = CreateDescendantFont(descendantFontDict, resourceCache)
		if err != nil {
			return nil, err
		}
		if resourceCache != nil && isIndirect {
			resourceCache.PutCIDFont(indirect, cachedCIDFont)
		}
	}
	f.descendantFont = cachedCIDFont
	if err := f.readEncoding(); err != nil {
		return nil, err
	}
	f.fetchCMapUCS2()
	return f, nil
}

// readEncoding reads the font's Encoding entry, which should be a CMap
// name/stream.
func (f *PDType0Font) readEncoding() error {
	encodingBase := f.dict.GetDictionaryObject(cos.Encoding)
	if encodingName, ok := encodingBase.(*cos.Name); ok {
		// predefined CMap
		predefined, err := GetPredefinedCMap(encodingName.Name())
		if err != nil {
			return err
		}
		f.cMap = predefined
		f.isCMapPredefined = true
	} else if encodingBase != nil {
		cMap, err := f.readCMap(encodingBase)
		if err != nil {
			return err
		}
		if cMap == nil {
			return errors.New("Missing required CMap")
		}
		f.cMap = cMap
		if !cMap.HasCIDMappings() {
			slog.Warn("Invalid Encoding CMap in font", "font", f.Name())
		}
	}

	// check if the descendant font is CJK
	if ros := f.descendantFont.CIDSystemInfo(); ros != nil {
		ordering := ros.Ordering()
		f.isDescendantCJK = ros.Registry() == "Adobe" &&
			(ordering == "GB1" || ordering == "CNS1" || ordering == "Japan1" ||
				ordering == "Korea1")
	}
	return nil
}

// fetchCMapUCS2 fetches the corresponding UCS2 CMap if the font's CMap is
// predefined.
func (f *PDType0Font) fetchCMapUCS2() {
	// if the font is composite and uses a predefined cmap (excluding
	// Identity-H/V) or whose descendant CIDFont uses the Adobe-GB1, Adobe-CNS1,
	// Adobe-Japan1, or Adobe-Korea1 character collection:
	name := f.dict.GetCOSName(cos.Encoding)
	if !(f.isCMapPredefined && !(name == cos.IdentityH || name == cos.IdentityV) ||
		f.isDescendantCJK) {
		return
	}
	// a) Map the character code to a CID using the font's CMap
	// b) Obtain the ROS from the font's CIDSystemInfo
	// c) Construct a second CMap name by concatenating the ROS in the format
	//    "R-O-UCS2"
	// d) Obtain the CMap with the constructed name
	// e) Map the CID according to the CMap from step d), producing a Unicode
	//    value

	// todo: not sure how to interpret the PDF spec here, do we always override?
	// or only when Identity-H/V?
	strName := ""
	if f.isDescendantCJK {
		if cidSystemInfo := f.descendantFont.CIDSystemInfo(); cidSystemInfo != nil {
			strName = fmt.Sprintf("%s-%s-%d", cidSystemInfo.Registry(),
				cidSystemInfo.Ordering(), cidSystemInfo.Supplement())
		}
	} else if name != nil {
		strName = name.Name()
	}

	// try to find the corresponding Unicode (UC2) CMap
	if strName == "" {
		return
	}
	prdCMap, err := GetPredefinedCMap(strName)
	if err != nil {
		slog.Warn("Could not get UC2 map", "name", strName, "font", f.Name(), "err", err)
		return
	}
	ucs2Name := prdCMap.Registry() + "-" + prdCMap.Ordering() + "-UCS2"
	cMapUCS2, err := GetPredefinedCMap(ucs2Name)
	if err != nil {
		slog.Warn("Could not get UC2 map", "name", strName, "font", f.Name(), "err", err)
		return
	}
	f.cMapUCS2 = cMapUCS2
}

// BaseFont returns the PostScript name of the font.
func (f *PDType0Font) BaseFont() string { return f.dict.GetNameAsString(cos.BaseFont, "") }

// DescendantFont returns the descendant font.
func (f *PDType0Font) DescendantFont() PDCIDFont { return f.descendantFont }

// CMap returns the font's CMap.
func (f *PDType0Font) CMap() *cmap.CMap { return f.cMap }

// CMapUCS2 returns the font's UCS2 CMap, only present when this font uses a
// predefined CMap.
func (f *PDType0Font) CMapUCS2() *cmap.CMap { return f.cMapUCS2 }

// ToUnicodeCMap returns the font's /ToUnicode CMap, or nil where it has none.
func (f *PDType0Font) ToUnicodeCMap() *cmap.CMap { return f.toUnicodeCMap }

// FontDescriptor returns what the PDF says about the font.
func (f *PDType0Font) FontDescriptor() *PDFontDescriptor {
	return f.descendantFont.FontDescriptor()
}

// FontMatrix returns the transform from glyph space to text space.
func (f *PDType0Font) FontMatrix() *util.Matrix { return f.descendantFont.FontMatrix() }

// IsVertical reports whether the font is set vertically.
func (f *PDType0Font) IsVertical() bool { return f.cMap != nil && f.cMap.WMode() == 1 }

// Height returns how tall the given glyph is.
func (f *PDType0Font) Height(code int) (float32, error) {
	return f.descendantFont.Height(code, f)
}

// encodeCodePoint returns the bytes that draw the given code point.
func (f *PDType0Font) encodeCodePoint(unicode int) ([]byte, error) {
	return f.descendantFont.Encode(unicode, f)
}

// HasExplicitWidth reports whether the PDF gives a width for the glyph itself.
func (f *PDType0Font) HasExplicitWidth(code int) (bool, error) {
	return f.descendantFont.HasExplicitWidth(code, f)
}

// AverageFontWidth returns the average width of the glyphs.
func (f *PDType0Font) AverageFontWidth() float32 { return f.descendantFont.AverageFontWidth() }

// PositionVector returns where a vertically set glyph sits relative to its
// origin.
func (f *PDType0Font) PositionVector(code int) util.Vector {
	// units are always 1/1000 text space, font matrix is not used, see FOP-2252
	return f.descendantFont.PositionVector(code, f).Scale(-1 / 1000.0)
}

// Displacement returns how far the pen moves after the given glyph, in text
// space.
func (f *PDType0Font) Displacement(code int) (util.Vector, error) {
	if f.IsVertical() {
		return util.NewVector(0, f.descendantFont.VerticalDisplacementVectorY(code, f)/1000), nil
	}
	return f.pdFont.Displacement(code)
}

// Width returns how far the pen moves after the given glyph.
func (f *PDType0Font) Width(code int) (float32, error) {
	return f.descendantFont.Width(code, f)
}

// standard14Width is not supported for a Type 0 font.
//
// Java throws UnsupportedOperationException, which is unchecked.
func (f *PDType0Font) standard14Width(code int) float32 { panic("not supported") }

// WidthFromFont returns the width the font program gives for the glyph.
func (f *PDType0Font) WidthFromFont(code int) (float32, error) {
	return f.descendantFont.WidthFromFont(code, f)
}

// IsEmbedded reports whether the font program is inside the PDF.
func (f *PDType0Font) IsEmbedded() bool { return f.descendantFont.IsEmbedded() }

// ToUnicode returns what the given character code stands for, or the empty
// string where the font cannot say.
func (f *PDType0Font) ToUnicode(code int) (string, error) {
	// try to use a ToUnicode CMap
	unicode, err := f.pdFont.ToUnicode(code)
	if err != nil {
		return "", err
	}
	if unicode != "" {
		return unicode, nil
	}

	// Use identity mapping if the given ToUnicode CMap doesn't provide any
	// valid mapping. A predefined map shall only be used if there isn't any
	// ToUnicode CMap.
	// PDFBOX-6022: not when there's a predefined cmap
	if f.toUnicodeCMap != nil && !f.isCMapPredefined {
		// Java casts the code to a char, so anything past 0xFFFF wraps.
		return string(rune(uint16(code))), nil
	}

	if (f.isCMapPredefined || f.isDescendantCJK) && f.cMapUCS2 != nil {
		// if the font is composite and uses a predefined cmap (excluding
		// Identity-H/V) then or if its descendant font uses
		// Adobe-GB1/CNS1/Japan1/Korea1

		// a) Map the character code to a character identifier (CID) according
		//    to the font's CMap
		cid := f.CodeToCID(code)

		// e) Map the CID according to the CMap from step d), producing a
		//    Unicode value
		mapped, _ := f.cMapUCS2.ToUnicode(cid)
		return mapped, nil
	}

	// PDFBOX-5324: try to get unicode from font cmap
	if type2, ok := f.descendantFont.(*PDCIDFontType2); ok {
		if font := type2.TrueTypeFont(); font != nil {
			if unicode, ok := f.unicodeFromFontCmap(code, type2); ok {
				return unicode, nil
			}
		}
	}

	if !f.noUnicode[code] {
		// if no value has been produced, there is no way to obtain Unicode for
		// the character.
		cid := fmt.Sprintf("CID+%d", f.CodeToCID(code))
		slog.Warn("No Unicode mapping", "cid", cid, "code", code, "font", f.Name())
		// we keep track of which warnings have been issued, so we don't log
		// multiple times
		f.noUnicode[code] = true
	}
	return "", nil
}

// unicodeFromFontCmap is the PDFBOX-5324 fallback: read the character back out
// of the font's own cmap table.
func (f *PDType0Font) unicodeFromFontCmap(code int, type2 *PDCIDFontType2) (string, bool) {
	font := type2.TrueTypeFont()
	lookup, err := font.UnicodeCmapLookup(false)
	if err != nil {
		slog.Warn("get unicode from font cmap fail", "err", err)
		return "", false
	}
	if lookup == nil {
		return "", false
	}
	var gid int
	if f.descendantFont.IsEmbedded() {
		// original PDFBOX-5324 supported only embedded fonts
		if gid, err = f.descendantFont.CodeToGID(code, f); err != nil {
			slog.Warn("get unicode from font cmap fail", "err", err)
			return "", false
		}
	} else {
		// PDFBOX-5331: this bypasses the fallback attempt in
		// PDCIDFontType2.codeToGID() which would bring a stackoverflow
		gid = f.descendantFont.CodeToCID(code, f)
	}
	codes := lookup.GetCharCodes(gid)
	if len(codes) == 0 {
		return "", false
	}
	return string(rune(uint16(codes[0]))), true
}

// Name returns the name of the font as the PDF gives it.
func (f *PDType0Font) Name() string { return f.BaseFont() }

// BoundingBox returns the box every glyph of the font fits in.
func (f *PDType0Font) BoundingBox() (*fontutil.BoundingBox, error) {
	// Will be cached by underlying font
	return f.descendantFont.BoundingBox()
}

// ReadCode reads one character code from the stream.
func (f *PDType0Font) ReadCode(in *bytes.Reader) (int, error) {
	if f.cMap == nil {
		return 0, errors.New("required cmap is null")
	}
	return f.cMap.ReadCode(in)
}

// CodeToCID returns the CID for the given character code, or CID 0 if not
// found.
func (f *PDType0Font) CodeToCID(code int) int {
	return f.descendantFont.CodeToCID(code, f)
}

// CodeToGID returns the GID for the given character code.
func (f *PDType0Font) CodeToGID(code int) (int, error) {
	return f.descendantFont.CodeToGID(code, f)
}

// IsStandard14 reports whether the font is one of the fourteen every reader
// has, which a Type 0 font never is.
func (f *PDType0Font) IsStandard14() bool { return false }

// IsDamaged reports whether the font program could not be read.
func (f *PDType0Font) IsDamaged() bool { return f.descendantFont.IsDamaged() }

// String describes the font.
func (f *PDType0Font) String() string {
	descendant := "null"
	if f.descendantFont != nil {
		descendant = fmt.Sprintf("%T", f.descendantFont)
	}
	return fmt.Sprintf("PDType0Font/%s, PostScript name: %s", descendant, f.BaseFont())
}

// GetPath returns the outline of the given glyph, in glyph space.
func (f *PDType0Font) GetPath(code int) (*geom.Path2D, error) {
	return f.descendantFont.GetPath(code, f)
}

// GetNormalizedPath returns the outline of the given glyph, scaled so that the
// font matrix is the default one.
func (f *PDType0Font) GetNormalizedPath(code int) (*geom.Path2D, error) {
	return f.descendantFont.GetNormalizedPath(code, f)
}

// HasGlyphForCode reports whether the font has an outline for the glyph.
func (f *PDType0Font) HasGlyphForCode(code int) (bool, error) {
	return f.descendantFont.HasGlyph(code, f)
}

// GsubData returns the GSUB data if present.
//
// Java sets the field from the font program only in the embedding constructor;
// the one that reads a font out of a PDF sets GsubData.NO_DATA_FOUND, which is
// what this returns until the embedding half lands.
func (f *PDType0Font) GsubData() model.GsubData { return f.gsubData }

// EncodeGlyphID returns the encoded value for the given glyph ID.
func (f *PDType0Font) EncodeGlyphID(glyphID int) []byte {
	return f.descendantFont.EncodeGlyphID(glyphID)
}

// CmapLookup returns the CMap lookup table if present.
//
// Java sets the field from the font program only in the embedding constructor;
// the one that reads a font out of a PDF leaves it null.
func (f *PDType0Font) CmapLookup() ttf.CmapLookup { return f.cmapLookup }

// ToUnicodeWithGlyphList returns what the given character code stands for.
func (f *PDType0Font) ToUnicodeWithGlyphList(code int,
	customGlyphList *encoding.GlyphList) (string, error) {
	return f.ToUnicode(code)
}

// WillBeSubset reports whether this font will be subset when the document is
// saved, which for a font read out of a PDF is never: Java asks the embedder,
// and a read font has none. The embedding half arrives with a later slice.
func (f *PDType0Font) WillBeSubset() bool { return false }

// AddToSubset keeps the given code point when the font is subset.
//
// Java throws IllegalStateException where the font is not being subset, which
// is unchecked, so the port panics. A font read out of a PDF is never subset,
// so this always panics until the embedding half arrives.
func (f *PDType0Font) AddToSubset(codePoint int) {
	panic("This font was created with subsetting disabled")
}

// AddGlyphsToSubset keeps the given glyph ids when the font is subset.
//
// Java throws IllegalStateException where the font is not being subset.
func (f *PDType0Font) AddGlyphsToSubset(glyphIds map[int]bool) {
	panic("This font was created with subsetting disabled")
}

// Subset writes the font, subsetting it to the code points kept so far.
//
// Java hands this to the embedder, which a font read out of a PDF has none of.
func (f *PDType0Font) Subset() error {
	panic("This font was created with subsetting disabled")
}
