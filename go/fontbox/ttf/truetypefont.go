package ttf

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/fontbox"
	"github.com/shinguakira/pdfbox-go/go/fontbox/util"
)

// TrueTypeFont is a TrueType font, read from its table directory.
//
// Port of org.apache.fontbox.ttf.TrueTypeFont.
//
// The tables this port reads are the ones text extraction needs: head, maxp,
// hhea, hmtx, loca, name, post, OS/2, cmap and glyf. The rest of the directory
// is kept as UnknownTable, so that a later slice can read one without the file
// having to be walked again. See migration/STATUS.md.
//
// A font is not safe for concurrent use. Java guards the lazy table reads with
// a lock, and the port keeps one for the same reason: reading a table moves the
// shared cursor and puts it back.
type TrueTypeFont struct {
	version        float32
	numberOfGlyphs int
	unitsPerEm     int
	tables         map[string]tableBase
	data           DataStream

	postScriptNames map[string]int

	lockReadTable sync.Mutex
	lockPSNames   sync.Mutex
}

// NewTrueTypeFont returns a font reading from the given stream.
func NewTrueTypeFont(data DataStream) *TrueTypeFont {
	return &TrueTypeFont{
		numberOfGlyphs: -1,
		unitsPerEm:     -1,
		tables:         map[string]tableBase{},
		data:           data,
	}
}

// Close releases the stream the font reads through.
func (f *TrueTypeFont) Close() error { return f.data.Close() }

// Version returns the version of the font.
func (f *TrueTypeFont) Version() float32 { return f.version }

// SetVersion sets the version of the font.
func (f *TrueTypeFont) SetVersion(version float32) { f.version = version }

// AddTable adds a table to the directory.
func (f *TrueTypeFont) AddTable(table tableBase) {
	f.tables[table.base().Tag()] = table
}

// Tables returns every table in the directory.
func (f *TrueTypeFont) Tables() []tableBase {
	out := make([]tableBase, 0, len(f.tables))
	for _, table := range f.tables {
		out = append(out, table)
	}
	return out
}

// TableMap returns the directory keyed by tag.
func (f *TrueTypeFont) TableMap() map[string]tableBase {
	out := make(map[string]tableBase, len(f.tables))
	for tag, table := range f.tables {
		out[tag] = table
	}
	return out
}

// TableBytes returns the raw bytes of a table, leaving the cursor where it
// found it.
func (f *TrueTypeFont) TableBytes(table tableBase) ([]byte, error) {
	f.lockReadTable.Lock()
	defer f.lockReadTable.Unlock()

	entry := table.base()
	currentPosition := f.data.CurrentPosition()
	if err := f.data.SeekTo(entry.Offset()); err != nil {
		return nil, err
	}
	bytes, err := readBytes(f.data, int(entry.Length()))
	if err != nil {
		return nil, err
	}
	if err := f.data.SeekTo(currentPosition); err != nil {
		return nil, err
	}
	return bytes, nil
}

// table returns the named table, reading it if it has not been read.
func (f *TrueTypeFont) table(tag string) (tableBase, error) {
	table, ok := f.tables[tag]
	if !ok {
		return nil, nil
	}
	if !table.base().Initialized() {
		if err := f.ReadTable(table); err != nil {
			return nil, err
		}
	}
	return table, nil
}

// ReadTable reads one table, leaving the cursor where it found it.
//
// Java does not lock here, only in getTableBytes; a table read reaches back
// into the font for the tables it depends on, and a lock here would deadlock on
// that where Java simply re-enters.
func (f *TrueTypeFont) ReadTable(table tableBase) error {
	entry := table.base()
	currentPosition := f.data.CurrentPosition()
	if err := f.data.SeekTo(entry.Offset()); err != nil {
		return err
	}
	if reader, ok := table.(TableReader); ok {
		if err := reader.Read(f, f.data); err != nil {
			return err
		}
	} else {
		// Java's TTFTable.read is an empty method, so a table it does not know
		// is simply marked read.
		entry.SetInitialized(true)
	}
	return f.data.SeekTo(currentPosition)
}

// OriginalData returns a reader over the whole font file.
func (f *TrueTypeFont) OriginalData() (io.Reader, error) { return f.data.OriginalData() }

// OriginalDataSize returns the size of the font file.
func (f *TrueTypeFont) OriginalDataSize() int64 { return f.data.OriginalDataSize() }

// --- the typed table accessors ---

// Header returns the head table, or nil where the font has none.
func (f *TrueTypeFont) Header() (*HeaderTable, error) {
	return tableAs[*HeaderTable](f, HeaderTag)
}

// MaximumProfile returns the maxp table, or nil.
func (f *TrueTypeFont) MaximumProfile() (*MaximumProfileTable, error) {
	return tableAs[*MaximumProfileTable](f, MaximumProfileTag)
}

// HorizontalHeader returns the hhea table, or nil.
func (f *TrueTypeFont) HorizontalHeader() (*HorizontalHeaderTable, error) {
	return tableAs[*HorizontalHeaderTable](f, HorizontalHeaderTag)
}

// HorizontalMetrics returns the hmtx table, or nil.
func (f *TrueTypeFont) HorizontalMetrics() (*HorizontalMetricsTable, error) {
	return tableAs[*HorizontalMetricsTable](f, HorizontalMetricsTag)
}

// IndexToLocation returns the loca table, or nil.
func (f *TrueTypeFont) IndexToLocation() (*IndexToLocationTable, error) {
	return tableAs[*IndexToLocationTable](f, IndexToLocationTag)
}

// Naming returns the name table, or nil.
func (f *TrueTypeFont) Naming() (*NamingTable, error) {
	return tableAs[*NamingTable](f, NamingTag)
}

// PostScript returns the post table, or nil.
func (f *TrueTypeFont) PostScript() (*PostScriptTable, error) {
	return tableAs[*PostScriptTable](f, PostScriptTag)
}

// OS2Windows returns the OS/2 table, or nil.
func (f *TrueTypeFont) OS2Windows() (*OS2WindowsMetricsTable, error) {
	return tableAs[*OS2WindowsMetricsTable](f, OS2WindowsMetricsTag)
}

// Cmap returns the cmap table, or nil.
func (f *TrueTypeFont) Cmap() (*CmapTable, error) {
	return tableAs[*CmapTable](f, CmapTag)
}

// Glyph returns the glyf table, or nil.
func (f *TrueTypeFont) Glyph() (*GlyphTable, error) {
	return tableAs[*GlyphTable](f, GlyphTag)
}

// tableAs reads the named table and returns it as its concrete type, or nil
// where the font has no such table.
//
// Java casts the result of getTable; Go needs the assertion written out, and a
// generic function saves writing it ten times.
func tableAs[T tableBase](f *TrueTypeFont, tag string) (T, error) {
	var zero T
	table, err := f.table(tag)
	if err != nil || table == nil {
		return zero, err
	}
	typed, ok := table.(T)
	if !ok {
		return zero, fmt.Errorf("ttf: table '%s' is a %T", tag, table)
	}
	return typed, nil
}

// NumberOfGlyphs returns how many glyphs the font has.
func (f *TrueTypeFont) NumberOfGlyphs() (int, error) {
	if f.numberOfGlyphs == -1 {
		maximumProfile, err := f.MaximumProfile()
		if err != nil {
			return 0, err
		}
		if maximumProfile != nil {
			f.numberOfGlyphs = maximumProfile.NumGlyphs()
		} else {
			// this should never happen
			f.numberOfGlyphs = 0
		}
	}
	return f.numberOfGlyphs, nil
}

// UnitsPerEm returns the size of the em square the glyphs are drawn in.
func (f *TrueTypeFont) UnitsPerEm() (int, error) {
	if f.unitsPerEm == -1 {
		header, err := f.Header()
		if err != nil {
			return 0, err
		}
		if header != nil {
			f.unitsPerEm = header.UnitsPerEm()
		} else {
			// this should never happen
			f.unitsPerEm = 0
		}
	}
	return f.unitsPerEm, nil
}

// AdvanceWidth returns how far the pen moves after the given glyph.
func (f *TrueTypeFont) AdvanceWidth(gid int) (int, error) {
	hmtx, err := f.HorizontalMetrics()
	if err != nil {
		return 0, err
	}
	if hmtx != nil {
		return hmtx.AdvanceWidth(gid), nil
	}
	// this should never happen
	return 250, nil
}

// Name returns the PostScript name of the font.
func (f *TrueTypeFont) Name() (string, error) {
	namingTable, err := f.Naming()
	if err != nil || namingTable == nil {
		return "", err
	}
	return namingTable.PostScriptName(), nil
}

// --- what FontBoxFont asks of a font ---

// postScriptNames indexes the glyph names of the post table by name, reading
// the table the first time it is asked for.
func (f *TrueTypeFont) readPostScriptNames() error {
	f.lockPSNames.Lock()
	defer f.lockPSNames.Unlock()
	if f.postScriptNames != nil {
		return nil
	}
	post, err := f.PostScript()
	if err != nil {
		return err
	}
	var names []string
	if post != nil {
		names = post.GlyphNames()
	}
	psnames := make(map[string]int, len(names))
	for i, name := range names {
		psnames[name] = i
	}
	f.postScriptNames = psnames
	return nil
}

// NameToGID returns the glyph the given name stands for, or zero where the font
// has no such glyph.
func (f *TrueTypeFont) NameToGID(name string) (int, error) {
	// look up in 'post' table
	if err := f.readPostScriptNames(); err != nil {
		return 0, err
	}
	if f.postScriptNames != nil {
		if gid, ok := f.postScriptNames[name]; ok {
			maximumProfile, err := f.MaximumProfile()
			if err != nil {
				return 0, err
			}
			if gid > 0 && gid < maximumProfile.NumGlyphs() {
				return gid, nil
			}
		}
	}

	// look up in 'cmap'
	uni := parseUniName(name)
	if uni > -1 {
		cmap, err := f.UnicodeCmapLookup(false)
		if err != nil {
			return 0, err
		}
		return cmap.GetGlyphID(uni), nil
	}

	// PDFBOX-5604: assume gnnnnn is a gid
	if gidName.MatchString(name) {
		gid, err := strconv.Atoi(name[1:])
		if err != nil {
			// Java's Integer.parseInt throws where the digits overrun an int,
			// and nothing catches it.
			panic(err)
		}
		return gid, nil
	}
	return 0, nil
}

// gidName matches the gnnnnn form of a glyph name.
var gidName = regexp.MustCompile(`^g\d+$`)

// parseUniName returns the code point a uniXXXX name stands for, or -1 where
// the name is not one.
func parseUniName(name string) int {
	if strings.HasPrefix(name, "uni") && len(name) == 7 {
		nameLength := len(name)
		var uniStr strings.Builder
		for chPos := 3; chPos+4 <= nameLength; chPos += 4 {
			codePoint, err := strconv.ParseInt(name[chPos:chPos+4], 16, 32)
			if err != nil {
				return -1
			}
			if codePoint <= 0xD7FF || codePoint >= 0xE000 { // disallowed code area
				uniStr.WriteRune(rune(codePoint))
			}
		}
		unicode := uniStr.String()
		if unicode == "" {
			return -1
		}
		return int([]rune(unicode)[0])
	}
	return -1
}

// UnicodeCmapLookup returns the Unicode subtable of the cmap table.
//
// Where isStrict is false and the font has no Unicode subtable at all, the
// result is a nil subtable, which is what Java returns; calling through it
// panics, as dereferencing Java's null does.
func (f *TrueTypeFont) UnicodeCmapLookup(isStrict bool) (CmapLookup, error) {
	// Java wraps the subtable in a SubstitutingCmapLookup where GSUB features
	// are enabled; the GSUB table is read by a later slice, and no feature can
	// be enabled until it is. See migration/STATUS.md.
	return f.unicodeCmapImpl(isStrict)
}

// unicodeCmapImpl picks the best Unicode subtable the font carries.
func (f *TrueTypeFont) unicodeCmapImpl(isStrict bool) (*CmapSubtable, error) {
	cmapTable, err := f.Cmap()
	if err != nil {
		return nil, err
	}
	if cmapTable == nil {
		if isStrict {
			name, err := f.Name()
			if err != nil {
				return nil, err
			}
			return nil, fmt.Errorf("ttf: The TrueType font %s does not contain a 'cmap' table", name)
		}
		return nil, nil
	}

	cmap := cmapTable.GetSubtable(CmapPlatformUnicode, CmapEncodingUnicode20Full)
	if cmap == nil {
		cmap = cmapTable.GetSubtable(CmapPlatformWindows, EncodingWinUnicodeFull)
	}
	if cmap == nil {
		cmap = cmapTable.GetSubtable(CmapPlatformUnicode, CmapEncodingUnicode20BMP)
	}
	if cmap == nil {
		cmap = cmapTable.GetSubtable(CmapPlatformWindows, EncodingWinUnicodeBMP)
	}
	if cmap == nil {
		// Microsoft's "Recommendations for OpenType Fonts" says that "Symbol"
		// encoding actually means "Unicode, non-standard character set"
		cmap = cmapTable.GetSubtable(CmapPlatformWindows, EncodingWinSymbol)
	}
	if cmap == nil {
		// PDFBOX-6015
		cmap = cmapTable.GetSubtable(CmapPlatformUnicode, CmapEncodingUnicode11)
	}
	if cmap == nil {
		if isStrict {
			return nil, fmt.Errorf("ttf: The TrueType font does not contain a Unicode cmap")
		} else if len(cmapTable.Cmaps()) > 0 {
			// fallback to the first cmap (may not be Unicode, so may produce
			// poor results)
			cmap = cmapTable.Cmaps()[0]
		}
	}
	return cmap, nil
}

// GetPath returns the outline of the named glyph.
//
// Rendering a glyph to a path is left to a later slice, which ports
// GlyphRenderer; see migration/STATUS.md.
func (f *TrueTypeFont) GetPath(name string) (*geom.Path2D, error) {
	return nil, fmt.Errorf("ttf: glyph outlines are not ported yet")
}

// GetWidth returns how far the pen moves after the named glyph.
func (f *TrueTypeFont) GetWidth(name string) (float32, error) {
	gid, err := f.NameToGID(name)
	if err != nil {
		return 0, err
	}
	advanceWidth, err := f.AdvanceWidth(gid)
	if err != nil {
		return 0, err
	}
	return float32(advanceWidth), nil
}

// HasGlyph reports whether the font has the named glyph.
func (f *TrueTypeFont) HasGlyph(name string) (bool, error) {
	gid, err := f.NameToGID(name)
	if err != nil {
		return false, err
	}
	return gid != 0, nil
}

// FontBBox returns the box every glyph of the font fits in, scaled to a
// thousandth of an em.
func (f *TrueTypeFont) FontBBox() (*util.BoundingBox, error) {
	headerTable, err := f.Header()
	if err != nil {
		return nil, err
	}
	xMin := headerTable.XMin()
	xMax := headerTable.XMax()
	yMin := headerTable.YMin()
	yMax := headerTable.YMax()
	unitsPerEm, err := f.UnitsPerEm()
	if err != nil {
		return nil, err
	}
	scale := 1000.0 / float32(unitsPerEm)
	return util.NewBoundingBoxOf(float32(xMin)*scale, float32(yMin)*scale,
		float32(xMax)*scale, float32(yMax)*scale), nil
}

// FontMatrix returns the transform from glyph space to text space.
//
// Java returns a List<Number> holding a mix of boxed floats and ints, and every
// caller reads it with floatValue.
func (f *TrueTypeFont) FontMatrix() ([]float32, error) {
	unitsPerEm, err := f.UnitsPerEm()
	if err != nil {
		return nil, err
	}
	scale := 1000.0 / float32(unitsPerEm)
	return []float32{0.001 * scale, 0, 0, 0.001 * scale, 0, 0}, nil
}

// String returns the PostScript name of the font.
func (f *TrueTypeFont) String() string {
	name, err := f.Name()
	if err != nil {
		// Java's toString swallows the failure and reports it in place of the
		// name.
		return err.Error()
	}
	return name
}

// A TrueType font is one of the fonts this library reads.
var (
	_ fontbox.FontBoxFont = (*TrueTypeFont)(nil)
)
