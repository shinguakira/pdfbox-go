package color

import (
	"fmt"
	goimage "image"
	"math"

	awtimage "github.com/shinguakira/pdfbox-go/go/awt/image"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// PDIndexed is a colour space whose values index a table of colours in another
// space.
//
// Port of org.apache.pdfbox.pdmodel.graphics.color.PDIndexed.
type PDIndexed struct {
	array        *cos.Array
	initialColor *PDColor

	baseColorSpace PDColorSpace

	// cached lookup data
	lookupData     []byte
	colorTable     [][]float32
	actualMaxIndex int
	rgbColorTable  [][]int
}

var _ PDColorSpace = (*PDIndexed)(nil)

// NewPDIndexedOfArray reads an indexed colour space out of its array.
func NewPDIndexedOfArray(indexedArray *cos.Array, resources ResourcesLike) (*PDIndexed, error) {
	c := &PDIndexed{array: indexedArray}
	c.initialColor = NewPDColorOfComponents([]float32{0}, c)

	// don't call getObject(1), we want to pass a reference if possible
	// to profit from caching (PDFBOX-4149)
	base, err := CreateOfResources(indexedArray.Get(1), resources)
	if err != nil {
		return nil, err
	}
	c.baseColorSpace = base
	if err := c.readColorTable(); err != nil {
		return nil, err
	}
	if err := c.initRgbColorTable(); err != nil {
		return nil, err
	}
	return c, nil
}

// NewPDIndexed builds an indexed colour space over the given base.
//
// Port of the static PDIndexed.create. Java throws IllegalArgumentException for
// each of the four checks, which is unchecked, so the port panics.
func NewPDIndexed(base PDColorSpace, hival int, lookupData []byte) (*PDIndexed, error) {
	if base == nil {
		panic("base must not be null")
	}
	if lookupData == nil {
		panic("lookupData must not be null")
	}
	if hival < 0 || hival > 255 {
		panic(" hival has to be a positive value <= 255")
	}
	expected := (hival + 1) * base.NumberOfComponents()
	if len(lookupData) < expected {
		panic(fmt.Sprintf("lookupData too short: expected at least %d bytes "+
			"((hival+1) * components), got %d", expected, len(lookupData)))
	}

	c := &PDIndexed{}
	c.initialColor = NewPDColorOfComponents([]float32{0}, c)
	c.array = cos.NewArray()
	c.array.Add(cos.Indexed)
	c.baseColorSpace = base
	c.array.AddAt(1, base.COSObject())
	c.array.AddAt(2, cos.GetInteger(int64(hival)))
	c.lookupData = append([]byte(nil), lookupData...)
	c.array.AddAt(3, cos.NewStringObjBytesHex(c.lookupData, true))
	if err := c.readColorTable(); err != nil {
		return nil, err
	}
	if err := c.initRgbColorTable(); err != nil {
		return nil, err
	}
	return c, nil
}

// COSObject returns the array below this colour space.
func (c *PDIndexed) COSObject() cos.Base { return c.array }

// Name returns "Indexed".
func (c *PDIndexed) Name() string { return cos.Indexed.Name() }

// NumberOfComponents returns 1: an index.
func (c *PDIndexed) NumberOfComponents() int { return 1 }

// DefaultDecode maps the full range of the sample to the index range.
func (c *PDIndexed) DefaultDecode(bitsPerComponent int) []float32 {
	return []float32{0, float32(math.Pow(2, float64(bitsPerComponent))) - 1}
}

// InitialColor returns index 0.
func (c *PDIndexed) InitialColor() *PDColor { return c.initialColor }

// WARNING: this method is performance sensitive, modify with care!
func (c *PDIndexed) initRgbColorTable() error {
	numBaseComponents := c.baseColorSpace.NumberOfComponents()

	// convert the color table into a 1-row image in the base color space,
	// using a writable raster for high performance
	if c.actualMaxIndex+1 <= 0 || numBaseComponents <= 0 {
		// PDFBOX-4503: when stream is empty or null. Java's
		// Raster.createBandedRaster throws IllegalArgumentException here and
		// PDIndexed turns it into an IOException.
		return fmt.Errorf("color: cannot build a colour table of %d entries of %d components",
			c.actualMaxIndex+1, numBaseComponents)
	}
	baseRaster := awtimage.NewInterleavedRaster(awtimage.TypeByte,
		c.actualMaxIndex+1, 1, numBaseComponents)

	base := make([]int, numBaseComponents)
	for i, n := 0, c.actualMaxIndex; i <= n; i++ {
		for comp := 0; comp < numBaseComponents; comp++ {
			base[comp] = int(c.colorTable[i][comp] * 255)
		}
		baseRaster.SetPixel(i, 0, base)
	}

	// convert the base image to RGB
	rgbImage, err := c.baseColorSpace.ToRGBImage(baseRaster)
	if err != nil {
		return err
	}

	// build an RGB lookup table from the raster
	c.rgbColorTable = make([][]int, c.actualMaxIndex+1)
	for i, n := 0, c.actualMaxIndex; i <= n; i++ {
		c.rgbColorTable[i] = pixelOfImage(rgbImage, i, 0)
	}
	return nil
}

// pixelOfImage reads one pixel of a colour space's RGB image back as three
// samples, which is Java's rgbRaster.getPixel(x, y, null).
func pixelOfImage(img goimage.Image, x, y int) []int {
	if rgba, ok := img.(*goimage.RGBA); ok {
		return pixelAt(rgba, x, y, make([]int, 3))
	}
	r, g, b, _ := img.At(x, y).RGBA()
	return []int{int(r >> 8), int(g >> 8), int(b >> 8)}
}

// ToRGB looks the index up in the colour table.
//
// WARNING: this method is performance sensitive, modify with care!
func (c *PDIndexed) ToRGB(value []float32) ([]float32, error) {
	if len(value) != 1 {
		// Java throws IllegalArgumentException, which is unchecked.
		panic("Indexed color spaces must have one color value")
	}

	// scale and clamp input value
	index := int(math.Round(float64(value[0])))
	if index < 0 {
		index = 0
	}
	if index > c.actualMaxIndex {
		index = c.actualMaxIndex
	}

	// lookup rgb
	rgb := c.rgbColorTable[index]
	return []float32{float32(rgb[0]) / 255, float32(rgb[1]) / 255, float32(rgb[2]) / 255}, nil
}

// ToRGBImage looks every sample up in the colour table.
//
// WARNING: this method is performance sensitive, modify with care!
func (c *PDIndexed) ToRGBImage(raster *awtimage.Raster) (goimage.Image, error) {
	// use lookup table
	width := raster.Width()
	height := raster.Height()
	rgbImage := newRGBImage(width, height)

	src := make([]int, width*raster.NumBands())
	for y := 0; y < height; y++ {
		raster.GetPixels(0, y, width, 1, src)
		for x := 0; x < width; x++ {
			// lookup
			index := src[x]
			if index > c.actualMaxIndex {
				index = c.actualMaxIndex
			}
			rgb := c.rgbColorTable[index]
			setRGB(rgbImage, x, y, float32(rgb[0]), float32(rgb[1]), float32(rgb[2]))
		}
	}
	return rgbImage, nil
}

// ToRawImage returns nil.
//
// Java can answer here for an indexed space over an sRGB ICC profile, by
// building an IndexColorModel; Go has no indexed colour model and no ICC
// engine, so this port cannot, and the caller falls back to ToRGBImage.
func (c *PDIndexed) ToRawImage(raster *awtimage.Raster) (goimage.Image, error) {
	// We can't handle all other cases at the moment.
	return nil, nil
}

// BaseColorSpace returns the colour space the table holds colours in.
func (c *PDIndexed) BaseColorSpace() PDColorSpace { return c.baseColorSpace }

// hival returns the "hival" array element.
func (c *PDIndexed) hival() int {
	return int(c.array.GetObject(2).(cos.Number).IntValue())
}

// readLookupData reads the lookup table data from the array.
func (c *PDIndexed) readLookupData() error {
	if c.lookupData != nil {
		return nil
	}
	switch lookupTable := c.array.GetObject(3).(type) {
	case *cos.StringObj:
		c.lookupData = lookupTable.Bytes()
	case *cos.Stream:
		data, err := common.NewPDStream(lookupTable).ToByteArray()
		if err != nil {
			return err
		}
		c.lookupData = data
	case nil:
		c.lookupData = []byte{}
	default:
		return fmt.Errorf("Error: Unknown type for lookup table %v", lookupTable)
	}
	return nil
}

// readColorTable reads the colour table out of the lookup data.
//
// WARNING: this method is performance sensitive, modify with care!
func (c *PDIndexed) readColorTable() error {
	if err := c.readLookupData(); err != nil {
		return err
	}

	maxIndex := c.hival()
	if maxIndex > 255 {
		maxIndex = 255
	}
	numComponents := c.baseColorSpace.NumberOfComponents()

	// some tables are too short
	if numComponents > 0 && len(c.lookupData)/numComponents < maxIndex+1 {
		maxIndex = len(c.lookupData)/numComponents - 1
	}
	c.actualMaxIndex = maxIndex // TODO "actual" is ugly, tidy this up

	c.colorTable = make([][]float32, maxIndex+1)
	offset := 0
	for i := 0; i <= maxIndex; i++ {
		c.colorTable[i] = make([]float32, numComponents)
		for comp := 0; comp < numComponents; comp++ {
			c.colorTable[i][comp] = float32(c.lookupData[offset]&0xff) / 255
			offset++
		}
	}
	return nil
}

// String is Java's toString.
func (c *PDIndexed) String() string {
	return fmt.Sprintf("Indexed{base:%v hival:%d lookup:(%d entries)}",
		c.baseColorSpace, c.hival(), len(c.colorTable))
}
