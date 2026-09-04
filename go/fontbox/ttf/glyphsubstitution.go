package ttf

import (
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"slices"
	"sort"
	"strings"

	gsubpkg "github.com/shinguakira/pdfbox-go/go/fontbox/ttf/gsub"
	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf/model"
	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf/table/common"
	tablegsub "github.com/shinguakira/pdfbox-go/go/fontbox/ttf/table/gsub"
)

// is4CharWord is 4 'word characters' [a-zA-Z_0-9], matching Java's \w{4}.
//
// Note: the ' '-character is not matched.
var is4CharWord = regexp.MustCompile(`^\w{4}$`)

// GlyphSubstitutionTable is a glyph substitution 'GSUB' table in a TrueType or
// OpenType font.
//
// Port of org.apache.fontbox.ttf.GlyphSubstitutionTable.
type GlyphSubstitutionTable struct {
	Table

	scriptList *gsubpkg.ScriptList
	// featureListTable and lookupListTable are not maps because we need to
	// index into them
	featureListTable *common.FeatureListTable
	lookupListTable  *common.LookupListTable

	lookupCache   map[int]int
	reverseLookup map[int]int

	lastUsedSupportedScript string
	hasLastUsedScript       bool

	gsubData model.GsubData
}

var _ TableReader = (*GlyphSubstitutionTable)(nil)

// Read reads the required data from the stream.
func (t *GlyphSubstitutionTable) Read(ttf *TrueTypeFont, data DataStream) error {
	t.lookupCache = map[int]int{}
	t.reverseLookup = map[int]int{}

	start := data.CurrentPosition()
	r := newReader(data)
	_ = r.unsignedShort() // majorVersion
	minorVersion := r.unsignedShort()
	scriptListOffset := r.unsignedShort()
	featureListOffset := r.unsignedShort()
	lookupListOffset := r.unsignedShort()
	if r.err != nil {
		return r.err
	}
	if minorVersion == 1 {
		// featureVariationsOffset, not used
		if _ = r.unsignedInt(); r.err != nil {
			return r.err
		}
	}

	var err error
	if t.scriptList, err = t.readScriptList(data, start+int64(scriptListOffset)); err != nil {
		return err
	}
	if t.featureListTable, err = t.readFeatureList(data, start+int64(featureListOffset)); err != nil {
		return err
	}
	if lookupListOffset > 0 {
		if t.lookupListTable, err = t.readLookupList(data, start+int64(lookupListOffset)); err != nil {
			return err
		}
	} else {
		// happened with NotoSansNewTaiLue-Regular.ttf in
		// noto-fonts-20201206-phase3.zip
		slog.Warn("lookupListOffset is 0, LookupListTable is considered empty")
		t.lookupListTable = common.NewLookupListTable(0, nil)
	}

	// PDFBOX-5729: for debugging only
	lookupTable := t.lookupListTable.Lookups()
	for _, rec := range t.featureListTable.FeatureRecords() {
		tab := rec.FeatureTable()
		tag := rec.FeatureTag()
		indices := tab.LookupListIndices()
		for i, idx := range indices {
			if idx < 0 || idx >= len(lookupTable) {
				slog.Debug("LookupListIndex invalid", "i", i, "index", idx, "tag", tag,
					"lookupTable length", len(lookupTable))
				break
			}
			lookupType := lookupTable[idx].LookupType()

			lst := lookupTable[indices[i]].SubTables()
			if len(lst) == 0 || lst[0] == nil {
				slog.Debug("GSUB feature unavailable", "lookupType", lookupType, "tag", tag,
					"index", indices[i])
			}
		}
	}

	t.gsubData = gsubpkg.NewDataExtractor().GetGsubData(t.scriptList, t.featureListTable,
		t.lookupListTable)

	t.SetInitialized(true)
	return nil
}

func (t *GlyphSubstitutionTable) readScriptList(data DataStream,
	offset int64) (*gsubpkg.ScriptList, error) {
	if err := data.SeekTo(offset); err != nil {
		return nil, err
	}
	r := newReader(data)
	scriptCount := r.unsignedShort()
	if r.err != nil {
		return nil, r.err
	}
	scriptOffsets := make([]int, scriptCount)
	scriptTags := make([]string, scriptCount)
	resultScriptList := gsubpkg.NewScriptList()
	for i := 0; i < scriptCount; i++ {
		scriptTags[i] = r.str(4)
		scriptOffsets[i] = r.unsignedShort()
		if r.err != nil {
			return nil, r.err
		}
		if int64(scriptOffsets[i]) < data.CurrentPosition()-offset {
			// can't be before the current position
			slog.Error("scriptOffsets implausible", "i", i, "offset", scriptOffsets[i],
				"position - offset", data.CurrentPosition()-offset)
			return resultScriptList, nil
		}
	}
	for i := 0; i < scriptCount; i++ {
		if resultScriptList.Get(scriptTags[i]) != nil {
			// PDFBOX-6146
			continue
		}
		scriptTable, err := t.readScriptTable(data, offset+int64(scriptOffsets[i]))
		if err != nil {
			return nil, err
		}
		resultScriptList.Put(scriptTags[i], scriptTable)
	}
	return resultScriptList, nil
}

func (t *GlyphSubstitutionTable) readScriptTable(data DataStream,
	offset int64) (*common.ScriptTable, error) {
	if err := data.SeekTo(offset); err != nil {
		return nil, err
	}
	r := newReader(data)
	defaultLangSysOffset := r.unsignedShort()
	langSysCount := r.unsignedShort()
	if r.err != nil {
		return nil, r.err
	}
	langSysTags := make([]string, langSysCount)
	langSysOffsets := make([]int, langSysCount)
	for i := 0; i < langSysCount; i++ {
		langSysTags[i] = r.str(4)
		langSysOffsets[i] = r.unsignedShort()
		if r.err != nil {
			return nil, r.err
		}
		if int64(langSysOffsets[i]) < data.CurrentPosition()-offset {
			// can't be before the current position
			slog.Error("langSysOffsets implausible", "i", i, "offset", langSysOffsets[i],
				"position - offset", data.CurrentPosition()-offset)
			return common.NewScriptTable(nil, map[string]*common.LangSysTable{}), nil
		}
		if i > 0 && langSysTags[i] < langSysTags[i-1] {
			// PDFBOX-4489: catch corrupt file
			// https://docs.microsoft.com/en-us/typography/opentype/spec/chapter2#slTbl_sRec
			slog.Error("LangSysRecords not alphabetically sorted by LangSys tag",
				"this", langSysTags[i], "previous", langSysTags[i-1])
			return common.NewScriptTable(nil, map[string]*common.LangSysTable{}), nil
		}
	}

	var defaultLangSysTable *common.LangSysTable
	var err error
	if defaultLangSysOffset != 0 {
		if defaultLangSysTable, err = t.readLangSysTable(data,
			offset+int64(defaultLangSysOffset)); err != nil {
			return nil, err
		}
	}
	langSysTables := make(map[string]*common.LangSysTable, langSysCount)
	for i := 0; i < langSysCount; i++ {
		langSysTable, err := t.readLangSysTable(data, offset+int64(langSysOffsets[i]))
		if err != nil {
			return nil, err
		}
		langSysTables[langSysTags[i]] = langSysTable
	}
	return common.NewScriptTable(defaultLangSysTable, langSysTables), nil
}

func (t *GlyphSubstitutionTable) readLangSysTable(data DataStream,
	offset int64) (*common.LangSysTable, error) {
	if err := data.SeekTo(offset); err != nil {
		return nil, err
	}
	r := newReader(data)
	lookupOrder := r.unsignedShort()
	requiredFeatureIndex := r.unsignedShort()
	featureIndexCount := r.unsignedShort()
	if r.err != nil {
		return nil, r.err
	}
	featureIndices := make([]int, featureIndexCount)
	for i := 0; i < featureIndexCount; i++ {
		featureIndices[i] = r.unsignedShort()
	}
	if r.err != nil {
		return nil, r.err
	}
	return common.NewLangSysTable(lookupOrder, requiredFeatureIndex, featureIndexCount,
		featureIndices), nil
}

func (t *GlyphSubstitutionTable) readFeatureList(data DataStream,
	offset int64) (*common.FeatureListTable, error) {
	if err := data.SeekTo(offset); err != nil {
		return nil, err
	}
	r := newReader(data)
	featureCount := r.unsignedShort()
	if r.err != nil {
		return nil, r.err
	}
	featureRecords := make([]*common.FeatureRecord, featureCount)
	featureOffsets := make([]int, featureCount)
	featureTags := make([]string, featureCount)
	for i := 0; i < featureCount; i++ {
		featureTags[i] = r.str(4)
		if r.err != nil {
			return nil, r.err
		}
		if i > 0 && featureTags[i] < featureTags[i-1] {
			// catch corrupt file
			// https://docs.microsoft.com/en-us/typography/opentype/spec/chapter2#flTbl
			if is4CharWord.MatchString(featureTags[i]) && is4CharWord.MatchString(featureTags[i-1]) {
				// ArialUni.ttf has many warnings but isn't corrupt, so we assume
				// that only strings with trash characters indicate real
				// corruption
				slog.Debug("FeatureRecord array not alphabetically sorted by FeatureTag",
					"this", featureTags[i], "previous", featureTags[i-1])
			} else {
				slog.Warn("FeatureRecord array not alphabetically sorted by FeatureTag",
					"this", featureTags[i], "previous", featureTags[i-1])
				return common.NewFeatureListTable(0, nil), nil
			}
		}
		featureOffsets[i] = r.unsignedShort()
		if r.err != nil {
			return nil, r.err
		}
	}
	for i := 0; i < featureCount; i++ {
		featureTable, err := t.readFeatureTable(data, offset+int64(featureOffsets[i]))
		if err != nil {
			return nil, err
		}
		featureRecords[i] = common.NewFeatureRecord(featureTags[i], featureTable)
	}
	return common.NewFeatureListTable(featureCount, featureRecords), nil
}

func (t *GlyphSubstitutionTable) readFeatureTable(data DataStream,
	offset int64) (*common.FeatureTable, error) {
	if err := data.SeekTo(offset); err != nil {
		return nil, err
	}
	r := newReader(data)
	featureParams := r.unsignedShort()
	lookupIndexCount := r.unsignedShort()
	if r.err != nil {
		return nil, r.err
	}
	lookupListIndices := make([]int, lookupIndexCount)
	for i := 0; i < lookupIndexCount; i++ {
		lookupListIndices[i] = r.unsignedShort()
	}
	if r.err != nil {
		return nil, r.err
	}
	return common.NewFeatureTable(featureParams, lookupIndexCount, lookupListIndices), nil
}

func (t *GlyphSubstitutionTable) readLookupList(data DataStream,
	offset int64) (*common.LookupListTable, error) {
	if err := data.SeekTo(offset); err != nil {
		return nil, err
	}
	r := newReader(data)
	lookupCount := r.unsignedShort()
	if r.err != nil {
		return nil, r.err
	}
	lookups := make([]int, lookupCount)
	for i := 0; i < lookupCount; i++ {
		lookups[i] = r.unsignedShort()
		if r.err != nil {
			return nil, r.err
		}
		if lookups[i] == 0 {
			// no early return here and in the other one; if we do, the file
			// from PDFBOX-6066 no longer renders properly.
			slog.Error("lookups is 0", "i", i, "offset", data.CurrentPosition()-2)
		} else if offset+int64(lookups[i]) > data.OriginalDataSize() {
			slog.Error("lookup past the end of the data",
				"offset", offset+int64(lookups[i]), "size", data.OriginalDataSize())
		}
	}
	lookupTables := make([]*common.LookupTable, lookupCount)
	lookupTableMap := map[int]*common.LookupTable{} // PDFBOX-6146
	for i := 0; i < lookupCount; i++ {
		lookupTable, ok := lookupTableMap[lookups[i]]
		if !ok {
			var err error
			if lookupTable, err = t.readLookupTable(data, offset+int64(lookups[i])); err != nil {
				return nil, err
			}
			lookupTableMap[lookups[i]] = lookupTable
		}
		lookupTables[i] = lookupTable
	}
	return common.NewLookupListTable(lookupCount, lookupTables), nil
}

func (t *GlyphSubstitutionTable) readLookupSubtable(data DataStream, offset int64,
	lookupType int) (common.LookupSubTable, error) {
	switch lookupType {
	case 1:
		// Single Substitution Subtable
		// https://docs.microsoft.com/en-us/typography/opentype/spec/gsub#SS
		return t.readSingleLookupSubTable(data, offset)
	case 2:
		// Multiple Substitution Subtable
		// https://learn.microsoft.com/en-us/typography/opentype/spec/gsub#lookuptype-2-multiple-substitution-subtable
		return t.readMultipleSubstitutionSubtable(data, offset)
	case 3:
		// Alternate Substitution Subtable
		// https://learn.microsoft.com/en-us/typography/opentype/spec/gsub#lookuptype-3-alternate-substitution-subtable
		return t.readAlternateSubstitutionSubtable(data, offset)
	case 4:
		// Ligature Substitution Subtable
		// https://docs.microsoft.com/en-us/typography/opentype/spec/gsub#LS
		return t.readLigatureSubstitutionSubtable(data, offset)

		// when creating a new LookupSubTable derived type, don't forget to add
		// a "switch" in readLookupTable() and add the type in
		// GlyphSubstitutionDataExtractor.extractData()

	default:
		// Other lookup types are not supported
		slog.Debug("GSUB lookup table is not supported and will be ignored",
			"lookupType", lookupType)
		return nil, nil
		// TODO next: implement type 6
		// https://learn.microsoft.com/en-us/typography/opentype/spec/gsub#lookuptype-6-chained-contexts-substitution-subtable
		// see e.g. readChainedContextualSubTable in Apache FOP
	}
}

// readLookupTable reads one lookup table.
//
// https://learn.microsoft.com/en-us/typography/opentype/spec/chapter2#lookup-table
func (t *GlyphSubstitutionTable) readLookupTable(data DataStream,
	offset int64) (*common.LookupTable, error) {
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
	subTableOffsets := make([]int, subTableCount)
	for i := 0; i < subTableCount; i++ {
		subTableOffsets[i] = r.unsignedShort()
		if r.err != nil {
			return nil, r.err
		}
		if subTableOffsets[i] == 0 {
			slog.Error("subTableOffsets is 0", "i", i, "offset", data.CurrentPosition()-2)
			return common.NewLookupTable(lookupType, lookupFlag, 0, nil), nil
		}
		if offset+int64(subTableOffsets[i]) > data.OriginalDataSize() {
			slog.Error("subtable past the end of the data",
				"offset", offset+int64(subTableOffsets[i]), "size", data.OriginalDataSize())
			return common.NewLookupTable(lookupType, lookupFlag, 0, nil), nil
		}
	}

	markFilteringSet := 0
	if lookupFlag&0x0010 != 0 {
		markFilteringSet = r.unsignedShort()
		if r.err != nil {
			return nil, r.err
		}
	}
	subTables := make([]common.LookupSubTable, subTableCount)
	switch lookupType {
	case 1, 2, 3, 4:
		for i := 0; i < subTableCount; i++ {
			subTable, err := t.readLookupSubtable(data, offset+int64(subTableOffsets[i]), lookupType)
			if err != nil {
				return nil, err
			}
			if subTable != nil {
				subTables[i] = subTable
			}
		}
	case 7:
		// Extension Substitution
		// https://learn.microsoft.com/en-us/typography/opentype/spec/gsub#ES
		for i := 0; i < subTableCount; i++ {
			if err := data.SeekTo(offset + int64(subTableOffsets[i])); err != nil {
				return nil, err
			}
			r := newReader(data)
			substFormat := r.unsignedShort() // always 1
			if r.err != nil {
				return nil, r.err
			}
			if substFormat != 1 {
				slog.Error("The expected SubstFormat for ExtensionSubstFormat1 subtable should be 1",
					"substFormat", substFormat, "offset", offset+int64(subTableOffsets[i]))
				continue
			}
			extensionLookupType := r.unsignedShort()
			if r.err != nil {
				return nil, r.err
			}
			if lookupType != 7 && lookupType != extensionLookupType {
				// "If a lookup table uses extension subtables, then all of the
				//  extension subtables must have the same extensionLookupType"
				slog.Error("extensionLookupType changed", "from", lookupType,
					"to", extensionLookupType, "offset", offset+int64(subTableOffsets[i])+2)
				continue
			}
			lookupType = extensionLookupType
			extensionOffset := r.unsignedInt()
			if r.err != nil {
				return nil, r.err
			}
			extensionLookupTableAddress := offset + int64(subTableOffsets[i]) + extensionOffset
			subTable, err := t.readLookupSubtable(data, extensionLookupTableAddress,
				extensionLookupType)
			if err != nil {
				return nil, err
			}
			if subTable != nil {
				subTables[i] = subTable
			}
		}
	default:
		// Other lookup types are not supported
		slog.Debug("GSUB lookup table is not supported and will be ignored",
			"lookupType", lookupType)
	}
	return common.NewLookupTable(lookupType, lookupFlag, markFilteringSet, subTables), nil
}

func (t *GlyphSubstitutionTable) readSingleLookupSubTable(data DataStream,
	offset int64) (common.LookupSubTable, error) {
	if err := data.SeekTo(offset); err != nil {
		return nil, err
	}
	r := newReader(data)
	substFormat := r.unsignedShort()
	if r.err != nil {
		return nil, r.err
	}
	switch substFormat {
	case 1:
		// LookupType 1: Single Substitution Subtable
		// https://docs.microsoft.com/en-us/typography/opentype/spec/gsub#11-single-substitution-format-1
		coverageOffset := r.unsignedShort()
		deltaGlyphID := r.signedShort()
		if r.err != nil {
			return nil, r.err
		}
		coverageTable, err := t.readCoverageTable(data, offset+int64(coverageOffset))
		if err != nil {
			return nil, err
		}
		return tablegsub.NewLookupTypeSingleSubstFormat1(substFormat, coverageTable,
			deltaGlyphID), nil
	case 2:
		// Single Substitution Format 2
		// https://docs.microsoft.com/en-us/typography/opentype/spec/gsub#12-single-substitution-format-2
		coverageOffset := r.unsignedShort()
		glyphCount := r.unsignedShort()
		if r.err != nil {
			return nil, r.err
		}
		substituteGlyphIDs := make([]int, glyphCount)
		for i := 0; i < glyphCount; i++ {
			substituteGlyphIDs[i] = r.unsignedShort()
		}
		if r.err != nil {
			return nil, r.err
		}
		coverageTable, err := t.readCoverageTable(data, offset+int64(coverageOffset))
		if err != nil {
			return nil, err
		}
		return tablegsub.NewLookupTypeSingleSubstFormat2(substFormat, coverageTable,
			substituteGlyphIDs), nil
	default:
		slog.Warn("Unknown substFormat", "substFormat", substFormat)
		return nil, nil
	}
}

func (t *GlyphSubstitutionTable) readMultipleSubstitutionSubtable(data DataStream,
	offset int64) (common.LookupSubTable, error) {
	if err := data.SeekTo(offset); err != nil {
		return nil, err
	}
	r := newReader(data)
	substFormat := r.unsignedShort()
	if r.err != nil {
		return nil, r.err
	}
	if substFormat != 1 {
		return nil, errors.New("The expected SubstFormat for LigatureSubstitutionTable is 1")
	}

	coverage := r.unsignedShort()
	sequenceCount := r.unsignedShort()
	if r.err != nil {
		return nil, r.err
	}
	sequenceOffsets := make([]int, sequenceCount)
	for i := 0; i < sequenceCount; i++ {
		sequenceOffsets[i] = r.unsignedShort()
	}
	if r.err != nil {
		return nil, r.err
	}

	coverageTable, err := t.readCoverageTable(data, offset+int64(coverage))
	if err != nil {
		return nil, err
	}

	if sequenceCount != coverageTable.Size() {
		return nil, errors.New("According to the OpenTypeFont specifications, the coverage " +
			"count should be equal to the no. of SequenceTables")
	}

	sequenceTables := make([]*tablegsub.SequenceTable, sequenceCount)
	for i := 0; i < sequenceCount; i++ {
		if err := data.SeekTo(offset + int64(sequenceOffsets[i])); err != nil {
			return nil, err
		}
		r := newReader(data)
		glyphCount := r.unsignedShort()
		substituteGlyphIDs := r.unsignedShortArray(glyphCount)
		if r.err != nil {
			return nil, r.err
		}
		sequenceTables[i] = tablegsub.NewSequenceTable(glyphCount, substituteGlyphIDs)
	}

	return tablegsub.NewLookupTypeMultipleSubstitutionFormat1(substFormat, coverageTable,
		sequenceTables), nil
}

func (t *GlyphSubstitutionTable) readAlternateSubstitutionSubtable(data DataStream,
	offset int64) (common.LookupSubTable, error) {
	if err := data.SeekTo(offset); err != nil {
		return nil, err
	}
	r := newReader(data)
	substFormat := r.unsignedShort()
	if r.err != nil {
		return nil, r.err
	}
	if substFormat != 1 {
		return nil, errors.New("The expected SubstFormat for AlternateSubstitutionTable is 1")
	}

	coverage := r.unsignedShort()
	altSetCount := r.unsignedShort()
	if r.err != nil {
		return nil, r.err
	}

	alternateOffsets := make([]int, altSetCount)
	for i := 0; i < altSetCount; i++ {
		alternateOffsets[i] = r.unsignedShort()
	}
	if r.err != nil {
		return nil, r.err
	}

	coverageTable, err := t.readCoverageTable(data, offset+int64(coverage))
	if err != nil {
		return nil, err
	}

	if altSetCount != coverageTable.Size() {
		return nil, errors.New("According to the OpenTypeFont specifications, the coverage " +
			"count should be equal to the no. of AlternateSetTable")
	}

	alternateSetTables := make([]*tablegsub.AlternateSetTable, altSetCount)
	for i := 0; i < altSetCount; i++ {
		if err := data.SeekTo(offset + int64(alternateOffsets[i])); err != nil {
			return nil, err
		}
		r := newReader(data)
		glyphCount := r.unsignedShort()
		alternateGlyphIDs := r.unsignedShortArray(glyphCount)
		if r.err != nil {
			return nil, r.err
		}
		alternateSetTables[i] = tablegsub.NewAlternateSetTable(glyphCount, alternateGlyphIDs)
	}

	return tablegsub.NewLookupTypeAlternateSubstitutionFormat1(substFormat, coverageTable,
		alternateSetTables), nil
}

func (t *GlyphSubstitutionTable) readLigatureSubstitutionSubtable(data DataStream,
	offset int64) (common.LookupSubTable, error) {
	if err := data.SeekTo(offset); err != nil {
		return nil, err
	}
	r := newReader(data)
	substFormat := r.unsignedShort()
	if r.err != nil {
		return nil, r.err
	}
	if substFormat != 1 {
		return nil, errors.New("The expected SubstFormat for LigatureSubstitutionTable is 1")
	}

	coverage := r.unsignedShort()
	ligSetCount := r.unsignedShort()
	if r.err != nil {
		return nil, r.err
	}

	ligatureOffsets := make([]int, ligSetCount)
	for i := 0; i < ligSetCount; i++ {
		ligatureOffsets[i] = r.unsignedShort()
	}
	if r.err != nil {
		return nil, r.err
	}

	coverageTable, err := t.readCoverageTable(data, offset+int64(coverage))
	if err != nil {
		return nil, err
	}

	if ligSetCount != coverageTable.Size() {
		return nil, errors.New("According to the OpenTypeFont specifications, the coverage " +
			"count should be equal to the no. of LigatureSetTables")
	}

	ligatureSetTables := make([]*tablegsub.LigatureSetTable, ligSetCount)
	for i := 0; i < ligSetCount; i++ {
		coverageGlyphID := coverageTable.GlyphID(i)
		if ligatureSetTables[i], err = t.readLigatureSetTable(data,
			offset+int64(ligatureOffsets[i]), coverageGlyphID); err != nil {
			return nil, err
		}
	}

	return tablegsub.NewLookupTypeLigatureSubstitutionSubstFormat1(substFormat, coverageTable,
		ligatureSetTables), nil
}

func (t *GlyphSubstitutionTable) readLigatureSetTable(data DataStream,
	ligatureSetTableLocation int64, coverageGlyphID int) (*tablegsub.LigatureSetTable, error) {
	if err := data.SeekTo(ligatureSetTableLocation); err != nil {
		return nil, err
	}
	r := newReader(data)
	ligatureCount := r.unsignedShort()
	if r.err != nil {
		return nil, r.err
	}

	ligatureOffsets := make([]int, ligatureCount)
	ligatureTables := make([]*tablegsub.LigatureTable, ligatureCount)

	for i := range ligatureOffsets {
		ligatureOffsets[i] = r.unsignedShort()
	}
	if r.err != nil {
		return nil, r.err
	}

	for i, ligatureOffset := range ligatureOffsets {
		table, err := t.readLigatureTable(data,
			ligatureSetTableLocation+int64(ligatureOffset), coverageGlyphID)
		if err != nil {
			return nil, err
		}
		ligatureTables[i] = table
	}

	return tablegsub.NewLigatureSetTable(ligatureCount, ligatureTables), nil
}

func (t *GlyphSubstitutionTable) readLigatureTable(data DataStream, ligatureTableLocation int64,
	coverageGlyphID int) (*tablegsub.LigatureTable, error) {
	if err := data.SeekTo(ligatureTableLocation); err != nil {
		return nil, err
	}
	r := newReader(data)
	ligatureGlyph := r.unsignedShort()

	componentCount := r.unsignedShort()
	if r.err != nil {
		return nil, r.err
	}
	if componentCount > 100 {
		return nil, fmt.Errorf("componentCount in ligature table is %d, font likely corrupt",
			componentCount)
	}

	componentGlyphIDs := make([]int, componentCount)

	if componentCount > 0 {
		componentGlyphIDs[0] = coverageGlyphID
	}

	for i := 1; i <= componentCount-1; i++ {
		componentGlyphIDs[i] = r.unsignedShort()
	}
	if r.err != nil {
		return nil, r.err
	}

	return tablegsub.NewLigatureTable(ligatureGlyph, componentCount, componentGlyphIDs), nil
}

func (t *GlyphSubstitutionTable) readCoverageTable(data DataStream,
	offset int64) (common.CoverageTable, error) {
	if err := data.SeekTo(offset); err != nil {
		return nil, err
	}
	r := newReader(data)
	coverageFormat := r.unsignedShort()
	if r.err != nil {
		return nil, r.err
	}
	switch coverageFormat {
	case 1:
		glyphCount := r.unsignedShort()
		if r.err != nil {
			return nil, r.err
		}
		glyphArray := make([]int, glyphCount)
		for i := 0; i < glyphCount; i++ {
			glyphArray[i] = r.unsignedShort()
		}
		if r.err != nil {
			return nil, r.err
		}
		return common.NewCoverageTableFormat1(coverageFormat, glyphArray), nil
	case 2:
		rangeCount := r.unsignedShort()
		if r.err != nil {
			return nil, r.err
		}
		rangeRecords := make([]common.RangeRecord, rangeCount)
		for i := 0; i < rangeCount; i++ {
			record, err := readRangeRecord(data)
			if err != nil {
				return nil, err
			}
			rangeRecords[i] = record
		}
		return common.NewCoverageTableFormat2(coverageFormat, rangeRecords), nil
	default:
		// Should not happen (the spec indicates only format 1 and format 2)
		return nil, fmt.Errorf("Unknown coverage format: %d", coverageFormat)
	}
}

// selectScriptTag chooses from one of the supplied OpenType script tags,
// depending on what the font supports and potentially on context.
func (t *GlyphSubstitutionTable) selectScriptTag(tags []string) string {
	if len(tags) == 1 {
		tag := tags[0]
		if tag == ScriptInherited || (tag == TagDefault && !t.scriptList.ContainsKey(tag)) {
			// We don't know what script this should be.
			if !t.hasLastUsedScript {
				// We have no past context and (currently) no way to get future
				// context so we guess.
				if t.scriptList.Size() == 0 {
					// Java's iterator().next() throws NoSuchElementException on
					// an empty script list, which is unchecked.
					panic("gsub: no scripts in the GSUB table")
				}
				t.lastUsedSupportedScript = t.scriptList.Tags()[0]
				t.hasLastUsedScript = true
			}
			// else use past context

			return t.lastUsedSupportedScript
		}
	}
	for _, tag := range tags {
		if t.scriptList.ContainsKey(tag) {
			// Use the first recognized tag. We assume a single font only
			// recognizes one version ("ver. 2") of a single script, or if it
			// recognizes more than one that it prefers the latest one.
			t.lastUsedSupportedScript = tag
			t.hasLastUsedScript = true
			return t.lastUsedSupportedScript
		}
	}
	return tags[0]
}

func (t *GlyphSubstitutionTable) langSysTables(scriptTag string) []*common.LangSysTable {
	scriptTable := t.scriptList.Get(scriptTag)
	if scriptTable == nil {
		return nil
	}
	// Java iterates the LangSysTables map, whose order a HashMap does not
	// define; the port sorts by tag so that the walk is repeatable.
	tags := make([]string, 0, len(scriptTable.LangSysTables()))
	for tag := range scriptTable.LangSysTables() {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	result := make([]*common.LangSysTable, 0, len(tags)+1)
	for _, tag := range tags {
		result = append(result, scriptTable.LangSysTables()[tag])
	}
	if scriptTable.DefaultLangSysTable() != nil {
		result = append(result, scriptTable.DefaultLangSysTable())
	}
	return result
}

// featureRecords gets the FeatureRecords the given LangSysTables indicate,
// optionally filtered by the list of allowed feature tags in enabledFeatures.
//
// Note that features listed as required are included even if not explicitly
// enabled.
func (t *GlyphSubstitutionTable) featureRecords(langSysTables []*common.LangSysTable,
	enabledFeatures []string) []*common.FeatureRecord {
	if len(langSysTables) == 0 {
		return nil
	}
	var result []*common.FeatureRecord
	for _, langSysTable := range langSysTables {
		required := langSysTable.RequiredFeatureIndex()
		featureRecords := t.featureListTable.FeatureRecords()
		// if no required features = 0xFFFF
		if required != 0xffff && required < len(featureRecords) {
			result = append(result, featureRecords[required])
		}
		for _, featureIndex := range langSysTable.FeatureIndices() {
			if featureIndex < len(featureRecords) &&
				(enabledFeatures == nil ||
					slices.Contains(enabledFeatures, featureRecords[featureIndex].FeatureTag())) {
				result = append(result, featureRecords[featureIndex])
			}
		}
	}

	// 'vrt2' supersedes 'vert' and they should not be used together
	// https://www.microsoft.com/typography/otspec/features_uz.htm
	if containsFeature(result, "vrt2") {
		result = removeFeature(result, "vert")
	}

	if enabledFeatures != nil && len(result) > 1 {
		// Java sorts by the position in enabledFeatures with a stable sort.
		sort.SliceStable(result, func(i, j int) bool {
			return slices.Index(enabledFeatures, result[i].FeatureTag()) <
				slices.Index(enabledFeatures, result[j].FeatureTag())
		})
	}

	return result
}

func containsFeature(featureRecords []*common.FeatureRecord, featureTag string) bool {
	for _, featureRecord := range featureRecords {
		if featureRecord.FeatureTag() == featureTag {
			return true
		}
	}
	return false
}

func removeFeature(featureRecords []*common.FeatureRecord,
	featureTag string) []*common.FeatureRecord {
	out := featureRecords[:0]
	for _, featureRecord := range featureRecords {
		if featureRecord.FeatureTag() != featureTag {
			out = append(out, featureRecord)
		}
	}
	return out
}

func (t *GlyphSubstitutionTable) applyFeature(featureRecord *common.FeatureRecord, gid int) int {
	lookupResult := gid
	lookups := t.lookupListTable.Lookups()
	for _, lookupListIndex := range featureRecord.FeatureTable().LookupListIndices() {
		if lookupListIndex < 0 || lookupListIndex >= len(lookups) {
			slog.Warn("Skipping GSUB feature with invalid lookupListIndex",
				"feature", featureRecord.FeatureTag(), "index", lookupListIndex,
				"len", len(lookups))
			continue
		}
		lookupTable := lookups[lookupListIndex]
		if lookupTable.LookupType() != 1 {
			slog.Warn("Skipping GSUB feature because it requires an unsupported lookup table type",
				"feature", featureRecord.FeatureTag(), "lookupType", lookupTable.LookupType())
			continue
		}
		lookupResult = doLookup(lookupTable, lookupResult)
	}
	return lookupResult
}

func doLookup(lookupTable *common.LookupTable, gid int) int {
	for _, lookupSubtable := range lookupTable.SubTables() {
		if lookupSubtable == nil {
			continue
		}
		coverageIndex := lookupSubtable.CoverageTable().CoverageIndex(gid)
		if coverageIndex >= 0 {
			return lookupSubtable.DoSubstitution(gid, coverageIndex)
		}
	}
	return gid
}

// GetSubstitution applies glyph substitutions to the supplied gid. The
// applicable substitutions are determined by the scriptTags, which indicate the
// language of the gid, and by the list of enabledFeatures.
//
// To ensure that a single gid isn't mapped to multiple substitutions,
// subsequent invocations with the same gid return the same result as the first,
// regardless of script or enabled features.
func (t *GlyphSubstitutionTable) GetSubstitution(gid int, scriptTags []string,
	enabledFeatures []string) int {
	if gid == -1 {
		return -1
	}
	if cached, ok := t.lookupCache[gid]; ok {
		// Because script detection for indeterminate scripts (COMMON, INHERIT,
		// etc.) depends on context, it is possible to return a different
		// substitution for the same input. However, we don't want that, as we
		// need a one-to-one mapping.
		return cached
	}
	scriptTag := t.selectScriptTag(scriptTags)
	langSysTables := t.langSysTables(scriptTag)
	featureRecords := t.featureRecords(langSysTables, enabledFeatures)
	sgid := gid
	for _, featureRecord := range featureRecords {
		sgid = t.applyFeature(featureRecord, sgid)
	}
	t.lookupCache[gid] = sgid
	t.reverseLookup[sgid] = gid
	return sgid
}

// GetUnsubstitution retrieves the original gid of a substitute gid obtained
// from GetSubstitution.
//
// Only gids previously substituted by this instance can be un-substituted. If
// you are trying to unsubstitute before you substitute, something is wrong.
func (t *GlyphSubstitutionTable) GetUnsubstitution(sgid int) int {
	gid, ok := t.reverseLookup[sgid]
	if !ok {
		slog.Warn("Trying to un-substitute a never-before-seen gid", "gid", sgid)
		return sgid
	}
	return gid
}

// GsubData returns the substitution data of every script of the table.
func (t *GlyphSubstitutionTable) GsubData() model.GsubData { return t.gsubData }

// GsubDataForScript builds a new GsubData for the given script tag. In contrast
// to GsubData, this one does not try to find the first supported language and
// load GSUB data for it. Instead, it fetches the data for the given scriptTag
// (if it's supported by the font) leaving the language unspecified. It means
// that even after successful reading of GSUB data, the actual glyph
// substitution may not work if there is no corresponding GsubWorker for it.
//
// Note: This method performs searching on every invocation (no results are
// cached). It returns nil where the font has no such script.
func (t *GlyphSubstitutionTable) GsubDataForScript(scriptTag string) model.GsubData {
	scriptTable := t.scriptList.Get(scriptTag)
	if scriptTable == nil {
		return nil
	}
	return gsubpkg.NewDataExtractor().GetGsubDataForScript(scriptTag, scriptTable,
		t.featureListTable, t.lookupListTable)
}

// SupportedScriptTags returns the script tags for which this GSUB table has
// records.
func (t *GlyphSubstitutionTable) SupportedScriptTags() []string {
	tags := make([]string, len(t.scriptList.Tags()))
	copy(tags, t.scriptList.Tags())
	return tags
}

func readRangeRecord(data DataStream) (common.RangeRecord, error) {
	r := newReader(data)
	startGlyphID := r.unsignedShort()
	endGlyphID := r.unsignedShort()
	startCoverageIndex := r.unsignedShort()
	if r.err != nil {
		return common.RangeRecord{}, r.err
	}
	return common.NewRangeRecord(startGlyphID, endGlyphID, startCoverageIndex), nil
}

// String describes the table.
func (t *GlyphSubstitutionTable) String() string {
	return "GlyphSubstitutionTable[" + strings.Join(t.SupportedScriptTags(), ",") + "]"
}
