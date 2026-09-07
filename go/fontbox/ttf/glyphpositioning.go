package ttf

// The OpenType GPOS table: glyph positioning.
//
// **This is not a port.** PDFBox has no GPOS reader — `OTFParser.readTable`
// answers a bare `OTLTable` for the tag, with the comment "todo: this is a
// stub, a full implementation is needed" — and it does not need one, because it
// borrows layout from `java.awt.font.TextLayout` or from Apache FOP. Go has
// neither, so `track/pdfbox-layout` has nothing to translate and this is
// written from the OpenType specification instead.
//
// What it is for: GSUB, which slice 4 did port, decides *which* glyph to draw —
// an `f` and an `i` become an `fi`, an Arabic letter takes its initial or medial
// shape. GPOS decides *where* each one goes — the kern that pulls the `V`
// under the `A`, the vowel mark that sits over the right letter. Together they
// are what `layoutGlyphVector` answers in one call.
//
// Scope: lookup types 1, 2, 4, 6 and 9, which are single adjustment, pair
// adjustment (kerning), mark-to-base, mark-to-mark and the extension
// indirection. Types 3, 5, 7 and 8 are read far enough to be skipped and are
// recorded in migration/STATUS.md. Device tables are read and ignored, which is
// what a renderer at one size does with them.

import (
	"fmt"

	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf/table/common"
)

// GlyphPositioningTag is the table's tag.
const GlyphPositioningTag = "GPOS"

// GlyphPosition is the adjustment GPOS makes to one glyph, in font design
// units.
type GlyphPosition struct {
	// XPlacement and YPlacement move the glyph without moving the pen. For a
	// glyph with AttachedTo set they are measured from the origin of that
	// glyph instead, which is what an anchor in the font says.
	XPlacement, YPlacement int

	// XAdvance and YAdvance move the pen after it.
	XAdvance, YAdvance int

	// AttachedTo is the glyph this one hangs off -- the letter under a vowel
	// mark, or the mark under a second mark -- as an index into the run, or
	// NotAttached.
	//
	// A caller resolves it into a distance from the pen once it knows in which
	// order the glyphs are drawn: a right-to-left run draws them the other way
	// round, and then the mark comes before the letter it belongs to. Doing
	// that here would mean assuming the order, and the assumption would be
	// wrong half the time.
	AttachedTo int
}

// NotAttached is the AttachedTo of a glyph that hangs off nothing.
const NotAttached = -1

// IsZero reports whether the position leaves the glyph where it was.
func (p GlyphPosition) IsZero() bool {
	return p.XPlacement == 0 && p.YPlacement == 0 && p.XAdvance == 0 &&
		p.YAdvance == 0 && p.AttachedTo == NotAttached
}

// GlyphPositioningTable is the GPOS table of a font.
type GlyphPositioningTable struct {
	Table

	scriptList  map[string]*common.ScriptTable
	featureList []*common.FeatureRecord
	lookupList  []*positioningLookup
}

var _ TableReader = (*GlyphPositioningTable)(nil)

// Read parses the table.
//
// The header is the same shape as GSUB's -- a script list, a feature list and a
// lookup list -- and is read the same way; only the lookup subtables differ.
func (t *GlyphPositioningTable) Read(ttf *TrueTypeFont, data DataStream) error {
	start := data.CurrentPosition()
	r := newReader(data)
	majorVersion := r.unsignedShort()
	r.unsignedShort() // minorVersion, which only adds a feature variations offset
	if r.err != nil {
		return r.err
	}
	if majorVersion != 1 {
		return fmt.Errorf("ttf: unsupported GPOS table version %d", majorVersion)
	}
	scriptListOffset := r.unsignedShort()
	featureListOffset := r.unsignedShort()
	lookupListOffset := r.unsignedShort()
	if r.err != nil {
		return r.err
	}

	scripts, err := readLayoutScriptList(data, start+int64(scriptListOffset))
	if err != nil {
		return err
	}
	t.scriptList = scripts

	features, err := readLayoutFeatureList(data, start+int64(featureListOffset))
	if err != nil {
		return err
	}
	t.featureList = features

	lookups, err := t.readLookupList(data, start+int64(lookupListOffset))
	if err != nil {
		return err
	}
	t.lookupList = lookups
	t.collectMarks()

	t.initialized = true
	return nil
}

// markSet is which glyphs of a font are combining marks.
//
// The authority on that is GDEF's glyph class definition, which neither PDFBox
// nor this port reads. What stands in for it is the table's own account of
// itself: every glyph any mark attachment subtable covers as a mark is a mark.
// A font that positions marks says so in those coverages, so the set is
// complete for the fonts where the answer matters.
type markSet struct {
	coverages []common.CoverageTable
}

// contains reports whether the glyph is a combining mark.
func (s *markSet) contains(gid int) bool {
	for _, coverage := range s.coverages {
		if coverage.CoverageIndex(gid) >= 0 {
			return true
		}
	}
	return false
}

// collectMarks builds the set of mark glyphs and hands it to every mark
// attachment subtable, which needs to know which glyphs to step over when it
// looks back for the letter a mark belongs to.
//
// It has to happen after every lookup is read: the mark of one subtable is a
// glyph another subtable has to step over, and the two are not read together.
func (t *GlyphPositioningTable) collectMarks() {
	marks := &markSet{}
	for _, lookup := range t.lookupList {
		for _, subtable := range lookup.subTables {
			if attachment, isAttachment := subtable.(*markAttachment); isAttachment {
				marks.coverages = append(marks.coverages, attachment.markCoverage)
			}
		}
	}
	for _, lookup := range t.lookupList {
		for _, subtable := range lookup.subTables {
			if attachment, isAttachment := subtable.(*markAttachment); isAttachment {
				attachment.everyMark = marks
			}
		}
	}
}

// ScriptTags returns the scripts the table carries.
func (t *GlyphPositioningTable) ScriptTags() []string {
	tags := make([]string, 0, len(t.scriptList))
	for tag := range t.scriptList {
		tags = append(tags, tag)
	}
	return tags
}

// positioningLookup is one lookup, holding subtables that position rather than
// substitute.
type positioningLookup struct {
	lookupType int
	lookupFlag int
	subTables  []positioningSubtable
}

// positioningSubtable is one subtable of a positioning lookup.
type positioningSubtable interface {
	// position adjusts the glyph at index i of the run, answering the
	// adjustment and how many glyphs the subtable consumed. A subtable that
	// does not apply answers a zero position and 0.
	position(glyphs []int, i int, out []GlyphPosition) int
}

// readLookupList reads the lookup list and every lookup in it.
func (t *GlyphPositioningTable) readLookupList(data DataStream,
	offset int64) ([]*positioningLookup, error) {
	if err := data.SeekTo(offset); err != nil {
		return nil, err
	}
	r := newReader(data)
	lookupCount := r.unsignedShort()
	if r.err != nil {
		return nil, r.err
	}
	offsets := make([]int, lookupCount)
	for i := 0; i < lookupCount; i++ {
		offsets[i] = r.unsignedShort()
	}
	if r.err != nil {
		return nil, r.err
	}
	lookups := make([]*positioningLookup, 0, lookupCount)
	for _, lookupOffset := range offsets {
		lookup, err := t.readLookup(data, offset+int64(lookupOffset))
		if err != nil {
			return nil, err
		}
		lookups = append(lookups, lookup)
	}
	return lookups, nil
}

// readLookup reads one lookup and its subtables.
func (t *GlyphPositioningTable) readLookup(data DataStream,
	offset int64) (*positioningLookup, error) {
	if err := data.SeekTo(offset); err != nil {
		return nil, err
	}
	r := newReader(data)
	lookupType := r.unsignedShort()
	lookupFlag := r.unsignedShort()
	subTableCount := r.unsignedShort()
	if r.err != nil {
		return nil, r.err
	}
	offsets := make([]int, subTableCount)
	for i := 0; i < subTableCount; i++ {
		offsets[i] = r.unsignedShort()
	}
	if r.err != nil {
		return nil, r.err
	}
	// A lookup with the useMarkFilteringSet flag carries one more field, which
	// this reader does not use but has to step over where it reads on.
	if lookupFlag&useMarkFilteringSet != 0 {
		r.unsignedShort()
	}

	lookup := &positioningLookup{lookupType: lookupType, lookupFlag: lookupFlag}
	for _, subOffset := range offsets {
		subtable, err := t.readSubtable(data, lookupType, offset+int64(subOffset))
		if err != nil {
			return nil, err
		}
		if subtable != nil {
			lookup.subTables = append(lookup.subTables, subtable)
		}
	}
	return lookup, nil
}

// useMarkFilteringSet is the lookup flag that adds a mark filtering set field.
//
// It is the only lookup flag this reader acts on, and it acts on it only to
// step over the extra field it adds. The rest -- ignoreBaseGlyphs,
// ignoreLigatures, ignoreMarks, the mark attachment type in the high byte, and
// the mark filtering set itself -- say which glyphs a lookup should skip over
// while matching, and none of them is applied: a lookup that asks to ignore
// marks still sees them. Applying them needs the GDEF table, which neither
// PDFBox nor this port reads. See migration/STATUS.md.
const useMarkFilteringSet = 0x0010

// readSubtable reads one positioning subtable, answering nil for a lookup type
// this reader does not implement.
func (t *GlyphPositioningTable) readSubtable(data DataStream, lookupType int,
	offset int64) (positioningSubtable, error) {
	switch lookupType {
	case 1:
		return readSingleAdjustment(data, offset)
	case 2:
		return readPairAdjustment(data, offset)
	case 4:
		return readMarkToBase(data, offset)
	case 6:
		return readMarkToMark(data, offset)
	case 9:
		// Extension positioning: a type and a 32-bit offset to the real
		// subtable, which lets a lookup reach past 64K.
		if err := data.SeekTo(offset); err != nil {
			return nil, err
		}
		r := newReader(data)
		format := r.unsignedShort()
		extensionType := r.unsignedShort()
		extensionOffset := r.unsignedInt()
		if r.err != nil {
			return nil, r.err
		}
		if format != 1 || extensionType == 9 {
			return nil, nil
		}
		return t.readSubtable(data, extensionType, offset+extensionOffset)
	default:
		// 3 cursive, 5 mark-to-ligature, 7 contextual, 8 chained contextual.
		// See migration/STATUS.md.
		return nil, nil
	}
}

// Position adjusts a run of glyphs, answering one GlyphPosition per glyph.
//
// scriptTags says which script the run is in, and featureTags which features to
// turn on -- "kern" for kerning, "mark" and "mkmk" for mark attachment. A
// feature the font does not carry is skipped.
//
// The result is in font design units, which is what the rest of fontbox works
// in; a caller scales by fontSize/unitsPerEm.
func (t *GlyphPositioningTable) Position(glyphs []int, scriptTags []string,
	featureTags []string) []GlyphPosition {
	positions := make([]GlyphPosition, len(glyphs))
	for i := range positions {
		positions[i].AttachedTo = NotAttached
	}
	if len(glyphs) == 0 {
		return positions
	}

	for _, lookupIndex := range t.lookupsFor(scriptTags, featureTags) {
		if lookupIndex >= len(t.lookupList) {
			continue
		}
		lookup := t.lookupList[lookupIndex]
		// One walk of the run per lookup, trying the lookup's subtables in
		// order at each glyph and taking the first that applies -- which is
		// what the specification says a lookup does. Walking the run once per
		// subtable instead lets two subtables of one lookup both adjust the
		// same glyph, and they are usually alternative ways of saying the same
		// thing about different glyph ranges.
		for i := 0; i < len(glyphs); {
			consumed := 0
			for _, subtable := range lookup.subTables {
				if consumed = subtable.position(glyphs, i, positions); consumed > 0 {
					break
				}
			}
			if consumed <= 0 {
				consumed = 1
			}
			i += consumed
		}
	}
	return positions
}

// lookupsFor answers which lookups the wanted features run, in the order the
// lookup list holds them, which is the order the specification applies them in.
func (t *GlyphPositioningTable) lookupsFor(scriptTags, featureTags []string) []int {
	wanted := make(map[string]bool, len(featureTags))
	for _, tag := range featureTags {
		wanted[tag] = true
	}

	// Which feature indices the script's language systems turn on.
	enabled := map[int]bool{}
	for _, scriptTag := range t.selectScriptTags(scriptTags) {
		script := t.scriptList[scriptTag]
		if script == nil {
			continue
		}
		langSysTables := []*common.LangSysTable{}
		if script.DefaultLangSysTable() != nil {
			langSysTables = append(langSysTables, script.DefaultLangSysTable())
		}
		for _, table := range script.LangSysTables() {
			langSysTables = append(langSysTables, table)
		}
		for _, langSys := range langSysTables {
			for _, index := range langSys.FeatureIndices() {
				enabled[index] = true
			}
		}
	}

	seen := map[int]bool{}
	var lookups []int
	for index, record := range t.featureList {
		if !enabled[index] || !wanted[record.FeatureTag()] {
			continue
		}
		for _, lookupIndex := range record.FeatureTable().LookupListIndices() {
			if !seen[lookupIndex] {
				seen[lookupIndex] = true
				lookups = append(lookups, lookupIndex)
			}
		}
	}
	sortInts(lookups)
	return lookups
}

// selectScriptTags answers which of the wanted scripts the font carries,
// falling back to the default script every font is meant to have.
func (t *GlyphPositioningTable) selectScriptTags(wanted []string) []string {
	var found []string
	for _, tag := range wanted {
		if _, ok := t.scriptList[tag]; ok {
			found = append(found, tag)
		}
	}
	if len(found) == 0 {
		if _, ok := t.scriptList["DFLT"]; ok {
			found = append(found, "DFLT")
		}
	}
	return found
}

// sortInts puts the lookup indices in order, which is the order the
// specification applies them in.
func sortInts(values []int) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j-1] > values[j]; j-- {
			values[j-1], values[j] = values[j], values[j-1]
		}
	}
}
