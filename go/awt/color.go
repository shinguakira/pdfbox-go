// Package awt is the part of java.awt PDFBox names outside of geometry and
// rasters.
//
// Port of java.awt. Only Color is here; the rest of the toolkit is a windowing
// system the port has no use for.
package awt

// Color is a colour in the default sRGB space.
//
// Port of java.awt.Color, cut down to what PDFBox uses of it: a colour built
// from a packed integer or from three components, and read back as three
// components. Java holds both an 8-bit packed value and, when the colour was
// built from floats, the floats themselves; getRGBColorComponents answers the
// floats where they are there and the packed value scaled otherwise, so the
// port holds the floats and derives the packed value.
type Color struct {
	r, g, b float32
}

// NewColorOfRGB returns the colour the given packed integer holds, which is
// 0xRRGGBB with the alpha ignored.
//
// Port of the Color(int) constructor, whose hasalpha is false.
func NewColorOfRGB(rgb int) Color {
	return Color{
		r: float32((rgb>>16)&0xFF) / 255,
		g: float32((rgb>>8)&0xFF) / 255,
		b: float32(rgb&0xFF) / 255,
	}
}

// NewColor returns the colour with the given components, each of zero to one.
//
// Port of the Color(float, float, float) constructor. Java throws
// IllegalArgumentException outside that range, which is unchecked, so the port
// panics.
func NewColor(r, g, b float32) Color {
	if outsideUnitInterval(r) || outsideUnitInterval(g) || outsideUnitInterval(b) {
		panic("Color parameter outside of expected range")
	}
	return Color{r: r, g: g, b: b}
}

// outsideUnitInterval reports whether the component is outside zero to one, or
// is not a number, which is the check the Java constructor makes.
func outsideUnitInterval(value float32) bool {
	return !(value >= 0 && value <= 1)
}

// RGBColorComponents returns the three components of the colour, each of zero
// to one.
//
// Port of getRGBColorComponents(float[]), which fills the array it is given, or
// a new one for null. Go returns the three values.
func (c Color) RGBColorComponents() (r, g, b float32) { return c.r, c.g, c.b }

// Red returns the red component, of zero to 255.
func (c Color) Red() int { return int(c.r*255 + 0.5) }

// Green returns the green component, of zero to 255.
func (c Color) Green() int { return int(c.g*255 + 0.5) }

// Blue returns the blue component, of zero to 255.
func (c Color) Blue() int { return int(c.b*255 + 0.5) }

// RGB returns the colour packed as 0xFFRRGGBB, which is what getRGB answers.
func (c Color) RGB() int {
	return 0xFF000000 | c.Red()<<16 | c.Green()<<8 | c.Blue()
}
