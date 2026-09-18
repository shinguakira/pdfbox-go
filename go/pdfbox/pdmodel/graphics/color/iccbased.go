package color

import (
	"errors"
	"fmt"
	goimage "image"
	"log/slog"

	awtimage "github.com/shinguakira/pdfbox-go/go/awt/image"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// PDICCBased is a colour space based on a supplied ICC profile.
//
// Port of org.apache.pdfbox.pdmodel.graphics.color.PDICCBased.
//
// **The profile is not read.** Java hands the stream to
// java.awt.color.ICC_Profile and converts through LittleCMS; Go has no ICC
// engine and PDFBox has no ICC code of its own to port. So this port always
// takes the path Java takes when the profile will not load, or when the system
// property org.apache.pdfbox.rendering.UseAlternateInsteadOfICCColorSpace is
// set: it uses the /Alternate colour space, or the device space the component
// count implies. That is Java's own fallback, taken deliberately rather than on
// an error, and it is recorded in migration/STATUS.md.
type PDICCBased struct {
	array *cos.Array

	stream              *common.PDStream
	alternateColorSpace PDColorSpace
	initialColor        *PDColor
	isRGBProfile        bool
}

var _ PDColorSpace = (*PDICCBased)(nil)

// NewPDICCBasedOfStream creates a new ICC colour space with an empty stream to
// hold the profile.
//
// Port of PDICCBased(PDDocument), which builds the stream from the document's
// stream cache; the port takes the stream, because pdmodel is above this
// package and PDStream is not.
func NewPDICCBasedOfStream(stream *cos.Stream) *PDICCBased {
	c := &PDICCBased{array: cos.NewArray()}
	c.array.Add(cos.ICCBased)
	c.stream = common.NewPDStream(stream)
	c.array.Add(stream)
	return c
}

// NewPDICCBased reads an ICC based colour space out of its array.
//
// Port of the static PDICCBased.create together with the constructor; Java's
// create looks in the resource cache first, which the port does one level up in
// createFromCOSObject, because only an indirect reference is cacheable. The
// resources are for that cache only: the alternate colour space is built
// without them, as Java builds it.
func NewPDICCBased(iccArray *cos.Array, resources ResourcesLike) (*PDICCBased, error) {
	if err := checkArray(iccArray); err != nil {
		return nil, err
	}
	c := &PDICCBased{array: iccArray}
	c.stream = common.NewPDStream(iccArray.GetObject(1).(*cos.Stream))
	if err := c.loadICCProfile(); err != nil {
		return nil, err
	}
	return c, nil
}

func checkArray(iccArray *cos.Array) error {
	if iccArray.Size() < 2 {
		return errors.New("ICCBased colorspace array must have two elements")
	}
	if _, ok := iccArray.GetObject(1).(*cos.Stream); !ok {
		return errors.New("ICCBased colorspace array must have a stream as second element")
	}
	return nil
}

// loadICCProfile takes the alternate colour space, which is what Java's
// fallbackToAlternateColorSpace does.
func (c *PDICCBased) loadICCProfile() error {
	alternate, err := c.createAlternateColorSpace()
	if err != nil {
		return err
	}
	c.alternateColorSpace = alternate
	if _, isRGB := alternate.(*PDDeviceRGB); isRGB {
		c.isRGBProfile = true
	} else {
		slog.Warn("color: an ICC profile is not read; using the alternate colour space",
			"alternate", alternate.Name())
	}
	c.initialColor = alternate.InitialColor()
	return nil
}

// createAlternateColorSpace is Java's getAlternateColorSpace: the /Alternate
// entry, or the device space the /N entry implies where there is none.
//
// Java wraps a name in an array and builds the array with the one-argument
// PDColorSpace.create, which has no resources, so a name here is a device space
// or an error and is never looked up. Looking it up can lead back to this same
// space: where the resources map /DefaultRGB to an ICCBased space whose
// alternate is /DeviceRGB, /DeviceRGB resolves to that ICCBased space again, and
// round without end.
func (c *PDICCBased) createAlternateColorSpace() (PDColorSpace, error) {
	var alternateArray *cos.Array
	switch alternate := c.stream.Stream().GetDictionaryObject(cos.Alternate).(type) {
	case nil:
		alternateArray = cos.NewArray()
		switch numComponents := c.NumberOfComponents(); numComponents {
		case 1:
			alternateArray.Add(cos.DeviceGray)
		case 3:
			alternateArray.Add(cos.DeviceRGB)
		case 4:
			alternateArray.Add(cos.DeviceCMYK)
		default:
			return nil, fmt.Errorf("Unknown color space number of components:%d", numComponents)
		}
	case *cos.Array:
		alternateArray = alternate
	case *cos.Name:
		alternateArray = cos.NewArray()
		alternateArray.Add(alternate)
	default:
		return nil, fmt.Errorf("Error: expected COSArray or COSName and not %T", alternate)
	}
	return Create(alternateArray)
}

// COSObject returns the array below this colour space.
func (c *PDICCBased) COSObject() cos.Base { return c.array }

// Name returns "ICCBased".
func (c *PDICCBased) Name() string { return cos.ICCBased.Name() }

// PDStream returns the stream holding the profile.
func (c *PDICCBased) PDStream() *common.PDStream { return c.stream }

// NumberOfComponents returns the /N entry of the stream.
func (c *PDICCBased) NumberOfComponents() int {
	if c.stream == nil {
		return 0
	}
	return c.stream.Stream().GetIntDefault(cos.N, 0)
}

// DefaultDecode returns the decode array of the alternate colour space.
//
// Java reads it from the ICC profile, component by component; without the
// profile the alternate is the best the port has, and for the device spaces the
// two agree.
func (c *PDICCBased) DefaultDecode(bitsPerComponent int) []float32 {
	return c.alternateColorSpace.DefaultDecode(bitsPerComponent)
}

// InitialColor returns the initial colour of the alternate colour space.
func (c *PDICCBased) InitialColor() *PDColor { return c.initialColor }

// IsSRGB reports whether the profile is the sRGB one.
//
// Without the profile the port cannot tell, so it answers for the alternate:
// true only where the alternate is DeviceRGB. Java asks the profile itself.
func (c *PDICCBased) IsSRGB() bool { return c.isRGBProfile }

// AlternateColorSpace returns the colour space the profile falls back to.
func (c *PDICCBased) AlternateColorSpace() PDColorSpace { return c.alternateColorSpace }

// ToRGB converts one colour value through the alternate colour space.
func (c *PDICCBased) ToRGB(value []float32) ([]float32, error) {
	return c.alternateColorSpace.ToRGB(value)
}

// ToRGBImage converts a raster through the alternate colour space.
func (c *PDICCBased) ToRGBImage(raster *awtimage.Raster) (goimage.Image, error) {
	return c.alternateColorSpace.ToRGBImage(raster)
}

// ToRawImage returns nil.
//
// Java can answer here, by wrapping the raster in a colour model that carries
// the ICC profile. Without the profile there is nothing to carry.
func (c *PDICCBased) ToRawImage(raster *awtimage.Raster) (goimage.Image, error) {
	return nil, nil
}

// String is Java's toString.
func (c *PDICCBased) String() string { return c.Name() }
