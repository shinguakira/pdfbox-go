package ttf

import (
	"fmt"
	"time"
)

// HeaderTable is the head table: the global metrics of the font.
//
// Port of org.apache.fontbox.ttf.HeaderTable.
type HeaderTable struct {
	Table

	version            float32
	fontRevision       float32
	checkSumAdjustment int64
	magicNumber        int64
	flags              int
	unitsPerEm         int
	created            time.Time
	modified           time.Time
	xMin               int16
	yMin               int16
	xMax               int16
	yMax               int16
	macStyle           int
	lowestRecPPEM      int
	fontDirectionHint  int16
	indexToLocFormat   int16
	glyphDataFormat    int16
}

// The two forms the loca table takes, named by the head table.
const (
	ShortOffsets = 0
	LongOffsets  = 1
)

var _ TableReader = (*HeaderTable)(nil)

// Read reads the head table.
func (t *HeaderTable) Read(ttf *TrueTypeFont, data DataStream) error {
	r := newReader(data)
	t.version = r.fixed()
	t.fontRevision = r.fixed()
	t.checkSumAdjustment = r.unsignedInt()
	t.magicNumber = r.unsignedInt()
	t.flags = r.unsignedShort()
	t.unitsPerEm = r.unsignedShort()
	t.created = r.date()
	t.modified = r.date()
	t.xMin = r.signedShort()
	t.yMin = r.signedShort()
	t.xMax = r.signedShort()
	t.yMax = r.signedShort()
	t.macStyle = r.unsignedShort()
	t.lowestRecPPEM = r.unsignedShort()
	t.fontDirectionHint = r.signedShort()
	t.indexToLocFormat = r.signedShort()
	t.glyphDataFormat = r.signedShort()
	if r.err != nil {
		return r.err
	}
	t.SetInitialized(true)
	return nil
}

// Version returns the version of the table.
func (t *HeaderTable) Version() float32 { return t.version }

// FontRevision returns the revision of the font.
func (t *HeaderTable) FontRevision() float32 { return t.fontRevision }

// CheckSumAdjustment returns the checksum adjustment.
func (t *HeaderTable) CheckSumAdjustment() int64 { return t.checkSumAdjustment }

// MagicNumber returns the magic number, which is 0x5F0F3CF5 in a valid font.
func (t *HeaderTable) MagicNumber() int64 { return t.magicNumber }

// Flags returns the header flags.
func (t *HeaderTable) Flags() int { return t.flags }

// UnitsPerEm returns the size of the em square the glyphs are drawn in.
func (t *HeaderTable) UnitsPerEm() int { return t.unitsPerEm }

// Created returns when the font was made.
func (t *HeaderTable) Created() time.Time { return t.created }

// Modified returns when the font was last changed.
func (t *HeaderTable) Modified() time.Time { return t.modified }

// XMin returns the left edge of the bounding box of all glyphs.
func (t *HeaderTable) XMin() int16 { return t.xMin }

// YMin returns the bottom edge of the bounding box of all glyphs.
func (t *HeaderTable) YMin() int16 { return t.yMin }

// XMax returns the right edge of the bounding box of all glyphs.
func (t *HeaderTable) XMax() int16 { return t.xMax }

// YMax returns the top edge of the bounding box of all glyphs.
func (t *HeaderTable) YMax() int16 { return t.yMax }

// MacStyle returns the Mac style bits.
func (t *HeaderTable) MacStyle() int { return t.macStyle }

// LowestRecPPEM returns the smallest size the font is readable at.
func (t *HeaderTable) LowestRecPPEM() int { return t.lowestRecPPEM }

// FontDirectionHint returns the direction hint.
func (t *HeaderTable) FontDirectionHint() int16 { return t.fontDirectionHint }

// IndexToLocFormat returns which of the two forms the loca table takes.
func (t *HeaderTable) IndexToLocFormat() int16 { return t.indexToLocFormat }

// GlyphDataFormat returns the format of the glyph data.
func (t *HeaderTable) GlyphDataFormat() int16 { return t.glyphDataFormat }

// MaximumProfileTable is the maxp table: how much of everything the font has.
//
// Port of org.apache.fontbox.ttf.MaximumProfileTable.
type MaximumProfileTable struct {
	Table

	version               float32
	numGlyphs             int
	maxPoints             int
	maxContours           int
	maxCompositePoints    int
	maxCompositeContours  int
	maxZones              int
	maxTwilightPoints     int
	maxStorage            int
	maxFunctionDefs       int
	maxInstructionDefs    int
	maxStackElements      int
	maxSizeOfInstructions int
	maxComponentElements  int
	maxComponentDepth     int
}

var _ TableReader = (*MaximumProfileTable)(nil)

// Read reads the maxp table. Version 0.5 carries only the glyph count.
func (t *MaximumProfileTable) Read(ttf *TrueTypeFont, data DataStream) error {
	r := newReader(data)
	t.version = r.fixed()
	t.numGlyphs = r.unsignedShort()
	if t.version >= 1.0 {
		t.maxPoints = r.unsignedShort()
		t.maxContours = r.unsignedShort()
		t.maxCompositePoints = r.unsignedShort()
		t.maxCompositeContours = r.unsignedShort()
		t.maxZones = r.unsignedShort()
		t.maxTwilightPoints = r.unsignedShort()
		t.maxStorage = r.unsignedShort()
		t.maxFunctionDefs = r.unsignedShort()
		t.maxInstructionDefs = r.unsignedShort()
		t.maxStackElements = r.unsignedShort()
		t.maxSizeOfInstructions = r.unsignedShort()
		t.maxComponentElements = r.unsignedShort()
		t.maxComponentDepth = r.unsignedShort()
		if t.maxComponentDepth == 0 {
			t.maxComponentDepth = 1
		}
	}
	if r.err != nil {
		return r.err
	}
	t.SetInitialized(true)
	return nil
}

// Version returns the version of the table.
func (t *MaximumProfileTable) Version() float32 { return t.version }

// NumGlyphs returns how many glyphs the font has.
func (t *MaximumProfileTable) NumGlyphs() int { return t.numGlyphs }

// MaxPoints returns the most points in a simple glyph.
func (t *MaximumProfileTable) MaxPoints() int { return t.maxPoints }

// MaxContours returns the most contours in a simple glyph.
func (t *MaximumProfileTable) MaxContours() int { return t.maxContours }

// MaxCompositePoints returns the most points in a composite glyph.
func (t *MaximumProfileTable) MaxCompositePoints() int { return t.maxCompositePoints }

// MaxCompositeContours returns the most contours in a composite glyph.
func (t *MaximumProfileTable) MaxCompositeContours() int { return t.maxCompositeContours }

// MaxComponentDepth returns how deeply composite glyphs nest.
func (t *MaximumProfileTable) MaxComponentDepth() int { return t.maxComponentDepth }

// HorizontalHeaderTable is the hhea table: the metrics that apply to every
// glyph set horizontally.
//
// Port of org.apache.fontbox.ttf.HorizontalHeaderTable.
type HorizontalHeaderTable struct {
	Table

	version             float32
	ascender            int16
	descender           int16
	lineGap             int16
	advanceWidthMax     int
	minLeftSideBearing  int16
	minRightSideBearing int16
	xMaxExtent          int16
	caretSlopeRise      int16
	caretSlopeRun       int16
	reserved1           int16
	reserved2           int16
	reserved3           int16
	reserved4           int16
	reserved5           int16
	metricDataFormat    int16
	numberOfHMetrics    int
}

var _ TableReader = (*HorizontalHeaderTable)(nil)

// Read reads the hhea table.
func (t *HorizontalHeaderTable) Read(ttf *TrueTypeFont, data DataStream) error {
	r := newReader(data)
	t.version = r.fixed()
	t.ascender = r.signedShort()
	t.descender = r.signedShort()
	t.lineGap = r.signedShort()
	t.advanceWidthMax = r.unsignedShort()
	t.minLeftSideBearing = r.signedShort()
	t.minRightSideBearing = r.signedShort()
	t.xMaxExtent = r.signedShort()
	t.caretSlopeRise = r.signedShort()
	t.caretSlopeRun = r.signedShort()
	t.reserved1 = r.signedShort()
	t.reserved2 = r.signedShort()
	t.reserved3 = r.signedShort()
	t.reserved4 = r.signedShort()
	t.reserved5 = r.signedShort()
	t.metricDataFormat = r.signedShort()
	t.numberOfHMetrics = r.unsignedShort()
	if r.err != nil {
		return r.err
	}
	t.SetInitialized(true)
	return nil
}

// Version returns the version of the table.
func (t *HorizontalHeaderTable) Version() float32 { return t.version }

// Ascender returns how far above the baseline the font reaches.
func (t *HorizontalHeaderTable) Ascender() int16 { return t.ascender }

// Descender returns how far below the baseline the font reaches.
func (t *HorizontalHeaderTable) Descender() int16 { return t.descender }

// LineGap returns the gap between lines.
func (t *HorizontalHeaderTable) LineGap() int16 { return t.lineGap }

// AdvanceWidthMax returns the widest advance in the font.
func (t *HorizontalHeaderTable) AdvanceWidthMax() int { return t.advanceWidthMax }

// NumberOfHMetrics returns how many glyphs the hmtx table carries a width for.
func (t *HorizontalHeaderTable) NumberOfHMetrics() int { return t.numberOfHMetrics }

// HorizontalMetricsTable is the hmtx table: the advance width of every glyph.
//
// Port of org.apache.fontbox.ttf.HorizontalMetricsTable.
type HorizontalMetricsTable struct {
	Table

	advanceWidth                 []int
	leftSideBearing              []int16
	nonHorizontalLeftSideBearing []int16
	numHMetrics                  int
}

var _ TableReader = (*HorizontalMetricsTable)(nil)

// Read reads the hmtx table. The last glyphs of a monospaced font may carry no
// width of their own, and share the last one.
func (t *HorizontalMetricsTable) Read(ttf *TrueTypeFont, data DataStream) error {
	hHeader, err := ttf.HorizontalHeader()
	if err != nil {
		return err
	}
	if hHeader == nil {
		return fmt.Errorf("ttf: could not get hmtx table")
	}
	t.numHMetrics = hHeader.NumberOfHMetrics()
	numGlyphs, err := ttf.NumberOfGlyphs()
	if err != nil {
		return err
	}

	r := newReader(data)
	bytesRead := int64(0)
	t.advanceWidth = make([]int, t.numHMetrics)
	t.leftSideBearing = make([]int16, t.numHMetrics)
	for i := 0; i < t.numHMetrics; i++ {
		t.advanceWidth[i] = r.unsignedShort()
		t.leftSideBearing[i] = r.signedShort()
		bytesRead += 4
	}
	if r.err != nil {
		return r.err
	}

	numberNonHorizontal := numGlyphs - t.numHMetrics
	if numberNonHorizontal < 0 {
		numberNonHorizontal = numGlyphs
	}
	t.nonHorizontalLeftSideBearing = make([]int16, numberNonHorizontal)
	if bytesRead < t.Length() {
		for i := 0; i < numberNonHorizontal; i++ {
			if bytesRead < t.Length() {
				t.nonHorizontalLeftSideBearing[i] = r.signedShort()
				bytesRead += 2
			}
		}
	}
	if r.err != nil {
		return r.err
	}
	t.SetInitialized(true)
	return nil
}

// AdvanceWidth returns how far the pen moves after the given glyph.
func (t *HorizontalMetricsTable) AdvanceWidth(gid int) int {
	if len(t.advanceWidth) == 0 {
		return 250
	}
	if gid < t.numHMetrics {
		return t.advanceWidth[gid]
	}
	// monospaced fonts may not have a width for every glyph
	// the last one is for subsequent glyphs
	return t.advanceWidth[len(t.advanceWidth)-1]
}

// LeftSideBearing returns the left side bearing of the given glyph.
func (t *HorizontalMetricsTable) LeftSideBearing(gid int) int16 {
	if len(t.leftSideBearing) == 0 {
		return 0
	}
	if gid < t.numHMetrics {
		return t.leftSideBearing[gid]
	}
	index := gid - t.numHMetrics
	if index >= len(t.nonHorizontalLeftSideBearing) {
		return 0
	}
	return t.nonHorizontalLeftSideBearing[index]
}

// IndexToLocationTable is the loca table: where each glyph starts in glyf.
//
// Port of org.apache.fontbox.ttf.IndexToLocationTable.
type IndexToLocationTable struct {
	Table

	offsets []int64
}

var _ TableReader = (*IndexToLocationTable)(nil)

// Read reads the loca table, in whichever of the two forms the head table says.
func (t *IndexToLocationTable) Read(ttf *TrueTypeFont, data DataStream) error {
	head, err := ttf.Header()
	if err != nil {
		return err
	}
	if head == nil {
		return fmt.Errorf("ttf: could not get head table")
	}
	numGlyphs, err := ttf.NumberOfGlyphs()
	if err != nil {
		return err
	}

	r := newReader(data)
	t.offsets = make([]int64, numGlyphs+1)
	for i := 0; i < numGlyphs+1; i++ {
		switch head.IndexToLocFormat() {
		case ShortOffsets:
			t.offsets[i] = int64(r.unsignedShort()) * 2
		case LongOffsets:
			t.offsets[i] = r.unsignedInt()
		default:
			return fmt.Errorf("ttf: TTF.loca unknown offset format: %d", head.IndexToLocFormat())
		}
	}
	if r.err != nil {
		return r.err
	}
	if numGlyphs == 1 && t.offsets[0] == 0 && t.offsets[1] == 0 {
		// PDFBOX-5794 empty glyph
		return fmt.Errorf("ttf: the font has no glyphs")
	}
	t.SetInitialized(true)
	return nil
}

// Offsets returns where each glyph starts in the glyf table, with one extra
// entry marking the end of the last.
func (t *IndexToLocationTable) Offsets() []int64 { return t.offsets }

// ReadHeaders reads just the macStyle out of the head table.
func (t *HeaderTable) ReadHeaders(ttf *TrueTypeFont, data DataStream, outHeaders *FontHeaders) error {
	// 44 == 4 + 4 + 4 + 4 + 2 + 2 + 2*8 + 4*2, see Read
	if err := data.SeekTo(data.CurrentPosition() + 44); err != nil {
		return err
	}
	r := newReader(data)
	t.macStyle = r.unsignedShort()
	if r.err != nil {
		return r.err
	}
	macStyle := t.macStyle
	outHeaders.setHeaderMacStyle(&macStyle)
	return nil
}

// MaxZones returns the number of zones the font's instructions use.
func (t *MaximumProfileTable) MaxZones() int { return t.maxZones }

// MaxTwilightPoints returns how many points the twilight zone holds.
func (t *MaximumProfileTable) MaxTwilightPoints() int { return t.maxTwilightPoints }

// MaxStorage returns how many storage locations the instructions use.
func (t *MaximumProfileTable) MaxStorage() int { return t.maxStorage }

// MaxFunctionDefs returns how many function definitions the font has.
func (t *MaximumProfileTable) MaxFunctionDefs() int { return t.maxFunctionDefs }

// MaxInstructionDefs returns how many instruction definitions the font has.
func (t *MaximumProfileTable) MaxInstructionDefs() int { return t.maxInstructionDefs }

// MaxStackElements returns how deep the instruction stack goes.
func (t *MaximumProfileTable) MaxStackElements() int { return t.maxStackElements }

// MaxSizeOfInstructions returns the size of the largest glyph's instructions.
func (t *MaximumProfileTable) MaxSizeOfInstructions() int { return t.maxSizeOfInstructions }

// MaxComponentElements returns how many components a composite glyph refers to.
func (t *MaximumProfileTable) MaxComponentElements() int { return t.maxComponentElements }

// MinLeftSideBearing returns the smallest left side bearing of the font.
func (t *HorizontalHeaderTable) MinLeftSideBearing() int16 { return t.minLeftSideBearing }

// MinRightSideBearing returns the smallest right side bearing of the font.
func (t *HorizontalHeaderTable) MinRightSideBearing() int16 { return t.minRightSideBearing }

// XMaxExtent returns the largest horizontal extent of the font.
func (t *HorizontalHeaderTable) XMaxExtent() int16 { return t.xMaxExtent }

// CaretSlopeRise returns the rise of the caret slope.
func (t *HorizontalHeaderTable) CaretSlopeRise() int16 { return t.caretSlopeRise }

// CaretSlopeRun returns the run of the caret slope.
func (t *HorizontalHeaderTable) CaretSlopeRun() int16 { return t.caretSlopeRun }

// Reserved1 returns the first reserved field, which holds the caret offset.
func (t *HorizontalHeaderTable) Reserved1() int16 { return t.reserved1 }

// Reserved2 returns the second reserved field.
func (t *HorizontalHeaderTable) Reserved2() int16 { return t.reserved2 }

// Reserved3 returns the third reserved field.
func (t *HorizontalHeaderTable) Reserved3() int16 { return t.reserved3 }

// Reserved4 returns the fourth reserved field.
func (t *HorizontalHeaderTable) Reserved4() int16 { return t.reserved4 }

// Reserved5 returns the fifth reserved field.
func (t *HorizontalHeaderTable) Reserved5() int16 { return t.reserved5 }

// MetricDataFormat returns the format of the horizontal metrics.
func (t *HorizontalHeaderTable) MetricDataFormat() int16 { return t.metricDataFormat }
