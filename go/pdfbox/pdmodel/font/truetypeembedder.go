package font

// Common functionality for embedding TrueType fonts.
//
// Port of the abstract, package-private
// org.apache.pdfbox.pdmodel.font.TrueTypeEmbedder, and of the interface
// Subsetter it implements.
//
// Java's abstract buildSubset is a function field here. Go has no abstract
// method, and the two concrete embedders are in this package, so a field the
// constructor of each fills in says the same thing as `extends` without an
// interface that only ever has two implementations.

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// The two fsSelection bits TrueTypeEmbedder reads.
const (
	fsSelectionItalic  = 1
	fsSelectionOblique = 512
)

// base25 is the alphabet a subset tag is written in.
const base25 = "BCDEFGHIJKLMNOPQRSTUVWXYZ"

// subsetTables are the tables a subset font keeps: the ones the PDF
// specification requires where they are present, and everything else is
// removed.
var subsetTables = []string{
	"head", "hhea", "loca", "maxp", "cvt ", "prep", "glyf", "hmtx", "fpgm",
	// Windows ClearType
	"gasp",
}

// subsetter is the interface a font subsetter satisfies.
//
// Port of the package-private interface Subsetter. The port names the methods
// the Go way; PDType0Font and trueTypeEmbedder both satisfy it.
type subsetter interface {
	// AddToSubset adds the given Unicode code point to this subset.
	AddToSubset(codePoint int)

	// Subset subsets this font now.
	Subset() error
}

// trueTypeEmbedder holds what the two TrueType embedders share.
type trueTypeEmbedder struct {
	document common.COSDocumentLike

	ttf            *ttf.TrueTypeFont
	fontDescriptor *PDFontDescriptor

	cmapLookup ttf.CmapLookup

	// subsetCodePoints is Java's LinkedHashSet: the code points actually used,
	// in first-occurrence order, which the ToUnicode CMap is built from.
	subsetCodePoints      []int
	subsetCodePointsSeen  map[int]bool
	embedSubset           bool
	allGlyphIds           map[int]bool
	buildSubsetFromStream func(ttfSubset []byte, tag string, gidToCid map[int]int) error
}

var _ subsetter = (*trueTypeEmbedder)(nil)

// newTrueTypeEmbedder creates a new TrueType font for embedding.
//
// Port of the TrueTypeEmbedder constructor. The caller fills in
// buildSubsetFromStream before anything calls Subset.
func newTrueTypeEmbedder(document common.COSDocumentLike, dict *cos.Dictionary,
	font *ttf.TrueTypeFont, embedSubset bool) (*trueTypeEmbedder, error) {
	e := &trueTypeEmbedder{
		document:             document,
		embedSubset:          embedSubset,
		ttf:                  font,
		subsetCodePointsSeen: map[int]bool{},
		allGlyphIds:          map[int]bool{},
	}

	descriptor, err := createFontDescriptor(font)
	if err != nil {
		return nil, err
	}
	e.fontDescriptor = descriptor

	permitted, err := isEmbeddingPermitted(font)
	if err != nil {
		return nil, err
	}
	if !permitted {
		return nil, errors.New("This font does not permit embedding")
	}

	if !embedSubset {
		// full embedding

		// TrueType collections are not supported
		original, err := font.OriginalData()
		if err != nil {
			return nil, err
		}
		header := make([]byte, 4)
		read, _ := io.ReadFull(original, header)
		if read == len(header) && string(header) == "ttcf" {
			return nil, errors.New("Full embedding of TrueType font collections not supported")
		}
		// Java rewinds the stream where it can and reopens it where it cannot;
		// OriginalData hands back a fresh reader each time, so reopening is the
		// only shape the port needs.
		original, err = font.OriginalData()
		if err != nil {
			return nil, err
		}
		stream, err := common.NewPDStreamOfInput(document, original, cos.FlateDecode)
		if err != nil {
			return nil, err
		}
		stream.Stream().SetLong(cos.Length1, font.OriginalDataSize())
		e.fontDescriptor.SetFontFile2(stream)
	}

	name, err := font.Name()
	if err != nil {
		return nil, err
	}
	dict.SetName(cos.BaseFont, name)

	// choose a Unicode "cmap"
	//
	// Java calls the no-argument getUnicodeCmapLookup, which is the *strict*
	// one: a font with no Unicode subtable is refused here rather than embedded
	// and found unusable later.
	lookup, err := font.UnicodeCmapLookup(true)
	if err != nil {
		return nil, err
	}
	e.cmapLookup = lookup
	return e, nil
}

// buildFontFile2 embeds the given font program and rebuilds the descriptor
// around it.
func (e *trueTypeEmbedder) buildFontFile2(ttfStream io.Reader) error {
	stream, err := common.NewPDStreamOfInput(e.document, ttfStream, cos.FlateDecode)
	if err != nil {
		return err
	}

	// as the stream was closed within the PDStream constructor, we have to
	// recreate it
	input, err := stream.CreateInputStream()
	if err != nil {
		return err
	}
	parsed, err := ttf.NewParserEmbedded(false).ParseEmbedded(input)
	if err != nil {
		return err
	}
	e.ttf = parsed
	permitted, err := isEmbeddingPermitted(e.ttf)
	if err != nil {
		return err
	}
	if !permitted {
		return errors.New("This font does not permit embedding")
	}
	if e.fontDescriptor == nil {
		descriptor, err := createFontDescriptor(e.ttf)
		if err != nil {
			return err
		}
		e.fontDescriptor = descriptor
	}
	stream.Stream().SetLong(cos.Length1, e.ttf.OriginalDataSize())
	e.fontDescriptor.SetFontFile2(stream)
	return nil
}

// isEmbeddingPermitted reports whether the fsType in the OS/2 table permits
// embedding.
func isEmbeddingPermitted(font *ttf.TrueTypeFont) (bool, error) {
	os2, err := font.OS2Windows()
	if err != nil {
		return false, err
	}
	if os2 == nil {
		return true, nil
	}
	return IsEmbeddingPermittedForFsType(os2.FsType()), nil
}

// IsEmbeddingPermittedForFsType reports whether the given OS/2 fsType permits
// embedding.
//
// Split out of isEmbeddingPermitted so that the permission table can be tested
// without a font: Java's test mocks TrueTypeFont with Mockito to set exactly
// this value, and the port's TrueTypeFont is a struct.
//
// PDFBOX-5191: the restricted-licence check compares the whole low nibble
// rather than testing the bit, because the permissions are exclusive.
func IsEmbeddingPermittedForFsType(fsType int16) bool {
	masked := fsType & 0x000F
	if masked == ttf.FsTypeRestricted {
		// restricted License embedding
		return false
	}
	if fsType&ttf.FsTypeBitmapOnly == ttf.FsTypeBitmapOnly {
		// bitmap embedding only
		return false
	}
	return true
}

// isSubsettingPermitted reports whether the fsType in the OS/2 table permits
// subsetting.
func isSubsettingPermitted(font *ttf.TrueTypeFont) (bool, error) {
	os2, err := font.OS2Windows()
	if err != nil {
		return false, err
	}
	if os2 == nil {
		return true, nil
	}
	fsType := os2.FsType()
	return fsType&ttf.FsTypeNoSubsetting != ttf.FsTypeNoSubsetting, nil
}

// FontDescriptor returns the font descriptor.
func (e *trueTypeEmbedder) FontDescriptor() *PDFontDescriptor { return e.fontDescriptor }

// AddToSubset adds the given Unicode code point to this subset.
func (e *trueTypeEmbedder) AddToSubset(codePoint int) {
	if e.subsetCodePointsSeen[codePoint] {
		return
	}
	e.subsetCodePointsSeen[codePoint] = true
	e.subsetCodePoints = append(e.subsetCodePoints, codePoint)
}

// SubsetCodePoints returns the code points that were passed to AddToSubset --
// the ones actually used in the document -- in first-occurrence order. The
// ToUnicode CMap is built from it, to map a glyph back to the code point that
// was really typed.
func (e *trueTypeEmbedder) SubsetCodePoints() []int { return e.subsetCodePoints }

// AddGlyphIds keeps the given glyph ids in the subset.
func (e *trueTypeEmbedder) AddGlyphIds(glyphIds map[int]bool) {
	for id := range glyphIds {
		e.allGlyphIds[id] = true
	}
}

// Subset subsets the font now.
func (e *trueTypeEmbedder) Subset() error {
	permitted, err := isSubsettingPermitted(e.ttf)
	if err != nil {
		return err
	}
	if !permitted {
		return errors.New("This font does not permit subsetting")
	}
	if !e.embedSubset {
		// Java throws IllegalStateException, which is unchecked, so the port
		// panics -- and PDType0Font.Subset does the same where no embedder is
		// there at all.
		panic("Subsetting is disabled")
	}

	// set the GIDs to subset
	sub, err := ttf.NewTTFSubsetterTables(e.ttf, subsetTables)
	if err != nil {
		return err
	}
	sub.AddAll(e.subsetCodePoints)
	sub.ForceInvisible(0x200B) // ZWSP
	sub.ForceInvisible(0x200C) // ZWNJ
	sub.ForceInvisible(0x2060) // WJ
	sub.ForceInvisible(0xFEFF) // ZWNBSP

	if len(e.allGlyphIds) > 0 {
		ids := make([]int, 0, len(e.allGlyphIds))
		for id := range e.allGlyphIds {
			ids = append(ids, id)
		}
		sub.AddGlyphIds(ids)
	}

	// calculate deterministic tag based on the chosen subset
	gidToCid, err := sub.GetGIDMap()
	if err != nil {
		return err
	}
	tag := subsetTag(gidToCid)
	sub.SetPrefix(tag)

	// save the subset font
	var out bytes.Buffer
	if err := sub.WriteToStream(&out); err != nil {
		return err
	}

	// re-build the embedded font
	if err := e.buildSubsetFromStream(out.Bytes(), tag, gidToCid); err != nil {
		return err
	}
	return e.ttf.Close()
}

// NeedsSubset reports whether the font needs to be subset.
func (e *trueTypeEmbedder) NeedsSubset() bool { return e.embedSubset }

// subsetTag returns an upper-case six-character tag for the given subset.
//
// Port of getTag. The tag has to match Java's byte for byte, because it becomes
// part of the font's name in the file, so the hash is Java's:
// AbstractMap.hashCode sums the entry hashes, an entry hash is
// key.hashCode() ^ value.hashCode(), and Integer.hashCode() is the value
// itself -- all of it in 32-bit arithmetic that wraps.
func subsetTag(gidToCid map[int]int) string {
	var hash int32
	for gid, cid := range gidToCid {
		hash += int32(gid) ^ int32(cid)
	}

	// hash might be negative due to an overflow if the map contains lots of
	// values.
	//
	// Java writes Math.abs(int), which answers Integer.MIN_VALUE unchanged for
	// that one input -- and the loop below then indexes BASE25 with a negative
	// remainder. Ported as written; see migration/JAVA-BUGS.md entry 74.
	abs := hash
	if abs > 0 {
		// nothing to do
	} else if abs != math.MinInt32 {
		abs = -abs
	}
	num := int64(abs)

	// base25 encode
	var sb []byte
	for {
		div := num / 25
		mod := int(num % 25)
		sb = append(sb, base25[mod])
		num = div
		if num == 0 || len(sb) >= 6 {
			break
		}
	}

	// pad
	for len(sb) < 6 {
		sb = append([]byte{'A'}, sb...)
	}

	return string(sb) + "+"
}

// createFontDescriptor builds a new font descriptor dictionary for the given
// TTF.
func createFontDescriptor(font *ttf.TrueTypeFont) (*PDFontDescriptor, error) {
	ttfName, err := font.Name()
	if err != nil {
		return nil, err
	}
	os2, err := font.OS2Windows()
	if err != nil {
		return nil, err
	}
	if os2 == nil {
		return nil, fmt.Errorf("os2 table is missing in font %s", ttfName)
	}
	post, err := font.PostScript()
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, fmt.Errorf("post table is missing in font %s", ttfName)
	}

	fd := NewPDFontDescriptor()
	fd.SetFontName(ttfName)

	hhea, err := font.HorizontalHeader()
	if err != nil {
		return nil, err
	}

	// Flags
	fd.SetFixedPitch(post.IsFixedPitch() > 0 || hhea.NumberOfHMetrics() == 1)

	fsSelection := os2.FsSelection()
	fd.SetItalic(fsSelection&(fsSelectionItalic|fsSelectionOblique) != 0)

	switch os2.FamilyClass() {
	case ttf.FamilyClassClaredonSerifs,
		ttf.FamilyClassFreeformSerifs,
		ttf.FamilyClassModernSerifs,
		ttf.FamilyClassOldstyleSerifs,
		ttf.FamilyClassSlabSerifs:
		fd.SetSerif(true)
	case ttf.FamilyClassScripts:
		fd.SetScript(true)
	}

	fd.SetFontWeight(float32(os2.WeightClass()))

	fd.SetSymbolic(true)
	fd.SetNonSymbolic(false)

	// ItalicAngle
	fd.SetItalicAngle(post.ItalicAngle())

	// FontBBox
	header, err := font.Header()
	if err != nil {
		return nil, err
	}
	rect := common.NewPDRectangle()
	scaling := 1000 / float32(header.UnitsPerEm())
	rect.SetLowerLeftX(float32(header.XMin()) * scaling)
	rect.SetLowerLeftY(float32(header.YMin()) * scaling)
	rect.SetUpperRightX(float32(header.XMax()) * scaling)
	rect.SetUpperRightY(float32(header.YMax()) * scaling)
	fd.SetFontBoundingBox(rect)

	// Ascent, Descent
	fd.SetAscent(float32(hhea.Ascender()) * scaling)
	fd.SetDescent(float32(hhea.Descender()) * scaling)

	// CapHeight, XHeight
	// Java compares an int version against the float 1.2, so it is really
	// "version 2 or later". The port keeps the comparison the Java shape has.
	if float64(os2.Version()) >= 1.2 {
		fd.SetCapHeight(float32(os2.CapHeight()) * scaling)
		fd.SetXHeight(float32(os2.Height()) * scaling)
	} else {
		capHPath, err := font.GetPath("H")
		if err != nil {
			return nil, err
		}
		if capHPath != nil {
			fd.SetCapHeight(float32(javaRoundLong(capHPath.Bounds2D().MaxY())) * scaling)
		} else {
			// estimate by summing the typographical +ve ascender and -ve
			// descender
			fd.SetCapHeight(float32(os2.TypoAscender()+os2.TypoDescender()) * scaling)
		}
		xPath, err := font.GetPath("x")
		if err != nil {
			return nil, err
		}
		if xPath != nil {
			fd.SetXHeight(float32(javaRoundLong(xPath.Bounds2D().MaxY())) * scaling)
		} else {
			// estimate by halving the typographical ascender
			fd.SetXHeight(float32(os2.TypoAscender()) / 2.0 * scaling)
		}
	}

	// StemV - there's no true TTF equivalent of this, so we estimate it
	fd.SetStemV(fd.FontBoundingBox().Width() * .13)

	return fd, nil
}

// javaRound is java.lang.Math.round(float): the closest int, with a tie going
// towards positive infinity. Go's math.Round takes a tie away from zero, which
// is the other direction for a negative value -- and every /W2 entry of a
// vertical font is a negated metric.
//
// The widening to float64 before the +0.5 is what keeps the two apart at the
// one input floor(x + 0.5f) gets wrong in float arithmetic.
func javaRound(v float32) int {
	return int(math.Floor(float64(v) + 0.5))
}

// javaRoundLong is java.lang.Math.round(double), which answers a long.
func javaRoundLong(v float64) int64 {
	return int64(math.Floor(v + 0.5))
}
