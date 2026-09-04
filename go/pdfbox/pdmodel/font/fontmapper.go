package font

import (
	"github.com/shinguakira/pdfbox-go/go/fontbox"
	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf"
)

// FontMapper locates non-embedded fonts. If you implement this then you're
// responsible for caching the fonts.
//
// Port of org.apache.pdfbox.pdmodel.font.FontMapper. Java recommends a
// SoftReference<FontBoxFont> for that cache; Go has no soft reference, so
// FontCache holds its fonts outright.
type FontMapper interface {
	// GetTrueTypeFont finds a TrueType font with the given PostScript name, or
	// a suitable substitute, or nil.
	GetTrueTypeFont(baseFont string, fontDescriptor *PDFontDescriptor) *FontMapping[*ttf.TrueTypeFont]

	// GetFontBoxFont finds a font with the given PostScript name, or a suitable
	// substitute, or nil. This allows any font to be substituted with a PFB,
	// TTF or OTF.
	GetFontBoxFont(baseFont string, fontDescriptor *PDFontDescriptor) *FontMapping[fontbox.FontBoxFont]

	// GetCIDFont finds a CFF CID-Keyed font with the given PostScript name, or
	// a suitable substitute, or nil. This method can also map CJK fonts via
	// their CIDSystemInfo (ROS).
	GetCIDFont(baseFont string, fontDescriptor *PDFontDescriptor,
		cidSystemInfo *PDCIDSystemInfo) *CIDFontMapping
}
