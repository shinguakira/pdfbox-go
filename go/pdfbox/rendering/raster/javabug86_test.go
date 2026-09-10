package raster

// JAVA-BUGS 86: `TilingPaintFactory`'s cache exists so that a pattern filled
// many times has its tile rendered once, and for an **uncoloured** pattern it
// never answers. Its key ends with `this.color.hashCode()`; `PDColor`
// overrides neither `hashCode` nor `equals`, so that is an identity hash; and
// `SetColor.process` builds `new PDColor(array, colorSpace)` for every `scn`.
// Two fills of the same pattern therefore carry two colours with two different
// hashes and land in two different buckets.
//
// **The expected behaviour here is not the Java's.** It comes from what the
// class is for, which its own name states: two fills that would produce the
// same tile are one entry. What distinguishes two fills of an uncoloured
// pattern is the colour's components and the space they are in -- the two
// things `drawTilingPattern` is handed and paints the tile with -- so those are
// what the key must hold, and comparing them directly is also what avoids the
// `toRGB` the Java's `equals` would have thrown on.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	pdfbox "github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/rendering"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// uncolouredPaintOf is the paint PageDrawer builds for an uncoloured pattern:
// the pattern, the underlying colour space, and the colour the `scn` carried.
//
// The pattern is passed in rather than built, because two `scn` operators on a
// page name **one** pattern -- it is the colour that is rebuilt each time, and
// that is the whole of the entry.
func uncolouredPaintOf(pattern rendering.TilingPaint,
	components []float32) rendering.TilingPaint {
	paint := pattern
	paint.ColorSpace = color.DeviceRGB
	paint.Color = color.NewPDColorOfComponents(components, color.DeviceRGB)
	return paint
}

// TestAnUncolouredPatternIsCachedByItsColour is the fix.
//
// Two `scn` operators naming the same pattern in the same colour build two
// PDColor values, as Java's do; they must still be one entry, because they
// paint the same tile.
func TestAnUncolouredPatternIsCachedByItsColour(t *testing.T) {
	identity := geom.NewAffineTransform(1, 0, 0, 1, 0, 0)

	pattern := aTilingPaint(t)
	first, ok := tilingKeyOf(uncolouredPaintOf(pattern, []float32{1, 0, 0}), identity, true)
	if !ok {
		t.Fatal("an uncoloured pattern has no key, so its tile is rendered per fill")
	}
	second, _ := tilingKeyOf(uncolouredPaintOf(pattern, []float32{1, 0, 0}), identity, true)
	if first != second {
		t.Error("the same pattern in the same colour keys differently, so the " +
			"cache cannot answer -- which is what the Java does")
	}
}

// TestAnUncolouredPatternInAnotherColourIsAnotherTile is the other half: the
// colour is what the tile is painted in, so two of them are two tiles and
// keying them together would paint one of the fills wrong.
func TestAnUncolouredPatternInAnotherColourIsAnotherTile(t *testing.T) {
	identity := geom.NewAffineTransform(1, 0, 0, 1, 0, 0)

	pattern := aTilingPaint(t)
	red, _ := tilingKeyOf(uncolouredPaintOf(pattern, []float32{1, 0, 0}), identity, true)
	blue, _ := tilingKeyOf(uncolouredPaintOf(pattern, []float32{0, 0, 1}), identity, true)
	if red == blue {
		t.Error("the same pattern in two colours keys the same, so one fill " +
			"would take the other's tile")
	}

	// And so is the same components in another space, which is the check
	// Java's equals makes before it compares the colours at all.
	inGray := pattern
	inGray.ColorSpace = color.DeviceGray
	inGray.Color = color.NewPDColorOfComponents([]float32{1, 0, 0}, color.DeviceGray)
	gray, _ := tilingKeyOf(inGray, identity, true)
	if red == gray {
		t.Error("the same components in two colour spaces key the same")
	}
}

// TestAColouredPatternStillHasNoColourInItsKey is the case Java gets right:
// getPaint passes null for the colour of a coloured pattern, so nothing about
// a colour may enter the key, and two fills of it are one entry.
func TestAColouredPatternStillHasNoColourInItsKey(t *testing.T) {
	identity := geom.NewAffineTransform(1, 0, 0, 1, 0, 0)

	paint := aTilingPaint(t)
	first, ok := tilingKeyOf(paint, identity, true)
	if !ok {
		t.Fatal("a coloured pattern has no key")
	}
	same := rendering.TilingPaint{
		Pattern:       paint.Pattern,
		Transform:     geom.NewAffineTransform(1, 0, 0, 1, 0, 0),
		PatternMatrix: util.NewMatrixOf(1, 0, 0, 1, 0, 0),
	}
	second, _ := tilingKeyOf(same, geom.NewAffineTransform(1, 0, 0, 1, 0, 0), true)
	if first != second {
		t.Error("two fills of one coloured pattern key differently")
	}
}

// TestAPageOfUncolouredPatternsRendersEachTileOnce is the entry end to end.
//
// `patterns.pdf` names two patterns and fills four times: the coloured one
// filled and stroked with, and the uncoloured one filled in two colours. The
// tiles those need are three -- one for the coloured pattern, and one for each
// colour of the uncoloured one -- so three is what the cache should hold when
// the page is done.
//
// Java holds **one**: the coloured pattern's. Its two uncoloured fills each
// build a PDColor whose identity hash is its own, so neither finds the other
// and both render a tile that is thrown away.
func TestAPageOfUncolouredPatternsRendersEachTileOnce(t *testing.T) {
	document, err := pdfbox.LoadPDF("testdata/patterns.pdf")
	if err != nil {
		t.Fatalf("loading the page: %v", err)
	}
	defer document.Close()

	renderer := rendering.NewPDFRenderer(document)
	width, height, surfaceType, err := renderer.SurfaceSizeOfPage(0, 1, rendering.RGB)
	if err != nil {
		t.Fatalf("SurfaceSizeOfPage: %v", err)
	}
	backend := NewImage(width, height, surfaceType)
	renderer.SetBackend(backend, false)
	if err := renderer.RenderImageOfType(0, 1, rendering.RGB); err != nil {
		t.Fatalf("rendering the page: %v", err)
	}

	const wantTiles = 3
	if got := len(backend.tiles); got != wantTiles {
		t.Errorf("the page rendered %d tiles, want %d -- the Java renders one "+
			"per fill for an uncoloured pattern and caches none of them",
			got, wantTiles)
	}
}
