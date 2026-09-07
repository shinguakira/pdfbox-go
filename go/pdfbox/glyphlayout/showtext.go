package glyphlayout

// The two methods AbstractGlyphLayoutProcessor calls once per run.
//
// These two *are* close ports: `GlyphLayoutProcessorAwt.getStringWidthUni` and
// `showTextUni` read a GlyphVector and write a content stream, and everything
// they do after the shaping is arithmetic this port can follow line for line.
// What differs is where the GlyphVector comes from -- see processor.go.

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
)

// delta is the distance below which an adjustment is not worth writing.
//
// Java's `final float delta = 1e-5f`.
const delta = 1e-5

// stringWidthUni computes the width of a run that goes one way.
//
// Port of getStringWidthUni, which asks the GlyphVector for its logical bounds.
// The width of a laid-out run is the sum of its advances, which is what those
// bounds are for a horizontal run.
func (p *Processor) stringWidthUni(f *font.PDType0Font, fontSize float32, text string,
	bidiLevel int) (float32, error) {
	glyphs, err := p.layout(f, text, bidiLevel)
	if err != nil {
		return 0, err
	}
	total := 0
	for _, glyph := range glyphs {
		total += glyph.advance
	}
	unitsPerEm, err := p.unitsPerEm(f)
	if err != nil {
		return 0, err
	}
	return float32(total) * fontSize / float32(unitsPerEm), nil
}

// showTextUni writes a run that goes one way, with glyph positioning.
//
// Port of showTextUni, line for line. Java walks the GlyphVector reading two
// different numbers off it:
//
//   - `getGlyphPosition(i)`, where the layout actually put the glyph, which
//     carries every adjustment the shaping made; and
//   - `getGlyphMetrics(i-1).getAdvanceX()`, the *unadjusted* advance the font
//     declares for the previous glyph.
//
// `dx` is the difference between the two -- how far the glyph sits from where
// the pen would have been had nothing been adjusted -- and that difference is
// what goes into the TJ array. A kern reaches the stream through the advance,
// not through the placement, so the port has to keep both numbers as well: it
// walks a pen across the adjusted advances and compares against the natural
// ones, which is the same subtraction with the same two inputs.
func (p *Processor) showTextUni(contentStream pdmodel.ContentStreamForGlyphLayout,
	f *font.PDType0Font, fontSize float32, text string, bidiLevel int) error {
	glyphs, err := p.layout(f, text, bidiLevel)
	if err != nil {
		return err
	}
	unitsPerEm, err := p.unitsPerEm(f)
	if err != nil {
		return err
	}

	// Java's factorX is 1000/fontSize, because a TJ adjustment is in
	// thousandths of the current font size and Java's offsets are in points at
	// that size. This port's offsets are in font design units, so the scale is
	// 1000/unitsPerEm and does not depend on the size at all.
	factorX := float32(fontScale) / float32(unitsPerEm)
	// The same for a text rise, which is in points rather than thousandths.
	factorY := fontSize / float32(unitsPerEm)

	ga := &pdmodel.GlyphsAndPositions{}

	// pen is where the run has got to, which is Java's accumulated glyph
	// position; lastX is the position of the glyph before this one.
	pen, lastX := 0, 0

	for i, glyph := range glyphs {
		// Where the layout put this glyph: the pen, moved sideways by whatever
		// placement the positioning gave it.
		px := pen + glyph.xOffset
		// The advance the font declares for the glyph before this one, before
		// positioning touched it -- Java's getGlyphMetrics(i-1).getAdvanceX().
		ax := 0
		if i != 0 {
			ax = glyphs[i-1].natural
		}
		dx := float32(px-lastX-ax) * factorX
		// Java negates the y position because AWT's y grows downwards and a
		// text rise grows upwards. A GPOS y placement already grows upwards,
		// so the port does not negate it.
		py := float32(glyph.yOffset) * factorY

		if absFloat(py) >= delta {
			if !ga.IsEmpty() {
				if err := contentStream.ShowGlyphsWithPositioning(ga); err != nil {
					return err
				}
				ga.Clear()
			}
			if err := contentStream.SetTextRise(py); err != nil {
				return err
			}
		}
		if absFloat(dx) >= delta {
			ga.AddPosition(-dx)
		}
		ga.AddGlyph(glyph.code)
		if absFloat(py) >= delta {
			if err := contentStream.ShowGlyphsWithPositioning(ga); err != nil {
				return err
			}
			ga.Clear()
			if err := contentStream.SetTextRise(0); err != nil {
				return err
			}
		}
		lastX = px
		pen += glyph.advance
	}

	// Adjust the end position, so the pen finishes where the layout says the
	// run ends rather than where the last glyph's own advance would leave it.
	// Java reads getGlyphPosition(numGlyphs), which is that end position.
	if len(glyphs) != 0 {
		ax := glyphs[len(glyphs)-1].natural
		dx := float32(pen-lastX-ax) * factorX
		if absFloat(dx) >= delta {
			ga.AddPosition(-dx)
		}
	}
	if err := contentStream.ShowGlyphsWithPositioning(ga); err != nil {
		return err
	}
	ga.Clear()
	return nil
}

// unitsPerEm answers how many design units the font's em is, which every
// measurement here is in.
func (p *Processor) unitsPerEm(f *font.PDType0Font) (int, error) {
	header, err := f.TrueTypeFont().Header()
	if err != nil {
		return 0, err
	}
	return header.UnitsPerEm(), nil
}

// absFloat is Math.abs(float).
func absFloat(value float32) float32 {
	if value < 0 {
		return -value
	}
	return value
}
