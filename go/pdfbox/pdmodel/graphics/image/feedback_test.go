package image

import (
	goimage "image"
	goimagecolor "image/color"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// What the slice 6 feedback asked, written down as tests.

// TestLosslessCMYKRoundTrip is the case the lossless tests missed: an
// *image.CMYK.
//
// The predictor encoder picked DeviceCMYK for it and then wrote nothing,
// because the branch meant to read the four channels tested for a CMYK()
// method that image/color.CMYK does not have. Every sample stayed zero, which
// in CMYK is white, so a colour picture came back blank.
func TestLosslessCMYKRoundTrip(t *testing.T) {
	const width, height = 17, 13
	img := goimage.NewCMYK(goimage.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetCMYK(x, y, goimagecolor.CMYK{
				C: uint8(x * 15), M: uint8(y * 19), Y: uint8((x + y) * 7), K: uint8(x * y % 256),
			})
		}
	}

	ximage, err := CreateFromImage(testDocument{}, img)
	if err != nil {
		t.Fatalf("CreateFromImage: %v", err)
	}
	validate(t, ximage, 8, width, height, "png", "DeviceCMYK")

	// the samples must come back exactly, which is what lossless means
	raster, err := ximage.RawRaster()
	if err != nil {
		t.Fatalf("RawRaster: %v", err)
	}
	pixel := make([]int, 4)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			raster.GetPixel(x, y, pixel)
			want := img.CMYKAt(x, y)
			if pixel[0] != int(want.C) || pixel[1] != int(want.M) ||
				pixel[2] != int(want.Y) || pixel[3] != int(want.K) {
				t.Fatalf("sample (%d,%d) = %v, want (%d,%d,%d,%d)",
					x, y, pixel, want.C, want.M, want.Y, want.K)
			}
		}
	}
}

// TestJPEGFromCMYKImageDeclaresWhatItEncoded pins the agreement between the
// data and the dictionary.
//
// Go's image/jpeg has no four component encoder: it writes every non-grey image
// as a three component YCbCr JPEG. The factory read the Go image type and
// declared DeviceCMYK with a four entry decode array, so a reader would take
// three samples per pixel as four.
func TestJPEGFromCMYKImageDeclaresWhatItEncoded(t *testing.T) {
	const width, height = 12, 9
	img := goimage.NewCMYK(goimage.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetCMYK(x, y, goimagecolor.CMYK{C: uint8(x * 20), M: 60, Y: 90, K: uint8(y * 25)})
		}
	}

	ximage, err := CreateJPEGFromImage(testDocument{}, img, 0.75, 72)
	if err != nil {
		t.Fatalf("CreateJPEGFromImage: %v", err)
	}

	// the components the JPEG really has
	raw, err := ximage.PDStream().Stream().CreateRawReader()
	if err != nil {
		t.Fatalf("CreateRawReader: %v", err)
	}
	encoded := make([]byte, 4096)
	n, _ := readFull(raw, encoded)
	_, _, components, err := jpegFrameHeader(encoded[:n])
	if err != nil {
		t.Fatalf("reading the encoded frame header: %v", err)
	}

	colorSpace, err := ximage.ColorSpace()
	if err != nil {
		t.Fatalf("ColorSpace: %v", err)
	}
	if got := colorSpace.NumberOfComponents(); got != components {
		t.Errorf("the image declares %s, %d components, but the JPEG has %d",
			colorSpace.Name(), got, components)
	}
	decode := ximage.Decode()
	if decode != nil && decode.Size() != components*2 {
		t.Errorf("the decode array has %d entries for %d components",
			decode.Size(), components)
	}

	// and it must still decode to the right size
	picture, err := ximage.Image()
	if err != nil {
		t.Fatalf("Image: %v", err)
	}
	assertSize(t, "image", picture, width, height)
}

// readFull reads as much as it can without failing on a short stream.
func readFull(r interface{ Read([]byte) (int, error) }, buf []byte) (int, error) {
	total := 0
	for total < len(buf) {
		n, err := r.Read(buf[total:])
		total += n
		if err != nil {
			return total, nil
		}
		if n == 0 {
			return total, nil
		}
	}
	return total, nil
}

// TestJPEGFromGrayImageStaysGray checks the other two arms of the same
// agreement, which were already right and must stay so.
func TestJPEGFromGrayImageStaysGray(t *testing.T) {
	gray := goimage.NewGray(goimage.Rect(0, 0, 8, 8))
	ximage, err := CreateJPEGFromImage(testDocument{}, gray, 0.75, 72)
	if err != nil {
		t.Fatalf("CreateJPEGFromImage: %v", err)
	}
	colorSpace, err := ximage.ColorSpace()
	if err != nil {
		t.Fatal(err)
	}
	if got := colorSpace.Name(); got != "DeviceGray" {
		t.Errorf("a grey image gave %q", got)
	}
	if ximage.Decode() != nil {
		t.Error("a grey JPEG should have no decode array")
	}

	rgba := goimage.NewRGBA(goimage.Rect(0, 0, 8, 8))
	ximage, err = CreateJPEGFromImage(testDocument{}, rgba, 0.75, 72)
	if err != nil {
		t.Fatalf("CreateJPEGFromImage: %v", err)
	}
	colorSpace, err = ximage.ColorSpace()
	if err != nil {
		t.Fatal(err)
	}
	if got := colorSpace.Name(); got != "DeviceRGB" {
		t.Errorf("an RGB image gave %q", got)
	}
	if ximage.Decode() != nil {
		t.Error("an RGB JPEG should have no decode array")
	}
}

// TestLosslessCMYKDecodeArray checks that the lossless path does not add the
// inverted decode array a DCT encoded CMYK image needs -- a flate encoded one
// holds its samples the right way up.
func TestLosslessCMYKDecodeArray(t *testing.T) {
	img := goimage.NewCMYK(goimage.Rect(0, 0, 4, 4))
	ximage, err := CreateFromImage(testDocument{}, img)
	if err != nil {
		t.Fatalf("CreateFromImage: %v", err)
	}
	if got := ximage.COSDictionary().GetCOSArray(cos.Decode); got != nil {
		t.Errorf("a flate encoded CMYK image should have no decode array, got %v", got)
	}
}
