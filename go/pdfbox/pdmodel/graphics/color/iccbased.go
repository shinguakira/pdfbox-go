package color

import (
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
// createFromCOSObject, because only an indirect reference is cacheable.
func NewPDICCBased(iccArray *cos.Array, resources ResourcesLike) (*PDICCBased, error) {
	c := &PDICCBased{array: iccArray}
	stream, ok := iccArray.GetObject(1).(*cos.Stream)
	if ok {
		c.stream = common.NewPDStream(stream)
	}
	if err := c.loadICCProfile(resources); err != nil {
		return nil, err
	}
	return c, nil
}

// loadICCProfile takes the alternate colour space, which is what Java's
// fallbackToAlternateColorSpace does.
func (c *PDICCBased) loadICCProfile(resources ResourcesLike) error {
	alternate, err := c.alternateColorSpaceOf(resources)
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

// alternateColorSpaceOf is Java's getAlternateColorSpace, which reads the
// /Alternate entry and falls back to the device space the /N entry implies.
func (c *PDICCBased) alternateColorSpaceOf(resources ResourcesLike) (PDColorSpace, error) {
	alternate := c.array.GetObject(1)
	if stream, ok := alternate.(*cos.Stream); ok {
		if entry := stream.GetDictionaryObject(cos.Alternate); entry != nil {
			if array, ok := entry.(*cos.Array); !ok || !array.IsEmpty() {
				return CreateWithResources(entry, resources, false)
			}
		}
		// no alternate color space, use the color space with the correct
		// number of components
		switch c.NumberOfComponents() {
		case 1:
			return DeviceGray, nil
		case 3:
			return DeviceRGB, nil
		case 4:
			return DeviceCMYK, nil
		}
	}
	// PDFBOX-4801: the /N entry is missing or wrong; Java's getNumberOfComponents
	// reads it the same way and its switch has no default, which leaves the
	// alternate null and throws later. The port reports it here instead.
	return nil, cosNumberOfComponentsError(c.NumberOfComponents())
}

func cosNumberOfComponentsError(n int) error {
	return &numberOfComponentsError{n}
}

type numberOfComponentsError struct{ n int }

func (e *numberOfComponentsError) Error() string {
	return "Invalid /N value in ICC based colour space: " + itoa(e.n)
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	negative := v < 0
	if negative {
		v = -v
	}
	var digits []byte
	for v > 0 {
		digits = append([]byte{byte('0' + v%10)}, digits...)
		v /= 10
	}
	if negative {
		return "-" + string(digits)
	}
	return string(digits)
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
