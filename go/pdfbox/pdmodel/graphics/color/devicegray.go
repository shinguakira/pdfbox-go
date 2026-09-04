package color

import (
	goimage "image"

	awtimage "github.com/shinguakira/pdfbox-go/go/awt/image"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

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

// ToRGBImage spreads the one component of each sample across all three
// channels.
//
// Java's PDDeviceGray builds a TYPE_BYTE_GRAY BufferedImage and sets the
// raster into it, which leaves the caller to convert; the port writes the RGB
// image directly, because Go's image.Gray would have to be converted by every
// caller anyway and the spread is the same three assignments.
func (c *PDDeviceGray) ToRGBImage(raster *awtimage.Raster) (goimage.Image, error) {
	width := raster.Width()
	height := raster.Height()
	image := newRGBImage(width, height)
	pixel := make([]int, raster.NumBands())
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			raster.GetPixel(x, y, pixel)
			gray := float32(pixel[0])
			setRGB(image, x, y, gray, gray, gray)
		}
	}
	return image, nil
}

// ToRawImage returns the raster as a grey image, which is the one case Java's
// PDDeviceGray can answer: its raw image is a TYPE_BYTE_GRAY BufferedImage.
func (c *PDDeviceGray) ToRawImage(raster *awtimage.Raster) (goimage.Image, error) {
	if raster.DataType() != awtimage.TypeByte || raster.NumBands() != 1 {
		return nil, nil
	}
	width := raster.Width()
	height := raster.Height()
	image := goimage.NewGray(goimage.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			image.Pix[y*image.Stride+x] = byte(raster.Samples()[y*width+x])
		}
	}
	return image, nil
}
