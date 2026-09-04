// Package model holds what a font's glyph substitution data looks like, apart
// from how it is read.
//
// Port of org.apache.fontbox.ttf.model.
package model

import (
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// GlyphKey is a list of glyph IDs used as a map key, which Go cannot do with a
// slice.
//
// Java keys its substitution maps on List<Integer>, whose equals and hashCode
// are by value; this string carries the same values in the same order.
type GlyphKey string

// NewGlyphKey returns the key of the given glyph IDs.
func NewGlyphKey(glyphIDs []int) GlyphKey {
	var sb strings.Builder
	for i, id := range glyphIDs {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(strconv.Itoa(id))
	}
	return GlyphKey(sb.String())
}

// IDs returns the glyph IDs the key was built from.
func (k GlyphKey) IDs() []int {
	if k == "" {
		return nil
	}
	parts := strings.Split(string(k), ",")
	ids := make([]int, len(parts))
	for i, part := range parts {
		ids[i], _ = strconv.Atoi(part)
	}
	return ids
}

// String renders the key the way Java's List.toString does, for the messages
// the glyph IDs appear in.
func (k GlyphKey) String() string {
	ids := k.IDs()
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.Itoa(id)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// Language is the script a font's substitution data belongs to.
//
// Port of the org.apache.fontbox.ttf.model.Language enum.
type Language int

// The languages, in the order the Java enum declares them.
const (
	Bengali Language = iota
	Devanagari
	Gujarati
	Tamil
	Latin
	Dflt
	Unspecified
)

// languageScriptNames is the script names of each language.
var languageScriptNames = [][]string{
	Bengali:     {"bng2", "beng"},
	Devanagari:  {"dev2", "deva"},
	Gujarati:    {"gjr2", "gujr"},
	Tamil:       {"tml2", "taml"},
	Latin:       {"latn"},
	Dflt:        {"DFLT"},
	Unspecified: {},
}

// languageNames names each language, as Java's enum does.
var languageNames = [...]string{
	Bengali:     "BENGALI",
	Devanagari:  "DEVANAGARI",
	Gujarati:    "GUJARATI",
	Tamil:       "TAMIL",
	Latin:       "LATIN",
	Dflt:        "DFLT",
	Unspecified: "UNSPECIFIED",
}

// Languages is every language, in declaration order.
var Languages = []Language{Bengali, Devanagari, Gujarati, Tamil, Latin, Dflt, Unspecified}

// ScriptNames returns the script names the language covers.
func (l Language) ScriptNames() []string { return languageScriptNames[l] }

// String names the language.
func (l Language) String() string {
	if int(l) < len(languageNames) {
		return languageNames[l]
	}
	return strconv.Itoa(int(l))
}

// ScriptFeature is one feature of one script, and the substitutions it makes.
//
// Port of the org.apache.fontbox.ttf.model.ScriptFeature interface.
type ScriptFeature interface {
	// Name returns the name of the feature.
	Name() string

	// AllGlyphIDsForSubstitution returns every glyph run the feature replaces.
	AllGlyphIDsForSubstitution() []GlyphKey

	// CanReplaceGlyphs reports whether the feature replaces the given run.
	CanReplaceGlyphs(glyphIDs []int) bool

	// ReplacementForGlyphs returns what the feature replaces the given run
	// with.
	ReplacementForGlyphs(glyphIDs []int) []int
}

// GsubData is a font's glyph substitution data.
//
// Port of the org.apache.fontbox.ttf.model.GsubData interface.
type GsubData interface {
	// Language returns the script the data belongs to.
	Language() Language

	// ActiveScriptName returns the name of the script the data was read for.
	ActiveScriptName() string

	// IsFeatureSupported reports whether the data carries the named feature.
	IsFeatureSupported(featureName string) bool

	// Feature returns the named feature.
	Feature(featureName string) ScriptFeature

	// SupportedFeatures returns the names of every feature the data carries.
	SupportedFeatures() []string
}

// NoDataFound is the GsubData of a font that has none.
//
// Port of GsubData.NO_DATA_FOUND, whose every method throws
// UnsupportedOperationException; the port panics for the same reason.
var NoDataFound GsubData = noDataFound{}

type noDataFound struct{}

func (noDataFound) IsFeatureSupported(featureName string) bool { panic("no gsub data found") }
func (noDataFound) Language() Language                         { panic("no gsub data found") }
func (noDataFound) Feature(featureName string) ScriptFeature   { panic("no gsub data found") }
func (noDataFound) ActiveScriptName() string                   { panic("no gsub data found") }
func (noDataFound) SupportedFeatures() []string                { panic("no gsub data found") }

// MapBackedGsubData is GsubData built from a map of features.
//
// Port of org.apache.fontbox.ttf.model.MapBackedGsubData.
type MapBackedGsubData struct {
	language             Language
	activeScriptName     string
	glyphSubstitutionMap map[string]map[GlyphKey][]int
}

var _ GsubData = (*MapBackedGsubData)(nil)

// NewMapBackedGsubData returns GsubData over the given feature map.
func NewMapBackedGsubData(language Language, activeScriptName string,
	glyphSubstitutionMap map[string]map[GlyphKey][]int) *MapBackedGsubData {
	return &MapBackedGsubData{
		language:             language,
		activeScriptName:     activeScriptName,
		glyphSubstitutionMap: glyphSubstitutionMap,
	}
}

// Language returns the script the data belongs to.
func (d *MapBackedGsubData) Language() Language { return d.language }

// ActiveScriptName returns the name of the script the data was read for.
func (d *MapBackedGsubData) ActiveScriptName() string { return d.activeScriptName }

// IsFeatureSupported reports whether the data carries the named feature.
func (d *MapBackedGsubData) IsFeatureSupported(featureName string) bool {
	_, ok := d.glyphSubstitutionMap[featureName]
	return ok
}

// Feature returns the named feature.
//
// Java throws UnsupportedOperationException for a feature it does not carry,
// which is unchecked; the port panics.
func (d *MapBackedGsubData) Feature(featureName string) ScriptFeature {
	if !d.IsFeatureSupported(featureName) {
		panic(fmt.Sprintf("The feature %s is not supported!", featureName))
	}
	return NewMapBackedScriptFeature(featureName, d.glyphSubstitutionMap[featureName])
}

// SupportedFeatures returns the names of every feature the data carries.
//
// Java hands back the map's key set, whose order a HashMap does not define; the
// port sorts, so that a caller iterating it twice sees the same thing.
func (d *MapBackedGsubData) SupportedFeatures() []string {
	names := make([]string, 0, len(d.glyphSubstitutionMap))
	for name := range d.glyphSubstitutionMap {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// MapBackedScriptFeature is a ScriptFeature built from a map of substitutions.
//
// Port of org.apache.fontbox.ttf.model.MapBackedScriptFeature.
type MapBackedScriptFeature struct {
	name       string
	featureMap map[GlyphKey][]int
}

var _ ScriptFeature = (*MapBackedScriptFeature)(nil)

// NewMapBackedScriptFeature returns a feature over the given substitutions.
func NewMapBackedScriptFeature(name string, featureMap map[GlyphKey][]int) *MapBackedScriptFeature {
	return &MapBackedScriptFeature{name: name, featureMap: featureMap}
}

// Name returns the name of the feature.
func (f *MapBackedScriptFeature) Name() string { return f.name }

// AllGlyphIDsForSubstitution returns every glyph run the feature replaces.
//
// Java hands back the map's key set; the port sorts, so that a caller iterating
// it twice sees the same thing.
func (f *MapBackedScriptFeature) AllGlyphIDsForSubstitution() []GlyphKey {
	keys := make([]GlyphKey, 0, len(f.featureMap))
	for key := range f.featureMap {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}

// CanReplaceGlyphs reports whether the feature replaces the given run.
func (f *MapBackedScriptFeature) CanReplaceGlyphs(glyphIDs []int) bool {
	_, ok := f.featureMap[NewGlyphKey(glyphIDs)]
	return ok
}

// ReplacementForGlyphs returns what the feature replaces the given run with.
//
// Java throws UnsupportedOperationException for a run it does not replace,
// which is unchecked; the port panics.
func (f *MapBackedScriptFeature) ReplacementForGlyphs(glyphIDs []int) []int {
	key := NewGlyphKey(glyphIDs)
	replacement, ok := f.featureMap[key]
	if !ok {
		panic(fmt.Sprintf("The glyphs %s cannot be replaced", key))
	}
	return replacement
}

// Equals reports whether two features carry the same name and the same
// substitutions, which is Java's equals.
func (f *MapBackedScriptFeature) Equals(other *MapBackedScriptFeature) bool {
	if f == other {
		return true
	}
	if other == nil || f.name != other.name || len(f.featureMap) != len(other.featureMap) {
		return false
	}
	for key, value := range f.featureMap {
		otherValue, ok := other.featureMap[key]
		if !ok || !slices.Equal(value, otherValue) {
			return false
		}
	}
	return true
}
