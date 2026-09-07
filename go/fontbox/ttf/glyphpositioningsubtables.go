package ttf

// The positioning subtables of GPOS: single adjustment, pair adjustment,
// mark-to-base and mark-to-mark.
//
// Written from the OpenType specification; see glyphpositioning.go for why this
// is not a port of anything.

import (
	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf/table/common"
)

// The bits of a ValueFormat, each saying that one more field follows.
const (
	valueXPlacement       = 0x0001
	valueYPlacement       = 0x0002
	valueXAdvance         = 0x0004
	valueYAdvance         = 0x0008
	valueXPlacementDevice = 0x0010
	valueYPlacementDevice = 0x0020
	valueXAdvanceDevice   = 0x0040
	valueYAdvanceDevice   = 0x0080
)

// readValueRecord reads the fields a ValueFormat says are present.
//
// The four device-table offsets are read and dropped: they carry per-pixel-size
// corrections for a hinted rasteriser, and a PDF is laid out in font design
// units at no particular size.
func readValueRecord(r *reader, valueFormat int) GlyphPosition {
	// A value record moves a glyph; it never hangs one off another, and a
	// record that moves it by nothing has to answer true to IsZero.
	position := GlyphPosition{AttachedTo: NotAttached}
	if valueFormat&valueXPlacement != 0 {
		position.XPlacement = int(r.signedShort())
	}
	if valueFormat&valueYPlacement != 0 {
		position.YPlacement = int(r.signedShort())
	}
	if valueFormat&valueXAdvance != 0 {
		position.XAdvance = int(r.signedShort())
	}
	if valueFormat&valueYAdvance != 0 {
		position.YAdvance = int(r.signedShort())
	}
	for _, bit := range []int{valueXPlacementDevice, valueYPlacementDevice,
		valueXAdvanceDevice, valueYAdvanceDevice} {
		if valueFormat&bit != 0 {
			r.unsignedShort()
		}
	}
	return position
}

// add applies one adjustment on top of another, which is what a second lookup
// touching the same glyph does.
func (p *GlyphPosition) add(other GlyphPosition) {
	p.XPlacement += other.XPlacement
	p.YPlacement += other.YPlacement
	p.XAdvance += other.XAdvance
	p.YAdvance += other.YAdvance
}

// singleAdjustment is lookup type 1: every covered glyph takes the same
// adjustment, or one of its own.
type singleAdjustment struct {
	coverage common.CoverageTable
	// values holds one entry in format 1, and one per covered glyph in
	// format 2.
	values []GlyphPosition
}

func readSingleAdjustment(data DataStream, offset int64) (positioningSubtable, error) {
	if err := data.SeekTo(offset); err != nil {
		return nil, err
	}
	r := newReader(data)
	format := r.unsignedShort()
	coverageOffset := r.unsignedShort()
	valueFormat := r.unsignedShort()
	if r.err != nil {
		return nil, r.err
	}

	t := &singleAdjustment{}
	switch format {
	case 1:
		t.values = []GlyphPosition{readValueRecord(r, valueFormat)}
	case 2:
		valueCount := r.unsignedShort()
		if r.err != nil {
			return nil, r.err
		}
		t.values = make([]GlyphPosition, valueCount)
		for i := 0; i < valueCount; i++ {
			t.values[i] = readValueRecord(r, valueFormat)
		}
	default:
		return nil, nil
	}
	if r.err != nil {
		return nil, r.err
	}

	coverage, err := readLayoutCoverageTable(data, offset+int64(coverageOffset))
	if err != nil {
		return nil, err
	}
	t.coverage = coverage
	return t, nil
}

func (t *singleAdjustment) position(glyphs []int, i int, out []GlyphPosition) int {
	index := t.coverage.CoverageIndex(glyphs[i])
	if index < 0 {
		return 0
	}
	switch {
	case len(t.values) == 1:
		out[i].add(t.values[0])
	case index < len(t.values):
		out[i].add(t.values[index])
	default:
		return 0
	}
	return 1
}

// pairAdjustment is lookup type 2: a pair of glyphs takes an adjustment
// between them, which is kerning.
type pairAdjustment struct {
	coverage common.CoverageTable

	// format 1: one set of explicit second glyphs per covered first glyph.
	pairSets []map[int]pairValue

	// format 2: a class of the first glyph and a class of the second index a
	// grid.
	classDef1, classDef2 *classDefinition
	classValues          [][]pairValue
}

// pairValue is the two adjustments a pair carries, one for each glyph.
type pairValue struct{ first, second GlyphPosition }

func readPairAdjustment(data DataStream, offset int64) (positioningSubtable, error) {
	if err := data.SeekTo(offset); err != nil {
		return nil, err
	}
	r := newReader(data)
	format := r.unsignedShort()
	coverageOffset := r.unsignedShort()
	valueFormat1 := r.unsignedShort()
	valueFormat2 := r.unsignedShort()
	if r.err != nil {
		return nil, r.err
	}

	t := &pairAdjustment{}
	switch format {
	case 1:
		pairSetCount := r.unsignedShort()
		if r.err != nil {
			return nil, r.err
		}
		pairSetOffsets := make([]int, pairSetCount)
		for i := 0; i < pairSetCount; i++ {
			pairSetOffsets[i] = r.unsignedShort()
		}
		if r.err != nil {
			return nil, r.err
		}
		t.pairSets = make([]map[int]pairValue, pairSetCount)
		for i, pairSetOffset := range pairSetOffsets {
			set, err := readPairSet(data, offset+int64(pairSetOffset),
				valueFormat1, valueFormat2)
			if err != nil {
				return nil, err
			}
			t.pairSets[i] = set
		}

	case 2:
		classDef1Offset := r.unsignedShort()
		classDef2Offset := r.unsignedShort()
		class1Count := r.unsignedShort()
		class2Count := r.unsignedShort()
		if r.err != nil {
			return nil, r.err
		}
		t.classValues = make([][]pairValue, class1Count)
		for i := 0; i < class1Count; i++ {
			t.classValues[i] = make([]pairValue, class2Count)
			for j := 0; j < class2Count; j++ {
				t.classValues[i][j] = pairValue{
					first:  readValueRecord(r, valueFormat1),
					second: readValueRecord(r, valueFormat2),
				}
			}
		}
		if r.err != nil {
			return nil, r.err
		}
		var err error
		if t.classDef1, err = readClassDefinition(data, offset+int64(classDef1Offset)); err != nil {
			return nil, err
		}
		if t.classDef2, err = readClassDefinition(data, offset+int64(classDef2Offset)); err != nil {
			return nil, err
		}

	default:
		return nil, nil
	}

	coverage, err := readLayoutCoverageTable(data, offset+int64(coverageOffset))
	if err != nil {
		return nil, err
	}
	t.coverage = coverage
	return t, nil
}

// readPairSet reads the second glyphs of one covered first glyph.
func readPairSet(data DataStream, offset int64,
	valueFormat1, valueFormat2 int) (map[int]pairValue, error) {
	if err := data.SeekTo(offset); err != nil {
		return nil, err
	}
	r := newReader(data)
	pairValueCount := r.unsignedShort()
	if r.err != nil {
		return nil, r.err
	}
	set := make(map[int]pairValue, pairValueCount)
	for i := 0; i < pairValueCount; i++ {
		secondGlyph := r.unsignedShort()
		value := pairValue{
			first:  readValueRecord(r, valueFormat1),
			second: readValueRecord(r, valueFormat2),
		}
		if r.err != nil {
			return nil, r.err
		}
		set[secondGlyph] = value
	}
	return set, nil
}

func (t *pairAdjustment) position(glyphs []int, i int, out []GlyphPosition) int {
	if i+1 >= len(glyphs) {
		return 0
	}
	index := t.coverage.CoverageIndex(glyphs[i])
	if index < 0 {
		return 0
	}
	second := glyphs[i+1]

	var value pairValue
	switch {
	case t.pairSets != nil:
		if index >= len(t.pairSets) {
			return 0
		}
		found, ok := t.pairSets[index][second]
		if !ok {
			return 0
		}
		value = found
	case t.classValues != nil:
		class1 := t.classDef1.classOf(glyphs[i])
		class2 := t.classDef2.classOf(second)
		if class1 >= len(t.classValues) || class2 >= len(t.classValues[class1]) {
			return 0
		}
		value = t.classValues[class1][class2]
	default:
		return 0
	}

	out[i].add(value.first)
	out[i+1].add(value.second)
	// The specification advances past the second glyph only where it took an
	// adjustment of its own; otherwise the second glyph may still start a pair.
	if value.second.IsZero() {
		return 1
	}
	return 2
}

// anchor is a point on a glyph that another glyph attaches to.
type anchor struct{ x, y int }

// readAnchor reads an anchor table. Formats 2 and 3 add a contour point and
// device tables, which say how to snap the anchor to a hinted outline; the
// coordinates are the same in all three.
func readAnchor(data DataStream, offset int64) (anchor, error) {
	if err := data.SeekTo(offset); err != nil {
		return anchor{}, err
	}
	r := newReader(data)
	r.unsignedShort() // anchorFormat
	x := int(r.signedShort())
	y := int(r.signedShort())
	if r.err != nil {
		return anchor{}, r.err
	}
	return anchor{x: x, y: y}, nil
}

// markRecord is one mark: which class it belongs to and where it attaches.
type markRecord struct {
	class  int
	anchor anchor
}

// markAttachment is lookup types 4 and 6: a mark attaches to a base glyph, or
// to another mark.
//
// The two differ only in what the second coverage covers, so one type serves
// both.
type markAttachment struct {
	markCoverage common.CoverageTable
	baseCoverage common.CoverageTable

	marks []markRecord

	// baseAnchors is one row per covered base, one column per mark class.
	baseAnchors [][]anchor

	// toMark says this is lookup type 6 rather than type 4: what the mark
	// attaches to is the mark before it, not the letter before it. The two
	// subtables have the same layout but not the same rule for finding what
	// the mark hangs off -- see position.
	toMark bool
}

func readMarkToBase(data DataStream, offset int64) (positioningSubtable, error) {
	return readMarkAttachment(data, offset, false)
}

func readMarkToMark(data DataStream, offset int64) (positioningSubtable, error) {
	return readMarkAttachment(data, offset, true)
}

// readMarkAttachment reads a MarkBasePos or MarkMarkPos subtable, which have
// the same layout.
func readMarkAttachment(data DataStream, offset int64, toMark bool) (positioningSubtable, error) {
	if err := data.SeekTo(offset); err != nil {
		return nil, err
	}
	r := newReader(data)
	format := r.unsignedShort()
	markCoverageOffset := r.unsignedShort()
	baseCoverageOffset := r.unsignedShort()
	markClassCount := r.unsignedShort()
	markArrayOffset := r.unsignedShort()
	baseArrayOffset := r.unsignedShort()
	if r.err != nil {
		return nil, r.err
	}
	if format != 1 {
		return nil, nil
	}

	t := &markAttachment{toMark: toMark}
	var err error
	if t.markCoverage, err = readLayoutCoverageTable(data,
		offset+int64(markCoverageOffset)); err != nil {
		return nil, err
	}
	if t.baseCoverage, err = readLayoutCoverageTable(data,
		offset+int64(baseCoverageOffset)); err != nil {
		return nil, err
	}
	if t.marks, err = readMarkArray(data, offset+int64(markArrayOffset)); err != nil {
		return nil, err
	}
	if t.baseAnchors, err = readBaseArray(data, offset+int64(baseArrayOffset),
		markClassCount); err != nil {
		return nil, err
	}
	return t, nil
}

// readMarkArray reads the class and anchor of every mark.
func readMarkArray(data DataStream, offset int64) ([]markRecord, error) {
	if err := data.SeekTo(offset); err != nil {
		return nil, err
	}
	r := newReader(data)
	markCount := r.unsignedShort()
	if r.err != nil {
		return nil, r.err
	}
	classes := make([]int, markCount)
	anchorOffsets := make([]int, markCount)
	for i := 0; i < markCount; i++ {
		classes[i] = r.unsignedShort()
		anchorOffsets[i] = r.unsignedShort()
	}
	if r.err != nil {
		return nil, r.err
	}
	marks := make([]markRecord, markCount)
	for i := 0; i < markCount; i++ {
		marks[i].class = classes[i]
		if anchorOffsets[i] == 0 {
			continue
		}
		a, err := readAnchor(data, offset+int64(anchorOffsets[i]))
		if err != nil {
			return nil, err
		}
		marks[i].anchor = a
	}
	return marks, nil
}

// readBaseArray reads one anchor per base glyph per mark class.
func readBaseArray(data DataStream, offset int64, markClassCount int) ([][]anchor, error) {
	if err := data.SeekTo(offset); err != nil {
		return nil, err
	}
	r := newReader(data)
	baseCount := r.unsignedShort()
	if r.err != nil {
		return nil, r.err
	}
	offsets := make([][]int, baseCount)
	for i := 0; i < baseCount; i++ {
		offsets[i] = make([]int, markClassCount)
		for j := 0; j < markClassCount; j++ {
			offsets[i][j] = r.unsignedShort()
		}
	}
	if r.err != nil {
		return nil, r.err
	}
	anchors := make([][]anchor, baseCount)
	for i := 0; i < baseCount; i++ {
		anchors[i] = make([]anchor, markClassCount)
		for j := 0; j < markClassCount; j++ {
			if offsets[i][j] == 0 {
				continue
			}
			a, err := readAnchor(data, offset+int64(offsets[i][j]))
			if err != nil {
				return nil, err
			}
			anchors[i][j] = a
		}
	}
	return anchors, nil
}

func (t *markAttachment) position(glyphs []int, i int, out []GlyphPosition) int {
	markIndex := t.markCoverage.CoverageIndex(glyphs[i])
	if markIndex < 0 || markIndex >= len(t.marks) || i == 0 {
		return 0
	}
	// What the mark hangs off.
	//
	// A mark-to-mark lookup takes the glyph immediately before, and nothing
	// else: two marks that are not next to each other are not stacked. A
	// mark-to-base lookup steps back over the marks in between -- that is what
	// the specification means by skipping glyphs the lookup flags exclude --
	// and takes the first glyph that is not one.
	//
	// Neither may run past a letter to reach something further back. Doing so
	// attaches a mark to a letter it does not belong to, and puts it a word
	// away on the page.
	baseAt := i - 1
	if !t.toMark {
		for baseAt >= 0 && t.markCoverage.CoverageIndex(glyphs[baseAt]) >= 0 {
			baseAt--
		}
	}
	if baseAt < 0 || t.baseCoverage.CoverageIndex(glyphs[baseAt]) < 0 {
		return 0
	}
	baseIndex := t.baseCoverage.CoverageIndex(glyphs[baseAt])
	if baseIndex >= len(t.baseAnchors) {
		return 0
	}
	mark := t.marks[markIndex]
	if mark.class >= len(t.baseAnchors[baseIndex]) {
		return 0
	}
	base := t.baseAnchors[baseIndex][mark.class]

	// The mark is placed so that its anchor meets the base's. Both anchors are
	// measured from the origin of the glyph they belong to, so what comes out
	// here is a distance from the base's origin -- not from the pen, which by
	// now has moved past it. Turning one into the other needs to know in which
	// order the run is drawn, and only the caller knows that.
	out[i].XPlacement = base.x - mark.anchor.x
	out[i].YPlacement = base.y - mark.anchor.y
	out[i].AttachedTo = baseAt
	return 1
}

// classDefinition maps a glyph to the class a class-based subtable groups it
// by. A glyph the table does not name is class 0.
type classDefinition struct {
	format int

	// format 1: a first glyph and one class per glyph after it.
	startGlyph int
	classes    []int

	// format 2: ranges.
	ranges []classRange
}

type classRange struct{ start, end, class int }

// readClassDefinition reads a class definition table.
func readClassDefinition(data DataStream, offset int64) (*classDefinition, error) {
	if err := data.SeekTo(offset); err != nil {
		return nil, err
	}
	r := newReader(data)
	format := r.unsignedShort()
	if r.err != nil {
		return nil, r.err
	}
	t := &classDefinition{format: format}
	switch format {
	case 1:
		t.startGlyph = r.unsignedShort()
		glyphCount := r.unsignedShort()
		if r.err != nil {
			return nil, r.err
		}
		t.classes = make([]int, glyphCount)
		for i := 0; i < glyphCount; i++ {
			t.classes[i] = r.unsignedShort()
		}
	case 2:
		rangeCount := r.unsignedShort()
		if r.err != nil {
			return nil, r.err
		}
		t.ranges = make([]classRange, rangeCount)
		for i := 0; i < rangeCount; i++ {
			t.ranges[i] = classRange{
				start: r.unsignedShort(),
				end:   r.unsignedShort(),
				class: r.unsignedShort(),
			}
		}
	}
	if r.err != nil {
		return nil, r.err
	}
	return t, nil
}

// classOf answers which class a glyph is in, which is 0 where the table does
// not name it.
func (t *classDefinition) classOf(gid int) int {
	if t == nil {
		return 0
	}
	switch t.format {
	case 1:
		if gid >= t.startGlyph && gid-t.startGlyph < len(t.classes) {
			return t.classes[gid-t.startGlyph]
		}
	case 2:
		low, high := 0, len(t.ranges)-1
		for low <= high {
			middle := (low + high) / 2
			switch {
			case gid < t.ranges[middle].start:
				high = middle - 1
			case gid > t.ranges[middle].end:
				low = middle + 1
			default:
				return t.ranges[middle].class
			}
		}
	}
	return 0
}
