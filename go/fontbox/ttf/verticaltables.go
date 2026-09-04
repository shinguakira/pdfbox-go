package ttf

import "errors"

// VerticalHeaderTable is a vertical header 'vhea' table in a TrueType or
// OpenType font.
//
// This table is required by the OpenType CJK Font Guidelines for "all OpenType
// fonts that are used for vertical writing".
//
// This table is specified in both the TrueType and OpenType specifications.
//
// Port of org.apache.fontbox.ttf.VerticalHeaderTable.
type VerticalHeaderTable struct {
	Table

	version              float32
	ascender             int16
	descender            int16
	lineGap              int16
	advanceHeightMax     int
	minTopSideBearing    int16
	minBottomSideBearing int16
	yMaxExtent           int16
	caretSlopeRise       int16
	caretSlopeRun        int16
	caretOffset          int16
	reserved1            int16
	reserved2            int16
	reserved3            int16
	reserved4            int16
	metricDataFormat     int16
	numberOfVMetrics     int
}

var _ TableReader = (*VerticalHeaderTable)(nil)

// Read reads the required data from the stream.
func (t *VerticalHeaderTable) Read(ttf *TrueTypeFont, data DataStream) error {
	r := newReader(data)
	t.version = r.fixed()
	t.ascender = r.signedShort()
	t.descender = r.signedShort()
	t.lineGap = r.signedShort()
	t.advanceHeightMax = r.unsignedShort()
	t.minTopSideBearing = r.signedShort()
	t.minBottomSideBearing = r.signedShort()
	t.yMaxExtent = r.signedShort()
	t.caretSlopeRise = r.signedShort()
	t.caretSlopeRun = r.signedShort()
	t.caretOffset = r.signedShort()
	t.reserved1 = r.signedShort()
	t.reserved2 = r.signedShort()
	t.reserved3 = r.signedShort()
	t.reserved4 = r.signedShort()
	t.metricDataFormat = r.signedShort()
	t.numberOfVMetrics = r.unsignedShort()
	if r.err != nil {
		return r.err
	}
	t.SetInitialized(true)
	return nil
}

// AdvanceHeightMax returns the advanceHeightMax.
func (t *VerticalHeaderTable) AdvanceHeightMax() int { return t.advanceHeightMax }

// Ascender returns the ascender.
func (t *VerticalHeaderTable) Ascender() int16 { return t.ascender }

// CaretSlopeRise returns the caretSlopeRise.
func (t *VerticalHeaderTable) CaretSlopeRise() int16 { return t.caretSlopeRise }

// CaretSlopeRun returns the caretSlopeRun.
func (t *VerticalHeaderTable) CaretSlopeRun() int16 { return t.caretSlopeRun }

// CaretOffset returns the caretOffset.
func (t *VerticalHeaderTable) CaretOffset() int16 { return t.caretOffset }

// Descender returns the descender.
func (t *VerticalHeaderTable) Descender() int16 { return t.descender }

// LineGap returns the lineGap.
func (t *VerticalHeaderTable) LineGap() int16 { return t.lineGap }

// MetricDataFormat returns the metricDataFormat.
func (t *VerticalHeaderTable) MetricDataFormat() int16 { return t.metricDataFormat }

// MinTopSideBearing returns the minTopSideBearing.
func (t *VerticalHeaderTable) MinTopSideBearing() int16 { return t.minTopSideBearing }

// MinBottomSideBearing returns the minBottomSideBearing.
func (t *VerticalHeaderTable) MinBottomSideBearing() int16 { return t.minBottomSideBearing }

// NumberOfVMetrics returns the numberOfVMetrics.
func (t *VerticalHeaderTable) NumberOfVMetrics() int { return t.numberOfVMetrics }

// Reserved1 returns the first reserved value.
func (t *VerticalHeaderTable) Reserved1() int16 { return t.reserved1 }

// Reserved2 returns the second reserved value.
func (t *VerticalHeaderTable) Reserved2() int16 { return t.reserved2 }

// Reserved3 returns the third reserved value.
func (t *VerticalHeaderTable) Reserved3() int16 { return t.reserved3 }

// Reserved4 returns the fourth reserved value.
func (t *VerticalHeaderTable) Reserved4() int16 { return t.reserved4 }

// Version returns the version.
func (t *VerticalHeaderTable) Version() float32 { return t.version }

// YMaxExtent returns the yMaxExtent.
func (t *VerticalHeaderTable) YMaxExtent() int16 { return t.yMaxExtent }

// VerticalMetricsTable is a vertical metrics 'vmtx' table in a TrueType or
// OpenType font.
//
// This table is required by the OpenType CJK Font Guidelines for "all OpenType
// fonts that are used for vertical writing".
//
// This table is specified in both the TrueType and OpenType specifications.
//
// Port of org.apache.fontbox.ttf.VerticalMetricsTable.
type VerticalMetricsTable struct {
	Table

	advanceHeight            []int
	topSideBearing           []int16
	additionalTopSideBearing []int16
	numVMetrics              int
}

var _ TableReader = (*VerticalMetricsTable)(nil)

// Read reads the required data from the stream.
func (t *VerticalMetricsTable) Read(ttf *TrueTypeFont, data DataStream) error {
	vHeader, err := ttf.VerticalHeader()
	if err != nil {
		return err
	}
	if vHeader == nil {
		return errors.New("ttf: Could not get vhea table")
	}
	t.numVMetrics = vHeader.NumberOfVMetrics()
	numGlyphs, err := ttf.NumberOfGlyphs()
	if err != nil {
		return err
	}

	r := newReader(data)
	bytesRead := int64(0)
	t.advanceHeight = make([]int, t.numVMetrics)
	t.topSideBearing = make([]int16, t.numVMetrics)
	for i := 0; i < t.numVMetrics; i++ {
		t.advanceHeight[i] = r.unsignedShort()
		t.topSideBearing[i] = r.signedShort()
		bytesRead += 4
	}
	if r.err != nil {
		return r.err
	}

	if bytesRead < t.Length() {
		numberNonVertical := numGlyphs - t.numVMetrics

		// handle bad fonts with too many vmetrics
		if numberNonVertical < 0 {
			numberNonVertical = numGlyphs
		}

		t.additionalTopSideBearing = make([]int16, numberNonVertical)
		for i := 0; i < numberNonVertical; i++ {
			if bytesRead < t.Length() {
				t.additionalTopSideBearing[i] = r.signedShort()
				bytesRead += 2
			}
		}
		if r.err != nil {
			return r.err
		}
	}

	t.SetInitialized(true)
	return nil
}

// TopSideBearing returns the top sidebearing for the given GID.
func (t *VerticalMetricsTable) TopSideBearing(gid int) int {
	if gid < t.numVMetrics {
		return int(t.topSideBearing[gid])
	}
	return int(t.additionalTopSideBearing[gid-t.numVMetrics])
}

// AdvanceHeight returns the advance height for the given GID.
func (t *VerticalMetricsTable) AdvanceHeight(gid int) int {
	if gid < t.numVMetrics {
		return t.advanceHeight[gid]
	}
	// monospaced fonts may not have a height for every glyph
	// the last one is for subsequent glyphs
	return t.advanceHeight[len(t.advanceHeight)-1]
}

// VerticalOriginTable is a vertical origin 'VORG' table in an OpenType font.
//
// The purpose of this table is to improve the efficiency of determining
// vertical origins in CFF fonts where absent this information the bounding box
// would have to be extracted from CFF charstring data.
//
// This table is strongly recommended by the OpenType CJK Font Guidelines for
// "CFF OpenType fonts that are used for vertical writing".
//
// This table is specified only in the OpenType specification (1.3 and later).
//
// Port of org.apache.fontbox.ttf.VerticalOriginTable.
type VerticalOriginTable struct {
	Table

	version            float32
	defaultVertOriginY int16
	origins            map[int]int
}

var _ TableReader = (*VerticalOriginTable)(nil)

// Read reads the required data from the stream.
func (t *VerticalOriginTable) Read(ttf *TrueTypeFont, data DataStream) error {
	r := newReader(data)
	t.version = r.fixed()
	t.defaultVertOriginY = r.signedShort()
	numVertOriginYMetrics := r.unsignedShort()
	if r.err != nil {
		return r.err
	}
	t.origins = make(map[int]int, numVertOriginYMetrics)
	for i := 0; i < numVertOriginYMetrics; i++ {
		g := r.unsignedShort()
		y := r.signedShort()
		if r.err != nil {
			return r.err
		}
		t.origins[g] = int(y)
	}
	t.SetInitialized(true)
	return nil
}

// Version returns the version.
func (t *VerticalOriginTable) Version() float32 { return t.version }

// OriginY returns the y-coordinate of the vertical origin for the given GID if
// known, or the default value if not specified in table data.
func (t *VerticalOriginTable) OriginY(gid int) int {
	if y, ok := t.origins[gid]; ok {
		return y
	}
	return int(t.defaultVertOriginY)
}

// DigitalSignatureTable is a table in a true type font.
//
// Port of org.apache.fontbox.ttf.DigitalSignatureTable, which reads nothing.
type DigitalSignatureTable struct {
	Table
}
