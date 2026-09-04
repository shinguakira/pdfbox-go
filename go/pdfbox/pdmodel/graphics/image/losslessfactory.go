package image

import (
	"bytes"
	goimage "image"
	goimagecolor "image/color"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/filter"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
)

// The factory that builds an image XObject from a picture, without losing
// anything.
//
// Port of org.apache.pdfbox.pdmodel.graphics.image.LosslessFactory.
//
// The one place the port departs in shape is the predictor encoder. Java
// switches on BufferedImage.getType and on the raster's transfer type, and
// returns null for a type it does not know, which sends createFromImage down
// its fallback; Go's image package has no such type codes, so the port switches
// on the Go image type and answers the same question -- how many components,
// how many bytes each, and whether there is alpha. What it computes for a row,
// and how it chooses between the five PNG predictors, is the Java's.

// UsePredictorEncoder is Java's static final LosslessFactory.USE_PREDICTOR_ENCODER.
const UsePredictorEncoder = true

// CreateFromImage creates an image XObject from a picture, losslessly.
//
// Port of LosslessFactory.createFromImage.
func CreateFromImage(document DocumentLike, img goimage.Image) (*PDImageXObject, error) {
	if isGrayImage(img) {
		return createFromGrayImage(img, document)
	}

	// We try to encode the image with predictor
	if UsePredictorEncoder {
		pdImageXObject, err := newPredictorEncoder(document, img).encode()
		if err != nil {
			return nil, err
		}
		if pdImageXObject != nil {
			colorSpace, err := pdImageXObject.ColorSpace()
			if err != nil {
				return nil, err
			}
			bounds := img.Bounds()
			if colorSpace == color.PDColorSpace(color.DeviceRGB) &&
				pdImageXObject.BitsPerComponent() < 16 &&
				bounds.Dx()*bounds.Dy() <= 50*50 {
				// also create classic compressed image, compare sizes
				pdImageXObjectClassic, err := createFromRGBImage(img, document)
				if err != nil {
					return nil, err
				}
				classicLength, _ := pdImageXObjectClassic.Stream().Length()
				predictorLength, _ := pdImageXObject.Stream().Length()
				if classicLength < predictorLength {
					return pdImageXObjectClassic, nil
				}
			}
			return pdImageXObject, nil
		}
	}

	// Fallback: We export the image as 8-bit sRGB and might lose color information
	return createFromRGBImage(img, document)
}

// isGrayImage reports whether the image is opaque grey of at most 8 bits.
//
// Java asks the BufferedImage for TYPE_BYTE_GRAY or TYPE_BYTE_BINARY and its
// pixel size; Go's *image.Gray is the first and has no equivalent of the second,
// so a 1 bit picture reaches the port as an 8 bit grey one and is written as
// such. That costs bytes in the output, not information.
func isGrayImage(img goimage.Image) bool {
	if _, ok := img.(*goimage.Gray); ok {
		return true
	}
	return false
}

// createFromGrayImage writes one sample per pixel.
//
// Grayscale images need one color per sample.
func createFromGrayImage(img goimage.Image, document DocumentLike) (*PDImageXObject, error) {
	bounds := img.Bounds()
	height := bounds.Dy()
	width := bounds.Dx()
	const bpc = 8

	data := make([]byte, 0, width*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, _, _, _ := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			data = append(data, byte(r>>8))
		}
		// Java pads each row to a byte boundary; at 8 bits per component there
		// is never anything to pad.
	}
	return prepareImageXObject(document, data, width, height, bpc, color.DeviceGray)
}

// createFromRGBImage writes three samples per pixel, and the alpha channel as a
// soft mask.
func createFromRGBImage(img goimage.Image, document DocumentLike) (*PDImageXObject, error) {
	bounds := img.Bounds()
	height := bounds.Dy()
	width := bounds.Dx()
	const bpc = 8

	// Java asks the image for its Transparency, one of OPAQUE, BITMASK and
	// TRANSLUCENT, and writes one bit per pixel for a bitmask and one byte
	// otherwise. Go's image types do not distinguish a bitmask from a
	// translucent alpha, so the port writes a byte whenever there is any alpha
	// at all -- which is what Java does for TRANSLUCENT, and which a bitmask
	// image also decodes to correctly, at eight times the mask size.
	const apbc = 8

	imageData := make([]byte, 0, width*height*3)
	alphaImageData := make([]byte, 0, width*height)
	transparent := false
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b, a := unpremultiply(img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA())
			imageData = append(imageData, byte(r>>8), byte(g>>8), byte(b>>8))
			// we have the alpha right here, so no need to do it separately
			// as done prior April 2018
			alphaImageData = append(alphaImageData, byte(a>>8))
			if a>>8 != 0xFF {
				transparent = true
			}
		}
	}

	pdImage, err := prepareImageXObject(document, imageData, width, height, bpc, color.DeviceRGB)
	if err != nil {
		return nil, err
	}
	if transparent {
		pdMask, err := prepareImageXObject(document, alphaImageData, width, height,
			apbc, color.DeviceGray)
		if err != nil {
			return nil, err
		}
		pdImage.COSDictionary().SetItem(cos.SMask, pdMask.COSObject())
	}
	return pdImage, nil
}

// unpremultiply divides the alpha back out of Go's premultiplied channels,
// because a PDF holds the colour and the soft mask apart.
func unpremultiply(r, g, b, a uint32) (uint32, uint32, uint32, uint32) {
	if a == 0 || a == 0xffff {
		return r, g, b, a
	}
	return r * 0xffff / a, g * 0xffff / a, b * 0xffff / a, a
}

// prepareImageXObject deflates the samples and wraps them in an image XObject.
//
// Port of the package-private LosslessFactory.prepareImageXObject.
func prepareImageXObject(document DocumentLike, byteArray []byte, width, height,
	bitsPerComponent int, initColorSpace color.PDColorSpace) (*PDImageXObject, error) {
	var encoded bytes.Buffer
	if err := (filter.Flate{}).Encode(&encoded, bytes.NewReader(byteArray),
		cos.NewDictionary()); err != nil {
		return nil, err
	}
	return newPDImageXObjectOfEncoded(document, encoded.Bytes(), cos.FlateDecode,
		width, height, bitsPerComponent, initColorSpace)
}

// predictorEncoder writes the samples through the five PNG predictors and keeps
// the smallest row of each.
//
// Port of the inner class LosslessFactory.PredictorEncoder.
type predictorEncoder struct {
	document DocumentLike
	img      goimage.Image

	// The raw count of components per pixel including optional alpha
	componentsPerPixel int
	bytesPerComponent  int
	// Only the bytes we need in the output (excluding alpha)
	bytesPerPixel int
	height        int
	width         int

	hasAlpha       bool
	alphaImageData []byte

	dataRawRowNone    []byte
	dataRawRowSub     []byte
	dataRawRowUp      []byte
	dataRawRowAverage []byte
	dataRawRowPaeth   []byte

	// c | b
	// -----
	// a | x
	//
	// x => current pixel
	aValues []byte
	cValues []byte
	bValues []byte
	xValues []byte

	colorSpace color.PDColorSpace
	colorComps int
}

func newPredictorEncoder(document DocumentLike, img goimage.Image) *predictorEncoder {
	bounds := img.Bounds()
	e := &predictorEncoder{
		document:          document,
		img:               img,
		bytesPerComponent: 1,
		height:            bounds.Dy(),
		width:             bounds.Dx(),
	}

	switch img.(type) {
	case *goimage.Gray:
		e.colorSpace, e.colorComps, e.hasAlpha = color.DeviceGray, 1, false
	case *goimage.Gray16:
		e.colorSpace, e.colorComps, e.hasAlpha = color.DeviceGray, 1, false
		e.bytesPerComponent = 2
	case *goimage.CMYK:
		e.colorSpace, e.colorComps, e.hasAlpha = color.DeviceCMYK, 4, false
	case *goimage.RGBA, *goimage.NRGBA:
		e.colorSpace, e.colorComps, e.hasAlpha = color.DeviceRGB, 3, true
	case *goimage.RGBA64, *goimage.NRGBA64:
		e.colorSpace, e.colorComps, e.hasAlpha = color.DeviceRGB, 3, true
		e.bytesPerComponent = 2
	default:
		// We can not handle this unknown format
		return e
	}

	e.componentsPerPixel = e.colorComps
	if e.hasAlpha {
		e.componentsPerPixel++
		e.alphaImageData = make([]byte, e.width*e.height*e.bytesPerComponent)
	}
	e.bytesPerPixel = e.colorComps * e.bytesPerComponent

	// The rows have 1-byte encoding marker and width * BYTES_PER_PIXEL pixel-bytes
	dataRowByteCount := e.width*e.bytesPerPixel + 1
	e.dataRawRowNone = make([]byte, dataRowByteCount)
	e.dataRawRowSub = make([]byte, dataRowByteCount)
	e.dataRawRowUp = make([]byte, dataRowByteCount)
	e.dataRawRowAverage = make([]byte, dataRowByteCount)
	e.dataRawRowPaeth = make([]byte, dataRowByteCount)

	// Write the encoding markers
	e.dataRawRowNone[0] = 0
	e.dataRawRowSub[0] = 1
	e.dataRawRowUp[0] = 2
	e.dataRawRowAverage[0] = 3
	e.dataRawRowPaeth[0] = 4

	e.aValues = make([]byte, e.bytesPerPixel)
	e.cValues = make([]byte, e.bytesPerPixel)
	e.bValues = make([]byte, e.bytesPerPixel)
	e.xValues = make([]byte, e.bytesPerPixel)
	return e
}

// encode returns the image, or nil where the format is one the encoder does not
// handle -- which is Java returning null and sending createFromImage down its
// fallback.
func (e *predictorEncoder) encode() (*PDImageXObject, error) {
	if e.colorSpace == nil {
		return nil, nil
	}

	bounds := e.img.Bounds()
	prevRow := make([]byte, e.width*e.bytesPerPixel)
	transferRow := make([]byte, e.width*e.bytesPerPixel)

	var stream bytes.Buffer
	alphaPtr := 0
	rows := make([]byte, 0, e.height*(e.width*e.bytesPerPixel+1))

	for rowNum := 0; rowNum < e.height; rowNum++ {
		// read the row into the transfer buffer, and the alpha into its own
		e.readRow(bounds, rowNum, transferRow, &alphaPtr)

		// We start to write at index one, as the predictor marker is in index zero
		writerPtr := 1
		clearBytes(e.aValues)
		clearBytes(e.cValues)

		for pixel := 0; pixel < e.width; pixel++ {
			base := pixel * e.bytesPerPixel
			copy(e.xValues, transferRow[base:base+e.bytesPerPixel])
			copy(e.bValues, prevRow[base:base+e.bytesPerPixel])

			// Encode the pixel values in the different encodings
			for bytePtr := 0; bytePtr < len(e.xValues); bytePtr++ {
				x := int(e.xValues[bytePtr])
				a := int(e.aValues[bytePtr])
				b := int(e.bValues[bytePtr])
				c := int(e.cValues[bytePtr])
				e.dataRawRowNone[writerPtr] = byte(x)
				e.dataRawRowSub[writerPtr] = pngFilterSub(x, a)
				e.dataRawRowUp[writerPtr] = pngFilterUp(x, b)
				e.dataRawRowAverage[writerPtr] = pngFilterAverage(x, a, b)
				e.dataRawRowPaeth[writerPtr] = pngFilterPaeth(x, a, b, c)
				writerPtr++
			}

			//  We shift the values into the prev / upper left values for the next pixel
			copy(e.aValues, e.xValues)
			copy(e.cValues, e.bValues)
		}

		rows = append(rows, e.chooseDataRowToWrite()...)

		// We swap prev and transfer row, so that we have the prev row for the next row.
		prevRow, transferRow = transferRow, prevRow
	}

	// Java writes each row into a DeflaterOutputStream as it goes, at the
	// compression level Filter.getCompressionLevel gives; the port deflates the
	// rows in one go through the flate filter, which reads the same setting.
	if err := (filter.Flate{}).Encode(&stream, bytes.NewReader(rows),
		cos.NewDictionary()); err != nil {
		return nil, err
	}

	return e.preparePredictorPDImage(stream.Bytes(), e.bytesPerComponent*8)
}

// readRow copies one row of the image into the transfer buffer, most
// significant byte first for a 16 bit image, and the alpha into its own buffer.
//
// Java reads the raster's data elements, which are already in the image's own
// layout; Go's images do not share one, so the port asks each pixel for its
// channels and writes them in the order the colour space expects.
func (e *predictorEncoder) readRow(bounds goimage.Rectangle, rowNum int, transferRow []byte,
	alphaPtr *int) {
	for x := 0; x < e.width; x++ {
		at := e.img.At(bounds.Min.X+x, bounds.Min.Y+rowNum)
		base := x * e.bytesPerPixel

		if e.colorSpace == color.PDColorSpace(color.DeviceCMYK) {
			// image/color.CMYK carries its four channels as fields and has no
			// accessor for them; the model conversion is the way to ask any
			// colour for them, and for an *image.CMYK it is the identity.
			cmyk := goimagecolor.CMYKModel.Convert(at).(goimagecolor.CMYK)
			transferRow[base] = cmyk.C
			transferRow[base+1] = cmyk.M
			transferRow[base+2] = cmyk.Y
			transferRow[base+3] = cmyk.K
			continue
		}

		r, g, b, a := unpremultiply(at.RGBA())
		switch {
		case e.colorSpace == color.PDColorSpace(color.DeviceGray):
			e.putComponent(transferRow, base, r)
		default:
			e.putComponent(transferRow, base, r)
			e.putComponent(transferRow, base+e.bytesPerComponent, g)
			e.putComponent(transferRow, base+2*e.bytesPerComponent, b)
		}

		if e.hasAlpha {
			if e.bytesPerComponent == 2 {
				e.alphaImageData[*alphaPtr] = byte(a >> 8)
				e.alphaImageData[*alphaPtr+1] = byte(a)
			} else {
				e.alphaImageData[*alphaPtr] = byte(a >> 8)
			}
			*alphaPtr += e.bytesPerComponent
		}
	}
}

func (e *predictorEncoder) putComponent(row []byte, at int, value uint32) {
	if e.bytesPerComponent == 2 {
		row[at] = byte(value >> 8)
		row[at+1] = byte(value)
		return
	}
	row[at] = byte(value >> 8)
}

func clearBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

func (e *predictorEncoder) preparePredictorPDImage(encoded []byte,
	bitsPerComponent int) (*PDImageXObject, error) {
	// Java encodes the ICC profile of the source image where it has one that is
	// not sRGB; Go's image types carry no profile, so there is none to encode.
	imageXObject, err := newPDImageXObjectOfEncoded(e.document, encoded, cos.FlateDecode,
		e.width, e.height, bitsPerComponent, e.colorSpace)
	if err != nil {
		return nil, err
	}

	decodeParms := cos.NewDictionary()
	decodeParms.SetItem(cos.BitsPerComponent, cos.GetInteger(int64(bitsPerComponent)))
	decodeParms.SetItem(cos.Predictor, cos.GetInteger(15))
	decodeParms.SetItem(cos.Columns, cos.GetInteger(int64(e.width)))
	decodeParms.SetItem(cos.Colors, cos.GetInteger(int64(e.colorComps)))
	imageXObject.COSDictionary().SetItem(cos.DecodeParms, decodeParms)

	if e.hasAlpha && !isOpaque(e.alphaImageData) {
		pdMask, err := prepareImageXObject(e.document, e.alphaImageData, e.width, e.height,
			8*e.bytesPerComponent, color.DeviceGray)
		if err != nil {
			return nil, err
		}
		imageXObject.COSDictionary().SetItem(cos.SMask, pdMask.COSObject())
	}
	return imageXObject, nil
}

// isOpaque reports whether every alpha sample is full, which is Java asking the
// image for Transparency.OPAQUE.
func isOpaque(alpha []byte) bool {
	for _, a := range alpha {
		if a != 0xFF {
			return false
		}
	}
	return true
}

func (e *predictorEncoder) chooseDataRowToWrite() []byte {
	rowToWrite := e.dataRawRowNone
	best := estCompressSum(e.dataRawRowNone)
	estCompressSumSub := estCompressSum(e.dataRawRowSub)
	estCompressSumUp := estCompressSum(e.dataRawRowUp)
	estCompressSumAvg := estCompressSum(e.dataRawRowAverage)
	estCompressSumPaeth := estCompressSum(e.dataRawRowPaeth)
	if best > estCompressSumSub {
		rowToWrite = e.dataRawRowSub
		best = estCompressSumSub
	}
	if best > estCompressSumUp {
		rowToWrite = e.dataRawRowUp
		best = estCompressSumUp
	}
	if best > estCompressSumAvg {
		rowToWrite = e.dataRawRowAverage
		best = estCompressSumAvg
	}
	if best > estCompressSumPaeth {
		rowToWrite = e.dataRawRowPaeth
	}
	return rowToWrite
}

func pngFilterSub(x, a int) byte { return byte((x & 0xFF) - (a & 0xFF)) }

// pngFilterUp is the same as pngFilterSub, just called with the prior row.
func pngFilterUp(x, b int) byte { return pngFilterSub(x, b) }

func pngFilterAverage(x, a, b int) byte { return byte(x - ((b + a) / 2)) }

func pngFilterPaeth(x, a, b, c int) byte {
	p := a + b - c
	pa := absInt(p - a)
	pb := absInt(p - b)
	pc := absInt(p - c)
	var pr int
	switch {
	case pa <= pb && pa <= pc:
		pr = a
	case pb <= pc:
		pr = b
	default:
		pr = c
	}
	return byte(x - pr)
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// estCompressSum estimates how well a row will compress.
//
// https://www.w3.org/TR/PNG-Encoders.html#E.Filter-selection
//
// Java sums Math.abs of each byte, and a Java byte is signed: the sum is of the
// values -128 to 127, not 0 to 255. The port keeps the signed reading, because
// which of the five rows wins depends on it.
func estCompressSum(row []byte) int64 {
	var sum int64
	for _, b := range row {
		v := int64(int8(b))
		if v < 0 {
			v = -v
		}
		sum += v
	}
	return sum
}
