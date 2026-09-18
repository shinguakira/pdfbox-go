package image

import (
	goimage "image"
	goimagecolor "image/color"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// TestSoftMaskedImageReadsBackStraight checks what an image with a soft mask
// reads back as, pixel by pixel.
//
// Java's getImage applies the soft mask into a TYPE_INT_ARGB BufferedImage,
// which holds straight colour: the colour of a pixel does not depend on its
// alpha, and getRGB answers exactly what was stored. The expected values are
// PDFBox's, printed by LosslessFactory.createFromImage for these four pixels
// followed by getImage().getRGB(x, 0): every one comes back as it went in,
// including the fully transparent pixel and the nearly transparent one, and
// isAlphaPremultiplied() is false.
//
// Found by the corpus comparison of image pixels: PDFBox's
// podofo/TestImage1.pdf reads the same decoded bytes on both sides and different
// pixels, all of them where the alpha is below 255.
func TestSoftMaskedImageReadsBackStraight(t *testing.T) {
	want := []goimagecolor.NRGBA{
		{R: 0xC8, G: 0x64, B: 0x32, A: 0x80},
		{R: 0x0A, G: 0x14, B: 0x1E, A: 0x00},
		{R: 0xFF, G: 0x80, B: 0x40, A: 0x01},
		{R: 0x10, G: 0x20, B: 0x30, A: 0xFF},
	}
	in := goimage.NewNRGBA(goimage.Rect(0, 0, len(want), 1))
	for x, c := range want {
		in.SetNRGBA(x, 0, c)
	}

	ximage, err := CreateFromImage(testDocument{}, in)
	if err != nil {
		t.Fatalf("CreateFromImage: %v", err)
	}
	if ximage.SoftMask() == nil {
		t.Fatal("an image with alpha should have a soft mask")
	}
	out, err := ximage.Image()
	if err != nil {
		t.Fatalf("Image: %v", err)
	}
	for x, w := range want {
		got := goimagecolor.NRGBAModel.Convert(out.At(out.Bounds().Min.X+x, out.Bounds().Min.Y)).(goimagecolor.NRGBA)
		if got != w {
			t.Errorf("pixel %d reads back as %+v, want %+v, which is what PDFBox's getRGB answers", x, got, w)
		}
	}
}

// TestColorKeyMaskedImageReadsBackStraight is the same for a colour key mask.
//
// Java's applyColorKeyMask composes into a TYPE_INT_ARGB image too, copying the
// colour and setting the alpha from the key. PDFBox's answers, printed for a
// LosslessFactory image of these two pixels with /Mask [10 10 20 20 30 30]:
// 000a141e and ff28323c -- the keyed-out pixel keeps its colour.
func TestColorKeyMaskedImageReadsBackStraight(t *testing.T) {
	in := goimage.NewRGBA(goimage.Rect(0, 0, 2, 1))
	in.SetRGBA(0, 0, goimagecolor.RGBA{R: 0x0A, G: 0x14, B: 0x1E, A: 0xFF})
	in.SetRGBA(1, 0, goimagecolor.RGBA{R: 0x28, G: 0x32, B: 0x3C, A: 0xFF})

	ximage, err := CreateFromImage(testDocument{}, in)
	if err != nil {
		t.Fatalf("CreateFromImage: %v", err)
	}
	key := cos.NewArray()
	for _, v := range []int64{10, 10, 20, 20, 30, 30} {
		key.Add(cos.GetInteger(v))
	}
	ximage.COSDictionary().SetItem(cos.Mask, key)

	out, err := ximage.Image()
	if err != nil {
		t.Fatalf("Image: %v", err)
	}
	want := []goimagecolor.NRGBA{
		{R: 0x0A, G: 0x14, B: 0x1E, A: 0x00},
		{R: 0x28, G: 0x32, B: 0x3C, A: 0xFF},
	}
	for x, w := range want {
		got := goimagecolor.NRGBAModel.Convert(out.At(out.Bounds().Min.X+x, out.Bounds().Min.Y)).(goimagecolor.NRGBA)
		if got != w {
			t.Errorf("pixel %d reads back as %+v, want %+v, which is what PDFBox's getRGB answers", x, got, w)
		}
	}
}
