package image

import (
	goimage "image"
	goimagecolor "image/color"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// Tests for CCITTFactory.
//
// CCITTFactoryTest is not ported whole: five of its ten tests save the document
// and the rest compare against images built by the other factories, which is
// the same comparison PDImageXObjectTest makes. What is here is the part that
// needs neither -- the TIFF reader, over the four checked-in TIFF files, and the
// round trip through the fax encoder and back.

func TestCCITTFromTIFF(t *testing.T) {
	cases := []struct {
		file   string
		width  int
		height int
		k      int
	}{
		// ccittg4.tif is T6, which the reader reports as K -1
		{"ccittg4.tif", 344, 287, -1},
		// ccittg3.tif is T4 one dimensional, K 0
		{"ccittg3.tif", 344, 287, 0},
	}
	for _, c := range cases {
		t.Run(c.file, func(t *testing.T) {
			ximage, err := CreateCCITTFromByteArray(testDocument{}, fixtureBytes(t, c.file))
			if err != nil {
				t.Fatalf("CreateCCITTFromByteArray: %v", err)
			}
			if ximage == nil {
				t.Fatal("the factory returned nothing")
			}
			validate(t, ximage, 1, c.width, c.height, "tiff", "DeviceGray")

			decodeParms := ximage.COSDictionary().GetCOSDictionary(cos.DecodeParms)
			if decodeParms == nil {
				t.Fatal("the factory should write /DecodeParms")
			}
			if got := decodeParms.GetInt(cos.K); got != c.k {
				t.Errorf("/K = %d, want %d", got, c.k)
			}
			if got := decodeParms.GetInt(cos.Columns); got != c.width {
				t.Errorf("/Columns = %d, want %d", got, c.width)
			}
			if got := decodeParms.GetInt(cos.Rows); got != c.height {
				t.Errorf("/Rows = %d, want %d", got, c.height)
			}
		})
	}
}

// TestCCITTMultiPage reads the second image of a multi-page TIFF, which is what
// the numbered overloads are for.
func TestCCITTMultiPage(t *testing.T) {
	data := fixtureBytes(t, "ccittg4multi.tif")
	first, err := CreateCCITTFromByteArrayNumbered(testDocument{}, data, 0)
	if err != nil {
		t.Fatalf("page 0: %v", err)
	}
	if first == nil {
		t.Fatal("page 0 is missing")
	}
	second, err := CreateCCITTFromByteArrayNumbered(testDocument{}, data, 1)
	if err != nil {
		t.Fatalf("page 1: %v", err)
	}
	if second == nil {
		t.Fatal("page 1 is missing")
	}
	// past the end the reader returns nothing rather than failing, which is
	// Java returning null from createFromRandomAccessImpl.
	past, err := CreateCCITTFromByteArrayNumbered(testDocument{}, data, 99)
	if err != nil {
		t.Fatalf("page 99: %v", err)
	}
	if past != nil {
		t.Error("a page past the end should give nothing")
	}
}

// TestCCITTNotATiff checks the three ways the header check fails.
func TestCCITTNotATiff(t *testing.T) {
	for _, data := range [][]byte{
		{},
		[]byte("not a tiff at all"),
		{'I', 'M', 42, 0},             // the two endianness bytes disagree
		{'X', 'X', 42, 0},             // neither M nor I
		{'I', 'I', 43, 0, 8, 0, 0, 0}, // bigtiff
	} {
		if _, err := CreateCCITTFromByteArray(testDocument{}, data); err == nil {
			t.Errorf("%q should be refused", data)
		}
	}
}

// TestCCITTFromImageRoundTrip encodes a bitmap as Group 4 fax and reads it back
// through the filter, which is the property the factory rests on.
func TestCCITTFromImageRoundTrip(t *testing.T) {
	const width, height = 40, 24
	img := goimage.NewGray(goimage.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// a pattern with runs of both colours, which is what a fax encodes
			value := byte(0)
			if (x/3+y/2)%2 == 0 {
				value = 255
			}
			img.SetGray(x, y, goimagecolor.Gray{Y: value})
		}
	}

	ximage, err := CreateCCITTFromImage(testDocument{}, img)
	if err != nil {
		t.Fatalf("CreateCCITTFromImage: %v", err)
	}
	validate(t, ximage, 1, width, height, "tiff", "DeviceGray")

	got, err := ximage.Image()
	if err != nil {
		t.Fatalf("Image: %v", err)
	}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			want := 0
			if (x/3+y/2)%2 == 0 {
				want = 0xFFFFFF
			}
			if pixel := rgbAt(got, x, y); pixel != want {
				t.Fatalf("pixel (%d,%d) = %#06x, want %#06x", x, y, pixel, want)
			}
		}
	}
}

// TestCCITTByteShortPaddedWithGarbage is CCITTFactoryTest.testByteShortPaddedWithGarbage,
// minus its save. It reads two TIFFs whose byte and short tag values are padded
// with garbage in the three or two bytes after them, which is what the comment
// in extractFromTiff is about: "when the type is shorter than 4 bytes, the rest
// can be garbage and must be ignored". Reading those bytes as part of the value
// gives 842530817 where the answer is 1.
func TestCCITTByteShortPaddedWithGarbage(t *testing.T) {
	for _, ext := range []string{".tif", "-bigendian.tif"} {
		name := "ccittg3-garbage-padded-fields" + ext
		t.Run(name, func(t *testing.T) {
			ximage, err := CreateCCITTFromByteArray(testDocument{}, fixtureBytes(t, name))
			if err != nil {
				t.Fatalf("CreateCCITTFromByteArray: %v", err)
			}
			if ximage == nil {
				t.Fatal("the factory returned nothing")
			}
			validate(t, ximage, 1, 344, 287, "tiff", "DeviceGray")
		})
	}
}
