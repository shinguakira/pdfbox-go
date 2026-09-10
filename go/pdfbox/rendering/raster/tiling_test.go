package raster

// The parts of TilingPaint the pattern page cannot reach.
//
// `page_test.go` renders four patterns against PDFBox and the coloured one is
// exact, which covers the anchor rectangle, the tile raster, the repeat and the
// pattern matrix. What it does not reach is the arithmetic at the edges: a
// method whose name lies, and the clamp PDFBOX-3653 added for a pattern that
// asks for a surface of billions of pixels.

import (
	goimage "image"
	goimagecolor "image/color"

	"math"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
)

// What TilingPaint.ceiling answers is no longer what tilingCeiling answers:
// JAVA-BUGS.md 85 is fixed, and javabug85_test.go holds both columns -- the
// corrected value and, beside each, what the Java gives. The test that pinned
// the Java's answers alone was here, and is there now.

// TestSignumIsJavas is Math.signum, which keeps the sign of a zero.
func TestSignumIsJavas(t *testing.T) {
	for _, c := range []struct {
		v, want float32
	}{
		{3, 1},
		{-3, -1},
		{0, 0},
		{float32(math.Copysign(0, -1)), float32(math.Copysign(0, -1))},
	} {
		got := signum(c.v)
		if got != c.want || math.Signbit(float64(got)) != math.Signbit(float64(c.want)) {
			t.Errorf("signum(%v) = %v, want %v", c.v, got, c.want)
		}
	}
}

// TestIsPositiveZeroIsFloatCompare is `Float.compare(v, 0) == 0`, which orders
// -0.0 below +0.0 and so answers zero for +0.0 alone.
func TestIsPositiveZeroIsFloatCompare(t *testing.T) {
	for _, c := range []struct {
		v    float32
		want bool
	}{
		{0, true},
		{float32(math.Copysign(0, -1)), false},
		{1, false},
		{-1, false},
		{float32(math.NaN()), false},
	} {
		if got := isPositiveZero(c.v); got != c.want {
			t.Errorf("isPositiveZero(%v) = %v, want %v", c.v, got, c.want)
		}
	}
}

// TestATileIsSampledWithTheFourTexelsAroundThePoint is TexturePaintContext's
// `filter`, which KEY_INTERPOLATION decides and PDFRenderer sets to BICUBIC.
//
// A tile that lands on whole pixels cannot show it -- the sample falls on a
// texel corner and the blend answers that texel, which is why `patterns.pdf`
// is exact with the filter on or off. This asks it of the sampler directly.
func TestATileIsSampledWithTheFourTexelsAroundThePoint(t *testing.T) {
	// Two texels across, black then white, so a sample halfway between them is
	// grey only if the four around it are blended.
	tile := goimage.NewNRGBA(goimage.Rect(0, 0, 2, 1))
	tile.SetNRGBA(0, 0, goimagecolor.NRGBA{A: 0xFF})
	tile.SetNRGBA(1, 0, goimagecolor.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF})

	source := &tilingSource{
		tile:     tile,
		anchor:   geom.NewRectangle2D(0, 0, 2, 1),
		toAnchor: geom.NewAffineTransform(1, 0, 0, 1, 0, 0),
		filter:   true,
	}

	// Halfway into the first texel: half of it and half of the second.
	if c, _ := source.colorAt(0, 0); c.R != 0 {
		t.Errorf("the texel's own corner is %d, want 0 -- a blend at weight zero "+
			"must answer the texel it landed in", c.R)
	}

	// With the filter off the same point answers the texel and nothing else,
	// wherever in it the sample fell.
	source.filter = false
	if c, _ := source.colorAt(1, 0); c.R != 0xFF {
		t.Errorf("unfiltered, the second texel is %d, want 255", c.R)
	}
	source.filter = true
	// Filtered, the second texel blends with the first, because the texture
	// repeats and the texel after the last is the first.
	if c, _ := source.colorAt(1, 0); c.R != 0xFF {
		t.Errorf("filtered at a texel corner, the second texel is %d, want 255", c.R)
	}
}

// TestAFilteredSampleBetweenTwoTexelsIsBetweenTheirColours is the same sampler
// asked at a point that is not a texel corner, which is the case a scaled tile
// puts it in.
func TestAFilteredSampleBetweenTwoTexelsIsBetweenTheirColours(t *testing.T) {
	tile := goimage.NewNRGBA(goimage.Rect(0, 0, 2, 1))
	tile.SetNRGBA(0, 0, goimagecolor.NRGBA{A: 0xFF})
	tile.SetNRGBA(1, 0, goimagecolor.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF})

	// An anchor four units wide over a two-texel tile, so one device pixel is
	// half a texel and pixel 1 lands in the middle of the first.
	source := &tilingSource{
		tile:     tile,
		anchor:   geom.NewRectangle2D(0, 0, 4, 1),
		toAnchor: geom.NewAffineTransform(1, 0, 0, 1, 0, 0),
		filter:   true,
	}

	// Pixel 1 is at 0.25 of the anchor, which is 0.5 of the way into texel 0.
	c, _ := source.colorAt(1, 0)
	if c.R != 128 {
		t.Errorf("halfway between black and white the sample is %d, want 128", c.R)
	}
	source.filter = false
	if c, _ := source.colorAt(1, 0); c.R != 0 {
		t.Errorf("unfiltered the same point is %d, want the texel it is in, 0", c.R)
	}
}
