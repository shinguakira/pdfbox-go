// Package common holds the tables of the OpenType Layout Common Table Format,
// which GSUB and GPOS both build on.
//
// Port of org.apache.fontbox.ttf.table.common.
package common

import (
	"fmt"
	"sort"
	"strings"
)

// CoverageTable says which glyphs a lookup applies to.
//
// Port of the abstract org.apache.fontbox.ttf.table.common.CoverageTable.
type CoverageTable interface {
	// CoverageIndex returns where the given glyph sits in the table, or a
	// negative number where it is not in it.
	CoverageIndex(gid int) int

	// GlyphID returns the glyph at the given index.
	GlyphID(index int) int

	// Size returns how many glyphs the table covers.
	Size() int

	// CoverageFormat returns the format the table is stored in.
	CoverageFormat() int
}

// CoverageTableFormat1 is a coverage table holding a sorted list of glyphs.
//
// Port of org.apache.fontbox.ttf.table.common.CoverageTableFormat1.
type CoverageTableFormat1 struct {
	coverageFormat int
	glyphArray     []int
}

var _ CoverageTable = (*CoverageTableFormat1)(nil)

// NewCoverageTableFormat1 returns a coverage table over the given glyphs.
func NewCoverageTableFormat1(coverageFormat int, glyphArray []int) *CoverageTableFormat1 {
	return &CoverageTableFormat1{coverageFormat: coverageFormat, glyphArray: glyphArray}
}

// CoverageIndex returns where the given glyph sits in the table.
//
// Java uses Arrays.binarySearch, which gives -(insertion point) - 1 for a miss;
// every caller only tests the sign, and the port gives -1 for any miss.
func (t *CoverageTableFormat1) CoverageIndex(gid int) int {
	index := sort.SearchInts(t.glyphArray, gid)
	if index < len(t.glyphArray) && t.glyphArray[index] == gid {
		return index
	}
	return -1
}

// GlyphID returns the glyph at the given index.
func (t *CoverageTableFormat1) GlyphID(index int) int { return t.glyphArray[index] }

// Size returns how many glyphs the table covers.
func (t *CoverageTableFormat1) Size() int { return len(t.glyphArray) }

// CoverageFormat returns the format the table is stored in.
func (t *CoverageTableFormat1) CoverageFormat() int { return t.coverageFormat }

// GlyphArray returns the glyphs the table covers.
func (t *CoverageTableFormat1) GlyphArray() []int { return t.glyphArray }

// String describes the table.
func (t *CoverageTableFormat1) String() string {
	return fmt.Sprintf("CoverageTableFormat1[coverageFormat=%d,glyphArray=%s]",
		t.coverageFormat, intsString(t.glyphArray))
}

// CoverageTableFormat2 is a coverage table holding ranges of glyphs.
//
// Port of org.apache.fontbox.ttf.table.common.CoverageTableFormat2, which
// extends the format 1 table with the ranges expanded into its glyph array.
type CoverageTableFormat2 struct {
	*CoverageTableFormat1

	rangeRecords []RangeRecord
}

var _ CoverageTable = (*CoverageTableFormat2)(nil)

// NewCoverageTableFormat2 returns a coverage table over the given ranges.
func NewCoverageTableFormat2(coverageFormat int, rangeRecords []RangeRecord) *CoverageTableFormat2 {
	return &CoverageTableFormat2{
		CoverageTableFormat1: NewCoverageTableFormat1(coverageFormat,
			rangeRecordsAsArray(rangeRecords)),
		rangeRecords: rangeRecords,
	}
}

// RangeRecords returns the ranges the table covers.
func (t *CoverageTableFormat2) RangeRecords() []RangeRecord { return t.rangeRecords }

func rangeRecordsAsArray(rangeRecords []RangeRecord) []int {
	var glyphIDs []int
	for _, rangeRecord := range rangeRecords {
		for glyphID := rangeRecord.StartGlyphID(); glyphID <= rangeRecord.EndGlyphID(); glyphID++ {
			glyphIDs = append(glyphIDs, glyphID)
		}
	}
	return glyphIDs
}

// String describes the table.
func (t *CoverageTableFormat2) String() string {
	return fmt.Sprintf("CoverageTableFormat2[coverageFormat=%d]", t.CoverageFormat())
}

// RangeRecord is one run of glyphs a format 2 coverage table covers.
//
// Port of org.apache.fontbox.ttf.table.common.RangeRecord.
type RangeRecord struct {
	startGlyphID       int
	endGlyphID         int
	startCoverageIndex int
}

// NewRangeRecord returns a range record.
func NewRangeRecord(startGlyphID, endGlyphID, startCoverageIndex int) RangeRecord {
	return RangeRecord{
		startGlyphID:       startGlyphID,
		endGlyphID:         endGlyphID,
		startCoverageIndex: startCoverageIndex,
	}
}

// StartGlyphID returns the first glyph of the range.
func (r RangeRecord) StartGlyphID() int { return r.startGlyphID }

// EndGlyphID returns the last glyph of the range.
func (r RangeRecord) EndGlyphID() int { return r.endGlyphID }

// StartCoverageIndex returns where the range begins in the coverage.
func (r RangeRecord) StartCoverageIndex() int { return r.startCoverageIndex }

// String describes the record.
func (r RangeRecord) String() string {
	return fmt.Sprintf("RangeRecord[startGlyphID=%d,endGlyphID=%d,startCoverageIndex=%d]",
		r.startGlyphID, r.endGlyphID, r.startCoverageIndex)
}

// LangSysTable is the features one language system uses.
//
// Port of org.apache.fontbox.ttf.table.common.LangSysTable.
type LangSysTable struct {
	lookupOrder          int
	requiredFeatureIndex int
	featureIndexCount    int
	featureIndices       []int
}

// NewLangSysTable returns a language system table.
func NewLangSysTable(lookupOrder, requiredFeatureIndex, featureIndexCount int,
	featureIndices []int) *LangSysTable {
	return &LangSysTable{
		lookupOrder:          lookupOrder,
		requiredFeatureIndex: requiredFeatureIndex,
		featureIndexCount:    featureIndexCount,
		featureIndices:       featureIndices,
	}
}

// LookupOrder returns the lookup order, which the format reserves.
func (t *LangSysTable) LookupOrder() int { return t.lookupOrder }

// RequiredFeatureIndex returns the feature the language system requires.
func (t *LangSysTable) RequiredFeatureIndex() int { return t.requiredFeatureIndex }

// FeatureIndexCount returns how many features the language system uses.
func (t *LangSysTable) FeatureIndexCount() int { return t.featureIndexCount }

// FeatureIndices returns the features the language system uses.
func (t *LangSysTable) FeatureIndices() []int { return t.featureIndices }

// String describes the table.
func (t *LangSysTable) String() string {
	return fmt.Sprintf("LangSysTable[requiredFeatureIndex=%d]", t.requiredFeatureIndex)
}

// ScriptTable is the language systems of one script.
//
// Port of org.apache.fontbox.ttf.table.common.ScriptTable.
type ScriptTable struct {
	defaultLangSysTable *LangSysTable
	langSysTables       map[string]*LangSysTable
}

// NewScriptTable returns a script table.
func NewScriptTable(defaultLangSysTable *LangSysTable,
	langSysTables map[string]*LangSysTable) *ScriptTable {
	return &ScriptTable{defaultLangSysTable: defaultLangSysTable, langSysTables: langSysTables}
}

// DefaultLangSysTable returns the language system used where none is named.
func (t *ScriptTable) DefaultLangSysTable() *LangSysTable { return t.defaultLangSysTable }

// LangSysTables returns the language systems by their tag.
func (t *ScriptTable) LangSysTables() map[string]*LangSysTable { return t.langSysTables }

// String describes the table.
func (t *ScriptTable) String() string {
	return fmt.Sprintf("ScriptTable[hasDefault=%t,langSysRecordsCount=%d]",
		t.defaultLangSysTable != nil, len(t.langSysTables))
}

// FeatureRecord names one feature.
//
// Port of org.apache.fontbox.ttf.table.common.FeatureRecord.
type FeatureRecord struct {
	featureTag   string
	featureTable *FeatureTable
}

// NewFeatureRecord returns a feature record.
func NewFeatureRecord(featureTag string, featureTable *FeatureTable) *FeatureRecord {
	return &FeatureRecord{featureTag: featureTag, featureTable: featureTable}
}

// FeatureTag returns the four-character tag naming the feature.
func (r *FeatureRecord) FeatureTag() string { return r.featureTag }

// FeatureTable returns the feature itself.
func (r *FeatureRecord) FeatureTable() *FeatureTable { return r.featureTable }

// String describes the record.
func (r *FeatureRecord) String() string {
	return fmt.Sprintf("FeatureRecord[featureTag=%s]", r.featureTag)
}

// FeatureTable is the lookups one feature uses.
//
// Port of org.apache.fontbox.ttf.table.common.FeatureTable.
type FeatureTable struct {
	featureParams     int
	lookupIndexCount  int
	lookupListIndices []int
}

// NewFeatureTable returns a feature table.
func NewFeatureTable(featureParams, lookupIndexCount int, lookupListIndices []int) *FeatureTable {
	return &FeatureTable{
		featureParams:     featureParams,
		lookupIndexCount:  lookupIndexCount,
		lookupListIndices: lookupListIndices,
	}
}

// FeatureParams returns the offset to the feature's parameters.
func (t *FeatureTable) FeatureParams() int { return t.featureParams }

// LookupIndexCount returns how many lookups the feature uses.
func (t *FeatureTable) LookupIndexCount() int { return t.lookupIndexCount }

// LookupListIndices returns the lookups the feature uses.
func (t *FeatureTable) LookupListIndices() []int { return t.lookupListIndices }

// String describes the table.
func (t *FeatureTable) String() string {
	return fmt.Sprintf("FeatureTable[lookupIndexCount=%d]", t.lookupIndexCount)
}

// FeatureListTable is every feature of a layout table.
//
// Port of org.apache.fontbox.ttf.table.common.FeatureListTable.
type FeatureListTable struct {
	featureCount   int
	featureRecords []*FeatureRecord
}

// NewFeatureListTable returns a feature list table.
func NewFeatureListTable(featureCount int, featureRecords []*FeatureRecord) *FeatureListTable {
	return &FeatureListTable{featureCount: featureCount, featureRecords: featureRecords}
}

// FeatureCount returns how many features there are.
func (t *FeatureListTable) FeatureCount() int { return t.featureCount }

// FeatureRecords returns the features.
func (t *FeatureListTable) FeatureRecords() []*FeatureRecord { return t.featureRecords }

// String describes the table.
func (t *FeatureListTable) String() string {
	return fmt.Sprintf("FeatureListTable[featureCount=%d]", t.featureCount)
}

// LookupSubTable is one subtable of a lookup.
//
// Port of the abstract org.apache.fontbox.ttf.table.common.LookupSubTable.
type LookupSubTable interface {
	// DoSubstitution returns what the subtable replaces the given glyph with.
	DoSubstitution(gid, coverageIndex int) int

	// SubstFormat returns the format the subtable is stored in.
	SubstFormat() int

	// CoverageTable returns which glyphs the subtable applies to.
	CoverageTable() CoverageTable
}

// LookupSubTableBase is the state every lookup subtable carries, which the
// concrete ones embed.
type LookupSubTableBase struct {
	substFormat   int
	coverageTable CoverageTable
}

// NewLookupSubTableBase returns the shared state of a lookup subtable.
func NewLookupSubTableBase(substFormat int, coverageTable CoverageTable) LookupSubTableBase {
	return LookupSubTableBase{substFormat: substFormat, coverageTable: coverageTable}
}

// SubstFormat returns the format the subtable is stored in.
func (t *LookupSubTableBase) SubstFormat() int { return t.substFormat }

// CoverageTable returns which glyphs the subtable applies to.
func (t *LookupSubTableBase) CoverageTable() CoverageTable { return t.coverageTable }

// LookupTable is one lookup of a layout table.
//
// Port of org.apache.fontbox.ttf.table.common.LookupTable.
type LookupTable struct {
	lookupType       int
	lookupFlag       int
	markFilteringSet int
	subTables        []LookupSubTable
}

// NewLookupTable returns a lookup table.
func NewLookupTable(lookupType, lookupFlag, markFilteringSet int,
	subTables []LookupSubTable) *LookupTable {
	return &LookupTable{
		lookupType:       lookupType,
		lookupFlag:       lookupFlag,
		markFilteringSet: markFilteringSet,
		subTables:        subTables,
	}
}

// LookupType returns what kind of lookup this is.
func (t *LookupTable) LookupType() int { return t.lookupType }

// LookupFlag returns the flags of the lookup.
func (t *LookupTable) LookupFlag() int { return t.lookupFlag }

// MarkFilteringSet returns the mark filtering set of the lookup.
func (t *LookupTable) MarkFilteringSet() int { return t.markFilteringSet }

// SubTables returns the subtables of the lookup.
func (t *LookupTable) SubTables() []LookupSubTable { return t.subTables }

// String describes the table.
func (t *LookupTable) String() string {
	return fmt.Sprintf("LookupTable[lookupType=%d,lookupFlag=%d,markFilteringSet=%d]",
		t.lookupType, t.lookupFlag, t.markFilteringSet)
}

// LookupListTable is every lookup of a layout table.
//
// Port of org.apache.fontbox.ttf.table.common.LookupListTable.
type LookupListTable struct {
	lookupCount int
	lookups     []*LookupTable
}

// NewLookupListTable returns a lookup list table.
func NewLookupListTable(lookupCount int, lookups []*LookupTable) *LookupListTable {
	return &LookupListTable{lookupCount: lookupCount, lookups: lookups}
}

// LookupCount returns how many lookups there are.
func (t *LookupListTable) LookupCount() int { return t.lookupCount }

// Lookups returns the lookups.
func (t *LookupListTable) Lookups() []*LookupTable { return t.lookups }

// String describes the table.
func (t *LookupListTable) String() string {
	return fmt.Sprintf("LookupListTable[lookupCount=%d]", t.lookupCount)
}

// intsString renders an int slice the way Java's Arrays.toString does.
func intsString(values []int) string {
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = fmt.Sprint(value)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
