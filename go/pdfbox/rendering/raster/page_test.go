package raster_test

// A page, rendered by PDFBox and by the port, compared.
//
// Everything else in this package looks at one call. This looks at the whole
// of both renderers: the same PDF goes into PDFBox's PDFRenderer -- its
// parser, its PageDrawer, Graphics2D underneath -- and into this port's, and
// the two images are put side by side.
//
// `testdata/graphics.pdf` is written by `testdata/genpdf.go` and checked in,
// so both sides read the same bytes. `testdata/graphics-java.png` is what
// PDFBox made of it, written by `testdata/RenderDrv.java`. Regenerate both
// together; the drivers say how.
//
// The page has no text on it, deliberately. The glyph tests already compare
// text against the AWT references, and a font here would make this a test of
// the shaper rather than of the backend.

import (
	goimage "image"
	_ "image/png"
	"os"
	"testing"

	pdfbox "github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/rendering"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/rendering/raster"
)

// TestAPageRendersAsPDFBoxRendersIt is the comparison the branch exists to
// make possible.
//
// Every flat fill on the page is exact -- the two rectangles, the even-odd
// one, the half-alpha overlap, the multiplied one -- and so is every interior.
// What differs is three things, all of them measured on their own in
// java2d_test.go and none of them a wrong shape:
//
//	strokes    1145 pixels, up to 191. PDFBox leaves KEY_STROKE_CONTROL at
//	           the JDK default, which moves stroke geometry onto pixel
//	           centres before stroking, and this backend renders the geometry
//	           as given, so every stroke edge sits half a pixel over.
//	shading     2220 pixels, up to 2. The colour table is indexed by a
//	           truncated product, and the last place of that product is not
//	           the same in float32 as it was in Java's float; the band drifts
//	           one step either way and never further.
//	diagonals     68 pixels, up to 6. The rotated square's edges, where the
//	           two rasterisers put different fractions on an edge pixel.
//
// The counts are pinned exactly and in both directions: ink that moves fails
// this, and ink that stops moving fails it too, so the record has to be
// updated with the code.
func TestAPageRendersAsPDFBoxRendersIt(t *testing.T) {
	// differing is how many of the 40000 pixels are not PDFBox's, and beyond
	// is how many of those are more than a quarter of one channel away, which
	// is the stroke shift and nothing else.
	const (
		differingPixels = 3382
		beyondEdges     = 1002
	)

	reference := readPNG(t, "testdata/graphics-java.png")

	document, err := pdfbox.LoadPDF("testdata/graphics.pdf")
	if err != nil {
		t.Fatalf("loading the page: %v", err)
	}
	defer document.Close()

	rendered, err := raster.RenderPage(document, 0, 1, rendering.RGB)
	if err != nil {
		t.Fatalf("rendering the page: %v", err)
	}

	if got, want := rendered.Bounds(), reference.Bounds(); got != want {
		t.Fatalf("the port rendered %v and PDFBox rendered %v", got, want)
	}

	differing, beyond := 0, 0
	for y := reference.Bounds().Min.Y; y < reference.Bounds().Max.Y; y++ {
		for x := reference.Bounds().Min.X; x < reference.Bounds().Max.X; x++ {
			worst := 0
			gotR, gotG, gotB, _ := rendered.At(x, y).RGBA()
			wantR, wantG, wantB, _ := reference.At(x, y).RGBA()
			for _, channel := range [][2]uint32{{gotR, wantR}, {gotG, wantG}, {gotB, wantB}} {
				delta := int(channel[0]>>8) - int(channel[1]>>8)
				if delta < 0 {
					delta = -delta
				}
				if delta > worst {
					worst = delta
				}
			}
			if worst == 0 {
				continue
			}
			differing++
			if worst > 64 {
				beyond++
			}
		}
	}

	if differing != differingPixels || beyond != beyondEdges {
		t.Errorf("%d of the page's pixels are not PDFBox's, %d of them by more "+
			"than a quarter of a channel; it was %d and %d",
			differing, beyond, differingPixels, beyondEdges)
	}
}

func readPNG(t *testing.T, path string) goimage.Image {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("reading PDFBox's render: %v", err)
	}
	defer file.Close()
	image, _, err := goimage.Decode(file)
	if err != nil {
		t.Fatalf("decoding PDFBox's render: %v", err)
	}
	return image
}
