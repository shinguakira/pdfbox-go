package gsub

import (
	"log/slog"
	"slices"

	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf/model"
)

// The features GsubWorkerForBengali applies, in order.
//
// See https://docs.microsoft.com/en-us/typography/script-development/bengali
const initFeature = "init"

var bengaliFeaturesInOrder = []string{"locl", "nukt", "akhn", "rphf", "blwf", "pstf", "half",
	"vatu", "cjct", initFeature, "pres", "abvs", "blws", "psts", "haln", "calt"}

var bengaliBeforeHalfChars = []rune{'ি', 'ে', 'ৈ'}

var bengaliBeforeAndAfterSpanChars = []beforeAndAfterSpanComponent{
	{originalCharacter: 'ো', beforeComponentCharacter: 'ে', afterComponentCharacter: 'া'},
	{originalCharacter: 'ৌ', beforeComponentCharacter: 'ে', afterComponentCharacter: 'ৗ'},
}

// beforeAndAfterSpanComponent is a character that a script writes as two, one
// before and one after the consonant.
type beforeAndAfterSpanComponent struct {
	originalCharacter        rune
	beforeComponentCharacter rune
	afterComponentCharacter  rune
}

// GsubWorkerForBengali is the Bengali-specific implementation of the GSUB
// system.
//
// Port of org.apache.fontbox.ttf.gsub.GsubWorkerForBengali.
type GsubWorkerForBengali struct {
	featureApplier

	lookup   CmapLookup
	gsubData model.GsubData

	beforeHalfGlyphIDs         []int
	beforeAndAfterSpanGlyphIDs map[int]beforeAndAfterSpanComponent
}

var _ GsubWorker = (*GsubWorkerForBengali)(nil)

func newGsubWorkerForBengali(cmapLookup CmapLookup, gsubData model.GsubData) *GsubWorkerForBengali {
	w := &GsubWorkerForBengali{
		featureApplier: newFeatureApplier(),
		lookup:         cmapLookup,
		gsubData:       gsubData,
	}
	w.beforeHalfGlyphIDs = w.getBeforeHalfGlyphIDs()
	w.beforeAndAfterSpanGlyphIDs = w.getBeforeAndAfterSpanGlyphIDs()
	return w
}

// ApplyTransforms returns the glyphs the substitutions leave behind.
func (w *GsubWorkerForBengali) ApplyTransforms(originalGlyphIDs []int) []int {
	glyphs := w.applyFeaturesInOrder(w.gsubData, bengaliFeaturesInOrder, originalGlyphIDs)
	return slices.Clone(w.repositionGlyphs(glyphs))
}

func (w *GsubWorkerForBengali) repositionGlyphs(originalGlyphIDs []int) []int {
	glyphsRepositionedByBeforeHalf := w.repositionBeforeHalfGlyphIDs(originalGlyphIDs)
	return w.repositionBeforeAndAfterSpanGlyphIDs(glyphsRepositionedByBeforeHalf)
}

func (w *GsubWorkerForBengali) repositionBeforeHalfGlyphIDs(originalGlyphIDs []int) []int {
	repositionedGlyphIDs := slices.Clone(originalGlyphIDs)
	for index := 1; index < len(originalGlyphIDs); index++ {
		glyphID := originalGlyphIDs[index]
		if slices.Contains(w.beforeHalfGlyphIDs, glyphID) {
			previousGlyphID := originalGlyphIDs[index-1]
			repositionedGlyphIDs[index] = previousGlyphID
			repositionedGlyphIDs[index-1] = glyphID
		}
	}
	return repositionedGlyphIDs
}

func (w *GsubWorkerForBengali) repositionBeforeAndAfterSpanGlyphIDs(originalGlyphIDs []int) []int {
	repositionedGlyphIDs := slices.Clone(originalGlyphIDs)
	for index := 1; index < len(originalGlyphIDs); index++ {
		glyphID := originalGlyphIDs[index]
		component, ok := w.beforeAndAfterSpanGlyphIDs[glyphID]
		if !ok {
			continue
		}
		previousGlyphID := originalGlyphIDs[index-1]
		repositionedGlyphIDs[index] = previousGlyphID
		repositionedGlyphIDs[index-1] = w.getGlyphID(component.beforeComponentCharacter)
		repositionedGlyphIDs = slices.Insert(repositionedGlyphIDs, index+1,
			w.getGlyphID(component.afterComponentCharacter))
	}
	return repositionedGlyphIDs
}

func (w *GsubWorkerForBengali) getBeforeHalfGlyphIDs() []int {
	glyphIDs := make([]int, 0, len(bengaliBeforeHalfChars))
	for _, character := range bengaliBeforeHalfChars {
		glyphIDs = append(glyphIDs, w.getGlyphID(character))
	}
	if w.gsubData.IsFeatureSupported(initFeature) {
		feature := w.gsubData.Feature(initFeature)
		for _, glyphCluster := range feature.AllGlyphIDsForSubstitution() {
			glyphIDs = append(glyphIDs, feature.ReplacementForGlyphs(glyphCluster.IDs())...)
		}
	}
	return glyphIDs
}

func (w *GsubWorkerForBengali) getGlyphID(character rune) int {
	return w.lookup.GetGlyphID(int(character))
}

func (w *GsubWorkerForBengali) getBeforeAndAfterSpanGlyphIDs() map[int]beforeAndAfterSpanComponent {
	result := map[int]beforeAndAfterSpanComponent{}
	for _, component := range bengaliBeforeAndAfterSpanChars {
		result[w.getGlyphID(component.originalCharacter)] = component
	}
	return result
}

// The features the three reph-repositioning workers apply, in order.
const (
	rkrfFeature = "rkrf"
	vatuFeature = "vatu"
)

var rephFeaturesInOrder = []string{"locl", "nukt", "akhn", "rphf", rkrfFeature, "blwf", "half",
	vatuFeature, "cjct", "pres", "abvs", "blws", "psts", "haln", "calt"}

var tamilFeaturesInOrder = []string{"locl", "nukt", "akhn", "rphf", "pref", "half", "pres",
	"abvs", "blws", "psts", "haln", "calt"}

// rephWorker is the shared body of the Devanagari, Gujarati and Tamil workers.
//
// Java writes those three out as separate classes whose bodies are identical
// apart from four character constants and, for Tamil, the feature list and the
// missing rkrf fallback; the port keeps one implementation and names which
// class each field belongs to.
type rephWorker struct {
	featureApplier

	lookup   CmapLookup
	gsubData model.GsubData

	featuresInOrder []string
	// applyRKRF says whether the worker builds an rkrf feature out of vatu,
	// which Devanagari and Gujarati do and Tamil does not.
	applyRKRF bool

	rephGlyphIDs       []int
	beforeRephGlyphIDs []int
	beforeHalfGlyphIDs []int
}

var _ GsubWorker = (*rephWorker)(nil)

func newRephWorker(cmapLookup CmapLookup, gsubData model.GsubData, featuresInOrder []string,
	applyRKRF bool, rephChars, beforeRephChars []rune, beforeHalfChar rune) *rephWorker {
	w := &rephWorker{
		featureApplier:  newFeatureApplier(),
		lookup:          cmapLookup,
		gsubData:        gsubData,
		featuresInOrder: featuresInOrder,
		applyRKRF:       applyRKRF,
	}
	w.beforeHalfGlyphIDs = []int{w.getGlyphID(beforeHalfChar)}
	w.rephGlyphIDs = w.glyphIDsOf(rephChars)
	w.beforeRephGlyphIDs = w.glyphIDsOf(beforeRephChars)
	return w
}

// newGsubWorkerForDevanagari returns the Devanagari-specific implementation of
// the GSUB system.
//
// Port of org.apache.fontbox.ttf.gsub.GsubWorkerForDevanagari. See
// https://docs.microsoft.com/en-us/typography/script-development/devanagari
func newGsubWorkerForDevanagari(cmapLookup CmapLookup, gsubData model.GsubData) *rephWorker {
	return newRephWorker(cmapLookup, gsubData, rephFeaturesInOrder, true,
		// Reph glyphs
		[]rune{'र', '्'},
		// Glyphs to precede reph
		[]rune{'ा', 'ी'},
		// Devanagari vowel sign I
		'ि')
}

// newGsubWorkerForGujarati returns the Gujarati-specific implementation of the
// GSUB system.
//
// Port of org.apache.fontbox.ttf.gsub.GsubWorkerForGujarati. See
// https://docs.microsoft.com/en-us/typography/script-development/gujarati
func newGsubWorkerForGujarati(cmapLookup CmapLookup, gsubData model.GsubData) *rephWorker {
	return newRephWorker(cmapLookup, gsubData, rephFeaturesInOrder, true,
		// Reph glyphs
		[]rune{'ર', '્'},
		// Glyphs to precede reph
		[]rune{'ા', 'ી'},
		// Gujarati vowel sign I
		'િ')
}

// NewGsubWorkerForTamil returns the Tamil-specific implementation of the GSUB
// system.
//
// Port of org.apache.fontbox.ttf.gsub.GsubWorkerForTamil, whose constants carry
// the note "TODO adjust all below this line. The existing code has been copied
// from Gujarati" -- which is why the vowel sign it repositions is still the
// Gujarati one. The factory does not choose this worker; Java leaves Tamil
// falling through to the default one.
func NewGsubWorkerForTamil(cmapLookup CmapLookup, gsubData model.GsubData) GsubWorker {
	return newRephWorker(cmapLookup, gsubData, tamilFeaturesInOrder, false,
		// Reph glyphs
		[]rune{'ர', '்'},
		// Glyphs to precede reph
		[]rune{'ஸ', '்'},
		// Gujarati vowel sign I
		'િ')
}

// ApplyTransforms returns the glyphs the substitutions leave behind.
func (w *rephWorker) ApplyTransforms(originalGlyphIDs []int) []int {
	glyphs := w.adjustRephPosition(originalGlyphIDs)
	glyphs = w.repositionGlyphs(glyphs)
	for _, feature := range w.featuresInOrder {
		if !w.gsubData.IsFeatureSupported(feature) {
			if w.applyRKRF && feature == rkrfFeature &&
				w.gsubData.IsFeatureSupported(vatuFeature) {
				// Create your own rkrf feature from vatu feature
				glyphs = w.applyRKRFFeature(w.gsubData.Feature(vatuFeature), glyphs)
			}
			slog.Debug("the feature was not found", "feature", feature)
			continue
		}
		slog.Debug("applying the feature", "feature", feature)
		glyphs = w.applyGsubFeature(w.gsubData.Feature(feature), glyphs)
	}
	return slices.Clone(glyphs)
}

func (w *rephWorker) applyRKRFFeature(rkrfGlyphsForSubstitution model.ScriptFeature,
	originalGlyphIDs []int) []int {
	rkrfGlyphIDs := rkrfGlyphsForSubstitution.AllGlyphIDsForSubstitution()
	if len(rkrfGlyphIDs) == 0 {
		slog.Debug("Glyph substitution list is empty", "feature", rkrfGlyphsForSubstitution.Name())
		return originalGlyphIDs
	}
	// Replace this with better implementation to get second GlyphId from
	// rkrfGlyphIds
	rkrfReplacement := 0
	for _, firstList := range rkrfGlyphIDs {
		ids := firstList.IDs()
		if len(ids) > 1 {
			rkrfReplacement = ids[1]
			break
		}
	}
	if rkrfReplacement == 0 {
		slog.Debug("Cannot find rkrf candidate. The rkrfGlyphIds doesn't contain lists of two elements.")
		return originalGlyphIDs
	}
	rkrfList := slices.Clone(originalGlyphIDs)
	for index := len(originalGlyphIDs) - 1; index > 1; index-- {
		raGlyph := originalGlyphIDs[index]
		if raGlyph == w.rephGlyphIDs[0] {
			viramaGlyph := originalGlyphIDs[index-1]
			if viramaGlyph == w.rephGlyphIDs[1] {
				rkrfList[index-1] = rkrfReplacement
				rkrfList = slices.Delete(rkrfList, index, index+1)
			}
		}
	}
	return rkrfList
}

func (w *rephWorker) adjustRephPosition(originalGlyphIDs []int) []int {
	rephAdjustedList := slices.Clone(originalGlyphIDs)
	for index := 0; index < len(originalGlyphIDs)-2; index++ {
		raGlyph := originalGlyphIDs[index]
		viramaGlyph := originalGlyphIDs[index+1]
		if raGlyph != w.rephGlyphIDs[0] || viramaGlyph != w.rephGlyphIDs[1] {
			continue
		}
		// reph virama cons => cons reph virama
		nextConsonantGlyph := originalGlyphIDs[index+2]
		rephAdjustedList[index] = nextConsonantGlyph
		rephAdjustedList[index+1] = raGlyph
		rephAdjustedList[index+2] = viramaGlyph

		if index+3 < len(originalGlyphIDs) {
			// reph virama cons matra => cons matra reph virama
			matraGlyph := originalGlyphIDs[index+3]
			if slices.Contains(w.beforeRephGlyphIDs, matraGlyph) {
				rephAdjustedList[index+1] = matraGlyph
				rephAdjustedList[index+2] = raGlyph
				rephAdjustedList[index+3] = viramaGlyph
			}
		}
	}
	return rephAdjustedList
}

func (w *rephWorker) repositionGlyphs(originalGlyphIDs []int) []int {
	repositionedGlyphIDs := slices.Clone(originalGlyphIDs)
	listSize := len(repositionedGlyphIDs)
	foundIndex := listSize - 1
	nextIndex := listSize - 2
	for nextIndex > -1 {
		glyph := repositionedGlyphIDs[foundIndex]
		prevIndex := foundIndex + 1
		if slices.Contains(w.beforeHalfGlyphIDs, glyph) {
			repositionedGlyphIDs = slices.Delete(repositionedGlyphIDs, foundIndex, foundIndex+1)
			repositionedGlyphIDs = slices.Insert(repositionedGlyphIDs, nextIndex, glyph)
			nextIndex--
		} else if w.rephGlyphIDs[1] == glyph && prevIndex < listSize {
			prevGlyph := repositionedGlyphIDs[prevIndex]
			if slices.Contains(w.beforeHalfGlyphIDs, prevGlyph) {
				repositionedGlyphIDs = slices.Delete(repositionedGlyphIDs, prevIndex, prevIndex+1)
				repositionedGlyphIDs = slices.Insert(repositionedGlyphIDs, nextIndex, prevGlyph)
				nextIndex--
			}
		}
		foundIndex = nextIndex
		nextIndex--
	}
	return repositionedGlyphIDs
}

func (w *rephWorker) glyphIDsOf(characters []rune) []int {
	glyphIDs := make([]int, 0, len(characters))
	for _, character := range characters {
		glyphIDs = append(glyphIDs, w.getGlyphID(character))
	}
	return glyphIDs
}

func (w *rephWorker) getGlyphID(character rune) int {
	return w.lookup.GetGlyphID(int(character))
}
