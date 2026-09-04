package ttf

import (
	"fmt"
	"io"

	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// The tags of the remaining tables.
const (
	CFFTag               = "CFF "
	DigitalSignatureTag  = "DSIG"
	KerningTag           = "kern"
	VerticalHeaderTag    = "vhea"
	VerticalMetricsTag   = "vmtx"
	VerticalOriginTag    = "VORG"
	GlyphSubstitutionTag = "GSUB"
)

// Parser reads a TrueType font file.
//
// Port of org.apache.fontbox.ttf.TTFParser.
type Parser struct {
	isEmbedded bool

	// isOTF and readTableHook are what OTFParser overrides. Java overrides
	// newFont and readTable; Go embedding has no virtual dispatch, so the two
	// travel as fields the OTF constructor sets.
	isOTF         bool
	readTableHook func(tag string) tableBase
}

// NewParser returns a parser for a stand-alone font file, which must carry
// every mandatory table.
func NewParser() *Parser { return &Parser{} }

// NewParserEmbedded returns a parser for a font embedded in a PDF, which is
// allowed to leave out the tables the PDF itself supplies.
func NewParserEmbedded(isEmbedded bool) *Parser { return &Parser{isEmbedded: isEmbedded} }

// Parse reads a font from the given source, which it closes.
func (p *Parser) Parse(randomAccessRead pdfio.RandomAccessRead) (*TrueTypeFont, error) {
	dataStream, err := NewRandomAccessReadDataStream(randomAccessRead)
	if err != nil {
		randomAccessRead.Close()
		return nil, err
	}
	return p.parseAndClose(randomAccessRead, dataStream)
}

// ParseEmbedded reads a font embedded in a PDF from the given reader, which it
// closes where it is a Closer.
func (p *Parser) ParseEmbedded(inputStream io.Reader) (*TrueTypeFont, error) {
	p.isEmbedded = true
	source, err := pdfio.NewReadBufferFromReader(inputStream)
	if err != nil {
		return nil, err
	}
	dataStream, err := NewRandomAccessReadDataStream(source)
	if err != nil {
		source.Close()
		return nil, err
	}
	return p.parseAndClose(source, dataStream)
}

// parseAndClose reads the font, closing the input either way and the stream it
// reads through where the read fails.
func (p *Parser) parseAndClose(input io.Closer, dataStream *RandomAccessReadDataStream) (*TrueTypeFont, error) {
	font, err := p.ParseStream(dataStream)
	if err != nil {
		dataStream.Close()
		input.Close()
		return nil, err
	}
	if err := input.Close(); err != nil {
		return nil, err
	}
	return font, nil
}

// ParseStream reads a font from an already-open stream, which the font takes
// ownership of.
func (p *Parser) ParseStream(raf DataStream) (*TrueTypeFont, error) {
	font, err := p.createFontWithTables(raf)
	if err != nil {
		return nil, err
	}
	if err := p.parseTables(font); err != nil {
		return nil, err
	}
	return font, nil
}

// createFontWithTables reads the table directory.
func (p *Parser) createFontWithTables(raf DataStream) (*TrueTypeFont, error) {
	font := p.newFont(raf)
	font.isOpenType = p.isOTF
	r := newReader(raf)
	font.SetVersion(r.fixed())
	numberOfTables := r.unsignedShort()
	_ = r.unsignedShort() // searchRange
	_ = r.unsignedShort() // entrySelector
	_ = r.unsignedShort() // rangeShift
	if r.err != nil {
		return nil, r.err
	}
	for i := 0; i < numberOfTables; i++ {
		table, err := p.readTableDirectory(raf)
		if err != nil {
			return nil, err
		}
		if table != nil {
			if table.base().Offset()+table.base().Length() > font.OriginalDataSize() {
				// Skip a table which goes past the file size.
				continue
			}
			font.AddTable(table)
		}
	}
	return font, nil
}

// newFont returns the font the directory is read into.
//
// Java has OTFParser override this to build an OpenTypeFont; the port wraps the
// font the parse produces instead, which comes to the same thing because
// OpenTypeFont carries no state of its own.
func (p *Parser) newFont(raf DataStream) *TrueTypeFont {
	return NewTrueTypeFont(raf)
}

// parseTables reads every table of the directory and checks that the ones the
// format requires are there.
func (p *Parser) parseTables(font *TrueTypeFont) error {
	for _, table := range font.Tables() {
		if !table.base().Initialized() {
			if err := font.ReadTable(table); err != nil {
				return err
			}
		}
	}

	_, hasCFF := font.tables[CFFTag]
	isOTF := p.isOTF
	isPostScript := hasCFF
	if isOTF {
		isPostScript = (&OpenTypeFont{TrueTypeFont: font}).IsPostScript()
	}

	head, err := font.Header()
	if err != nil {
		return err
	}
	if head == nil {
		return fmt.Errorf("ttf: 'head' table is mandatory")
	}
	hh, err := font.HorizontalHeader()
	if err != nil {
		return err
	}
	if hh == nil {
		return fmt.Errorf("ttf: 'hhea' table is mandatory")
	}
	maxp, err := font.MaximumProfile()
	if err != nil {
		return err
	}
	if maxp == nil {
		return fmt.Errorf("ttf: 'maxp' table is mandatory")
	}
	post, err := font.PostScript()
	if err != nil {
		return err
	}
	if post == nil && !p.isEmbedded {
		return fmt.Errorf("ttf: 'post' table is mandatory")
	}

	if !isPostScript {
		loca, err := font.IndexToLocation()
		if err != nil {
			return err
		}
		if loca == nil {
			return fmt.Errorf("ttf: 'loca' table is mandatory")
		}
		glyf, err := font.Glyph()
		if err != nil {
			return err
		}
		if glyf == nil {
			return fmt.Errorf("ttf: 'glyf' table is mandatory")
		}
	} else if !isOTF {
		return fmt.Errorf("ttf: True Type fonts using CFF outlines are not supported")
	}

	naming, err := font.Naming()
	if err != nil {
		return err
	}
	if naming == nil && !p.isEmbedded {
		return fmt.Errorf("ttf: 'name' table is mandatory")
	}
	hmtx, err := font.HorizontalMetrics()
	if err != nil {
		return err
	}
	if hmtx == nil {
		return fmt.Errorf("ttf: 'hmtx' table is mandatory")
	}
	if !p.isEmbedded {
		cmap, err := font.Cmap()
		if err != nil {
			return err
		}
		if cmap == nil {
			return fmt.Errorf("ttf: 'cmap' table is mandatory")
		}
	}
	return nil
}

// readTableDirectory reads one entry of the table directory, returning nil for
// an entry that carries nothing.
func (p *Parser) readTableDirectory(raf DataStream) (tableBase, error) {
	r := newReader(raf)
	tag := r.str(4)
	if r.err != nil {
		return nil, r.err
	}

	table := p.newTable(tag)
	entry := table.base()
	entry.SetTag(tag)
	entry.SetCheckSum(r.unsignedInt())
	entry.SetOffset(r.unsignedInt())
	entry.SetLength(r.unsignedInt())
	if r.err != nil {
		return nil, r.err
	}

	// skip tables with zero length (except glyf)
	if entry.Length() == 0 && tag != GlyphTag {
		return nil, nil
	}
	return table, nil
}

// newTable returns the table the given tag names.
func (p *Parser) newTable(tag string) tableBase {
	switch tag {
	case CmapTag:
		return &CmapTable{}
	case GlyphTag:
		return &GlyphTable{}
	case HeaderTag:
		return &HeaderTable{}
	case HorizontalHeaderTag:
		return &HorizontalHeaderTable{}
	case HorizontalMetricsTag:
		return &HorizontalMetricsTable{}
	case IndexToLocationTag:
		return &IndexToLocationTable{}
	case MaximumProfileTag:
		return &MaximumProfileTable{}
	case NamingTag:
		return &NamingTable{}
	case OS2WindowsMetricsTag:
		return NewOS2WindowsMetricsTable()
	case PostScriptTag:
		return &PostScriptTable{}
	case DigitalSignatureTag:
		return &DigitalSignatureTable{}
	case KerningTag:
		return &KerningTable{}
	case VerticalHeaderTag:
		return &VerticalHeaderTable{}
	case VerticalMetricsTag:
		return &VerticalMetricsTable{}
	case VerticalOriginTag:
		return &VerticalOriginTable{}
	default:
		return p.readTable(tag)
	}
}

// readTable returns the table a subclass of the parser would read, which for
// this one is always a table it keeps but does not read.
func (p *Parser) readTable(tag string) tableBase {
	if p.readTableHook != nil {
		return p.readTableHook(tag)
	}
	// PDFBOX-5344: the parser of an OpenType font overrides this
	return &UnknownTable{}
}

// allowCFF reports whether the parser accepts a font with CFF outlines.
//
// Nothing in the Java calls it; OTFParser overrides it and both are ported so
// that the pair stays visible.
func (p *Parser) allowCFF() bool { return false }

// parseTableHeaders parses all table headers and checks if all needed tables
// are present, which is based on parseTables.
//
// This method can be optimized further by skipping unused portions inside each
// individual table parser.
//
// Port of org.apache.fontbox.ttf.TTFParser.parseTableHeaders.
func (p *Parser) parseTableHeaders(raf DataStream) (*FontHeaders, error) {
	outHeaders := NewFontHeaders()
	font, err := p.createFontWithTables(raf)
	if err != nil {
		return nil, err
	}
	defer font.Close()

	if err := font.readTableHeaders(NamingTag, outHeaders); err != nil {
		return nil, err
	}
	if err := font.readTableHeaders(HeaderTag, outHeaders); err != nil {
		return nil, err
	}

	// only these 5 are used
	//   sFamilyClass = os2WindowsMetricsTable.getFamilyClass();
	//   usWeightClass = os2WindowsMetricsTable.getWeightClass();
	//   ulCodePageRange1 = (int) os2WindowsMetricsTable.getCodePageRange1();
	//   ulCodePageRange2 = (int) os2WindowsMetricsTable.getCodePageRange2();
	//   panose = os2WindowsMetricsTable.getPanose();
	os2Windows, err := font.OS2Windows()
	if err != nil {
		return nil, err
	}
	outHeaders.setOs2Windows(os2Windows)

	isOTFAndPostScript := false
	otf := font.AsOpenType()
	_, hasCFF := font.tables[CFFTag]
	switch {
	case otf != nil && otf.IsPostScript():
		if !otf.IsSupportedOTF() {
			outHeaders.SetError("OpenType fonts using CFF2 outlines are not supported")
			return outHeaders, nil
		}
		isOTFAndPostScript = true
		if err := font.readTableHeaders(CFFTag, outHeaders); err != nil {
			return nil, err
		}
	case otf == nil && hasCFF:
		outHeaders.SetError("True Type fonts using CFF outlines are not supported")
		return outHeaders, nil
	default:
		if gcid := font.TableMap()["gcid"]; gcid != nil && gcid.base().Length() >= bytesGCID {
			bytes, err := font.TableNBytes(gcid, bytesGCID)
			if err != nil {
				return nil, err
			}
			outHeaders.setNonOtfGcid142(bytes)
		}
	}
	outHeaders.setIsOTFAndPostScript(isOTFAndPostScript)

	// list taken from parseTables(), detect them, but don't spend time parsing
	mandatoryTables := []string{HeaderTag, HorizontalHeaderTag, MaximumProfileTag}
	if !p.isEmbedded {
		// in an embedded font this table is optional
		mandatoryTables = append(mandatoryTables, PostScriptTag)
	}
	if !isOTFAndPostScript {
		mandatoryTables = append(mandatoryTables, IndexToLocationTag, GlyphTag)
	}
	if !p.isEmbedded {
		mandatoryTables = append(mandatoryTables, NamingTag)
	}
	mandatoryTables = append(mandatoryTables, HorizontalMetricsTag)
	if !p.isEmbedded {
		mandatoryTables = append(mandatoryTables, CmapTag)
	}

	for _, tag := range mandatoryTables {
		if _, ok := font.tables[tag]; !ok {
			outHeaders.SetError("'" + tag + "' table is mandatory")
			return outHeaders, nil
		}
	}
	return outHeaders, nil
}
