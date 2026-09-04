package font

import (
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/fontbox"
	"github.com/shinguakira/pdfbox-go/go/fontbox/cff"
	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf"
	fontutil "github.com/shinguakira/pdfbox-go/go/fontbox/util"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// PDCIDFontType0 is a Type 0 CIDFont (CFF).
//
// Port of org.apache.pdfbox.pdmodel.font.PDCIDFontType0.
//
// The substitute the system supplies for a font that is not embedded needs the
// font mapper chain (B6); until that lands both fonts are nil for such a font
// and every path that reads the font program fails rather than guessing, which
// also means substituteUnicodeCmap is always nil here. See migration/STATUS.md.
type PDCIDFontType0 struct {
	pdCIDFont

	cidFont *cff.CFFCIDFont     // Top DICT that uses CIDFont operators
	t1Font  fontbox.FontBoxFont // Top DICT that does not use CIDFont operators

	// substitute is CID-keyed with a different ROS: this font's CIDs are
	// meaningless in it, resolve glyphs via Unicode instead (PDFBOX-6249)
	substituteUnicodeCmap ttf.CmapLookup

	glyphHeights        map[int]float32
	fontMatrixTransform *geom.AffineTransform
	avgWidth            float32
	avgWidthSet         bool
	fontMatrix          *util.Matrix
	fontBBox            *fontutil.BoundingBox
	cid2gid             []int
}

var _ PDCIDFont = (*PDCIDFontType0)(nil)

// NewPDCIDFontType0 returns the Type 0 CIDFont the given dictionary describes.
func NewPDCIDFontType0(fontDictionary *cos.Dictionary,
	resourceCache ResourceCache) (*PDCIDFontType0, error) {
	f := &PDCIDFontType0{
		pdCIDFont:    newPDCIDFont(fontDictionary),
		glyphHeights: map[int]float32{},
	}
	f.pdCIDFont.self = f
	f.initCIDFont(resourceCache)

	fontIsDamaged := false
	var cffFont cff.CFFFont
	fd := f.FontDescriptor()
	if fd != nil {
		if ff3Stream := fd.FontFile3(); ff3Stream != nil {
			parsed, damaged, err := readEmbeddedCFFAny(ff3Stream.Stream(), fd.FontName())
			if err != nil {
				slog.Error("Can't read the embedded CFF font", "font", fd.FontName(), "err", err)
				fontIsDamaged = true
			} else {
				cffFont = parsed
				fontIsDamaged = damaged
			}
		}
	}

	if cffFont != nil {
		// embedded
		if cidFont, ok := cffFont.(*cff.CFFCIDFont); ok {
			f.cidFont = cidFont
			f.t1Font = nil
		} else {
			f.cidFont = nil
			f.t1Font = cffFont
		}
		var err error
		if f.cid2gid, err = f.readCIDToGIDMap(); err != nil {
			return nil, err
		}
		f.isEmbedded = true
		f.isDamaged = false
	} else {
		// Java finds the font or a substitute here, through the font mapper;
		// that is B6, so both fonts stay nil.
		f.isEmbedded = false
		f.isDamaged = fontIsDamaged
	}
	f.fontMatrixTransform = f.FontMatrix().CreateAffineTransform()
	f.fontMatrixTransform.Scale(1000, 1000)
	return f, nil
}

// readEmbeddedCFFAny parses the /FontFile3 stream of a CIDFont, which may hold
// either kind of CFF font. The second result says whether what it found leaves
// the font damaged.
func readEmbeddedCFFAny(stream *cos.Stream, name string) (cff.CFFFont, bool, error) {
	randomAccessRead, err := stream.CreateView()
	if err != nil {
		return nil, false, err
	}
	defer pdfio.CloseQuietly(randomAccessRead)
	length, err := randomAccessRead.Length()
	if err != nil {
		return nil, false, err
	}
	if length > 0 {
		first, err := peekByte(randomAccessRead)
		if err != nil {
			return nil, false, err
		}
		if first == '%' {
			// PDFBOX-2642 contains a corrupt PFB font instead of a CFF
			slog.Warn("Found PFB but expected embedded CFF font", "font", name)
			return nil, true, nil
		}
	}
	fonts, err := cff.NewCFFParser().Parse(randomAccessRead)
	if err != nil {
		return nil, false, err
	}
	if len(fonts) == 0 {
		// Java indexes the list and throws IndexOutOfBounds on an empty parse,
		// which the catch turns into a damaged font.
		return nil, true, nil
	}
	return fonts[0], false, nil
}

// peekByte reads one byte and puts the cursor back, which is Java's
// RandomAccessRead.peek.
func peekByte(randomAccessRead pdfio.RandomAccessRead) (int, error) {
	position, err := randomAccessRead.Position()
	if err != nil {
		return -1, err
	}
	b, err := randomAccessRead.ReadByte()
	if err != nil {
		return -1, err
	}
	if _, err := randomAccessRead.Seek(position, 0); err != nil {
		return -1, err
	}
	return int(b), nil
}

// FontMatrix returns the transformation from glyph space to text space.
func (f *PDCIDFontType0) FontMatrix() *util.Matrix {
	if f.fontMatrix != nil {
		return f.fontMatrix
	}
	var numbers []float32
	if f.cidFont != nil {
		numbers, _ = f.cidFont.FontMatrix()
	} else if f.t1Font != nil {
		var err error
		if numbers, err = f.t1Font.FontMatrix(); err != nil {
			slog.Debug("Couldn't get font matrix - returning default value", "err", err)
			return util.NewMatrixOf(0.001, 0, 0, 0.001, 0, 0)
		}
	}

	if len(numbers) == 6 {
		f.fontMatrix = util.NewMatrixOf(numbers[0], numbers[1], numbers[2], numbers[3],
			numbers[4], numbers[5])
	} else {
		f.fontMatrix = util.NewMatrixOf(0.001, 0, 0, 0.001, 0, 0)
	}
	return f.fontMatrix
}

// BoundingBox returns the font's bounding box.
func (f *PDCIDFontType0) BoundingBox() (*fontutil.BoundingBox, error) {
	if f.fontBBox == nil {
		f.fontBBox = f.generateBoundingBox()
	}
	return f.fontBBox, nil
}

func (f *PDCIDFontType0) generateBoundingBox() *fontutil.BoundingBox {
	if fd := f.FontDescriptor(); fd != nil {
		bbox := fd.FontBoundingBox()
		if isNonZeroBoundingBox(bbox) {
			return fontutil.NewBoundingBoxOf(bbox.LowerLeftX(), bbox.LowerLeftY(),
				bbox.UpperRightX(), bbox.UpperRightY())
		}
	}
	var bbox *fontutil.BoundingBox
	var err error
	if f.cidFont != nil {
		bbox, err = f.cidFont.FontBBox()
	} else if f.t1Font != nil {
		bbox, err = f.t1Font.FontBBox()
	} else {
		err = errNoCIDFontProgram
	}
	if err != nil {
		slog.Debug("Couldn't get font bounding box - returning default value", "err", err)
		return fontutil.NewBoundingBox()
	}
	return bbox
}

// CFFFont returns the embedded CFF CIDFont, or nil if the substitute is not a
// CFF font.
func (f *PDCIDFontType0) CFFFont() cff.CFFFont {
	if f.cidFont != nil {
		return f.cidFont
	}
	if type1, ok := f.t1Font.(*cff.CFFType1Font); ok {
		return type1
	}
	return nil
}

// FontBoxFont returns the embedded or substituted font.
func (f *PDCIDFontType0) FontBoxFont() fontbox.FontBoxFont {
	if f.cidFont != nil {
		return f.cidFont
	}
	return f.t1Font
}

// GetType2CharString returns the Type 2 charstring for the given CID, or nil if
// the substituted font does not contain Type 2 charstrings.
func (f *PDCIDFontType0) GetType2CharString(cid int) (*cff.Type2CharString, error) {
	if f.cidFont != nil {
		return f.cidFont.GetType2CharString(cid)
	}
	if type1, ok := f.t1Font.(*cff.CFFType1Font); ok {
		return type1.GetType2CharString(cid)
	}
	return nil, nil
}

// codeToSubstituteGID is the GID in the substitute for the given code, via
// Unicode; -1 if unmapped. The substitute's Identity charset makes GIDs address
// its charstrings directly.
func (f *PDCIDFontType0) codeToSubstituteGID(code int, parent *PDType0Font) int {
	unicodes, err := parent.ToUnicode(code)
	if err != nil || unicodes == "" {
		return -1
	}
	return f.substituteUnicodeCmap.GetGlyphID(codePointAt(utf16Units(unicodes), 0))
}

// getGlyphName returns the name of the glyph with the given character code.
// This is done by looking up the code in the parent font's ToUnicode map and
// generating a glyph name from that.
func (f *PDCIDFontType0) getGlyphName(code int, parent *PDType0Font) string {
	unicodes, err := parent.ToUnicode(code)
	if err != nil || unicodes == "" {
		return ".notdef"
	}
	return getUniNameOfCodePoint(codePointAt(utf16Units(unicodes), 0))
}

// GetPath returns the glyph path for the given character code.
func (f *PDCIDFontType0) GetPath(code int, parent *PDType0Font) (*geom.Path2D, error) {
	cid := f.CodeToCID(code, parent)
	if f.substituteUnicodeCmap != nil {
		gid := max(f.codeToSubstituteGID(code, parent), 0)
		charString, err := f.GetType2CharString(gid)
		if err != nil {
			return nil, err
		}
		return charString.Path(), nil
	}
	if f.cid2gid != nil && f.isEmbedded {
		// PDFBOX-4093: despite being a type 0 font, there is a CIDToGIDMap
		cid = f.cid2gid[cid]
	}
	charstring, err := f.GetType2CharString(cid)
	if err != nil {
		return nil, err
	}
	if charstring != nil {
		return charstring.Path(), nil
	}
	if type1, ok := f.t1Font.(*cff.CFFType1Font); ok && f.isEmbedded {
		charString, err := type1.GetType2CharString(cid)
		if err != nil {
			return nil, err
		}
		return charString.Path(), nil
	}
	if f.t1Font == nil {
		return nil, errNoCIDFontProgram
	}
	return f.t1Font.GetPath(f.getGlyphName(code, parent))
}

// GetNormalizedPath returns the glyph path for the given character code.
func (f *PDCIDFontType0) GetNormalizedPath(code int, parent *PDType0Font) (*geom.Path2D, error) {
	return f.GetPath(code, parent)
}

// HasGlyph reports whether this font contains a glyph for the given code.
func (f *PDCIDFontType0) HasGlyph(code int, parent *PDType0Font) (bool, error) {
	cid := f.CodeToCID(code, parent)
	if f.substituteUnicodeCmap != nil {
		return f.codeToSubstituteGID(code, parent) > 0, nil
	}
	charstring, err := f.GetType2CharString(cid)
	if err != nil {
		return false, err
	}
	if charstring != nil {
		return charstring.GID() != 0, nil
	}
	if type1, ok := f.t1Font.(*cff.CFFType1Font); ok && f.isEmbedded {
		charString, err := type1.GetType2CharString(cid)
		if err != nil {
			return false, err
		}
		return charString.GID() != 0, nil
	}
	if f.t1Font == nil {
		return false, errNoCIDFontProgram
	}
	return f.t1Font.HasGlyph(f.getGlyphName(code, parent))
}

// CodeToCID returns the CID for the given character code, or CID 0 if not
// found.
func (f *PDCIDFontType0) CodeToCID(code int, parent *PDType0Font) int {
	return parent.CMap().ToCID(code)
}

// CodeToGID returns the GID for the given character code.
func (f *PDCIDFontType0) CodeToGID(code int, parent *PDType0Font) (int, error) {
	cid := f.CodeToCID(code, parent)
	if f.substituteUnicodeCmap != nil {
		return max(f.codeToSubstituteGID(code, parent), 0), nil
	}
	if f.cidFont != nil {
		// The CIDs shall be used to determine the GID value for the glyph
		// procedure using the charset table in the CFF program
		return f.cidFont.Charset().GIDForCID(cid), nil
	}
	// The CIDs shall be used directly as GID values
	return cid, nil
}

// EncodeGlyphID is not supported for a Type 0 CIDFont.
//
// Java throws UnsupportedOperationException, which is unchecked.
func (f *PDCIDFontType0) EncodeGlyphID(glyphID int) []byte { panic("not supported") }

// Encode is not supported for a Type 0 CIDFont.
//
// Java's PDCIDFont.encode throws UnsupportedOperationException for this font:
// todo: we can use a known character collection CMap for a CIDFont and an
// Encoding for Type 1-equivalent.
func (f *PDCIDFontType0) Encode(unicode int, parent *PDType0Font) ([]byte, error) {
	panic("not supported")
}

// WidthFromFont returns the width the font program gives for the glyph.
func (f *PDCIDFontType0) WidthFromFont(code int, parent *PDType0Font) (float32, error) {
	cid := f.CodeToCID(code, parent)
	var width float32
	switch {
	case f.substituteUnicodeCmap != nil:
		charString, err := f.GetType2CharString(max(f.codeToSubstituteGID(code, parent), 0))
		if err != nil {
			return 0, err
		}
		width = float32(charString.Width())
	case f.cidFont != nil:
		charString, err := f.GetType2CharString(cid)
		if err != nil {
			return 0, err
		}
		width = float32(charString.Width())
	default:
		if type1, ok := f.t1Font.(*cff.CFFType1Font); ok && f.isEmbedded {
			charString, err := type1.GetType2CharString(cid)
			if err != nil {
				return 0, err
			}
			width = float32(charString.Width())
		} else {
			if f.t1Font == nil {
				return 0, errNoCIDFontProgram
			}
			var err error
			if width, err = f.t1Font.GetWidth(f.getGlyphName(code, parent)); err != nil {
				return 0, err
			}
		}
	}

	p := geom.NewPointFloat(width, 0)
	f.fontMatrixTransform.Transform(p, p)
	return float32(p.X()), nil
}

// Height returns how tall the given glyph is.
func (f *PDCIDFontType0) Height(code int, parent *PDType0Font) (float32, error) {
	cid := f.CodeToCID(code, parent)
	if height, ok := f.glyphHeights[cid]; ok {
		return height, nil
	}
	charString, err := f.GetType2CharString(cid)
	if err != nil {
		return 0, err
	}
	if charString == nil {
		return 0, errNoCIDFontProgram
	}
	height := float32(charString.Bounds().Height)
	f.glyphHeights[cid] = height
	return height, nil
}

// AverageFontWidth returns the average width of the glyphs.
func (f *PDCIDFontType0) AverageFontWidth() float32 {
	if !f.avgWidthSet {
		f.avgWidth = f.averageCharacterWidth()
		f.avgWidthSet = true
	}
	return f.avgWidth
}

// averageCharacterWidth is a replacement for a FontMetrics method.
//
// todo: not implemented, highly suspect
func (f *PDCIDFontType0) averageCharacterWidth() float32 { return 500 }
