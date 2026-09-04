package cff

import (
	"fmt"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/fontbox"
	"github.com/shinguakira/pdfbox-go/go/fontbox/encoding"
	"github.com/shinguakira/pdfbox-go/go/fontbox/util"
)

// ByteSource is a source of bytes, used to re-read the CFF data in the future.
//
// Port of the org.apache.fontbox.cff.CFFParser.ByteSource interface.
type ByteSource interface {
	// Bytes returns the CFF data.
	Bytes() ([]byte, error)
}

// CFFFont is an Adobe Compact Font Format (CFF) font. Thread safe.
//
// Port of the abstract org.apache.fontbox.cff.CFFFont, whose one abstract
// method is GetType2CharString; the shared state and behaviour live in the
// embedded cffFontBase.
type CFFFont interface {
	fontbox.FontBoxFont

	// TopDict returns the top dictionary.
	TopDict() map[string]any

	// Charset returns the CFFCharset of the font.
	Charset() CFFCharset

	// CharStringBytes returns the character strings dictionary, as a list of
	// byte arrays. For expert users only.
	CharStringBytes() [][]byte

	// Data returns the CFF data.
	Data() ([]byte, error)

	// NumCharStrings returns the number of charstrings in the font.
	NumCharStrings() int

	// GlobalSubrIndex returns the list containing the global subroutines.
	GlobalSubrIndex() [][]byte

	// GetType2CharString returns the Type 2 charstring for the given CID for a
	// CIDFont, or GID for a Type 1 font.
	GetType2CharString(cidOrGid int) (*Type2CharString, error)
}

// cffFontBase is the state and behaviour every CFF font has.
type cffFontBase struct {
	// self is the font this base belongs to, for the one method Java leaves
	// abstract.
	self CFFFont

	fontName string
	charset  CFFCharset
	source   ByteSource

	// topDict is Java's LinkedHashMap, so the insertion order is kept for the
	// sake of toString.
	topDict      map[string]any
	topDictOrder []string

	charStrings     [][]byte
	globalSubrIndex [][]byte
}

func newCFFFontBase() cffFontBase {
	return cffFontBase{topDict: map[string]any{}}
}

// Name returns the name of the font.
//
// Java's CFFFont.getName does not throw; the FontBoxFont interface the port
// declares carries an error for the fonts that do, and this one never sets it.
func (f *cffFontBase) Name() (string, error) { return f.fontName, nil }

// setName sets the name of the font.
func (f *cffFontBase) setName(name string) { f.fontName = name }

// AddValueToTopDict adds the given key/value pair to the top dictionary.
func (f *cffFontBase) AddValueToTopDict(name string, value any) {
	if value != nil {
		if _, seen := f.topDict[name]; !seen {
			f.topDictOrder = append(f.topDictOrder, name)
		}
		f.topDict[name] = value
	}
}

// TopDict returns the top dictionary.
func (f *cffFontBase) TopDict() map[string]any { return f.topDict }

// FontMatrix returns the FontMatrix.
//
// Java hands back the List<Number> the parser stored; the port's FontBoxFont
// declares the six floats the callers all want, so the list is narrowed here.
func (f *cffFontBase) FontMatrix() ([]float32, error) {
	numbers, ok := f.topDict["FontMatrix"].([]any)
	if !ok {
		return nil, nil
	}
	matrix := make([]float32, len(numbers))
	for i, n := range numbers {
		matrix[i] = numberFloat(n)
	}
	return matrix, nil
}

// FontBBox returns the FontBBox, reporting an error if there are less than 4
// numbers.
func (f *cffFontBase) FontBBox() (*util.BoundingBox, error) {
	numbers, _ := f.topDict["FontBBox"].([]any)
	if len(numbers) < 4 {
		return nil, fmt.Errorf("FontBBox must have 4 numbers, but is %s", sequenceString(numbers))
	}
	return util.NewBoundingBoxOf(
		numberFloat(numbers[0]), numberFloat(numbers[1]),
		numberFloat(numbers[2]), numberFloat(numbers[3])), nil
}

// Charset returns the CFFCharset of the font.
func (f *cffFontBase) Charset() CFFCharset { return f.charset }

// setCharset sets the CFFCharset of the font.
func (f *cffFontBase) setCharset(charset CFFCharset) { f.charset = charset }

// CharStringBytes returns the character strings dictionary, as a list of byte
// arrays. For expert users only.
func (f *cffFontBase) CharStringBytes() [][]byte { return f.charStrings }

// setData sets a byte source to re-read the CFF data in the future.
func (f *cffFontBase) setData(source ByteSource) { f.source = source }

// Data returns the CFF data.
func (f *cffFontBase) Data() ([]byte, error) { return f.source.Bytes() }

// NumCharStrings returns the number of charstrings in the font.
func (f *cffFontBase) NumCharStrings() int { return len(f.charStrings) }

// setGlobalSubrIndex sets the global subroutine index data.
func (f *cffFontBase) setGlobalSubrIndex(globalSubrIndexValue [][]byte) {
	f.globalSubrIndex = globalSubrIndexValue
}

// GlobalSubrIndex returns the list containing the global subroutines.
func (f *cffFontBase) GlobalSubrIndex() [][]byte { return f.globalSubrIndex }

// dictString renders a dictionary the way Java's LinkedHashMap.toString does,
// in insertion order.
func dictString(dict map[string]any, order []string) string {
	var sb strings.Builder
	sb.WriteByte('{')
	for i, key := range order {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(key)
		sb.WriteByte('=')
		sb.WriteString(dictValueString(dict[key]))
	}
	sb.WriteByte('}')
	return sb.String()
}

// dictValueString renders one dictionary value, which may be a number, a
// string, a list of numbers, or an index of byte arrays.
func dictValueString(value any) string {
	switch v := value.(type) {
	case []any:
		return sequenceString(v)
	case [][]byte:
		parts := make([]string, len(v))
		for i, b := range v {
			parts[i] = byteArrayString(b)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case string:
		return v
	}
	return entryString(value)
}

// byteArrayString renders a byte array the way Java's Arrays.toString does,
// with the bytes signed.
func byteArrayString(bytes []byte) string {
	parts := make([]string, len(bytes))
	for i, b := range bytes {
		parts[i] = fmt.Sprint(int8(b))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// describe renders the font the way Java's CFFFont.toString does, className
// standing for getClass().getSimpleName().
func (f *cffFontBase) describe(className string) string {
	charStrings := make([]string, len(f.charStrings))
	for i, b := range f.charStrings {
		charStrings[i] = byteArrayString(b)
	}
	return fmt.Sprintf("%s[name=%s, topDict=%s, charset=%v, charStrings=[%s]]",
		className, f.fontName, dictString(f.topDict, f.topDictOrder), f.charset,
		strings.Join(charStrings, ", "))
}

// CFFType1Font is a Type 1-equivalent font program represented in a CFF file.
// Thread safe.
//
// Port of org.apache.fontbox.cff.CFFType1Font.
type CFFType1Font struct {
	cffFontBase

	privateDict      map[string]any
	privateDictOrder []string
	encoding         *CFFEncoding

	charStringCache map[int]*Type2CharString

	charStringParser *Type2CharStringParser

	defaultWidthX  int
	nominalWidthX  int
	localSubrIndex [][]byte
	localSubrRead  bool
}

// minInt is Java's Integer.MIN_VALUE, the sentinel the two width fields start
// at.
const minInt = -2147483648

var (
	_ CFFFont               = (*CFFType1Font)(nil)
	_ fontbox.EncodedFont   = (*CFFType1Font)(nil)
	_ Type1CharStringReader = (*CFFType1Font)(nil)
)

// NewCFFType1Font returns an empty Type 1-equivalent CFF font, which the parser
// then fills in.
func NewCFFType1Font() *CFFType1Font {
	f := &CFFType1Font{
		cffFontBase:     newCFFFontBase(),
		privateDict:     map[string]any{},
		charStringCache: map[int]*Type2CharString{},
		defaultWidthX:   minInt,
		nominalWidthX:   minInt,
	}
	f.self = f
	return f
}

// GetPath returns the outline of the named glyph.
func (f *CFFType1Font) GetPath(name string) (*geom.Path2D, error) {
	charString, err := f.Type1CharString(name)
	if err != nil {
		return nil, err
	}
	return charString.Path(), nil
}

// GetWidth returns how far the pen moves after the named glyph.
func (f *CFFType1Font) GetWidth(name string) (float32, error) {
	charString, err := f.Type1CharString(name)
	if err != nil {
		return 0, err
	}
	return float32(charString.Width()), nil
}

// HasGlyph reports whether the font has the named glyph.
func (f *CFFType1Font) HasGlyph(name string) (bool, error) {
	return f.NameToGID(name) != 0, nil
}

// Type1CharString returns the Type 1 charstring for the given PostScript glyph
// name.
//
// This also makes the font a Type1CharStringReader: Java hands the charstrings
// a private inner class rather than the font itself, because only CFFType1Font
// can expose this publicly, as CIDFonts only support it for legacy 'seac'
// commands. In Go the method is on the font and CFFCIDFont has its own.
func (f *CFFType1Font) Type1CharString(name string) (*Type1CharString, error) {
	// lookup via charset
	gid := f.NameToGID(name)

	// lookup in CharStrings INDEX
	charString, err := f.getType2CharString(gid, name)
	if err != nil {
		return nil, err
	}
	return charString.Type1CharString, nil
}

// NameToGID returns the GID for the given PostScript glyph name.
func (f *CFFType1Font) NameToGID(name string) int {
	// some fonts have glyphs beyond their encoding, so we look up by charset SID
	sid := f.Charset().SID(name)
	return f.Charset().GIDForSID(sid)
}

// GetType2CharString returns the Type 1 charstring for the given GID.
func (f *CFFType1Font) GetType2CharString(gid int) (*Type2CharString, error) {
	name := "GID+" + fmt.Sprint(gid) // for debugging only
	return f.getType2CharString(gid, name)
}

// getType2CharString returns the Type 2 charstring for the given GID, with name
// for debugging.
func (f *CFFType1Font) getType2CharString(gid int, name string) (*Type2CharString, error) {
	if type2, ok := f.charStringCache[gid]; ok {
		return type2, nil
	}
	var bytes []byte
	if gid < len(f.charStrings) {
		bytes = f.charStrings[gid]
	}
	if bytes == nil {
		// .notdef
		bytes = f.charStrings[0]
	}
	type2seq, err := f.parser().Parse(bytes, f.globalSubrIndex, f.getLocalSubrIndex())
	if err != nil {
		return nil, err
	}
	fontName, _ := f.Name()
	type2 := NewType2CharString(f, fontName, name, gid, type2seq,
		f.getDefaultWidthX(), f.getNominalWidthX())
	f.charStringCache[gid] = type2
	return type2, nil
}

func (f *CFFType1Font) parser() *Type2CharStringParser {
	if f.charStringParser == nil {
		fontName, _ := f.Name()
		f.charStringParser = NewType2CharStringParser(fontName)
	}
	return f.charStringParser
}

// PrivateDict returns the private dictionary.
func (f *CFFType1Font) PrivateDict() map[string]any { return f.privateDict }

// addToPrivateDict adds the given key/value pair to the private dictionary.
func (f *CFFType1Font) addToPrivateDict(name string, value any) {
	if value != nil {
		if _, seen := f.privateDict[name]; !seen {
			f.privateDictOrder = append(f.privateDictOrder, name)
		}
		f.privateDict[name] = value
	}
}

// Encoding returns the encoding built into the font, which is what the
// EncodedFont interface asks for.
//
// Java declares getEncoding on CFFType1Font with the narrower CFFEncoding
// return type, a covariant override Go has no equivalent for; CFFEncoding below
// is that method.
func (f *CFFType1Font) Encoding() *encoding.Encoding {
	if f.encoding == nil {
		return nil
	}
	return f.encoding.Encoding
}

// CFFEncoding returns the CFFEncoding of the font.
func (f *CFFType1Font) CFFEncoding() *CFFEncoding { return f.encoding }

// setEncoding sets the CFFEncoding of the font.
func (f *CFFType1Font) setEncoding(encoding *CFFEncoding) { f.encoding = encoding }

func (f *CFFType1Font) getLocalSubrIndex() [][]byte {
	if !f.localSubrRead {
		f.localSubrIndex, _ = f.privateDict["Subrs"].([][]byte)
		f.localSubrRead = true
	}
	return f.localSubrIndex
}

// getProperty is a helper for looking up keys/values.
func (f *CFFType1Font) getProperty(name string) any {
	if topDictValue, ok := f.topDict[name]; ok && topDictValue != nil {
		return topDictValue
	}
	return f.privateDict[name]
}

func (f *CFFType1Font) getDefaultWidthX() int {
	if f.defaultWidthX == minInt {
		f.defaultWidthX = 1000
		if num := f.getProperty("defaultWidthX"); isNumber(num) {
			f.defaultWidthX = numberInt(num)
		}
	}
	return f.defaultWidthX
}

func (f *CFFType1Font) getNominalWidthX() int {
	if f.nominalWidthX == minInt {
		f.nominalWidthX = 0
		if num := f.getProperty("nominalWidthX"); isNumber(num) {
			f.nominalWidthX = numberInt(num)
		}
	}
	return f.nominalWidthX
}

// String describes the font.
func (f *CFFType1Font) String() string { return f.describe("CFFType1Font") }

// FDSelect returns the Font DICT index for a GID.
//
// Port of the org.apache.fontbox.cff.FDSelect interface.
type FDSelect interface {
	// FDIndex returns the font dictionary index of the given GID.
	FDIndex(gid int) int
}

// CFFCIDFont is a Type 0 CIDFont represented in a CFF file. Thread safe.
//
// Port of org.apache.fontbox.cff.CFFCIDFont.
type CFFCIDFont struct {
	cffFontBase

	registry   string
	ordering   string
	supplement int

	fontDictionaries    []map[string]any
	privateDictionaries []map[string]any
	fdSelect            FDSelect

	charStringCache  map[int]*CIDKeyedType2CharString
	charStringParser *Type2CharStringParser
}

var (
	_ CFFFont               = (*CFFCIDFont)(nil)
	_ Type1CharStringReader = (*CFFCIDFont)(nil)
)

// NewCFFCIDFont returns an empty CID-keyed CFF font, which the parser then
// fills in.
func NewCFFCIDFont() *CFFCIDFont {
	f := &CFFCIDFont{
		cffFontBase:     newCFFFontBase(),
		charStringCache: map[int]*CIDKeyedType2CharString{},
	}
	f.self = f
	return f
}

// Registry returns the registry value.
func (f *CFFCIDFont) Registry() string { return f.registry }

// setRegistry sets the registry value.
func (f *CFFCIDFont) setRegistry(registry string) { f.registry = registry }

// Ordering returns the ordering value.
func (f *CFFCIDFont) Ordering() string { return f.ordering }

// setOrdering sets the ordering value.
func (f *CFFCIDFont) setOrdering(ordering string) { f.ordering = ordering }

// Supplement returns the supplement value.
func (f *CFFCIDFont) Supplement() int { return f.supplement }

// setSupplement sets the supplement value.
func (f *CFFCIDFont) setSupplement(supplement int) { f.supplement = supplement }

// FontDicts returns the font dictionaries.
func (f *CFFCIDFont) FontDicts() []map[string]any { return f.fontDictionaries }

// setFontDict sets the font dictionaries.
func (f *CFFCIDFont) setFontDict(fontDict []map[string]any) { f.fontDictionaries = fontDict }

// PrivDicts returns the private dictionaries.
func (f *CFFCIDFont) PrivDicts() []map[string]any { return f.privateDictionaries }

// setPrivDict sets the private dictionaries.
func (f *CFFCIDFont) setPrivDict(privDict []map[string]any) { f.privateDictionaries = privDict }

// FdSelect returns the fdSelect value.
func (f *CFFCIDFont) FdSelect() FDSelect { return f.fdSelect }

// setFdSelect sets the fdSelect value.
func (f *CFFCIDFont) setFdSelect(fdSelect FDSelect) { f.fdSelect = fdSelect }

// getDefaultWidthX returns the defaultWidthX for the given GID.
func (f *CFFCIDFont) getDefaultWidthX(gid int) int {
	fdArrayIndex := f.fdSelect.FDIndex(gid)
	if fdArrayIndex == -1 || fdArrayIndex >= len(f.privateDictionaries) {
		return 1000
	}
	privDictValue := f.privateDictionaries[fdArrayIndex]["defaultWidthX"]
	if isNumber(privDictValue) {
		return numberInt(privDictValue)
	}
	return 1000
}

// getNominalWidthX returns the nominalWidthX for the given GID.
func (f *CFFCIDFont) getNominalWidthX(gid int) int {
	fdArrayIndex := f.fdSelect.FDIndex(gid)
	if fdArrayIndex == -1 || fdArrayIndex >= len(f.privateDictionaries) {
		return 0
	}
	privDictValue := f.privateDictionaries[fdArrayIndex]["nominalWidthX"]
	if isNumber(privDictValue) {
		return numberInt(privDictValue)
	}
	return 0
}

// getLocalSubrIndex returns the LocalSubrIndex for the given GID.
func (f *CFFCIDFont) getLocalSubrIndex(gid int) [][]byte {
	fdArrayIndex := f.fdSelect.FDIndex(gid)
	if fdArrayIndex == -1 || fdArrayIndex >= len(f.privateDictionaries) {
		return nil
	}
	privDict := f.privateDictionaries[fdArrayIndex]
	subrs, _ := privDict["Subrs"].([][]byte)
	return subrs
}

// CIDKeyedCharString returns the Type 2 charstring for the given CID.
func (f *CFFCIDFont) CIDKeyedCharString(cid int) (*CIDKeyedType2CharString, error) {
	if type2, ok := f.charStringCache[cid]; ok {
		return type2, nil
	}
	gid := f.Charset().GIDForCID(cid)

	bytes := f.charStrings[gid]
	if bytes == nil {
		bytes = f.charStrings[0] // .notdef
	}
	type2seq, err := f.parser().Parse(bytes, f.globalSubrIndex, f.getLocalSubrIndex(gid))
	if err != nil {
		return nil, err
	}
	fontName, _ := f.Name()
	type2 := NewCIDKeyedType2CharString(f, fontName, cid, gid, type2seq,
		f.getDefaultWidthX(gid), f.getNominalWidthX(gid))
	f.charStringCache[cid] = type2
	return type2, nil
}

// GetType2CharString returns the Type 2 charstring for the given CID.
//
// Java's override narrows the return type to CIDKeyedType2CharString, which Go
// cannot do; CIDKeyedCharString above is that method, and this one is what the
// CFFFont interface asks for.
func (f *CFFCIDFont) GetType2CharString(cid int) (*Type2CharString, error) {
	type2, err := f.CIDKeyedCharString(cid)
	if err != nil {
		return nil, err
	}
	return type2.Type2CharString, nil
}

func (f *CFFCIDFont) parser() *Type2CharStringParser {
	if f.charStringParser == nil {
		fontName, _ := f.Name()
		f.charStringParser = NewType2CharStringParser(fontName)
	}
	return f.charStringParser
}

// GetPath returns the outline of the glyph the selector names.
func (f *CFFCIDFont) GetPath(selector string) (*geom.Path2D, error) {
	cid, err := selectorToCID(selector)
	if err != nil {
		return nil, err
	}
	charString, err := f.GetType2CharString(cid)
	if err != nil {
		return nil, err
	}
	return charString.Path(), nil
}

// GetWidth returns how far the pen moves after the glyph the selector names.
func (f *CFFCIDFont) GetWidth(selector string) (float32, error) {
	cid, err := selectorToCID(selector)
	if err != nil {
		return 0, err
	}
	charString, err := f.GetType2CharString(cid)
	if err != nil {
		return 0, err
	}
	return float32(charString.Width()), nil
}

// HasGlyph reports whether the font has the glyph the selector names.
func (f *CFFCIDFont) HasGlyph(selector string) (bool, error) {
	cid, err := selectorToCID(selector)
	if err != nil {
		return false, err
	}
	return cid != 0, nil
}

// Type1CharString returns the .notdef charstring, whatever the name.
//
// Java hands the charstrings a private inner class doing exactly this, because
// CIDFonts only support the reader for legacy 'seac' commands.
func (f *CFFCIDFont) Type1CharString(name string) (*Type1CharString, error) {
	charString, err := f.GetType2CharString(0) // .notdef
	if err != nil {
		return nil, err
	}
	return charString.Type1CharString, nil
}

// selectorToCID parses a CID selector of the form \ddddd.
//
// Java throws IllegalArgumentException for a selector without the backslash and
// NumberFormatException for one that is not a number; both are unchecked, so
// the port panics rather than reporting either.
func selectorToCID(selector string) (int, error) {
	if !strings.HasPrefix(selector, "\\") {
		panic("Invalid selector")
	}
	var cid int
	if _, err := fmt.Sscanf(selector[1:], "%d", &cid); err != nil {
		panic("For input string: " + selector[1:])
	}
	return cid, nil
}

// String describes the font.
func (f *CFFCIDFont) String() string { return f.describe("CFFCIDFont") }

// CIDKeyedType2CharString is a CID-Keyed Type 2 CharString.
//
// Port of org.apache.fontbox.cff.CIDKeyedType2CharString.
type CIDKeyedType2CharString struct {
	*Type2CharString

	cid int
}

// NewCIDKeyedType2CharString builds the charstring, font being the parent CFF
// font and sequence the Type 2 char string sequence.
func NewCIDKeyedType2CharString(font Type1CharStringReader, fontName string, cid, gid int,
	sequence []any, defaultWidthX, nomWidthX int) *CIDKeyedType2CharString {
	return &CIDKeyedType2CharString{
		// glyph name is for debugging only
		Type2CharString: NewType2CharString(font, fontName, fmt.Sprintf("%04x", cid), gid,
			sequence, defaultWidthX, nomWidthX),
		cid: cid,
	}
}

// CID returns the CID (character id) of this charstring.
func (c *CIDKeyedType2CharString) CID() int { return c.cid }
