package taggedpdf

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
)

// PDFourColours is the four colours of a border or a text decoration: before,
// after, start and end.
//
// Port of PDFourColours.
type PDFourColours struct {
	array *cos.Array
}

var _ common.COSObjectable = (*PDFourColours)(nil)

// NewPDFourColours builds four unset colours.
func NewPDFourColours() *PDFourColours {
	return &PDFourColours{array: cos.NewArrayOf([]cos.Base{
		cos.NullObject,
		cos.NullObject,
		cos.NullObject,
		cos.NullObject,
	})}
}

// NewPDFourColoursOfArray builds them over the given array, padding it out.
//
// JAVA BUG: the padding starts one entry early, so an array shorter than four
// comes out five long. See migration/JAVA-BUGS.md entry 41. The port keeps it.
func NewPDFourColoursOfArray(array *cos.Array) *PDFourColours {
	c := &PDFourColours{array: array}
	// ensure that array has 4 items
	if c.array.Size() < 4 {
		for i := c.array.Size() - 1; i < 4; i++ {
			c.array.Add(cos.NullObject)
		}
	}
	return c
}

// BeforeColour returns the colour of the before edge, or nil.
func (c *PDFourColours) BeforeColour() *color.PDGamma {
	return c.colourByIndex(0)
}

// SetBeforeColour sets the colour of the before edge.
func (c *PDFourColours) SetBeforeColour(colour *color.PDGamma) {
	c.setColourByIndex(0, colour)
}

// AfterColour returns the colour of the after edge, or nil.
func (c *PDFourColours) AfterColour() *color.PDGamma {
	return c.colourByIndex(1)
}

// SetAfterColour sets the colour of the after edge.
func (c *PDFourColours) SetAfterColour(colour *color.PDGamma) {
	c.setColourByIndex(1, colour)
}

// StartColour returns the colour of the start edge, or nil.
func (c *PDFourColours) StartColour() *color.PDGamma {
	return c.colourByIndex(2)
}

// SetStartColour sets the colour of the start edge.
func (c *PDFourColours) SetStartColour(colour *color.PDGamma) {
	c.setColourByIndex(2, colour)
}

// EndColour returns the colour of the end edge, or nil.
func (c *PDFourColours) EndColour() *color.PDGamma {
	return c.colourByIndex(3)
}

// SetEndColour sets the colour of the end edge.
func (c *PDFourColours) SetEndColour(colour *color.PDGamma) {
	c.setColourByIndex(3, colour)
}

// COSObject returns the array below these colours.
func (c *PDFourColours) COSObject() cos.Base { return c.array }

// colourByIndex returns one colour, or nil where the entry is not an array.
func (c *PDFourColours) colourByIndex(index int) *color.PDGamma {
	if item, isArray := c.array.GetObject(index).(*cos.Array); isArray {
		return color.NewPDGammaOfArray(item)
	}
	return nil
}

// setColourByIndex sets one colour, writing null where it is nil.
func (c *PDFourColours) setColourByIndex(index int, colour *color.PDGamma) {
	var base cos.Base = cos.NullObject
	if colour != nil {
		base = colour.COSArray()
	}
	c.array.Set(index, base)
}
