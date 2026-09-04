package color

import (
	goimage "image"
	"math"

	awtimage "github.com/shinguakira/pdfbox-go/go/awt/image"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// PDCalRGB is a CIE-based ABC colour space with three independent
// transformations.
//
// Port of org.apache.pdfbox.pdmodel.graphics.color.PDCalRGB.
type PDCalRGB struct {
	pdCIEDictionaryBasedColorSpace

	initialColor *PDColor
}

var _ PDColorSpace = (*PDCalRGB)(nil)

// NewPDCalRGB returns an empty CalRGB colour space.
func NewPDCalRGB() *PDCalRGB {
	c := &PDCalRGB{pdCIEDictionaryBasedColorSpace: newCIEDictionaryBasedOfName(cos.CalRGB)}
	c.initialColor = NewPDColorOfComponents([]float32{0, 0, 0}, c)
	return c
}

// NewPDCalRGBOfArray reads a CalRGB colour space out of its array.
func NewPDCalRGBOfArray(array *cos.Array) *PDCalRGB {
	c := &PDCalRGB{pdCIEDictionaryBasedColorSpace: newCIEDictionaryBasedOfArray(array)}
	c.initialColor = NewPDColorOfComponents([]float32{0, 0, 0}, c)
	return c
}

// Name returns "CalRGB".
func (c *PDCalRGB) Name() string { return cos.CalRGB.Name() }

// NumberOfComponents returns 3.
func (c *PDCalRGB) NumberOfComponents() int { return 3 }

// DefaultDecode maps the full range of each component to 0 to 1.
func (c *PDCalRGB) DefaultDecode(bitsPerComponent int) []float32 {
	return []float32{0, 1, 0, 1, 0, 1}
}

// InitialColor returns black.
func (c *PDCalRGB) InitialColor() *PDColor { return c.initialColor }

// ToRGB converts one colour value.
func (c *PDCalRGB) ToRGB(value []float32) ([]float32, error) {
	if !c.isWhitePoint() {
		// this is a hack, we simply skip CIE calibration of the RGB value
		// this works only with whitepoint D65 (0.9505 1.0 1.089)
		// see PDFBOX-2553
		return []float32{value[0], value[1], value[2]}, nil
	}
	a := value[0]
	b := value[1]
	cc := value[2]

	gamma := c.Gamma()
	powAR := float32(math.Pow(float64(a), float64(gamma.R())))
	powBG := float32(math.Pow(float64(b), float64(gamma.G())))
	powCB := float32(math.Pow(float64(cc), float64(gamma.B())))

	matrix := c.Matrix()
	mXA := matrix[0]
	mYA := matrix[1]
	mZA := matrix[2]
	mXB := matrix[3]
	mYB := matrix[4]
	mZB := matrix[5]
	mXC := matrix[6]
	mYC := matrix[7]
	mZC := matrix[8]

	x := mXA*powAR + mXB*powBG + mXC*powCB
	y := mYA*powAR + mYB*powBG + mYC*powCB
	z := mZA*powAR + mZB*powBG + mZC*powCB

	return convXYZtoRGB(x, y, z), nil
}

// ToRGBImage converts a raster of colour values.
func (c *PDCalRGB) ToRGBImage(raster *awtimage.Raster) (goimage.Image, error) {
	return toRGBImageByPixel(raster, c.ToRGB)
}

// Gamma returns the /Gamma entry, writing the default 1 1 1 into the
// dictionary where there is none, which is what Java does.
func (c *PDCalRGB) Gamma() *PDGamma {
	gammaArray := c.dictionary.GetCOSArray(cos.Gamma)
	if gammaArray == nil {
		gammaArray = cos.NewArray()
		gammaArray.Add(cos.FloatOne)
		gammaArray.Add(cos.FloatOne)
		gammaArray.Add(cos.FloatOne)
		c.dictionary.SetItem(cos.Gamma, gammaArray)
	}
	return NewPDGammaOfArray(gammaArray)
}

// Matrix returns the /Matrix entry, defaulting to the identity.
func (c *PDCalRGB) Matrix() []float32 {
	matrix := c.dictionary.GetCOSArray(cos.Matrix)
	if matrix == nil {
		return []float32{1, 0, 0, 0, 1, 0, 0, 0, 1}
	}
	return matrix.ToFloatArray()
}

// SetGamma sets the /Gamma entry.
func (c *PDCalRGB) SetGamma(gamma *PDGamma) {
	var gammaArray cos.Base
	if gamma != nil {
		gammaArray = gamma.COSArray()
	}
	c.dictionary.SetItem(cos.Gamma, gammaArray)
}

// SetMatrix sets the /Matrix entry.
func (c *PDCalRGB) SetMatrix(matrix *util.Matrix) {
	var matrixArray cos.Base
	if matrix != nil {
		// We can't use matrix.toCOSArray(), as it only returns a subset of the matrix
		values := matrix.Values()
		array := cos.NewArray()
		array.Add(cos.NewFloat(values[0][0]))
		array.Add(cos.NewFloat(values[0][1]))
		array.Add(cos.NewFloat(values[0][2]))
		array.Add(cos.NewFloat(values[1][0]))
		array.Add(cos.NewFloat(values[1][1]))
		array.Add(cos.NewFloat(values[1][2]))
		array.Add(cos.NewFloat(values[2][0]))
		array.Add(cos.NewFloat(values[2][1]))
		array.Add(cos.NewFloat(values[2][2]))
		matrixArray = array
	}
	c.dictionary.SetItem(cos.Matrix, matrixArray)
}

// String is Java's PDCIEBasedColorSpace.toString, which returns the name.
func (c *PDCalRGB) String() string { return c.Name() }
