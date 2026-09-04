package image

import (
	"errors"
	"fmt"
	goimage "image"
	goimagecolor "image/color"
	"io"
	"log/slog"
	"math"

	awtgeom "github.com/shinguakira/pdfbox-go/go/awt/geom"
	awtimage "github.com/shinguakira/pdfbox-go/go/awt/image"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/filter"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
)

// The reader that turns the samples of a PDF image into a picture.
//
// Port of org.apache.pdfbox.pdmodel.graphics.image.SampledImageReader, a final
// class of static methods, which the port keeps as functions of the package.

// getStencilImage returns the content of a stencil mask as an image filled with
// the given colour.
//
// Port of getStencilImage(PDImage, Paint). Java fills the whole image with the
// Paint and then clears the alpha where the mask says so; the port takes a solid
// colour rather than a Paint, because a Paint is a rendering object and slice 9
// owns those.
func getStencilImage(pdImage PDImage, paint goimagecolor.Color) (goimage.Image, error) {
	width := pdImage.Width()
	height := pdImage.Height()

	// compose to ARGB
	masked := goimage.NewRGBA(goimage.Rect(0, 0, width, height))
	r, g, b, a := paint.RGBA()
	fill := goimagecolor.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			masked.SetRGBA(x, y, fill)
		}
	}

	// avoid getting a BufferedImage for the mask to lessen memory footprint.
	// Such masks are always bpc=1 and have no colorspace, but have a decode.
	// (see 8.9.6.2 Stencil Masking)
	iis, err := pdImage.CreateInputStream()
	if err != nil {
		return nil, err
	}
	decode, err := getDecodeArray(pdImage)
	if err != nil {
		return nil, err
	}
	value := 0
	if decode[0] < decode[1] {
		value = 1
	}
	rowLen := width / 8
	if width%8 > 0 {
		rowLen++
	}
	buff := make([]byte, rowLen)
	for y := 0; y < height; y++ {
		x := 0
		readLen, _ := io.ReadFull(iis, buff)
		for r := 0; r < rowLen && r < readLen; r++ {
			byteValue := int(int8(buff[r]))
			mask := 128
			shift := uint(7)
			for i := 0; i < 8; i++ {
				bit := (byteValue & mask) >> shift
				mask >>= 1
				if shift > 0 {
					shift--
				}
				if bit == value {
					masked.SetRGBA(x, y, goimagecolor.RGBA{})
				}
				x++
				if x == width {
					break
				}
			}
		}
		if readLen != rowLen {
			slog.Warn("image: premature EOF, image will be incomplete")
			break
		}
	}
	return masked, nil
}

// getRGBImage returns the whole image as RGB.
//
// Port of getRGBImage(PDImage, COSArray).
func getRGBImage(pdImage PDImage, colorKey *cos.Array) (goimage.Image, error) {
	return getRGBImageOfRegion(pdImage, nil, 1, colorKey)
}

// clipRegion is Java's clipRegion.
func clipRegion(pdImage PDImage, region *awtgeom.Rectangle) awtgeom.Rectangle {
	if region == nil {
		return awtgeom.Rectangle{X: 0, Y: 0, Width: pdImage.Width(), Height: pdImage.Height()}
	}
	x := max(0, region.X)
	y := max(0, region.Y)
	return awtgeom.Rectangle{
		X:      x,
		Y:      y,
		Width:  min(region.Width, pdImage.Width()-x),
		Height: min(region.Height, pdImage.Height()-y),
	}
}

// getRGBImageOfRegion returns part of an image as RGB, subsampled.
//
// Port of getRGBImage(PDImage, Rectangle, int, COSArray).
func getRGBImageOfRegion(pdImage PDImage, region *awtgeom.Rectangle, subsampling int,
	colorKey *cos.Array) (goimage.Image, error) {
	if pdImage.IsEmpty() {
		return nil, errors.New("Image stream is empty")
	}
	clipped := clipRegion(pdImage, region)

	// get parameters, they must be valid or have been repaired
	colorSpace, err := pdImage.ColorSpace()
	if err != nil {
		return nil, err
	}
	numComponents := colorSpace.NumberOfComponents()
	width := int(math.Ceil(float64(clipped.Width) / float64(subsampling)))
	height := int(math.Ceil(float64(clipped.Height) / float64(subsampling)))
	bitsPerComponent := pdImage.BitsPerComponent()

	if width <= 0 || height <= 0 || pdImage.Width() <= 0 || pdImage.Height() <= 0 {
		return nil, errors.New("image width and height must be positive")
	}

	if bitsPerComponent == 1 && colorKey == nil && numComponents == 1 {
		return from1Bit(pdImage, clipped, subsampling, width, height)
	}

	// An AWT raster must use 8/16/32 bits per component. Images with < 8bpc
	// will be unpacked into a byte-backed raster. Images with 16bpc will be reduced
	// in depth to 8bpc as they will be drawn to TYPE_INT_RGB images anyway. All code
	// in PDColorSpace#toRGBImage expects an 8-bit range, i.e. 0-255.
	// Interleaved raster allows chunk-copying for 8-bit images.
	raster := awtimage.NewInterleavedRaster(awtimage.TypeByte, width, height, numComponents)
	defaultDecode := colorSpace.DefaultDecode(8)
	decode, err := getDecodeArray(pdImage)
	if err != nil {
		return nil, err
	}
	if bitsPerComponent == 8 && colorKey == nil && floatsEqual(decode, defaultDecode) {
		// convert image, faster path for non-decoded, non-colormasked 8-bit images
		return from8bit(pdImage, raster, clipped, subsampling, width, height)
	}
	return fromAny(pdImage, raster, colorKey, clipped, subsampling, width, height)
}

func floatsEqual(a, b []float32) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// getRawRaster returns the samples of an image, undecoded.
//
// Port of getRawRaster(PDImage).
func getRawRaster(pdImage PDImage) (*awtimage.Raster, error) {
	if pdImage.IsEmpty() {
		return nil, errors.New("Image stream is empty")
	}

	// get parameters, they must be valid or have been repaired
	colorSpace, err := pdImage.ColorSpace()
	if err != nil {
		return nil, err
	}
	numComponents := colorSpace.NumberOfComponents()
	width := pdImage.Width()
	height := pdImage.Height()
	bitsPerComponent := pdImage.BitsPerComponent()

	if width <= 0 || height <= 0 {
		return nil, errors.New("image width and height must be positive")
	}

	dataBufferType := awtimage.TypeByte
	if bitsPerComponent > 8 {
		dataBufferType = awtimage.TypeUShort
	}
	raster := awtimage.NewInterleavedRaster(dataBufferType, width, height, numComponents)
	if err := readRasterFromAny(pdImage, raster); err != nil {
		return nil, err
	}
	return raster, nil
}

func readRasterFromAny(pdImage PDImage, raster *awtimage.Raster) error {
	colorSpace, err := pdImage.ColorSpace()
	if err != nil {
		return err
	}
	numComponents := colorSpace.NumberOfComponents()
	bitsPerComponent := pdImage.BitsPerComponent()
	decode, err := getDecodeArray(pdImage)
	if err != nil {
		return err
	}

	options := filter.NewDecodeOptions()
	imageStream, err := pdImage.CreateInputStreamWithOptions(options)
	if err != nil {
		return err
	}
	// read bit stream
	iis := newImageBitReader(imageStream)

	inputWidth := pdImage.Width()
	scanWidth := pdImage.Width()
	scanHeight := pdImage.Height()

	// create stream
	sampleMax := float32(math.Pow(2, float64(bitsPerComponent))) - 1
	_, isIndexed := colorSpace.(*color.PDIndexed)

	// calculate row padding
	padding := inputWidth * numComponents * bitsPerComponent % 8
	if padding > 0 {
		padding = 8 - padding
	}

	// read stream
	isShort := raster.DataType() == awtimage.TypeUShort
	srcColorValues := make([]int, numComponents)
	for y := 0; y < scanHeight; y++ {
		for x := 0; x < scanWidth; x++ {
			for c := 0; c < numComponents; c++ {
				value := int(iis.readBits(bitsPerComponent))

				// decode array
				dMin := decode[c*2]
				dMax := decode[c*2+1]

				// interpolate to domain
				output := dMin + (float32(value) * ((dMax - dMin) / sampleMax))

				switch {
				case isIndexed:
					// indexed color spaces get the raw value, because the TYPE_BYTE
					// below cannot be reversed by the color space without it having
					// knowledge of the number of bits per component
					srcColorValues[c] = int(byte(javaRound(output)))
				case isShort:
					// interpolate to TYPE_SHORT
					outputShort := javaRound(((output - minFloat(dMin, dMax)) /
						absFloat(dMax-dMin)) * 65535)
					srcColorValues[c] = int(uint16(outputShort))
				default:
					// interpolate to TYPE_BYTE
					outputByte := javaRound(((output - minFloat(dMin, dMax)) /
						absFloat(dMax-dMin)) * 255)
					srcColorValues[c] = int(byte(outputByte))
				}
			}
			raster.SetPixel(x, y, srcColorValues)
		}
		// rows are padded to the nearest byte
		iis.readBits(padding)
	}
	return nil
}

func from1Bit(pdImage PDImage, clipped awtgeom.Rectangle, subsampling, width,
	height int) (goimage.Image, error) {
	currentSubsampling := subsampling
	colorSpace, err := pdImage.ColorSpace()
	if err != nil {
		return nil, err
	}
	decode, err := getDecodeArray(pdImage)
	if err != nil {
		return nil, err
	}

	options := filter.NewDecodeOptionsOfSubsampling(currentSubsampling)
	options.SetSourceRegion(&clipped)

	// read bit stream
	iis, err := pdImage.CreateInputStreamWithOptions(options)
	if err != nil {
		return nil, err
	}

	var inputWidth, startx, starty, scanWidth, scanHeight int
	if options.IsFilterSubsampled() {
		// Decode options were honored, and so there is no need for additional
		// clipping or subsampling
		inputWidth = width
		startx = 0
		starty = 0
		scanWidth = width
		scanHeight = height
		currentSubsampling = 1
	} else {
		// Decode options not honored, so we need to clip and subsample ourselves.
		inputWidth = pdImage.Width()
		startx = clipped.X
		starty = clipped.Y
		scanWidth = clipped.Width
		scanHeight = clipped.Height
	}

	// Java builds a TYPE_BYTE_GRAY image for DeviceGray and a one band raster
	// otherwise, and writes into the raster of whichever; the port writes into a
	// raster throughout and lets the colour space make the picture, which is
	// what the two branches come to.
	raster := awtimage.NewBandedRaster(awtimage.TypeByte, width, height, 1)
	output := raster.Samples()

	idx := 0
	// read stream byte per byte, invert pixel bits if necessary,
	// and then simply shift bits out to the left, detecting set bits via sign
	nosubsampling := currentSubsampling == 1
	stride := (inputWidth + 7) / 8
	invert := int32(0)
	if decode[0] >= decode[1] {
		invert = -1
	}
	endX := startx + scanWidth
	buff := make([]byte, stride)
	for y := 0; y < starty+scanHeight; y++ {
		read, _ := io.ReadFull(iis, buff)
		if y >= starty && y%currentSubsampling == 0 {
			x := startx
			for r := x / 8; r < stride && r < read; r++ {
				// Java holds the row byte in an int, XORs the sign extended byte
				// with the inversion mask and shifts it up so that the bit under
				// test is the sign bit; the port writes the same in int32.
				value := (int32(int8(buff[r])) ^ invert) << uint(24+(x&7))
				for count := min(8-(x&7), endX-x); count > 0; x, count = x+1, count-1 {
					if nosubsampling || x%currentSubsampling == 0 {
						// Java has no bound check here either; the loop counts
						// are what keep idx inside the raster.
						if value < 0 {
							output[idx] = 255
						}
						idx++
					}
					value <<= 1
				}
			}
		}
		if read != stride {
			slog.Warn("image: premature EOF, image will be incomplete")
			break
		}
	}

	if _, isGray := colorSpace.(*color.PDDeviceGray); isGray {
		// TYPE_BYTE_GRAY and not TYPE_BYTE_BINARY because this one is handled
		// without conversion to RGB by Graphics.drawImage
		// this reduces the memory footprint, only one byte per pixel instead of three.
		return colorSpace.ToRawImage(raster)
	}
	// use the color space to convert the image to RGB
	return colorSpace.ToRGBImage(raster)
}

// from8bit is the faster, 8-bit non-decoded, non-colormasked image conversion.
func from8bit(pdImage PDImage, raster *awtimage.Raster, clipped awtgeom.Rectangle,
	subsampling, width, height int) (goimage.Image, error) {
	currentSubsampling := subsampling
	options := filter.NewDecodeOptionsOfSubsampling(currentSubsampling)
	options.SetSourceRegion(&clipped)

	input, err := pdImage.CreateInputStreamWithOptions(options)
	if err != nil {
		return nil, err
	}

	var inputWidth, startx, starty, scanWidth, scanHeight int
	if options.IsFilterSubsampled() {
		inputWidth = width
		startx = 0
		starty = 0
		scanWidth = width
		scanHeight = height
		currentSubsampling = 1
	} else {
		inputWidth = pdImage.Width()
		startx = clipped.X
		starty = clipped.Y
		scanWidth = clipped.Width
		scanHeight = clipped.Height
	}

	colorSpace, err := pdImage.ColorSpace()
	if err != nil {
		return nil, err
	}
	numComponents := colorSpace.NumberOfComponents()

	// get the raster's underlying sample buffer
	bank := raster.Samples()

	if startx == 0 && starty == 0 && scanWidth == width && scanHeight == height &&
		currentSubsampling == 1 {
		// we just need to copy all sample data, then convert to RGB image.
		buf := make([]byte, len(bank))
		inputResult, _ := io.ReadFull(input, buf)
		if inputResult != width*height*numComponents {
			slog.Debug("image: tried reading bytes but fewer were read",
				"want", width*height*numComponents, "got", inputResult)
		}
		for i := 0; i < inputResult; i++ {
			bank[i] = uint16(buf[i])
		}
		return colorSpace.ToRGBImage(raster)
	}

	// either subsampling is required, or reading only part of the image, so its
	// not possible to blindly copy all data.
	tempBytes := make([]byte, numComponents*inputWidth)
	// compromise between memory and time usage:
	// reading the whole image consumes too much memory
	// reading one pixel at a time makes it slow in our buffering infrastructure
	i := 0
	for y := 0; y < starty+scanHeight; y++ {
		inputResult, _ := io.ReadFull(input, tempBytes)
		if inputResult != len(tempBytes) {
			slog.Debug("image: tried reading bytes but fewer were read",
				"want", len(tempBytes), "got", inputResult)
		}
		if y < starty || y%currentSubsampling > 0 {
			continue
		}
		if currentSubsampling == 1 {
			// Not the entire region was requested, but if no subsampling should
			// be performed, we can still copy the entire part of this row
			//
			// JAVA BUG 31: the destination offset is the *source* row times the
			// *source* width, where the raster being filled is the destination
			// region. For any region that is a strict subset this writes to the
			// wrong place and runs off the end of the raster; Java throws
			// ArrayIndexOutOfBoundsException, which getRGBImage does not catch.
			// The port indexes the same way and panics for the same.
			dst := y * inputWidth * numComponents
			for c := 0; c < scanWidth*numComponents; c++ {
				bank[dst+c] = uint16(tempBytes[startx*numComponents+c])
			}
		} else {
			for x := startx; x < startx+scanWidth; x += currentSubsampling {
				for c := 0; c < numComponents; c++ {
					bank[i] = uint16(tempBytes[x*numComponents+c])
					i++
				}
			}
		}
	}

	// use the color space to convert the image to RGB
	return colorSpace.ToRGBImage(raster)
}

// fromAny is the slower, general-purpose image conversion from any image
// format.
func fromAny(pdImage PDImage, raster *awtimage.Raster, colorKey *cos.Array,
	clipped awtgeom.Rectangle, subsampling, width, height int) (goimage.Image, error) {
	currentSubsampling := subsampling
	colorSpace, err := pdImage.ColorSpace()
	if err != nil {
		return nil, err
	}
	numComponents := colorSpace.NumberOfComponents()
	bitsPerComponent := pdImage.BitsPerComponent()
	decode, err := getDecodeArray(pdImage)
	if err != nil {
		return nil, err
	}

	options := filter.NewDecodeOptionsOfSubsampling(currentSubsampling)
	options.SetSourceRegion(&clipped)

	imageStream, err := pdImage.CreateInputStreamWithOptions(options)
	if err != nil {
		return nil, err
	}
	// read bit stream
	iis := newImageBitReader(imageStream)

	var inputWidth, startx, starty, scanWidth, scanHeight int
	if options.IsFilterSubsampled() {
		inputWidth = width
		startx = 0
		starty = 0
		scanWidth = width
		scanHeight = height
		currentSubsampling = 1
	} else {
		inputWidth = pdImage.Width()
		startx = clipped.X
		starty = clipped.Y
		scanWidth = clipped.Width
		scanHeight = clipped.Height
	}

	sampleMax := float32(math.Pow(2, float64(bitsPerComponent))) - 1
	_, isIndexed := colorSpace.(*color.PDIndexed)

	// init color key mask
	var colorKeyRanges []float32
	var colorKeyMask *goimage.Gray
	if colorKey != nil {
		if colorKey.Size() >= numComponents*2 {
			colorKeyRanges = colorKey.ToFloatArray()
			colorKeyMask = goimage.NewGray(goimage.Rect(0, 0, width, height))
		} else {
			slog.Warn("image: colorKey mask has the wrong size, ignored",
				"size", colorKey.Size(), "want", numComponents*2)
		}
	}

	// calculate row padding
	padding := inputWidth * numComponents * bitsPerComponent % 8
	if padding > 0 {
		padding = 8 - padding
	}

	// read stream
	srcColorValues := make([]int, numComponents)
	for y := 0; y < starty+scanHeight; y++ {
		for x := 0; x < startx+scanWidth; x++ {
			isMasked := true
			for c := 0; c < numComponents; c++ {
				value := int(iis.readBits(bitsPerComponent))

				// color key mask requires values before they are decoded
				if colorKeyRanges != nil {
					isMasked = isMasked && float32(value) >= colorKeyRanges[c*2] &&
						float32(value) <= colorKeyRanges[c*2+1]
				}

				// decode array
				dMin := decode[c*2]
				dMax := decode[c*2+1]

				// interpolate to domain
				output := dMin + (float32(value) * ((dMax - dMin) / sampleMax))

				if isIndexed {
					// indexed color spaces get the raw value, because the TYPE_BYTE
					// below cannot be reversed by the color space without it having
					// knowledge of the number of bits per component
					srcColorValues[c] = int(byte(javaRound(output)))
				} else {
					// interpolate to TYPE_BYTE
					outputByte := javaRound(((output - minFloat(dMin, dMax)) /
						absFloat(dMax-dMin)) * 255)
					srcColorValues[c] = int(byte(outputByte))
				}
			}

			// only write to output if within requested region and subsample.
			if x >= startx && y >= starty && x%currentSubsampling == 0 &&
				y%currentSubsampling == 0 {
				destX := (x - startx) / currentSubsampling
				destY := (y - starty) / currentSubsampling
				if destX < width && destY < height {
					raster.SetPixel(destX, destY, srcColorValues)

					// set alpha channel in color key mask, if any
					if colorKeyMask != nil {
						alpha := byte(0)
						if isMasked {
							alpha = 255
						}
						colorKeyMask.SetGray(destX, destY, goimagecolor.Gray{Y: alpha})
					}
				}
			}
		}
		// rows are padded to the nearest byte
		iis.readBits(padding)
	}

	// use the color space to convert the image to RGB
	rgbImage, err := colorSpace.ToRGBImage(raster)
	if err != nil {
		return nil, err
	}

	// apply color mask, if any
	if colorKeyMask != nil {
		return applyColorKeyMask(rgbImage, colorKeyMask), nil
	}
	return rgbImage, nil
}

// applyColorKeyMask composes an RGB image and a binary mask into ARGB.
func applyColorKeyMask(img goimage.Image, mask *goimage.Gray) goimage.Image {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// compose to ARGB
	masked := goimage.NewRGBA(goimage.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b, _ := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			alphaPixel := mask.GrayAt(x, y).Y
			masked.SetRGBA(x, y, goimagecolor.RGBA{
				R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8),
				A: 255 - alphaPixel,
			})
		}
	}
	return masked
}

// getDecodeArray gets the decode array from the dictionary or returns the
// default.
func getDecodeArray(pdImage PDImage) ([]float32, error) {
	colorSpace, err := pdImage.ColorSpace()
	if err != nil {
		return nil, err
	}
	if cosDecode := pdImage.Decode(); cosDecode != nil {
		numberOfComponents := colorSpace.NumberOfComponents()
		if cosDecode.Size() >= numberOfComponents*2 {
			decodeError := false
			decode := make([]float32, numberOfComponents*2)
			for i := range decode {
				number, ok := cosDecode.Get(i).(cos.Number)
				if !ok {
					decodeError = true
					break
				}
				decode[i] = number.FloatValue()
			}
			if !decodeError && pdImage.IsStencil() &&
				(decode[0] < 0 || decode[0] > 1 || decode[1] < 0 || decode[1] > 1) {
				decodeError = true
			}
			if !decodeError {
				return decode, nil
			}
		}
		slog.Error("image: decode array not compatible with color space, using default",
			"decode", fmt.Sprintf("%v", cosDecode))
	}
	// use color space default
	return colorSpace.DefaultDecode(pdImage.BitsPerComponent()), nil
}

// javaRound is java.lang.Math.round(float), which is floor(x + 0.5) and so
// takes a half towards positive infinity, where Go's math.Round takes it away
// from zero.
func javaRound(value float32) int {
	if value != value { // NaN
		return 0
	}
	rounded := math.Floor(float64(value) + 0.5)
	switch {
	case rounded >= math.MaxInt32:
		return math.MaxInt32
	case rounded <= math.MinInt32:
		return math.MinInt32
	}
	return int(rounded)
}

func minFloat(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func absFloat(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
