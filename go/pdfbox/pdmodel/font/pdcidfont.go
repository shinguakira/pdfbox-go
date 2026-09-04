package font

import (
	"fmt"
	"io"
	"log/slog"
	"math"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	fontutil "github.com/shinguakira/pdfbox-go/go/fontbox/util"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// PDCIDFont is a CIDFont: a PDF object that contains information about a
// CIDFont program. Although its Type value is Font, a CIDFont is not actually
// a font.
//
// It is not usually necessary to use this interface directly, prefer
// PDType0Font.
//
// Port of the abstract org.apache.pdfbox.pdmodel.font.PDCIDFont. The shared
// state lives in pdCIDFont, which the two descendants embed, and what varies is
// behind this interface.
type PDCIDFont interface {
	// COSObject returns the font dictionary.
	COSObject() cos.Base

	// BaseFont returns the PostScript name of the font.
	BaseFont() string

	// FontDescriptor returns the font descriptor, or nil.
	FontDescriptor() *PDFontDescriptor

	// FontMatrix returns the transformation from glyph space to text space.
	FontMatrix() *util.Matrix

	// BoundingBox returns the font's bounding box.
	BoundingBox() (*fontutil.BoundingBox, error)

	// WidthFromFont returns the width of a glyph in the embedded font file, in
	// glyph space.
	WidthFromFont(code int, parent *PDType0Font) (float32, error)

	// Height returns the height of the given character, in glyph space. This
	// can be expensive to calculate and the result is only approximate.
	//
	// Deprecated: Java marks it so, because there is no meaningful value it can
	// return.
	Height(code int, parent *PDType0Font) (float32, error)

	// GetPath returns the glyph path for the given character code, which is not
	// to be confused with unicode.
	GetPath(code int, parent *PDType0Font) (*geom.Path2D, error)

	// GetNormalizedPath returns the normalized glyph path for the given
	// character code in a PDF, normalized to the PostScript 1000 unit square,
	// with fallback glyphs where appropriate.
	GetNormalizedPath(code int, parent *PDType0Font) (*geom.Path2D, error)

	// HasGlyph reports whether this font contains a glyph for the given
	// character code in a PDF.
	HasGlyph(code int, parent *PDType0Font) (bool, error)

	// IsEmbedded reports whether the font file is embedded in the PDF.
	IsEmbedded() bool

	// IsDamaged reports whether the embedded font file is damaged.
	IsDamaged() bool

	// HasExplicitWidth reports whether the Font dictionary specifies an
	// explicit width for the given glyph. This includes Width and W but not
	// default width entries.
	HasExplicitWidth(code int, parent *PDType0Font) (bool, error)

	// PositionVector returns the position vector (v), in text space, for the
	// given character.
	PositionVector(code int, parent *PDType0Font) util.Vector

	// VerticalDisplacementVectorY returns the y-component of the vertical
	// displacement vector (w1).
	VerticalDisplacementVectorY(code int, parent *PDType0Font) float32

	// Width returns the advance width of the given character, in glyph space.
	Width(code int, parent *PDType0Font) (float32, error)

	// AverageFontWidth returns the average width of the glyphs.
	//
	// todo: this method is highly suspicious, the average glyph width is not
	// usually a good metric
	AverageFontWidth() float32

	// CIDSystemInfo returns the CIDSystemInfo, or nil if it is missing (which
	// isn't allowed but could happen).
	CIDSystemInfo() *PDCIDSystemInfo

	// CodeToCID returns the CID for the given character code, or CID 0 if not
	// found.
	CodeToCID(code int, parent *PDType0Font) int

	// CodeToGID returns the GID for the given character code.
	CodeToGID(code int, parent *PDType0Font) (int, error)

	// EncodeGlyphID returns the encoded value for the given glyph ID.
	EncodeGlyphID(glyphID int) []byte

	// Encode encodes the given Unicode code point for use in a PDF content
	// stream, as 1 to 4 bytes.
	Encode(unicode int, parent *PDType0Font) ([]byte, error)

	// cid returns the shared part, which is how the descendants reach it.
	cid() *pdCIDFont
}

// verticalDisplacementRange is one run of CIDs sharing a position vector and a
// vertical displacement.
type verticalDisplacementRange struct {
	rangeStart          int
	rangeEnd            int
	positionVector      util.Vector
	verticalDisplacment float32
}

func (r verticalDisplacementRange) rangeMatches(value int) bool {
	return value >= r.rangeStart && value <= r.rangeEnd
}

// pdCIDFont is the state every CIDFont carries.
type pdCIDFont struct {
	// self is the CIDFont this state belongs to, which is how the shared
	// methods reach the ones each descendant implements.
	self PDCIDFont

	widths       map[int]float32
	defaultWidth float32
	averageWidth float32

	// vertical displacement, individual values
	verticalDisplacementY map[int]float32
	// position vectors, individual values
	positionVectors map[int]util.Vector
	// cid-ranges for verticalDisplacements and positionVectors
	displacementRanges []verticalDisplacementRange

	dw2 [2]float32

	dict       *cos.Dictionary
	isEmbedded bool
	isDamaged  bool

	fontDescriptor *PDFontDescriptor
}

// newPDCIDFont returns the shared state of a CIDFont read out of a PDF.
//
// The descendant sets self and then calls initCIDFont: Java does the whole of
// this in one constructor.
func newPDCIDFont(fontDictionary *cos.Dictionary) pdCIDFont {
	return pdCIDFont{
		widths:                map[int]float32{},
		verticalDisplacementY: map[int]float32{},
		positionVectors:       map[int]util.Vector{},
		dw2:                   [2]float32{880, -1000},
		dict:                  fontDictionary,
	}
}

// initCIDFont finishes what newPDCIDFont starts.
func (f *pdCIDFont) initCIDFont(resourceCache ResourceCache) {
	f.readWidths()
	f.readVerticalDisplacements()

	var fd *PDFontDescriptor
	fdIndirectObject := f.dict.GetCOSObject(cos.FontDescriptor)
	if fdIndirectObject != nil && resourceCache != nil {
		fd = resourceCache.GetFontDescriptor(fdIndirectObject)
	}
	if fd == nil {
		if fdDict := f.dict.GetCOSDictionary(cos.FontDescriptor); fdDict != nil {
			fd = NewPDFontDescriptorFromDictionary(fdDict)
			if resourceCache != nil && fdIndirectObject != nil {
				resourceCache.PutFontDescriptor(fdIndirectObject, fd)
			}
		}
	}
	f.fontDescriptor = fd
}

// cid returns the shared part of a CIDFont.
func (f *pdCIDFont) cid() *pdCIDFont { return f }

func (f *pdCIDFont) readWidths() {
	// see 9.7.4.3, "Glyph Metrics in CIDFonts"
	wArray := f.dict.GetCOSArray(cos.W)
	if wArray == nil {
		return
	}
	size := wArray.Size()
	counter := 0
	for counter < size-1 {
		firstCodeBase := wArray.GetObject(counter)
		counter++
		firstCode, ok := firstCodeBase.(cos.Number)
		if !ok {
			slog.Warn("Expected a number array member", "got", firstCodeBase)
			continue
		}
		next := wArray.GetObject(counter)
		counter++
		if array, ok := next.(*cos.Array); ok {
			startRange := firstCode.IntValue()
			arraySize := array.Size()
			for i := 0; i < arraySize; i++ {
				widthBase := array.GetObject(i)
				if width, ok := widthBase.(cos.Number); ok {
					f.widths[startRange+i] = width.FloatValue()
				} else {
					slog.Warn("Expected a number array member", "got", widthBase)
				}
			}
			continue
		}
		if counter >= size {
			slog.Warn("premature end of widths array")
			break
		}
		secondCodeBase := next
		rangeWidthBase := wArray.GetObject(counter)
		counter++
		secondCode, okSecond := secondCodeBase.(cos.Number)
		rangeWidth, okWidth := rangeWidthBase.(cos.Number)
		if !okSecond || !okWidth {
			slog.Warn("Expected two numbers", "first", secondCodeBase, "second", rangeWidthBase)
			continue
		}
		startRange := firstCode.IntValue()
		endRange := secondCode.IntValue()
		width := rangeWidth.FloatValue()
		for i := startRange; i <= endRange; i++ {
			f.widths[i] = width
		}
	}
}

func (f *pdCIDFont) readVerticalDisplacements() {
	// default position vector and vertical displacement vector
	if dw2Array := f.dict.GetCOSArray(cos.DW2); dw2Array != nil {
		base0 := dw2Array.GetObject(0)
		base1 := dw2Array.GetObject(1)
		number0, ok0 := base0.(cos.Number)
		number1, ok1 := base1.(cos.Number)
		if ok0 && ok1 {
			f.dw2[0] = number0.FloatValue()
			f.dw2[1] = number1.FloatValue()
		}
	}

	// vertical metrics for individual CIDs.
	w2Array := f.dict.GetCOSArray(cos.W2)
	if w2Array == nil {
		return
	}
	// Java casts each entry to COSNumber and throws ClassCastException on
	// anything else, which is unchecked; the port stops reading instead of
	// taking down the whole page, and says so.
	numberAt := func(index int) (cos.Number, bool) {
		if index >= w2Array.Size() {
			return nil, false
		}
		number, ok := w2Array.GetObject(index).(cos.Number)
		return number, ok
	}
	for i := 0; i < w2Array.Size(); i++ {
		c, ok := numberAt(i)
		if !ok {
			slog.Warn("Expected a number in the W2 array, stopped reading it")
			return
		}
		i++
		next := w2Array.GetObject(i)
		if array, ok := next.(*cos.Array); ok {
			for j := 0; j < array.Size(); j++ {
				cid := c.IntValue() + j/3
				w1y, ok1 := array.GetObject(j).(cos.Number)
				j++
				v1x, ok2 := array.GetObject(j).(cos.Number)
				j++
				v1y, ok3 := array.GetObject(j).(cos.Number)
				if !ok1 || !ok2 || !ok3 {
					slog.Warn("Expected a number in the W2 array, stopped reading it")
					return
				}
				f.verticalDisplacementY[cid] = w1y.FloatValue()
				f.positionVectors[cid] = util.NewVector(v1x.FloatValue(), v1y.FloatValue())
			}
			continue
		}
		last, ok := next.(cos.Number)
		if !ok {
			slog.Warn("Expected a number in the W2 array, stopped reading it")
			return
		}
		first := c.IntValue()
		i++
		w1y, ok1 := numberAt(i)
		i++
		v1x, ok2 := numberAt(i)
		i++
		v1y, ok3 := numberAt(i)
		if !ok1 || !ok2 || !ok3 {
			slog.Warn("Expected a number in the W2 array, stopped reading it")
			return
		}
		f.displacementRanges = append(f.displacementRanges, verticalDisplacementRange{
			rangeStart:          first,
			rangeEnd:            last.IntValue(),
			positionVector:      util.NewVector(v1x.FloatValue(), v1y.FloatValue()),
			verticalDisplacment: w1y.FloatValue(),
		})
	}
}

// COSObject returns the font dictionary.
func (f *pdCIDFont) COSObject() cos.Base { return f.dict }

// BaseFont returns the PostScript name of the font.
func (f *pdCIDFont) BaseFont() string { return f.dict.GetNameAsString(cos.BaseFont, "") }

// FontDescriptor returns the font descriptor, may be nil.
func (f *pdCIDFont) FontDescriptor() *PDFontDescriptor { return f.fontDescriptor }

// IsEmbedded reports whether the font file is embedded in the PDF.
func (f *pdCIDFont) IsEmbedded() bool { return f.isEmbedded }

// IsDamaged reports whether the embedded font file is damaged.
func (f *pdCIDFont) IsDamaged() bool { return f.isDamaged }

// getDefaultWidth gets the default width, which defaults to 1000.
func (f *pdCIDFont) getDefaultWidth() float32 {
	if f.defaultWidth == 0 {
		if base, ok := f.dict.GetDictionaryObject(cos.DW).(cos.Number); ok {
			f.defaultWidth = base.FloatValue()
		} else {
			f.defaultWidth = 1000
		}
	}
	return f.defaultWidth
}

// getDefaultPositionVector returns the default position vector (v).
func (f *pdCIDFont) getDefaultPositionVector(cid int) util.Vector {
	return util.NewVector(f.getWidthForCID(cid)/2, f.dw2[0])
}

func (f *pdCIDFont) getWidthForCID(cid int) float32 {
	if width, ok := f.widths[cid]; ok {
		return width
	}
	return f.getDefaultWidth()
}

// HasExplicitWidth reports whether the Font dictionary specifies an explicit
// width for the given glyph.
func (f *pdCIDFont) HasExplicitWidth(code int, parent *PDType0Font) (bool, error) {
	_, ok := f.widths[f.self.CodeToCID(code, parent)]
	return ok, nil
}

// PositionVector returns the position vector (v), in text space, for the given
// character. This represents the position of vertical origin relative to
// horizontal origin; for horizontal writing it will always be (0, 0), for
// vertical writing both x and y are set.
func (f *pdCIDFont) PositionVector(code int, parent *PDType0Font) util.Vector {
	cid := f.self.CodeToCID(code, parent)
	if v, ok := f.positionVectors[cid]; ok {
		return v
	}
	for _, vdRange := range f.displacementRanges {
		if vdRange.rangeMatches(cid) {
			return vdRange.positionVector
		}
	}
	return f.getDefaultPositionVector(cid)
}

// VerticalDisplacementVectorY returns the y-component of the vertical
// displacement vector (w1).
func (f *pdCIDFont) VerticalDisplacementVectorY(code int, parent *PDType0Font) float32 {
	cid := f.self.CodeToCID(code, parent)
	if w1y, ok := f.verticalDisplacementY[cid]; ok {
		return w1y
	}
	for _, vdRange := range f.displacementRanges {
		if vdRange.rangeMatches(cid) {
			return vdRange.verticalDisplacment
		}
	}
	return f.dw2[1]
}

// Width returns the advance width of the given character, in glyph space.
//
// If you want the visual bounds of the glyph then call GetPath instead.
func (f *pdCIDFont) Width(code int, parent *PDType0Font) (float32, error) {
	// these widths are supposed to be consistent with the actual widths given
	// in the CIDFont program, but PDFBOX-563 shows that when they are not,
	// Acrobat overrides the embedded font widths with the widths given in the
	// font dictionary
	return f.getWidthForCID(f.self.CodeToCID(code, parent)), nil
}

// AverageFontWidth returns the average width of the glyphs.
//
// todo: this method is highly suspicious, the average glyph width is not
// usually a good metric
func (f *pdCIDFont) AverageFontWidth() float32 {
	if f.averageWidth == 0 {
		var totalWidths float32
		characterCount := 0
		for _, width := range f.widths {
			if width > 0 {
				totalWidths += width
				characterCount++
			}
		}
		if characterCount != 0 {
			f.averageWidth = totalWidths / float32(characterCount)
		}
		if f.averageWidth <= 0 || math.IsNaN(float64(f.averageWidth)) {
			f.averageWidth = f.getDefaultWidth()
		}
	}
	return f.averageWidth
}

// CIDSystemInfo returns the CIDSystemInfo, or nil if it is missing.
func (f *pdCIDFont) CIDSystemInfo() *PDCIDSystemInfo {
	if cidSystemInfo := f.dict.GetCOSDictionary(cos.CIDSystemInfo); cidSystemInfo != nil {
		return NewPDCIDSystemInfoFromDictionary(cidSystemInfo)
	}
	return nil
}

// Encode encodes the given Unicode code point for use in a PDF content stream.
//
// Java puts the shared method here and dispatches on the concrete class; a
// PDCIDFontType0 throws UnsupportedOperationException, which is unchecked, so
// the port panics from that font's own Encode.
func (f *pdCIDFont) Encode(unicode int, parent *PDType0Font) ([]byte, error) {
	return f.self.Encode(unicode, parent)
}

// readCIDToGIDMap reads the /CIDToGIDMap stream, or nil where there is none.
func (f *pdCIDFont) readCIDToGIDMap() ([]int, error) {
	stream, ok := f.dict.GetDictionaryObject(cos.CIDToGIDMap).(*cos.Stream)
	if !ok {
		return nil, nil
	}
	reader, err := stream.CreateReader()
	if err != nil {
		return nil, err
	}
	mapAsBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	numberOfInts := len(mapAsBytes) / 2
	cid2gid := make([]int, numberOfInts)
	offset := 0
	for index := 0; index < numberOfInts; index++ {
		gid := int(mapAsBytes[offset])&0xff<<8 | int(mapAsBytes[offset+1])&0xff
		cid2gid[index] = gid
		offset += 2
	}
	return cid2gid, nil
}

// PDCIDSystemInfo represents a CIDSystemInfo.
//
// Port of org.apache.pdfbox.pdmodel.font.PDCIDSystemInfo.
type PDCIDSystemInfo struct {
	dictionary *cos.Dictionary
}

// NewPDCIDSystemInfo returns a CIDSystemInfo naming the given collection.
func NewPDCIDSystemInfo(registry, ordering string, supplement int) *PDCIDSystemInfo {
	dictionary := cos.NewDictionary()
	dictionary.SetString(cos.Registry, registry)
	dictionary.SetString(cos.Ordering, ordering)
	dictionary.SetInt(cos.Supplement, supplement)
	return &PDCIDSystemInfo{dictionary: dictionary}
}

// NewPDCIDSystemInfoFromDictionary returns the CIDSystemInfo the given
// dictionary describes.
func NewPDCIDSystemInfoFromDictionary(dictionary *cos.Dictionary) *PDCIDSystemInfo {
	return &PDCIDSystemInfo{dictionary: dictionary}
}

// Registry returns the registry of the character collection.
func (i *PDCIDSystemInfo) Registry() string {
	return i.dictionary.GetNameAsString(cos.Registry, "")
}

// Ordering returns the ordering of the character collection.
func (i *PDCIDSystemInfo) Ordering() string {
	return i.dictionary.GetNameAsString(cos.Ordering, "")
}

// Supplement returns the supplement of the character collection.
func (i *PDCIDSystemInfo) Supplement() int { return i.dictionary.GetInt(cos.Supplement) }

// COSObject returns the dictionary.
func (i *PDCIDSystemInfo) COSObject() cos.Base { return i.dictionary }

// String names the character collection.
func (i *PDCIDSystemInfo) String() string {
	return fmt.Sprintf("%s-%s-%d", i.Registry(), i.Ordering(), i.Supplement())
}
