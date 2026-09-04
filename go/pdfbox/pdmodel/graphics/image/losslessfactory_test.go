package image

import (
	"bytes"
	goimage "image"
	goimagecolor "image/color"
	"image/png"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// Tests for LosslessFactory, written from the source.
//
// LosslessFactoryTest is not ported: all fifteen of its tests write the image
// into a document and save it, and several then render the page back, which
// needs the writer of slice 7 and the renderer of slice 9. What can be checked
// without either is the property the factory is named for -- that what goes in
// comes back out unchanged -- and that is what these do: build a picture, run it
// through the factory, read the image XObject back, and compare every pixel.
//
// That is a strict test of the predictor encoder in particular. It writes each
// row through all five PNG predictors and keeps the smallest; a wrong Paeth or
// a wrong average would still deflate, and would still decode, and the pixels
// would be wrong.

// checkIdent is the name Java's LosslessFactoryTest gives the same idea.
func checkIdent(t *testing.T, expected goimage.Image, ximage *PDImageXObject) {
	t.Helper()
	got, err := ximage.Image()
	if err != nil {
		t.Fatalf("Image: %v", err)
	}
	bounds := expected.Bounds()
	if got.Bounds().Dx() != bounds.Dx() || got.Bounds().Dy() != bounds.Dy() {
		t.Fatalf("the image is %v, want %v", got.Bounds(), bounds)
	}
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			er, eg, eb, _ := unpremultiply(expected.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA())
			ar, ag, ab, _ := got.At(got.Bounds().Min.X+x, got.Bounds().Min.Y+y).RGBA()
			if er>>8 != ar>>8 || eg>>8 != ag>>8 || eb>>8 != ab>>8 {
				t.Fatalf("pixel (%d,%d) = (%d,%d,%d), want (%d,%d,%d)",
					x, y, ar>>8, ag>>8, ab>>8, er>>8, eg>>8, eb>>8)
			}
		}
	}
}

func TestLosslessRGBRoundTrip(t *testing.T) {
	// A picture with a gradient in each direction and a hard edge down the
	// middle: the gradient favours the Sub and Up predictors, the edge favours
	// None, so the row-by-row choice is exercised both ways.
	const width, height = 61, 37
	img := goimage.NewRGBA(goimage.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := goimagecolor.RGBA{
				R: uint8(x * 4),
				G: uint8(y * 7),
				B: uint8((x ^ y) * 3),
				A: 255,
			}
			if x > width/2 {
				c.R, c.B = 255-c.R, 0
			}
			img.SetRGBA(x, y, c)
		}
	}

	ximage, err := CreateFromImage(testDocument{}, img)
	if err != nil {
		t.Fatalf("CreateFromImage: %v", err)
	}
	validate(t, ximage, 8, width, height, "png", "DeviceRGB")
	checkIdent(t, img, ximage)
}

func TestLosslessGrayRoundTrip(t *testing.T) {
	const width, height = 23, 19
	img := goimage.NewGray(goimage.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetGray(x, y, goimagecolor.Gray{Y: uint8(x*11 + y*5)})
		}
	}

	ximage, err := CreateFromImage(testDocument{}, img)
	if err != nil {
		t.Fatalf("CreateFromImage: %v", err)
	}
	validate(t, ximage, 8, width, height, "png", "DeviceGray")
	checkIdent(t, img, ximage)
}

// TestLosslessAlphaBecomesASoftMask checks that an image with transparency
// writes its alpha channel as an /SMask, which is how a PDF carries one.
func TestLosslessAlphaBecomesASoftMask(t *testing.T) {
	const width, height = 16, 16
	img := goimage.NewNRGBA(goimage.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetNRGBA(x, y, goimagecolor.NRGBA{
				R: uint8(x * 16), G: uint8(y * 16), B: 128, A: uint8(x * 16),
			})
		}
	}

	ximage, err := CreateFromImage(testDocument{}, img)
	if err != nil {
		t.Fatalf("CreateFromImage: %v", err)
	}
	softMask := ximage.SoftMask()
	if softMask == nil {
		t.Fatal("an image with alpha should have a soft mask")
	}
	if got := softMask.Width(); got != width {
		t.Errorf("the soft mask is %d wide, want %d", got, width)
	}
	if got := softMask.Height(); got != height {
		t.Errorf("the soft mask is %d high, want %d", got, height)
	}
	colorSpace, err := softMask.ColorSpace()
	if err != nil {
		t.Fatalf("the soft mask colour space: %v", err)
	}
	if got := colorSpace.Name(); got != "DeviceGray" {
		t.Errorf("the soft mask colour space is %q, want DeviceGray", got)
	}

	// the alpha ramps with x, so read it back and check it
	maskImage, err := softMask.Image()
	if err != nil {
		t.Fatalf("the soft mask image: %v", err)
	}
	for x := 0; x < width; x++ {
		want := uint32(x * 16)
		r, _, _, _ := maskImage.At(x, 0).RGBA()
		if r>>8 != want {
			t.Fatalf("the soft mask at (%d,0) = %d, want %d", x, r>>8, want)
		}
	}
}

// TestLosslessOpaqueHasNoSoftMask checks the other side of it.
func TestLosslessOpaqueHasNoSoftMask(t *testing.T) {
	img := goimage.NewRGBA(goimage.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.SetRGBA(x, y, goimagecolor.RGBA{R: 10, G: 20, B: 30, A: 255})
		}
	}
	ximage, err := CreateFromImage(testDocument{}, img)
	if err != nil {
		t.Fatalf("CreateFromImage: %v", err)
	}
	if ximage.SoftMask() != nil {
		t.Error("an opaque image should have no soft mask")
	}
}

// TestLosslessPredictorParameters checks the /DecodeParms the predictor encoder
// writes, which is what tells a reader the rows carry PNG predictor markers.
func TestLosslessPredictorParameters(t *testing.T) {
	img := goimage.NewRGBA(goimage.Rect(0, 0, 100, 100))
	ximage, err := CreateFromImage(testDocument{}, img)
	if err != nil {
		t.Fatalf("CreateFromImage: %v", err)
	}
	decodeParms := ximage.COSDictionary().GetCOSDictionary(cos.DecodeParms)
	if decodeParms == nil {
		t.Fatal("the predictor encoder should write /DecodeParms")
	}
	if got := decodeParms.GetInt(cos.Predictor); got != 15 {
		t.Errorf("/Predictor = %d, want 15", got)
	}
	if got := decodeParms.GetInt(cos.Columns); got != 100 {
		t.Errorf("/Columns = %d, want 100", got)
	}
	if got := decodeParms.GetInt(cos.Colors); got != 3 {
		t.Errorf("/Colors = %d, want 3", got)
	}
	if got := decodeParms.GetInt(cos.BitsPerComponent); got != 8 {
		t.Errorf("/BitsPerComponent = %d, want 8", got)
	}
}

// TestLosslessFromRealPNG runs a checked-in PNG through the factory and back.
func TestLosslessFromRealPNG(t *testing.T) {
	for _, name := range []string{"png.png", "png_gray.png", "png_alpha_rgb.png"} {
		t.Run(name, func(t *testing.T) {
			img, err := png.Decode(bytes.NewReader(fixtureBytes(t, name)))
			if err != nil {
				t.Fatalf("decoding %s: %v", name, err)
			}
			ximage, err := CreateFromImage(testDocument{}, img)
			if err != nil {
				t.Fatalf("CreateFromImage: %v", err)
			}
			checkIdent(t, img, ximage)
		})
	}
}
