package color

import (
	goimage "image"
	"math"

	awtimage "github.com/shinguakira/pdfbox-go/go/awt/image"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// PDCalGray is a CIE-based A colour space with a single transformation.
//
// Port of org.apache.pdfbox.pdmodel.graphics.color.PDCalGray.
type PDCalGray struct {
	pdCIEDictionaryBasedColorSpace

	initialColor *PDColor

	// PDFBOX-4119: cache the results for much improved performance.
	// Java clones the cached value on the way in and out, because the caller
	// modifies it -- see the rendering of PDFBOX-1724; the port returns a copy
	// for the same reason.
	map1 map[float32][]float32
}

var _ PDColorSpace = (*PDCalGray)(nil)

// NewPDCalGray returns an empty CalGray colour space.
func NewPDCalGray() *PDCalGray {
	c := &PDCalGray{
		pdCIEDictionaryBasedColorSpace: newCIEDictionaryBasedOfName(cos.CalGray),
		map1:                           map[float32][]float32{},
	}
	c.initialColor = NewPDColorOfComponents([]float32{0}, c)
	return c
}

// NewPDCalGrayOfArray reads a CalGray colour space out of its array.
func NewPDCalGrayOfArray(array *cos.Array) *PDCalGray {
	c := &PDCalGray{
		pdCIEDictionaryBasedColorSpace: newCIEDictionaryBasedOfArray(array),
		map1:                           map[float32][]float32{},
	}
	c.initialColor = NewPDColorOfComponents([]float32{0}, c)
	return c
}

// Name returns "CalGray".
func (c *PDCalGray) Name() string { return cos.CalGray.Name() }

// NumberOfComponents returns 1.
func (c *PDCalGray) NumberOfComponents() int { return 1 }

// DefaultDecode maps the full range to 0 to 1.
func (c *PDCalGray) DefaultDecode(bitsPerComponent int) []float32 { return []float32{0, 1} }

// InitialColor returns black.
func (c *PDCalGray) InitialColor() *PDColor { return c.initialColor }

// ToRGB converts one grey value.
func (c *PDCalGray) ToRGB(value []float32) ([]float32, error) {
	// see implementation of toRGB in PDCalRGB, and PDFBOX-2971
	if !c.isWhitePoint() {
		return []float32{value[0], value[0], value[0]}, nil
	}
	a := value[0]
	if result, ok := c.map1[a]; ok {
		return append([]float32(nil), result...), nil
	}
	gamma := c.Gamma()
	powAG := float32(math.Pow(float64(a), float64(gamma)))
	result := convXYZtoRGB(powAG, powAG, powAG)
	c.map1[a] = append([]float32(nil), result...)
	return result, nil
}

// ToRGBImage converts a raster of grey values.
func (c *PDCalGray) ToRGBImage(raster *awtimage.Raster) (goimage.Image, error) {
	return toRGBImageByPixel(raster, c.ToRGB)
}

// Gamma returns the /Gamma entry, defaulting to 1.
func (c *PDCalGray) Gamma() float32 {
	return c.dictionary.GetFloat(cos.Gamma, 1.0)
}

// SetGamma sets the /Gamma entry.
func (c *PDCalGray) SetGamma(value float32) {
	c.dictionary.SetItem(cos.Gamma, cos.NewFloat(value))
}

// String is Java's PDCIEBasedColorSpace.toString, which returns the name.
func (c *PDCalGray) String() string { return c.Name() }
