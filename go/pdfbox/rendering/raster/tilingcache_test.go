package raster

// The cache in front of the tile.
//
// What it must not do is change a pixel, and `page_test.go` says so: the
// patterns page fills four times, three of them with the same two patterns,
// and its counts did not move when the cache went in. What is asserted here is
// that it is a cache at all -- that the second fill of a pattern does not
// render the tile again -- because nothing about the pixels can tell.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/pattern"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/rendering"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// aTilingPaint is a coloured tiling pattern over the given stream.
func aTilingPaint(t *testing.T) rendering.TilingPaint {
	t.Helper()
	p := pattern.NewPDTilingPattern(nil)
	p.SetPaintType(1)
	p.SetTilingType(1)
	p.SetBBox(common.NewPDRectangleOfSize(10, 10))
	p.SetXStep(10)
	p.SetYStep(10)

	out, err := p.ContentStream().CreateOutputStream()
	if err != nil {
		t.Fatalf("opening the pattern stream: %v", err)
	}
	if _, err := out.Write([]byte("1 0 0 rg 0 0 5 5 re f\n")); err != nil {
		t.Fatalf("writing the pattern stream: %v", err)
	}
	if err := out.Close(); err != nil {
		t.Fatalf("closing the pattern stream: %v", err)
	}

	return rendering.TilingPaint{
		Pattern:       p,
		Transform:     geom.NewAffineTransform(1, 0, 0, 1, 0, 0),
		PatternMatrix: util.NewMatrixOf(1, 0, 0, 1, 0, 0),
	}
}

// TestTheSamePatternKeysTheSame is TilingPaintParameter.equals: the key is the
// two matrices, the pattern's stream and the colour space, so the same pattern
// painted the same way is the same entry.
func TestTheSamePatternKeysTheSame(t *testing.T) {
	paint := aTilingPaint(t)
	identity := geom.NewAffineTransform(1, 0, 0, 1, 0, 0)

	first, ok := tilingKeyOf(paint, identity)
	if !ok {
		t.Fatal("the paint has no key")
	}
	// A second value naming the same pattern, built separately, as a second
	// `scn` on the same page would.
	same := rendering.TilingPaint{
		Pattern:       paint.Pattern,
		Transform:     geom.NewAffineTransform(1, 0, 0, 1, 0, 0),
		PatternMatrix: util.NewMatrixOf(1, 0, 0, 1, 0, 0),
	}
	second, _ := tilingKeyOf(same, geom.NewAffineTransform(1, 0, 0, 1, 0, 0))
	if first != second {
		t.Error("the same pattern painted the same way keys differently")
	}

	// A different transform is a different tile: the raster is sized by it.
	under, _ := tilingKeyOf(paint, geom.NewAffineTransform(2, 0, 0, 2, 0, 0))
	if first == under {
		t.Error("the same pattern under a different transform keys the same")
	}

	// So is a different pattern matrix.
	moved := paint
	moved.PatternMatrix = util.NewMatrixOf(1, 0, 0, 1, 5, 5)
	shifted, _ := tilingKeyOf(moved, identity)
	if first == shifted {
		t.Error("the same pattern under a different pattern matrix keys the same")
	}

	// And so is a different pattern.
	other, _ := tilingKeyOf(aTilingPaint(t), identity)
	if first == other {
		t.Error("two different patterns key the same")
	}
}

// TestAPatternsTileIsRenderedOnce is what the cache is for.
func TestAPatternsTileIsRenderedOnce(t *testing.T) {
	paint := aTilingPaint(t)
	i := NewImage(20, 20, rendering.RGB)

	// newTilingSource needs a drawer to run the tile's stream, and there is
	// none here, so the cache is exercised through its own two calls: the
	// second must answer the first's value without going near the drawer.
	key, ok := tilingKeyOf(paint, i.transform)
	if !ok {
		t.Fatal("the paint has no key")
	}
	rendered := &tilingSource{}
	i.tiles = map[tilingKey]*tilingSource{key: rendered}

	source, err := i.cachedTilingSource(paint)
	if err != nil {
		t.Fatalf("cachedTilingSource: %v", err)
	}
	if source != rendered {
		t.Error("the second fill rendered the tile again instead of taking the cached one")
	}

	i.ClearTileCache()
	if i.tiles != nil {
		t.Error("ClearTileCache left the tiles behind")
	}
}

// TestAnUncolouredPatternIsNotCached is what Java does without meaning to: its
// key ends with the colour's identity hash and every `scn` makes a new PDColor,
// so no two fills of an uncoloured pattern ever meet in the map.
func TestAnUncolouredPatternIsNotCached(t *testing.T) {
	paint := aTilingPaint(t)
	paint.Color = color.NewPDColorOfComponents([]float32{1, 0, 0}, color.DeviceRGB)
	if _, ok := tilingKeyOf(paint, geom.NewAffineTransform(1, 0, 0, 1, 0, 0)); ok {
		t.Error("an uncoloured pattern was given a key")
	}
}

// TestAPaintWithNoStreamIsNotCached is the other guard: a pattern read back
// from a dictionary has no stream to key on, and drawing it every time is
// better than keying two of them together.
func TestAPaintWithNoStreamIsNotCached(t *testing.T) {
	fromDictionary := pattern.NewPDTilingPatternOf(cos.NewDictionary())
	_, ok := tilingKeyOf(rendering.TilingPaint{
		Pattern:       fromDictionary,
		Transform:     geom.NewAffineTransform(1, 0, 0, 1, 0, 0),
		PatternMatrix: util.NewMatrixOf(1, 0, 0, 1, 0, 0),
	}, geom.NewAffineTransform(1, 0, 0, 1, 0, 0))
	if ok {
		t.Error("a pattern with no stream was given a key")
	}
}

// TestADifferentDeviceScaleKeysDifferently is Java's `xform` field: it decides
// how many pixels the tile is rasterized into, so two of them are two tiles.
func TestADifferentDeviceScaleKeysDifferently(t *testing.T) {
	paint := aTilingPaint(t)
	identity := geom.NewAffineTransform(1, 0, 0, 1, 0, 0)

	first, ok := tilingKeyOf(paint, identity)
	if !ok {
		t.Fatal("the paint has no key")
	}
	atTwice := paint
	atTwice.Transform = geom.NewAffineTransform(2, 0, 0, 2, 0, 0)
	second, _ := tilingKeyOf(atTwice, identity)
	if first == second {
		t.Error("the same pattern at two device scales keys the same")
	}
}
