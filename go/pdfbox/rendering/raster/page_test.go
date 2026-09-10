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
	return comparePageOfType(t, name, name, rendering.RGB)
}

// comparePageOfType is comparePage for a page rendered as something other than
// RGB, where the reference is named for the type and the PDF is not.
func comparePageOfType(t *testing.T, reference, page string,
	imageType rendering.ImageType) (differing, beyond int) {
	t.Helper()
	referenceImage := readPNG(t, "testdata/"+reference+"-java.png")

	document, err := pdfbox.LoadPDF("testdata/" + page + ".pdf")
	if err != nil {
		t.Fatalf("loading %s: %v", page, err)
	}
	defer document.Close()

	rendered, err := raster.RenderPage(document, 0, 1, imageType)
	if err != nil {
		t.Fatalf("rendering %s: %v", page, err)
	}
	if got, want := rendered.Bounds(), referenceImage.Bounds(); got != want {
		t.Fatalf("the port rendered %v and PDFBox rendered %v", got, want)
	}

	for y := referenceImage.Bounds().Min.Y; y < referenceImage.Bounds().Max.Y; y++ {
		for x := referenceImage.Bounds().Min.X; x < referenceImage.Bounds().Max.X; x++ {
			worst := 0
			gotR, gotG, gotB, _ := rendered.At(x, y).RGBA()
			wantR, wantG, wantB, _ := referenceImage.At(x, y).RGBA()
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

// TestTheOtherImageTypesRenderAsPDFBoxRendersThem is `-color GRAY` and
// `-color BILEVEL`, which are what `ImageType.GRAY` and `ImageType.BINARY`
// mean.
//
// Java draws onto a BufferedImage of that type and the image quantizes what is
// written into it, so a later composite reads back what the surface really
// holds. quantize.go is that, at the same moment, and `asImageType` hands back
// the same kind of image ImageIO would write as a greyscale or one-bit PNG.
//
// **Grey is the RGB page's differences, collapsed.** 2182 pixels of 40000, none
// of them more than a quarter of a channel: the shading's colour drift is worth
// less once three channels become one.
//
// **Every difference in the one-bit page is a flipped pixel, and every one of
// them is on a stroke.** A surface with two colours in it has no way to be a
// little bit out: the one unit of edge coverage that the two rasterisers
// disagree about, which is worth 1 in 255 on the RGB page, is worth the whole
// pixel here. The fills, the shading and the rotated square are exact.
func TestTheOtherImageTypesRenderAsPDFBoxRendersThem(t *testing.T) {
	for _, c := range []struct {
		name      string
		kind      rendering.ImageType
		differing int
		beyond    int
	}{
		{"gray", rendering.Gray, 2182, 0},
		{"binary", rendering.Binary, 937, 937},
	} {
		t.Run(c.name, func(t *testing.T) {
			differing, beyond := comparePageOfType(t, "graphics-"+c.name, "graphics", c.kind)
			if differing != c.differing || beyond != c.beyond {
				t.Errorf("%d of the page's pixels are not PDFBox's, %d of them by more "+
					"than a quarter of a channel; it was %d and %d",
					differing, beyond, c.differing, c.beyond)
			}
		})
	}
}

// TestStencilsRenderAsPDFBoxRendersThem is `/ImageMask true`: one bit a sample,
// no colour space, and where the bit says so the colour in force shows and
// everywhere else nothing is painted at all.
//
// It is the one image the backend is handed with a paint beside it, and the
// only thing it may take from the image is which samples are in. Taking that
// from `ImageOfRegion` was wrong -- an image mask has no colour space, so what
// comes back is opaque wherever it comes back at all, and every stencil
// painted its whole rectangle. Against this page that was 3526 pixels of
// 16000; asking for the stencil image, whose alpha is the mask's bits and
// nothing else, makes it 101.
//
// The 101 are the sample boundaries. A stencil has no partial coverage, so
// where the two renderers put a boundary differently the pixel flips whole,
// which is why every one of them is more than a quarter of a channel.
func TestStencilsRenderAsPDFBoxRendersThem(t *testing.T) {
	const (
		differingPixels = 101
		beyondEdges     = 101
	)
	differing, beyond := comparePage(t, "stencil")
	if differing != differingPixels || beyond != beyondEdges {
		t.Errorf("%d of the page's pixels are not PDFBox's, %d of them by more "+
			"than a quarter of a channel; it was %d and %d",
			differing, beyond, differingPixels, beyondEdges)
	}
}

// TestAScaledTilingPatternRendersAsThePortMeansTo is the one page in this
// package whose numbers are **not** a claim that the port matches PDFBox.
//
// The pattern carries a `/Matrix` that scales by 1.37, so its 10-unit step is
// 13.7 device pixels and its tile does not land on whole ones. Two things
// separate the two renderers there, and only one of them is deliberate:
//
//	852 pixels  predate this branch. A tile stretched over a fraction of a
//	            pixel resamples differently here than TexturePaint does, and
//	            `patterns.pdf` could not show it because its tiles are 1:1.
//	            Sampling the pixel's centre rather than its corner was tried
//	            and is not the answer -- it takes that page from 525 differing
//	            pixels to 3876 -- so what it is remains open.
//	 64 pixels  are JAVA-BUGS.md 85, on purpose. TilingPaint.ceiling truncates,
//	            so Java rasterizes a 13.7-pixel tile 13 pixels wide and
//	            stretches it; the port rounds up, as the method's javadoc asks,
//	            and rasterizes 14.
//
// The count is pinned so that neither half moves unremarked.
func TestAScaledTilingPatternRendersAsThePortMeansTo(t *testing.T) {
	const (
		differingPixels = 916
		beyondEdges     = 677
	)
	differing, beyond := comparePage(t, "patternscale")
	if differing != differingPixels || beyond != beyondEdges {
		t.Errorf("%d of the page's pixels are not PDFBox's, %d of them by more "+
			"than a quarter of a channel; it was %d and %d",
			differing, beyond, differingPixels, beyondEdges)
	}
}
