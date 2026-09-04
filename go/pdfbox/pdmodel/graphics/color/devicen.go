package color

import (
	"fmt"
	goimage "image"
	"strings"

	awtimage "github.com/shinguakira/pdfbox-go/go/awt/image"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common/function"
)

// The array indexes of a /DeviceN colour space.
const (
	deviceNColorantNames = 1
	deviceNAlternateCS   = 2
	deviceNTintTransform = 3
	deviceNAttributes    = 4
)

// PDDeviceN is a colour space with any number of colorants, painted through a
// tint transform into another space.
//
// Port of org.apache.pdfbox.pdmodel.graphics.color.PDDeviceN.
type PDDeviceN struct {
	array *cos.Array

	alternateColorSpace PDColorSpace
	tintTransform       function.PDFunction
	attributes          *PDDeviceNAttributes
	initialColor        *PDColor

	// color conversion cache
	numColorants        int
	colorantToComponent []int
	processColorSpace   PDColorSpace
	spotColorSpaces     []*PDSeparation
}

var _ PDColorSpace = (*PDDeviceN)(nil)

// NewPDDeviceN returns an empty DeviceN colour space.
func NewPDDeviceN() *PDDeviceN {
	c := &PDDeviceN{array: cos.NewArray()}
	c.array.Add(cos.DeviceN)
	// empty placeholder
	c.array.Add(cos.NullObject)
	c.array.Add(cos.NullObject)
	c.array.Add(cos.NullObject)
	return c
}

// NewPDDeviceNOfArray reads a DeviceN colour space out of its array.
func NewPDDeviceNOfArray(deviceN *cos.Array, resources ResourcesLike) (*PDDeviceN, error) {
	c := &PDDeviceN{array: deviceN}

	alternate, err := CreateOfResources(deviceN.GetObject(deviceNAlternateCS), resources)
	if err != nil {
		return nil, err
	}
	c.alternateColorSpace = alternate

	tint, err := function.NewPDFunction(deviceN.GetObject(deviceNTintTransform))
	if err != nil {
		return nil, err
	}
	c.tintTransform = tint

	if deviceN.Size() > deviceNAttributes {
		// Java casts without a check, so a /DeviceN whose fifth entry is not a
		// dictionary throws ClassCastException; the port panics for the same.
		c.attributes = NewPDDeviceNAttributesOfDictionary(
			deviceN.GetObject(deviceNAttributes).(*cos.Dictionary))
	}

	if err := c.initColorConversionCache(resources); err != nil {
		return nil, err
	}

	// set initial color space
	n := c.NumberOfComponents()
	initial := make([]float32, n)
	for i := range initial {
		initial[i] = 1
	}
	c.initialColor = NewPDColorOfComponents(initial, c)
	return c, nil
}

// initColorConversionCache initializes the color conversion cache.
func (c *PDDeviceN) initColorConversionCache(resources ResourcesLike) error {
	// there's nothing to cache for non-attribute spaces
	if c.attributes == nil {
		return nil
	}

	// colorant names
	colorantNames := c.ColorantNames()
	c.numColorants = len(colorantNames)

	// process components
	c.colorantToComponent = make([]int, c.numColorants)
	process := c.attributes.Process()
	if process != nil {
		components := process.Components()
		// map each colorant name to the corresponding process component name (if any)
		for comp := 0; comp < c.numColorants; comp++ {
			c.colorantToComponent[comp] = indexOfString(components, colorantNames[comp])
		}
		// process color space
		space, err := process.ColorSpace()
		if err != nil {
			return err
		}
		c.processColorSpace = space
	} else {
		for comp := 0; comp < c.numColorants; comp++ {
			c.colorantToComponent[comp] = -1
		}
	}

	// spot colorants
	c.spotColorSpaces = make([]*PDSeparation, c.numColorants)

	// spot color spaces
	spotColorants, err := c.attributes.Colorants(resources)
	if err != nil {
		return err
	}

	// map each colorant to the corresponding spot color space
	for comp := 0; comp < c.numColorants; comp++ {
		name := colorantNames[comp]
		spot := spotColorants[name]
		if spot != nil {
			// spot colorant
			c.spotColorSpaces[comp] = spot

			// spot colors may replace process colors with same name
			// providing that the subtype is not NChannel.
			if !c.IsNChannel() {
				c.colorantToComponent[comp] = -1
			}
		} else {
			// process colorant
			c.spotColorSpaces[comp] = nil
		}
	}
	return nil
}

func indexOfString(values []string, want string) int {
	for i, value := range values {
		if value == want {
			return i
		}
	}
	return -1
}

// COSObject returns the array below this colour space.
func (c *PDDeviceN) COSObject() cos.Base { return c.array }

// ToRGBImage converts a raster of colorant values.
func (c *PDDeviceN) ToRGBImage(raster *awtimage.Raster) (goimage.Image, error) {
	if c.attributes != nil {
		return c.toRGBImageWithAttributes(raster)
	}
	return c.toRGBImageWithTintTransform(raster)
}

// WARNING: this method is performance sensitive, modify with care!
func (c *PDDeviceN) toRGBImageWithAttributes(raster *awtimage.Raster) (goimage.Image, error) {
	width := raster.Width()
	height := raster.Height()
	rgbImage := newRGBImage(width, height)

	// white background
	for i := range rgbImage.Pix {
		rgbImage.Pix[i] = 0xFF
	}

	// look up each colorant
	for comp := 0; comp < c.numColorants; comp++ {
		var componentColorSpace PDColorSpace
		isProcessColorant := c.colorantToComponent[comp] >= 0
		switch {
		case isProcessColorant:
			// process color
			componentColorSpace = c.processColorSpace
		case c.spotColorSpaces[comp] == nil:
			// TODO this happens in the Altona Visual test, is there a better workaround?
			// missing spot color, fallback to using tintTransform
			return c.toRGBImageWithTintTransform(raster)
		default:
			// spot color
			componentColorSpace = c.spotColorSpaces[comp]
		}

		numberOfComponents := componentColorSpace.NumberOfComponents()

		// copy single-component to its own raster in the component color space
		componentRaster := awtimage.NewInterleavedRaster(awtimage.TypeByte,
			width, height, numberOfComponents)

		samples := make([]int, max(c.numColorants, raster.NumBands()))
		componentSamples := make([]int, numberOfComponents)
		componentIndex := c.colorantToComponent[comp]
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				raster.GetPixel(x, y, samples)
				if isProcessColorant {
					// process color
					componentSamples[componentIndex] = samples[comp]
				} else {
					// spot color
					componentSamples[0] = samples[comp]
				}
				componentRaster.SetPixel(x, y, componentSamples)
			}
		}

		// convert single-component raster to RGB
		rgbComponentImage, err := componentColorSpace.ToRGBImage(componentRaster)
		if err != nil {
			return nil, err
		}

		// combine the RGB component with the RGB composite raster
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				rgbChannel := pixelOfImage(rgbComponentImage, x, y)
				rgbComposite := pixelAt(rgbImage, x, y, make([]int, 3))
				// multiply (blend mode)
				rgbChannel[0] = rgbChannel[0] * rgbComposite[0] >> 8
				rgbChannel[1] = rgbChannel[1] * rgbComposite[1] >> 8
				rgbChannel[2] = rgbChannel[2] * rgbComposite[2] >> 8
				setRGB(rgbImage, x, y, float32(rgbChannel[0]),
					float32(rgbChannel[1]), float32(rgbChannel[2]))
			}
		}
	}
	return rgbImage, nil
}

// WARNING: this method is performance sensitive, modify with care!
func (c *PDDeviceN) toRGBImageWithTintTransform(raster *awtimage.Raster) (goimage.Image, error) {
	// cache color mappings
	map1 := map[string][]int{}
	var keyBuilder strings.Builder

	width := raster.Width()
	height := raster.Height()

	// use the tint transform to convert the sample into
	// the alternate color space (this is usually 1:many)
	rgbImage := newRGBImage(width, height)

	numSrcComponents := len(c.ColorantNames())
	src := make([]float32, numSrcComponents)
	pixel := make([]int, max(numSrcComponents, raster.NumBands()))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			raster.GetPixel(x, y, pixel)
			for s := 0; s < numSrcComponents; s++ {
				src[s] = float32(pixel[s])
			}

			// use a string representation as key
			fmt.Fprintf(&keyBuilder, "%v", src[0])
			for s := 1; s < numSrcComponents; s++ {
				fmt.Fprintf(&keyBuilder, "#%v", src[s])
			}
			key := keyBuilder.String()
			keyBuilder.Reset()

			if pxl, ok := map1[key]; ok {
				setRGB(rgbImage, x, y, float32(pxl[0]), float32(pxl[1]), float32(pxl[2]))
				continue
			}

			// scale to 0..1
			for s := 0; s < numSrcComponents; s++ {
				src[s] = src[s] / 255
			}

			// convert to alternate color space via tint transform
			result, err := c.tintTransform.Eval(src)
			if err != nil {
				return nil, err
			}
			// convert from alternate color space to RGB
			rgbFloat, err := c.alternateColorSpace.ToRGB(result)
			if err != nil {
				return nil, err
			}

			// scale to 0..255
			rgb := []int{int(rgbFloat[0] * 255), int(rgbFloat[1] * 255), int(rgbFloat[2] * 255)}
			map1[key] = rgb
			setRGB(rgbImage, x, y, float32(rgb[0]), float32(rgb[1]), float32(rgb[2]))
		}
	}
	return rgbImage, nil
}

// ToRGB converts one colour value.
func (c *PDDeviceN) ToRGB(value []float32) ([]float32, error) {
	if c.attributes != nil {
		return c.toRGBWithAttributes(value)
	}
	return c.toRGBWithTintTransform(value)
}

func (c *PDDeviceN) toRGBWithAttributes(value []float32) ([]float32, error) {
	rgbValue := []float32{1, 1, 1}

	// look up each colorant
	for comp := 0; comp < c.numColorants; comp++ {
		var componentColorSpace PDColorSpace
		isProcessColorant := c.colorantToComponent[comp] >= 0
		switch {
		case isProcessColorant:
			// process color
			componentColorSpace = c.processColorSpace
		case c.spotColorSpaces[comp] == nil:
			// TODO this happens in the Altona Visual test, is there a better workaround?
			// missing spot color, fallback to using tintTransform
			return c.toRGBWithTintTransform(value)
		default:
			// spot color
			componentColorSpace = c.spotColorSpaces[comp]
		}

		// get the single component
		componentSamples := make([]float32, componentColorSpace.NumberOfComponents())
		if isProcessColorant {
			// process color
			componentIndex := c.colorantToComponent[comp]
			componentSamples[componentIndex] = value[comp]
		} else {
			// spot color
			componentSamples[0] = value[comp]
		}

		// convert single component to RGB
		rgbComponent, err := componentColorSpace.ToRGB(componentSamples)
		if err != nil {
			return nil, err
		}

		// combine the RGB component value with the RGB composite value
		// multiply (blend mode)
		rgbValue[0] *= rgbComponent[0]
		rgbValue[1] *= rgbComponent[1]
		rgbValue[2] *= rgbComponent[2]
	}

	return rgbValue, nil
}

func (c *PDDeviceN) toRGBWithTintTransform(value []float32) ([]float32, error) {
	// use the tint transform to convert the sample into
	// the alternate color space (this is usually 1:many)
	altValue, err := c.tintTransform.Eval(value)
	if err != nil {
		return nil, err
	}
	// convert the alternate color space to RGB
	return c.alternateColorSpace.ToRGB(altValue)
}

// ToRawImage returns nil: we don't know how to convert that.
func (c *PDDeviceN) ToRawImage(raster *awtimage.Raster) (goimage.Image, error) {
	return nil, nil
}

// IsNChannel reports whether the attributes name the NChannel subtype.
func (c *PDDeviceN) IsNChannel() bool {
	return c.attributes != nil && c.attributes.IsNChannel()
}

// Name returns "DeviceN".
func (c *PDDeviceN) Name() string { return cos.DeviceN.Name() }

// NumberOfComponents returns how many colorants the space has.
func (c *PDDeviceN) NumberOfComponents() int { return len(c.ColorantNames()) }

// DefaultDecode maps the full range of each colorant to 0 to 1.
func (c *PDDeviceN) DefaultDecode(bitsPerComponent int) []float32 {
	n := c.NumberOfComponents()
	decode := make([]float32, n*2)
	for i := 0; i < n; i++ {
		decode[i*2+1] = 1
	}
	return decode
}

// InitialColor returns every colorant at full tint.
func (c *PDDeviceN) InitialColor() *PDColor { return c.initialColor }

// ColorantNames returns the names of the colorants.
func (c *PDDeviceN) ColorantNames() []string {
	if array, ok := c.array.GetObject(deviceNColorantNames).(*cos.Array); ok {
		names := make([]string, 0, array.Size())
		for i := 0; i < array.Size(); i++ {
			if name, ok := array.GetObject(i).(*cos.Name); ok {
				names = append(names, name.Name())
			}
		}
		return names
	}
	return nil
}

// Attributes returns the /DeviceN attributes, or nil where there are none.
func (c *PDDeviceN) Attributes() *PDDeviceNAttributes { return c.attributes }

// SetColorantNames sets the names of the colorants.
func (c *PDDeviceN) SetColorantNames(names []string) {
	c.array.Set(deviceNColorantNames, cos.ArrayOfNames(names))
}

// SetAttributes sets the /DeviceN attributes.
func (c *PDDeviceN) SetAttributes(attributes *PDDeviceNAttributes) {
	c.attributes = attributes
	if attributes == nil {
		c.array.RemoveAt(deviceNAttributes)
		return
	}
	// make sure array is large enough
	for c.array.Size() <= deviceNAttributes {
		c.array.Add(cos.NullObject)
	}
	c.array.Set(deviceNAttributes, attributes.COSDictionary())
}

// AlternateColorSpace returns the colour space the tint transform paints into.
func (c *PDDeviceN) AlternateColorSpace() (PDColorSpace, error) {
	if c.alternateColorSpace == nil {
		alternate, err := Create(c.array.GetObject(deviceNAlternateCS))
		if err != nil {
			return nil, err
		}
		c.alternateColorSpace = alternate
	}
	return c.alternateColorSpace, nil
}

// SetAlternateColorSpace sets the colour space the tint transform paints into.
func (c *PDDeviceN) SetAlternateColorSpace(cs PDColorSpace) {
	c.alternateColorSpace = cs
	var space cos.Base
	if cs != nil {
		space = cs.COSObject()
	}
	c.array.Set(deviceNAlternateCS, space)
}

// TintTransform returns the tint transform.
func (c *PDDeviceN) TintTransform() (function.PDFunction, error) {
	if c.tintTransform == nil {
		tint, err := function.NewPDFunction(c.array.GetObject(deviceNTintTransform))
		if err != nil {
			return nil, err
		}
		c.tintTransform = tint
	}
	return c.tintTransform, nil
}

// SetTintTransform sets the tint transform.
func (c *PDDeviceN) SetTintTransform(tint function.PDFunction) {
	c.tintTransform = tint
	c.array.Set(deviceNTintTransform, tint.COSObject())
}

// String is Java's toString.
func (c *PDDeviceN) String() string {
	var sb strings.Builder
	sb.WriteString(c.Name())
	sb.WriteByte('{')
	for _, col := range c.ColorantNames() {
		sb.WriteByte('"')
		sb.WriteString(col)
		sb.WriteString("\" ")
	}
	sb.WriteString(c.alternateColorSpace.Name())
	sb.WriteByte(' ')
	fmt.Fprintf(&sb, "%v", c.tintTransform)
	sb.WriteByte(' ')
	if c.attributes != nil {
		fmt.Fprintf(&sb, "%v", c.attributes)
	}
	sb.WriteByte('}')
	return sb.String()
}
