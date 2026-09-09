package raster_test

// Pages, rendered by PDFBox and by the port, compared.
//
// Everything else in this package looks at one call. This looks at the whole
// of both renderers: the same PDF goes into PDFBox's PDFRenderer -- its
// parser, its PageDrawer, Graphics2D underneath -- and into this port's, and
// the two images are put side by side.
//
// The PDFs are written by `testdata/genpdf.go` and `testdata/genpatterns.go`
// and checked in, so both sides read the same bytes; the PNGs beside them are
// what PDFBox made of those bytes, written by `testdata/RenderDrv.java`.
// Regenerate a PDF and its PNG together; the drivers say how.
//
// Neither page has text on it, deliberately. The glyph tests already compare
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

// comparePage renders a PDF and counts the pixels that are not PDFBox's.
//
// differing is all of them; beyond is the ones more than a quarter of one
// channel out, which on these pages is the stroke shift and nothing else.
func comparePage(t *testing.T, name string) (differing, beyond int) {
	t.Helper()
	reference := readPNG(t, "testdata/"+name+"-java.png")

	document, err := pdfbox.LoadPDF("testdata/" + name + ".pdf")
	if err != nil {
		t.Fatalf("loading %s: %v", name, err)
	}
	defer document.Close()

	rendered, err := raster.RenderPage(document, 0, 1, rendering.RGB)
	if err != nil {
		t.Fatalf("rendering %s: %v", name, err)
	}
	if got, want := rendered.Bounds(), reference.Bounds(); got != want {
		t.Fatalf("the port rendered %v and PDFBox rendered %v", got, want)
	}

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
	return differing, beyond
}

// TestAPageRendersAsPDFBoxRendersIt is the comparison the branch exists to
// make possible.
//
// Every flat fill on the page is exact -- the two rectangles, the even-odd
// one, the half-alpha overlap, the multiplied one -- and so is every interior,
// and **no pixel anywhere is more than a quarter of a channel out**. What
// differs is three things, all of them measured on their own in java2d_test.go
// and none of them a wrong shape:
//
//	strokes    1028 pixels, and 68 of them by more than one. The strokes are
//	           in the right place -- normalize.go puts them there, the way
//	           PDFBox's own hints do -- and what is left is the coverage a
//	           partly covered edge pixel is given.
//	shading    2220 pixels, up to 2. The colour table is indexed by a
//	           truncated product, and the last place of that product is not
//	           the same in float32 as it was in Java's float; the band drifts
//	           one step either way and never further.
//	diagonals    68 pixels, up to 6. The rotated square's edges, where the
//	           two rasterisers put different fractions on an edge pixel.
//
// The counts are pinned exactly and in both directions: ink that moves fails
// this, and ink that stops moving fails it too, so the record has to be
// updated with the code.
func TestAPageRendersAsPDFBoxRendersIt(t *testing.T) {
	const (
		differingPixels = 3265
		beyondEdges     = 0
	)
	differing, beyond := comparePage(t, "graphics")
	if differing != differingPixels || beyond != beyondEdges {
		t.Errorf("%d of the page's pixels are not PDFBox's, %d of them by more "+
			"than a quarter of a channel; it was %d and %d",
			differing, beyond, differingPixels, beyondEdges)
	}
}

// TestTilingPatternsRenderAsPDFBoxRendersThem is the paint a Backend cannot
// answer on its own, because the tile is a content stream and drawing it means
// going back through the PageDrawer.
//
// The page has four cases: a coloured pattern filled, an uncoloured one
// filled, a coloured one stroked with, and an uncoloured one filled through a
// clip. **The coloured fill is exact** -- 3036 pixels, none of them different
// -- and it is the one whose tile is nothing but axis-aligned rectangles, so
// it is the case that says the anchor rectangle, the tile raster, the repeat
// and the pattern matrix are all right.
//
// The other three draw the uncoloured tile, which is two diagonal strokes, so
// they carry the same edge coverage as everything else that is not
// axis-aligned -- 525 pixels of 19200, none of them more than a quarter of a
// channel out.
func TestTilingPatternsRenderAsPDFBoxRendersThem(t *testing.T) {
	const (
		differingPixels = 525
		beyondEdges     = 0
	)
	differing, beyond := comparePage(t, "patterns")
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

// TestSoftMasksRenderAsPDFBoxRendersThem is the deepest thing a Backend is
// asked for: the mask is a transparency group, so building one means running a
// second content stream through the drawer, into a surface of its own, at its
// own scale, and reading either its luminosity or its alpha back as an alpha
// channel.
//
// Two of the four cases are **exact**: the Alpha mask, which reads the group's
// alpha, and the Luminosity mask with a /BC backdrop, whose group covers only
// half its box so that the backdrop decides the rest. Between them they say
// that the group's box, its size, its origin, the backdrop fill and both
// channel readings are right.
//
// The other two are Luminosity masks and differ only in how bright a grey is:
//
//	over a grey group     2720 pixels, up to 6. The group's colours make the
//	                      round trip through DeviceGray and back, and come out
//	                      a step or two along.
//	over an RGB group     3360 pixels, up to 38. This is the one real gap.
//	                      isGray is false, so Java draws an ARGB image onto a
//	                      TYPE_BYTE_GRAY one, and that conversion is the JDK's
//	                      colour management -- an ICC transform through
//	                      CS_GRAY, whose grey diagonal is measurably
//	                      `1.055*x^(1/2.4)-0.055` and not the weighted sum its
//	                      name suggests. There is no ICC engine here and there
//	                      is not going to be one, so luma is the standard sRGB
//	                      luminance instead, and this is what that costs.
func TestSoftMasksRenderAsPDFBoxRendersThem(t *testing.T) {
	const (
		differingPixels = 6080
		beyondEdges     = 0
	)
	differing, beyond := comparePage(t, "masks")
	if differing != differingPixels || beyond != beyondEdges {
		t.Errorf("%d of the page's pixels are not PDFBox's, %d of them by more "+
			"than a quarter of a channel; it was %d and %d",
			differing, beyond, differingPixels, beyondEdges)
	}
}

// TestSoftMasksOnARotatedPageRenderAsPDFBoxRendersThem is the same four masks
// on a page with `/Rotate 90`, which is the one thing that makes
// PageDrawer.adjustImage do anything.
//
// adjustImage puts the mask back through the device transform with its own
// scaling taken out. For a page whose transform is a plain scale that is the
// identity and Java short-circuits, so every other fixture in this package
// takes the short circuit and the redraw was never run. A quarter turn makes
// it a rotation, and the redraw happens.
//
// **The counts are the unrotated page's, exactly.** The mask lands in the same
// place relative to what it masks however the page is turned, so what is left
// is the same colour conversion and nothing else. Writing this fixture found a
// defect: the redraw sampled at the destination pixel's corner rather than its
// centre, which moved every hard mask edge by up to a pixel.
func TestSoftMasksOnARotatedPageRenderAsPDFBoxRendersThem(t *testing.T) {
	const (
		differingPixels = 6080
		beyondEdges     = 0
	)
	differing, beyond := comparePage(t, "masksrot")
	if differing != differingPixels || beyond != beyondEdges {
		t.Errorf("%d of the page's pixels are not PDFBox's, %d of them by more "+
			"than a quarter of a channel; it was %d and %d",
			differing, beyond, differingPixels, beyondEdges)
	}
}
