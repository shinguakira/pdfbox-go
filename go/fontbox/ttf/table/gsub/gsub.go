// Package gsub holds the lookup subtables of the glyph substitution table.
//
// Port of org.apache.fontbox.ttf.table.gsub.
package gsub

import (
	"fmt"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf/table/common"
)

// SequenceTable is the glyphs one multiple substitution replaces a glyph with.
//
// Port of org.apache.fontbox.ttf.table.gsub.SequenceTable.
type SequenceTable struct {
	glyphCount         int
	substituteGlyphIDs []int
}

// NewSequenceTable returns a sequence table.
func NewSequenceTable(glyphCount int, substituteGlyphIDs []int) *SequenceTable {
	return &SequenceTable{glyphCount: glyphCount, substituteGlyphIDs: substituteGlyphIDs}
}

// GlyphCount returns how many glyphs the sequence holds.
func (t *SequenceTable) GlyphCount() int { return t.glyphCount }

// SubstituteGlyphIDs returns the glyphs of the sequence.
func (t *SequenceTable) SubstituteGlyphIDs() []int { return t.substituteGlyphIDs }

// String describes the table.
func (t *SequenceTable) String() string {
	return fmt.Sprintf("SequenceTable{glyphCount=%d, substituteGlyphIDs=%s}",
		t.glyphCount, intsString(t.substituteGlyphIDs))
}

// AlternateSetTable is the glyphs one alternate substitution may choose from.
//
// Port of org.apache.fontbox.ttf.table.gsub.AlternateSetTable.
type AlternateSetTable struct {
	glyphCount        int
	alternateGlyphIDs []int
}

// NewAlternateSetTable returns an alternate set table.
func NewAlternateSetTable(glyphCount int, alternateGlyphIDs []int) *AlternateSetTable {
	return &AlternateSetTable{glyphCount: glyphCount, alternateGlyphIDs: alternateGlyphIDs}
}

// GlyphCount returns how many alternates there are.
func (t *AlternateSetTable) GlyphCount() int { return t.glyphCount }

// AlternateGlyphIDs returns the alternates.
func (t *AlternateSetTable) AlternateGlyphIDs() []int { return t.alternateGlyphIDs }

// String describes the table.
func (t *AlternateSetTable) String() string {
	return fmt.Sprintf("AlternateSetTable{glyphCount=%d, alternateGlyphIDs=%s}",
		t.glyphCount, intsString(t.alternateGlyphIDs))
}

// LigatureTable is one ligature and the glyphs it is made of.
//
// Port of org.apache.fontbox.ttf.table.gsub.LigatureTable.
type LigatureTable struct {
	ligatureGlyph     int
	componentCount    int
	componentGlyphIDs []int
}

// NewLigatureTable returns a ligature table.
func NewLigatureTable(ligatureGlyph, componentCount int, componentGlyphIDs []int) *LigatureTable {
	return &LigatureTable{
		ligatureGlyph:     ligatureGlyph,
		componentCount:    componentCount,
		componentGlyphIDs: componentGlyphIDs,
	}
}

// LigatureGlyph returns the glyph the components make.
func (t *LigatureTable) LigatureGlyph() int { return t.ligatureGlyph }

// ComponentCount returns how many glyphs make the ligature.
func (t *LigatureTable) ComponentCount() int { return t.componentCount }

// ComponentGlyphIDs returns the glyphs after the first that make the ligature.
func (t *LigatureTable) ComponentGlyphIDs() []int { return t.componentGlyphIDs }

// String describes the table.
func (t *LigatureTable) String() string {
	return fmt.Sprintf("LigatureTable[ligatureGlyph=%d, componentCount=%d]",
		t.ligatureGlyph, t.componentCount)
}

// LigatureSetTable is every ligature one glyph begins.
//
// Port of org.apache.fontbox.ttf.table.gsub.LigatureSetTable.
type LigatureSetTable struct {
	ligatureCount  int
	ligatureTables []*LigatureTable
}

// NewLigatureSetTable returns a ligature set table.
func NewLigatureSetTable(ligatureCount int, ligatureTables []*LigatureTable) *LigatureSetTable {
	return &LigatureSetTable{ligatureCount: ligatureCount, ligatureTables: ligatureTables}
}

// LigatureCount returns how many ligatures there are.
func (t *LigatureSetTable) LigatureCount() int { return t.ligatureCount }

// LigatureTables returns the ligatures.
func (t *LigatureSetTable) LigatureTables() []*LigatureTable { return t.ligatureTables }

// String describes the table.
func (t *LigatureSetTable) String() string {
	return fmt.Sprintf("LigatureSetTable[ligatureCount=%d]", t.ligatureCount)
}

// LookupTypeSingleSubstFormat1 replaces a glyph by adding a delta to it.
//
// Port of org.apache.fontbox.ttf.table.gsub.LookupTypeSingleSubstFormat1.
type LookupTypeSingleSubstFormat1 struct {
	common.LookupSubTableBase

	deltaGlyphID int16
}

var _ common.LookupSubTable = (*LookupTypeSingleSubstFormat1)(nil)

// NewLookupTypeSingleSubstFormat1 returns a format 1 single substitution.
func NewLookupTypeSingleSubstFormat1(substFormat int, coverageTable common.CoverageTable,
	deltaGlyphID int16) *LookupTypeSingleSubstFormat1 {
	return &LookupTypeSingleSubstFormat1{
		LookupSubTableBase: common.NewLookupSubTableBase(substFormat, coverageTable),
		deltaGlyphID:       deltaGlyphID,
	}
}

// DoSubstitution returns what the subtable replaces the given glyph with.
func (t *LookupTypeSingleSubstFormat1) DoSubstitution(gid, coverageIndex int) int {
	if coverageIndex < 0 {
		return gid
	}
	return gid + int(t.deltaGlyphID)
}

// DeltaGlyphID returns the delta the substitution adds.
func (t *LookupTypeSingleSubstFormat1) DeltaGlyphID() int16 { return t.deltaGlyphID }

// String describes the table.
func (t *LookupTypeSingleSubstFormat1) String() string {
	return fmt.Sprintf("LookupTypeSingleSubstFormat1[substFormat=%d,deltaGlyphID=%d]",
		t.SubstFormat(), t.deltaGlyphID)
}

// LookupTypeSingleSubstFormat2 replaces a glyph by looking it up in a list.
//
// Port of org.apache.fontbox.ttf.table.gsub.LookupTypeSingleSubstFormat2.
type LookupTypeSingleSubstFormat2 struct {
	common.LookupSubTableBase

	substituteGlyphIDs []int
}

var _ common.LookupSubTable = (*LookupTypeSingleSubstFormat2)(nil)

// NewLookupTypeSingleSubstFormat2 returns a format 2 single substitution.
func NewLookupTypeSingleSubstFormat2(substFormat int, coverageTable common.CoverageTable,
	substituteGlyphIDs []int) *LookupTypeSingleSubstFormat2 {
	return &LookupTypeSingleSubstFormat2{
		LookupSubTableBase: common.NewLookupSubTableBase(substFormat, coverageTable),
		substituteGlyphIDs: substituteGlyphIDs,
	}
}

// DoSubstitution returns what the subtable replaces the given glyph with.
func (t *LookupTypeSingleSubstFormat2) DoSubstitution(gid, coverageIndex int) int {
	if coverageIndex < 0 {
		return gid
	}
	return t.substituteGlyphIDs[coverageIndex]
}

// SubstituteGlyphIDs returns the glyphs the substitution chooses from.
func (t *LookupTypeSingleSubstFormat2) SubstituteGlyphIDs() []int { return t.substituteGlyphIDs }

// String describes the table.
func (t *LookupTypeSingleSubstFormat2) String() string {
	return fmt.Sprintf("LookupTypeSingleSubstFormat2[substFormat=%d,substituteGlyphIDs=%s]",
		t.SubstFormat(), intsString(t.substituteGlyphIDs))
}

// LookupTypeMultipleSubstitutionFormat1 replaces one glyph with several.
//
// Port of
// org.apache.fontbox.ttf.table.gsub.LookupTypeMultipleSubstitutionFormat1.
type LookupTypeMultipleSubstitutionFormat1 struct {
	common.LookupSubTableBase

	sequenceTables []*SequenceTable
}

var _ common.LookupSubTable = (*LookupTypeMultipleSubstitutionFormat1)(nil)

// NewLookupTypeMultipleSubstitutionFormat1 returns a multiple substitution.
func NewLookupTypeMultipleSubstitutionFormat1(substFormat int, coverageTable common.CoverageTable,
	sequenceTables []*SequenceTable) *LookupTypeMultipleSubstitutionFormat1 {
	return &LookupTypeMultipleSubstitutionFormat1{
		LookupSubTableBase: common.NewLookupSubTableBase(substFormat, coverageTable),
		sequenceTables:     sequenceTables,
	}
}

// SequenceTables returns the sequences the substitution chooses from.
func (t *LookupTypeMultipleSubstitutionFormat1) SequenceTables() []*SequenceTable {
	return t.sequenceTables
}

// DoSubstitution is not applicable to a multiple substitution.
//
// Java throws UnsupportedOperationException, which is unchecked.
func (t *LookupTypeMultipleSubstitutionFormat1) DoSubstitution(gid, coverageIndex int) int {
	panic("not applicable")
}

// LookupTypeAlternateSubstitutionFormat1 replaces a glyph with one of several
// alternates.
//
// Port of
// org.apache.fontbox.ttf.table.gsub.LookupTypeAlternateSubstitutionFormat1.
type LookupTypeAlternateSubstitutionFormat1 struct {
	common.LookupSubTableBase

	alternateSetTables []*AlternateSetTable
}

var _ common.LookupSubTable = (*LookupTypeAlternateSubstitutionFormat1)(nil)

// NewLookupTypeAlternateSubstitutionFormat1 returns an alternate substitution.
func NewLookupTypeAlternateSubstitutionFormat1(substFormat int, coverageTable common.CoverageTable,
	alternateSetTables []*AlternateSetTable) *LookupTypeAlternateSubstitutionFormat1 {
	return &LookupTypeAlternateSubstitutionFormat1{
		LookupSubTableBase: common.NewLookupSubTableBase(substFormat, coverageTable),
		alternateSetTables: alternateSetTables,
	}
}

// AlternateSetTables returns the alternates the substitution chooses from.
func (t *LookupTypeAlternateSubstitutionFormat1) AlternateSetTables() []*AlternateSetTable {
	return t.alternateSetTables
}

// DoSubstitution is not applicable to an alternate substitution.
//
// Java throws UnsupportedOperationException, which is unchecked.
func (t *LookupTypeAlternateSubstitutionFormat1) DoSubstitution(gid, coverageIndex int) int {
	panic("not applicable")
}

// LookupTypeLigatureSubstitutionSubstFormat1 replaces several glyphs with one.
//
// Port of
// org.apache.fontbox.ttf.table.gsub.LookupTypeLigatureSubstitutionSubstFormat1.
type LookupTypeLigatureSubstitutionSubstFormat1 struct {
	common.LookupSubTableBase

	ligatureSetTables []*LigatureSetTable
}

var _ common.LookupSubTable = (*LookupTypeLigatureSubstitutionSubstFormat1)(nil)

// NewLookupTypeLigatureSubstitutionSubstFormat1 returns a ligature
// substitution.
func NewLookupTypeLigatureSubstitutionSubstFormat1(substFormat int,
	coverageTable common.CoverageTable,
	ligatureSetTables []*LigatureSetTable) *LookupTypeLigatureSubstitutionSubstFormat1 {
	return &LookupTypeLigatureSubstitutionSubstFormat1{
		LookupSubTableBase: common.NewLookupSubTableBase(substFormat, coverageTable),
		ligatureSetTables:  ligatureSetTables,
	}
}

// DoSubstitution is not applicable to a ligature substitution.
//
// Java throws UnsupportedOperationException, which is unchecked.
func (t *LookupTypeLigatureSubstitutionSubstFormat1) DoSubstitution(gid, coverageIndex int) int {
	panic("not applicable")
}

// LigatureSetTables returns the ligatures the substitution makes.
func (t *LookupTypeLigatureSubstitutionSubstFormat1) LigatureSetTables() []*LigatureSetTable {
	return t.ligatureSetTables
}

// String describes the table.
func (t *LookupTypeLigatureSubstitutionSubstFormat1) String() string {
	return fmt.Sprintf("LookupTypeLigatureSubstitutionSubstFormat1[substFormat=%d]",
		t.SubstFormat())
}

// intsString renders an int slice the way Java's Arrays.toString does.
func intsString(values []int) string {
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = fmt.Sprint(value)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
