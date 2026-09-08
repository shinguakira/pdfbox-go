package imageio_test

// Readers for the two formats Go cannot read and this port writes.
//
// They exist so the writers can be checked against something other than
// themselves: a file that only its own writer can read has not been shown to be
// a file at all. Both read only what the writers here produce -- uncompressed,
// one strip, bottom-up for BMP and top-down for TIFF -- and say so rather than
// pretending to be general.

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"testing"
)

// decodeBMP reads a 24-bit uncompressed BMP.
func decodeBMP(content []byte) (image.Image, error) {
	if len(content) < 54 || content[0] != 'B' || content[1] != 'M' {
		return nil, fmt.Errorf("not a BMP")
	}
	dataOffset := int(binary.LittleEndian.Uint32(content[10:14]))
	width := int(int32(binary.LittleEndian.Uint32(content[18:22])))
	height := int(int32(binary.LittleEndian.Uint32(content[22:26])))
	bits := int(binary.LittleEndian.Uint16(content[28:30]))
	if bits != 24 {
		return nil, fmt.Errorf("this reader takes 24-bit BMPs, not %d-bit", bits)
	}
	if compression := binary.LittleEndian.Uint32(content[30:34]); compression != 0 {
		return nil, fmt.Errorf("this reader takes uncompressed BMPs, not %d", compression)
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	stride := (width*3 + 3) &^ 3
	for y := 0; y < height; y++ {
		// A BMP is stored bottom-up.
		row := dataOffset + (height-1-y)*stride
		if row+width*3 > len(content) {
			return nil, fmt.Errorf("the BMP data is short")
		}
		for x := 0; x < width; x++ {
			at := row + x*3
			img.Set(x, y, color.RGBA{
				B: content[at], G: content[at+1], R: content[at+2], A: 255,
			})
		}
	}
	return img, nil
}

// decodeUncompressedTIFF reads what the TIFF writer here produces: one strip,
// no compression, 8 bits per sample in one or three channels.
func decodeUncompressedTIFF(t *testing.T, content []byte) (image.Image, error) {
	t.Helper()
	width := tiffShort(t, content, 0x0100)
	height := tiffShort(t, content, 0x0101)
	bits := bitsPerSampleOfTIFF(t, content)
	samples := tiffShort(t, content, 0x0115)
	if bits != 8 {
		return nil, fmt.Errorf("this reader takes 8-bit TIFFs, not %d-bit", bits)
	}
	entry, found := tiffEntries(t, content)[0x0111] // StripOffsets
	if !found {
		return nil, fmt.Errorf("the TIFF carries no StripOffsets")
	}
	at := int(tiffOrder.Uint32(entry[8:12]))

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	stride := width * samples
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			pixel := at + y*stride + x*samples
			if pixel+samples > len(content) {
				return nil, fmt.Errorf("the TIFF data is short")
			}
			switch samples {
			case 1:
				grey := content[pixel]
				img.Set(x, y, color.RGBA{R: grey, G: grey, B: grey, A: 255})
			case 4:
				img.Set(x, y, color.CMYK{
					C: content[pixel], M: content[pixel+1],
					Y: content[pixel+2], K: content[pixel+3],
				})
			default:
				img.Set(x, y, color.RGBA{
					R: content[pixel], G: content[pixel+1], B: content[pixel+2], A: 255,
				})
			}
		}
	}
	return img, nil
}
