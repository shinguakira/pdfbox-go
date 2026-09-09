package raster

// The surfaces hold straight colour, and the type says so.
//
// `image.RGBA` is documented alpha-premultiplied and `image.NRGBA` is not, and
// what this package computes is straight: `blendInto` writes what
// BlendComposite's three lines produce, which are components normalized to
// 0..1 and put back, exactly as `getNormalizedComponents` and
// `getDataElements` do either side of Java's `compose`.
//
// It was an `image.RGBA` until a review asked which it was. Nothing visible
// moved -- for an opaque pixel the two are the same, and every page this
// package renders to RGB, Gray or Binary ends opaque -- but two things were
// wrong underneath, and both are tested here.

import (
	"bytes"
	goimage "image"
	goimagecolor "image/color"
	"image/png"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/rendering"
)

// TestAnARGBSurfaceExportsItsOwnColours is the one a caller would have seen.
//
// `RenderPage` answers the surface, and a caller encodes it. Go's PNG encoder
// asks the image what it holds: an `image.RGBA` is premultiplied, so it
// unpremultiplies on the way out, and a straight colour stored in one came
// back changed. Blue at half alpha went out as 253 rather than 255.
func TestAnARGBSurfaceExportsItsOwnColours(t *testing.T) {
	i := NewImage(1, 1, rendering.ARGB)
	i.SetAntiAliasing(false)
	i.SetPaint(rendering.ColorPaint{Red: 0, Green: 0, Blue: 1, Alpha: 0.5})
	if err := i.Fill(rectangle(0, 0, 1, 1)); err != nil {
		t.Fatalf("Fill: %v", err)
	}

	want := goimagecolor.NRGBA{R: 0, G: 0, B: 0xFF, A: 0x80}
	if got := i.dst.NRGBAAt(0, 0); got != want {
		t.Errorf("the surface holds %v, want %v", got, want)
	}

	var buffer bytes.Buffer
	if err := png.Encode(&buffer, i.Image()); err != nil {
		t.Fatalf("png.Encode: %v", err)
	}
	decoded, err := png.Decode(&buffer)
	if err != nil {
		t.Fatalf("png.Decode: %v", err)
	}
	exported, isStraight := decoded.(*goimage.NRGBA)
	if !isStraight {
		t.Fatalf("the PNG decoded as %T, want an *image.NRGBA", decoded)
	}
	if got := exported.NRGBAAt(0, 0); got != want {
		t.Errorf("the PNG holds %v, want %v", got, want)
	}
}

// TestAPartlyTransparentGroupKeepsItsColour is the other one.
//
// PopGroup called unpremultiply on the group's pixels, which were already
// straight, so a group whose contents were partly transparent came back paler
// than it was drawn. It could not show while the surfaces were opaque -- the
// method answers its argument unchanged for alpha 0 and 255 -- so a group with
// a half-alpha fill inside it is what asks the question.
func TestAPartlyTransparentGroupKeepsItsColour(t *testing.T) {
	i := NewImage(4, 4, rendering.ARGB)
	i.SetAntiAliasing(false)

	if err := i.PushGroup(nil, false, false, nil); err != nil {
		t.Fatalf("PushGroup: %v", err)
	}
	// A saturated blue at half alpha, drawn onto the group's empty surface.
	i.SetPaint(rendering.ColorPaint{Red: 0, Green: 0, Blue: 1, Alpha: 0.5})
	if err := i.Fill(rectangle(0, 0, 4, 4)); err != nil {
		t.Fatalf("Fill: %v", err)
	}
	if err := i.PopGroup(); err != nil {
		t.Fatalf("PopGroup: %v", err)
	}

	// The group is composited onto nothing at its own alpha, so the colour is
	// the colour and the alpha is the alpha.
	want := goimagecolor.NRGBA{R: 0, G: 0, B: 0xFF, A: 0x80}
	if got := i.dst.NRGBAAt(2, 2); got != want {
		t.Errorf("the group came out %v, want %v", got, want)
	}
}
