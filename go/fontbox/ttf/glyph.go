package ttf

import (
	"fmt"
	"math"
	"sync"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/fontbox/util"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// How much of the glyf table is cached: no cache at all past maxCacheSize
// glyphs, and never more than maxCachedGlyphs of them.
const (
	maxCacheSize    = 5000
	maxCachedGlyphs = 100
)

// GlyphTable is the glyf table: the outlines of the glyphs.
//
// Port of org.apache.fontbox.ttf.GlyphTable.
type GlyphTable struct {
	Table

	glyphs    []*GlyphData
	data      DataStream
	loca      *IndexToLocationTable
	numGlyphs int
	cached    int
	hmt       *HorizontalMetricsTable
	maxp      *MaximumProfileTable

	// lockData stands for the synchronized block Java puts round the reads,
	// which share one cursor.
	lockData sync.Mutex
}

var _ TableReader = (*GlyphTable)(nil)

// Read takes a copy of the table, which the glyph reads then work through
// without touching the font's own cursor.
func (t *GlyphTable) Read(ttf *TrueTypeFont, data DataStream) error {
	loca, err := ttf.IndexToLocation()
	if err != nil {
		return err
	}
	t.loca = loca
	numGlyphs, err := ttf.NumberOfGlyphs()
	if err != nil {
		return err
	}
	t.numGlyphs = numGlyphs
	if t.numGlyphs < maxCacheSize {
		t.glyphs = make([]*GlyphData, t.numGlyphs)
	}
	dataBytes, err := readBytes(data, int(t.Length()))
	if err != nil {
		return err
	}
	read := pdfio.NewReadBufferBytes(dataBytes)
	t.data, err = NewRandomAccessReadDataStream(read)
	if err != nil {
		return err
	}
	if err := read.Close(); err != nil {
		return err
	}
	t.hmt, err = ttf.HorizontalMetrics()
	if err != nil {
		return err
	}
	t.maxp, err = ttf.MaximumProfile()
	if err != nil {
		return err
	}
	t.SetInitialized(true)
	return nil
}

// SetGlyphs sets the cached glyphs.
func (t *GlyphTable) SetGlyphs(glyphs []*GlyphData) { t.glyphs = glyphs }

// GetGlyph returns the outline of the given glyph, or nil where the font has no
// such glyph.
func (t *GlyphTable) GetGlyph(gid int) (*GlyphData, error) {
	return t.getGlyph(gid, 0)
}

// getGlyph returns the outline of the given glyph, level counting how deeply
// composite glyphs have nested to reach it.
func (t *GlyphTable) getGlyph(gid, level int) (*GlyphData, error) {
	if gid < 0 || gid >= t.numGlyphs {
		return nil, nil
	}
	if t.glyphs != nil && t.glyphs[gid] != nil {
		return t.glyphs[gid], nil
	}

	t.lockData.Lock()
	defer t.lockData.Unlock()
	return t.readGlyph(gid, level)
}

// getGlyphLocked is getGlyph with the lock over the cursor already held, which
// is how a composite glyph reaches the glyphs it is assembled from. Java's
// monitor is reentrant and a Go mutex is not.
func (t *GlyphTable) getGlyphLocked(gid, level int) (*GlyphData, error) {
	if gid < 0 || gid >= t.numGlyphs {
		return nil, nil
	}
	if t.glyphs != nil && t.glyphs[gid] != nil {
		return t.glyphs[gid], nil
	}
	return t.readGlyph(gid, level)
}

// readGlyph reads one glyph, with the lock over the cursor already held.
//
// A composite glyph reaches back into the table for its components, so this is
// also what that recursion calls; Java's monitor is reentrant and Go's mutex is
// not, which is why the locking sits in getGlyph rather than here.
func (t *GlyphTable) readGlyph(gid, level int) (*GlyphData, error) {
	var glyph *GlyphData
	offsets := t.loca.Offsets()
	if offsets[gid] == offsets[gid+1] || offsets[gid] == t.data.OriginalDataSize() {
		glyph = &GlyphData{}
		glyph.initEmptyData()
	} else {
		currentPosition := t.data.CurrentPosition()
		if err := t.data.SeekTo(offsets[gid]); err != nil {
			return nil, err
		}
		var err error
		glyph, err = t.getGlyphData(gid, level)
		if err != nil {
			return nil, err
		}
		if err := t.data.SeekTo(currentPosition); err != nil {
			return nil, err
		}
	}
	if t.glyphs != nil && t.glyphs[gid] == nil && t.cached < maxCachedGlyphs {
		t.glyphs[gid] = glyph
		t.cached++
	}
	return glyph, nil
}

// getGlyphData reads the glyph the cursor is sitting on.
func (t *GlyphTable) getGlyphData(gid, level int) (*GlyphData, error) {
	if level > t.maxp.MaxComponentDepth() {
		return nil, fmt.Errorf("ttf: composite glyph maximum level (%d) reached",
			t.maxp.MaxComponentDepth())
	}
	glyph := &GlyphData{}
	leftSideBearing := 0
	if t.hmt != nil {
		leftSideBearing = int(t.hmt.LeftSideBearing(gid))
	}
	if err := glyph.initData(t, t.data, leftSideBearing, level); err != nil {
		return nil, err
	}
	if glyph.Description().IsComposite() {
		glyph.Description().Resolve()
	}
	return glyph, nil
}

// GlyphData is the outline of one glyph.
//
// Port of org.apache.fontbox.ttf.GlyphData. Rendering the outline to a path is
// left to a later slice; see migration/STATUS.md.
type GlyphData struct {
	xMin             int16
	yMin             int16
	xMax             int16
	yMax             int16
	boundingBox      *util.BoundingBox
	numberOfContours int16
	glyphDescription GlyphDescription
}

// initData reads the glyph the cursor is sitting on.
func (g *GlyphData) initData(glyphTable *GlyphTable, data DataStream, leftSideBearing, level int) error {
	r := newReader(data)
	g.numberOfContours = r.signedShort()
	g.xMin = r.signedShort()
	g.yMin = r.signedShort()
	g.xMax = r.signedShort()
	g.yMax = r.signedShort()
	if r.err != nil {
		return r.err
	}
	g.boundingBox = util.NewBoundingBoxOf(
		float32(g.xMin), float32(g.yMin), float32(g.xMax), float32(g.yMax))

	if g.numberOfContours >= 0 {
		// number of contours == 0 is not defined by the spec, but some fonts
		// have it, so it is read as a simple glyph with no points
		x0 := int16(leftSideBearing) - g.xMin
		simple, err := newGlyfSimpleDescript(g.numberOfContours, data, x0)
		if err != nil {
			return err
		}
		g.glyphDescription = simple
	} else {
		composite, err := newGlyfCompositeDescript(data, glyphTable, level+1)
		if err != nil {
			return err
		}
		g.glyphDescription = composite
	}
	return nil
}

// initEmptyData gives the glyph an empty outline, which is what a glyph with no
// bytes of its own has.
func (g *GlyphData) initEmptyData() {
	g.glyphDescription = newEmptyGlyfSimpleDescript()
	g.boundingBox = util.NewBoundingBox()
}

// BoundingBox returns the box the outline sits in.
func (g *GlyphData) BoundingBox() *util.BoundingBox { return g.boundingBox }

// NumberOfContours returns how many contours the outline has, negative where
// the glyph is composite.
func (g *GlyphData) NumberOfContours() int16 { return g.numberOfContours }

// Description returns the outline itself.
func (g *GlyphData) Description() GlyphDescription { return g.glyphDescription }

// Path returns the path of the glyph.
func (g *GlyphData) Path() *geom.Path2D {
	return newGlyphRenderer(g.glyphDescription).Path()
}

// XMaximum returns the right edge of the outline.
func (g *GlyphData) XMaximum() int16 { return g.xMax }

// XMinimum returns the left edge of the outline.
func (g *GlyphData) XMinimum() int16 { return g.xMin }

// YMaximum returns the top edge of the outline.
func (g *GlyphData) YMaximum() int16 { return g.yMax }

// YMinimum returns the bottom edge of the outline.
func (g *GlyphData) YMinimum() int16 { return g.yMin }

// GlyphDescription is an outline, whether it is drawn directly or assembled
// from other glyphs.
//
// Port of org.apache.fontbox.ttf.GlyphDescription.
type GlyphDescription interface {
	// EndPtOfContours returns where the given contour ends.
	EndPtOfContours(i int) int
	// Flags returns the flags of the given point.
	Flags(i int) byte
	// XCoordinate returns the x of the given point.
	XCoordinate(i int) int16
	// YCoordinate returns the y of the given point.
	YCoordinate(i int) int16
	// IsComposite reports whether the outline is assembled from other glyphs.
	IsComposite() bool
	// PointCount returns how many points the outline has.
	PointCount() int
	// ContourCount returns how many contours the outline has.
	ContourCount() int
	// Resolve works out where the components of a composite outline sit.
	Resolve()
}

// The flags a point of a simple outline can carry.
const (
	OnCurve      byte = 0x01
	XShortVector byte = 0x02
	YShortVector byte = 0x04
	Repeat       byte = 0x08
	XDual        byte = 0x10
	YDual        byte = 0x20
)

// glyfDescript is what the two kinds of outline share.
//
// Port of org.apache.fontbox.ttf.GlyfDescript.
type glyfDescript struct {
	instructions []int
	contourCount int
}

// Resolve does nothing for an outline that is not composite.
func (d *glyfDescript) Resolve() {}

// ContourCount returns how many contours the outline has.
func (d *glyfDescript) ContourCount() int { return d.contourCount }

// Instructions returns the hinting instructions of the outline.
func (d *glyfDescript) Instructions() []int { return d.instructions }

// readInstructions reads the hinting instructions of the outline.
func (d *glyfDescript) readInstructions(bais DataStream, count int) error {
	instructions, err := readUnsignedByteArray(bais, count)
	if err != nil {
		return err
	}
	d.instructions = instructions
	return nil
}

// GlyfSimpleDescript is an outline drawn directly, as a list of points.
//
// Port of org.apache.fontbox.ttf.GlyfSimpleDescript.
type GlyfSimpleDescript struct {
	glyfDescript

	endPtsOfContours []int
	flags            []byte
	xCoordinates     []int16
	yCoordinates     []int16
	pointCount       int
}

var _ GlyphDescription = (*GlyfSimpleDescript)(nil)

// newEmptyGlyfSimpleDescript returns the outline of a glyph with no bytes of
// its own.
func newEmptyGlyfSimpleDescript() *GlyfSimpleDescript {
	return &GlyfSimpleDescript{glyfDescript: glyfDescript{contourCount: 0}}
}

// newGlyfSimpleDescript reads a simple outline.
func newGlyfSimpleDescript(numberOfContours int16, bais DataStream, x0 int16) (*GlyfSimpleDescript, error) {
	d := &GlyfSimpleDescript{
		glyfDescript: glyfDescript{contourCount: int(numberOfContours)},
	}
	if numberOfContours == 0 {
		return d, nil
	}

	r := newReader(bais)
	d.endPtsOfContours = r.unsignedShortArray(int(numberOfContours))
	if r.err != nil {
		return nil, r.err
	}
	lastEndPt := d.endPtsOfContours[numberOfContours-1]
	if numberOfContours == 1 && lastEndPt == 65535 {
		// PDFBOX-2939: assume an empty glyph
		return d, nil
	}
	d.pointCount = lastEndPt + 1

	d.flags = make([]byte, d.pointCount)
	d.xCoordinates = make([]int16, d.pointCount)
	d.yCoordinates = make([]int16, d.pointCount)

	instructionCount := r.unsignedShort()
	if r.err != nil {
		return nil, r.err
	}
	if err := d.readInstructions(bais, instructionCount); err != nil {
		return nil, err
	}
	if err := d.readFlags(d.pointCount, bais); err != nil {
		return nil, err
	}
	if err := d.readCoords(d.pointCount, bais, x0); err != nil {
		return nil, err
	}
	return d, nil
}

// EndPtOfContours returns where the given contour ends.
func (d *GlyfSimpleDescript) EndPtOfContours(i int) int { return d.endPtsOfContours[i] }

// Flags returns the flags of the given point.
func (d *GlyfSimpleDescript) Flags(i int) byte { return d.flags[i] }

// XCoordinate returns the x of the given point.
func (d *GlyfSimpleDescript) XCoordinate(i int) int16 { return d.xCoordinates[i] }

// YCoordinate returns the y of the given point.
func (d *GlyfSimpleDescript) YCoordinate(i int) int16 { return d.yCoordinates[i] }

// IsComposite reports that the outline is drawn directly.
func (d *GlyfSimpleDescript) IsComposite() bool { return false }

// PointCount returns how many points the outline has.
func (d *GlyfSimpleDescript) PointCount() int { return d.pointCount }

// readCoords reads the points, each stored as a step from the one before.
func (d *GlyfSimpleDescript) readCoords(count int, bais DataStream, x0 int16) error {
	x := x0
	y := int16(0)
	for i := 0; i < count; i++ {
		if d.flags[i]&XDual != 0 {
			if d.flags[i]&XShortVector != 0 {
				step, err := readUnsignedByte(bais)
				if err != nil {
					return err
				}
				x += int16(step)
			}
		} else {
			if d.flags[i]&XShortVector != 0 {
				step, err := readUnsignedByte(bais)
				if err != nil {
					return err
				}
				x -= int16(step)
			} else {
				step, err := readSignedShort(bais)
				if err != nil {
					return err
				}
				x += step
			}
		}
		d.xCoordinates[i] = x
	}

	for i := 0; i < count; i++ {
		if d.flags[i]&YDual != 0 {
			if d.flags[i]&YShortVector != 0 {
				step, err := readUnsignedByte(bais)
				if err != nil {
					return err
				}
				y += int16(step)
			}
		} else {
			if d.flags[i]&YShortVector != 0 {
				step, err := readUnsignedByte(bais)
				if err != nil {
					return err
				}
				y -= int16(step)
			} else {
				step, err := readSignedShort(bais)
				if err != nil {
					return err
				}
				y += step
			}
		}
		d.yCoordinates[i] = y
	}
	return nil
}

// readFlags reads the flags of the points, which run-length encode.
func (d *GlyfSimpleDescript) readFlags(flagCount int, bais DataStream) error {
	for index := 0; index < flagCount; index++ {
		flag, err := readUnsignedByte(bais)
		if err != nil {
			return err
		}
		d.flags[index] = byte(flag)
		if d.flags[index]&Repeat != 0 {
			repeats, err := readUnsignedByte(bais)
			if err != nil {
				return err
			}
			for i := 1; i <= repeats; i++ {
				if index+i >= len(d.flags) {
					return fmt.Errorf("ttf: repeat count (%d) higher than remaining space", repeats)
				}
				d.flags[index+i] = d.flags[index]
			}
			index += repeats
		}
	}
	return nil
}

// The flags a component of a composite outline can carry.
const (
	Arg1And2AreWords   int16 = 0x0001
	ArgsAreXYValues    int16 = 0x0002
	RoundXYToGrid      int16 = 0x0004
	WeHaveAScale       int16 = 0x0008
	MoreComponents     int16 = 0x0020
	WeHaveAnXAndYScale int16 = 0x0040
	WeHaveATwoByTwo    int16 = 0x0080
	WeHaveInstructions int16 = 0x0100
	UseMyMetrics       int16 = 0x0200
)

// GlyfCompositeComp is one component of a composite outline: another glyph,
// placed and possibly scaled.
//
// Port of org.apache.fontbox.ttf.GlyfCompositeComp.
type GlyfCompositeComp struct {
	firstIndex   int
	firstContour int
	argument1    int16
	argument2    int16
	flags        int16
	glyphIndex   int
	xscale       float64
	yscale       float64
	scale01      float64
	scale10      float64
	xtranslate   int
	ytranslate   int
	point1       int
	point2       int
}

// newGlyfCompositeComp reads one component of a composite outline.
func newGlyfCompositeComp(bais DataStream) (*GlyfCompositeComp, error) {
	c := &GlyfCompositeComp{xscale: 1.0, yscale: 1.0}
	r := newReader(bais)
	c.flags = r.signedShort()
	c.glyphIndex = r.unsignedShort() // number of glyph in a font is uint16

	if c.flags&Arg1And2AreWords != 0 {
		c.argument1 = r.signedShort()
		c.argument2 = r.signedShort()
	} else {
		c.argument1 = int16(r.signedByte())
		c.argument2 = int16(r.signedByte())
	}
	if r.err != nil {
		return nil, r.err
	}

	if c.flags&ArgsAreXYValues != 0 {
		c.xtranslate = int(c.argument1)
		c.ytranslate = int(c.argument2)
	} else {
		c.point1 = int(c.argument1)
		c.point2 = int(c.argument2)
	}

	switch {
	case c.flags&WeHaveAScale != 0:
		i := r.signedShort()
		c.xscale = float64(i) / float64(0x4000)
		c.yscale = c.xscale
	case c.flags&WeHaveAnXAndYScale != 0:
		i := r.signedShort()
		c.xscale = float64(i) / float64(0x4000)
		i = r.signedShort()
		c.yscale = float64(i) / float64(0x4000)
	case c.flags&WeHaveATwoByTwo != 0:
		i := r.signedShort()
		c.xscale = float64(i) / float64(0x4000)
		i = r.signedShort()
		c.scale01 = float64(i) / float64(0x4000)
		i = r.signedShort()
		c.scale10 = float64(i) / float64(0x4000)
		i = r.signedShort()
		c.yscale = float64(i) / float64(0x4000)
	}
	if r.err != nil {
		return nil, r.err
	}
	return c, nil
}

// SetFirstIndex sets where the points of the component start.
func (c *GlyfCompositeComp) SetFirstIndex(idx int) { c.firstIndex = idx }

// FirstIndex returns where the points of the component start.
func (c *GlyfCompositeComp) FirstIndex() int { return c.firstIndex }

// SetFirstContour sets where the contours of the component start.
func (c *GlyfCompositeComp) SetFirstContour(idx int) { c.firstContour = idx }

// FirstContour returns where the contours of the component start.
func (c *GlyfCompositeComp) FirstContour() int { return c.firstContour }

// Argument1 returns the first of the two placement arguments.
func (c *GlyfCompositeComp) Argument1() int16 { return c.argument1 }

// Argument2 returns the second of the two placement arguments.
func (c *GlyfCompositeComp) Argument2() int16 { return c.argument2 }

// Flags returns the flags of the component.
func (c *GlyfCompositeComp) Flags() int16 { return c.flags }

// GlyphIndex returns which glyph the component is.
func (c *GlyfCompositeComp) GlyphIndex() int { return c.glyphIndex }

// Scale01 returns the first off-diagonal of the transform.
func (c *GlyfCompositeComp) Scale01() float64 { return c.scale01 }

// Scale10 returns the second off-diagonal of the transform.
func (c *GlyfCompositeComp) Scale10() float64 { return c.scale10 }

// XScale returns how far the component is scaled horizontally.
func (c *GlyfCompositeComp) XScale() float64 { return c.xscale }

// YScale returns how far the component is scaled vertically.
func (c *GlyfCompositeComp) YScale() float64 { return c.yscale }

// XTranslate returns how far the component is moved horizontally.
func (c *GlyfCompositeComp) XTranslate() int { return c.xtranslate }

// YTranslate returns how far the component is moved vertically.
func (c *GlyfCompositeComp) YTranslate() int { return c.ytranslate }

// ScaleX transforms a point and returns its x.
func (c *GlyfCompositeComp) ScaleX(x, y int) int {
	return javaRound(float32(float64(x)*c.xscale + float64(y)*c.scale10))
}

// ScaleY transforms a point and returns its y.
func (c *GlyfCompositeComp) ScaleY(x, y int) int {
	return javaRound(float32(float64(x)*c.scale01 + float64(y)*c.yscale))
}

// javaRound rounds half up, as Java's Math.round(float) does, rather than to
// even as Go's math.Round does at a tie.
func javaRound(v float32) int {
	return int(math.Floor(float64(v) + 0.5))
}

// GlyfCompositeDescript is an outline assembled from other glyphs.
//
// Port of org.apache.fontbox.ttf.GlyfCompositeDescript.
type GlyfCompositeDescript struct {
	glyfDescript

	components            []*GlyfCompositeComp
	descriptions          map[int]GlyphDescription
	glyphTable            *GlyphTable
	beingResolved         bool
	resolved              bool
	pointCount            int
	compositeContourCount int
}

var _ GlyphDescription = (*GlyfCompositeDescript)(nil)

// newGlyfCompositeDescript reads a composite outline and the glyphs it is
// assembled from.
func newGlyfCompositeDescript(bais DataStream, glyphTable *GlyphTable, level int) (*GlyfCompositeDescript, error) {
	d := &GlyfCompositeDescript{
		glyfDescript:          glyfDescript{contourCount: -1},
		descriptions:          map[int]GlyphDescription{},
		glyphTable:            glyphTable,
		pointCount:            -1,
		compositeContourCount: -1,
	}

	var comp *GlyfCompositeComp
	for {
		var err error
		comp, err = newGlyfCompositeComp(bais)
		if err != nil {
			return nil, err
		}
		d.components = append(d.components, comp)
		if comp.Flags()&MoreComponents == 0 {
			break
		}
	}

	if comp.Flags()&WeHaveInstructions != 0 {
		instructionCount, err := readUnsignedShort(bais)
		if err != nil {
			return nil, err
		}
		if err := d.readInstructions(bais, instructionCount); err != nil {
			return nil, err
		}
	}
	d.initDescriptions(level)
	return d, nil
}

// Resolve works out where each component sits in the assembled outline.
func (d *GlyfCompositeDescript) Resolve() {
	if d.resolved {
		return
	}
	if d.beingResolved {
		// Circular reference in GlyfCompositeDesc
		return
	}
	d.beingResolved = true

	firstIndex := 0
	firstContour := 0
	for _, comp := range d.components {
		if firstIndex > math.MaxInt16 {
			// firstIndex has run away; stop resolving
			break
		}
		comp.SetFirstIndex(firstIndex)
		comp.SetFirstContour(firstContour)
		if desc, ok := d.descriptions[comp.GlyphIndex()]; ok {
			desc.Resolve()
			firstIndex += desc.PointCount()
			firstContour += desc.ContourCount()
		}
	}
	d.resolved = true
	d.beingResolved = false
}

// EndPtOfContours returns where the given contour of the assembled outline
// ends.
func (d *GlyfCompositeDescript) EndPtOfContours(i int) int {
	c := d.compositeCompEndPt(i)
	if c != nil {
		gd := d.descriptions[c.GlyphIndex()]
		return gd.EndPtOfContours(i-c.FirstContour()) + c.FirstIndex()
	}
	return 0
}

// Flags returns the flags of the given point of the assembled outline.
func (d *GlyfCompositeDescript) Flags(i int) byte {
	c := d.compositeComp(i)
	if c != nil {
		gd := d.descriptions[c.GlyphIndex()]
		return gd.Flags(i - c.FirstIndex())
	}
	return 0
}

// XCoordinate returns the x of the given point of the assembled outline.
func (d *GlyfCompositeDescript) XCoordinate(i int) int16 {
	c := d.compositeComp(i)
	if c != nil {
		gd := d.descriptions[c.GlyphIndex()]
		n := i - c.FirstIndex()
		x := int(gd.XCoordinate(n))
		y := int(gd.YCoordinate(n))
		return int16(c.ScaleX(x, y) + c.XTranslate())
	}
	return 0
}

// YCoordinate returns the y of the given point of the assembled outline.
func (d *GlyfCompositeDescript) YCoordinate(i int) int16 {
	c := d.compositeComp(i)
	if c != nil {
		gd := d.descriptions[c.GlyphIndex()]
		n := i - c.FirstIndex()
		x := int(gd.XCoordinate(n))
		y := int(gd.YCoordinate(n))
		return int16(c.ScaleY(x, y) + c.YTranslate())
	}
	return 0
}

// IsComposite reports that the outline is assembled from other glyphs.
func (d *GlyfCompositeDescript) IsComposite() bool { return true }

// PointCount returns how many points the assembled outline has.
func (d *GlyfCompositeDescript) PointCount() int {
	// Java logs when this is called before Resolve; the count it then works out
	// is whatever the unresolved components say.
	if d.pointCount < 0 {
		c := d.components[len(d.components)-1]
		gd, ok := d.descriptions[c.GlyphIndex()]
		if !ok {
			d.pointCount = 0
		} else {
			d.pointCount = c.FirstIndex() + gd.PointCount()
		}
	}
	return d.pointCount
}

// ContourCount returns how many contours the assembled outline has.
func (d *GlyfCompositeDescript) ContourCount() int {
	// Java logs when this is called before Resolve, as PointCount does.
	if d.compositeContourCount < 0 {
		c := d.components[len(d.components)-1]
		gd, ok := d.descriptions[c.GlyphIndex()]
		if !ok {
			d.compositeContourCount = 0
		} else {
			d.compositeContourCount = c.FirstContour() + gd.ContourCount()
		}
	}
	return d.compositeContourCount
}

// ComponentCount returns how many glyphs the outline is assembled from.
func (d *GlyfCompositeDescript) ComponentCount() int { return len(d.components) }

// Components returns the components of the outline.
func (d *GlyfCompositeDescript) Components() []*GlyfCompositeComp {
	out := make([]*GlyfCompositeComp, len(d.components))
	copy(out, d.components)
	return out
}

// compositeComp returns which component the given point of the assembled
// outline belongs to.
func (d *GlyfCompositeDescript) compositeComp(i int) *GlyfCompositeComp {
	for _, c := range d.components {
		gd, ok := d.descriptions[c.GlyphIndex()]
		if c.FirstIndex() <= i && ok && i < c.FirstIndex()+gd.PointCount() {
			return c
		}
	}
	return nil
}

// compositeCompEndPt returns which component the given contour of the assembled
// outline belongs to.
func (d *GlyfCompositeDescript) compositeCompEndPt(i int) *GlyfCompositeComp {
	for _, c := range d.components {
		gd, ok := d.descriptions[c.GlyphIndex()]
		if c.FirstContour() <= i && ok && i < c.FirstContour()+gd.ContourCount() {
			return c
		}
	}
	return nil
}

// initDescriptions reads the glyphs the outline is assembled from.
func (d *GlyfCompositeDescript) initDescriptions(level int) {
	for _, component := range d.components {
		index := component.GlyphIndex()
		// A component that cannot be read is left out, as it is in Java, which
		// logs the failure and carries on.
		glyph, err := d.glyphTable.getGlyphLocked(index, level)
		if err != nil {
			continue
		}
		if glyph != nil {
			d.descriptions[index] = glyph.Description()
		}
	}
}
