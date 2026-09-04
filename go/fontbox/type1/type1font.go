package type1

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/fontbox"
	"github.com/shinguakira/pdfbox-go/go/fontbox/cff"
	"github.com/shinguakira/pdfbox-go/go/fontbox/encoding"
	"github.com/shinguakira/pdfbox-go/go/fontbox/pfb"
	"github.com/shinguakira/pdfbox-go/go/fontbox/util"
)

// Type1CharStringReader is something which can read Type 1 CharStrings, namely
// Type 1 and CFF fonts.
//
// Port of the org.apache.fontbox.type1.Type1CharStringReader interface, which
// is declared in cff because it names Type1CharString and Go forbids the import
// cycle Java has. This alias puts the Java name in the Java place.
type Type1CharStringReader = cff.Type1CharStringReader

// Type1Font represents an Adobe Type 1 (.pfb) font. Thread safe.
//
// Port of org.apache.fontbox.type1.Type1Font.
type Type1Font struct {
	// font dictionary
	fontName    string
	encoding    *encoding.Encoding
	paintType   int
	fontType    int
	fontMatrix  []any
	fontBBox    []any
	uniqueID    int
	strokeWidth float32
	fontID      string

	// FontInfo dictionary
	version            string
	notice             string
	fullName           string
	familyName         string
	weight             string
	italicAngle        float32
	isFixedPitch       bool
	underlinePosition  float32
	underlineThickness float32

	// Private dictionary
	blueValues       []any
	otherBlues       []any
	familyBlues      []any
	familyOtherBlues []any
	blueScale        float32
	blueShift        int
	blueFuzz         int
	stdHW            []any
	stdVW            []any
	stemSnapH        []any
	stemSnapV        []any
	forceBold        bool
	languageGroup    int

	// Subrs array, and CharStrings dictionary
	subrs [][]byte
	// charstrings is Java's LinkedHashMap, so charstringNames keeps the order
	// the parser read the glyphs in.
	charstrings     map[string][]byte
	charstringNames []string

	// private caches
	charStringCache map[string]*cff.Type1CharString

	charStringParser *cff.Type1CharStringParser

	// raw data
	segment1 []byte
	segment2 []byte
}

var (
	_ fontbox.FontBoxFont       = (*Type1Font)(nil)
	_ fontbox.EncodedFont       = (*Type1Font)(nil)
	_ cff.Type1CharStringReader = (*Type1Font)(nil)
)

// CreateWithPFB constructs a new Type1Font object from a .pfb stream, including
// headers.
func CreateWithPFB(pfbStream io.Reader) (*Type1Font, error) {
	parsedPfb, err := pfb.NewPfbParser(pfbStream)
	if err != nil {
		return nil, err
	}
	return newType1Parser().parse(parsedPfb.Segment1(), parsedPfb.Segment2())
}

// CreateWithPFBBytes constructs a new Type1Font object from .pfb data,
// including headers.
func CreateWithPFBBytes(pfbBytes []byte) (*Type1Font, error) {
	parsedPfb, err := pfb.NewPfbParserBytes(pfbBytes)
	if err != nil {
		return nil, err
	}
	return newType1Parser().parse(parsedPfb.Segment1(), parsedPfb.Segment2())
}

// CreateWithSegments constructs a new Type1Font object from two header-less
// .pfb segments.
func CreateWithSegments(segment1, segment2 []byte) (*Type1Font, error) {
	return newType1Parser().parse(segment1, segment2)
}

// newType1Font constructs a new Type1Font, called by the parser.
func newType1Font(segment1, segment2 []byte) *Type1Font {
	return &Type1Font{
		segment1:        segment1,
		segment2:        segment2,
		charstrings:     map[string][]byte{},
		charStringCache: map[string]*cff.Type1CharString{},
	}
}

// SubrsArray returns the /Subrs array as raw bytes.
//
// Java wraps it in Collections.unmodifiableList; Go has no such wrapper, so the
// port returns a copy.
func (f *Type1Font) SubrsArray() [][]byte {
	out := make([][]byte, len(f.subrs))
	copy(out, f.subrs)
	return out
}

// CharStringsDict returns the /CharStrings dictionary as raw bytes.
//
// Java wraps it in Collections.unmodifiableMap; the port returns a copy.
func (f *Type1Font) CharStringsDict() map[string][]byte {
	out := make(map[string][]byte, len(f.charstrings))
	for name, bytes := range f.charstrings {
		out[name] = bytes
	}
	return out
}

// CharStringNames returns the glyph names in the order the /CharStrings
// dictionary lists them, which Java's LinkedHashMap keeps and a Go map does
// not.
func (f *Type1Font) CharStringNames() []string {
	out := make([]string, len(f.charstringNames))
	copy(out, f.charstringNames)
	return out
}

// Name returns the font name.
func (f *Type1Font) Name() (string, error) { return f.fontName, nil }

// GetPath returns the outline of the named glyph.
func (f *Type1Font) GetPath(name string) (*geom.Path2D, error) {
	charString, err := f.Type1CharString(name)
	if err != nil {
		return nil, err
	}
	return charString.Path(), nil
}

// GetWidth returns how far the pen moves after the named glyph.
func (f *Type1Font) GetWidth(name string) (float32, error) {
	charString, err := f.Type1CharString(name)
	if err != nil {
		return 0, err
	}
	return float32(charString.Width()), nil
}

// HasGlyph reports whether the font has the named glyph.
func (f *Type1Font) HasGlyph(name string) (bool, error) {
	_, ok := f.charstrings[name]
	return ok, nil
}

// Type1CharString returns the Type 1 CharString for the character with the
// given name.
func (f *Type1Font) Type1CharString(name string) (*cff.Type1CharString, error) {
	if type1, ok := f.charStringCache[name]; ok {
		return type1, nil
	}
	bytes, ok := f.charstrings[name]
	if !ok {
		if bytes, ok = f.charstrings[".notdef"]; !ok {
			return nil, errors.New(".notdef is not defined")
		}
	}
	sequence, err := f.parser().Parse(bytes, f.subrs, name)
	if err != nil {
		return nil, err
	}
	type1 := cff.NewType1CharString(f, f.fontName, name, sequence)
	f.charStringCache[name] = type1
	return type1, nil
}

func (f *Type1Font) parser() *cff.Type1CharStringParser {
	if f.charStringParser == nil {
		f.charStringParser = cff.NewType1CharStringParser(f.fontName)
	}
	return f.charStringParser
}

// font dictionary

// FontName returns the font name.
func (f *Type1Font) FontName() string { return f.fontName }

// Encoding returns the Encoding, or nil if not present.
func (f *Type1Font) Encoding() *encoding.Encoding { return f.encoding }

// PaintType returns the paint type.
func (f *Type1Font) PaintType() int { return f.paintType }

// FontType returns the font type.
func (f *Type1Font) FontType() int { return f.fontType }

// FontMatrix returns the font matrix.
//
// Java hands back the List<Number> the parser stored; the port's FontBoxFont
// declares the six floats the callers all want, so the list is narrowed here.
func (f *Type1Font) FontMatrix() ([]float32, error) {
	matrix := make([]float32, len(f.fontMatrix))
	for i, n := range f.fontMatrix {
		matrix[i] = numberFloat(n)
	}
	return matrix, nil
}

// FontMatrixNumbers returns the font matrix as the parser read it, an Integer
// or a Float per entry.
func (f *Type1Font) FontMatrixNumbers() []any { return copyNumbers(f.fontMatrix) }

// FontBBox returns the font bounding box, reporting an error if there are less
// than 4 numbers.
func (f *Type1Font) FontBBox() (*util.BoundingBox, error) {
	if len(f.fontBBox) < 4 {
		return nil, fmt.Errorf("FontBBox must have 4 numbers, but is %s", numbersString(f.fontBBox))
	}
	return util.NewBoundingBoxOf(
		numberFloat(f.fontBBox[0]), numberFloat(f.fontBBox[1]),
		numberFloat(f.fontBBox[2]), numberFloat(f.fontBBox[3])), nil
}

// UniqueID returns the unique ID.
func (f *Type1Font) UniqueID() int { return f.uniqueID }

// StrokeWidth returns the stroke width.
func (f *Type1Font) StrokeWidth() float32 { return f.strokeWidth }

// FontID returns the font ID.
func (f *Type1Font) FontID() string { return f.fontID }

// FontInfo dictionary

// Version returns the version.
func (f *Type1Font) Version() string { return f.version }

// Notice returns the notice.
func (f *Type1Font) Notice() string { return f.notice }

// FullName returns the full name.
func (f *Type1Font) FullName() string { return f.fullName }

// FamilyName returns the family name.
func (f *Type1Font) FamilyName() string { return f.familyName }

// Weight returns the weight.
func (f *Type1Font) Weight() string { return f.weight }

// ItalicAngle returns the italic angle.
func (f *Type1Font) ItalicAngle() float32 { return f.italicAngle }

// IsFixedPitch determines if the font has a fixed pitch.
func (f *Type1Font) IsFixedPitch() bool { return f.isFixedPitch }

// UnderlinePosition returns the underline position.
func (f *Type1Font) UnderlinePosition() float32 { return f.underlinePosition }

// UnderlineThickness returns the underline thickness.
func (f *Type1Font) UnderlineThickness() float32 { return f.underlineThickness }

// Private dictionary

// BlueValues returns the blues values.
func (f *Type1Font) BlueValues() []any { return copyNumbers(f.blueValues) }

// OtherBlues returns the other blues values.
func (f *Type1Font) OtherBlues() []any { return copyNumbers(f.otherBlues) }

// FamilyBlues returns the family blues values.
func (f *Type1Font) FamilyBlues() []any { return copyNumbers(f.familyBlues) }

// FamilyOtherBlues returns the other family blues values.
func (f *Type1Font) FamilyOtherBlues() []any { return copyNumbers(f.familyOtherBlues) }

// BlueScale returns the blue scale.
func (f *Type1Font) BlueScale() float32 { return f.blueScale }

// BlueShift returns the blue shift.
func (f *Type1Font) BlueShift() int { return f.blueShift }

// BlueFuzz returns the blue fuzz.
func (f *Type1Font) BlueFuzz() int { return f.blueFuzz }

// StdHW returns the StdHW value.
func (f *Type1Font) StdHW() []any { return copyNumbers(f.stdHW) }

// StdVW returns the StdVW value.
func (f *Type1Font) StdVW() []any { return copyNumbers(f.stdVW) }

// StemSnapH returns the StemSnapH value.
func (f *Type1Font) StemSnapH() []any { return copyNumbers(f.stemSnapH) }

// StemSnapV returns the StemSnapV value.
func (f *Type1Font) StemSnapV() []any { return copyNumbers(f.stemSnapV) }

// IsForceBold determines if the font is bold.
func (f *Type1Font) IsForceBold() bool { return f.forceBold }

// LanguageGroup returns the language group.
func (f *Type1Font) LanguageGroup() int { return f.languageGroup }

// ASCIISegment returns the ASCII segment.
func (f *Type1Font) ASCIISegment() []byte { return f.segment1 }

// BinarySegment returns the binary segment.
func (f *Type1Font) BinarySegment() []byte { return f.segment2 }

// String describes the font.
func (f *Type1Font) String() string {
	var charStrings strings.Builder
	charStrings.WriteByte('{')
	for i, name := range f.charstringNames {
		if i > 0 {
			charStrings.WriteString(", ")
		}
		fmt.Fprintf(&charStrings, "%s=%v", name, f.charstrings[name])
	}
	charStrings.WriteByte('}')
	return fmt.Sprintf(
		"org.apache.fontbox.type1.Type1Font[fontName=%s, fullName=%s, encoding=%v, charStringsDict=%s]",
		f.fontName, f.fullName, f.encoding, charStrings.String())
}

// copyNumbers returns a copy of a number list, standing in for Java's
// Collections.unmodifiableList.
func copyNumbers(numbers []any) []any {
	out := make([]any, len(numbers))
	copy(out, numbers)
	return out
}

// numberFloat is Java's Number.floatValue over the two number types the parser
// produces.
func numberFloat(entry any) float32 {
	switch v := entry.(type) {
	case int:
		return float32(v)
	case float32:
		return v
	}
	panic("type1: not a number")
}

// numbersString renders a number list the way Java's List.toString does.
func numbersString(numbers []any) string {
	parts := make([]string, len(numbers))
	for i, n := range numbers {
		switch v := n.(type) {
		case int:
			parts[i] = fmt.Sprint(v)
		case float32:
			parts[i] = javaFloatString(v)
		}
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// javaFloatString renders a float32 the way Java's Float.toString does, which
// always leaves a decimal point in.
func javaFloatString(value float32) string {
	s := fmt.Sprintf("%v", value)
	if !strings.ContainsAny(s, ".eE") {
		s += ".0"
	}
	return s
}
