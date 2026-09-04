package image

import (
	"bytes"
	"errors"
	"fmt"
	goimage "image"
	"os"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/filter"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
)

// The factory that builds an image XObject from a black and white picture or
// from the CCITT data inside a TIFF file.
//
// Port of org.apache.pdfbox.pdmodel.graphics.image.CCITTFactory.

// CreateCCITTFromImage creates an image XObject from a one bit black and white
// picture.
//
// Port of CCITTFactory.createFromImage. Java throws IllegalArgumentException
// for anything but a one bit image, which is unchecked, so the port panics;
// what it tests is the BufferedImage type and the colour model's pixel size,
// which Go has no equivalent of, so the port takes any image and reads the low
// bit of each pixel as the Java does.
func CreateCCITTFromImage(document DocumentLike, img goimage.Image) (*PDImageXObject, error) {
	bounds := img.Bounds()
	height := bounds.Dy()
	width := bounds.Dx()

	rowBytes := (width + 7) / 8
	data := make([]byte, rowBytes*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, _, _, _ := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			// flip bit to avoid having to set /BlackIs1
			bit := ^(r >> 8) & 1
			if bit != 0 {
				data[y*rowBytes+x/8] |= 1 << uint(7-x%8)
			}
		}
		// Java pads the row to a byte boundary with zeroes, which the buffer
		// above already is.
	}
	return prepareCCITTImageXObject(document, data, width, height, color.DeviceGray)
}

// CreateCCITTFromByteArray creates an image XObject from the first CCITT image
// in a TIFF file held in memory.
//
// Port of CCITTFactory.createFromByteArray(PDDocument, byte[]).
func CreateCCITTFromByteArray(document DocumentLike, byteArray []byte) (*PDImageXObject, error) {
	return CreateCCITTFromByteArrayNumbered(document, byteArray, 0)
}

// CreateCCITTFromByteArrayNumbered creates an image XObject from the numbered
// CCITT image in a TIFF file held in memory.
//
// Port of CCITTFactory.createFromByteArray(PDDocument, byte[], int).
func CreateCCITTFromByteArrayNumbered(document DocumentLike, byteArray []byte,
	number int) (*PDImageXObject, error) {
	return createCCITTFromRandomAccess(document, byteArray, number)
}

// CreateCCITTFromFile creates an image XObject from the first CCITT image in a
// TIFF file.
//
// Port of CCITTFactory.createFromFile(PDDocument, File).
func CreateCCITTFromFile(document DocumentLike, path string) (*PDImageXObject, error) {
	return CreateCCITTFromFileNumbered(document, path, 0)
}

// CreateCCITTFromFileNumbered creates an image XObject from the numbered CCITT
// image in a TIFF file.
//
// Port of CCITTFactory.createFromFile(PDDocument, File, int). Java reads the
// file through a RandomAccessRead; the port reads it in, because the TIFF tags
// send it back and forth over the whole file anyway.
func CreateCCITTFromFileNumbered(document DocumentLike, path string,
	number int) (*PDImageXObject, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return createCCITTFromRandomAccess(document, data, number)
}

// prepareCCITTImageXObject encodes the bitmap as Group 4 fax and wraps it.
//
// Port of the private CCITTFactory.prepareImageXObject.
func prepareCCITTImageXObject(document DocumentLike, byteArray []byte, width, height int,
	initColorSpace color.PDColorSpace) (*PDImageXObject, error) {
	var encoded bytes.Buffer
	dict := cos.NewDictionary()
	dict.SetInt(cos.Columns, width)
	dict.SetInt(cos.Rows, height)
	if err := (filter.CCITTFax{}).Encode(&encoded, bytes.NewReader(byteArray), dict); err != nil {
		return nil, err
	}

	img, err := newPDImageXObjectOfEncoded(document, encoded.Bytes(), cos.CCITTFaxDecode,
		width, height, 1, initColorSpace)
	if err != nil {
		return nil, err
	}
	dict.SetInt(cos.K, -1)
	img.COSDictionary().SetItem(cos.DecodeParms, dict)
	return img, nil
}

func createCCITTFromRandomAccess(document DocumentLike, data []byte,
	number int) (*PDImageXObject, error) {
	decodeParms := cos.NewDictionary()
	var bos bytes.Buffer
	if err := extractFromTiff(data, &bos, decodeParms, number); err != nil {
		return nil, err
	}
	if bos.Len() == 0 {
		return nil, nil
	}

	pdImage, err := newPDImageXObjectOfEncoded(document, bos.Bytes(), cos.CCITTFaxDecode,
		decodeParms.GetInt(cos.Columns), decodeParms.GetInt(cos.Rows), 1, color.DeviceGray)
	if err != nil {
		return nil, err
	}
	pdImage.COSDictionary().SetItem(cos.DecodeParms, decodeParms)
	return pdImage, nil
}

// tiffReader reads the TIFF the way Java's RandomAccessRead does: read past the
// end gives -1 rather than an error.
type tiffReader struct {
	data []byte
	pos  int
}

func (r *tiffReader) seek(at int64) { r.pos = int(at) }

func (r *tiffReader) read() int {
	if r.pos < 0 || r.pos >= len(r.data) {
		r.pos++
		return -1
	}
	b := int(r.data[r.pos])
	r.pos++
	return b
}

// extractFromTiff extracts the CCITT stream from the TIFF file.
func extractFromTiff(data []byte, os *bytes.Buffer, params *cos.Dictionary, number int) error {
	reader := &tiffReader{data: data}

	// First check the basic tiff header
	reader.seek(0)
	endianess := reader.read()
	if reader.read() != endianess {
		return errors.New("Not a valid tiff file")
	}
	// ensure that endianess is either M or I
	if endianess != 'M' && endianess != 'I' {
		return errors.New("Not a valid tiff file")
	}
	magicNumber := readshort(endianess, reader)
	if magicNumber != 42 {
		return errors.New("Not a valid tiff file")
		// 43 is bigtiff.
	}

	// Relocate to the first set of tags
	address := readlong(endianess, reader)
	reader.seek(address)

	// If some higher page number is required, skip this page's tags,
	// then read the next page's address
	for i := 0; i < number; i++ {
		numtags := readshort(endianess, reader)
		if numtags > 50 {
			return errors.New("Not a valid tiff file")
		}
		reader.seek(address + 2 + int64(numtags)*12)
		address = readlong(endianess, reader)
		if address == 0 {
			return nil
		}
		reader.seek(address)
	}

	numtags := readshort(endianess, reader)

	// The number 50 is somewhat arbitrary, it just stops us load up junk from
	// somewhere and tramping on
	if numtags > 50 {
		return errors.New("Not a valid tiff file")
	}

	// Loop through the tags, some will convert to items in the params dictionary
	// Other point us to where to find the data stream.
	// The only param which might change as a result of other TIFF tags is K, so
	// we'll deal with that differently.

	// Default value to detect error
	k := -1000
	dataoffset := 0
	datalength := 0
	fillorder := 1

	for i := 0; i < numtags; i++ {
		tag := readshort(endianess, reader)
		kind := readshort(endianess, reader)
		count := int(readlong(endianess, reader))
		var val int

		// Note that when the type is shorter than 4 bytes, the rest can be garbage
		// and must be ignored. E.g. short (2 bytes) from "01 00 38 32" (little endian)
		// is 1, not 842530817 (seen in a real-life TIFF image).
		switch kind {
		case 1: // byte value
			val = reader.read()
			reader.read()
			reader.read()
			reader.read()
		case 3: // short value
			val = readshort(endianess, reader)
			reader.read()
			reader.read()
		default: // long and other types
			val = int(readlong(endianess, reader))
		}

		switch tag {
		case 256:
			params.SetInt(cos.Columns, val)
		case 257:
			params.SetInt(cos.Rows, val)
		case 259:
			if val == 4 {
				k = -1
			}
			if val == 3 {
				k = 0
			}
			// T6/T4 Compression
		case 262:
			if val == 1 {
				params.SetBoolean(cos.BlackIs1, true)
			}
		case 266:
			// http://www.awaresystems.be/imaging/tiff/tifftags/fillorder.html
			if val != 1 && val != 2 {
				return fmt.Errorf("FillOrder %d is not supported", val)
			}
			fillorder = val
		case 273:
			if count == 1 {
				dataoffset = val
			}
		case 274:
			// http://www.awaresystems.be/imaging/tiff/tifftags/orientation.html
			if val != 1 {
				return fmt.Errorf("Orientation %d is not supported", val)
			}
		case 279:
			if count == 1 {
				datalength = val
			}
		case 292:
			if val&1 != 0 {
				// T4 2D - arbitrary positive K value
				k = 50
			}
			// http://www.awaresystems.be/imaging/tiff/tifftags/t4options.html
			if val&2 != 0 {
				return errors.New("CCITT Group 3 'uncompressed mode' is not supported")
			}
			if val&4 != 0 {
				// sample file in PDFBOX-934
				// Note that this TIFF option is not the same as the PDF EncodedByteAlign option
				return errors.New("CCITT Group 3 'fill bits before EOL' is not supported")
			}
		case 324:
			if count == 1 {
				dataoffset = val
			}
		case 325:
			if count == 1 {
				datalength = val
			}
		default:
			// do nothing
		}
	}

	if k == -1000 {
		return errors.New("First image in tiff is not CCITT T4 or T6 compressed")
	}
	if dataoffset == 0 {
		return errors.New("First image in tiff is not a single tile/strip")
	}

	params.SetInt(cos.K, k)

	reader.seek(int64(dataoffset))
	buf := make([]byte, 8192)
	for datalength > 0 {
		want := min(8192, datalength)
		amountRead := 0
		for amountRead < want {
			b := reader.read()
			if b == -1 {
				break
			}
			buf[amountRead] = byte(b)
			amountRead++
		}
		if amountRead == 0 {
			break
		}
		datalength -= amountRead
		if fillorder == 2 {
			for x := 0; x < amountRead; x++ {
				buf[x] = fliptable[buf[x]]
			}
		}
		os.Write(buf[:amountRead])
	}
	return nil
}

func readshort(endianess int, raf *tiffReader) int {
	if endianess == 'I' {
		return raf.read() | (raf.read() << 8)
	}
	return (raf.read() << 8) | raf.read()
}

func readlong(endianess int, raf *tiffReader) int64 {
	// TIFF LONG is an unsigned 32-bit value; mask so it widens correctly
	//
	// Java evaluates the four read() calls left to right and combines them in
	// an int before the mask widens it; the port writes the reads out in
	// separate statements, because Go does not fix the order of evaluation of
	// operands the way Java does and the four are not interchangeable.
	if endianess == 'I' {
		b0 := raf.read()
		b1 := raf.read()
		b2 := raf.read()
		b3 := raf.read()
		return int64(uint32(int32(b0 | (b1 << 8) | (b2 << 16) | (b3 << 24))))
	}
	b0 := raf.read()
	b1 := raf.read()
	b2 := raf.read()
	b3 := raf.read()
	return int64(uint32(int32((b0 << 24) | (b1 << 16) | (b2 << 8) | b3)))
}

// fliptable reverses the bits of a byte, for a TIFF whose fill order is 2.
var fliptable = [256]byte{
	0x00, 0x80, 0x40, 0xc0, 0x20, 0xa0, 0x60, 0xe0,
	0x10, 0x90, 0x50, 0xd0, 0x30, 0xb0, 0x70, 0xf0,
	0x08, 0x88, 0x48, 0xc8, 0x28, 0xa8, 0x68, 0xe8,
	0x18, 0x98, 0x58, 0xd8, 0x38, 0xb8, 0x78, 0xf8,
	0x04, 0x84, 0x44, 0xc4, 0x24, 0xa4, 0x64, 0xe4,
	0x14, 0x94, 0x54, 0xd4, 0x34, 0xb4, 0x74, 0xf4,
	0x0c, 0x8c, 0x4c, 0xcc, 0x2c, 0xac, 0x6c, 0xec,
	0x1c, 0x9c, 0x5c, 0xdc, 0x3c, 0xbc, 0x7c, 0xfc,
	0x02, 0x82, 0x42, 0xc2, 0x22, 0xa2, 0x62, 0xe2,
	0x12, 0x92, 0x52, 0xd2, 0x32, 0xb2, 0x72, 0xf2,
	0x0a, 0x8a, 0x4a, 0xca, 0x2a, 0xaa, 0x6a, 0xea,
	0x1a, 0x9a, 0x5a, 0xda, 0x3a, 0xba, 0x7a, 0xfa,
	0x06, 0x86, 0x46, 0xc6, 0x26, 0xa6, 0x66, 0xe6,
	0x16, 0x96, 0x56, 0xd6, 0x36, 0xb6, 0x76, 0xf6,
	0x0e, 0x8e, 0x4e, 0xce, 0x2e, 0xae, 0x6e, 0xee,
	0x1e, 0x9e, 0x5e, 0xde, 0x3e, 0xbe, 0x7e, 0xfe,
	0x01, 0x81, 0x41, 0xc1, 0x21, 0xa1, 0x61, 0xe1,
	0x11, 0x91, 0x51, 0xd1, 0x31, 0xb1, 0x71, 0xf1,
	0x09, 0x89, 0x49, 0xc9, 0x29, 0xa9, 0x69, 0xe9,
	0x19, 0x99, 0x59, 0xd9, 0x39, 0xb9, 0x79, 0xf9,
	0x05, 0x85, 0x45, 0xc5, 0x25, 0xa5, 0x65, 0xe5,
	0x15, 0x95, 0x55, 0xd5, 0x35, 0xb5, 0x75, 0xf5,
	0x0d, 0x8d, 0x4d, 0xcd, 0x2d, 0xad, 0x6d, 0xed,
	0x1d, 0x9d, 0x5d, 0xdd, 0x3d, 0xbd, 0x7d, 0xfd,
	0x03, 0x83, 0x43, 0xc3, 0x23, 0xa3, 0x63, 0xe3,
	0x13, 0x93, 0x53, 0xd3, 0x33, 0xb3, 0x73, 0xf3,
	0x0b, 0x8b, 0x4b, 0xcb, 0x2b, 0xab, 0x6b, 0xeb,
	0x1b, 0x9b, 0x5b, 0xdb, 0x3b, 0xbb, 0x7b, 0xfb,
	0x07, 0x87, 0x47, 0xc7, 0x27, 0xa7, 0x67, 0xe7,
	0x17, 0x97, 0x57, 0xd7, 0x37, 0xb7, 0x77, 0xf7,
	0x0f, 0x8f, 0x4f, 0xcf, 0x2f, 0xaf, 0x6f, 0xef,
	0x1f, 0x9f, 0x5f, 0xdf, 0x3f, 0xbf, 0x7f, 0xff,
}
