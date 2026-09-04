package ttf

import (
	"log/slog"
	"sort"
)

// KerningTable is a 'kern' table in a true type font.
//
// Port of org.apache.fontbox.ttf.KerningTable.
type KerningTable struct {
	Table

	subtables []*KerningSubtable
}

var _ TableReader = (*KerningTable)(nil)

// Read reads the required data from the stream.
func (t *KerningTable) Read(ttf *TrueTypeFont, data DataStream) error {
	r := newReader(data)
	version := r.unsignedShort()
	if version != 0 {
		version = version<<16 | r.unsignedShort()
	}
	if r.err != nil {
		return r.err
	}
	numSubtables := 0
	switch version {
	case 0:
		numSubtables = r.unsignedShort()
	case 1:
		// Java narrows the unsigned count to a signed 32-bit int, so a count
		// with bit 31 set goes negative and the check below skips it; a Go int
		// is 64 bits and would keep it positive and allocate on it.
		//
		// JAVA-BUGS entry 22: this case cannot be reached. version is zero or
		// at least 0x10000 by the time the switch sees it, so an Apple
		// version 1.0 'kern' table falls through to the default and is skipped.
		numSubtables = int(int32(r.unsignedInt()))
	default:
		slog.Debug("Skipped kerning table due to an unsupported kerning table version",
			"version", version)
	}
	if r.err != nil {
		return r.err
	}
	if numSubtables > 0 {
		t.subtables = make([]*KerningSubtable, numSubtables)
		for i := 0; i < numSubtables; i++ {
			subtable := &KerningSubtable{}
			if err := subtable.read(data, version); err != nil {
				return err
			}
			t.subtables[i] = subtable
		}
	}
	t.SetInitialized(true)
	return nil
}

// HorizontalKerningSubtable obtains the first subtable that supports
// non-cross-stream horizontal kerning, or nil where there is none.
func (t *KerningTable) HorizontalKerningSubtable() *KerningSubtable {
	return t.HorizontalKerningSubtableCross(false)
}

// HorizontalKerningSubtableCross obtains the first subtable that supports
// horizontal kerning with the specified cross stream, or nil where there is
// none.
func (t *KerningTable) HorizontalKerningSubtableCross(cross bool) *KerningSubtable {
	for _, s := range t.subtables {
		if s.IsHorizontalKerningCross(cross) {
			return s
		}
	}
	return nil
}

// The coverage field bit masks and values.
const (
	coverageHorizontal  = 0x0001
	coverageMinimums    = 0x0002
	coverageCrossStream = 0x0004
	coverageFormat      = 0xFF00

	coverageHorizontalShift  = 0
	coverageMinimumsShift    = 1
	coverageCrossStreamShift = 2
	coverageFormatShift      = 8
)

// KerningSubtable is one subtable of a 'kern' table in a true type font.
//
// Port of org.apache.fontbox.ttf.KerningSubtable.
type KerningSubtable struct {
	// horizontal is true if horizontal kerning
	horizontal bool
	// minimums is true if minimum adjustment values (versus kerning values)
	minimums bool
	// crossStream is true if cross-stream (block progression) kerning
	crossStream bool
	// pairs is the format specific pair data
	pairs kerningPairData
}

// read reads the required data from the stream, version being the version of
// the table to be read.
func (t *KerningSubtable) read(data DataStream, version int) error {
	switch version {
	case 0:
		return t.readSubtable0(data)
	case 1:
		t.readSubtable1()
		return nil
	}
	// Java throws IllegalStateException, which is unchecked.
	panic("ttf: unsupported kerning subtable version")
}

// IsHorizontalKerning determines if the subtable is designated for use in
// horizontal writing modes and contains inline progression kerning pairs (not
// block progression "cross stream" kerning pairs).
func (t *KerningSubtable) IsHorizontalKerning() bool {
	return t.IsHorizontalKerningCross(false)
}

// IsHorizontalKerningCross determines if the subtable is designated for use in
// horizontal writing modes, contains kerning pairs (as opposed to minimum
// pairs), and, if cross is true, returns the cross stream designator;
// otherwise, if cross is false, returns true if the cross stream designator is
// false.
func (t *KerningSubtable) IsHorizontalKerningCross(cross bool) bool {
	switch {
	case !t.horizontal:
		return false
	case t.minimums:
		return false
	case cross:
		return t.crossStream
	default:
		return !t.crossStream
	}
}

// Kerning obtains the kerning adjustments for the given glyph sequence, where
// the Nth returned adjustment is associated with the Nth glyph and the
// succeeding non-zero glyph in the sequence.
//
// Kerning adjustments are returned in font design coordinates.
func (t *KerningSubtable) Kerning(glyphs []int) []int {
	if t.pairs == nil {
		slog.Warn("No kerning subtable data available due to an unsupported kerning subtable version")
		return nil
	}
	ng := len(glyphs)
	kerning := make([]int, ng)
	for i := 0; i < ng; i++ {
		l := glyphs[i]
		r := -1
		for k := i + 1; k < ng; k++ {
			if g := glyphs[k]; g >= 0 {
				r = g
				break
			}
		}
		kerning[i] = t.KerningPair(l, r)
	}
	return kerning
}

// KerningPair obtains the kerning adjustment for the glyph pair {l, r}.
func (t *KerningSubtable) KerningPair(l, r int) int {
	if t.pairs == nil {
		slog.Warn("No kerning subtable data available due to an unsupported kerning subtable version")
		return 0
	}
	return t.pairs.kerning(l, r)
}

func (t *KerningSubtable) readSubtable0(data DataStream) error {
	r := newReader(data)
	version := r.unsignedShort()
	if r.err != nil {
		return r.err
	}
	if version != 0 {
		slog.Info("Unsupported kerning sub-table version", "version", version)
		return nil
	}
	length := r.unsignedShort()
	if r.err != nil {
		return r.err
	}
	if length < 6 {
		slog.Warn("Kerning sub-table too short", "got", length, "expect", "6 or more")
		return nil
	}
	coverage := r.unsignedShort()
	if r.err != nil {
		return r.err
	}
	if isBitsSet(coverage, coverageHorizontal, coverageHorizontalShift) {
		t.horizontal = true
	}
	if isBitsSet(coverage, coverageMinimums, coverageMinimumsShift) {
		t.minimums = true
	}
	if isBitsSet(coverage, coverageCrossStream, coverageCrossStreamShift) {
		t.crossStream = true
	}
	format := getBits(coverage, coverageFormat, coverageFormatShift)
	switch format {
	case 0:
		return t.readSubtable0Format0(data)
	case 2:
		slog.Info("Kerning subtable format 2 not yet supported.")
	default:
		slog.Debug("Skipped kerning subtable due to an unsupported kerning subtable version",
			"version", format)
	}
	return nil
}

func (t *KerningSubtable) readSubtable0Format0(data DataStream) error {
	pairs := &pairData0Format0{}
	if err := pairs.read(data); err != nil {
		return err
	}
	t.pairs = pairs
	return nil
}

func (t *KerningSubtable) readSubtable1() {
	slog.Info("Kerning subtable format 1 not yet supported.")
}

func isBitsSet(bits, mask, shift int) bool { return getBits(bits, mask, shift) != 0 }

func getBits(bits, mask, shift int) int { return (bits & mask) >> shift }

// kerningPairData is the format specific pair data of a kerning subtable.
type kerningPairData interface {
	read(data DataStream) error
	kerning(l, r int) int
}

// pairData0Format0 is the format 0 pair data of a version 0 subtable.
type pairData0Format0 struct {
	pairs [][3]int
}

func (p *pairData0Format0) read(data DataStream) error {
	r := newReader(data)
	numPairs := r.unsignedShort()
	_ = r.unsignedShort() / 6 // searchRange
	_ = r.unsignedShort()     // entrySelector
	_ = r.unsignedShort()     // rangeShift
	if r.err != nil {
		return r.err
	}
	p.pairs = make([][3]int, numPairs)
	for i := 0; i < numPairs; i++ {
		left := r.unsignedShort()
		right := r.unsignedShort()
		value := r.signedShort()
		if r.err != nil {
			return r.err
		}
		p.pairs[i][0] = left
		p.pairs[i][1] = right
		p.pairs[i][2] = int(value)
	}
	return nil
}

func (p *pairData0Format0) kerning(l, r int) int {
	// Java uses Arrays.binarySearch with a comparator over the first two
	// columns, which needs the table to already be sorted by them; the format
	// requires that, and this search assumes it too.
	index := sort.Search(len(p.pairs), func(i int) bool {
		if p.pairs[i][0] != l {
			return p.pairs[i][0] >= l
		}
		return p.pairs[i][1] >= r
	})
	if index < len(p.pairs) && p.pairs[index][0] == l && p.pairs[index][1] == r {
		return p.pairs[index][2]
	}
	return 0
}
