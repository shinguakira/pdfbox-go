package ttf

import (
	"errors"
	"math"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/fontbox/cff"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// The tags of the OpenType tables. CFFTag is declared beside the other table
// tags in ttfparser.go.
const (
	// CFF2Tag identifies the version 2 PostScript font program table, whose
	// outlines this library does not read.
	CFF2Tag = "CFF2"

	// OTLTag identifies the OpenType Layout table.
	OTLTag = "JSTF"
)

// CFFTable is a PostScript font program (compact font format).
//
// Port of org.apache.fontbox.ttf.CFFTable.
type CFFTable struct {
	Table

	cffFont cff.CFFFont
}

var _ TableReader = (*CFFTable)(nil)

// Read reads the required data from the stream.
func (t *CFFTable) Read(ttf *TrueTypeFont, data DataStream) error {
	bytes := make([]byte, int(t.Length()))
	if _, err := data.ReadInto(bytes, 0, len(bytes)); err != nil {
		return err
	}

	fonts, err := cff.NewCFFParser().ParseBytes(bytes, &cffByteSource{ttf: ttf})
	if err != nil {
		return err
	}
	if len(fonts) == 0 {
		// Java indexes the list and throws IndexOutOfBounds on an empty parse.
		return errors.New("ttf: no CFF font in the CFF table")
	}
	t.cffFont = fonts[0]

	t.SetInitialized(true)
	return nil
}

// Font returns the CFF font, which is a compact representation of a PostScript
// Type 1, or CIDFont.
func (t *CFFTable) Font() cff.CFFFont { return t.cffFont }

// cffByteSource allows bytes to be re-read later by the CFF parser.
type cffByteSource struct {
	ttf *TrueTypeFont
}

func (s *cffByteSource) Bytes() ([]byte, error) {
	return s.ttf.TableBytes(s.ttf.TableMap()[CFFTag])
}

// OTLTable is an OpenType Layout (OTL) table, which uses the OpenType Layout
// Common Table Format.
//
// Port of org.apache.fontbox.ttf.OTLTable. Java's is a stub, a full
// implementation is needed.
type OTLTable struct {
	Table
}

// OpenTypeFont is an OpenType (OTF/TTF) font.
//
// Port of org.apache.fontbox.ttf.OpenTypeFont, which extends TrueTypeFont; the
// port embeds that one.
type OpenTypeFont struct {
	*TrueTypeFont
}

// CFF returns the "CFF" table for this OTF, reporting an error where the
// current font isn't a CFF font.
//
// Java throws UnsupportedOperationException there, which is unchecked; the port
// panics for the same reason it does elsewhere.
func (f *OpenTypeFont) CFF() (*CFFTable, error) {
	if !f.hasPostScriptTag {
		panic("TTF fonts do not have a CFF table")
	}
	table, err := f.table(CFFTag)
	if err != nil {
		return nil, err
	}
	cffTable, _ := table.(*CFFTable)
	return cffTable, nil
}

// Glyph returns the glyph table, which an OTF font with PostScript outlines
// does not have.
func (f *OpenTypeFont) Glyph() (*GlyphTable, error) {
	if f.hasPostScriptTag {
		panic("OTF fonts do not have a glyf table")
	}
	return f.TrueTypeFont.Glyph()
}

// GetPath returns the outline of the named glyph, in glyph space.
func (f *OpenTypeFont) GetPath(name string) (*geom.Path2D, error) {
	if !f.hasPostScriptTag || !f.IsSupportedOTF() {
		return f.TrueTypeFont.GetPath(name)
	}
	gid, err := f.NameToGID(name)
	if err != nil {
		return nil, err
	}
	cffTable, err := f.CFF()
	if err != nil {
		return nil, err
	}
	charString, err := cffTable.Font().GetType2CharString(gid)
	if err != nil {
		return nil, err
	}
	return charString.Path(), nil
}

// IsPostScript reports whether this font is a PostScript outline font.
func (f *OpenTypeFont) IsPostScript() bool {
	if f.hasPostScriptTag {
		return true
	}
	_, hasCFF := f.tables[CFFTag]
	_, hasCFF2 := f.tables[CFF2Tag]
	return hasCFF || hasCFF2
}

// IsSupportedOTF reports whether this font is supported.
//
// There are 3 kind of OpenType fonts, fonts using TrueType outlines, fonts
// using CFF outlines (version 1 and 2). Fonts using CFF outlines version 2
// aren't supported yet.
func (f *OpenTypeFont) IsSupportedOTF() bool {
	_, hasCFF := f.tables[CFFTag]
	_, hasCFF2 := f.tables[CFF2Tag]
	// OTF using CFF2 based outlines aren't yet supported
	return !(f.hasPostScriptTag && !hasCFF && hasCFF2)
}

// HasLayoutTables reports whether this font uses OpenType Layout (Advanced
// Typographic) tables.
func (f *OpenTypeFont) HasLayoutTables() bool {
	for _, tag := range []string{"BASE", "GDEF", "GPOS", GlyphSubstitutionTag, OTLTag} {
		if _, ok := f.tables[tag]; ok {
			return true
		}
	}
	return false
}

// ottoVersion is the bit pattern of the version float an OpenType font with
// PostScript outlines carries, which is the four bytes "OTTO".
const ottoVersion = 0x469EA8A9

// OTFParser is an OpenType font file parser.
//
// Port of org.apache.fontbox.ttf.OTFParser, which extends TTFParser and
// overrides three methods; the port embeds the parser and sets the two hooks it
// carries for them.
type OTFParser struct {
	*Parser
}

// NewOTFParser returns a parser for a stand-alone OpenType font file.
func NewOTFParser() *OTFParser { return NewOTFParserEmbedded(false) }

// NewOTFParserEmbedded returns a parser for an OpenType font, isEmbedded saying
// whether it is embedded in a PDF.
func NewOTFParserEmbedded(isEmbedded bool) *OTFParser {
	p := &OTFParser{Parser: NewParserEmbedded(isEmbedded)}
	p.isOTF = true
	p.readTableHook = otfReadTable
	return p
}

// otfReadTable is the OTFParser override of TTFParser.readTable.
//
// todo: this is a stub, a full implementation is needed
func otfReadTable(tag string) tableBase {
	switch tag {
	case "BASE", "GDEF", "GPOS", GlyphSubstitutionTag, OTLTag:
		return &OTLTable{}
	case CFFTag:
		return &CFFTable{}
	default:
		// PDFBOX-5344: the base parser keeps a table it does not read
		return &UnknownTable{}
	}
}

// Parse reads an OpenType font from the given source, which it closes.
func (p *OTFParser) Parse(randomAccessRead pdfio.RandomAccessRead) (*OpenTypeFont, error) {
	font, err := p.Parser.Parse(randomAccessRead)
	if err != nil {
		return nil, err
	}
	return &OpenTypeFont{TrueTypeFont: font}, nil
}

// ParseStream reads an OpenType font from an already-open stream, which the
// font takes ownership of.
func (p *OTFParser) ParseStream(raf DataStream) (*OpenTypeFont, error) {
	font, err := p.Parser.ParseStream(raf)
	if err != nil {
		return nil, err
	}
	return &OpenTypeFont{TrueTypeFont: font}, nil
}

// allowCFF reports whether the parser accepts a font with CFF outlines.
//
// Port of OTFParser.allowCFF, which overrides TTFParser.allowCFF. Nothing in
// the Java calls either one; both are ported so that the pair stays visible.
func (p *OTFParser) allowCFF() bool { return true }

// float32Bits is Java's Float.floatToIntBits, which OpenTypeFont.setVersion
// compares against the "OTTO" pattern.
func float32Bits(value float32) uint32 { return math.Float32bits(value) }

// AsOpenType returns the OpenType view of the font where the font came from an
// OTFParser, and nil otherwise.
//
// It stands for Java's `font instanceof OpenTypeFont`: OpenTypeFont carries no
// state of its own beyond the flag on TrueTypeFont, so the view can be built on
// demand.
func (f *TrueTypeFont) AsOpenType() *OpenTypeFont {
	if !f.isOpenType {
		return nil
	}
	return &OpenTypeFont{TrueTypeFont: f}
}
