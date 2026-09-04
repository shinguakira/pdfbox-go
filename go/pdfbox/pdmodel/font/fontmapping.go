package font

import (
	"github.com/shinguakira/pdfbox-go/go/fontbox"
	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf"
)

// FontMapping is a font mapping from a PDF font to a FontBox font.
//
// Port of org.apache.pdfbox.pdmodel.font.FontMapping<T extends FontBoxFont>.
type FontMapping[T fontbox.FontBoxFont] struct {
	font       T
	isFallback bool
}

// NewFontMapping returns the mapping of a PDF font to the given FontBox font.
func NewFontMapping[T fontbox.FontBoxFont](font T, isFallback bool) *FontMapping[T] {
	return &FontMapping[T]{font: font, isFallback: isFallback}
}

// Font returns the mapped, FontBox font. This is never nil.
func (m *FontMapping[T]) Font() T { return m.font }

// IsFallback reports whether the mapped font is a fallback, i.e. a substitute
// based on basic font style, such as bold/italic, rather than font name.
func (m *FontMapping[T]) IsFallback() bool { return m.isFallback }

// CIDFontMapping is a kind of FontMapping which allows for an additional
// TrueTypeFont substitute to be provided if a CID font is not available.
//
// Port of org.apache.pdfbox.pdmodel.font.CIDFontMapping, which extends
// FontMapping<OpenTypeFont>.
type CIDFontMapping struct {
	FontMapping[*ttf.OpenTypeFont]

	trueTypeFont fontbox.FontBoxFont
}

// NewCIDFontMapping returns the mapping of a CID font to the given fonts.
func NewCIDFontMapping(font *ttf.OpenTypeFont, fontBoxFont fontbox.FontBoxFont,
	isFallback bool) *CIDFontMapping {
	return &CIDFontMapping{
		FontMapping:  FontMapping[*ttf.OpenTypeFont]{font: font, isFallback: isFallback},
		trueTypeFont: fontBoxFont,
	}
}

// TrueTypeFont returns a TrueType font when IsCIDFont is false, otherwise nil.
//
// Java's doc comment says the other way round; what the method does, and what
// every caller relies on, is return the font the constructor was handed.
func (m *CIDFontMapping) TrueTypeFont() fontbox.FontBoxFont { return m.trueTypeFont }

// IsCIDFont reports whether this is a CID font.
func (m *CIDFontMapping) IsCIDFont() bool { return m.Font() != nil }
