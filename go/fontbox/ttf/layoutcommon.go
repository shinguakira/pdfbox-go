package ttf

// The parts of an OpenType Layout header that GSUB and GPOS share: the script
// list, the feature list and the coverage tables.
//
// GlyphSubstitutionTable reads its own copies as methods, because it was ported
// from PDFBox's class and that is how the Java is written. GPOS is not a port
// of anything (see glyphpositioning.go), so it reads them through the functions
// here rather than growing a second set of methods that say the same thing.

import (
	"fmt"

	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf/table/common"
)

// readLayoutScriptList reads a ScriptList, answering the script table of each
// tag.
func readLayoutScriptList(data DataStream, offset int64) (map[string]*common.ScriptTable, error) {
	if err := data.SeekTo(offset); err != nil {
		return nil, err
	}
	r := newReader(data)
	scriptCount := r.unsignedShort()
	if r.err != nil {
		return nil, r.err
	}
	tags := make([]string, scriptCount)
	offsets := make([]int, scriptCount)
	for i := 0; i < scriptCount; i++ {
		tag, err := readTag(data)
		if err != nil {
			return nil, err
		}
		tags[i] = tag
		offsets[i] = r.unsignedShort()
	}
	if r.err != nil {
		return nil, r.err
	}

	scripts := make(map[string]*common.ScriptTable, scriptCount)
	for i := 0; i < scriptCount; i++ {
		table, err := readLayoutScriptTable(data, offset+int64(offsets[i]))
		if err != nil {
			return nil, err
		}
		scripts[tags[i]] = table
	}
	return scripts, nil
}

// readLayoutScriptTable reads one script's default and named language systems.
func readLayoutScriptTable(data DataStream, offset int64) (*common.ScriptTable, error) {
	if err := data.SeekTo(offset); err != nil {
		return nil, err
	}
	r := newReader(data)
	defaultLangSys := r.unsignedShort()
	langSysCount := r.unsignedShort()
	if r.err != nil {
		return nil, r.err
	}
	tags := make([]string, langSysCount)
	offsets := make([]int, langSysCount)
	for i := 0; i < langSysCount; i++ {
		tag, err := readTag(data)
		if err != nil {
			return nil, err
		}
		tags[i] = tag
		offsets[i] = r.unsignedShort()
	}
	if r.err != nil {
		return nil, r.err
	}

	var defaultTable *common.LangSysTable
	if defaultLangSys != 0 {
		table, err := readLayoutLangSysTable(data, offset+int64(defaultLangSys), "")
		if err != nil {
			return nil, err
		}
		defaultTable = table
	}
	langSysTables := make(map[string]*common.LangSysTable, langSysCount)
	for i := 0; i < langSysCount; i++ {
		table, err := readLayoutLangSysTable(data, offset+int64(offsets[i]), tags[i])
		if err != nil {
			return nil, err
		}
		langSysTables[tags[i]] = table
	}
	return common.NewScriptTable(defaultTable, langSysTables), nil
}

// readLayoutLangSysTable reads which features one language system turns on.
func readLayoutLangSysTable(data DataStream, offset int64,
	tag string) (*common.LangSysTable, error) {
	if err := data.SeekTo(offset); err != nil {
		return nil, err
	}
	r := newReader(data)
	lookupOrder := r.unsignedShort() // the specification reserves it
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
	_ = tag
	return common.NewLangSysTable(lookupOrder, requiredFeatureIndex, featureIndexCount,
		featureIndices), nil
}

// readLayoutFeatureList reads a FeatureList, answering the record of each
// feature in the order the table holds them, because a language system names
// them by index.
func readLayoutFeatureList(data DataStream, offset int64) ([]*common.FeatureRecord, error) {
	if err := data.SeekTo(offset); err != nil {
		return nil, err
	}
	r := newReader(data)
	featureCount := r.unsignedShort()
	if r.err != nil {
		return nil, r.err
	}
	tags := make([]string, featureCount)
	offsets := make([]int, featureCount)
	for i := 0; i < featureCount; i++ {
		tag, err := readTag(data)
		if err != nil {
			return nil, err
		}
		tags[i] = tag
		offsets[i] = r.unsignedShort()
	}
	if r.err != nil {
		return nil, r.err
	}

	records := make([]*common.FeatureRecord, featureCount)
	for i := 0; i < featureCount; i++ {
		table, err := readLayoutFeatureTable(data, offset+int64(offsets[i]))
		if err != nil {
			return nil, err
		}
		records[i] = common.NewFeatureRecord(tags[i], table)
	}
	return records, nil
}

// readLayoutFeatureTable reads which lookups one feature runs.
func readLayoutFeatureTable(data DataStream, offset int64) (*common.FeatureTable, error) {
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

// readLayoutCoverageTable reads a coverage table, which says which glyphs a
// subtable applies to and in what order.
func readLayoutCoverageTable(data DataStream, offset int64) (common.CoverageTable, error) {
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
		return nil, fmt.Errorf("ttf: unknown coverage format: %d", coverageFormat)
	}
}
