package filter

import (
	"bufio"
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// DCT is the DCTDecode filter, JPEG.
//
// Port of org.apache.pdfbox.filter.DCTFilter. What the filter writes is the
// raster -- the interleaved component samples -- and not a picture: one band
// for grey, three for RGB, four for CMYK. The colour space of the image
// XObject says how to read them.
//
// Java gets the raster out of the JPEG reader the JRE ships. Go decodes with
// image/jpeg, and the two differ in three ways this port has to state rather
// than hide:
//
//   - image/jpeg hands back an image, not a raster, and it has already applied
//     the Adobe inversion that a CMYK JPEG stores its samples with. Java writes
//     the samples as stored and lets the /Decode array of the image invert
//     them. The port takes image/jpeg's inversion back out, so what it writes
//     is what Java writes; that step is exact, because it is one subtraction.
//   - Two JPEG decoders do not agree to the last bit. The inverse DCT and the
//     YCbCr to RGB conversion are approximations, and image/jpeg's differ from
//     the JRE's in the last place on some samples. A DCT port cannot be byte
//     identical to Java's and this one is not.
//   - image/jpeg refuses a four component JPEG with no Adobe APP14 marker
//     ("unknown color model"); Java's getAdobeTransformByBruteForce falls back
//     to reading it as CMYK. That file decodes in Java and does not here.
//
// See migration/STATUS.md.
type DCT struct{}

var _ Filter = DCT{}

// Decode writes the raster of the JPEG.
func (DCT) Decode(w io.Writer, r io.Reader, parameters *cos.Dictionary,
	index int) (DecodeResult, error) {
	return DCT{}.DecodeWithOptions(w, r, parameters, index, DefaultDecodeOptions)
}

// DecodeWithOptions is Java's decode overload taking DecodeOptions.
//
// Java pushes the subsampling and the source region down into the JPEG reader,
// which honours both, and then sets filterSubsampled. image/jpeg decodes the
// whole image and has no such parameters, so the port leaves filterSubsampled
// alone: the caller sees that the filter did not subsample and does it itself,
// which is the path Java takes for every filter that ignores the options.
func (DCT) DecodeWithOptions(w io.Writer, r io.Reader, parameters *cos.Dictionary,
	index int, options *DecodeOptions) (DecodeResult, error) {
	result := DecodeResult{Parameters: parameters}

	encoded := bufio.NewReader(r)
	// skip one LF if there
	if first, err := encoded.ReadByte(); err == nil && first != 0x0A {
		if err := encoded.UnreadByte(); err != nil {
			return result, err
		}
	}

	img, err := jpeg.Decode(encoded)
	if err != nil {
		return result, fmt.Errorf("filter: could not read JPEG image: %w", err)
	}

	raster, err := rasterOf(img)
	if err != nil {
		return result, err
	}
	if _, err := w.Write(raster); err != nil {
		return result, err
	}
	return result, nil
}

// rasterOf turns a decoded JPEG into the interleaved samples Java's
// DataBufferByte holds.
func rasterOf(img image.Image) ([]byte, error) {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	switch m := img.(type) {
	case *image.Gray:
		// one band, already the samples
		raster := make([]byte, width*height)
		for y := 0; y < height; y++ {
			copy(raster[y*width:(y+1)*width],
				m.Pix[y*m.Stride:y*m.Stride+width])
		}
		return raster, nil

	case *image.CMYK:
		// Four bands. image/jpeg stores 255-sample for every channel, of both
		// a plain CMYK and a YCCK file -- see applyBlack in its reader -- and
		// Java writes the sample. So the whole conversion is one subtraction.
		//
		// For YCCK that also puts the port on Java's own arithmetic: Java's
		// fromYCCKtoCMYK computes cyan = 255 - (Y + 1.402*(Cr-128)) and the two
		// after it, which is 255 minus exactly what image/jpeg's YCbCrToRGB
		// computes for the same inputs, in fixed point rather than float.
		raster := make([]byte, width*height*4)
		for y := 0; y < height; y++ {
			row := m.Pix[y*m.Stride : y*m.Stride+width*4]
			out := raster[y*width*4 : (y+1)*width*4]
			for i, v := range row {
				out[i] = 255 - v
			}
		}
		return raster, nil

	case *image.YCbCr:
		// Three bands. Java reads a three channel JPEG as a BufferedImage,
		// whose raster is BGR, and then swaps the outer two bands to get RGB;
		// the port converts to RGB directly, which is the same order.
		raster := make([]byte, width*height*3)
		i := 0
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				red, green, blue := color.YCbCrToRGB(
					m.Y[m.YOffset(x, y)],
					m.Cb[m.COffset(x, y)],
					m.Cr[m.COffset(x, y)])
				raster[i] = red
				raster[i+1] = green
				raster[i+2] = blue
				i += 3
			}
		}
		return raster, nil

	default:
		return nil, fmt.Errorf("filter: unexpected JPEG image type %T", img)
	}
}

// Encode panics, where Java throws UnsupportedOperationException.
func (DCT) Encode(w io.Writer, r io.Reader, parameters *cos.Dictionary) error {
	panic("DCTFilter encoding not implemented, use the JPEGFactory methods instead")
}

// jpegNumChannels reports how many components the JPEG declares, by reading its
// start of frame marker.
//
// Java asks the image reader's metadata for NumChannels and falls back to an
// empty string, which decides between read() and readRaster(); the port has no
// such choice to make -- image/jpeg picks the image type itself -- so this
// exists for the image code above the filter, which needs the count before it
// decodes.
func jpegNumChannels(data []byte) (int, error) {
	in := bytes.NewReader(data)
	var marker [2]byte
	if _, err := io.ReadFull(in, marker[:]); err != nil || marker[0] != 0xFF || marker[1] != 0xD8 {
		return 0, fmt.Errorf("filter: not a JPEG stream")
	}
	for {
		b, err := in.ReadByte()
		if err != nil {
			return 0, fmt.Errorf("filter: no JPEG start of frame")
		}
		if b != 0xFF {
			continue
		}
		// a marker may be preceded by any number of fill bytes
		for b == 0xFF {
			if b, err = in.ReadByte(); err != nil {
				return 0, fmt.Errorf("filter: no JPEG start of frame")
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
				return 0, fmt.Errorf("filter: truncated JPEG start of frame")
			}
			return int(head[7]), nil
		default:
			var length [2]byte
			if _, err := io.ReadFull(in, length[:]); err != nil {
				return 0, fmt.Errorf("filter: truncated JPEG segment")
			}
			size := int(length[0])<<8 | int(length[1])
			if size < 2 {
				return 0, fmt.Errorf("filter: bad JPEG segment length %d", size)
			}
			if _, err := in.Seek(int64(size-2), io.SeekCurrent); err != nil {
				return 0, err
			}
		}
	}
}
