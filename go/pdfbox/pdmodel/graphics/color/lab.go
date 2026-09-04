package color

import (
	goimage "image"

	awtimage "github.com/shinguakira/pdfbox-go/go/awt/image"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// PDLab is a CIE-based ABC colour space with two ranges, the Lab space.
//
// Port of org.apache.pdfbox.pdmodel.graphics.color.PDLab.
type PDLab struct {
	pdCIEDictionaryBasedColorSpace

	initialColor *PDColor
}

var _ PDColorSpace = (*PDLab)(nil)

// NewPDLab returns an empty Lab colour space.
func NewPDLab() *PDLab {
	return &PDLab{pdCIEDictionaryBasedColorSpace: newCIEDictionaryBasedOfName(cos.Lab)}
}

// NewPDLabOfArray reads a Lab colour space out of its array.
func NewPDLabOfArray(array *cos.Array) *PDLab {
	return &PDLab{pdCIEDictionaryBasedColorSpace: newCIEDictionaryBasedOfArray(array)}
}

// Name returns "Lab".
func (c *PDLab) Name() string { return cos.Lab.Name() }

// ToRGBImage converts a raster of Lab values.
//
// The Lab components are not 0 to 1 like the other CIE spaces, so this does not
// go through the shared per-pixel loop: it scales each of the three into its
// own range first.
func (c *PDLab) ToRGBImage(raster *awtimage.Raster) (goimage.Image, error) {
	//
	// WARNING: this method is performance sensitive, modify with care!
	//
	width := raster.Width()
	height := raster.Height()
	rgbImage := newRGBImage(width, height)

	aRange := c.ARange()
	bRange := c.BRange()
	minA := aRange.Min()
	maxA := aRange.Max()
	minB := bRange.Min()
	maxB := bRange.Max()
	deltaA := maxA - minA
	deltaB := maxB - minB

	// always three components: ABC
	bands := raster.NumBands()
	pixel := make([]int, max(3, bands))
	abc := make([]float32, 3)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			raster.GetPixel(x, y, pixel)
			// 0..255 -> 0..1
			for b := 0; b < 3 && b < bands; b++ {
				abc[b] = float32(pixel[b]) / 255
			}
			// scale to range
			abc[0] *= 100
			abc[1] = minA + abc[1]*deltaA
			abc[2] = minB + abc[2]*deltaB

			rgb, err := c.ToRGB(abc)
			if err != nil {
				return nil, err
			}
			// 0..1 -> 0..255
			setRGB(rgbImage, x, y, rgb[0]*255, rgb[1]*255, rgb[2]*255)
		}
	}
	return rgbImage, nil
}

// ToRawImage returns nil: not handled at the moment.
func (c *PDLab) ToRawImage(raster *awtimage.Raster) (goimage.Image, error) { return nil, nil }

// ToRGB converts one Lab value.
func (c *PDLab) ToRGB(value []float32) ([]float32, error) {
	// CIE LAB to RGB, see http://en.wikipedia.org/wiki/Lab_color_space

	// L*
	lstar := (value[0] + 16) * (1.0 / 116.0)

	// TODO: how to use the blackpoint? scale linearly between black & white?

	// XYZ
	x := c.wpX * labInverse(lstar+value[1]*(1.0/500.0))
	y := c.wpY * labInverse(lstar)
	z := c.wpZ * labInverse(lstar-value[2]*(1.0/200.0))

	return convXYZtoRGB(x, y, z), nil
}

// labInverse is the reverse transformation (f^-1).
func labInverse(x float32) float32 {
	if float64(x) > 6.0/29.0 {
		return x * x * x
	}
	return (108.0 / 841.0) * (x - (4.0 / 29.0))
}

// NumberOfComponents returns 3.
func (c *PDLab) NumberOfComponents() int { return 3 }

// DefaultDecode returns the L range and the two component ranges.
func (c *PDLab) DefaultDecode(bitsPerComponent int) []float32 {
	a := c.ARange()
	b := c.BRange()
	return []float32{0, 100, a.Min(), a.Max(), b.Min(), b.Max()}
}

// InitialColor returns black at the bottom of both ranges.
func (c *PDLab) InitialColor() *PDColor {
	if c.initialColor == nil {
		c.initialColor = NewPDColorOfComponents([]float32{
			0,
			maxFloat32(0, c.ARange().Min()),
			maxFloat32(0, c.BRange().Min()),
		}, c)
	}
	return c.initialColor
}

func maxFloat32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func (c *PDLab) defaultRangeArray() *cos.Array {
	minus100 := cos.NewFloat(-100)
	plus100 := cos.NewFloat(100)
	rangeArray := cos.NewArray()
	rangeArray.Add(minus100)
	rangeArray.Add(plus100)
	rangeArray.Add(minus100)
	rangeArray.Add(plus100)
	return rangeArray
}

// ARange returns the range of the a component.
func (c *PDLab) ARange() *common.PDRange {
	rangeArray := c.dictionary.GetCOSArray(cos.Range)
	if rangeArray == nil {
		rangeArray = c.defaultRangeArray()
	}
	return common.NewPDRangeOfIndex(rangeArray, 0)
}

// BRange returns the range of the b component.
func (c *PDLab) BRange() *common.PDRange {
	rangeArray := c.dictionary.GetCOSArray(cos.Range)
	if rangeArray == nil {
		rangeArray = c.defaultRangeArray()
	}
	return common.NewPDRangeOfIndex(rangeArray, 1)
}

// SetARange sets the range of the a component.
func (c *PDLab) SetARange(r *common.PDRange) { c.setComponentRangeArray(r, 0) }

// SetBRange sets the range of the b component.
func (c *PDLab) SetBRange(r *common.PDRange) { c.setComponentRangeArray(r, 2) }

func (c *PDLab) setComponentRangeArray(r *common.PDRange, index int) {
	rangeArray := c.dictionary.GetCOSArray(cos.Range)
	if rangeArray == nil {
		rangeArray = c.defaultRangeArray()
	}
	if r == nil {
		// reset to defaults
		rangeArray.Set(index, cos.NewFloat(-100))
		rangeArray.Set(index+1, cos.NewFloat(100))
	} else {
		rangeArray.Set(index, cos.NewFloat(r.Min()))
		rangeArray.Set(index+1, cos.NewFloat(r.Max()))
	}
	c.dictionary.SetItem(cos.Range, rangeArray)
	c.initialColor = nil
}

// String is Java's PDCIEBasedColorSpace.toString, which returns the name.
func (c *PDLab) String() string { return c.Name() }
