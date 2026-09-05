package rendering

// The outlines of the glyphs of one font, kept so that a repeated character is
// only converted once.
//
// Port of org.apache.pdfbox.rendering.GlyphCache, which Java declares final and
// package-private.

import (
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
)

// glyphCache holds the outline of each character code of one font.
type glyphCache struct {
	font  font.PDVectorFont
	cache map[int]*geom.Path2D
}

// newGlyphCache returns an empty cache over the given font.
func newGlyphCache(f font.PDVectorFont) *glyphCache {
	return &glyphCache{font: f, cache: map[int]*geom.Path2D{}}
}

// pathForCharacterCode returns the outline of the glyph the given code names,
// and an empty path where the font cannot supply one.
func (c *glyphCache) pathForCharacterCode(code int) *geom.Path2D {
	if path, cached := c.cache[code]; cached {
		return path
	}
	path, err := c.pathOf(code)
	if err != nil {
		// todo: escalate this error?
		slog.Error("rendering: glyph rendering failed",
			"code", code, "font", c.fontName(), "err", err)
		return geom.NewPathFloat()
	}
	c.cache[code] = path
	return path
}

// pathOf is the body of pathForCharacterCode, with the reporting of a missing
// glyph, which Java writes inline in the try block.
func (c *glyphCache) pathOf(code int) (*geom.Path2D, error) {
	hasGlyph, err := c.font.HasGlyphForCode(code)
	if err != nil {
		return nil, err
	}
	if !hasGlyph {
		fontName := c.fontName()
		switch f := c.font.(type) {
		case *font.PDType0Font:
			slog.Warn("rendering: no glyph for code",
				"code", code, "cid", f.CodeToCID(code), "font", fontName)
		case font.PDSimpleFont:
			slog.Warn("rendering: no glyph for code",
				"code", code, "font", fontName,
				"embeddedOrSystemFont", fontBoxFontName(f))
			if code == 10 && f.IsStandard14() {
				// PDFBOX-4001 return empty path for line feed on std14
				path := geom.NewPathFloat()
				c.cache[code] = path
				return path, nil
			}
		default:
			slog.Warn("rendering: no glyph for code", "code", code, "font", fontName)
		}
	}
	return c.font.GetNormalizedPath(code)
}

// fontName is Java's `((PDFontLike) font).getName()`, the cast this cache makes
// on the way to reporting.
func (c *glyphCache) fontName() string {
	if named, isNamed := c.font.(interface{ Name() string }); isNamed {
		return named.Name()
	}
	return ""
}

// fontBoxFontName is the name of the font program a simple font draws from,
// which Java reports beside the missing code.
func fontBoxFontName(f font.PDSimpleFont) string {
	program := f.FontBoxFont()
	if program == nil {
		return ""
	}
	name, err := program.Name()
	if err != nil {
		return ""
	}
	return name
}
