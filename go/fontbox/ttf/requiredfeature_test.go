package ttf

// Which features a language system turns on, and which language system.
//
// Two rules of the specification that none of the layout fonts exercises, so
// the table they are checked against is written here rather than read from a
// font:
//
//   - A language system's RequiredFeatureIndex names a feature that runs
//     whether or not the caller asked for it. Of the seven fonts the layout
//     tests use, not one declares a required feature in GPOS, so nothing else
//     in this repository would notice it being dropped.
//   - A script's named language systems are alternatives to its default -- the
//     Turkish way of setting Latin, the Serbian way of setting Cyrillic -- and
//     a run that names no language gets the default. Applying all of them at
//     once sets the text in a language nobody asked for.

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// TestRequiredFeatureRunsUnasked builds a GPOS table by hand and checks the
// three answers.
func TestRequiredFeatureRunsUnasked(t *testing.T) {
	table := readSyntheticGPOS(t, syntheticGPOS())

	// The required feature moves glyph 5, and the caller asked for nothing.
	if got := table.Position([]int{5}, []string{"TEST"}, nil); got[0].XAdvance != -100 {
		t.Errorf("the required feature adjusted glyph 5 by %d, want -100: a "+
			"language system that requires a feature gets it whatever the "+
			"caller asked for", got[0].XAdvance)
	}

	// The optional feature moves glyph 6, but only when asked for.
	if got := table.Position([]int{6}, []string{"TEST"}, nil); !got[0].IsZero() {
		t.Errorf("an optional feature moved glyph 6 by %+v with nothing asked for",
			got[0])
	}
	if got := table.Position([]int{6}, []string{"TEST"}, []string{"kern"}); got[0].XAdvance != -7 {
		t.Errorf("with kern asked for, glyph 6 moved by %d, want -7", got[0].XAdvance)
	}

	// The named language system's feature never runs, even when its tag is
	// asked for: the run named no language, so it gets the default.
	if got := table.Position([]int{7}, []string{"TEST"}, []string{"locl"}); !got[0].IsZero() {
		t.Errorf("glyph 7 moved by %+v; that feature belongs to a named language "+
			"system and this run named no language", got[0])
	}
}

// readSyntheticGPOS parses the bytes as a GPOS table.
func readSyntheticGPOS(t *testing.T, table []byte) *GlyphPositioningTable {
	t.Helper()
	data, err := NewDataStreamFromReader(bytes.NewReader(table))
	if err != nil {
		t.Fatalf("wrapping the table: %v", err)
	}
	read := &GlyphPositioningTable{}
	if err := read.Read(nil, data); err != nil {
		t.Fatalf("reading the table: %v", err)
	}
	return read
}

// syntheticGPOS writes a GPOS table with one script, whose default language
// system requires feature 0 and turns on feature 1, and whose named language
// system "TRK " turns on feature 2.
//
// Every offset is from the start of the structure that holds it, which is what
// the specification says and what the reader assumes.
func syntheticGPOS() []byte {
	// Laid out as: header, script list, feature list, lookup list, and the
	// coverage tables after them.
	const (
		headerSize     = 10
		scriptListAt   = headerSize
		scriptTableAt  = scriptListAt + 8   // count and one record
		defaultLangAt  = scriptTableAt + 10 // an offset, a count and one record
		namedLangAt    = defaultLangAt + 8  // lookupOrder, required, count, one index
		featureListAt  = namedLangAt + 8
		featureTableAt = featureListAt + 2 + 3*6 // count and three records
		lookupListAt   = featureTableAt + 3*6    // three tables of six bytes
		lookupAt       = lookupListAt + 2 + 3*2  // count and three offsets
		subtableAt     = lookupAt + 3*8          // three lookups of eight bytes
		coverageAt     = subtableAt + 3*8        // three subtables of eight bytes
	)

	out := &tableWriter{}
	// GPOS header.
	out.short(1, 0, scriptListAt, featureListAt, lookupListAt)

	// ScriptList: one script.
	out.short(1)
	out.tag("TEST")
	out.short(scriptTableAt - scriptListAt)

	// ScriptTable: a default language system and one named one.
	out.short(defaultLangAt-scriptTableAt, 1)
	out.tag("TRK ")
	out.short(namedLangAt - scriptTableAt)

	// The default language system: requires feature 0, turns on feature 1.
	out.short(0, 0, 1, 1)
	// The named one: requires nothing, turns on feature 2.
	out.short(0, noRequiredFeature, 1, 2)

	// FeatureList: three features.
	out.short(3)
	for i, tag := range []string{"requ", "kern", "locl"} {
		out.tag(tag)
		out.short(featureTableAt + i*6 - featureListAt)
	}
	// Each feature table names one lookup, its own.
	for i := 0; i < 3; i++ {
		out.short(0, 1, i)
	}

	// LookupList: three lookups, each a single adjustment over one glyph.
	out.short(3)
	for i := 0; i < 3; i++ {
		out.short(lookupAt + i*8 - lookupListAt)
	}
	for i := 0; i < 3; i++ {
		out.short(1, 0, 1, subtableAt+i*8-(lookupAt+i*8))
	}

	// The subtables: format 1, one value for every covered glyph.
	for i, advance := range []int{-100, -7, -3} {
		out.short(1, coverageAt+i*6-(subtableAt+i*8), valueXAdvance, advance)
	}

	// The coverage tables: format 1, one glyph each.
	for _, glyph := range []int{5, 6, 7} {
		out.short(1, 1, glyph)
	}
	return out.bytes
}

// tableWriter writes big-endian shorts and tags, which is all a font table is.
type tableWriter struct{ bytes []byte }

func (w *tableWriter) short(values ...int) {
	for _, value := range values {
		w.bytes = binary.BigEndian.AppendUint16(w.bytes, uint16(value))
	}
}

func (w *tableWriter) tag(tag string) { w.bytes = append(w.bytes, tag...) }
