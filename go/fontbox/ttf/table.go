package ttf

// The four-character tags of the tables this port reads.
const (
	CmapTag              = "cmap"
	GlyphTag             = "glyf"
	HeaderTag            = "head"
	HorizontalHeaderTag  = "hhea"
	HorizontalMetricsTag = "hmtx"
	IndexToLocationTag   = "loca"
	MaximumProfileTag    = "maxp"
	NamingTag            = "name"
	OS2WindowsMetricsTag = "OS/2"
	PostScriptTag        = "post"
)

// Table is one table of a TrueType font.
//
// Port of org.apache.fontbox.ttf.TTFTable. Java has the concrete tables extend
// it and override read; the port keeps the shared fields in this struct, which
// each table embeds, and puts read behind an interface.
type Table struct {
	tag         string
	checkSum    int64
	offset      int64
	length      int64
	initialized bool
}

// TableReader is a table that knows how to read itself.
//
// Java's TTFTable.read is an empty method the concrete tables override; the
// port makes it an interface, so that a table with nothing to read simply does
// not implement it.
type TableReader interface {
	// Read reads the table from the stream, which is already positioned at its
	// offset.
	Read(ttf *TrueTypeFont, data DataStream) error
}

// CheckSum returns the checksum of the table.
func (t *Table) CheckSum() int64 { return t.checkSum }

// SetCheckSum sets the checksum of the table.
func (t *Table) SetCheckSum(checkSum int64) { t.checkSum = checkSum }

// Length returns the length of the table in bytes.
func (t *Table) Length() int64 { return t.length }

// SetLength sets the length of the table.
func (t *Table) SetLength(length int64) { t.length = length }

// Offset returns where the table starts in the font.
func (t *Table) Offset() int64 { return t.offset }

// SetOffset sets where the table starts in the font.
func (t *Table) SetOffset(offset int64) { t.offset = offset }

// Tag returns the four-character tag naming the table.
func (t *Table) Tag() string { return t.tag }

// SetTag sets the tag naming the table.
func (t *Table) SetTag(tag string) { t.tag = tag }

// Initialized reports whether the table has been read.
func (t *Table) Initialized() bool { return t.initialized }

// SetInitialized records that the table has been read.
func (t *Table) SetInitialized(initialized bool) { t.initialized = initialized }

// base returns the shared part of the table, so that the font can reach it
// through the interface every table satisfies.
func (t *Table) base() *Table { return t }

// tableBase is what every table offers, whether or not it reads anything.
type tableBase interface {
	base() *Table
}

// UnknownTable is a table this port does not read, kept for its place in the
// directory and its bytes.
//
// Port of the anonymous TTFTable that TTFParser.readTable returns for a tag it
// does not recognise.
type UnknownTable struct {
	Table
}
