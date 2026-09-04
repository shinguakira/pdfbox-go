package font

import (
	"sync"

	"github.com/shinguakira/pdfbox-go/go/fontbox"
)

// FontCache is an in-memory cache for system fonts. This allows PDFBox to
// manage caching for a FontProvider. PDFBox is free to purge this cache at
// will.
//
// Port of org.apache.pdfbox.pdmodel.font.FontCache.
//
// Java holds each font through a SoftReference, so the garbage collector may
// drop a cached font when memory runs short and the next lookup re-reads it
// from disk. Go has no soft reference and no equivalent hook, so the port holds
// the font outright: a cached font stays cached. Nothing observes the
// difference beyond memory use, since a dropped Java reference only ever costs
// a re-parse.
type FontCache struct {
	mu    sync.Mutex
	cache map[FontInfo]fontbox.FontBoxFont
}

// NewFontCache returns an empty cache.
func NewFontCache() *FontCache {
	return &FontCache{cache: map[FontInfo]fontbox.FontBoxFont{}}
}

// AddFont adds the given FontBox font to the cache.
func (c *FontCache) AddFont(info FontInfo, font fontbox.FontBoxFont) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[info] = font
}

// GetFont returns the FontBox font associated with the given FontInfo, or nil
// where the cache has none.
func (c *FontCache) GetFont(info FontInfo) fontbox.FontBoxFont {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cache[info]
}
