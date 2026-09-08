package imageio_test

// Reading back what the writers wrote, which is what the Java test does with
// `ImageIO.getImageReadersBySuffix` and an `IIOMetadata` tree.
//
// Go has readers for PNG, JPEG and GIF and none for BMP or TIFF, and none of
// the three answers a resolution -- `image/png` drops the `pHYs` chunk on the
// floor. So the resolution is read out of the bytes here, which is what the
// Java's own `checkBmpResolution` does for the one format its readers could not
// answer for either.

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"testing"
)

// resolutionOf answers the horizontal and vertical resolution a written file
// carries, in dots per inch.
func resolutionOf(t *testing.T, format string, content []byte) (int, int) {
	t.Helper()
	switch format {
	case "png":
		return pngResolution(t, content)
	case "jpg", "jpeg":
		return jpegResolution(t, content)
	case "bmp":
		return bmpResolution(t, content)
	case "tiff", "tif":
		return tiffResolution(t, content)
	}
	t.Fatalf("no resolution reader for %s", format)
	return 0, 0
}

// pngResolution reads the pHYs chunk, which holds pixels per metre.
func pngResolution(t *testing.T, content []byte) (int, int) {
	t.Helper()
	chunk := pngChunk(t, content, "pHYs")
	if len(chunk) < 9 {
		t.Fatalf("the pHYs chunk is %d bytes, want 9", len(chunk))
	}
	if chunk[8] != 1 {
		t.Fatalf("the pHYs unit is %d, want 1 (the metre)", chunk[8])
	}
	x := binary.BigEndian.Uint32(chunk[0:4])
	y := binary.BigEndian.Uint32(chunk[4:8])
	return perMetreToDPI(int(x)), perMetreToDPI(int(y))
}

// pngChunk answers the payload of the named chunk.
func pngChunk(t *testing.T, content []byte, name string) []byte {
	t.Helper()
	const signature = 8
	at := signature
	for at+8 <= len(content) {
		length := int(binary.BigEndian.Uint32(content[at : at+4]))
		kind := string(content[at+4 : at+8])
		start := at + 8
		if start+length > len(content) {
			break
		}
		if kind == name {
			return content[start : start+length]
		}
		at = start + length + 4 // the payload and its CRC
	}
	t.Fatalf("the PNG carries no %s chunk", name)
	return nil
}

// jpegResolution reads the JFIF APP0 segment's density fields.
func jpegResolution(t *testing.T, content []byte) (int, int) {
	t.Helper()
	// SOI, then APP0 with the JFIF identifier.
	if len(content) < 4 || content[0] != 0xFF || content[1] != 0xD8 {
		t.Fatal("the file does not start with a JPEG SOI")
	}
	at := 2
	for at+4 <= len(content) {
		if content[at] != 0xFF {
			break
		}
		marker := content[at+1]
		length := int(binary.BigEndian.Uint16(content[at+2 : at+4]))
		segment := content[at+4 : at+2+length]
		if marker == 0xE0 && len(segment) >= 12 &&
			string(segment[0:5]) == "JFIF\x00" {
			units := segment[7]
			if units != 1 {
				t.Fatalf("the JFIF density unit is %d, want 1 (the inch)", units)
			}
			x := binary.BigEndian.Uint16(segment[8:10])
			y := binary.BigEndian.Uint16(segment[10:12])
			return int(x), int(y)
		}
		at += 2 + length
	}
	t.Fatal("the JPEG carries no JFIF APP0 segment")
	return 0, 0
}

// bmpResolution is the Java's checkBmpResolution: skip 38 bytes and read two
// little-endian integers of pixels per metre.
func bmpResolution(t *testing.T, content []byte) (int, int) {
	t.Helper()
	if len(content) < 46 {
		t.Fatalf("the BMP is %d bytes, too short to hold a resolution", len(content))
	}
	x := int(int32(binary.LittleEndian.Uint32(content[38:42])))
	y := int(int32(binary.LittleEndian.Uint32(content[42:46])))
	return perMetreToDPI(x), perMetreToDPI(y)
}

// perMetreToDPI is the Java's `Math.round(pixelsPerMeter / 100.0 * 2.54)`.
func perMetreToDPI(perMetre int) int {
	return int(float64(perMetre)/100.0*2.54 + 0.5)
}

// tiffResolution reads the XResolution and YResolution tags, which are
// rationals, and the ResolutionUnit that says what they are per.
func tiffResolution(t *testing.T, content []byte) (int, int) {
	t.Helper()
	x := tiffRational(t, content, 0x011A)
	y := tiffRational(t, content, 0x011B)
	if unit := tiffShort(t, content, 0x0128); unit != 2 {
		t.Fatalf("the TIFF resolution unit is %d, want 2 (the inch)", unit)
	}
	return int(x + 0.5), int(y + 0.5)
}

// tiffEntries walks the first IFD, answering the tag, type, count and value
// field of each entry.
func tiffEntries(t *testing.T, content []byte) map[uint16][]byte {
	t.Helper()
	if len(content) < 8 {
		t.Fatal("the TIFF is too short to hold a header")
	}
	var order binary.ByteOrder
	switch string(content[0:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		t.Fatalf("the TIFF byte order mark is %q", content[0:2])
	}
	if got := order.Uint16(content[2:4]); got != 42 {
		t.Fatalf("the TIFF magic is %d, want 42", got)
	}
	offset := int(order.Uint32(content[4:8]))
	if offset+2 > len(content) {
		t.Fatal("the TIFF directory is past the end of the file")
	}
	count := int(order.Uint16(content[offset : offset+2]))
	entries := map[uint16][]byte{}
	for i := 0; i < count; i++ {
		at := offset + 2 + i*12
		if at+12 > len(content) {
			t.Fatal("a TIFF directory entry is past the end of the file")
		}
		entries[order.Uint16(content[at:at+2])] = content[at : at+12]
	}
	tiffOrder = order
	return entries
}

// tiffOrder is the byte order of the file tiffEntries last read, which the two
// readers below need.
var tiffOrder binary.ByteOrder

// tiffShort answers a SHORT tag's value.
func tiffShort(t *testing.T, content []byte, tag uint16) int {
	t.Helper()
	entry, found := tiffEntries(t, content)[tag]
	if !found {
		t.Fatalf("the TIFF carries no tag %#x", tag)
	}
	return int(tiffOrder.Uint16(entry[8:10]))
}

// tiffRational answers a RATIONAL tag's value, which lives at an offset
// because it does not fit in the four bytes of the entry.
func tiffRational(t *testing.T, content []byte, tag uint16) float64 {
	t.Helper()
	entry, found := tiffEntries(t, content)[tag]
	if !found {
		t.Fatalf("the TIFF carries no tag %#x", tag)
	}
	at := int(tiffOrder.Uint32(entry[8:12]))
	if at+8 > len(content) {
		t.Fatalf("the value of tag %#x is past the end of the file", tag)
	}
	numerator := tiffOrder.Uint32(content[at : at+4])
	denominator := tiffOrder.Uint32(content[at+4 : at+8])
	if denominator == 0 {
		t.Fatalf("the value of tag %#x has a zero denominator", tag)
	}
	return float64(numerator) / float64(denominator)
}

// bitsPerSampleOfTIFF answers the depth of the first sample.
//
// BitsPerSample carries one short per sample, so it fits in the entry only
// where the image has one channel; with three it is an offset to three shorts,
// which is what the Java's writer produces for an RGB image too.
func bitsPerSampleOfTIFF(t *testing.T, content []byte) int {
	t.Helper()
	entry, found := tiffEntries(t, content)[0x0102]
	if !found {
		t.Fatal("the TIFF carries no BitsPerSample")
	}
	if tiffOrder.Uint32(entry[4:8]) == 1 {
		return int(tiffOrder.Uint16(entry[8:10]))
	}
	at := int(tiffOrder.Uint32(entry[8:12]))
	if at+2 > len(content) {
		t.Fatal("the BitsPerSample value is past the end of the file")
	}
	return int(tiffOrder.Uint16(content[at : at+2]))
}

// compressionOfTIFF answers the Compression tag, where 1 is no compression.
func compressionOfTIFF(t *testing.T, content []byte) int {
	t.Helper()
	return tiffShort(t, content, 0x0103)
}

// tiffTagValue answers a tag that carries one SHORT or one LONG.
func tiffTagValue(t *testing.T, content []byte, tag uint16) int {
	t.Helper()
	entry, found := tiffEntries(t, content)[tag]
	if !found {
		t.Fatalf("the TIFF carries no tag %#x", tag)
	}
	if tiffOrder.Uint16(entry[2:4]) == 3 { // SHORT
		return int(tiffOrder.Uint16(entry[8:10]))
	}
	return int(tiffOrder.Uint32(entry[8:12]))
}

// tiffTagCount answers how many values a tag carries.
func tiffTagCount(t *testing.T, content []byte, tag uint16) int {
	t.Helper()
	entry, found := tiffEntries(t, content)[tag]
	if !found {
		t.Fatalf("the TIFF carries no tag %#x", tag)
	}
	return int(tiffOrder.Uint32(entry[4:8]))
}

// tiffASCII answers an ASCII tag's value, without its terminating zero.
func tiffASCII(t *testing.T, content []byte, tag uint16) string {
	t.Helper()
	entry, found := tiffEntries(t, content)[tag]
	if !found {
		t.Fatalf("the TIFF carries no tag %#x", tag)
	}
	count := int(tiffOrder.Uint32(entry[4:8]))
	value := entry[8:12]
	if count > 4 {
		at := int(tiffOrder.Uint32(entry[8:12]))
		if at+count > len(content) {
			t.Fatalf("the value of tag %#x is past the end of the file", tag)
		}
		value = content[at : at+count]
	}
	return string(bytes.TrimRight(value[:min(count, len(value))], "\x00"))
}

// pngChunkNames answers the chunks of a PNG, in the order they appear.
func pngChunkNames(t *testing.T, content []byte) []string {
	t.Helper()
	const signature = 8
	var kinds []string
	at := signature
	for at+8 <= len(content) {
		length := int(binary.BigEndian.Uint32(content[at : at+4]))
		kinds = append(kinds, string(content[at+4:at+8]))
		next := at + 8 + length + 4
		if next <= at || next > len(content) {
			break
		}
		at = next
	}
	return kinds
}

// bigEndianUint32 reads the four bytes a PNG chunk stores a number in.
func bigEndianUint32(b []byte) uint32 { return binary.BigEndian.Uint32(b) }

// decodeAny reads a written file back with whatever can read it.
func decodeAny(t *testing.T, format string, content []byte) (image.Image, error) {
	t.Helper()
	switch format {
	case "png":
		return png.Decode(bytes.NewReader(content))
	case "jpg", "jpeg":
		return jpeg.Decode(bytes.NewReader(content))
	case "gif":
		return gif.Decode(bytes.NewReader(content))
	case "bmp":
		return decodeBMP(content)
	case "tiff", "tif":
		return decodeUncompressedTIFF(t, content)
	}
	return nil, fmt.Errorf("no reader for %s", format)
}
