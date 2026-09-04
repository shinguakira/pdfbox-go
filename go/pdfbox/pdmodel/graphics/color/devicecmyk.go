package color

import (
	goimage "image"

	awtimage "github.com/shinguakira/pdfbox-go/go/awt/image"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// PDDeviceCMYK is a colour space with cyan, magenta, yellow and black
// components.
//
// Port of org.apache.pdfbox.pdmodel.graphics.color.PDDeviceCMYK.
//
// **This one is not faithful, and cannot be.** Java converts through an ICC
// profile it ships as a resource -- CGATS001Compat-v2-micro, an open stand-in
// for the "U.S. Web Coated (SWOP) v2" profile Acrobat uses -- handed to
// java.awt.color.ICC_ColorSpace and from there to LittleCMS. Go has no ICC
// engine, in its standard library or anywhere PDFBox has code for, and PDFBox
// has none of its own to port: the whole conversion is three lines that call
// out to the platform.
//
// So the port converts naively, the way software without a profile does:
// R = (1-C)(1-K), and the same for the other two. That is a real difference in
// the colours a CMYK image comes out as, not a rounding one, and it is
// recorded in migration/STATUS.md as the largest gap in the slice. Everything
// else about the colour space -- the component count, the decode array, the
// initial colour, the name -- is the Java's.
type PDDeviceCMYK struct {
	PDDeviceColorSpace
	initialColor *PDColor
}

var _ PDColorSpace = (*PDDeviceCMYK)(nil)

// DeviceCMYK is the single instance of this colour space.
var DeviceCMYK = newDeviceCMYK()

func newDeviceCMYK() *PDDeviceCMYK {
	c := &PDDeviceCMYK{PDDeviceColorSpace: PDDeviceColorSpace{name: cos.DeviceCMYK.Name()}}
	c.initialColor = NewPDColorOfComponents([]float32{0, 0, 0, 1}, c)
	return c
}

// NumberOfComponents returns 4.
func (c *PDDeviceCMYK) NumberOfComponents() int { return 4 }

// DefaultDecode maps the full range of each component to 0 to 1.
func (c *PDDeviceCMYK) DefaultDecode(bitsPerComponent int) []float32 {
	return []float32{0, 1, 0, 1, 0, 1, 0, 1}
}

// InitialColor returns black, which in CMYK is no ink but the black one.
func (c *PDDeviceCMYK) InitialColor() *PDColor { return c.initialColor }

// ToRGB converts one colour value.
//
// See the note on the type: this is the naive conversion, not the profiled one
// Java makes.
func (c *PDDeviceCMYK) ToRGB(value []float32) ([]float32, error) {
	return cmykToRGB(value[0], value[1], value[2], value[3]), nil
}

// cmykToRGB is the conversion the whole colour space rests on, in one place so
// that an ICC transform could replace it in one place.
func cmykToRGB(cyan, magenta, yellow, black float32) []float32 {
	k := 1 - black
	return []float32{(1 - cyan) * k, (1 - magenta) * k, (1 - yellow) * k}
}

// ToRGBImage converts a raster of CMYK samples.
func (c *PDDeviceCMYK) ToRGBImage(raster *awtimage.Raster) (goimage.Image, error) {
	width := raster.Width()
	height := raster.Height()
	image := newRGBImage(width, height)
	pixel := make([]int, raster.NumBands())
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			raster.GetPixel(x, y, pixel)
			rgb := cmykToRGB(float32(pixel[0])/255, float32(pixel[1])/255,
				float32(pixel[2])/255, float32(pixel[3])/255)
			setRGB(image, x, y, rgb[0]*255, rgb[1]*255, rgb[2]*255)
		}
	}
	return image, nil
}

// ToRawImage returns nil.
//
// Device CMYK is not specified, as its the colors of whatever device you use.
// The user should fallback to the RGB image.
func (c *PDDeviceCMYK) ToRawImage(raster *awtimage.Raster) (goimage.Image, error) {
	return nil, nil
}
