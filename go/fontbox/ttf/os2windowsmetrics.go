package ttf

// The weight classes the OS/2 table can name.
const (
	WeightClassThin       = 100
	WeightClassUltraLight = 200
	WeightClassLight      = 300
	WeightClassNormal     = 400
	WeightClassMedium     = 500
	WeightClassSemiBold   = 600
	WeightClassBold       = 700
	WeightClassExtraBold  = 800
	WeightClassBlack      = 900
)

// The width classes the OS/2 table can name.
const (
	WidthClassUltraCondensed = 1
	WidthClassExtraCondensed = 2
	WidthClassCondensed      = 3
	WidthClassSemiCondensed  = 4
	WidthClassMedium         = 5
	WidthClassSemiExpanded   = 6
	WidthClassExpanded       = 7
	WidthClassExtraExpanded  = 8
	WidthClassUltraExpanded  = 9
)

// The family classes the OS/2 table can name.
const (
	FamilyClassNoClassification   = 0
	FamilyClassOldstyleSerifs     = 1
	FamilyClassTransitionalSerifs = 2
	FamilyClassModernSerifs       = 3
	FamilyClassClaredonSerifs     = 4
	FamilyClassSlabSerifs         = 5
	FamilyClassFreeformSerifs     = 7
	FamilyClassSansSerif          = 8
	FamilyClassOrnamentals        = 9
	FamilyClassScripts            = 10
	FamilyClassSymbolic           = 12
)

// The embedding restrictions the OS/2 table can carry.
const (
	FsTypeRestricted      int16 = 0x0002
	FsTypePreviewAndPrint int16 = 0x0004
	FsTypeEditible        int16 = 0x0008
	FsTypeNoSubsetting    int16 = 0x0100
	FsTypeBitmapOnly      int16 = 0x0200
)

// OS2WindowsMetricsTable is the OS/2 table: the metrics a Windows rasteriser
// wants, and the embedding restrictions the foundry set.
//
// Port of org.apache.fontbox.ttf.OS2WindowsMetricsTable.
type OS2WindowsMetricsTable struct {
	Table

	version            int
	averageCharWidth   int16
	weightClass        int
	widthClass         int
	fsType             int16
	subscriptXSize     int16
	subscriptYSize     int16
	subscriptXOffset   int16
	subscriptYOffset   int16
	superscriptXSize   int16
	superscriptYSize   int16
	superscriptXOffset int16
	superscriptYOffset int16
	strikeoutSize      int16
	strikeoutPosition  int16
	familyClass        int
	panose             []byte
	unicodeRange1      int64
	unicodeRange2      int64
	unicodeRange3      int64
	unicodeRange4      int64
	achVendID          string
	fsSelection        int
	firstCharIndex     int
	lastCharIndex      int
	typoAscender       int
	typoDescender      int
	typoLineGap        int
	winAscent          int
	winDescent         int
	codePageRange1     int64
	codePageRange2     int64
	sxHeight           int
	sCapHeight         int
	usDefaultChar      int
	usBreakChar        int
	usMaxContext       int
}

// NewOS2WindowsMetricsTable returns a table with the defaults Java gives its
// fields, which a short table leaves in place.
func NewOS2WindowsMetricsTable() *OS2WindowsMetricsTable {
	return &OS2WindowsMetricsTable{
		panose:    make([]byte, 10),
		achVendID: "XXXX",
	}
}

var _ TableReader = (*OS2WindowsMetricsTable)(nil)

// Read reads the OS/2 table.
//
// The table grew over three versions, and plenty of fonts carry a shorter one
// than they claim; running off the end is not an error, it stops the read and
// lowers the version to what was actually there.
func (t *OS2WindowsMetricsTable) Read(ttf *TrueTypeFont, data DataStream) error {

	r := newReader(data)
	t.version = r.unsignedShort()
	t.averageCharWidth = r.signedShort()
	t.weightClass = r.unsignedShort()
	t.widthClass = r.unsignedShort()
	t.fsType = r.signedShort()
	t.subscriptXSize = r.signedShort()
	t.subscriptYSize = r.signedShort()
	t.subscriptXOffset = r.signedShort()
	t.subscriptYOffset = r.signedShort()
	t.superscriptXSize = r.signedShort()
	t.superscriptYSize = r.signedShort()
	t.superscriptXOffset = r.signedShort()
	t.superscriptYOffset = r.signedShort()
	t.strikeoutSize = r.signedShort()
	t.strikeoutPosition = r.signedShort()
	t.familyClass = int(r.signedShort())
	if panose := r.bytes(10); panose != nil {
		t.panose = panose
	}
	t.unicodeRange1 = r.unsignedInt()
	t.unicodeRange2 = r.unsignedInt()
	t.unicodeRange3 = r.unsignedInt()
	t.unicodeRange4 = r.unsignedInt()
	t.achVendID = r.str(4)
	t.fsSelection = r.unsignedShort()
	t.firstCharIndex = r.unsignedShort()
	t.lastCharIndex = r.unsignedShort()
	if r.err != nil {
		return r.err
	}

	typoAscender := r.signedShort()
	typoDescender := r.signedShort()
	typoLineGap := r.signedShort()
	winAscent := r.unsignedShort()
	winDescent := r.unsignedShort()
	if r.err != nil {
		if isEOF(r.err) {
			// EOF, probably some legacy TrueType font
			t.SetInitialized(true)
			return nil
		}
		return r.err
	}
	t.typoAscender = int(typoAscender)
	t.typoDescender = int(typoDescender)
	t.typoLineGap = int(typoLineGap)
	t.winAscent = winAscent
	t.winDescent = winDescent

	if t.version >= 1 {
		codePageRange1 := r.unsignedInt()
		codePageRange2 := r.unsignedInt()
		if r.err != nil {
			if isEOF(r.err) {
				// Could not read all expected parts of version >= 1, setting
				// version to 0.
				t.version = 0
				t.SetInitialized(true)
				return nil
			}
			return r.err
		}
		t.codePageRange1 = codePageRange1
		t.codePageRange2 = codePageRange2
	}

	if t.version >= 2 {
		sxHeight := r.signedShort()
		sCapHeight := r.signedShort()
		usDefaultChar := r.unsignedShort()
		usBreakChar := r.unsignedShort()
		usMaxContext := r.unsignedShort()
		if r.err != nil {
			if isEOF(r.err) {
				// Could not read all expected parts of version >= 2, setting
				// version to 1.
				t.version = 1
				t.SetInitialized(true)
				return nil
			}
			return r.err
		}
		t.sxHeight = int(sxHeight)
		t.sCapHeight = int(sCapHeight)
		t.usDefaultChar = usDefaultChar
		t.usBreakChar = usBreakChar
		t.usMaxContext = usMaxContext
	}

	t.SetInitialized(true)
	return nil
}

// AchVendID returns the four-character identifier of the foundry.
func (t *OS2WindowsMetricsTable) AchVendID() string { return t.achVendID }

// AverageCharWidth returns the average width of the glyphs.
func (t *OS2WindowsMetricsTable) AverageCharWidth() int16 { return t.averageCharWidth }

// CodePageRange1 returns the first half of the code page range bits.
func (t *OS2WindowsMetricsTable) CodePageRange1() int64 { return t.codePageRange1 }

// CodePageRange2 returns the second half of the code page range bits.
func (t *OS2WindowsMetricsTable) CodePageRange2() int64 { return t.codePageRange2 }

// FamilyClass returns which family the font belongs to.
func (t *OS2WindowsMetricsTable) FamilyClass() int { return t.familyClass }

// FirstCharIndex returns the lowest character the font covers.
func (t *OS2WindowsMetricsTable) FirstCharIndex() int { return t.firstCharIndex }

// FsSelection returns the selection bits.
func (t *OS2WindowsMetricsTable) FsSelection() int { return t.fsSelection }

// FsType returns what the foundry allows the font to be embedded for.
func (t *OS2WindowsMetricsTable) FsType() int16 { return t.fsType }

// LastCharIndex returns the highest character the font covers.
func (t *OS2WindowsMetricsTable) LastCharIndex() int { return t.lastCharIndex }

// Panose returns the ten PANOSE classification bytes.
func (t *OS2WindowsMetricsTable) Panose() []byte { return t.panose }

// StrikeoutPosition returns where the strikeout sits.
func (t *OS2WindowsMetricsTable) StrikeoutPosition() int16 { return t.strikeoutPosition }

// StrikeoutSize returns how thick the strikeout is.
func (t *OS2WindowsMetricsTable) StrikeoutSize() int16 { return t.strikeoutSize }

// SubscriptXOffset returns how far right a subscript sits.
func (t *OS2WindowsMetricsTable) SubscriptXOffset() int16 { return t.subscriptXOffset }

// SubscriptXSize returns how wide a subscript is drawn.
func (t *OS2WindowsMetricsTable) SubscriptXSize() int16 { return t.subscriptXSize }

// SubscriptYOffset returns how far down a subscript sits.
func (t *OS2WindowsMetricsTable) SubscriptYOffset() int16 { return t.subscriptYOffset }

// SubscriptYSize returns how tall a subscript is drawn.
func (t *OS2WindowsMetricsTable) SubscriptYSize() int16 { return t.subscriptYSize }

// SuperscriptXOffset returns how far right a superscript sits.
func (t *OS2WindowsMetricsTable) SuperscriptXOffset() int16 { return t.superscriptXOffset }

// SuperscriptXSize returns how wide a superscript is drawn.
func (t *OS2WindowsMetricsTable) SuperscriptXSize() int16 { return t.superscriptXSize }

// SuperscriptYOffset returns how far up a superscript sits.
func (t *OS2WindowsMetricsTable) SuperscriptYOffset() int16 { return t.superscriptYOffset }

// SuperscriptYSize returns how tall a superscript is drawn.
func (t *OS2WindowsMetricsTable) SuperscriptYSize() int16 { return t.superscriptYSize }

// UnicodeRange1 returns the first quarter of the Unicode range bits.
func (t *OS2WindowsMetricsTable) UnicodeRange1() int64 { return t.unicodeRange1 }

// UnicodeRange2 returns the second quarter of the Unicode range bits.
func (t *OS2WindowsMetricsTable) UnicodeRange2() int64 { return t.unicodeRange2 }

// UnicodeRange3 returns the third quarter of the Unicode range bits.
func (t *OS2WindowsMetricsTable) UnicodeRange3() int64 { return t.unicodeRange3 }

// UnicodeRange4 returns the last quarter of the Unicode range bits.
func (t *OS2WindowsMetricsTable) UnicodeRange4() int64 { return t.unicodeRange4 }

// Version returns which of the OS/2 table versions this is.
func (t *OS2WindowsMetricsTable) Version() int { return t.version }

// WeightClass returns how heavy the font is.
func (t *OS2WindowsMetricsTable) WeightClass() int { return t.weightClass }

// WidthClass returns how wide the font is.
func (t *OS2WindowsMetricsTable) WidthClass() int { return t.widthClass }

// TypoAscender returns the typographic ascender.
func (t *OS2WindowsMetricsTable) TypoAscender() int { return t.typoAscender }

// TypoDescender returns the typographic descender.
func (t *OS2WindowsMetricsTable) TypoDescender() int { return t.typoDescender }

// TypoLineGap returns the typographic line gap.
func (t *OS2WindowsMetricsTable) TypoLineGap() int { return t.typoLineGap }

// WinAscent returns the ascender a Windows rasteriser uses.
func (t *OS2WindowsMetricsTable) WinAscent() int { return t.winAscent }

// WinDescent returns the descender a Windows rasteriser uses.
func (t *OS2WindowsMetricsTable) WinDescent() int { return t.winDescent }

// Height returns the height of a lowercase x.
func (t *OS2WindowsMetricsTable) Height() int { return t.sxHeight }

// CapHeight returns the height of a capital letter.
func (t *OS2WindowsMetricsTable) CapHeight() int { return t.sCapHeight }

// DefaultChar returns the character shown for one the font does not cover.
func (t *OS2WindowsMetricsTable) DefaultChar() int { return t.usDefaultChar }

// BreakChar returns the character a line breaks at.
func (t *OS2WindowsMetricsTable) BreakChar() int { return t.usBreakChar }

// MaxContext returns the longest context any lookup of the font needs.
func (t *OS2WindowsMetricsTable) MaxContext() int { return t.usMaxContext }
