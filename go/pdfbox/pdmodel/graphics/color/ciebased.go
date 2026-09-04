package color

import (
	goimage "image"
	"math"

	awtimage "github.com/shinguakira/pdfbox-go/go/awt/image"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// pdCIEBasedColorSpace carries what the CIE based colour spaces share.
//
// Port of the abstract PDCIEBasedColorSpace. Java's toRGBImage there converts
// one pixel at a time through toRGB, which the concrete space overrides; Go has
// no such dispatch from an embedded type, so the loop takes the conversion as
// an argument.
type pdCIEBasedColorSpace struct {
	array *cos.Array
}

// COSObject returns the array below this colour space.
func (c *pdCIEBasedColorSpace) COSObject() cos.Base { return c.array }

// ToRawImage returns nil.
//
// There is no direct equivalent of a CIE colorspace in Java. So we can not do
// anything here.
func (c *pdCIEBasedColorSpace) ToRawImage(raster *awtimage.Raster) (goimage.Image, error) {
	return nil, nil
}

// toRGBImageByPixel is Java's PDCIEBasedColorSpace.toRGBImage.
//
// This method calls toRGB to convert images one pixel at a time. For
// matrix-based CIE color spaces this is fast enough. However, it should not be
// used with any color space which uses an ICC Profile as it will be far too
// slow.
func toRGBImageByPixel(raster *awtimage.Raster,
	toRGB func(abc []float32) ([]float32, error)) (goimage.Image, error) {
	width := raster.Width()
	height := raster.Height()
	rgbImage := newRGBImage(width, height)

	// always three components: ABC
	//
	// Java hands getPixel a float[3] whatever the raster holds, so a one band
	// raster -- which is what CalGray has -- leaves abc[1] and abc[2] at
	// whatever the previous pixel left there. It does not matter, because the
	// toRGB of a one component space reads only abc[0]; the port keeps the same
	// three element array so that it does not matter here either.
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
			rgb, err := toRGB(abc)
			if err != nil {
				return nil, err
			}
			// 0..1 -> 0..255
			setRGB(rgbImage, x, y, rgb[0]*255, rgb[1]*255, rgb[2]*255)
		}
	}
	return rgbImage, nil
}

// pdCIEDictionaryBasedColorSpace is a CIE based colour space whose parameters
// live in a dictionary: CalGray, CalRGB and Lab.
//
// Port of the abstract PDCIEDictionaryBasedColorSpace.
type pdCIEDictionaryBasedColorSpace struct {
	pdCIEBasedColorSpace

	dictionary *cos.Dictionary

	// we need to cache whitepoint values, because using getWhitePoint()
	// would create a new default object for each pixel conversion if the
	// original PDF didn't have a whitepoint array
	wpX float32
	wpY float32
	wpZ float32
}

// newCIEDictionaryBasedOfName builds an empty colour space of the given kind.
func newCIEDictionaryBasedOfName(cosName *cos.Name) pdCIEDictionaryBasedColorSpace {
	c := pdCIEDictionaryBasedColorSpace{dictionary: cos.NewDictionary()}
	c.array = cos.NewArray()
	c.array.Add(cosName)
	c.array.Add(c.dictionary)
	c.fillWhitepointCache(c.Whitepoint())
	return c
}

// newCIEDictionaryBasedOfArray reads a colour space out of its array.
func newCIEDictionaryBasedOfArray(array *cos.Array) pdCIEDictionaryBasedColorSpace {
	c := pdCIEDictionaryBasedColorSpace{}
	c.array = array
	// Java casts without a check, so an array whose second entry is not a
	// dictionary throws ClassCastException; the port panics for the same.
	c.dictionary = array.GetObject(1).(*cos.Dictionary)
	c.fillWhitepointCache(c.Whitepoint())
	return c
}

func (c *pdCIEDictionaryBasedColorSpace) isWhitePoint() bool {
	return c.wpX == 1 && c.wpY == 1 && c.wpZ == 1
}

func (c *pdCIEDictionaryBasedColorSpace) fillWhitepointCache(whitepoint *PDTristimulus) {
	c.wpX = whitepoint.X()
	c.wpY = whitepoint.Y()
	c.wpZ = whitepoint.Z()
}

// convXYZtoRGB converts a CIE XYZ colour to sRGB.
//
// Java hands this to ColorSpace.getInstance(ColorSpace.CS_CIEXYZ).toRGB, which
// goes through the JRE built-in profiles and LittleCMS. Go has no ICC engine --
// see the note on PDDeviceCMYK -- so the port writes the transform out: the XYZ
// that CS_CIEXYZ carries is relative to the D50 white point, so this is the
// Bradford adaptation from D50 to D65 folded into the sRGB primaries, then the
// sRGB transfer function. That is the standard definition of the same
// conversion; it agrees with LittleCMS to within the rounding of the two, and
// not bit for bit.
func convXYZtoRGB(x, y, z float32) []float32 {
	// toRGB() malfunctions with negative values
	// XYZ must be non-negative anyway:
	// http://ninedegreesbelow.com/photography/icc-profile-negative-tristimulus.html
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if z < 0 {
		z = 0
	}

	bigX, bigY, bigZ := float64(x), float64(y), float64(z)
	r := 3.1338561*bigX - 1.6168667*bigY - 0.4906146*bigZ
	g := -0.9787684*bigX + 1.9161415*bigY + 0.0334540*bigZ
	b := 0.0719453*bigX - 0.2289914*bigY + 1.4052427*bigZ
	return []float32{srgbTransfer(r), srgbTransfer(g), srgbTransfer(b)}
}

// srgbTransfer applies the sRGB transfer function and clamps to 0..1.
func srgbTransfer(value float64) float32 {
	switch {
	case value <= 0:
		return 0
	case value >= 1:
		return 1
	case value <= 0.0031308:
		return float32(value * 12.92)
	}
	return float32(1.055*math.Pow(value, 1/2.4) - 0.055)
}

// Whitepoint returns the /WhitePoint entry, defaulting to 1 1 1.
func (c *pdCIEDictionaryBasedColorSpace) Whitepoint() *PDTristimulus {
	wp := c.dictionary.GetCOSArray(cos.WhitePoint)
	if wp == nil {
		wp = cos.NewArray()
		wp.Add(cos.FloatOne)
		wp.Add(cos.FloatOne)
		wp.Add(cos.FloatOne)
	}
	return NewPDTristimulusOfArray(wp)
}

// BlackPoint returns the /BlackPoint entry, defaulting to 0 0 0.
func (c *pdCIEDictionaryBasedColorSpace) BlackPoint() *PDTristimulus {
	bp := c.dictionary.GetCOSArray(cos.BlackPoint)
	if bp == nil {
		bp = cos.NewArray()
		bp.Add(cos.FloatZero)
		bp.Add(cos.FloatZero)
		bp.Add(cos.FloatZero)
	}
	return NewPDTristimulusOfArray(bp)
}

// SetWhitePoint sets the /WhitePoint entry.
//
// Java throws IllegalArgumentException for a null whitepoint, which is
// unchecked, so the port panics.
func (c *pdCIEDictionaryBasedColorSpace) SetWhitePoint(whitepoint *PDTristimulus) {
	if whitepoint == nil {
		panic("Whitepoint may not be null")
	}
	c.dictionary.SetItem(cos.WhitePoint, whitepoint.COSObject())
	c.fillWhitepointCache(whitepoint)
}

// SetBlackPoint sets the /BlackPoint entry.
func (c *pdCIEDictionaryBasedColorSpace) SetBlackPoint(blackpoint *PDTristimulus) {
	c.dictionary.SetItem(cos.BlackPoint, blackpoint.COSObject())
}
