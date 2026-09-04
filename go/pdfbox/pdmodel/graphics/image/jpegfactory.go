package image

import (
	"bytes"
	"fmt"
	goimage "image"
	goimagecolor "image/color"
	"image/jpeg"
	"io"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
)

// The factory that builds an image XObject from JPEG data.
//
// Port of org.apache.pdfbox.pdmodel.graphics.image.JPEGFactory, a final class of
// static methods, which the port keeps as functions.
//
// DocumentLike is what these take in place of a PDDocument: the factories need
// only a stream to write into, and pdmodel is above this package.

// DocumentLike is the part of PDDocument an image factory asks for.
//
// Java takes a PDDocument and calls document.getDocument().createCOSStream(),
// which is how the stream gets the document's stream cache. pdmodel imports
// this package, so the port names what it needs instead -- the same shape slice
// 5 used to break the cycle between the security handlers and PDDocument.
type DocumentLike interface {
	// CreateStream returns a new stream belonging to the document.
	CreateStream() *cos.Stream
}

// CreateFromJPEGStream creates an image XObject from a JPEG stream, without
// decoding it.
//
// Port of JPEGFactory.createFromStream.
func CreateFromJPEGStream(document DocumentLike, stream io.Reader) (*PDImageXObject, error) {
	data, err := io.ReadAll(stream)
	if err != nil {
		return nil, err
	}

	cosStream := document.CreateStream()
	output, err := cosStream.CreateRawWriter()
	if err != nil {
		return nil, err
	}
	if _, err := output.Write(data); err != nil {
		output.Close()
		return nil, err
	}
	if err := output.Close(); err != nil {
		return nil, err
	}

	meta, err := retrieveDimensions(data)
	if err != nil {
		return nil, err
	}

	var colorSpace color.PDColorSpace
	switch meta.numComponents {
	case 1:
		colorSpace = color.DeviceGray
	case 3:
		colorSpace = color.DeviceRGB
	case 4:
		colorSpace = color.DeviceCMYK
	default:
		// Java throws UnsupportedOperationException, which is unchecked.
		panic(fmt.Sprintf("number of data elements not supported: %d", meta.numComponents))
	}

	// create PDImageXObject around the already-populated stream, no further copying
	cosStream.SetItem(cos.Filter, cos.DCTDecode)
	pdImage := NewPDImageXObject(common.NewPDStream(cosStream), nil)
	pdImage.SetBitsPerComponent(8)
	pdImage.SetWidth(meta.width)
	pdImage.SetHeight(meta.height)
	pdImage.SetColorSpace(colorSpace)
	if _, isCMYK := colorSpace.(*color.PDDeviceCMYK); isCMYK {
		pdImage.SetDecode(invertedDecodeArray(4))
	}
	return pdImage, nil
}

// CreateFromJPEGByteArray creates an image XObject from JPEG bytes.
//
// Port of JPEGFactory.createFromByteArray.
func CreateFromJPEGByteArray(document DocumentLike, byteArray []byte) (*PDImageXObject, error) {
	return CreateFromJPEGStream(document, bytes.NewReader(byteArray))
}

// invertedDecodeArray builds the [1 0 1 0 ...] decode array an Adobe CMYK JPEG
// needs, which inverts every component.
func invertedDecodeArray(components int) *cos.Array {
	decode := cos.NewArray()
	for i := 0; i < components; i++ {
		decode.Add(cos.IntegerOne)
		decode.Add(cos.IntegerZero)
	}
	return decode
}

// jpegDimensions is Java's private Dimensions class.
type jpegDimensions struct {
	width         int
	height        int
	numComponents int
}

// retrieveDimensions reads the size and component count out of a JPEG.
//
// Java asks an ImageReader, and falls back to decoding the raster where the
// metadata will not parse. The port reads the start of frame marker directly,
// which is where the reader gets all three from and is what PDFBOX-4691 changed
// the Java to prefer, and so has no fallback to decode.
func retrieveDimensions(data []byte) (jpegDimensions, error) {
	width, height, numComponents, err := jpegFrameHeader(data)
	if err != nil {
		return jpegDimensions{}, err
	}
	return jpegDimensions{width: width, height: height, numComponents: numComponents}, nil
}

// CreateJPEGFromImage encodes an image as JPEG and wraps it in an image
// XObject.
//
// Port of JPEGFactory.createFromImage(PDDocument, BufferedImage, float, int).
// Two things Java does that the port cannot:
//
//   - The DPI. Java edits the JFIF APP0 marker of the encoded stream through
//     the writer's metadata tree; Go's image/jpeg writes a JFIF marker with a
//     density of 1 aspect ratio and gives no way to change it, so the dpi
//     argument is accepted and ignored. Nothing in a PDF reads it -- the image
//     is scaled by the content stream -- and PDFBOX-6235 notes that a CMYK JPEG
//     has no JFIF marker to carry it either.
//   - The bytes. Two JPEG encoders do not agree, so the stream this writes is
//     not the stream Java writes for the same image and quality.
//
// See migration/STATUS.md.
func CreateJPEGFromImage(document DocumentLike, img goimage.Image, quality float32,
	dpi int) (*PDImageXObject, error) {
	awtColorImage, alphaImage := splitAlpha(img)

	// Go's image/jpeg has no four component encoder: it writes every image that
	// is not grey as a three component YCbCr JPEG. Java writes a CMYK one, and
	// PDDeviceCMYK with the inverted decode array below is what reads it back;
	// declaring that over three component data would have a reader take three
	// samples per pixel as four. So the port converts a CMYK image to RGB here
	// and declares what it actually wrote. See migration/STATUS.md.
	awtColorImage = toRGBForJPEG(awtColorImage)

	// create XObject
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, awtColorImage,
		&jpeg.Options{Quality: int(quality*100 + 0.5)}); err != nil {
		return nil, err
	}

	colorSpace := colorSpaceOfImage(awtColorImage)
	pdImage, err := newPDImageXObjectOfEncoded(document, encoded.Bytes(), cos.DCTDecode,
		awtColorImage.Bounds().Dx(), awtColorImage.Bounds().Dy(), 8, colorSpace)
	if err != nil {
		return nil, err
	}
	if _, isCMYK := colorSpace.(*color.PDDeviceCMYK); isCMYK {
		pdImage.SetDecode(invertedDecodeArray(4))
	}

	// extract alpha channel (if any)
	if alphaImage != nil {
		// alpha -> soft mask
		xAlpha, err := CreateJPEGFromImage(document, alphaImage, quality, dpi)
		if err != nil {
			return nil, err
		}
		pdImage.COSDictionary().SetItem(cos.SMask, xAlpha.COSObject())
	}
	return pdImage, nil
}

// newPDImageXObjectOfEncoded is Java's PDImageXObject(PDDocument, InputStream,
// COSBase, int, int, int, PDColorSpace) constructor, which stores already
// encoded data and stamps the filter that decodes it.
func newPDImageXObjectOfEncoded(document DocumentLike, encoded []byte, cosFilter *cos.Name,
	width, height, bitsPerComponent int, initColorSpace color.PDColorSpace) (*PDImageXObject, error) {
	cosStream := document.CreateStream()
	output, err := cosStream.CreateRawWriter()
	if err != nil {
		return nil, err
	}
	if _, err := output.Write(encoded); err != nil {
		output.Close()
		return nil, err
	}
	if err := output.Close(); err != nil {
		return nil, err
	}

	pdImage := NewPDImageXObject(common.NewPDStream(cosStream), nil)
	pdImage.COSDictionary().SetItem(cos.Filter, cosFilter)
	pdImage.SetBitsPerComponent(bitsPerComponent)
	pdImage.SetWidth(width)
	pdImage.SetHeight(height)
	pdImage.SetColorSpace(initColorSpace)
	return pdImage, nil
}

// colorSpaceOfImage returns the PDF colour space for a Go image.
//
// Port of getColorSpaceFromAWT, whose ICC and unknown cases throw
// UnsupportedOperationException; Go's image types carry no colour space, so the
// port decides on the type and has no ICC case to refuse.
func colorSpaceOfImage(img goimage.Image) color.PDColorSpace {
	switch img.(type) {
	case *goimage.Gray, *goimage.Gray16:
		// 256 color (gray) JPEG
		return color.DeviceGray
	case *goimage.CMYK:
		return color.DeviceCMYK
	}
	return color.DeviceRGB
}

// splitAlpha returns the colour channels of an image and its alpha channel.
//
// Port of getColorImage and getAlphaImage together: Java calls them one after
// the other and the port walks the pixels once.
func splitAlpha(img goimage.Image) (goimage.Image, goimage.Image) {
	bounds := img.Bounds()
	switch img.(type) {
	case *goimage.Gray, *goimage.Gray16, *goimage.CMYK, *goimage.YCbCr:
		// no alpha channel
		return img, nil
	}

	rgb := goimage.NewRGBA(goimage.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	alpha := goimage.NewGray(goimage.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	hasAlpha := false
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			r, g, b, a := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			// Java converts the colour channels without the alpha, which for a
			// premultiplied source means dividing it back out; Go's RGBA() is
			// premultiplied too, so the port does the same division.
			if a > 0 && a < 0xffff {
				r = r * 0xffff / a
				g = g * 0xffff / a
				b = b * 0xffff / a
			}
			rgb.Set(x, y, colorFromRGBA(r, g, b))
			alpha.Pix[alpha.PixOffset(x, y)] = uint8(a >> 8)
			if a>>8 != 0xff {
				hasAlpha = true
			}
		}
	}
	if !hasAlpha {
		// happens sometimes (PDFBOX-2654) despite colormodel claiming to have alpha
		return rgb, nil
	}
	return rgb, alpha
}

// colorFromRGBA builds an opaque colour from three 16 bit channels.
func colorFromRGBA(r, g, b uint32) goimagecolor.RGBA {
	return goimagecolor.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: 255}
}

// jpegFrameHeader reads the width, height and component count out of a JPEG's
// start of frame marker.
//
// The same walk as jpegNumChannels in pdfbox/filter, which the image code
// cannot reach: that one is unexported there and the filter package must not
// depend on this one.
func jpegFrameHeader(data []byte) (width, height, components int, err error) {
	in := bytes.NewReader(data)
	var marker [2]byte
	if _, err := io.ReadFull(in, marker[:]); err != nil || marker[0] != 0xFF || marker[1] != 0xD8 {
		return 0, 0, 0, fmt.Errorf("image: not a JPEG stream")
	}
	for {
		b, err := in.ReadByte()
		if err != nil {
			return 0, 0, 0, fmt.Errorf("image: no JPEG start of frame")
		}
		if b != 0xFF {
			continue
		}
		// a marker may be preceded by any number of fill bytes
		for b == 0xFF {
			if b, err = in.ReadByte(); err != nil {
				return 0, 0, 0, fmt.Errorf("image: no JPEG start of frame")
			}
		}
		switch {
		case b == 0x01 || (b >= 0xD0 && b <= 0xD9):
			// standalone markers, no segment follows
			continue
		case b >= 0xC0 && b <= 0xCF && b != 0xC4 && b != 0xC8 && b != 0xCC:
			// a start of frame: length, precision, height, width, components
			var head [8]byte
			if _, err := io.ReadFull(in, head[:]); err != nil {
				return 0, 0, 0, fmt.Errorf("image: truncated JPEG start of frame")
			}
			height = int(head[3])<<8 | int(head[4])
			width = int(head[5])<<8 | int(head[6])
			return width, height, int(head[7]), nil
		default:
			var length [2]byte
			if _, err := io.ReadFull(in, length[:]); err != nil {
				return 0, 0, 0, fmt.Errorf("image: truncated JPEG segment")
			}
			size := int(length[0])<<8 | int(length[1])
			if size < 2 {
				return 0, 0, 0, fmt.Errorf("image: bad JPEG segment length %d", size)
			}
			if _, err := in.Seek(int64(size-2), io.SeekCurrent); err != nil {
				return 0, 0, 0, err
			}
		}
	}
}

// toRGBForJPEG converts an image image/jpeg cannot encode in its own colour
// model into one it can.
//
// The only such image is *image.CMYK: the encoder handles *image.Gray as one
// component and everything else as three, so a CMYK image would be written as
// three component YCbCr while the dictionary said four. Converting it here
// keeps the data and the dictionary in agreement, at the cost of the CMYK
// colour space Java would have written.
func toRGBForJPEG(img goimage.Image) goimage.Image {
	if _, isCMYK := img.(*goimage.CMYK); !isCMYK {
		return img
	}
	bounds := img.Bounds()
	rgb := goimage.NewRGBA(goimage.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			r, g, b, _ := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			rgb.SetRGBA(x, y, colorFromRGBA(r, g, b))
		}
	}
	return rgb
}
