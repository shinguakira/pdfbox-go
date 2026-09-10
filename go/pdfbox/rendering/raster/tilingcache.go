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
	"encoding/binary"
	"math"
	"strings"

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
	// transform is the transform the paint is drawn under, which the anchor
	// rectangle is mapped back through. Java has no such field, because its
	// TexturePaint is handed one afresh on every createContext; the port bakes
	// it into the source, so two fills under two transforms are two entries.
	transform [6]float64
	// deviceScale is Java's `xform`, the DPI transform, which decides how many
	// pixels the tile is rasterized into.
	deviceScale [6]float64

	// filter is whether the tile is sampled with the four texels around the
	// point or with the one it landed in, which KEY_INTERPOLATION decides. Java
	// has no such field, because its TexturePaint is handed the hints afresh on
	// every createContext; the port bakes the answer into the source.
	filter bool

	// pattern is the pattern's stream, which is what `patternDict` is.
	pattern *cos.Stream
	// colorSpace is the underlying space of an uncoloured pattern, and nil for
	// a coloured one.
	colorSpace color.PDColorSpace

	// components is the colour an uncoloured pattern is painted in, packed so
	// that the key stays comparable, and empty for a coloured one. It is what
	// JAVA-BUGS.md 86 is about: Java keys on the colour's identity instead.
	components string
}

// tilingKeyOf answers the key a paint caches under, and false where there is
// none, which is a pattern with no stream to key on.
//
// **An uncoloured pattern is keyed on its colour's components and their space,
// and Java keys it on the colour's identity.** See migration/JAVA-BUGS.md 86:
// TilingPaintParameter.hashCode ends with `this.color.hashCode()`, PDColor
// overrides neither hashCode nor equals, and SetColor builds a new PDColor for
// every `scn`, so no two fills of one uncoloured pattern ever land in the same
// bucket and Java's cache never answers. Its equals could not answer either:
// it compares two colours with `this.color.toRGB()`, and the colour of an
// uncoloured pattern has the /Pattern colour space, whose toRGB throws
// UnsupportedOperationException -- which the `catch (IOException)` around it
// does not catch. The components are what drawTilingPattern paints the tile
// with, so they are what decides whether two fills produce the same tile, and
// comparing them is also what keeps toRGB out of it.
func tilingKeyOf(paint rendering.TilingPaint, transform *geom.AffineTransform,
	transformFilter bool) (tilingKey, bool) {
	stream := paint.Pattern.ContentStream()
	if stream == nil {
		return tilingKey{}, false
	}
	key := tilingKey{
		patternMatrix: matrixValues(paint.PatternMatrix),
		colorSpace:    paint.ColorSpace,
	}
	if paint.Color != nil {
		key.components = componentsKey(paint.Color.Components())
	}
	if raw, isStream := stream.COSObject().(*cos.Stream); isStream {
		key.pattern = raw
	} else {
		return tilingKey{}, false
	}
	key.filter = transformFilter
	transform.GetMatrix(key.transform[:])
	if paint.Transform != nil {
		paint.Transform.GetMatrix(key.deviceScale[:])
	}
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
	key, cacheable := tilingKeyOf(paint, i.transform,
		i.interpolation != rendering.NearestNeighbor)
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

// componentsKey packs a colour's components into something comparable.
//
// A Go slice cannot be a map key and an array cannot hold a count that varies,
// so the numbers are written out. The exact bits are used rather than a
// rounding, because two colours that differ in the last place paint two tiles.
func componentsKey(components []float32) string {
	var packed strings.Builder
	for _, c := range components {
		var bytes [4]byte
		binary.BigEndian.PutUint32(bytes[:], math.Float32bits(c))
		packed.Write(bytes[:])
	}
	return packed.String()
}
