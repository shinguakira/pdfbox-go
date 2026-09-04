package color

import (
	goimage "image"

	awtimage "github.com/shinguakira/pdfbox-go/go/awt/image"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// PDDeviceRGB is a colour space with red, green and blue components.
//
// Port of org.apache.pdfbox.pdmodel.graphics.color.PDDeviceRGB.
type PDDeviceRGB struct {
	PDDeviceColorSpace
	initialColor *PDColor
}

var _ PDColorSpace = (*PDDeviceRGB)(nil)

// DeviceRGB is the single instance of this colour space.
var DeviceRGB = newDeviceRGB()

func newDeviceRGB() *PDDeviceRGB {
	c := &PDDeviceRGB{PDDeviceColorSpace: PDDeviceColorSpace{name: cos.DeviceRGB.Name()}}
	c.initialColor = NewPDColorOfComponents([]float32{0, 0, 0}, c)
	return c
}

// NumberOfComponents returns 3.
func (c *PDDeviceRGB) NumberOfComponents() int { return 3 }

// DefaultDecode maps the full range of each component to 0 to 1.
func (c *PDDeviceRGB) DefaultDecode(bitsPerComponent int) []float32 {
	return []float32{0, 1, 0, 1, 0, 1}
}

// InitialColor returns black.
func (c *PDDeviceRGB) InitialColor() *PDColor { return c.initialColor }

// ToRGB returns the value unchanged.
func (c *PDDeviceRGB) ToRGB(value []float32) ([]float32, error) { return value, nil }

// ToRGBImage copies the raster into an image.
func (c *PDDeviceRGB) ToRGBImage(raster *awtimage.Raster) (goimage.Image, error) {
	//
	// WARNING: this method is performance sensitive, modify with care!
	//
	// Please read PDFBOX-3854 and PDFBOX-2092 and look at the related commits first.
	// The current code returns TYPE_INT_RGB images which prevents slowness due to threads
	// blocking each other when TYPE_CUSTOM images are used.
	width := raster.Width()
	height := raster.Height()
	image := newRGBImage(width, height)
	pixel := make([]int, raster.NumBands())
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			raster.GetPixel(x, y, pixel)
			setRGB(image, x, y, float32(pixel[0]), float32(pixel[1]), float32(pixel[2]))
		}
	}
	return image, nil
}

// ToRawImage returns nil.
//
// Device RGB is not specified, as its the colors of whatever device you use.
// The user should use the ToRGBImage().
func (c *PDDeviceRGB) ToRawImage(raster *awtimage.Raster) (goimage.Image, error) {
	return nil, nil
}
