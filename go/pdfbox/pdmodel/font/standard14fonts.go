// Package font reads the fonts a PDF draws its text with.
//
// Port of org.apache.pdfbox.pdmodel.font.
package font

import (
	"bufio"
	"fmt"
	"strings"
	"sync"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/fontbox"
	"github.com/shinguakira/pdfbox-go/go/fontbox/afm"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font/encoding"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/resources"
)

// FontName is one of the fourteen fonts every PDF reader is required to have.
//
// Port of org.apache.pdfbox.pdmodel.font.Standard14Fonts.FontName. Java's enum
// carries the PostScript name; the port makes the name the value.
type FontName string

// The fourteen standard fonts.
const (
	TimesRoman           FontName = "Times-Roman"
	TimesBold            FontName = "Times-Bold"
	TimesItalic          FontName = "Times-Italic"
	TimesBoldItalic      FontName = "Times-BoldItalic"
	Helvetica            FontName = "Helvetica"
	HelveticaBold        FontName = "Helvetica-Bold"
	HelveticaOblique     FontName = "Helvetica-Oblique"
	HelveticaBoldOblique FontName = "Helvetica-BoldOblique"
	Courier              FontName = "Courier"
	CourierBold          FontName = "Courier-Bold"
	CourierOblique       FontName = "Courier-Oblique"
	CourierBoldOblique   FontName = "Courier-BoldOblique"
	SymbolFontName       FontName = "Symbol"
	ZapfDingbatsFontName FontName = "ZapfDingbats"
)

// Name returns the PostScript name of the font.
func (n FontName) Name() string { return string(n) }

// String returns the PostScript name of the font.
func (n FontName) String() string { return string(n) }

// standard14Aliases maps every name a standard 14 font is known by onto the
// font itself.
var standard14Aliases = map[string]FontName{
	// the 14 standard fonts
	string(Courier):              Courier,
	string(CourierBold):          CourierBold,
	string(CourierBoldOblique):   CourierBoldOblique,
	string(CourierOblique):       CourierOblique,
	string(Helvetica):            Helvetica,
	string(HelveticaBold):        HelveticaBold,
	string(HelveticaBoldOblique): HelveticaBoldOblique,
	string(HelveticaOblique):     HelveticaOblique,
	string(TimesRoman):           TimesRoman,
	string(TimesBold):            TimesBold,
	string(TimesBoldItalic):      TimesBoldItalic,
	string(TimesItalic):          TimesItalic,
	string(SymbolFontName):       SymbolFontName,
	string(ZapfDingbatsFontName): ZapfDingbatsFontName,

	// alternative names from Adobe Supplement to the ISO 32000
	"CourierCourierNew":        Courier,
	"CourierNew":               Courier,
	"CourierNew,Italic":        CourierOblique,
	"CourierNew,Bold":          CourierBold,
	"CourierNew,BoldItalic":    CourierBoldOblique,
	"Arial":                    Helvetica,
	"Arial,Italic":             HelveticaOblique,
	"Arial,Bold":               HelveticaBold,
	"Arial,BoldItalic":         HelveticaBoldOblique,
	"TimesNewRoman":            TimesRoman,
	"TimesNewRoman,Italic":     TimesItalic,
	"TimesNewRoman,Bold":       TimesBold,
	"TimesNewRoman,BoldItalic": TimesBoldItalic,

	// Acrobat treats these fonts as "standard 14" too (at least Acrobat
	// preflight says so)
	"Symbol,Italic":     SymbolFontName,
	"Symbol,Bold":       SymbolFontName,
	"Symbol,BoldItalic": SymbolFontName,
	"Times":             TimesRoman,
	"Times,Italic":      TimesItalic,
	"Times,Bold":        TimesBold,
	"Times,BoldItalic":  TimesBoldItalic,

	// PDFBOX-3457: PDF.js file bug864847.pdf
	"ArialMT":            Helvetica,
	"Arial-ItalicMT":     HelveticaOblique,
	"Arial-BoldMT":       HelveticaBold,
	"Arial-BoldItalicMT": HelveticaBoldOblique,
}

// The metrics and the generic font of each standard 14 font, read the first
// time they are asked for.
var (
	standard14Fonts       = map[FontName]*afm.FontMetrics{}
	standard14FontsLock   sync.Mutex
	standard14Generic     = map[FontName]fontbox.FontBoxFont{}
	standard14GenericLock sync.Mutex
)

// loadStandard14Metrics reads the AFM file of one standard 14 font.
func loadStandard14Metrics(fontName FontName) (*afm.FontMetrics, error) {
	resourceName := "afm/" + fontName.Name() + ".afm"
	resourceAsStream, err := resources.Open(resourceName)
	if err != nil {
		return nil, fmt.Errorf("font: resource '/org/apache/pdfbox/resources/%s' not found: %w",
			resourceName, err)
	}
	defer resourceAsStream.Close()
	parser := afm.NewAFMParser(bufio.NewReader(resourceAsStream))
	return parser.ParseReduced(true)
}

// GetAFM returns the metrics of the named standard 14 font, or nil where the
// name is not one of them.
//
// It panics where the font is a standard 14 font whose metrics cannot be read,
// which is what Java's IllegalArgumentException does; the AFM files are
// embedded, so that cannot happen short of a corrupted binary.
func GetAFM(fontName string) *afm.FontMetrics {
	baseName, ok := standard14Aliases[fontName]
	if !ok {
		return nil
	}
	standard14FontsLock.Lock()
	defer standard14FontsLock.Unlock()
	if metrics, ok := standard14Fonts[baseName]; ok {
		return metrics
	}
	metrics, err := loadStandard14Metrics(baseName)
	if err != nil {
		panic(err)
	}
	standard14Fonts[baseName] = metrics
	return metrics
}

// Standard14ContainsName reports whether the given name is one of the standard
// 14 fonts, under any of the names they go by.
func Standard14ContainsName(fontName string) bool {
	_, ok := standard14Aliases[fontName]
	return ok
}

// Standard14Names returns every name a standard 14 font is known by.
func Standard14Names() []string {
	names := make([]string, 0, len(standard14Aliases))
	for name := range standard14Aliases {
		names = append(names, name)
	}
	return names
}

// GetMappedFontName returns which standard 14 font the given name stands for.
// The second result is false where the name is not one of them.
func GetMappedFontName(fontName string) (FontName, bool) {
	baseName, ok := standard14Aliases[fontName]
	return baseName, ok
}

// getMappedFont returns the font the standard 14 font is drawn with, which is
// whatever the system has that stands in for it.
func getMappedFont(baseName FontName) (fontbox.FontBoxFont, error) {
	standard14GenericLock.Lock()
	defer standard14GenericLock.Unlock()
	if font, ok := standard14Generic[baseName]; ok {
		return font, nil
	}
	type1Font, err := NewPDType1FontStandard14(baseName)
	if err != nil {
		return nil, err
	}
	font := type1Font.FontBoxFont()
	standard14Generic[baseName] = font
	return font, nil
}

// GetGlyphPath returns the outline of the named glyph of a standard 14 font, or
// an empty path where the font has no such glyph.
func GetGlyphPath(baseName FontName, glyphName string) (*geom.Path2D, error) {
	// copied and adapted from PDType1Font.getNameInFont(String)
	if glyphName != ".notdef" {
		mappedFont, err := getMappedFont(baseName)
		if err != nil {
			return nil, err
		}
		if mappedFont != nil {
			hasGlyph, err := mappedFont.HasGlyph(glyphName)
			if err != nil {
				return nil, err
			}
			if hasGlyph {
				return mappedFont.GetPath(glyphName)
			}
			unicodes := standard14GlyphList(baseName).ToUnicode(glyphName)
			if runes := []rune(unicodes); len(runes) == 1 {
				uniName := getUniNameOfCodePoint(int(runes[0]))
				hasGlyph, err := mappedFont.HasGlyph(uniName)
				if err != nil {
					return nil, err
				}
				if hasGlyph {
					return mappedFont.GetPath(uniName)
				}
			}
			name, err := mappedFont.Name()
			if err != nil {
				return nil, err
			}
			if name == "SymbolMT" {
				if code, ok := encoding.SymbolEncodingInstance.NameToCodeMap()[glyphName]; ok {
					uniName := getUniNameOfCodePoint(code + 0xF000)
					hasGlyph, err := mappedFont.HasGlyph(uniName)
					if err != nil {
						return nil, err
					}
					if hasGlyph {
						return mappedFont.GetPath(uniName)
					}
				}
			}
		}
	}
	return geom.NewPathFloat(), nil
}

// standard14GlyphList returns the glyph list a standard 14 font is read
// through.
func standard14GlyphList(baseName FontName) *encoding.GlyphList {
	if baseName == ZapfDingbatsFontName {
		return encoding.ZapfDingbats()
	}
	return encoding.AdobeGlyphList()
}

// getUniNameOfCodePoint returns the uniXXXX name of a code point.
//
// Port of org.apache.pdfbox.pdmodel.font.UniUtil.
func getUniNameOfCodePoint(codePoint int) string {
	// faster than String.format("uni%04X", codePoint)
	hex := strings.ToUpper(fmt.Sprintf("%x", codePoint))
	switch len(hex) {
	case 1:
		return "uni000" + hex
	case 2:
		return "uni00" + hex
	case 3:
		return "uni0" + hex
	default:
		return "uni" + hex
	}
}
