package ttf

// bytesGCID is how much of the "gcid" table a non-OTF font's headers keep.
const bytesGCID = 142

// FontHeaders is used both as a marker (to skip unused data) and as a storage
// for collected data, to improve the performance of
// FileSystemFontProvider.scanFonts.
//
// Tables it needs: NamingTable, HeaderTable, OS2WindowsMetricsTable, CFFTable
// (for OTF) and "gcid" (for non-OTF).
//
// Port of org.apache.fontbox.ttf.FontHeaders.
type FontHeaders struct {
	err            string
	name           string
	headerMacStyle *int
	os2Windows     *OS2WindowsMetricsTable
	fontFamily     string
	fontSubFamily  string
	nonOtfGcid142  []byte

	isOTFAndPostScript bool
	otfRegistry        string
	otfOrdering        string
	otfSupplement      int
}

var _ cffFontHeadersSink = (*FontHeaders)(nil)

// cffFontHeadersSink is what cff.Parser.ParseFirstSubFontROS writes into. The
// interface itself is declared in cff, and this assertion is what keeps the two
// in step without ttf importing cff for the sake of it.
type cffFontHeadersSink interface {
	SetError(message string)
	SetOtfROS(registry, ordering string, supplement int)
}

// NewFontHeaders returns an empty set of headers.
func NewFontHeaders() *FontHeaders { return &FontHeaders{} }

// Error returns why the headers could not be read, or the empty string.
func (h *FontHeaders) Error() string { return h.err }

// Name returns the PostScript name of the font.
func (h *FontHeaders) Name() string { return h.name }

// HeaderMacStyle returns the macStyle of the head table, nil where the font has
// no HeaderTable.
func (h *FontHeaders) HeaderMacStyle() *int { return h.headerMacStyle }

// OS2Windows returns the OS/2 table.
func (h *FontHeaders) OS2Windows() *OS2WindowsMetricsTable { return h.os2Windows }

// FontFamily returns the family name, which is only collected when tracing.
func (h *FontHeaders) FontFamily() string { return h.fontFamily }

// FontSubFamily returns the subfamily name, which is only collected when
// tracing.
func (h *FontHeaders) FontSubFamily() string { return h.fontSubFamily }

// IsOpenTypePostScript reports whether the font is an OpenType one with
// PostScript outlines.
func (h *FontHeaders) IsOpenTypePostScript() bool { return h.isOTFAndPostScript }

// NonOtfTableGCID142 returns the first 142 bytes of the "gcid" table.
func (h *FontHeaders) NonOtfTableGCID142() []byte { return h.nonOtfGcid142 }

// OtfRegistry returns the registry of the character collection.
func (h *FontHeaders) OtfRegistry() string { return h.otfRegistry }

// OtfOrdering returns the ordering of the character collection.
func (h *FontHeaders) OtfOrdering() string { return h.otfOrdering }

// OtfSupplement returns the supplement of the character collection.
func (h *FontHeaders) OtfSupplement() int { return h.otfSupplement }

// SetError records why the headers could not be read.
func (h *FontHeaders) SetError(exception string) { h.err = exception }

func (h *FontHeaders) setName(name string) { h.name = name }

func (h *FontHeaders) setHeaderMacStyle(headerMacStyle *int) { h.headerMacStyle = headerMacStyle }

func (h *FontHeaders) setOs2Windows(os2Windows *OS2WindowsMetricsTable) {
	h.os2Windows = os2Windows
}

func (h *FontHeaders) setFontFamily(fontFamily, fontSubFamily string) {
	h.fontFamily = fontFamily
	h.fontSubFamily = fontSubFamily
}

func (h *FontHeaders) setNonOtfGcid142(nonOtfGcid142 []byte) { h.nonOtfGcid142 = nonOtfGcid142 }

func (h *FontHeaders) setIsOTFAndPostScript(isOTFAndPostScript bool) {
	h.isOTFAndPostScript = isOTFAndPostScript
}

// SetOtfROS records the Registry, Ordering and Supplement of the first CFF
// subfont.
//
// Java marks it public because CFFParser is in a different package, which is
// also why the port declares the interface it satisfies over in cff.
func (h *FontHeaders) SetOtfROS(otfRegistry, otfOrdering string, otfSupplement int) {
	h.otfRegistry = otfRegistry
	h.otfOrdering = otfOrdering
	h.otfSupplement = otfSupplement
}

// HeaderReader is a table that knows how to read just its headers.
//
// Java's TTFTable.readHeaders is an empty method the three tables that have
// headers override; the port makes it an interface, so that a table with no
// headers simply does not implement it.
type HeaderReader interface {
	// ReadHeaders reads the header fields from the stream, which is already
	// positioned at the table's offset.
	ReadHeaders(ttf *TrueTypeFont, data DataStream, outHeaders *FontHeaders) error
}
