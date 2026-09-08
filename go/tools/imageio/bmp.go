package imageio

// BMP and WBMP, which Go has no encoder for and `javax.imageio` does.
//
// Neither is a port of anything: Java asks the registry for a writer and the
// JDK supplies one. These are what the registry supplies, written to the two
// formats' own definitions. Both are simple enough that there is nothing to
// decide -- a BMP is a header and rows of pixels bottom-up, a WBMP is a header
// and rows of bits.

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"io"
)

// writeBMP writes a 24-bit uncompressed BMP with its resolution in the header.
//
// 24-bit because that is what every image this port hands it converts to
// without loss of anything a PDF image carries: a BMP has no alpha in this
// form, and neither does what `javax.imageio`'s writer produces for the
// TYPE_INT_RGB images PDFToImage renders.
//
// The resolution is always written, which is what the Java's own test asserts
// and what its writer loop is reaching for -- and is not what the Java does on
// a plain JDK, whose BMP writer answers read-only metadata so that
// `ImageIOUtil` skips `setDPI` and the fields stay zero. See JAVA-BUGS.md 81.
func writeBMP(img image.Image, output io.Writer, dpi int, quality float32) error {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= 0 || height <= 0 {
		return fmt.Errorf("imageio: a BMP cannot be %dx%d", width, height)
	}

	// Every row is padded to a multiple of four bytes.
	stride := (width*3 + 3) &^ 3
	const fileHeader = 14
	const infoHeader = 40
	dataSize := stride * height
	fileSize := fileHeader + infoHeader + dataSize

	header := make([]byte, 0, fileHeader+infoHeader)
	header = append(header, 'B', 'M')
	header = binary.LittleEndian.AppendUint32(header, uint32(fileSize))
	header = binary.LittleEndian.AppendUint32(header, 0) // two reserved shorts
	header = binary.LittleEndian.AppendUint32(header, fileHeader+infoHeader)

	perMetre := uint32(dpiToPixelsPerMetre(dpi))
	header = binary.LittleEndian.AppendUint32(header, infoHeader)
	header = binary.LittleEndian.AppendUint32(header, uint32(width))
	header = binary.LittleEndian.AppendUint32(header, uint32(height))
	header = binary.LittleEndian.AppendUint16(header, 1)  // one plane
	header = binary.LittleEndian.AppendUint16(header, 24) // bits per pixel
	header = binary.LittleEndian.AppendUint32(header, 0)  // no compression
	header = binary.LittleEndian.AppendUint32(header, uint32(dataSize))
	header = binary.LittleEndian.AppendUint32(header, perMetre)
	header = binary.LittleEndian.AppendUint32(header, perMetre)
	header = binary.LittleEndian.AppendUint32(header, 0) // no palette
	header = binary.LittleEndian.AppendUint32(header, 0) // every colour matters
	if _, err := output.Write(header); err != nil {
		return err
	}

	// The rows go bottom-up, which is what the format says and what the Java
	// test's checkBmpResolution assumes about everything after the header.
	row := make([]byte, stride)
	for y := height - 1; y >= 0; y-- {
		for i := range row {
			row[i] = 0
		}
		for x := 0; x < width; x++ {
			r, g, b := rgb8At(img, bounds.Min.X+x, bounds.Min.Y+y)
			row[x*3] = b
			row[x*3+1] = g
			row[x*3+2] = r
		}
		if _, err := output.Write(row); err != nil {
			return err
		}
	}
	return nil
}

// writeWBMP writes a type 0 WBMP, which is one bit per pixel and nothing else.
//
// It carries no resolution, and Java writes none: "no META data possible for
// WBMP, thus no dpi test".
func writeWBMP(img image.Image, output io.Writer, dpi int, quality float32) error {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= 0 || height <= 0 {
		return fmt.Errorf("imageio: a WBMP cannot be %dx%d", width, height)
	}

	header := []byte{0, 0} // type 0, no fixed header fields
	header = appendWBMPLength(header, width)
	header = appendWBMPLength(header, height)
	if _, err := output.Write(header); err != nil {
		return err
	}

	stride := (width + 7) / 8
	row := make([]byte, stride)
	for y := 0; y < height; y++ {
		for i := range row {
			row[i] = 0
		}
		for x := 0; x < width; x++ {
			// A WBMP bit is 1 for white, which is the way round a fax is not.
			if isWhite(img, bounds.Min.X+x, bounds.Min.Y+y) {
				row[x/8] |= 0x80 >> uint(x%8)
			}
		}
		if _, err := output.Write(row); err != nil {
			return err
		}
	}
	return nil
}

// appendWBMPLength writes a length as the format's variable-length integer:
// seven bits at a time, most significant first, with the high bit set on every
// byte but the last.
func appendWBMPLength(out []byte, value int) []byte {
	var digits []byte
	for {
		digits = append([]byte{byte(value & 0x7F)}, digits...)
		value >>= 7
		if value == 0 {
			break
		}
	}
	for i := 0; i < len(digits)-1; i++ {
		digits[i] |= 0x80
	}
	return append(out, digits...)
}

// rgb8At answers a pixel as three eight-bit channels.
func rgb8At(img image.Image, x, y int) (byte, byte, byte) {
	r, g, b, _ := img.At(x, y).RGBA()
	return byte(r >> 8), byte(g >> 8), byte(b >> 8)
}

// isWhite reports whether a pixel is nearer white than black, which is what a
// one-bit format has to decide for every pixel it is given.
func isWhite(img image.Image, x, y int) bool {
	grey := color.GrayModel.Convert(img.At(x, y)).(color.Gray)
	return grey.Y >= 128
}
