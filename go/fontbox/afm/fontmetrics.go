package afm

import (
	"fmt"

	"github.com/shinguakira/pdfbox-go/go/fontbox/util"
)

// FontMetrics is the contents of an AFM file: everything the metrics of one
// font are made of.
//
// Port of org.apache.fontbox.afm.FontMetrics.
type FontMetrics struct {
	afmVersion     float32
	metricSets     int
	fontName       string
	fullName       string
	familyName     string
	weight         string
	fontBBox       *util.BoundingBox
	fontVersion    string
	notice         string
	encodingScheme string
	mappingScheme  int
	escChar        int
	characterSet   string
	characters     int
	isBaseFont     bool
	vVector        []float32

	// isFixedV is nil until it is set. Java declares it as a boxed Boolean for
	// the same reason: unset means "work it out from vVector", which is not the
	// same as false.
	isFixedV *bool

	capHeight float32
	xHeight   float32
	ascender  float32
	descender float32
	comments  []string

	underlinePosition       float32
	underlineThickness      float32
	italicAngle             float32
	charWidth               []float32
	isFixedPitch            bool
	standardHorizontalWidth float32
	standardVerticalWidth   float32

	charMetrics    []*CharMetric
	charMetricsMap map[string]*CharMetric
	trackKern      []*TrackKern
	composites     []*Composite
	kernPairs      []*KernPair
	kernPairs0     []*KernPair
	kernPairs1     []*KernPair
}

// NewFontMetrics returns empty font metrics.
func NewFontMetrics() *FontMetrics {
	return &FontMetrics{
		isBaseFont:     true,
		charMetricsMap: map[string]*CharMetric{},
	}
}

// CharacterWidth returns the width of the named character, or 0 if the font
// does not have it.
func (f *FontMetrics) CharacterWidth(name string) float32 {
	var result float32
	if metric, ok := f.charMetricsMap[name]; ok {
		result = metric.Wx()
	}
	return result
}

// CharacterHeight returns the height of the named character, or 0 if the font
// does not have it. A character with no vertical width falls back to the height
// of its bounding box.
func (f *FontMetrics) CharacterHeight(name string) float32 {
	var result float32
	if metric, ok := f.charMetricsMap[name]; ok {
		result = metric.Wy()
		if result == 0 {
			result = metric.BoundingBox().Height()
		}
	}
	return result
}

// AverageCharacterWidth returns the mean width of the characters that have one.
// A character of zero width is left out of both the total and the count.
func (f *FontMetrics) AverageCharacterWidth() float32 {
	var average float32
	var totalWidths float32
	var characterCount float32
	for _, metric := range f.charMetrics {
		if metric.Wx() > 0 {
			totalWidths += metric.Wx()
			characterCount++
		}
	}
	if totalWidths > 0 {
		average = totalWidths / characterCount
	}
	return average
}

// AddComment adds a comment.
func (f *FontMetrics) AddComment(comment string) {
	f.comments = append(f.comments, comment)
}

// Comments returns the comments, as a copy.
func (f *FontMetrics) Comments() []string {
	return append([]string{}, f.comments...)
}

// AFMVersion returns the version of the AFM format the file was written in.
func (f *FontMetrics) AFMVersion() float32 { return f.afmVersion }

// SetAFMVersion sets the AFM format version.
func (f *FontMetrics) SetAFMVersion(afmVersionValue float32) { f.afmVersion = afmVersionValue }

// MetricSets returns which sets of metrics the file carries.
func (f *FontMetrics) MetricSets() int { return f.metricSets }

// SetMetricSets sets which sets of metrics the file carries, which the format
// limits to 0, 1 or 2.
//
// Java throws IllegalArgumentException outside that range; the port panics,
// since it is a caller's mistake and nothing in PDFBox catches it.
func (f *FontMetrics) SetMetricSets(metricSetsValue int) {
	if metricSetsValue < 0 || metricSetsValue > 2 {
		panic(fmt.Sprintf(
			"afm: the metricSets attribute must be in the set {0,1,2} and not '%d'",
			metricSetsValue))
	}
	f.metricSets = metricSetsValue
}

// FontName returns the name of the font.
func (f *FontMetrics) FontName() string { return f.fontName }

// SetFontName sets the name of the font.
func (f *FontMetrics) SetFontName(name string) { f.fontName = name }

// FullName returns the full name of the font.
func (f *FontMetrics) FullName() string { return f.fullName }

// SetFullName sets the full name of the font.
func (f *FontMetrics) SetFullName(fullNameValue string) { f.fullName = fullNameValue }

// FamilyName returns the family the font belongs to.
func (f *FontMetrics) FamilyName() string { return f.familyName }

// SetFamilyName sets the family the font belongs to.
func (f *FontMetrics) SetFamilyName(familyNameValue string) { f.familyName = familyNameValue }

// Weight returns the weight of the font.
func (f *FontMetrics) Weight() string { return f.weight }

// SetWeight sets the weight of the font.
func (f *FontMetrics) SetWeight(weightValue string) { f.weight = weightValue }

// FontBBox returns the bounding box of the font.
func (f *FontMetrics) FontBBox() *util.BoundingBox { return f.fontBBox }

// SetFontBBox sets the bounding box of the font.
func (f *FontMetrics) SetFontBBox(bBox *util.BoundingBox) { f.fontBBox = bBox }

// Notice returns the copyright notice.
func (f *FontMetrics) Notice() string { return f.notice }

// SetNotice sets the copyright notice.
func (f *FontMetrics) SetNotice(noticeValue string) { f.notice = noticeValue }

// EncodingScheme returns the encoding scheme.
func (f *FontMetrics) EncodingScheme() string { return f.encodingScheme }

// SetEncodingScheme sets the encoding scheme.
func (f *FontMetrics) SetEncodingScheme(encodingSchemeValue string) {
	f.encodingScheme = encodingSchemeValue
}

// MappingScheme returns the mapping scheme.
func (f *FontMetrics) MappingScheme() int { return f.mappingScheme }

// SetMappingScheme sets the mapping scheme.
func (f *FontMetrics) SetMappingScheme(mappingSchemeValue int) {
	f.mappingScheme = mappingSchemeValue
}

// EscChar returns the escape character.
func (f *FontMetrics) EscChar() int { return f.escChar }

// SetEscChar sets the escape character.
func (f *FontMetrics) SetEscChar(escCharValue int) { f.escChar = escCharValue }

// CharacterSet returns the character set of the font.
func (f *FontMetrics) CharacterSet() string { return f.characterSet }

// SetCharacterSet sets the character set of the font.
func (f *FontMetrics) SetCharacterSet(characterSetValue string) {
	f.characterSet = characterSetValue
}

// Characters returns how many characters the font has.
func (f *FontMetrics) Characters() int { return f.characters }

// SetCharacters sets how many characters the font has.
func (f *FontMetrics) SetCharacters(charactersValue int) { f.characters = charactersValue }

// IsBaseFont reports whether this is a base font. It is true unless the file
// says otherwise.
func (f *FontMetrics) IsBaseFont() bool { return f.isBaseFont }

// SetIsBaseFont sets whether this is a base font.
func (f *FontMetrics) SetIsBaseFont(isBaseFontValue bool) { f.isBaseFont = isBaseFontValue }

// VVector returns the vector from the origin of writing direction 0 to that of
// direction 1.
func (f *FontMetrics) VVector() []float32 { return f.vVector }

// SetVVector sets that vector.
func (f *FontMetrics) SetVVector(vVectorValue []float32) { f.vVector = vVectorValue }

// IsFixedV reports whether the vertical origin is fixed. Where the file does
// not say, it is fixed exactly when the font carries a VVector.
func (f *FontMetrics) IsFixedV() bool {
	// if not set the default value depends on the existence of vVector
	if f.isFixedV == nil {
		return f.vVector != nil
	}
	return *f.isFixedV
}

// SetIsFixedV sets whether the vertical origin is fixed.
func (f *FontMetrics) SetIsFixedV(isFixedVValue bool) { f.isFixedV = &isFixedVValue }

// CapHeight returns the height of a capital letter.
func (f *FontMetrics) CapHeight() float32 { return f.capHeight }

// SetCapHeight sets the height of a capital letter.
func (f *FontMetrics) SetCapHeight(capHeightValue float32) { f.capHeight = capHeightValue }

// XHeight returns the height of a lower-case x.
func (f *FontMetrics) XHeight() float32 { return f.xHeight }

// SetXHeight sets the height of a lower-case x.
func (f *FontMetrics) SetXHeight(xHeightValue float32) { f.xHeight = xHeightValue }

// Ascender returns the ascender of the font.
func (f *FontMetrics) Ascender() float32 { return f.ascender }

// SetAscender sets the ascender of the font.
func (f *FontMetrics) SetAscender(ascenderValue float32) { f.ascender = ascenderValue }

// Descender returns the descender of the font.
func (f *FontMetrics) Descender() float32 { return f.descender }

// SetDescender sets the descender of the font.
func (f *FontMetrics) SetDescender(descenderValue float32) { f.descender = descenderValue }

// FontVersion returns the version of the font.
func (f *FontMetrics) FontVersion() string { return f.fontVersion }

// SetFontVersion sets the version of the font.
func (f *FontMetrics) SetFontVersion(fontVersionValue string) { f.fontVersion = fontVersionValue }

// UnderlinePosition returns where an underline sits.
func (f *FontMetrics) UnderlinePosition() float32 { return f.underlinePosition }

// SetUnderlinePosition sets where an underline sits.
func (f *FontMetrics) SetUnderlinePosition(underlinePositionValue float32) {
	f.underlinePosition = underlinePositionValue
}

// UnderlineThickness returns how thick an underline is.
func (f *FontMetrics) UnderlineThickness() float32 { return f.underlineThickness }

// SetUnderlineThickness sets how thick an underline is.
func (f *FontMetrics) SetUnderlineThickness(underlineThicknessValue float32) {
	f.underlineThickness = underlineThicknessValue
}

// ItalicAngle returns the angle the font slants at.
func (f *FontMetrics) ItalicAngle() float32 { return f.italicAngle }

// SetItalicAngle sets the angle the font slants at.
func (f *FontMetrics) SetItalicAngle(italicAngleValue float32) { f.italicAngle = italicAngleValue }

// CharWidth returns the width of every character, for a fixed pitch font.
func (f *FontMetrics) CharWidth() []float32 { return f.charWidth }

// SetCharWidth sets the width of every character.
func (f *FontMetrics) SetCharWidth(charWidthValue []float32) { f.charWidth = charWidthValue }

// IsFixedPitch reports whether every character has the same width.
func (f *FontMetrics) IsFixedPitch() bool { return f.isFixedPitch }

// SetFixedPitch sets whether every character has the same width.
func (f *FontMetrics) SetFixedPitch(fixedPitchValue bool) { f.isFixedPitch = fixedPitchValue }

// StandardHorizontalWidth returns the dominant width of vertical stems.
func (f *FontMetrics) StandardHorizontalWidth() float32 { return f.standardHorizontalWidth }

// SetStandardHorizontalWidth sets the dominant width of vertical stems.
func (f *FontMetrics) SetStandardHorizontalWidth(standardHorizontalWidthValue float32) {
	f.standardHorizontalWidth = standardHorizontalWidthValue
}

// StandardVerticalWidth returns the dominant width of horizontal stems.
func (f *FontMetrics) StandardVerticalWidth() float32 { return f.standardVerticalWidth }

// SetStandardVerticalWidth sets the dominant width of horizontal stems.
func (f *FontMetrics) SetStandardVerticalWidth(standardVerticalWidthValue float32) {
	f.standardVerticalWidth = standardVerticalWidthValue
}

// CharMetrics returns the metrics of every character, as a copy.
func (f *FontMetrics) CharMetrics() []*CharMetric {
	return append([]*CharMetric{}, f.charMetrics...)
}

// AddCharMetric adds the metrics of one character, and indexes it by name.
func (f *FontMetrics) AddCharMetric(metric *CharMetric) {
	f.charMetrics = append(f.charMetrics, metric)
	f.charMetricsMap[metric.Name()] = metric
}

// TrackKern returns the track kerning entries, as a copy.
func (f *FontMetrics) TrackKern() []*TrackKern {
	return append([]*TrackKern{}, f.trackKern...)
}

// AddTrackKern adds a track kerning entry.
func (f *FontMetrics) AddTrackKern(kern *TrackKern) {
	f.trackKern = append(f.trackKern, kern)
}

// Composites returns the composite characters, as a copy.
func (f *FontMetrics) Composites() []*Composite {
	return append([]*Composite{}, f.composites...)
}

// AddComposite adds a composite character.
func (f *FontMetrics) AddComposite(composite *Composite) {
	f.composites = append(f.composites, composite)
}

// KernPairs returns the kerning pairs, as a copy.
func (f *FontMetrics) KernPairs() []*KernPair {
	return append([]*KernPair{}, f.kernPairs...)
}

// AddKernPair adds a kerning pair.
func (f *FontMetrics) AddKernPair(kernPair *KernPair) {
	f.kernPairs = append(f.kernPairs, kernPair)
}

// KernPairs0 returns the kerning pairs for writing direction 0, as a copy.
func (f *FontMetrics) KernPairs0() []*KernPair {
	return append([]*KernPair{}, f.kernPairs0...)
}

// AddKernPair0 adds a kerning pair for writing direction 0.
func (f *FontMetrics) AddKernPair0(kernPair *KernPair) {
	f.kernPairs0 = append(f.kernPairs0, kernPair)
}

// KernPairs1 returns the kerning pairs for writing direction 1, as a copy.
func (f *FontMetrics) KernPairs1() []*KernPair {
	return append([]*KernPair{}, f.kernPairs1...)
}

// AddKernPair1 adds a kerning pair for writing direction 1.
func (f *FontMetrics) AddKernPair1(kernPair *KernPair) {
	f.kernPairs1 = append(f.kernPairs1, kernPair)
}
