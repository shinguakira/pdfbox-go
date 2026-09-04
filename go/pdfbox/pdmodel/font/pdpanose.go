package font

import "fmt"

// PanoseLength is how many bytes a PANOSE entry takes: the family class and the
// ten classification bytes.
const PanoseLength = 12

// PDPanose is the PANOSE entry of a font descriptor.
//
// Port of org.apache.pdfbox.pdmodel.font.PDPanose.
type PDPanose struct {
	bytes []byte
}

// NewPDPanose returns the PANOSE entry the given bytes hold.
func NewPDPanose(bytes []byte) *PDPanose {
	return &PDPanose{bytes: bytes}
}

// FamilyClass returns the two-byte family class.
//
// Java shifts a signed byte left, so a high first byte gives a negative result;
// the port does the same.
func (p *PDPanose) FamilyClass() int {
	return int(int8(p.bytes[0]))<<8 | int(p.bytes[1])
}

// Panose returns the ten classification bytes.
func (p *PDPanose) Panose() *PDPanoseClassification {
	panose := make([]byte, 10)
	copy(panose, p.bytes[2:12])
	return NewPDPanoseClassification(panose)
}

// PanoseClassificationLength is how many bytes a PANOSE classification takes.
const PanoseClassificationLength = 10

// PDPanoseClassification is the ten-byte PANOSE classification of a font.
//
// Port of org.apache.pdfbox.pdmodel.font.PDPanoseClassification.
type PDPanoseClassification struct {
	bytes []byte
}

// NewPDPanoseClassification returns the classification the given bytes hold.
func NewPDPanoseClassification(bytes []byte) *PDPanoseClassification {
	return &PDPanoseClassification{bytes: bytes}
}

// FamilyKind returns which broad kind of design the font is.
//
// Java returns a signed byte from every one of these, so a value above 127
// comes back negative; the port does the same.
func (c *PDPanoseClassification) FamilyKind() int { return int(int8(c.bytes[0])) }

// SerifStyle returns what the serifs look like.
func (c *PDPanoseClassification) SerifStyle() int { return int(int8(c.bytes[1])) }

// Weight returns how heavy the font is.
func (c *PDPanoseClassification) Weight() int { return int(int8(c.bytes[2])) }

// Proportion returns how the glyph widths relate to each other.
func (c *PDPanoseClassification) Proportion() int { return int(int8(c.bytes[3])) }

// Contrast returns how much the stroke thickness varies.
func (c *PDPanoseClassification) Contrast() int { return int(int8(c.bytes[4])) }

// StrokeVariation returns how the stroke thickness varies.
func (c *PDPanoseClassification) StrokeVariation() int { return int(int8(c.bytes[5])) }

// ArmStyle returns what the ends of the strokes look like.
func (c *PDPanoseClassification) ArmStyle() int { return int(int8(c.bytes[6])) }

// Letterform returns the overall shape of the letters.
func (c *PDPanoseClassification) Letterform() int { return int(int8(c.bytes[7])) }

// Midline returns where the midline sits.
func (c *PDPanoseClassification) Midline() int { return int(int8(c.bytes[8])) }

// XHeight returns how tall a lowercase x is, as a classification rather than a
// measurement.
func (c *PDPanoseClassification) XHeight() int { return int(int8(c.bytes[9])) }

// Bytes returns the ten classification bytes.
func (c *PDPanoseClassification) Bytes() []byte { return c.bytes }

// String returns the classification written out.
func (c *PDPanoseClassification) String() string {
	return fmt.Sprintf("{ FamilyKind = %d, SerifStyle = %d, Weight = %d, "+
		"Proportion = %d, Contrast = %d, StrokeVariation = %d, ArmStyle = %d, "+
		"Letterform = %d, Midline = %d, XHeight = %d}",
		c.FamilyKind(), c.SerifStyle(), c.Weight(), c.Proportion(), c.Contrast(),
		c.StrokeVariation(), c.ArmStyle(), c.Letterform(), c.Midline(), c.XHeight())
}
