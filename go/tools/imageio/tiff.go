package imageio

// TIFF, which Go has no encoder for and `javax.imageio` does.
//
// **Not a port, and not compressed.** `TIFFUtil.setCompressionType` picks
// CCITT T.6 for a one-bit bitonal image and LZW for everything else, and
// `TIFFUtil.updateMetadata` writes the resolution into the metadata tree.
// Neither compression is in Go's standard library and one of them nearly is:
// `compress/lzw` implements the LZW of GIF and PDF, and TIFF's variant
// increments the code width one code early, so its output is a file some TIFF
// readers accept and others refuse. That is worse than not compressing.
//
// So this writes baseline uncompressed strips, which every reader takes, and
// migration/STATUS.md records the difference. Everything `TIFFUtil` decides is
// kept: the resolution, the RowsPerStrip of the whole image, the Software tag
// reading "PDFBOX", and the WhiteIsZero photometric interpretation a bitonal
// image gets "because of bug in Windows XP preview". The depth is kept too --
// a bitonal image is written at one bit per pixel, which is what
// `ExtractImages` converts an image to before asking for a TIFF.
//
// The tag values were read off the files the Java writes for the same images.
// One thing is deliberately not the same: Java's writer puts them in
// big-endian order and this writes little-endian, which the format allows and
// says in its first two bytes.

import (
	"encoding/binary"
	"fmt"
	"image"
	"io"
)

// The TIFF tags this writes, from the baseline set of the specification.
//
// The last five are the ones `TIFFUtil.updateMetadata` adds by hand.
const (
	tagImageWidth      = 0x0100
	tagImageLength     = 0x0101
	tagBitsPerSample   = 0x0102
	tagCompression     = 0x0103
	tagPhotometric     = 0x0106
	tagStripOffsets    = 0x0111
	tagSamplesPerPixel = 0x0115
	tagRowsPerStrip    = 0x0116
	tagStripByteCounts = 0x0117
	tagXResolution     = 0x011A
	tagYResolution     = 0x011B
	tagResolutionUnit  = 0x0128
	tagSoftware        = 0x0131
)

// The field types this writes.
const (
	typeASCII    = 2
	typeShort    = 3
	typeLong     = 4
	typeRational = 5
)

// compressionNone is the Compression tag's value for a file that is not
// compressed. Java writes 5 for LZW or 4 for CCITT T.6; see the head of this
// file.
const compressionNone = 1

// The PhotometricInterpretation values this writes.
//
// A bitonal image gets WhiteIsZero, which is what `TIFFUtil.updateMetadata`
// sets tag 262 to for a one-bit image and nothing else.
const (
	photometricWhiteIsZero = 0
	photometricBlackIsZero = 1
	photometricRGB         = 2
	photometricSeparated   = 5
)

// tiffSoftware is `createAsciiField(305, "Software", "PDFBOX")`.
const tiffSoftware = "PDFBOX"

// writeTIFF writes a baseline uncompressed TIFF with its resolution in the
// XResolution and YResolution tags.
func writeTIFF(img image.Image, output io.Writer, dpi int, quality float32) error {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= 0 || height <= 0 {
		return fmt.Errorf("imageio: a TIFF cannot be %dx%d", width, height)
	}

	bits, samples, photometric, pixels := tiffSamples(img)

	// header, then the directory, then the values too big for a directory
	// entry, then the pixels.
	const header = 8
	const entrySize = 12
	const entries = 13
	directory := 2 + entries*entrySize + 4
	valuesAt := header + directory

	// The values that live after the directory, in the order they are written.
	var values []byte
	// The two resolutions, each a numerator over a denominator of one.
	xResolutionAt := valuesAt + len(values)
	values = binary.LittleEndian.AppendUint32(values, uint32(dpi))
	values = binary.LittleEndian.AppendUint32(values, 1)
	yResolutionAt := valuesAt + len(values)
	values = binary.LittleEndian.AppendUint32(values, uint32(dpi))
	values = binary.LittleEndian.AppendUint32(values, 1)

	// BitsPerSample carries one short per sample, so it fits in the entry only
	// where there is one channel.
	bitsPerSampleValue := uint32(bits)
	if samples > 1 {
		bitsPerSampleValue = uint32(valuesAt + len(values))
		for i := 0; i < samples; i++ {
			values = binary.LittleEndian.AppendUint16(values, uint16(bits))
		}
	}

	softwareAt := valuesAt + len(values)
	values = append(values, tiffSoftware...)
	values = append(values, 0) // an ASCII field is terminated
	if len(values)%2 != 0 {
		values = append(values, 0) // a value starts on an even boundary
	}

	stripAt := valuesAt + len(values)

	out := make([]byte, 0, stripAt+len(pixels))
	out = append(out, 'I', 'I') // little-endian, which is what "II" means
	out = binary.LittleEndian.AppendUint16(out, 42)
	out = binary.LittleEndian.AppendUint32(out, header)

	out = binary.LittleEndian.AppendUint16(out, entries)
	// The entries have to be in ascending order of tag, which readers rely on.
	out = appendTIFFEntry(out, tagImageWidth, typeLong, 1, uint32(width))
	out = appendTIFFEntry(out, tagImageLength, typeLong, 1, uint32(height))
	out = appendTIFFEntry(out, tagBitsPerSample, typeShort, uint32(samples),
		bitsPerSampleValue)
	out = appendTIFFEntry(out, tagCompression, typeShort, 1, compressionNone)
	out = appendTIFFEntry(out, tagPhotometric, typeShort, 1, uint32(photometric))
	out = appendTIFFEntry(out, tagStripOffsets, typeLong, 1, uint32(stripAt))
	out = appendTIFFEntry(out, tagSamplesPerPixel, typeShort, 1, uint32(samples))
	out = appendTIFFEntry(out, tagRowsPerStrip, typeLong, 1, uint32(height))
	out = appendTIFFEntry(out, tagStripByteCounts, typeLong, 1, uint32(len(pixels)))
	out = appendTIFFEntry(out, tagXResolution, typeRational, 1, uint32(xResolutionAt))
	out = appendTIFFEntry(out, tagYResolution, typeRational, 1, uint32(yResolutionAt))
	out = appendTIFFEntry(out, tagResolutionUnit, typeShort, 1, 2) // the inch
	out = appendTIFFEntry(out, tagSoftware, typeASCII, uint32(len(tiffSoftware)+1),
		uint32(softwareAt))
	out = binary.LittleEndian.AppendUint32(out, 0) // no next directory

	if len(out) != valuesAt {
		return fmt.Errorf("imageio: the TIFF values are at %d and the directory "+
			"is %d long", len(out), valuesAt)
	}
	out = append(out, values...)
	out = append(out, pixels...)
	_, err := output.Write(out)
	return err
}

// appendTIFFEntry writes one directory entry, whose value goes in the last four
// bytes where it fits and is an offset where it does not.
//
// A single SHORT that fits is written into the first two of those four, which
// is what the specification says and what a reader looks at.
func appendTIFFEntry(out []byte, tag, fieldType uint16, count, value uint32) []byte {
	out = binary.LittleEndian.AppendUint16(out, tag)
	out = binary.LittleEndian.AppendUint16(out, fieldType)
	out = binary.LittleEndian.AppendUint32(out, count)
	if fieldType == typeShort && count == 1 {
		out = binary.LittleEndian.AppendUint16(out, uint16(value))
		return binary.LittleEndian.AppendUint16(out, 0)
	}
	return binary.LittleEndian.AppendUint32(out, value)
}

// tiffSamples answers the depth, the number of channels, the photometric
// interpretation and the pixels of an image, in the shape a TIFF strip wants.
//
// A one-bit image stays one bit: `ExtractImages` converts a bitonal page to
// one bit per pixel on purpose, and a writer that widened it back to eight
// would undo the only part of that conversion this port can keep.
func tiffSamples(img image.Image) (bits, samples, photometric int, pixels []byte) {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	if bitonal, isGray := img.(*image.Gray); isGray && isBitonal(bitonal) {
		stride := (width + 7) / 8
		pixels = make([]byte, stride*height)
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				// WhiteIsZero, which is what TIFFUtil sets tag 262 to for a
				// bitonal image: a set bit is black.
				if !isWhite(img, bounds.Min.X+x, bounds.Min.Y+y) {
					pixels[y*stride+x/8] |= 0x80 >> uint(x%8)
				}
			}
		}
		return 1, 1, photometricWhiteIsZero, pixels
	}

	if grey, isGray := img.(*image.Gray); isGray {
		pixels = make([]byte, width*height)
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				pixels[y*width+x] = grey.GrayAt(bounds.Min.X+x, bounds.Min.Y+y).Y
			}
		}
		return 8, 1, photometricBlackIsZero, pixels
	}

	if cmyk, isCMYK := img.(*image.CMYK); isCMYK {
		// Four channels, kept as four. This is the raster `ExtractImages`
		// chooses a TIFF for -- "More than 3 channels: That's likely CMYK. We
		// use tiff here" -- and converting it to RGB here would throw away the
		// separation the option `-noColorConvert` was given to keep. Java's
		// writer keeps it: measured, a four band image comes out with
		// BitsPerSample 8,8,8,8, SamplesPerPixel 4 and
		// PhotometricInterpretation 5.
		pixels = make([]byte, width*height*4)
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				from := cmyk.PixOffset(bounds.Min.X+x, bounds.Min.Y+y)
				copy(pixels[(y*width+x)*4:], cmyk.Pix[from:from+4])
			}
		}
		return 8, 4, photometricSeparated, pixels
	}

	pixels = make([]byte, width*height*3)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b := rgb8At(img, bounds.Min.X+x, bounds.Min.Y+y)
			at := (y*width + x) * 3
			pixels[at], pixels[at+1], pixels[at+2] = r, g, b
		}
	}
	return 8, 3, photometricRGB, pixels
}

// isBitonal reports whether a grey image uses only black and white, which is
// what makes it worth writing at one bit per pixel.
//
// It walks the rows rather than `Pix`, because `Pix` is not the image: a
// `SubImage` shares its parent's buffer and stride and its slice runs to the
// end of that buffer, so reading it straight through sees pixels outside the
// bounds -- and one grey pixel out there would send a bitonal image out at
// eight bits.
func isBitonal(img *image.Gray) bool {
	bounds := img.Bounds()
	width := bounds.Dx()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		row := img.Pix[img.PixOffset(bounds.Min.X, y):][:width]
		for _, value := range row {
			if value != 0 && value != 0xFF {
				return false
			}
		}
	}
	return true
}
