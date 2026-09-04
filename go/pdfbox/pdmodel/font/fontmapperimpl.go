package font

import (
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"
	"sync"

	"github.com/shinguakira/pdfbox-go/go/fontbox"
	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf"
	"github.com/shinguakira/pdfbox-go/go/fontbox/type1"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/resources"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// fontMapperFontCache is Java's static FontCache field of FontMapperImpl.
//
// todo: static cache isn't ideal
var fontMapperFontCache = NewFontCache()

// The lazy thread safe singleton FontMapperImpl.DefaultFontProvider.
var (
	defaultFontProviderOnce sync.Once
	defaultFontProvider     FontProvider
)

// defaultFontProviderInstance returns the FileSystemFontProvider every mapper
// falls back on, building it the first time it is asked for.
func defaultFontProviderInstance() FontProvider {
	defaultFontProviderOnce.Do(func() {
		defaultFontProvider = newFileSystemFontProvider(fontMapperFontCache)
	})
	return defaultFontProvider
}

// fontMapperImpl locates non-embedded fonts via a pluggable FontProvider.
//
// Port of the package-private final class
// org.apache.pdfbox.pdmodel.font.FontMapperImpl.
type fontMapperImpl struct {
	// mu guards fontProvider, fontInfoByName and fontInfoOrder, which Java's
	// synchronized setProvider and getProvider guard.
	mu             sync.Mutex
	fontProvider   FontProvider
	fontInfoByName map[string]FontInfo
	// fontInfoOrder is the insertion order of fontInfoByName, which Java gets
	// from a LinkedHashMap; the order decides which of two equally good
	// candidates the substitution queue hands back.
	fontInfoOrder []string

	lastResortFont *ttf.TrueTypeFont

	// substitutes is the map of PostScript name substitutes, in priority order.
	substitutes map[string][]string
}

var _ FontMapper = (*fontMapperImpl)(nil)

// newFontMapperImpl returns a mapper carrying the substitutes for the standard
// 14 fonts and the last-resort font.
//
// Java's constructor wraps a failure to read the last-resort font in a
// RuntimeException; the resource is compiled into the binary here, so a failure
// means a broken build and the port panics as Java does.
func newFontMapperImpl() *fontMapperImpl {
	f := &fontMapperImpl{substitutes: map[string][]string{}}

	// substitutes for standard 14 fonts
	f.addSubstitutes("Courier",
		[]string{"CourierNew", "CourierNewPSMT", "LiberationMono", "NimbusMonL-Regu"})
	f.addSubstitutes("Courier-Bold",
		[]string{"CourierNewPS-BoldMT", "CourierNew-Bold", "LiberationMono-Bold",
			"NimbusMonL-Bold"})
	f.addSubstitutes("Courier-Oblique",
		[]string{"CourierNewPS-ItalicMT", "CourierNew-Italic", "LiberationMono-Italic",
			"NimbusMonL-ReguObli"})
	f.addSubstitutes("Courier-BoldOblique",
		[]string{"CourierNewPS-BoldItalicMT", "CourierNew-BoldItalic",
			"LiberationMono-BoldItalic", "NimbusMonL-BoldObli"})
	f.addSubstitutes("Helvetica",
		[]string{"ArialMT", "Arial", "LiberationSans", "NimbusSanL-Regu"})
	f.addSubstitutes("Helvetica-Bold",
		[]string{"Arial-BoldMT", "Arial-Bold", "LiberationSans-Bold", "NimbusSanL-Bold"})
	f.addSubstitutes("Helvetica-Oblique",
		[]string{"Arial-ItalicMT", "Arial-Italic", "Helvetica-Italic",
			"LiberationSans-Italic", "NimbusSanL-ReguItal"})
	f.addSubstitutes("Helvetica-BoldOblique",
		[]string{"Arial-BoldItalicMT", "Helvetica-BoldItalic", "LiberationSans-BoldItalic",
			"NimbusSanL-BoldItal"})
	f.addSubstitutes("Times-Roman",
		[]string{"TimesNewRomanPSMT", "TimesNewRoman", "TimesNewRomanPS", "LiberationSerif",
			"NimbusRomNo9L-Regu"})
	f.addSubstitutes("Times-Bold",
		[]string{"TimesNewRomanPS-BoldMT", "TimesNewRomanPS-Bold", "TimesNewRoman-Bold",
			"LiberationSerif-Bold", "NimbusRomNo9L-Medi"})
	f.addSubstitutes("Times-Italic",
		[]string{"TimesNewRomanPS-ItalicMT", "TimesNewRomanPS-Italic", "TimesNewRoman-Italic",
			"LiberationSerif-Italic", "NimbusRomNo9L-ReguItal"})
	f.addSubstitutes("Times-BoldItalic",
		[]string{"TimesNewRomanPS-BoldItalicMT", "TimesNewRomanPS-BoldItalic",
			"TimesNewRoman-BoldItalic", "LiberationSerif-BoldItalic",
			"NimbusRomNo9L-MediItal"})
	f.addSubstitutes("Symbol", []string{"Symbol", "SymbolMT", "StandardSymL"})
	f.addSubstitutes("ZapfDingbats", []string{"ZapfDingbatsITCbyBT-Regular", "ZapfDingbatsITC",
		"Dingbats", "MS-Gothic", "DejaVuSans"})

	// Acrobat also uses alternative names for Standard 14 fonts, which we map
	// to those above these include names such as "Arial" and "TimesNewRoman"
	//
	// Java walks a HashSet, whose order it does not define; the port sorts the
	// names so that a run repeats. Every alias maps to one of the fourteen the
	// block above already filled in, so no name here depends on another, and
	// the order cannot change the result.
	baseNames := Standard14Names()
	sort.Strings(baseNames)
	for _, baseName := range baseNames {
		if len(f.getSubstitutes(baseName)) == 0 {
			mappedName, _ := GetMappedFontName(baseName)
			f.addSubstitutes(baseName, append([]string(nil),
				f.getSubstitutes(mappedName.Name())...))
		}
	}

	// -------------------------

	const resourceName = "ttf/LiberationSans-Regular.ttf"
	resourceBytes, err := resources.Read(resourceName)
	if err != nil {
		panic(fmt.Errorf("font: resource '/org/apache/pdfbox/resources/%s' not found: %w",
			resourceName, err))
	}
	lastResortFont, err := ttf.NewParser().Parse(pdfio.NewReadBufferBytes(resourceBytes))
	if err != nil {
		panic(fmt.Errorf("font: reading the last resort font: %w", err))
	}
	f.lastResortFont = lastResortFont
	return f
}

// SetProvider sets the font service provider.
func (f *fontMapperImpl) SetProvider(fontProvider FontProvider) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.setProviderLocked(fontProvider)
}

func (f *fontMapperImpl) setProviderLocked(fontProvider FontProvider) {
	f.fontInfoByName, f.fontInfoOrder = createFontInfoByName(fontProvider.GetFontInfo())
	f.fontProvider = fontProvider
}

// GetProvider returns the font service provider. Defaults to using
// FileSystemFontProvider.
func (f *fontMapperImpl) GetProvider() FontProvider {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fontProvider == nil {
		f.setProviderLocked(defaultFontProviderInstance())
	}
	return f.fontProvider
}

// GetFontCache returns the font cache associated with this FontMapper. This
// method is needed by FontProvider subclasses.
func (f *fontMapperImpl) GetFontCache() *FontCache { return fontMapperFontCache }

// createFontInfoByName returns the lookup from every name a font goes by, in
// lower case, to the font; the second result is the order the names went in,
// which Java gets from a LinkedHashMap.
func createFontInfoByName(fontInfoList []FontInfo) (map[string]FontInfo, []string) {
	m := map[string]FontInfo{}
	var order []string
	for _, info := range fontInfoList {
		for _, name := range getPostScriptNames(info.PostScriptName()) {
			key := strings.ToLower(name)
			if _, seen := m[key]; !seen {
				order = append(order, key)
			}
			m[key] = info
		}
	}
	return m, order
}

// getPostScriptNames gets alternative names, as seen in some PDFs, e.g.
// PDFBOX-142.
//
// Java collects them into a HashSet of two, whose order it does not define and
// which drops the second name where it equals the first; the port writes both
// out, the built-in name first.
func getPostScriptNames(postScriptName string) []string {
	// built-in PostScript name
	names := []string{postScriptName}

	// remove hyphens (e.g. Arial-Black -> ArialBlack)
	if stripped := strings.ReplaceAll(postScriptName, "-", ""); stripped != postScriptName {
		names = append(names, stripped)
	}
	return names
}

// AddSubstitute adds a top-priority substitute for the given font, match being
// the PostScript name of the font to match and replace the PostScript name of
// the font to use as a replacement.
func (f *fontMapperImpl) AddSubstitute(match, replace string) {
	lowerCaseMatch := strings.ToLower(match)
	f.substitutes[lowerCaseMatch] = append(f.substitutes[lowerCaseMatch], replace)
}

func (f *fontMapperImpl) addSubstitutes(match string, replacements []string) {
	f.substitutes[strings.ToLower(match)] = replacements
}

// getSubstitutes returns the substitutes for a given font.
func (f *fontMapperImpl) getSubstitutes(postScriptName string) []string {
	return f.substitutes[strings.ToLower(strings.ReplaceAll(postScriptName, " ", ""))]
}

// getFallbackFontName attempts to find a good fallback based on the font
// descriptor.
func getFallbackFontName(fontDescriptor *PDFontDescriptor) string {
	var fontName string
	if fontDescriptor != nil {
		// heuristic detection of bold
		isBold := false
		name := fontDescriptor.FontName()
		if name != "" {
			lower := strings.ToLower(fontDescriptor.FontName())
			isBold = strings.Contains(lower, "bold") ||
				strings.Contains(lower, "black") ||
				strings.Contains(lower, "heavy")
		}

		// font descriptor flags should describe the style
		switch {
		case fontDescriptor.IsFixedPitch():
			fontName = "Courier"
			switch {
			case isBold && fontDescriptor.IsItalic():
				fontName += "-BoldOblique"
			case isBold:
				fontName += "-Bold"
			case fontDescriptor.IsItalic():
				fontName += "-Oblique"
			}
		case fontDescriptor.IsSerif():
			fontName = "Times"
			switch {
			case isBold && fontDescriptor.IsItalic():
				fontName += "-BoldItalic"
			case isBold:
				fontName += "-Bold"
			case fontDescriptor.IsItalic():
				fontName += "-Italic"
			default:
				fontName += "-Roman"
			}
		default:
			fontName = "Helvetica"
			switch {
			case isBold && fontDescriptor.IsItalic():
				fontName += "-BoldOblique"
			case isBold:
				fontName += "-Bold"
			case fontDescriptor.IsItalic():
				fontName += "-Oblique"
			}
		}
	} else {
		// if there is no FontDescriptor then we just fall back to Times Roman
		fontName = "Times-Roman"
	}
	return fontName
}

// GetTrueTypeFont finds a TrueType font with the given PostScript name, or a
// suitable substitute, or nil.
func (f *fontMapperImpl) GetTrueTypeFont(baseFont string,
	fontDescriptor *PDFontDescriptor) *FontMapping[*ttf.TrueTypeFont] {
	var trueTypeFont *ttf.TrueTypeFont
	if font := f.findFont(FontFormatTTF, baseFont); font != nil {
		trueTypeFont = font.(*ttf.TrueTypeFont)
	}
	if trueTypeFont != nil {
		return NewFontMapping(trueTypeFont, false)
	}
	// fallback - todo: i.e. fuzzy match
	fontName := getFallbackFontName(fontDescriptor)
	if font := f.findFont(FontFormatTTF, fontName); font != nil {
		trueTypeFont = font.(*ttf.TrueTypeFont)
	}
	if trueTypeFont == nil {
		// we have to return something here as TTFs aren't strictly required on
		// the system
		trueTypeFont = f.lastResortFont
	}
	return NewFontMapping(trueTypeFont, true)
}

// GetFontBoxFont finds a font with the given PostScript name, or a suitable
// substitute, or nil. This allows any font to be substituted with a PFB, TTF or
// OTF.
func (f *fontMapperImpl) GetFontBoxFont(baseFont string,
	fontDescriptor *PDFontDescriptor) *FontMapping[fontbox.FontBoxFont] {
	font := f.findFontBoxFont(baseFont)
	if font != nil {
		return NewFontMapping(font, false)
	}
	// fallback - todo: i.e. fuzzy match
	fallbackName := getFallbackFontName(fontDescriptor)
	font = f.findFontBoxFont(fallbackName)
	if font == nil {
		// we have to return something here as TTFs aren't strictly required on
		// the system
		font = f.lastResortFont
	}
	return NewFontMapping(font, true)
}

// findFontBoxFont finds a font with the given PostScript name, or a suitable
// substitute, or nil.
func (f *fontMapperImpl) findFontBoxFont(postScriptName string) fontbox.FontBoxFont {
	if font := f.findFont(FontFormatPFB, postScriptName); font != nil {
		return font.(*type1.Type1Font)
	}
	if font := f.findFont(FontFormatTTF, postScriptName); font != nil {
		return font.(*ttf.TrueTypeFont)
	}
	if font := f.findFont(FontFormatOTF, postScriptName); font != nil {
		return font.(*ttf.OpenTypeFont)
	}
	return nil
}

// findFont finds a font with the given PostScript name, or a suitable
// substitute, or nil.
//
// Java returns early for a null name, to handle damaged PDFs -- see
// PDFBOX-2884. A Go string is never null; the empty name a missing /BaseFont
// leaves behind matches nothing here and the walk falls through to nil.
func (f *fontMapperImpl) findFont(format FontFormat, postScriptName string) fontbox.FontBoxFont {
	// make sure the font provider is initialized
	f.GetProvider()

	// first try to match the PostScript name
	info := f.getFont(format, postScriptName)
	if info != nil {
		return info.Font()
	}

	// remove hyphens (e.g. Arial-Black -> ArialBlack)
	info = f.getFont(format, strings.ReplaceAll(postScriptName, "-", ""))
	if info != nil {
		return info.Font()
	}

	// then try named substitutes
	for _, substituteName := range f.getSubstitutes(postScriptName) {
		info = f.getFont(format, substituteName)
		if info != nil {
			return info.Font()
		}
	}

	// then try converting Windows names e.g. (ArialNarrow,Bold) -> (ArialNarrow-Bold)
	info = f.getFont(format, strings.ReplaceAll(postScriptName, ",", "-"))
	if info != nil {
		return info.Font()
	}

	if strings.Contains(postScriptName, ",") {
		postScriptName = postScriptName[:strings.Index(postScriptName, ",")]
		// PDFBOX-5806: try cutting font style and getting the basefont
		// eg. for "Wingdings,Bolt" to "Wingding-Regular" (including the following step)
		info = f.getFont(format, postScriptName)
		if info != nil {
			return info.Font()
		}
	}

	// try appending "-Regular", works for Wingdings on windows
	info = f.getFont(format, postScriptName+"-Regular")
	if info != nil {
		return info.Font()
	}
	// no matches
	return nil
}

// getFont finds the named font with the given format.
func (f *fontMapperImpl) getFont(format FontFormat, postScriptName string) FontInfo {
	index := strings.IndexByte(postScriptName, '+')
	// strip subset tag (happens when we substitute a corrupt embedded font, see PDFBOX-2642)
	if index > -1 {
		postScriptName = postScriptName[index+1:]
	}

	// look up the PostScript name
	f.mu.Lock()
	info := f.fontInfoByName[strings.ToLower(postScriptName)]
	f.mu.Unlock()
	if info != nil && info.Format() == format {
		slog.Debug(fmt.Sprintf("getFont('%s','%s') returns %s", format, postScriptName,
			fontInfoString(info)))
		return info
	}
	return nil
}

// GetCIDFont finds a CFF CID-Keyed font with the given PostScript name, or a
// suitable substitute, or nil. This method can also map CJK fonts via their
// CIDSystemInfo (ROS).
func (f *fontMapperImpl) GetCIDFont(baseFont string, fontDescriptor *PDFontDescriptor,
	cidSystemInfo *PDCIDSystemInfo) *CIDFontMapping {
	// try name match or substitute with OTF
	if font := f.findFont(FontFormatOTF, baseFont); font != nil {
		return NewCIDFontMapping(font.(*ttf.OpenTypeFont), nil, false)
	}

	// try name match or substitute with TTF
	if font := f.findFont(FontFormatTTF, baseFont); font != nil {
		return NewCIDFontMapping(nil, font.(*ttf.TrueTypeFont), false)
	}

	if cidSystemInfo != nil && fontDescriptor != nil {
		// "In Acrobat 3.0.1 and later, Type 0 fonts that use a CMap whose CIDSystemInfo
		// dictionary defines the Adobe-GB1, Adobe-CNS1 Adobe-Japan1, or Adobe-Korea1 character
		// collection can also be substituted." - Adobe Supplement to the ISO 32000

		collection := cidSystemInfo.Registry() + "-" + cidSystemInfo.Ordering()

		if collection == "Adobe-GB1" || collection == "Adobe-CNS1" ||
			collection == "Adobe-Japan1" || collection == "Adobe-Korea1" {
			// try automatic substitutes via character collection
			queue := f.getFontMatches(fontDescriptor, cidSystemInfo)
			bestMatch := queue.poll()
			if bestMatch != nil {
				slog.Debug("Best match", "baseFont", baseFont,
					"info", fontInfoString(bestMatch.info))
				font := bestMatch.info.Font()
				if openType, ok := font.(*ttf.OpenTypeFont); ok {
					return NewCIDFontMapping(openType, nil, true)
				} else if font != nil {
					return NewCIDFontMapping(nil, font, true)
				}
			}
		}
	}

	// last-resort fallback
	return NewCIDFontMapping(nil, f.lastResortFont, true)
}

// getFontMatches returns a list of matching fonts, scored by suitability.
// Positive scores indicate matches for certain attributes, while negative
// scores indicate mismatches. Zero scores are neutral. fontDescriptor is always
// present; cidSystemInfo may be nil.
func (f *fontMapperImpl) getFontMatches(fontDescriptor *PDFontDescriptor,
	cidSystemInfo *PDCIDSystemInfo) *fontMatchQueue {
	// make sure the font provider is initialized
	f.GetProvider()

	queue := &fontMatchQueue{}

	for _, info := range f.fontInfoValues() {
		// filter by CIDSystemInfo, if given
		if cidSystemInfo != nil && !f.isCharSetMatch(cidSystemInfo, info) {
			continue
		}

		match := &fontMatch{info: info}

		// PDFBOX-6251: Avoid DroidSansFallback unless requested,
		// because latin glyphs are not properly supported
		if info.PostScriptName() == "DroidSansFallback" &&
			!(fontDescriptor.FontFamily() == "DroidSansFallback" ||
				fontDescriptor.FontName() == "DroidSansFallback") {
			continue
		}

		// Panose is the most reliable
		if fontDescriptor.Panose() != nil && info.Panose() != nil {
			panose := fontDescriptor.Panose().Panose()
			if panose.FamilyKind() == info.Panose().FamilyKind() {
				if panose.FamilyKind() == 0 &&
					(strings.Contains(strings.ToLower(info.PostScriptName()), "barcode") ||
						strings.HasPrefix(info.PostScriptName(), "Code")) &&
					!probablyBarcodeFont(fontDescriptor) {
					// PDFBOX-4268: ignore barcode font if we aren't searching for one.
					continue
				}
				// serifs
				switch {
				case panose.SerifStyle() == info.Panose().SerifStyle():
					// exact match
					match.score += 2
				case panose.SerifStyle() >= 2 && panose.SerifStyle() <= 5 &&
					info.Panose().SerifStyle() >= 2 && info.Panose().SerifStyle() <= 5:
					// cove (serif)
					match.score++
				case panose.SerifStyle() >= 11 && panose.SerifStyle() <= 13 &&
					info.Panose().SerifStyle() >= 11 && info.Panose().SerifStyle() <= 13:
					// sans-serif
					match.score++
				case panose.SerifStyle() != 0 && info.Panose().SerifStyle() != 0:
					// mismatch
					match.score--
				}

				// weight
				weight := info.Panose().Weight()
				weightClass := fontInfoWeightClassAsPanose(info)
				if absInt(weight-weightClass) > 2 {
					// inconsistent data in system font, usWeightClass wins
					weight = weightClass
				}

				if panose.Weight() == weight {
					// exact match
					match.score += 2
				} else if panose.Weight() > 1 && weight > 1 {
					dist := float32(absInt(panose.Weight() - weight))
					match.score += 1 - float64(dist)*0.5
				}

				// todo: italic
				// ...
			}
		} else if fontDescriptor.FontWeight() > 0 && info.WeightClass() > 0 {
			// usWeightClass is pretty reliable
			dist := absFloat32(fontDescriptor.FontWeight() - float32(info.WeightClass()))
			match.score += 1 - float64(dist/100)*0.5
		} else if info.WeightClass() > 0 {
			// no weight information in the descriptor: prefer a regular weight
			//
			// Java divides two ints here, so the distance is truncated before
			// it is scaled; the port keeps the integer division.
			match.score += 1 - float64(absInt(info.WeightClass()-400)/100)*0.5
		}
		// todo: italic
		// ...

		queue.add(match)
	}
	return queue
}

// fontInfoValues is the values() of Java's LinkedHashMap, in insertion order.
func (f *fontMapperImpl) fontInfoValues() []FontInfo {
	f.mu.Lock()
	defer f.mu.Unlock()
	values := make([]FontInfo, 0, len(f.fontInfoOrder))
	for _, key := range f.fontInfoOrder {
		values = append(values, f.fontInfoByName[key])
	}
	return values
}

func probablyBarcodeFont(fontDescriptor *PDFontDescriptor) bool {
	// Java replaces a null family or name with the empty string, which is what
	// the descriptor already returns for a missing entry here.
	ff := fontDescriptor.FontFamily()
	fn := fontDescriptor.FontName()
	return strings.HasPrefix(ff, "Code") || strings.Contains(strings.ToLower(ff), "barcode") ||
		strings.HasPrefix(fn, "Code") || strings.Contains(strings.ToLower(fn), "barcode")
}

// The code page bits isCharSetMatch reads out of the "OS/2" table.
const (
	jisJapan           int64 = 1 << 17
	chineseSimplified  int64 = 1 << 18
	koreanWansung      int64 = 1 << 19
	chineseTraditional int64 = 1 << 20
	koreanJohab        int64 = 1 << 21
)

// isCharSetMatch reports whether the character set described by CIDSystemInfo
// is present in the given font. Only applies to Adobe-GB1, Adobe-CNS1,
// Adobe-Japan1, Adobe-Korea1, as per the PDF spec.
func (f *fontMapperImpl) isCharSetMatch(cidSystemInfo *PDCIDSystemInfo, info FontInfo) bool {
	ordering := cidSystemInfo.Ordering()
	if ordering == "" {
		return false
	}
	if info.CIDSystemInfo() != nil {
		if info.CIDSystemInfo().Registry() == cidSystemInfo.Registry() &&
			info.CIDSystemInfo().Ordering() == ordering {
			return true
		}
		if info.CIDSystemInfo().Ordering() != "Identity" {
			return false
		}
		// PDFBOX-6249: Adobe-Identity-0 fonts (Noto CJK, Source Han) never match a ROS by
		// name; fall through to the code page bits
	}
	codePageRange := fontInfoCodePageRange(info)

	postScriptName := info.PostScriptName()
	if postScriptName != "" {
		if postScriptName == "MalgunGothic-Semilight" {
			// PDFBOX-4793 and PDF.js 10699: This font has only Korean, but has bits 17-21 set.
			codePageRange &= ^(jisJapan | chineseSimplified | chineseTraditional)
		}
		if strings.HasPrefix(postScriptName, "NotoSans") {
			suffix := postScriptName[8:]
			if strings.HasPrefix(suffix, "HK") ||
				strings.HasPrefix(suffix, "TC") ||
				strings.HasPrefix(suffix, "CJKsc") ||
				strings.HasPrefix(suffix, "CJKtc") ||
				strings.HasPrefix(suffix, "CJKhk") ||
				strings.HasPrefix(suffix, "KR") ||
				strings.HasPrefix(suffix, "CJKkr") {
				// PDFBOX-6249: These fonts have bit 17 set.
				codePageRange &= ^jisJapan
			}
			if strings.HasPrefix(suffix, "JP") ||
				strings.HasPrefix(suffix, "CJKjp") ||
				strings.HasPrefix(suffix, "KR") ||
				strings.HasPrefix(suffix, "CJKkr") {
				// PDFBOX-6249: These fonts have at least one chinese bit set.
				codePageRange &= ^(chineseSimplified | chineseTraditional)
			}
		}
	}
	switch {
	case ordering == "GB1" && codePageRange&chineseSimplified == chineseSimplified:
		return true
	case ordering == "CNS1" && codePageRange&chineseTraditional == chineseTraditional:
		return true
	case ordering == "Japan1" && codePageRange&jisJapan == jisJapan:
		return true
	default:
		return ordering == "Korea1" &&
			(codePageRange&koreanWansung == koreanWansung ||
				codePageRange&koreanJohab == koreanJohab)
	}
}

// fontMatch is a potential match for a font substitution.
//
// Port of the private static class FontMapperImpl.FontMatch, which implements
// Comparable<FontMatch>.
type fontMatch struct {
	score float64
	info  FontInfo
}

// compareTo orders the better match first, which is what FontMatch.compareTo
// does by comparing the other score against this one.
func (m *fontMatch) compareTo(match *fontMatch) int {
	return doubleCompare(match.score, m.score)
}

// doubleCompare is java.lang.Double.compare, which orders -0.0 below 0.0 and
// NaN above every number; the raw-bits comparison it falls back on is signed,
// and it collapses every NaN to one pattern first.
func doubleCompare(d1, d2 float64) int {
	if d1 < d2 {
		return -1
	}
	if d1 > d2 {
		return 1
	}
	bits1 := int64(math.Float64bits(canonicalNaN(d1)))
	bits2 := int64(math.Float64bits(canonicalNaN(d2)))
	switch {
	case bits1 == bits2:
		return 0
	case bits1 < bits2:
		return -1
	default:
		return 1
	}
}

// canonicalNaN is what Java's Double.doubleToLongBits does to a NaN.
func canonicalNaN(d float64) float64 {
	if math.IsNaN(d) {
		return math.Float64frombits(0x7ff8000000000000)
	}
	return d
}

// fontMatchQueue is the binary heap java.util.PriorityQueue keeps, ported so
// that two candidates of equal score come back in the order Java hands them
// over. A sort would order the ties differently.
type fontMatchQueue struct {
	es []*fontMatch
}

// add is PriorityQueue.offer.
func (q *fontMatchQueue) add(e *fontMatch) {
	i := len(q.es)
	q.es = append(q.es, e)
	if i == 0 {
		q.es[0] = e
	} else {
		q.siftUp(i, e)
	}
}

// siftUp is PriorityQueue.siftUpComparable.
func (q *fontMatchQueue) siftUp(k int, x *fontMatch) {
	for k > 0 {
		parent := (k - 1) >> 1
		e := q.es[parent]
		if x.compareTo(e) >= 0 {
			break
		}
		q.es[k] = e
		k = parent
	}
	q.es[k] = x
}

// siftDown is PriorityQueue.siftDownComparable.
func (q *fontMatchQueue) siftDown(k int, x *fontMatch, n int) {
	half := n >> 1
	for k < half {
		child := (k << 1) + 1 // assume left child is least
		c := q.es[child]
		right := child + 1
		if right < n && c.compareTo(q.es[right]) > 0 {
			child = right
			c = q.es[child]
		}
		if x.compareTo(c) <= 0 {
			break
		}
		q.es[k] = c
		k = child
	}
	q.es[k] = x
}

// peek returns the best match without removing it, or nil where the queue is
// empty.
func (q *fontMatchQueue) peek() *fontMatch {
	if len(q.es) == 0 {
		return nil
	}
	return q.es[0]
}

// poll removes and returns the best match, or nil where the queue is empty.
func (q *fontMatchQueue) poll() *fontMatch {
	if len(q.es) == 0 {
		return nil
	}
	result := q.es[0]
	n := len(q.es) - 1
	x := q.es[n]
	q.es = q.es[:n]
	if n > 0 {
		q.siftDown(0, x, n)
	}
	return result
}

// isEmpty reports whether the queue holds nothing.
func (q *fontMatchQueue) isEmpty() bool { return len(q.es) == 0 }

// printMatches prints all matches and returns the best match.
//
// Port of the private FontMapperImpl.printMatches, which nothing in the Java
// calls; it is what a developer reaches for when a substitution goes wrong.
func printMatches(queue *fontMatchQueue) *fontMatch {
	bestMatch := queue.peek()
	fmt.Println("-------")
	for !queue.isEmpty() {
		match := queue.poll()
		info := match.info
		fmt.Println(fmt.Sprint(match.score) + " | " + toHexString(info.MacStyle()) + " " +
			toHexString(info.FamilyClass()) + " " + panoseString(info.Panose()) + " " +
			cidSystemInfoString(info.CIDSystemInfo()) + " " + info.PostScriptName() + " " +
			info.Format().String())
	}
	fmt.Println("-------")
	return bestMatch
}

// panoseString is what Java's string concatenation makes of a Panose that may
// be null.
func panoseString(panose *PDPanoseClassification) string {
	if panose == nil {
		return "null"
	}
	return panose.String()
}

// cidSystemInfoString is what Java's string concatenation makes of a
// CIDSystemInfo that may be null.
func cidSystemInfoString(ros *CIDSystemInfo) string {
	if ros == nil {
		return "null"
	}
	return ros.String()
}

// absInt is Math.abs for an int.
func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

// absFloat32 is Math.abs for a float.
func absFloat32(value float32) float32 {
	if value < 0 {
		return -value
	}
	return value
}
