package image

// JAVA-BUGS 31: `SampledImageReader.from8bit` copies a row of a requested
// region to an offset computed from the source row and the source width, where
// the raster it is filling is the region. In-package, because the reader's
// entry points are.

import (
	goimage "image"
	goimagecolor "image/color"
	"testing"

	awtgeom "github.com/shinguakira/pdfbox-go/go/awt/geom"
)

// TestFrom8BitRegionLandsOnTheRightRows is the defect.
//
// The expected values are the source pixels of the requested region: asking
// for a rectangle of an image must answer that rectangle. Java computes the
// destination offset as `y * inputWidth * numComponents`, which is the source
// row and the source stride; for any region that is a strict subset that is
// the wrong row, and once `y` is large enough it is past the end of the
// raster, which throws ArrayIndexOutOfBoundsException past the two exceptions
// `getRGBImage` catches. The line below it, for the subsampled case, walks the
// destination with a running index and gets it right.
func TestFrom8BitRegionLandsOnTheRightRows(t *testing.T) {
	// every pixel says where it came from, so a row landing anywhere else shows
	const width, height = 8, 6
	img := goimage.NewRGBA(goimage.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetRGBA(x, y, goimagecolor.RGBA{
				R: uint8(10 + x*10),
				G: uint8(100 + y*10),
				B: uint8(x*8 + y),
				A: 255,
			})
		}
	}

	ximage, err := CreateFromImage(testDocument{}, img)
	if err != nil {
		t.Fatalf("CreateFromImage: %v", err)
	}

	// a rectangle away from both edges, so neither the row nor the stride can
	// be right by accident
	region := awtgeom.Rectangle{X: 2, Y: 1, Width: 3, Height: 3}
	got, err := getRGBImageOfRegion(ximage, &region, 1, nil)
	if err != nil {
		t.Fatalf("getRGBImageOfRegion: %v", err)
	}
	if got.Bounds().Dx() != region.Width || got.Bounds().Dy() != region.Height {
		t.Fatalf("the region came back %v, want %d by %d",
			got.Bounds(), region.Width, region.Height)
	}
	for y := 0; y < region.Height; y++ {
		for x := 0; x < region.Width; x++ {
			wr, wg, wb, _ := img.At(region.X+x, region.Y+y).RGBA()
			gr, gg, gb, _ := got.At(got.Bounds().Min.X+x, got.Bounds().Min.Y+y).RGBA()
			if wr>>8 != gr>>8 || wg>>8 != gg>>8 || wb>>8 != gb>>8 {
				t.Errorf("region pixel (%d,%d) = (%d,%d,%d), want the source pixel "+
					"(%d,%d) = (%d,%d,%d)", x, y, gr>>8, gg>>8, gb>>8,
					region.X+x, region.Y+y, wr>>8, wg>>8, wb>>8)
			}
		}
	}
}

// TestFrom8BitWholeImageIsUnchanged keeps the fast path the fix does not
// touch: a region that is the whole image copies in one go, and the offset the
// fix changes is not on that path.
func TestFrom8BitWholeImageIsUnchanged(t *testing.T) {
	const width, height = 5, 4
	img := goimage.NewRGBA(goimage.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetRGBA(x, y, goimagecolor.RGBA{R: uint8(x * 20), G: uint8(y * 30),
				B: uint8(x + y), A: 255})
		}
	}
	ximage, err := CreateFromImage(testDocument{}, img)
	if err != nil {
		t.Fatalf("CreateFromImage: %v", err)
	}
	checkIdent(t, img, ximage)
}
