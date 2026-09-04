package image

import (
	goimage "image"
	goimagecolor "image/color"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// Port of
// pdfbox/src/test/java/org/apache/pdfbox/pdmodel/graphics/image/PDInlineImageTest.java.
//
// The half of testInlineImage that builds the two images and checks every pixel
// of them is here. Its second half draws them into a page with a
// PDPageContentStream, saves the document and renders it back, which needs the
// writer of slice 7 and the renderer of slice 9.
//
// This is the end-to-end test of the one bit path: SampledImageReader.from1Bit,
// the decode array, and PDDeviceGray turning the raster into a picture.

func TestInlineImage(t *testing.T) {
	dict := cos.NewDictionary()
	dict.SetBoolean(cos.IM, true)
	const width = 31
	const height = 27
	dict.SetInt(cos.W, width)
	dict.SetInt(cos.H, height)
	dict.SetInt(cos.BPC, 1)
	rowbytes := width / 8
	if rowbytes*8 < width {
		// PDF spec:
		// If the number of data bits per row is not a multiple of 8,
		// the end of the row is padded with extra bits to fill out the last byte.
		rowbytes++
	}

	// draw a grid
	datalen := rowbytes * height
	data := make([]byte, datalen)
	for i := 0; i < datalen; i++ {
		if i/4%2 == 0 {
			data[i] = 0b10101010
		}
	}

	inlineImage1, err := NewPDInlineImage(dict, data, nil)
	if err != nil {
		t.Fatalf("NewPDInlineImage: %v", err)
	}
	if !inlineImage1.IsStencil() {
		t.Error("the image should be a stencil")
	}
	if got := inlineImage1.Width(); got != width {
		t.Errorf("Width = %d, want %d", got, width)
	}
	if got := inlineImage1.Height(); got != height {
		t.Errorf("Height = %d, want %d", got, height)
	}
	if got := inlineImage1.BitsPerComponent(); got != 1 {
		t.Errorf("BitsPerComponent = %d, want 1", got)
	}

	dict2 := cos.NewDictionary()
	dict2.AddAll(dict)

	// use decode array to revert in image2
	decodeArray := cos.NewArray()
	decodeArray.Add(cos.IntegerOne)
	decodeArray.Add(cos.IntegerZero)
	dict2.SetItem(cos.Decode, decodeArray)

	inlineImage2, err := NewPDInlineImage(dict2, data, nil)
	if err != nil {
		t.Fatalf("NewPDInlineImage: %v", err)
	}

	paint := goimagecolor.RGBA{A: 255}
	stencilImage, err := inlineImage1.StencilImage(paint)
	if err != nil {
		t.Fatalf("StencilImage: %v", err)
	}
	assertSize(t, "stencilImage", stencilImage, width, height)

	stencilImage2, err := inlineImage2.StencilImage(paint)
	if err != nil {
		t.Fatalf("StencilImage: %v", err)
	}
	assertSize(t, "stencilImage2", stencilImage2, width, height)

	image1, err := inlineImage1.Image()
	if err != nil {
		t.Fatalf("Image: %v", err)
	}
	assertSize(t, "image1", image1, width, height)

	image2, err := inlineImage2.Image()
	if err != nil {
		t.Fatalf("Image: %v", err)
	}
	assertSize(t, "image2", image2, width, height)

	// compare: pixels with even coordinates are white (FF), all others are black (0)
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			want := 0
			if x%2 == 0 && y%2 == 0 {
				want = 0xFFFFFF
			}
			if got := rgbAt(image1, x, y); got != want {
				t.Fatalf("image1 pixel (%d,%d) = %#06x, want %#06x", x, y, got, want)
			}
		}
	}

	// compare: pixels with odd coordinates are white (FF), all others are black (0)
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			want := 0xFFFFFF
			if x%2 == 0 && y%2 == 0 {
				want = 0
			}
			if got := rgbAt(image2, x, y); got != want {
				t.Fatalf("image2 pixel (%d,%d) = %#06x, want %#06x", x, y, got, want)
			}
		}
	}
}

func assertSize(t *testing.T, what string, img goimage.Image, width, height int) {
	t.Helper()
	if got := img.Bounds().Dx(); got != width {
		t.Errorf("%s width = %d, want %d", what, got, width)
	}
	if got := img.Bounds().Dy(); got != height {
		t.Errorf("%s height = %d, want %d", what, got, height)
	}
}

// rgbAt is Java's BufferedImage.getRGB(x, y) & 0xFFFFFF.
func rgbAt(img goimage.Image, x, y int) int {
	b := img.Bounds()
	r, g, bl, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
	return int(r>>8)<<16 | int(g>>8)<<8 | int(bl>>8)
}

// TestInlineImageSuffix pins the suffix an inline image reports, which Java
// derives from its filters and which has no JPX or JBIG2 case because those
// filters do not exist for inline images.
func TestInlineImageSuffix(t *testing.T) {
	cases := []struct {
		filters []string
		want    string
	}{
		{nil, "png"},
		{[]string{"DCTDecode"}, "jpg"},
		{[]string{"DCT"}, "jpg"},
		{[]string{"CCITTFaxDecode"}, "tiff"},
		{[]string{"CCF"}, "tiff"},
		{[]string{"FlateDecode"}, "png"},
		{[]string{"AHx"}, "png"},
	}
	for _, c := range cases {
		dict := cos.NewDictionary()
		dict.SetInt(cos.W, 1)
		dict.SetInt(cos.H, 1)
		image := &PDInlineImage{parameters: dict}
		image.SetFilters(c.filters)
		if got := image.Suffix(); got != c.want {
			t.Errorf("Suffix for %v = %q, want %q", c.filters, got, c.want)
		}
	}
}
