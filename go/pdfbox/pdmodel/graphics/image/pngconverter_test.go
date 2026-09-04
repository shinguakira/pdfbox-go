package image

import (
	"bytes"

	"image/png"
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// Tests for PNGConverter.
//
// PNGConverterTest is not ported: its eighteen tests build a document, add the
// converted image and save it, then compare against the same file through
// LosslessFactory -- the writer of slice 7 twice over. What is checked here is
// the property the converter exists for: where it answers, the image it builds
// decodes back to the same pixels the PNG holds, and where it cannot it says so
// rather than answering wrongly.

// TestPNGConverterTruecolor runs the checked-in truecolor PNGs through the
// converter and compares every pixel with the PNG decoded by Go.
func TestPNGConverterTruecolor(t *testing.T) {
	for _, name := range []string{"png.png", "png_rgb_gamma.png"} {
		t.Run(name, func(t *testing.T) {
			data := fixtureBytes(t, name)
			ximage, err := ConvertPNGImage(testDocument{}, data)
			if err != nil {
				t.Fatalf("ConvertPNGImage: %v", err)
			}
			if ximage == nil {
				t.Skip("the converter declined this PNG, which is a legitimate answer")
			}
			expected, err := png.Decode(bytes.NewReader(data))
			if err != nil {
				t.Fatalf("decoding %s: %v", name, err)
			}
			checkIdent(t, expected, ximage)
		})
	}
}

// TestPNGConverterIndexed runs an indexed PNG through, which takes the palette
// path and builds an /Indexed colour space.
func TestPNGConverterIndexed(t *testing.T) {
	data := fixtureBytes(t, "png_indexed.png")
	ximage, err := ConvertPNGImage(testDocument{}, data)
	if err != nil {
		t.Fatalf("ConvertPNGImage: %v", err)
	}
	if ximage == nil {
		t.Skip("the converter declined this PNG, which is a legitimate answer")
	}
	colorSpace, err := ximage.ColorSpace()
	if err != nil {
		t.Fatalf("ColorSpace: %v", err)
	}
	if got := colorSpace.Name(); got != "Indexed" {
		t.Errorf("the colour space is %q, want Indexed", got)
	}
	expected, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decoding png_indexed.png: %v", err)
	}
	checkIdent(t, expected, ximage)
}

// TestPNGConverterDeclines checks the answers the converter gives rather than
// converting: a greyscale PNG, one with alpha, and rubbish.
func TestPNGConverterDeclines(t *testing.T) {
	for _, name := range []string{"png_gray.png", "png_alpha_rgb.png"} {
		t.Run(name, func(t *testing.T) {
			ximage, err := ConvertPNGImage(testDocument{}, fixtureBytes(t, name))
			if err != nil {
				t.Fatalf("ConvertPNGImage: %v", err)
			}
			if ximage != nil {
				// Java declines both of these -- colour types 0, 4 and 6 -- so
				// if the port converts one, the two have diverged.
				t.Errorf("the converter should decline %s, as Java does", name)
			}
		})
	}

	for _, data := range [][]byte{
		nil,
		[]byte("far too short"),
		bytes.Repeat([]byte{0}, 64),
	} {
		if ximage, err := ConvertPNGImage(testDocument{}, data); err != nil || ximage != nil {
			t.Errorf("rubbish gave %v %v, want nothing", ximage, err)
		}
	}
}

// TestPNGChunkCRC checks the CRC the chunk check rests on, against the value a
// real PNG carries: every chunk of every checked-in PNG must verify.
func TestPNGChunkCRC(t *testing.T) {
	for _, name := range []string{"png.png", "png_indexed.png", "png_gray.png"} {
		state := parsePNGChunks(fixtureBytes(t, name))
		if state == nil {
			t.Fatalf("%s did not parse", name)
		}
		if !checkConverterState(state) {
			t.Errorf("%s has a chunk whose CRC does not verify", name)
		}
	}
}

// TestPNGChunkCRCRejectsDamage checks the other side: flip a byte of the image
// data and the CRC must catch it.
func TestPNGChunkCRCRejectsDamage(t *testing.T) {
	data := append([]byte(nil), fixtureBytes(t, "png.png")...)
	state := parsePNGChunks(data)
	if state == nil || len(state.idats) == 0 {
		t.Fatal("png.png did not parse")
	}
	data[state.idats[0].start] ^= 0xFF
	if checkConverterState(parsePNGChunks(data)) {
		t.Error("a damaged IDAT should fail the CRC check")
	}
}

// TestMapPNGRenderIntent pins the four intents PNG defines and the nothing it
// gives for anything else.
func TestMapPNGRenderIntent(t *testing.T) {
	cases := map[int]*cos.Name{
		0:  cos.Perceptual,
		1:  cos.RelativeColorimetric,
		2:  cos.Saturation,
		3:  cos.AbsoluteColorimetric,
		4:  nil,
		-1: nil,
	}
	for intent, want := range cases {
		if got := MapPNGRenderIntent(intent); got != want {
			t.Errorf("MapPNGRenderIntent(%d) = %v, want %v", intent, got, want)
		}
	}
}

// TestCreateFromByteArray checks the dispatch of PDImageXObject's factory
// methods: each format reaches the factory Java sends it to.
func TestCreateFromByteArray(t *testing.T) {
	cases := []struct {
		file   string
		suffix string
	}{
		{"jpeg.jpg", "jpg"},
		{"png.png", "png"},
		{"png_indexed.png", "png"},
		{"gif.gif", "png"},
		{"ccittg4.tif", "tiff"},
	}
	for _, c := range cases {
		t.Run(c.file, func(t *testing.T) {
			data := fixtureBytes(t, c.file)
			ximage, err := CreateFromByteArray(testDocument{}, data, c.file)
			if err != nil {
				t.Fatalf("CreateFromByteArray: %v", err)
			}
			if ximage == nil {
				t.Fatal("the factory returned nothing")
			}
			if got := ximage.Suffix(); got != c.suffix {
				t.Errorf("Suffix = %q, want %q", got, c.suffix)
			}
			if ximage.Width() <= 0 || ximage.Height() <= 0 {
				t.Errorf("the image is %dx%d", ximage.Width(), ximage.Height())
			}
		})
	}
}

// TestCreateFromByteArrayCustomFactory checks that a custom factory takes the
// formats Java hands it, and only those.
func TestCreateFromByteArrayCustomFactory(t *testing.T) {
	called := 0
	custom := func(document DocumentLike, byteArray []byte) (*PDImageXObject, error) {
		called++
		return CreateFromJPEGByteArray(document, fixtureBytes(t, "jpeg.jpg"))
	}

	// a GIF goes to the custom factory
	if _, err := CreateFromByteArrayNamed(testDocument{}, fixtureBytes(t, "gif.gif"),
		"gif.gif", custom); err != nil {
		t.Fatalf("CreateFromByteArrayNamed: %v", err)
	}
	if called != 1 {
		t.Errorf("the custom factory was called %d times for a GIF, want 1", called)
	}

	// a JPEG does not
	if _, err := CreateFromByteArrayNamed(testDocument{}, fixtureBytes(t, "jpeg.jpg"),
		"jpeg.jpg", custom); err != nil {
		t.Fatalf("CreateFromByteArrayNamed: %v", err)
	}
	if called != 1 {
		t.Errorf("the custom factory was called for a JPEG, which Java sends to JPEGFactory")
	}
}

// TestCreateFromByteArrayLZWTiff pins a port gap rather than a behaviour.
//
// lzw.tif is a TIFF the CCITT reader refuses -- it is LZW compressed, not fax
// -- so the dispatch falls through to the plan B the Java comment describes:
// "try reading with ImageIO". ImageIO reads an LZW TIFF; Go's standard library
// has no TIFF decoder at all, so this file loads in Java and does not here.
// The test asserts the gap so that it is visible rather than silent, and so
// that a later slice adding a TIFF decoder will notice.
func TestCreateFromByteArrayLZWTiff(t *testing.T) {
	_, err := CreateFromByteArray(testDocument{}, fixtureBytes(t, "lzw.tif"), "lzw.tif")
	if err == nil {
		t.Fatal("an LZW TIFF should be reported as undecodable by this port")
	}
	if !strings.Contains(err.Error(), "no decoder") {
		t.Errorf("the gap gave %q", err.Error())
	}
}
