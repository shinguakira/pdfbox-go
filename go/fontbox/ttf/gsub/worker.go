package gsub

import (
	"log/slog"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf/model"
)

// GsubWorker applies a font's glyph substitutions to a run of glyphs.
//
// Port of the org.apache.fontbox.ttf.gsub.GsubWorker interface.
type GsubWorker interface {
	// ApplyTransforms returns the glyphs the substitutions leave behind.
	ApplyTransforms(originalGlyphIDs []int) []int
}

// CmapLookup is what a worker looks a character up in.
//
// Java takes org.apache.fontbox.ttf.CmapLookup here; that package imports this
// one for SubstitutingCmapLookup, so the port declares the two methods it needs
// rather than the cycle Java allows.
type CmapLookup interface {
	// GetGlyphID returns the glyph for the given code point.
	GetGlyphID(codePointAt int) int

	// GetCharCodes returns the code points that map to the given glyph.
	GetCharCodes(gid int) []int
}

// glyphIDSeparator is what the tokenizer puts between glyph ids.
const glyphIDSeparator = "_"

// CompoundCharacterTokenizer splits a run of glyphs where any of the given
// compound words appears.
//
// Port of org.apache.fontbox.ttf.gsub.CompoundCharacterTokenizer.
type CompoundCharacterTokenizer struct {
	regexExpression *regexp.Regexp
}

// NewCompoundCharacterTokenizer returns a tokenizer over the given compound
// words, which must each start and end with the separator.
//
// Java throws IllegalArgumentException where they do not, which is unchecked;
// the port panics.
func NewCompoundCharacterTokenizer(compoundWords []string) *CompoundCharacterTokenizer {
	validateCompoundWords(compoundWords)
	return &CompoundCharacterTokenizer{
		regexExpression: regexp.MustCompile(regexFromTokens(compoundWords)),
	}
}

func validateCompoundWords(compoundWords []string) {
	if len(compoundWords) == 0 {
		panic("Compound words cannot be null or empty")
	}
	// Ensure all word are starting and ending with the glyphIDSeparator
	for _, word := range compoundWords {
		if !strings.HasPrefix(word, glyphIDSeparator) || !strings.HasSuffix(word, glyphIDSeparator) {
			panic("Compound words should start and end with " + glyphIDSeparator)
		}
	}
}

// Tokenize splits the text at every compound word it holds.
func (t *CompoundCharacterTokenizer) Tokenize(text string) []string {
	var tokens []string
	lastIndexOfPrevMatch := 0
	for lastIndexOfPrevMatch <= len(text) {
		// this is where the magic happens: the regexp is used to find a
		// matching pattern for substitution
		//
		// Java's Matcher.find(int) resets and searches from the index, which is
		// what searching the tail comes to.
		loc := t.regexExpression.FindStringIndex(text[lastIndexOfPrevMatch:])
		if loc == nil {
			break
		}
		beginIndexOfNextMatch := lastIndexOfPrevMatch + loc[0]
		endIndexOfNextMatch := lastIndexOfPrevMatch + loc[1]
		prevToken := text[lastIndexOfPrevMatch:beginIndexOfNextMatch]
		if prevToken != "" {
			tokens = append(tokens, prevToken)
		}
		tokens = append(tokens, text[beginIndexOfNextMatch:endIndexOfNextMatch])
		lastIndexOfPrevMatch = endIndexOfNextMatch
		if lastIndexOfPrevMatch < len(text) && text[lastIndexOfPrevMatch] != '_' {
			// because it is sometimes positioned after the "_", but it should
			// be positioned before the "_"
			lastIndexOfPrevMatch--
		}
	}
	if lastIndexOfPrevMatch < len(text) {
		tokens = append(tokens, text[lastIndexOfPrevMatch:])
	}
	return tokens
}

func regexFromTokens(compoundWords []string) string {
	quoted := make([]string, len(compoundWords))
	for i, word := range compoundWords {
		// Java joins the raw strings; every one of them is digits and
		// underscores, so quoting changes nothing and keeps the pattern honest.
		quoted[i] = regexp.QuoteMeta(word)
	}
	return "(" + strings.Join(quoted, ")|(") + ")"
}

// GlyphArraySplitter splits a run of glyphs into the chunks a feature replaces.
//
// Port of the org.apache.fontbox.ttf.gsub.GlyphArraySplitter interface.
type GlyphArraySplitter interface {
	// Split splits the given glyphs.
	Split(glyphIDs []int) [][]int
}

// GlyphArraySplitterRegexImpl splits a run of glyphs with a regular expression.
//
// Port of org.apache.fontbox.ttf.gsub.GlyphArraySplitterRegexImpl.
type GlyphArraySplitterRegexImpl struct {
	compoundCharacterTokenizer *CompoundCharacterTokenizer
}

var _ GlyphArraySplitter = (*GlyphArraySplitterRegexImpl)(nil)

// NewGlyphArraySplitterRegexImpl returns a splitter over the given matchers.
func NewGlyphArraySplitterRegexImpl(matchers []model.GlyphKey) *GlyphArraySplitterRegexImpl {
	return &GlyphArraySplitterRegexImpl{
		compoundCharacterTokenizer: NewCompoundCharacterTokenizer(matchersAsStrings(matchers)),
	}
}

// Split splits the given glyphs.
func (s *GlyphArraySplitterRegexImpl) Split(glyphIDs []int) [][]int {
	originalGlyphsAsText := convertGlyphIDsToString(glyphIDs)
	tokens := s.compoundCharacterTokenizer.Tokenize(originalGlyphsAsText)
	modifiedGlyphs := make([][]int, 0, len(tokens))
	for _, token := range tokens {
		modifiedGlyphs = append(modifiedGlyphs, convertGlyphIDsToList(token))
	}
	return modifiedGlyphs
}

// matchersAsStrings renders the matchers, longest first so that the alternation
// prefers the longer match.
//
// Java uses a TreeSet with a comparator that puts the larger string first, and
// among equal lengths the later one; the port sorts the same way.
func matchersAsStrings(matchers []model.GlyphKey) []string {
	stringMatchers := make([]string, 0, len(matchers))
	seen := map[string]bool{}
	for _, glyphIDs := range matchers {
		value := convertGlyphIDsToString(glyphIDs.IDs())
		if !seen[value] {
			seen[value] = true
			stringMatchers = append(stringMatchers, value)
		}
	}
	sort.Slice(stringMatchers, func(i, j int) bool {
		s1, s2 := stringMatchers[i], stringMatchers[j]
		if len(s1) == len(s2) {
			return s2 < s1
		}
		return len(s2) < len(s1)
	})
	return stringMatchers
}

func convertGlyphIDsToString(glyphIDs []int) string {
	var sb strings.Builder
	sb.WriteString(glyphIDSeparator)
	for _, glyphID := range glyphIDs {
		sb.WriteString(strconv.Itoa(glyphID))
		sb.WriteString(glyphIDSeparator)
	}
	return sb.String()
}

func convertGlyphIDsToList(glyphIDsAsString string) []int {
	var gsubProcessedGlyphsIDs []int
	for _, glyphID := range strings.Split(glyphIDsAsString, glyphIDSeparator) {
		glyphID = strings.TrimSpace(glyphID)
		if glyphID == "" {
			continue
		}
		value, err := strconv.Atoi(glyphID)
		if err != nil {
			// Java's Integer.valueOf throws NumberFormatException, which is
			// unchecked; nothing but digits ever reaches here.
			panic("gsub: For input string: " + glyphID)
		}
		gsubProcessedGlyphsIDs = append(gsubProcessedGlyphsIDs, value)
	}
	return gsubProcessedGlyphsIDs
}

// featureApplier is the applyGsubFeature every worker carries, which they all
// write out identically.
type featureApplier struct {
	splitters map[string]GlyphArraySplitter
}

func newFeatureApplier() featureApplier {
	return featureApplier{splitters: map[string]GlyphArraySplitter{}}
}

// applyGsubFeature replaces every run of glyphs the feature knows about.
//
// Java caches the splitter per feature name in a WeakHashMap; Go has no weak
// map and the cache is per worker, so a plain map stands in for it.
func (a *featureApplier) applyGsubFeature(scriptFeature model.ScriptFeature,
	originalGlyphs []int) []int {
	allGlyphIDsForSubstitution := scriptFeature.AllGlyphIDsForSubstitution()
	if len(allGlyphIDsForSubstitution) == 0 {
		// not stopping here results in really weird output, the regex goes wild
		slog.Debug("AllGlyphIDsForSubstitution is empty", "feature", scriptFeature.Name())
		return originalGlyphs
	}
	glyphArraySplitter, ok := a.splitters[scriptFeature.Name()]
	if !ok {
		glyphArraySplitter = NewGlyphArraySplitterRegexImpl(allGlyphIDsForSubstitution)
		a.splitters[scriptFeature.Name()] = glyphArraySplitter
	}
	tokens := glyphArraySplitter.Split(originalGlyphs)
	var gsubProcessedGlyphs []int
	for _, chunk := range tokens {
		if scriptFeature.CanReplaceGlyphs(chunk) {
			// gsub system kicks in, you get the glyphId directly
			gsubProcessedGlyphs = append(gsubProcessedGlyphs,
				scriptFeature.ReplacementForGlyphs(chunk)...)
		} else {
			gsubProcessedGlyphs = append(gsubProcessedGlyphs, chunk...)
		}
	}
	return gsubProcessedGlyphs
}

// applyFeaturesInOrder walks the features in the order the worker declares,
// applying each one the font supports.
func (a *featureApplier) applyFeaturesInOrder(gsubData model.GsubData, featuresInOrder []string,
	glyphs []int) []int {
	for _, feature := range featuresInOrder {
		if !gsubData.IsFeatureSupported(feature) {
			slog.Debug("the feature was not found", "feature", feature)
			continue
		}
		slog.Debug("applying the feature", "feature", feature)
		glyphs = a.applyGsubFeature(gsubData.Feature(feature), glyphs)
	}
	return glyphs
}

// DefaultGsubWorker performs no substitutions at all.
//
// Port of org.apache.fontbox.ttf.gsub.DefaultGsubWorker.
type DefaultGsubWorker struct{}

var _ GsubWorker = (*DefaultGsubWorker)(nil)

// ApplyTransforms returns the glyphs unchanged.
func (w *DefaultGsubWorker) ApplyTransforms(originalGlyphIDs []int) []int {
	slog.Warn("DefaultGsubWorker does not perform actual GSUB substitutions. " +
		"Perhaps the selected language is not yet supported by the FontBox library.")
	// Java wraps the result read-only to prevent accidental modifications of
	// the source list; the port copies for the same reason.
	return slices.Clone(originalGlyphIDs)
}

// dfltFeaturesInOrder is the features GsubWorkerForDflt applies.
var dfltFeaturesInOrder = []string{"ccmp", "liga", "clig", "calt"}

// GsubWorkerForDflt is the default-script implementation of the GSUB system.
//
// Port of org.apache.fontbox.ttf.gsub.GsubWorkerForDflt.
type GsubWorkerForDflt struct {
	featureApplier

	gsubData model.GsubData
}

var _ GsubWorker = (*GsubWorkerForDflt)(nil)

func newGsubWorkerForDflt(gsubData model.GsubData) *GsubWorkerForDflt {
	return &GsubWorkerForDflt{featureApplier: newFeatureApplier(), gsubData: gsubData}
}

// ApplyTransforms returns the glyphs the substitutions leave behind.
func (w *GsubWorkerForDflt) ApplyTransforms(originalGlyphIDs []int) []int {
	return slices.Clone(w.applyFeaturesInOrder(w.gsubData, dfltFeaturesInOrder, originalGlyphIDs))
}

// latinFeaturesInOrder is the features GsubWorkerForLatin applies.
var latinFeaturesInOrder = []string{"ccmp", "liga", "clig"}

// GsubWorkerForLatin is the Latin-specific implementation of the GSUB system.
//
// Port of org.apache.fontbox.ttf.gsub.GsubWorkerForLatin.
type GsubWorkerForLatin struct {
	featureApplier

	gsubData model.GsubData
}

var _ GsubWorker = (*GsubWorkerForLatin)(nil)

func newGsubWorkerForLatin(gsubData model.GsubData) *GsubWorkerForLatin {
	return &GsubWorkerForLatin{featureApplier: newFeatureApplier(), gsubData: gsubData}
}

// ApplyTransforms returns the glyphs the substitutions leave behind.
func (w *GsubWorkerForLatin) ApplyTransforms(originalGlyphIDs []int) []int {
	return slices.Clone(w.applyFeaturesInOrder(w.gsubData, latinFeaturesInOrder, originalGlyphIDs))
}

// Factory chooses the worker a font's substitution data needs.
//
// Port of org.apache.fontbox.ttf.gsub.GsubWorkerFactory.
type Factory struct{}

// NewFactory returns a worker factory.
func NewFactory() *Factory { return &Factory{} }

// GetGsubWorker returns the worker for the language of the given data.
func (f *Factory) GetGsubWorker(cmapLookup CmapLookup, gsubData model.GsubData) GsubWorker {
	// TODO this needs to be redesigned / improved because if a font supports
	// several languages, it will choose one of them and maybe not the one
	// expected. See also PDFBOX-5700 and PDFBOX-5729. For example,
	// NotoSans-Regular hits Devanagari first. See also
	// GlyphSubstitutionDataExtractor.getSupportedLanguage() which decides the
	// language?!
	slog.Debug("choosing a GSUB worker", "language", gsubData.Language())
	switch gsubData.Language() {
	case model.Bengali:
		return newGsubWorkerForBengali(cmapLookup, gsubData)
	case model.Devanagari:
		return newGsubWorkerForDevanagari(cmapLookup, gsubData)
	case model.Gujarati:
		return newGsubWorkerForGujarati(cmapLookup, gsubData)
	case model.Latin:
		return newGsubWorkerForLatin(gsubData)
	case model.Dflt:
		return newGsubWorkerForDflt(gsubData)
	default:
		// model.Tamil: TODO implement me
		return &DefaultGsubWorker{}
	}
}
