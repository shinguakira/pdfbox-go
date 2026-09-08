package imageio

// Putting the resolution into a file that has already been written.
//
// Port of what MetaUtil and JPEGUtil are for. Java edits an `IIOMetadata` tree
// before the writer runs -- `MetaUtil.debugLogMetadata` and
// `JPEGUtil.updateMetadata` walk a DOM of `javax_imageio_1.0` nodes and set
// `HorizontalPixelSize` or the JFIF `Xdensity`. Go's encoders take no metadata
// at all and write none, so the port puts the same numbers in the same places
// afterwards, in the bytes.
//
// It is the same information either way. A PNG says its resolution in a `pHYs`
// chunk, a JPEG in the JFIF APP0 segment, a BMP in two header fields and a TIFF
// in two tags; the last two are written by their own writers here, which build
// the whole file and have nowhere else to put it.

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"math"
)

// metresPerInch is what a resolution has to be converted into for the two
// formats that count in metres.
const metresPerInch = 0.0254

// dpiToPixelsPerMetre is Java's conversion, which `checkBmpResolution` inverts
// as `pixelsPerMeter / 100.0 * 2.54`.
func dpiToPixelsPerMetre(dpi int) int {
	return int(math.Round(float64(dpi) / metresPerInch))
}

// pngWithResolution inserts a pHYs chunk into a written PNG.
//
// The chunk has to come before the first IDAT, which the specification requires
// and readers enforce.
func pngWithResolution(content []byte, dpi int) ([]byte, error) {
	const signature = 8
	if len(content) < signature {
		return nil, fmt.Errorf("imageio: the PNG is %d bytes", len(content))
	}

	perMetre := uint32(dpiToPixelsPerMetre(dpi))
	payload := make([]byte, 9)
	binary.BigEndian.PutUint32(payload[0:4], perMetre)
	binary.BigEndian.PutUint32(payload[4:8], perMetre)
	payload[8] = 1 // the unit is the metre

	chunk := make([]byte, 0, 12+len(payload))
	chunk = binary.BigEndian.AppendUint32(chunk, uint32(len(payload)))
	chunk = append(chunk, "pHYs"...)
	chunk = append(chunk, payload...)
	chunk = binary.BigEndian.AppendUint32(chunk,
		crc32.ChecksumIEEE(chunk[4:]))

	at := signature
	for at+8 <= len(content) {
		length := int(binary.BigEndian.Uint32(content[at : at+4]))
		kind := string(content[at+4 : at+8])
		if kind == "IDAT" {
			out := make([]byte, 0, len(content)+len(chunk))
			out = append(out, content[:at]...)
			out = append(out, chunk...)
			return append(out, content[at:]...), nil
		}
		next := at + 8 + length + 4
		if next <= at || next > len(content) {
			break
		}
		at = next
	}
	return nil, fmt.Errorf("imageio: the PNG carries no IDAT chunk")
}

// jpegWithResolution puts the resolution into a written JPEG's JFIF segment,
// adding the segment where there is none.
//
// Port of JPEGUtil.updateMetadata, which sets `resUnits` to 1 and `Xdensity`
// and `Ydensity` to the dpi on the `app0JFIF` node of the metadata tree. Java
// has a node to edit because `javax.imageio`'s writer puts a JFIF segment in
// every file; **Go's `image/jpeg` writes none at all** -- it goes straight from
// the SOI to the quantisation tables -- so there is nothing to edit and the
// segment is written here. It is 18 bytes and it is what makes a JPEG say what
// size it is meant to be printed at.
func jpegWithResolution(content []byte, dpi int) ([]byte, error) {
	if len(content) < 4 || content[0] != 0xFF || content[1] != 0xD8 {
		return nil, fmt.Errorf("imageio: the JPEG does not start with an SOI")
	}
	out := make([]byte, len(content))
	copy(out, content)

	at := 2
	for at+4 <= len(out) {
		if out[at] != 0xFF {
			break
		}
		marker := out[at+1]
		length := int(binary.BigEndian.Uint16(out[at+2 : at+4]))
		if length < 2 || at+2+length > len(out) {
			break
		}
		segment := out[at+4 : at+2+length]
		if marker == 0xE0 && len(segment) >= 12 && string(segment[0:5]) == "JFIF\x00" {
			density := uint16(dpi)
			segment[7] = 1 // the unit is the inch
			binary.BigEndian.PutUint16(segment[8:10], density)
			binary.BigEndian.PutUint16(segment[10:12], density)
			return out, nil
		}
		if marker == 0xDA { // start of scan: the segments are over
			break
		}
		at += 2 + length
	}

	// No JFIF segment, which is every JPEG `image/jpeg` writes. It goes
	// immediately after the SOI, which is where the specification puts it and
	// where a reader looks for it.
	density := uint16(dpi)
	segment := make([]byte, 0, 20)
	segment = append(segment, 0xFF, 0xE0)
	segment = binary.BigEndian.AppendUint16(segment, 16) // the length, less the marker
	segment = append(segment, 'J', 'F', 'I', 'F', 0)
	// JFIF 1.02, which is JPEGUtil's majorVersion 1 and minorVersion 2.
	segment = append(segment, 1, 2)
	segment = append(segment, 1) // the unit is the inch, which is resUnits 1
	segment = binary.BigEndian.AppendUint16(segment, density)
	segment = binary.BigEndian.AppendUint16(segment, density)
	segment = append(segment, 0, 0) // no thumbnail

	out = make([]byte, 0, len(content)+len(segment))
	out = append(out, content[:2]...)
	out = append(out, segment...)
	return append(out, content[2:]...), nil
}
