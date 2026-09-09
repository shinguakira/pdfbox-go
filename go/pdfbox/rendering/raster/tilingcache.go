package raster

// One render of a tile per pattern, rather than one per fill.
//
// Port of rendering/TilingPaintFactory.java, which is a cache in front of the
// TilingPaint constructor. Java holds it in a `WeakHashMap` on the PageDrawer,
// so the entries live as long as the page is being drawn and the collector
// takes them afterwards.
//
// **Go has no weak reference, so the lifetime is written down instead.** The
// cache is a plain map on the surface, and `ClearTileCache` empties it; nothing
// in a render calls that, because a page's patterns are wanted for the whole
// page, and a caller who renders many documents through one surface can. What
// it is worth is what Java's is worth: a pattern filled a hundred times has its
// tile drawn once.
//
// The key is four of TilingPaintParameter's five fields: the pattern matrix,
// the device transform, the pattern's stream and the colour space. The fifth is
// the colour, and tilingKeyOf says why it is not here and why leaving it out
// changes nothing.

import (
	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/rendering"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// tilingKey is TilingPaintParameter: what makes two tiling paints the same.
//
// It is a comparable struct so that it can be a map key, which is what Java
// gets from equals and hashCode.
type tilingKey struct {
	// patternMatrix is `drawer.getInitialMatrix()` concatenated with the
	// pattern's own, which PageDrawer works out before the paint is built.
	patternMatrix [6]float32
	// transform is the device scale transform, Java's `xform`.
	transform [6]float64

	// pattern is the pattern's stream, which is what `patternDict` is.
	pattern *cos.Stream
	// colorSpace is the underlying space of an uncoloured pattern, and nil for
	// a coloured one.
	colorSpace color.PDColorSpace
}

// tilingKeyOf answers the key a paint caches under, and false where there is
// none, which is an **uncoloured** pattern or one with no stream to key on.
//
// Leaving out the uncoloured ones is what Java does, though not on purpose.
// TilingPaintParameter.hashCode ends with `this.color.hashCode()`; PDColor
// overrides neither hashCode nor equals, so that is its identity; and
// SetColor builds `new PDColor(array, colorSpace)` for every `scn`. So no two
// fills of an uncoloured pattern ever land in the same bucket and the cache
// never answers one. Its equals could not answer one either: it compares two
// colours with `this.color.toRGB()`, and the colour of an uncoloured pattern
// has the /Pattern colour space, whose toRGB throws
// UnsupportedOperationException -- which the `catch (IOException)` around it
// does not catch. See migration/JAVA-BUGS.md.
func tilingKeyOf(paint rendering.TilingPaint, transform *geom.AffineTransform) (tilingKey, bool) {
	if paint.Color != nil {
		return tilingKey{}, false
	}
	stream := paint.Pattern.ContentStream()
	if stream == nil {
		return tilingKey{}, false
	}
	key := tilingKey{
		patternMatrix: matrixValues(paint.PatternMatrix),
		colorSpace:    paint.ColorSpace,
	}
	if raw, isStream := stream.COSObject().(*cos.Stream); isStream {
		key.pattern = raw
	} else {
		return tilingKey{}, false
	}
	transform.GetMatrix(key.transform[:])
	return key, true
}

// matrixValues spreads a matrix into the six numbers that decide it.
func matrixValues(m *util.Matrix) [6]float32 {
	if m == nil {
		return [6]float32{}
	}
	return [6]float32{
		m.ScaleX(), m.ShearY(), m.ShearX(), m.ScaleY(), m.TranslateX(), m.TranslateY(),
	}
}

// cachedTilingSource is newTilingSource behind the cache.
func (i *Image) cachedTilingSource(paint rendering.TilingPaint) (paintSource, error) {
	key, cacheable := tilingKeyOf(paint, i.transform)
	if cacheable && i.tiles != nil {
		if cached, found := i.tiles[key]; found {
			return cached, nil
		}
	}
	source, err := i.newTilingSource(paint)
	if err != nil {
		return nil, err
	}
	if cacheable {
		if i.tiles == nil {
			i.tiles = map[tilingKey]*tilingSource{}
		}
		i.tiles[key] = source
	}
	return source, nil
}

// ClearTileCache drops the tiles rendered so far.
//
// Java has no such method: its cache is a WeakHashMap on a PageDrawer, and a
// PageDrawer is made for one page and dropped after it. A Backend here belongs
// to the caller and can outlive any number of pages, so the caller says when
// the tiles stop being worth keeping.
func (i *Image) ClearTileCache() { i.tiles = nil }
