// Package gsub turns a font's glyph substitution table into the substitutions
// a script needs, and applies them.
//
// Port of org.apache.fontbox.ttf.gsub.
package gsub

import (
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf/model"
	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf/table/common"
	tablegsub "github.com/shinguakira/pdfbox-go/go/fontbox/ttf/table/gsub"
)

// ScriptList is the scripts of a GSUB table, keyed by their tag and kept in the
// order they were read.
//
// Java uses a LinkedHashMap, whose iteration order GlyphSubstitutionTable's
// selectScriptTag depends on; a Go map has none, so the order travels beside
// it.
type ScriptList struct {
	tables map[string]*common.ScriptTable
	order  []string
}

// NewScriptList returns an empty script list.
func NewScriptList() *ScriptList {
	return &ScriptList{tables: map[string]*common.ScriptTable{}}
}

// Put records the script table under the given tag.
func (l *ScriptList) Put(tag string, table *common.ScriptTable) {
	if _, seen := l.tables[tag]; !seen {
		l.order = append(l.order, tag)
	}
	l.tables[tag] = table
}

// Get returns the script table under the given tag, or nil.
func (l *ScriptList) Get(tag string) *common.ScriptTable { return l.tables[tag] }

// ContainsKey reports whether the list has a table under the given tag.
func (l *ScriptList) ContainsKey(tag string) bool {
	_, ok := l.tables[tag]
	return ok
}

// Tags returns the tags in the order they were read.
func (l *ScriptList) Tags() []string { return l.order }

// Size returns how many scripts the list holds.
func (l *ScriptList) Size() int { return len(l.order) }

// substitutionMap is Java's Map<List<Integer>, List<Integer>>, kept in
// insertion order because Java uses a LinkedHashMap for it.
type substitutionMap struct {
	values map[model.GlyphKey][]int
	order  []model.GlyphKey
}

func newSubstitutionMap() *substitutionMap {
	return &substitutionMap{values: map[model.GlyphKey][]int{}}
}

func (m *substitutionMap) put(key model.GlyphKey, value []int) []int {
	old, seen := m.values[key]
	if !seen {
		m.order = append(m.order, key)
	}
	m.values[key] = value
	if seen {
		return old
	}
	return nil
}

// DataExtractor turns the tables of a GSUB table into GsubData.
//
// Port of org.apache.fontbox.ttf.gsub.GlyphSubstitutionDataExtractor.
type DataExtractor struct{}

// NewDataExtractor returns an extractor.
func NewDataExtractor() *DataExtractor { return &DataExtractor{} }

// scriptTableDetails is which script the data was read for.
type scriptTableDetails struct {
	language    model.Language
	featureName string
	scriptTable *common.ScriptTable
}

// GetGsubData returns the substitution data of the first script the library
// supports, or model.NoDataFound where there is none.
func (e *DataExtractor) GetGsubData(scriptList *ScriptList,
	featureListTable *common.FeatureListTable,
	lookupListTable *common.LookupListTable) model.GsubData {
	details, ok := e.supportedLanguage(scriptList)
	if !ok {
		return model.NoDataFound
	}
	return e.buildMapBackedGsubData(featureListTable, lookupListTable, details)
}

// GetGsubDataForScript returns the substitution data of the named script,
// leaving the language unspecified.
func (e *DataExtractor) GetGsubDataForScript(scriptName string, scriptTable *common.ScriptTable,
	featureListTable *common.FeatureListTable,
	lookupListTable *common.LookupListTable) model.GsubData {
	details := scriptTableDetails{
		language:    model.Unspecified,
		featureName: scriptName,
		scriptTable: scriptTable,
	}
	return e.buildMapBackedGsubData(featureListTable, lookupListTable, details)
}

func (e *DataExtractor) buildMapBackedGsubData(featureListTable *common.FeatureListTable,
	lookupListTable *common.LookupListTable, details scriptTableDetails) *model.MapBackedGsubData {
	scriptTable := details.scriptTable
	gsubData := map[string]map[model.GlyphKey][]int{}
	// the starting point is really the scriptTags
	if scriptTable.DefaultLangSysTable() != nil {
		e.populateGsubDataFromLangSys(gsubData, scriptTable.DefaultLangSysTable(),
			featureListTable, lookupListTable)
	}
	for _, langSysTable := range scriptTable.LangSysTables() {
		e.populateGsubDataFromLangSys(gsubData, langSysTable, featureListTable, lookupListTable)
	}
	return model.NewMapBackedGsubData(details.language, details.featureName, gsubData)
}

func (e *DataExtractor) supportedLanguage(scriptList *ScriptList) (scriptTableDetails, bool) {
	for _, lang := range model.Languages {
		for _, scriptName := range lang.ScriptNames() {
			if value := scriptList.Get(scriptName); value != nil {
				slog.Debug("Language decided", "language", lang, "script", scriptName)
				return scriptTableDetails{
					language:    lang,
					featureName: scriptName,
					scriptTable: value,
				}, true
			}
		}
	}
	return scriptTableDetails{}, false
}

func (e *DataExtractor) populateGsubDataFromLangSys(gsubData map[string]map[model.GlyphKey][]int,
	langSysTable *common.LangSysTable, featureListTable *common.FeatureListTable,
	lookupListTable *common.LookupListTable) {
	featureRecords := featureListTable.FeatureRecords()
	for _, featureIndex := range langSysTable.FeatureIndices() {
		if featureIndex < len(featureRecords) {
			e.populateGsubData(gsubData, featureRecords[featureIndex], lookupListTable)
		}
	}
}

// populateGsubData creates the substitutions of one feature from the lookup
// tables.
func (e *DataExtractor) populateGsubData(gsubData map[string]map[model.GlyphKey][]int,
	featureRecord *common.FeatureRecord, lookupListTable *common.LookupListTable) {
	lookups := lookupListTable.Lookups()
	glyphSubstitutionMap := newSubstitutionMap()
	for _, lookupIndex := range featureRecord.FeatureTable().LookupListIndices() {
		if lookupIndex < len(lookups) {
			e.extractData(glyphSubstitutionMap, lookups[lookupIndex])
		}
	}
	slog.Debug("extracting GSUB data", "feature", featureRecord.FeatureTag())
	gsubData[featureRecord.FeatureTag()] = glyphSubstitutionMap.values
}

func (e *DataExtractor) extractData(glyphSubstitutionMap *substitutionMap,
	lookupTable *common.LookupTable) {
	for _, lookupSubTable := range lookupTable.SubTables() {
		switch subTable := lookupSubTable.(type) {
		case *tablegsub.LookupTypeLigatureSubstitutionSubstFormat1:
			e.extractDataFromLigatureSubstitutionSubstFormat1Table(glyphSubstitutionMap, subTable)
		case *tablegsub.LookupTypeAlternateSubstitutionFormat1:
			e.extractDataFromAlternateSubstitutionSubstFormat1Table(glyphSubstitutionMap, subTable)
		case *tablegsub.LookupTypeSingleSubstFormat1:
			e.extractDataFromSingleSubstTableFormat1Table(glyphSubstitutionMap, subTable)
		case *tablegsub.LookupTypeSingleSubstFormat2:
			e.extractDataFromSingleSubstTableFormat2Table(glyphSubstitutionMap, subTable)
		case *tablegsub.LookupTypeMultipleSubstitutionFormat1:
			e.extractDataFromMultipleSubstitutionFormat1Table(glyphSubstitutionMap, subTable)
		default:
			// usually null, due to being skipped in
			// GlyphSubstitutionTable.readLookupTable()
			slog.Debug("The type is not yet supported, will be ignored",
				"subTable", lookupSubTable)
		}
	}
}

func (e *DataExtractor) extractDataFromSingleSubstTableFormat1Table(
	glyphSubstitutionMap *substitutionMap,
	singleSubstTableFormat1 *tablegsub.LookupTypeSingleSubstFormat1) {
	coverageTable := singleSubstTableFormat1.CoverageTable()
	for i := 0; i < coverageTable.Size(); i++ {
		coverageGlyphID := coverageTable.GlyphID(i)
		substituteGlyphID := coverageGlyphID + int(singleSubstTableFormat1.DeltaGlyphID())
		putNewSubstitutionEntry(glyphSubstitutionMap, []int{substituteGlyphID},
			[]int{coverageGlyphID})
	}
}

func (e *DataExtractor) extractDataFromSingleSubstTableFormat2Table(
	glyphSubstitutionMap *substitutionMap,
	singleSubstTableFormat2 *tablegsub.LookupTypeSingleSubstFormat2) {
	coverageTable := singleSubstTableFormat2.CoverageTable()
	if coverageTable.Size() != len(singleSubstTableFormat2.SubstituteGlyphIDs()) {
		slog.Warn("The coverage table size should be the same as the count of the substituteGlyphIDs tables",
			"coverage", coverageTable.Size(),
			"substituteGlyphIDs", len(singleSubstTableFormat2.SubstituteGlyphIDs()))
		return
	}
	for i := 0; i < coverageTable.Size(); i++ {
		coverageGlyphID := coverageTable.GlyphID(i)
		substituteGlyphID := singleSubstTableFormat2.SubstituteGlyphIDs()[i]
		putNewSubstitutionEntry(glyphSubstitutionMap, []int{substituteGlyphID},
			[]int{coverageGlyphID})
	}
}

func (e *DataExtractor) extractDataFromMultipleSubstitutionFormat1Table(
	glyphSubstitutionMap *substitutionMap,
	multipleSubstFormat1Subtable *tablegsub.LookupTypeMultipleSubstitutionFormat1) {
	coverageTable := multipleSubstFormat1Subtable.CoverageTable()
	if coverageTable.Size() != len(multipleSubstFormat1Subtable.SequenceTables()) {
		slog.Warn("The coverage table size should be the same as the count of the sequence tables",
			"coverage", coverageTable.Size(),
			"sequenceTables", len(multipleSubstFormat1Subtable.SequenceTables()))
		return
	}
	for i := 0; i < coverageTable.Size(); i++ {
		coverageGlyphID := coverageTable.GlyphID(i)
		sequenceTable := multipleSubstFormat1Subtable.SequenceTables()[i]
		substituteGlyphIDList := make([]int, len(sequenceTable.SubstituteGlyphIDs()))
		copy(substituteGlyphIDList, sequenceTable.SubstituteGlyphIDs())
		putNewSubstitutionEntry(glyphSubstitutionMap, substituteGlyphIDList,
			[]int{coverageGlyphID})
	}
}

func (e *DataExtractor) extractDataFromLigatureSubstitutionSubstFormat1Table(
	glyphSubstitutionMap *substitutionMap,
	ligatureSubstitutionTable *tablegsub.LookupTypeLigatureSubstitutionSubstFormat1) {
	for _, ligatureSetTable := range ligatureSubstitutionTable.LigatureSetTables() {
		for _, ligatureTable := range ligatureSetTable.LigatureTables() {
			e.extractDataFromLigatureTable(glyphSubstitutionMap, ligatureTable)
		}
	}
}

func (e *DataExtractor) extractDataFromAlternateSubstitutionSubstFormat1Table(
	glyphSubstitutionMap *substitutionMap,
	alternateSubstitutionFormat1 *tablegsub.LookupTypeAlternateSubstitutionFormat1) {
	coverageTable := alternateSubstitutionFormat1.CoverageTable()
	if coverageTable.Size() != len(alternateSubstitutionFormat1.AlternateSetTables()) {
		slog.Warn("The coverage table size should be the same as the count of the alternate set tables",
			"coverage", coverageTable.Size(),
			"alternateSetTables", len(alternateSubstitutionFormat1.AlternateSetTables()))
		return
	}
	for i := 0; i < coverageTable.Size(); i++ {
		coverageGlyphID := coverageTable.GlyphID(i)
		sequenceTable := alternateSubstitutionFormat1.AlternateSetTables()[i]
		// Loop through the substitute glyphs and pick the first one that is not
		// the same as the coverage glyph
		for _, alternateGlyphID := range sequenceTable.AlternateGlyphIDs() {
			if alternateGlyphID != coverageGlyphID {
				putNewSubstitutionEntry(glyphSubstitutionMap, []int{alternateGlyphID},
					[]int{coverageGlyphID})
				break
			}
		}
	}
}

func (e *DataExtractor) extractDataFromLigatureTable(glyphSubstitutionMap *substitutionMap,
	ligatureTable *tablegsub.LigatureTable) {
	componentGlyphIDs := ligatureTable.ComponentGlyphIDs()
	glyphsToBeSubstituted := make([]int, len(componentGlyphIDs))
	copy(glyphsToBeSubstituted, componentGlyphIDs)
	putNewSubstitutionEntry(glyphSubstitutionMap, []int{ligatureTable.LigatureGlyph()},
		glyphsToBeSubstituted)
}

func putNewSubstitutionEntry(glyphSubstitutionMap *substitutionMap,
	newGlyphList, glyphsToBeSubstituted []int) {
	glyphSubstitutionMap.put(model.NewGlyphKey(glyphsToBeSubstituted), newGlyphList)
}
