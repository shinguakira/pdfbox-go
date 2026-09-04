package color

import (
	"fmt"
	goimage "image"
	"math"

	awtimage "github.com/shinguakira/pdfbox-go/go/awt/image"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common/function"
)

// The array indexes of a /Separation colour space.
const (
	separationColorantNames = 1
	separationAlternateCS   = 2
	separationTintTransform = 3
)

// PDSeparation is a colour space with one colorant, painted through a tint
// transform into another space.
//
// Port of org.apache.pdfbox.pdmodel.graphics.color.PDSeparation.
type PDSeparation struct {
	array        *cos.Array
	initialColor *PDColor

	alternateColorSpace PDColorSpace
	tintTransform       function.PDFunction
	toRGBMap            map[int][]float32
}

var _ PDColorSpace = (*PDSeparation)(nil)

// NewPDSeparation returns an empty separation colour space.
func NewPDSeparation() *PDSeparation {
	c := &PDSeparation{array: cos.NewArray()}
	c.initialColor = NewPDColorOfComponents([]float32{1}, c)
	c.array.Add(cos.Separation)
	c.array.Add(cos.GetPDFName(""))
	// add some placeholder
	c.array.Add(cos.NullObject)
	c.array.Add(cos.NullObject)
	return c
}

// NewPDSeparationOfArray reads a separation colour space out of its array.
func NewPDSeparationOfArray(separation *cos.Array,
	resources ResourcesLike) (*PDSeparation, error) {
	c := &PDSeparation{array: separation}
	c.initialColor = NewPDColorOfComponents([]float32{1}, c)

	alternate, err := CreateOfResources(separation.GetObject(separationAlternateCS), resources)
	if err != nil {
		return nil, err
	}
	c.alternateColorSpace = alternate

	tint, err := function.NewPDFunction(separation.GetObject(separationTintTransform))
	if err != nil {
		return nil, err
	}
	c.tintTransform = tint

	numberOfOutputParameters := tint.NumberOfOutputParameters()
	if numberOfOutputParameters > 0 &&
		numberOfOutputParameters < alternate.NumberOfComponents() {
		return nil, fmt.Errorf("The tint transform function has less output parameters (%d) "+
			"than the alternate colorspace %v (%d)",
			numberOfOutputParameters, alternate, alternate.NumberOfComponents())
	}
	return c, nil
}

// COSObject returns the array below this colour space.
func (c *PDSeparation) COSObject() cos.Base { return c.array }

// Name returns "Separation".
func (c *PDSeparation) Name() string { return cos.Separation.Name() }

// NumberOfComponents returns 1: the tint.
func (c *PDSeparation) NumberOfComponents() int { return 1 }

// DefaultDecode maps the full range to 0 to 1.
func (c *PDSeparation) DefaultDecode(bitsPerComponent int) []float32 { return []float32{0, 1} }

// InitialColor returns full tint.
func (c *PDSeparation) InitialColor() *PDColor { return c.initialColor }

// ToRGB converts one tint value through the tint transform.
func (c *PDSeparation) ToRGB(value []float32) ([]float32, error) {
	if c.toRGBMap == nil {
		c.toRGBMap = map[int][]float32{}
	}
	key := int(value[0] * 255)
	if retval, ok := c.toRGBMap[key]; ok {
		return retval, nil
	}
	altColor, err := c.tintTransform.Eval(value)
	if err != nil {
		return nil, err
	}
	retval, err := c.alternateColorSpace.ToRGB(altColor)
	if err != nil {
		return nil, err
	}
	c.toRGBMap[key] = retval
	return retval, nil
}

// ToRGBImage converts a raster of tint values.
//
// WARNING: this method is performance sensitive, modify with care!
func (c *PDSeparation) ToRGBImage(raster *awtimage.Raster) (goimage.Image, error) {
	if _, isLab := c.alternateColorSpace.(*PDLab); isLab {
		// PDFBOX-3622 - regular converter fails for Lab colorspaces
		return c.toRGBImage2(raster)
	}
	if icc, isICC := c.alternateColorSpace.(*PDICCBased); isICC {
		// PDFBOX-5778 - same problem if Lab-based ICC colorspace
		if _, isLab := icc.AlternateColorSpace().(*PDLab); isLab {
			return c.toRGBImage2(raster)
		}
	}

	numAltComponents := c.alternateColorSpace.NumberOfComponents()
	width := raster.Width()
	height := raster.Height()

	// use the tint transform to convert the sample into
	// the alternate color space (this is usually 1:many)
	altRaster := awtimage.NewInterleavedRaster(awtimage.TypeByte, width, height, numAltComponents)

	samples := make([]float32, 1)
	pixel := make([]int, raster.NumBands())
	calculatedValues := map[int32][]int{}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			raster.GetPixel(x, y, pixel)
			samples[0] = float32(pixel[0])
			hash := int32(math.Float32bits(samples[0]))
			alt, ok := calculatedValues[hash]
			if !ok {
				alt = make([]int, numAltComponents)
				if err := c.tintTransformInto(samples, alt); err != nil {
					return nil, err
				}
				calculatedValues[hash] = alt
			}
			altRaster.SetPixel(x, y, alt)
		}
	}

	// convert the alternate color space to RGB
	return c.alternateColorSpace.ToRGBImage(altRaster)
}

// toRGBImage2 is Java's converter that works without using the super
// implementation of toRGBImage().
func (c *PDSeparation) toRGBImage2(raster *awtimage.Raster) (goimage.Image, error) {
	width := raster.Width()
	height := raster.Height()
	rgbImage := newRGBImage(width, height)

	samples := make([]float32, 1)
	pixel := make([]int, raster.NumBands())
	calculatedValues := map[int32][]int{}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			raster.GetPixel(x, y, pixel)
			samples[0] = float32(pixel[0])
			hash := int32(math.Float32bits(samples[0]))
			rgb, ok := calculatedValues[hash]
			if !ok {
				scaled := []float32{samples[0] / 255}
				altColor, err := c.tintTransform.Eval(scaled)
				if err != nil {
					return nil, err
				}
				fltab, err := c.alternateColorSpace.ToRGB(altColor)
				if err != nil {
					return nil, err
				}
				rgb = []int{int(fltab[0] * 255), int(fltab[1] * 255), int(fltab[2] * 255)}
				calculatedValues[hash] = rgb
			}
			setRGB(rgbImage, x, y, float32(rgb[0]), float32(rgb[1]), float32(rgb[2]))
		}
	}
	return rgbImage, nil
}

// tintTransformInto is Java's protected tintTransform, which PDDeviceN would
// override; nothing in PDFBox does.
func (c *PDSeparation) tintTransformInto(samples []float32, alt []int) error {
	scaled := []float32{samples[0] / 255} // 0..1
	result, err := c.tintTransform.Eval(scaled)
	if err != nil {
		return err
	}
	for s := 0; s < len(alt); s++ {
		// scale to 0..255
		alt[s] = int(result[s] * 255)
	}
	return nil
}

// ToRawImage returns the raster as a grey image.
//
// Java wraps it in a CS_GRAY colour model, which is one component like the
// separation itself; the port returns the grey image PDDeviceGray builds, which
// is the same samples.
func (c *PDSeparation) ToRawImage(raster *awtimage.Raster) (goimage.Image, error) {
	return DeviceGray.ToRawImage(raster)
}

// AlternateColorSpace returns the colour space the tint transform paints into.
func (c *PDSeparation) AlternateColorSpace() PDColorSpace { return c.alternateColorSpace }

// ColorantName returns the name of the colorant, or the empty string where the
// entry is not a name.
//
// Java returns null there; the port returns false in the second result.
func (c *PDSeparation) ColorantName() (string, bool) {
	if name, ok := c.array.GetObject(separationColorantNames).(*cos.Name); ok {
		return name.Name(), true
	}
	return "", false
}

// SetColorantName sets the name of the colorant.
func (c *PDSeparation) SetColorantName(name string) {
	c.array.Set(1, cos.GetPDFName(name))
}

// SetAlternateColorSpace sets the colour space the tint transform paints into.
func (c *PDSeparation) SetAlternateColorSpace(colorSpace PDColorSpace) {
	c.alternateColorSpace = colorSpace
	var space cos.Base
	if colorSpace != nil {
		space = colorSpace.COSObject()
	}
	c.array.Set(separationAlternateCS, space)
}

// SetTintTransform sets the tint transform.
func (c *PDSeparation) SetTintTransform(tint function.PDFunction) {
	c.tintTransform = tint
	c.array.Set(separationTintTransform, tint.COSObject())
}

// String is Java's toString.
func (c *PDSeparation) String() string {
	name, _ := c.ColorantName()
	return fmt.Sprintf("%s{%q %s %v}", c.Name(), name,
		c.alternateColorSpace.Name(), c.tintTransform)
}
