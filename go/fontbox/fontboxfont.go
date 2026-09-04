// Package fontbox is what every font this library reads has in common.
//
// Port of the root of org.apache.fontbox.
package fontbox

import (
	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/fontbox/encoding"
	"github.com/shinguakira/pdfbox-go/go/fontbox/util"
)

// FontBoxFont is a font, whatever format it is stored in.
//
// Port of org.apache.fontbox.FontBoxFont.
type FontBoxFont interface {
	// Name returns the PostScript name of the font.
	Name() (string, error)

	// FontBBox returns the box every glyph of the font fits in.
	FontBBox() (*util.BoundingBox, error)

	// FontMatrix returns the transform from glyph space to text space.
	FontMatrix() ([]float32, error)

	// GetPath returns the outline of the named glyph, in glyph space. The
	// caller scales it by the font matrix.
	GetPath(name string) (*geom.Path2D, error)

	// GetWidth returns how far the pen moves after the named glyph, in glyph
	// space.
	GetWidth(name string) (float32, error)

	// HasGlyph reports whether the font has the named glyph.
	HasGlyph(name string) (bool, error)
}

// EncodedFont is a font that carries an encoding of its own.
//
// Port of org.apache.fontbox.EncodedFont.
type EncodedFont interface {
	// Encoding returns the encoding built into the font.
	Encoding() *encoding.Encoding
}
