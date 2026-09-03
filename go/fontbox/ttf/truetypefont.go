package ttf

import (
	"fmt"
	"io"
	"sync"
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

	lockReadTable sync.Mutex
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
