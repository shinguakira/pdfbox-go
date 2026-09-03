package font

import (
	"math"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// The bits of the /Flags entry of a font descriptor.
const (
	flagFixedPitch  = 1
	flagSerif       = 2
	flagSymbolic    = 4
	flagScript      = 8
	flagNonSymbolic = 32
	flagItalic      = 64
	flagAllCap      = 65536
	flagSmallCap    = 131072
	flagForceBold   = 262144
)

// PDFontDescriptor is what a PDF says about a font besides its glyphs: the
// metrics, the flags, and where the font program is.
//
// Port of org.apache.pdfbox.pdmodel.font.PDFontDescriptor.
type PDFontDescriptor struct {
	dic *cos.Dictionary

	// xHeight, capHeight and flags are read once and kept, as they are in
	// Java; the sentinels stand for its Float.NEGATIVE_INFINITY and -1.
	xHeight      float32
	capHeight    float32
	flags        int
	hasXHeight   bool
	hasCapHeight bool
}

var _ common.COSObjectable = (*PDFontDescriptor)(nil)

// NewPDFontDescriptor returns an empty font descriptor.
func NewPDFontDescriptor() *PDFontDescriptor {
	dic := cos.NewDictionary()
	dic.SetItem(cos.Type, cos.FontDescriptor)
	return &PDFontDescriptor{dic: dic, flags: -1}
}

// NewPDFontDescriptorFromDictionary returns the font descriptor the given
// dictionary holds.
func NewPDFontDescriptorFromDictionary(desc *cos.Dictionary) *PDFontDescriptor {
	return &PDFontDescriptor{dic: desc, flags: -1}
}

// IsFixedPitch reports whether every glyph is the same width.
func (d *PDFontDescriptor) IsFixedPitch() bool { return d.isFlagBitOn(flagFixedPitch) }

// SetFixedPitch sets whether every glyph is the same width.
func (d *PDFontDescriptor) SetFixedPitch(flag bool) { d.setFlagBit(flagFixedPitch, flag) }

// IsSerif reports whether the glyphs have serifs.
func (d *PDFontDescriptor) IsSerif() bool { return d.isFlagBitOn(flagSerif) }

// SetSerif sets whether the glyphs have serifs.
func (d *PDFontDescriptor) SetSerif(flag bool) { d.setFlagBit(flagSerif, flag) }

// IsSymbolic reports whether the font uses an encoding of its own rather than
// the standard Latin one.
func (d *PDFontDescriptor) IsSymbolic() bool { return d.isFlagBitOn(flagSymbolic) }

// SetSymbolic sets whether the font uses an encoding of its own.
func (d *PDFontDescriptor) SetSymbolic(flag bool) { d.setFlagBit(flagSymbolic, flag) }

// IsScript reports whether the glyphs look like handwriting.
func (d *PDFontDescriptor) IsScript() bool { return d.isFlagBitOn(flagScript) }

// SetScript sets whether the glyphs look like handwriting.
func (d *PDFontDescriptor) SetScript(flag bool) { d.setFlagBit(flagScript, flag) }

// IsNonSymbolic reports whether the font uses the standard Latin encoding.
func (d *PDFontDescriptor) IsNonSymbolic() bool { return d.isFlagBitOn(flagNonSymbolic) }

// SetNonSymbolic sets whether the font uses the standard Latin encoding.
func (d *PDFontDescriptor) SetNonSymbolic(flag bool) { d.setFlagBit(flagNonSymbolic, flag) }

// IsItalic reports whether the glyphs lean.
func (d *PDFontDescriptor) IsItalic() bool { return d.isFlagBitOn(flagItalic) }

// SetItalic sets whether the glyphs lean.
func (d *PDFontDescriptor) SetItalic(flag bool) { d.setFlagBit(flagItalic, flag) }

// IsAllCap reports whether the lowercase letters are drawn as capitals.
func (d *PDFontDescriptor) IsAllCap() bool { return d.isFlagBitOn(flagAllCap) }

// SetAllCap sets whether the lowercase letters are drawn as capitals.
func (d *PDFontDescriptor) SetAllCap(flag bool) { d.setFlagBit(flagAllCap, flag) }

// IsSmallCap reports whether the lowercase letters are drawn as small
// capitals.
func (d *PDFontDescriptor) IsSmallCap() bool { return d.isFlagBitOn(flagSmallCap) }

// SetSmallCap sets whether the lowercase letters are drawn as small capitals.
func (d *PDFontDescriptor) SetSmallCap(flag bool) { d.setFlagBit(flagSmallCap, flag) }

// IsForceBold reports whether the glyphs are thickened at small sizes.
func (d *PDFontDescriptor) IsForceBold() bool { return d.isFlagBitOn(flagForceBold) }

// SetForceBold sets whether the glyphs are thickened at small sizes.
func (d *PDFontDescriptor) SetForceBold(flag bool) { d.setFlagBit(flagForceBold, flag) }

func (d *PDFontDescriptor) isFlagBitOn(bit int) bool {
	return d.Flags()&bit != 0
}

func (d *PDFontDescriptor) setFlagBit(bit int, value bool) {
	flags := d.Flags()
	if value {
		flags = flags | bit
	} else {
		flags = flags &^ bit
	}
	d.SetFlags(flags)
}

// COSObject returns the dictionary behind the descriptor.
func (d *PDFontDescriptor) COSObject() cos.Base { return d.dic }

// Dictionary returns the dictionary behind the descriptor, typed.
func (d *PDFontDescriptor) Dictionary() *cos.Dictionary { return d.dic }

// FontName returns the PostScript name of the font.
func (d *PDFontDescriptor) FontName() string {
	return d.dic.GetNameAsString(cos.FontName, "")
}

// SetFontName sets the PostScript name of the font.
func (d *PDFontDescriptor) SetFontName(fontName string) {
	var name cos.Base
	if fontName != "" {
		name = cos.GetPDFName(fontName)
	}
	d.dic.SetItem(cos.FontName, name)
}

// FontFamily returns the family the font belongs to.
func (d *PDFontDescriptor) FontFamily() string {
	return d.dic.GetString(cos.FontFamily, "")
}

// SetFontFamily sets the family the font belongs to.
func (d *PDFontDescriptor) SetFontFamily(fontFamily string) {
	var name cos.Base
	if fontFamily != "" {
		name = cos.NewStringObj(fontFamily)
	}
	d.dic.SetItem(cos.FontFamily, name)
}

// FontWeight returns how heavy the font is, from 100 to 900.
func (d *PDFontDescriptor) FontWeight() float32 {
	return d.dic.GetFloat(cos.FontWeight, 0)
}

// SetFontWeight sets how heavy the font is.
func (d *PDFontDescriptor) SetFontWeight(fontWeight float32) {
	d.dic.SetFloat(cos.FontWeight, fontWeight)
}

// FontStretch returns how wide the font is.
func (d *PDFontDescriptor) FontStretch() string {
	return d.dic.GetNameAsString(cos.FontStretch, "")
}

// SetFontStretch sets how wide the font is.
func (d *PDFontDescriptor) SetFontStretch(fontStretch string) {
	var name cos.Base
	if fontStretch != "" {
		name = cos.GetPDFName(fontStretch)
	}
	d.dic.SetItem(cos.FontStretch, name)
}

// Flags returns the flag bits of the descriptor.
func (d *PDFontDescriptor) Flags() int {
	if d.flags == -1 {
		d.flags = d.dic.GetIntDefault(cos.Flags, 0)
	}
	return d.flags
}

// SetFlags sets the flag bits of the descriptor.
func (d *PDFontDescriptor) SetFlags(flags int) {
	d.dic.SetInt(cos.Flags, flags)
	d.flags = flags
}

// FontBoundingBox returns the box every glyph fits in, or nil where the
// descriptor gives none.
func (d *PDFontDescriptor) FontBoundingBox() *common.PDRectangle {
	rect := d.dic.GetCOSArray(cos.FontBBox)
	if rect == nil {
		return nil
	}
	return common.NewPDRectangleOfCOSArray(rect)
}

// SetFontBoundingBox sets the box every glyph fits in.
func (d *PDFontDescriptor) SetFontBoundingBox(rect *common.PDRectangle) {
	var array cos.Base
	if rect != nil {
		array = rect.COSArray()
	}
	d.dic.SetItem(cos.FontBBox, array)
}

// ItalicAngle returns how far the glyphs lean, in degrees counterclockwise.
func (d *PDFontDescriptor) ItalicAngle() float32 {
	return d.dic.GetFloat(cos.ItalicAngle, 0)
}

// SetItalicAngle sets how far the glyphs lean.
func (d *PDFontDescriptor) SetItalicAngle(angle float32) {
	d.dic.SetFloat(cos.ItalicAngle, angle)
}

// Ascent returns how far the tallest glyph rises above the baseline.
func (d *PDFontDescriptor) Ascent() float32 {
	return d.dic.GetFloat(cos.Ascent, 0)
}

// SetAscent sets how far the tallest glyph rises above the baseline.
func (d *PDFontDescriptor) SetAscent(ascent float32) {
	d.dic.SetFloat(cos.Ascent, ascent)
}

// Descent returns how far the deepest glyph falls below the baseline.
func (d *PDFontDescriptor) Descent() float32 {
	return d.dic.GetFloat(cos.Descent, 0)
}

// SetDescent sets how far the deepest glyph falls below the baseline.
func (d *PDFontDescriptor) SetDescent(descent float32) {
	d.dic.SetFloat(cos.Descent, descent)
}

// Leading returns the gap between two lines of the font.
func (d *PDFontDescriptor) Leading() float32 {
	return d.dic.GetFloat(cos.Leading, 0)
}

// SetLeading sets the gap between two lines of the font.
func (d *PDFontDescriptor) SetLeading(leading float32) {
	d.dic.SetFloat(cos.Leading, leading)
}

// CapHeight returns how tall a capital letter is.
func (d *PDFontDescriptor) CapHeight() float32 {
	if !d.hasCapHeight {
		d.capHeight = float32(math.Abs(float64(d.dic.GetFloat(cos.CapHeight, 0))))
		d.hasCapHeight = true
	}
	return d.capHeight
}

// SetCapHeight sets how tall a capital letter is.
func (d *PDFontDescriptor) SetCapHeight(capHeight float32) {
	d.dic.SetFloat(cos.CapHeight, capHeight)
	d.capHeight = capHeight
	d.hasCapHeight = true
}

// XHeight returns how tall a lowercase x is.
func (d *PDFontDescriptor) XHeight() float32 {
	if !d.hasXHeight {
		d.xHeight = float32(math.Abs(float64(d.dic.GetFloat(cos.XHeight, 0))))
		d.hasXHeight = true
	}
	return d.xHeight
}

// SetXHeight sets how tall a lowercase x is.
func (d *PDFontDescriptor) SetXHeight(xHeight float32) {
	d.dic.SetFloat(cos.XHeight, xHeight)
	d.xHeight = xHeight
	d.hasXHeight = true
}

// StemV returns how thick the vertical stems are.
func (d *PDFontDescriptor) StemV() float32 {
	return d.dic.GetFloat(cos.StemV, 0)
}

// SetStemV sets how thick the vertical stems are.
func (d *PDFontDescriptor) SetStemV(stemV float32) {
	d.dic.SetFloat(cos.StemV, stemV)
}

// StemH returns how thick the horizontal stems are.
func (d *PDFontDescriptor) StemH() float32 {
	return d.dic.GetFloat(cos.StemH, 0)
}

// SetStemH sets how thick the horizontal stems are.
func (d *PDFontDescriptor) SetStemH(stemH float32) {
	d.dic.SetFloat(cos.StemH, stemH)
}

// AverageWidth returns the average width of the glyphs.
func (d *PDFontDescriptor) AverageWidth() float32 {
	return d.dic.GetFloat(cos.AvgWidth, 0)
}

// SetAverageWidth sets the average width of the glyphs.
func (d *PDFontDescriptor) SetAverageWidth(averageWidth float32) {
	d.dic.SetFloat(cos.AvgWidth, averageWidth)
}

// MaxWidth returns the width of the widest glyph.
func (d *PDFontDescriptor) MaxWidth() float32 {
	return d.dic.GetFloat(cos.MaxWidth, 0)
}

// SetMaxWidth sets the width of the widest glyph.
func (d *PDFontDescriptor) SetMaxWidth(maxWidth float32) {
	d.dic.SetFloat(cos.MaxWidth, maxWidth)
}

// HasWidths reports whether the descriptor says anything about glyph widths.
func (d *PDFontDescriptor) HasWidths() bool {
	return d.dic.ContainsKey(cos.Widths) || d.dic.ContainsKey(cos.MissingWidth)
}

// HasMissingWidth reports whether the descriptor gives a width for a glyph the
// widths array does not cover.
func (d *PDFontDescriptor) HasMissingWidth() bool {
	return d.dic.ContainsKey(cos.MissingWidth)
}

// MissingWidth returns the width of a glyph the widths array does not cover.
func (d *PDFontDescriptor) MissingWidth() float32 {
	return d.dic.GetFloat(cos.MissingWidth, 0)
}

// SetMissingWidth sets the width of a glyph the widths array does not cover.
func (d *PDFontDescriptor) SetMissingWidth(missingWidth float32) {
	d.dic.SetFloat(cos.MissingWidth, missingWidth)
}

// CharSet returns the character set of the font, as a string of glyph names.
func (d *PDFontDescriptor) CharSet() string {
	return d.dic.GetString(cos.CharSet, "")
}

// SetCharacterSet sets the character set of the font.
func (d *PDFontDescriptor) SetCharacterSet(charSet string) {
	var name cos.Base
	if charSet != "" {
		name = cos.NewStringObj(charSet)
	}
	d.dic.SetItem(cos.CharSet, name)
}

// FontFile returns the embedded Type 1 font program, or nil where there is
// none.
func (d *PDFontDescriptor) FontFile() *common.PDStream {
	return d.pdStream(cos.FontFile)
}

// SetFontFile sets the embedded Type 1 font program.
func (d *PDFontDescriptor) SetFontFile(type1Stream *common.PDStream) {
	d.setPDStream(cos.FontFile, type1Stream)
}

// FontFile2 returns the embedded TrueType font program, or nil where there is
// none.
func (d *PDFontDescriptor) FontFile2() *common.PDStream {
	return d.pdStream(cos.FontFile2)
}

// SetFontFile2 sets the embedded TrueType font program.
func (d *PDFontDescriptor) SetFontFile2(ttfStream *common.PDStream) {
	d.setPDStream(cos.FontFile2, ttfStream)
}

// FontFile3 returns the embedded font program of any other format, or nil
// where there is none.
func (d *PDFontDescriptor) FontFile3() *common.PDStream {
	return d.pdStream(cos.FontFile3)
}

// SetFontFile3 sets the embedded font program of any other format.
func (d *PDFontDescriptor) SetFontFile3(stream *common.PDStream) {
	d.setPDStream(cos.FontFile3, stream)
}

// CIDSet returns which CIDs the subset covers, or nil where the descriptor
// gives none.
func (d *PDFontDescriptor) CIDSet() *common.PDStream {
	return d.pdStream(cos.CIDSet)
}

// SetCIDSet sets which CIDs the subset covers.
func (d *PDFontDescriptor) SetCIDSet(stream *common.PDStream) {
	d.setPDStream(cos.CIDSet, stream)
}

// pdStream returns the stream under the given key, or nil where there is none.
func (d *PDFontDescriptor) pdStream(key *cos.Name) *common.PDStream {
	stream, _ := d.dic.GetDictionaryObject(key).(*cos.Stream)
	if stream == nil {
		return nil
	}
	return common.NewPDStream(stream)
}

// setPDStream sets the stream under the given key.
func (d *PDFontDescriptor) setPDStream(key *cos.Name, stream *common.PDStream) {
	var value cos.Base
	if stream != nil {
		value = stream.COSObject()
	}
	d.dic.SetItem(key, value)
}

// Panose returns the PANOSE classification of the font, or nil where the
// descriptor gives none.
//
// JAVA-BUGS entry 14: a /Style dictionary with no /Panose entry makes Java
// throw NullPointerException rather than returning null. Ported as written; the
// Go panics where Java does.
func (d *PDFontDescriptor) Panose() *PDPanose {
	style := d.dic.GetCOSDictionary(cos.Style)
	if style != nil {
		panose := style.GetDictionaryObject(cos.Panose).(*cos.StringObj)
		bytes := panose.Bytes()
		if len(bytes) >= PanoseLength {
			return NewPDPanose(bytes)
		}
	}
	return nil
}
