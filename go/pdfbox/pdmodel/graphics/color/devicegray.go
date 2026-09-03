package color

import "github.com/shinguakira/pdfbox-go/go/pdfbox/cos"

// PDDeviceGray is a colour space with black, white, and intermediate shades of
// gray.
//
// Port of org.apache.pdfbox.pdmodel.graphics.color.PDDeviceGray.
type PDDeviceGray struct {
	PDDeviceColorSpace
	initialColor *PDColor
}

var _ PDColorSpace = (*PDDeviceGray)(nil)

// DeviceGray is the single instance of this colour space.
var DeviceGray = newDeviceGray()

func newDeviceGray() *PDDeviceGray {
	c := &PDDeviceGray{PDDeviceColorSpace: PDDeviceColorSpace{name: cos.DeviceGray.Name()}}
	c.initialColor = NewPDColorOfComponents([]float32{0}, c)
	return c
}

// NumberOfComponents returns 1: a shade of gray is one value.
func (c *PDDeviceGray) NumberOfComponents() int { return 1 }

// DefaultDecode returns the decode array mapping the full range to black and
// white.
func (c *PDDeviceGray) DefaultDecode(bitsPerComponent int) []float32 {
	return []float32{0, 1}
}

// InitialColor returns black, which is what a content stream starts with.
func (c *PDDeviceGray) InitialColor() *PDColor { return c.initialColor }

// ToRGB spreads the one component across all three channels.
func (c *PDDeviceGray) ToRGB(value []float32) ([]float32, error) {
	return []float32{value[0], value[0], value[0]}, nil
}
