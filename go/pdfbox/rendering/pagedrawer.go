package rendering

// What to draw, and where.
//
// Port of org.apache.pdfbox.rendering.PageDrawer.
//
// Every decision is here: which paint applies to the current colour, what the
// stroke is made of, what the clip intersects to, whether an optional content
// group is visible, whether an annotation is skipped, how far an image may be
// subsampled. What happens to those decisions is the Backend's; see backend.go
// and migration/STATUS.md.
//
// Four things in Java's PageDrawer are raster work end to end and are not here:
// the TransparencyGroup inner class, which renders into a BufferedImage --
// PushGroup and PopGroup stand for it; the pixel work of the
// stencil-mask-with-pattern arm of drawImage (dilateAlpha, the inverted lookup
// table, the per-pixel alpha combine of PDFBOX-6077 and PDFBOX-5403) -- the
// port decides "this stencil, that paint" and DrawStencil stands for the rest;
// applySoftMaskToPaint's building of the mask raster -- SoftMaskedPaint names
// the mask instead; and applyTransferFunction, which maps an image's pixels.

import (
	"log/slog"
	"math"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/documentinterchange/markedcontent"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/blend"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/form"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/image"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/pattern"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/shading"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/state"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// PageDrawer paints a page onto a Backend.
//
// Port of PageDrawer, which extends PDFGraphicsStreamEngine.
type PageDrawer struct {
	*contentstream.PDFGraphicsStreamEngine

	renderer *PDFRenderer

	subsamplingAllowed bool

	// backend is Java's graphics field, and is nil until DrawPage is called.
	backend             Backend
	xform               *geom.AffineTransform
	xformScalingFactorX float32
	xformScalingFactorY float32

	pageSize *common.PDRectangle

	flipTG bool

	clipWindingRule int
	linePath        *geom.Path2D

	lastClips    []*geom.Path2D
	hasLastClips bool

	initialClip *geom.Area

	textClippings []geom.Shape

	glyphCaches map[font.PDFont]*glyphCache

	nestedHiddenOCGCount int

	destination                           RenderDestination
	renderingHints                        RenderingHints
	imageDownscalingOptimizationThreshold float32
	blendModeMap                          map[cos.Base]bool

	annotationFilter annotation.AnnotationFilter
}

var (
	_ contentstream.StreamEngineOverrides         = (*PageDrawer)(nil)
	_ contentstream.GraphicsStreamEngineOverrides = (*PageDrawer)(nil)
)

// NewPageDrawer returns a drawer for the given parameters.
//
// Port of the PageDrawer(PageDrawerParameters) constructor, plus the operator
// registrations Java's PDFGraphicsStreamEngine constructor makes; the port's
// cannot, because a processor's package imports contentstream. See
// contentstream.NewPDFGraphicsStreamEngine.
func NewPageDrawer(parameters PageDrawerParameters) (*PageDrawer, error) {
	d := &PageDrawer{
		PDFGraphicsStreamEngine: contentstream.NewPDFGraphicsStreamEngine(parameters.Page()),

		renderer:                              parameters.Renderer(),
		subsamplingAllowed:                    parameters.IsSubsamplingAllowed(),
		destination:                           parameters.Destination(),
		renderingHints:                        parameters.RenderingHints(),
		imageDownscalingOptimizationThreshold: parameters.ImageDownscalingOptimizationThreshold(),

		clipWindingRule:  -1,
		linePath:         geom.NewPathFloat(),
		glyphCaches:      map[font.PDFont]*glyphCache{},
		blendModeMap:     map[cos.Base]bool{},
		annotationFilter: func(annotation.PDAnnotation) bool { return true },
	}
	d.SetOverrides(d)
	addAllOperators(d.PDFGraphicsStreamEngine)
	return d, nil
}

// AnnotationFilter returns the annotation filter.
func (d *PageDrawer) AnnotationFilter() annotation.AnnotationFilter { return d.annotationFilter }

// SetAnnotationFilter sets the annotation filter, so that only the annotations
// it accepts are rendered.
func (d *PageDrawer) SetAnnotationFilter(filter annotation.AnnotationFilter) {
	d.annotationFilter = filter
}

// Renderer returns the parent renderer.
func (d *PageDrawer) Renderer() *PDFRenderer { return d.renderer }

// Backend returns the backend being drawn on, which is nil until DrawPage has
// been called.
//
// Port of the protected getGraphics.
func (d *PageDrawer) Backend() Backend { return d.backend }

// LinePath returns the current line path, which is reset to empty after each
// fill or stroke.
//
// Port of the protected getLinePath.
func (d *PageDrawer) LinePath() *geom.Path2D { return d.linePath }

// setRenderingHints sets the drawer's rendering hints on the backend.
//
// Port of the private setRenderingHints, which calls addRenderingHints.
func (d *PageDrawer) setRenderingHints() {
	d.backend.SetAntiAliasing(d.renderingHints.AntiAliasing)
	d.backend.SetInterpolation(d.renderingHints.Interpolation)
}

// DrawPage draws the page onto the given backend.
//
// Port of drawPage(Graphics2D, PDRectangle).
func (d *PageDrawer) DrawPage(backend Backend, pageSize *common.PDRectangle) error {
	if backend == nil {
		return ErrNoBackend
	}
	d.backend = backend
	d.xform = backend.Transform()
	m := util.NewMatrixFromAffineTransform(d.xform)
	d.xformScalingFactorX = float32(math.Abs(float64(m.ScalingFactorX())))
	d.xformScalingFactorY = float32(math.Abs(float64(m.ScalingFactorY())))
	d.initialClip = backend.Clip()
	d.pageSize = pageSize

	d.setRenderingHints()

	at := backend.Transform().Clone()
	at.Translate(0, float64(pageSize.Height()))
	at.Scale(1, -1)

	// adjust for non-(0,0) crop box
	at.Translate(float64(-pageSize.LowerLeftX()), float64(-pageSize.LowerLeftY()))
	backend.SetTransform(at)

	if err := d.ProcessPage(d.Page()); err != nil {
		return err
	}

	for _, a := range d.Page().AnnotationsOfFilter(d.annotationFilter).ToSlice() {
		if err := d.ShowAnnotation(a); err != nil {
			return err
		}
	}

	d.backend = nil
	return nil
}

// DrawTilingPattern draws the pattern stream onto the given backend.
//
// Port of the package-private drawTilingPattern, which TilingPaint calls to
// render one tile.
func (d *PageDrawer) DrawTilingPattern(backend Backend, tilingPattern *pattern.PDTilingPattern,
	colorSpace color.PDColorSpace, c *color.PDColor, patternMatrix *util.Matrix) error {
	savedBackend := d.backend
	d.backend = backend

	savedLinePath := d.linePath
	d.linePath = geom.NewPathFloat()
	savedClipWindingRule := d.clipWindingRule
	d.clipWindingRule = -1

	savedLastClips, savedHasLastClips := d.lastClips, d.hasLastClips
	d.lastClips, d.hasLastClips = nil, false
	savedInitialClip := d.initialClip
	d.initialClip = nil

	savedFlipTG := d.flipTG
	d.flipTG = true

	d.setRenderingHints()
	err := d.ProcessTilingPatternMatrix(tilingPattern, c, colorSpace, patternMatrix)

	d.flipTG = savedFlipTG
	d.backend = savedBackend
	d.linePath = savedLinePath
	d.lastClips, d.hasLastClips = savedLastClips, savedHasLastClips
	d.initialClip = savedInitialClip
	d.clipWindingRule = savedClipWindingRule
	return err
}

// clampColor keeps a converted component inside 0..1.
func clampColor(c float32) float32 {
	if c < 0 {
		return 0
	}
	if c > 1 {
		return 1
	}
	return c
}

// Paint returns the paint for the given colour.
//
// Port of the protected getPaint(PDColor).
func (d *PageDrawer) Paint(c *color.PDColor) (Paint, error) {
	colorSpace := c.ColorSpace()
	switch {
	case colorSpace == nil: // PDFBOX-5782
		slog.Error("rendering: colorSpace is null, will be rendered as transparency")
		return transparentPaint, nil

	case isNoneSeparation(colorSpace):
		// PDFBOX-4900: "The special colorant name None shall not produce any visible output"
		// TODO better solution needs to be found for all occurences where toRGB is called
		return transparentPaint, nil

	case !isPatternSpace(colorSpace):
		rgb, err := colorSpace.ToRGB(c.Components())
		if err != nil {
			return nil, err
		}
		return ColorPaint{
			Red:   clampColor(rgb[0]),
			Green: clampColor(rgb[1]),
			Blue:  clampColor(rgb[2]),
			Alpha: 1,
		}, nil
	}

	patternSpace := colorSpace.(*pattern.PDPattern)
	found, err := patternSpace.Pattern(c)
	if err != nil {
		return nil, err
	}
	switch p := found.(type) {
	case *pattern.PDTilingPattern:
		if p.PaintType() == pattern.PaintColored {
			// colored tiling pattern
			return TilingPaint{Pattern: p, Transform: d.xform}, nil
		}
		// uncolored tiling pattern
		return TilingPaint{
			Pattern:    p,
			ColorSpace: patternSpace.UnderlyingColorSpace(),
			Color:      c,
			Transform:  d.xform,
		}, nil

	case *pattern.PDShadingPattern:
		sh, err := p.Shading()
		if err != nil {
			return nil, err
		}
		if sh == nil {
			slog.Error("rendering: shadingPattern is null, will be filled with transparency")
			return transparentPaint, nil
		}
		return ShadingPaint{
			Shading: sh,
			Matrix:  util.Concatenate(d.InitialMatrix(), p.Matrix()),
		}, nil
	}
	return transparentPaint, nil
}

// isNoneSeparation is Java's `colorSpace instanceof PDSeparation &&
// "None".equals(((PDSeparation) colorSpace).getColorantName())`.
func isNoneSeparation(colorSpace color.PDColorSpace) bool {
	separation, isSeparation := colorSpace.(*color.PDSeparation)
	if !isSeparation {
		return false
	}
	name, _ := separation.ColorantName()
	return name == "None"
}

// isPatternSpace is Java's `colorSpace instanceof PDPattern`.
func isPatternSpace(colorSpace color.PDColorSpace) bool {
	_, isPattern := colorSpace.(*pattern.PDPattern)
	return isPattern
}

// setClip sets the clipping path, caching it because intersecting is slow.
//
// Port of the protected final setClip. Java tracks lastClip manually because
// Graphics2D.getClip returns a new object rather than the one setClip was
// given; the port tracks it the same way, since a Backend may do the same.
func (d *PageDrawer) setClip() {
	clippingPaths := d.GraphicsState().CurrentClippingPaths()
	if !d.hasLastClips || !sameClips(clippingPaths, d.lastClips) {
		d.TransferClip(d.backend)
		if d.initialClip != nil {
			// apply the remembered initial clip, but transform it first
			// TODO see PDFBOX-4583
			_ = d.initialClip
		}
		d.lastClips, d.hasLastClips = clippingPaths, true
	}
}

// sameClips is Java's `clippingPaths != lastClips`, which compares the list
// identity rather than its contents; a Go slice cannot be compared, so this
// compares the backing array and the length, which is the same question.
func sameClips(a, b []*geom.Path2D) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 {
		return true
	}
	return &a[0] == &b[0]
}

// TransferClip transfers the clip to the destination device. An embedder
// replaces it to avoid slow intersecting and let the device do the work, for
// example when writing SVG; the individual clippings are on the graphics state.
// See PDFBOX-5258.
//
// Port of the protected transferClip(Graphics2D).
func (d *PageDrawer) TransferClip(backend Backend) {
	clippingPath := d.GraphicsState().CurrentClippingPath()
	if clippingPath.PathIterator(nil).IsDone() {
		// PDFBOX-4821: avoid bug with java printing that empty clipping path is ignored by
		// replacing with empty rectangle, works because this is not an empty path
		backend.SetClip(geom.NewAreaOfShape(geom.NewRectangle2D(0, 0, 0, 0)))
	} else {
		backend.SetClip(clippingPath)
	}
}

// BeginText handles BT.
func (d *PageDrawer) BeginText() error {
	d.setClip()
	d.beginTextClip()
	return nil
}

// EndText handles ET.
func (d *PageDrawer) EndText() error {
	d.endTextClip()
	return nil
}

// beginTextClip begins buffering the text clipping path, if any.
func (d *PageDrawer) beginTextClip() {
	// buffer the text clippings because they represents a single clipping area
	d.textClippings = []geom.Shape{}
}

// endTextClip ends buffering the text clipping path, if any.
func (d *PageDrawer) endTextClip() {
	gs := d.GraphicsState()
	renderingMode := gs.TextState().RenderingMode()

	// apply the buffered clip as one area
	if renderingMode.IsClip() && len(d.textClippings) > 0 {
		// PDFBOX-4150: this is much faster than using textClippingArea.add(new Area(glyph))
		// https://stackoverflow.com/questions/21519007/fast-union-of-shapes-in-java
		path := geom.NewPathFloatRule(geom.WindNonZero)
		for _, shape := range d.textClippings {
			path.Append(shape, false)
		}
		gs.IntersectClippingPath(path)
		d.textClippings = []geom.Shape{}

		// PDFBOX-3681: lastClip needs to be reset, because after intersection it is still the same
		// object, thus setClip() would believe that it is cached.
		d.lastClips, d.hasLastClips = nil, false
	}
}

// ShowFontGlyph draws one glyph of a font that is not Type 3.
func (d *PageDrawer) ShowFontGlyph(textRenderingMatrix *util.Matrix, f font.PDFont, code int,
	displacement util.Vector) error {
	at := textRenderingMatrix.CreateAffineTransform()
	at.Concatenate(f.FontMatrix().CreateAffineTransform())

	// create cache if it does not exist
	vectorFont, isVectorFont := f.(font.PDVectorFont)
	if !isVectorFont {
		return nil
	}
	cache, cached := d.glyphCaches[f]
	if !cached {
		cache = newGlyphCache(vectorFont)
		d.glyphCaches[f] = cache
	}

	path := cache.pathForCharacterCode(code)
	return d.drawGlyph(path, f, code, displacement, at)
}

// drawGlyph renders one glyph.
func (d *PageDrawer) drawGlyph(path *geom.Path2D, f font.PDFont, code int,
	displacement util.Vector, at *geom.AffineTransform) error {
	gs := d.GraphicsState()
	renderingMode := gs.TextState().RenderingMode()

	if path == nil {
		return nil
	}
	// Stretch non-embedded glyph if it does not match the height/width contained in the PDF.
	// Vertical fonts have zero X displacement, so the following code scales to 0 if we don't skip it.
	// TODO: How should vertical fonts be handled?
	hasExplicitWidth, err := f.HasExplicitWidth(code)
	if err != nil {
		return err
	}
	if !f.IsEmbedded() && !f.IsVertical() && !f.IsStandard14() && hasExplicitWidth {
		fontWidth, err := f.WidthFromFont(code)
		if err != nil {
			return err
		}
		if displacement.X() > 0 && // PDFBOX-5611: ignore zero widths
			fontWidth > 0 && // ignore spaces
			math.Abs(float64(fontWidth-displacement.X()*1000)) > 0.0001 {
			pdfWidth := displacement.X() * 1000
			at.Scale(float64(pdfWidth/fontWidth), 1)
		}
	}

	// render glyph
	glyph := path.CreateTransformedShape(at)

	if d.isContentRendered() {
		if renderingMode.IsFill() {
			d.backend.SetComposite(gs.BlendMode(), gs.NonStrokeAlphaConstant())
			paint, err := d.NonStrokingPaint()
			if err != nil {
				return err
			}
			d.backend.SetPaint(paint)
			d.setClip()
			if err := d.backend.Fill(glyph); err != nil {
				return err
			}
		}

		if renderingMode.IsStroke() {
			d.backend.SetComposite(gs.BlendMode(), gs.AlphaConstant())
			paint, err := d.strokingPaint()
			if err != nil {
				return err
			}
			d.backend.SetPaint(paint)
			d.backend.SetStroke(d.stroke())
			d.setClip()
			if err := d.backend.Draw(glyph); err != nil {
				return err
			}
		}
	}

	if renderingMode.IsClip() {
		d.textClippings = append(d.textClippings, glyph)
	}
	return nil
}

// ShowType3Glyph draws one glyph of a Type 3 font, which is a content stream of
// its own.
func (d *PageDrawer) ShowType3Glyph(textRenderingMatrix *util.Matrix, f *font.PDType3Font,
	code int, displacement util.Vector) error {
	renderingMode := d.GraphicsState().TextState().RenderingMode()
	if renderingMode != state.Neither {
		return d.PDFGraphicsStreamEngine.ShowType3Glyph(textRenderingMatrix, f, code, displacement)
	}
	return nil
}

// AppendRectangle appends a rectangle to the current path.
func (d *PageDrawer) AppendRectangle(p0, p1, p2, p3 geom.Point2D) error {
	// to ensure that the path is created in the right direction, we have to create
	// it by combining single lines instead of creating a simple rectangle
	d.linePath.MoveTo(p0.X(), p0.Y())
	d.linePath.LineTo(p1.X(), p1.Y())
	d.linePath.LineTo(p2.X(), p2.Y())
	d.linePath.LineTo(p3.X(), p3.Y())

	// close the subpath instead of adding the last line so that a possible set line
	// cap style isn't taken into account at the "beginning" of the rectangle
	d.linePath.ClosePath()
	return nil
}

// applySoftMaskToPaint wraps the given paint in the given soft mask.
//
// Port of the private applySoftMaskToPaint. Java renders the mask's
// transparency group into a gray raster here and hands it to a SoftMask paint;
// the port names the mask, and a Backend renders it. What is ported is the
// decision: a mask with no group is no mask at all, and an /Alpha or
// /Luminosity subtype is required.
func (d *PageDrawer) applySoftMaskToPaint(parentPaint Paint, softMask *state.PDSoftMask) Paint {
	if softMask == nil || softMask.Group() == nil {
		return parentPaint
	}
	subType := softMask.SubType()
	if subType != cos.Alpha && subType != cos.Luminosity {
		slog.Error("rendering: invalid soft mask subtype", "subtype", subType)
		return parentPaint
	}
	return SoftMaskedPaint{Paint: parentPaint, Mask: softMask}
}

// strokingPaint returns the paint a stroke is made with.
//
// Port of the private getStrokingPaint.
func (d *PageDrawer) strokingPaint() (Paint, error) {
	gs := d.GraphicsState()
	paint, err := d.Paint(gs.StrokingColor())
	if err != nil {
		return nil, err
	}
	return d.applySoftMaskToPaint(paint, gs.SoftMask()), nil
}

// NonStrokingPaint returns the paint a fill is made with. An embedder that
// overrides the glyph hooks needs it; see PDFBOX-5093.
//
// Port of the protected final getNonStrokingPaint.
func (d *PageDrawer) NonStrokingPaint() (Paint, error) {
	gs := d.GraphicsState()
	paint, err := d.Paint(gs.NonStrokingColor())
	if err != nil {
		return nil, err
	}
	return d.applySoftMaskToPaint(paint, gs.SoftMask()), nil
}

// stroke returns the stroke the current state describes, with the CTM applied.
//
// Port of the private getStroke.
func (d *PageDrawer) stroke() *Stroke {
	gs := d.GraphicsState()

	// apply the CTM
	lineWidth := d.TransformWidth(gs.LineWidth())

	// minimum line width as used by Adobe Reader
	if lineWidth < 0.25 {
		lineWidth = 0.25
	}

	dashPattern := gs.LineDashPattern()
	// PDFBOX-5168: show an all-zero dash array line invisible like Adobe does
	// must do it here because getDashArray() sets minimum width because of JVM bugs
	if isAllZeroDash(dashPattern.DashArray()) {
		return &Stroke{Invisible: true}
	}
	phaseStart := float32(dashPattern.Phase())
	dashArray := d.dashArray(dashPattern)
	phaseStart = d.TransformWidth(phaseStart)

	lineCap := min(2, max(0, gs.LineCap())) // legal values 0..2
	lineJoin := min(2, max(0, gs.LineJoin()))
	miterLimit := gs.MiterLimit()
	if miterLimit < 1 {
		slog.Warn("rendering: miter limit must be >= 1, value is ignored", "value", miterLimit)
		miterLimit = 10
	}
	return &Stroke{
		LineWidth:  lineWidth,
		LineCap:    lineCap,
		LineJoin:   lineJoin,
		MiterLimit: miterLimit,
		DashArray:  dashArray,
		DashPhase:  phaseStart,
	}
}

// isAllZeroDash reports whether the dash array is non-empty and every entry is
// zero.
func isAllZeroDash(dashArray []float32) bool {
	if len(dashArray) > 0 {
		for i := 0; i < len(dashArray); i++ {
			if dashArray[i] != 0 {
				return false
			}
		}
		return true
	}
	return false
}

// dashArray returns the dash array with the CTM applied, or nil where the
// pattern cannot be used.
//
// Port of the private getDashArray. Java writes the transformed widths back
// into the pattern's own array; the port writes into a copy, because the
// pattern is shared by every state cloned from this one and Java's aliasing
// there is a bug in its own right. See migration/JAVA-BUGS.md.
func (d *PageDrawer) dashArray(dashPattern *graphics.PDLineDashPattern) []float32 {
	dashArray := dashPattern.DashArray()
	// avoid empty, infinite and NaN values (PDFBOX-3360)
	if len(dashArray) == 0 {
		return nil
	}
	for i := 0; i < len(dashArray); i++ {
		if math.IsInf(float64(dashArray[i]), 0) || math.IsNaN(float64(dashArray[i])) {
			return nil
		}
	}
	transformed := make([]float32, len(dashArray))
	for i := 0; i < len(dashArray); i++ {
		// apply the CTM
		w := d.TransformWidth(dashArray[i])
		// minimum line dash width avoids JVM crash,
		// see PDFBOX-2373, PDFBOX-2929, PDFBOX-3204, PDFBOX-3813
		// also avoid 0 in array like "[ 0 1000 ] 0 d", see PDFBOX-3724
		if d.xformScalingFactorX < 0.5 {
			// PDFBOX-4492
			transformed[i] = max(w, 0.2)
		} else {
			transformed[i] = max(w, 0.062)
		}
	}
	return transformed
}

// StrokePath strokes the current path.
func (d *PageDrawer) StrokePath() error {
	if d.isContentRendered() {
		gs := d.GraphicsState()
		d.backend.SetComposite(gs.BlendMode(), gs.AlphaConstant())
		paint, err := d.strokingPaint()
		if err != nil {
			return err
		}
		d.backend.SetPaint(paint)
		d.backend.SetStroke(d.stroke())
		d.setClip()
		if err := d.backend.Draw(d.linePath); err != nil {
			return err
		}
	}
	d.linePath.Reset()
	return nil
}

// FillPath fills the current path under the given winding rule.
func (d *PageDrawer) FillPath(windingRule int) error {
	gs := d.GraphicsState()
	d.backend.SetComposite(gs.BlendMode(), gs.NonStrokeAlphaConstant())
	d.setClip()
	d.linePath.SetWindingRule(windingRule)

	// disable anti-aliasing for rectangular paths, this is a workaround to avoid small stripes
	// which occur when solid fills are used to simulate piecewise gradients, see PDFBOX-2302
	// note that we ignore paths with a width/height under 1 as these are fills used as strokes,
	// see PDFBOX-1658 for an example
	bounds := d.linePath.Bounds2D()
	noAntiAlias := isRectangular(d.linePath) && bounds.Width > 1 && bounds.Height > 1
	if noAntiAlias {
		d.backend.SetAntiAliasing(false)
	}

	var shape geom.Shape = d.linePath
	if isPatternSpace(gs.NonStrokingColorSpace()) {
		// apply clip to path to avoid oversized device bounds in shading contexts (PDFBOX-2901)
		area := geom.NewAreaOfShape(d.linePath)
		if clip := d.backend.Clip(); clip != nil {
			area.Intersect(clip)
		}
		if err := d.intersectShadingBBox(gs.NonStrokingColor(), area); err != nil {
			return err
		}
		shape = area
	}
	if d.isContentRendered() && !shape.PathIterator(nil).IsDone() {
		// creating Paint is sometimes a costly operation, so avoid if possible
		paint, err := d.NonStrokingPaint()
		if err != nil {
			return err
		}
		d.backend.SetPaint(paint)
		if err := d.backend.Fill(shape); err != nil {
			return err
		}
	}

	d.linePath.Reset()

	if noAntiAlias {
		// JDK 1.7 has a bug where rendering hints are reset by the above call to
		// the setRenderingHint method, so we re-set all hints, see PDFBOX-2302
		d.setRenderingHints()
	}
	return nil
}

// intersectShadingBBox intersects the given area with a shading pattern's
// transformed bounding box, where the colour is one.
//
// It is done here and not in the shading's raster, because the shading may have
// been rotated.
func (d *PageDrawer) intersectShadingBBox(c *color.PDColor, area *geom.Area) error {
	if !isPatternSpace(c.ColorSpace()) {
		return nil
	}
	pat, err := c.ColorSpace().(*pattern.PDPattern).Pattern(c)
	if err != nil {
		return err
	}
	shadingPattern, isShading := pat.(*pattern.PDShadingPattern)
	if !isShading {
		return nil
	}
	sh, err := shadingPattern.Shading()
	if err != nil {
		return err
	}
	if sh == nil {
		return nil
	}
	bbox := shadingBBox(sh)
	if bbox == nil {
		return nil
	}
	m := util.Concatenate(d.InitialMatrix(), shadingPattern.Matrix())
	area.Intersect(geom.NewAreaOfShape(bbox.Transform(m)))
	return nil
}

// isRectangular reports whether the given path is rectangular.
func isRectangular(path *geom.Path2D) bool {
	iter := path.PathIterator(nil)
	coords := make([]float64, 6)
	count := 0
	var xs, ys [4]int
	for !iter.IsDone() {
		switch iter.CurrentSegment(coords) {
		case geom.SegMoveTo:
			if count == 0 {
				xs[count] = int(math.Floor(coords[0]))
				ys[count] = int(math.Floor(coords[1]))
			} else {
				return false
			}
			count++

		case geom.SegLineTo:
			if count < 4 {
				xs[count] = int(math.Floor(coords[0]))
				ys[count] = int(math.Floor(coords[1]))
			} else {
				return false
			}
			count++

		case geom.SegCubicTo:
			return false
		}
		iter.Next()
	}

	if count == 4 {
		return xs[0] == xs[1] || xs[0] == xs[2] ||
			ys[0] == ys[1] || ys[0] == ys[3]
	}
	return false
}

// FillAndStrokePath fills and then strokes the path.
func (d *PageDrawer) FillAndStrokePath(windingRule int) error {
	// Cloning needed because FillPath resets linePath
	path := d.linePath.Clone()
	if err := d.FillPath(windingRule); err != nil {
		return err
	}
	d.linePath = path
	return d.StrokePath()
}

// Clip intersects the clipping path with the current path.
func (d *PageDrawer) Clip(windingRule int) error {
	// the clipping path will not be updated until the succeeding painting operator is called
	d.clipWindingRule = windingRule
	if d.clipWindingRule != -1 {
		d.linePath.SetWindingRule(d.clipWindingRule)

		if !d.linePath.PathIterator(nil).IsDone() {
			// PDFBOX-4949 / PDF.js 12306: don't clip if "W n" only
			d.GraphicsState().IntersectClippingPath(d.adjustClip(d.linePath))
		}

		// PDFBOX-3836: lastClip needs to be reset, because after intersection it is still the same
		// object, thus setClip() would believe that it is cached.
		d.lastClips, d.hasLastClips = nil, false

		d.clipWindingRule = -1
	}
	return nil
}

// MoveTo begins a new subpath.
func (d *PageDrawer) MoveTo(x, y float32) error {
	d.linePath.MoveTo(float64(x), float64(y))
	return nil
}

// LineTo appends a straight segment.
func (d *PageDrawer) LineTo(x, y float32) error {
	d.linePath.LineTo(float64(x), float64(y))
	return nil
}

// CurveTo appends a cubic Bezier curve.
func (d *PageDrawer) CurveTo(x1, y1, x2, y2, x3, y3 float32) error {
	d.linePath.CurveTo(float64(x1), float64(y1), float64(x2), float64(y2),
		float64(x3), float64(y3))
	return nil
}

// CurrentPoint returns the current point of the path, or nil.
func (d *PageDrawer) CurrentPoint() (geom.Point2D, error) {
	return d.linePath.CurrentPoint(), nil
}

// ClosePath closes the current subpath.
func (d *PageDrawer) ClosePath() error {
	d.linePath.ClosePath()
	return nil
}

// EndPath ends the path without painting it.
func (d *PageDrawer) EndPath() error {
	d.linePath.Reset()
	return nil
}

// adjustClip widens a clip that would be thinner than a device pixel.
//
// PDFBOX-5715 / PR#73: this was added to fix a problem with missing fine lines
// when printing on MacOS. Lines vanish because CPrinterJob sets graphics scale
// to 1 for Printable so after scaling lines often have a width smaller than 1
// after scaling and clipping. This change enlarges the clip bounds to cover at
// least 1 point plus 0.5 on one and another side in the device space to allow
// to draw the linePath inside the clip. The linePath can consists from
// different lines but when its bounds width or height is less than 1.0 it seems
// safe to use a rectangle as a clip instead of the real path.
//
// Java asks AffineTransform.getType for the two cases; the port asks the matrix
// itself, since Go has no such bitmask. See migration/STATUS.md.
func (d *PageDrawer) adjustClip(linePath *geom.Path2D) *geom.Path2D {
	tx := d.backend.Transform()

	if isTranslationOrFlipOnly(tx) {
		return linePath
	}
	if isAxisAligned(tx) {
		sx := math.Abs(tx.ScaleX())
		sy := math.Abs(tx.ScaleY())
		if sx > 1.0 && sy > 1.0 {
			return linePath
		}

		b := linePath.Bounds()
		bounds := geom.NewRectangle2D(float64(b.X), float64(b.Y),
			float64(b.Width), float64(b.Height))
		w := bounds.Width
		h := bounds.Height
		sw := sx * w
		sh := sy * h
		const minSize = 2.0
		if sw < minSize || sh < minSize {
			x := bounds.X
			y := bounds.Y
			if sw < minSize {
				w = minSize / sx
				x = bounds.CenterX() - w/2
			}
			if sh < minSize {
				h = minSize / sy
				y = bounds.CenterY() - h/2
			}
			return geom.NewPathDoubleShape(geom.NewRectangle2D(x, y, w, h))
		}
	}
	return linePath
}

// isTranslationOrFlipOnly is Java's `(type & ~(TYPE_TRANSLATION | TYPE_FLIP))
// == 0`: no shear and no rotation, and both scale factors exactly ±1.
func isTranslationOrFlipOnly(tx *geom.AffineTransform) bool {
	return tx.ShearX() == 0 && tx.ShearY() == 0 &&
		math.Abs(tx.ScaleX()) == 1 && math.Abs(tx.ScaleY()) == 1
}

// isAxisAligned is Java's `(type & ~(TYPE_TRANSLATION | TYPE_FLIP |
// TYPE_MASK_SCALE)) == 0`: no shear and no rotation, any scale.
func isAxisAligned(tx *geom.AffineTransform) bool {
	return tx.ShearX() == 0 && tx.ShearY() == 0
}

// DrawImage draws the given image.
func (d *PageDrawer) DrawImage(pdImage image.PDImage) error {
	if xobject, isXObject := pdImage.(*image.PDImageXObject); isXObject &&
		d.isHiddenOCG(xobject.OptionalContent()) {
		return nil
	}
	if !d.isContentRendered() {
		return nil
	}
	gs := d.GraphicsState()
	ctm := gs.CurrentTransformationMatrix()
	at := ctm.CreateAffineTransform()

	subsampling := 1
	if d.subsamplingAllowed {
		subsampling = d.Subsampling(pdImage, at)
	}

	if !pdImage.Interpolate() {
		// if the image is scaled down, we use smooth interpolation, eg PDFBOX-2364
		// only when scaled up do we use nearest neighbour, eg PDFBOX-2302 / mori-cvpr01.pdf
		// PDFBOX-4930: we use the sizes of the ARGB image. These can be different
		// than the original sizes of the base image, when the mask is bigger.
		// PDFBOX-5091: also consider subsampling, the sizes are different too.
		bim, err := pdImage.ImageOfRegion(nil, subsampling)
		if err != nil {
			return err
		}
		size := bim.Bounds()
		isScaledUp := size.Dx() <= absRound(ctm.ScalingFactorX()*d.xformScalingFactorX) ||
			size.Dy() <= absRound(ctm.ScalingFactorY()*d.xformScalingFactorY)
		if isScaledUp {
			d.backend.SetInterpolation(NearestNeighbor)
		}
	}

	d.backend.SetComposite(gs.BlendMode(), gs.NonStrokeAlphaConstant())
	d.setClip()

	var err error
	if pdImage.IsStencil() {
		// The stencil is filled with the non-stroking paint, whether that is a
		// pattern or a plain colour. Java splits the two: a pattern is rendered
		// into a scratch image and combined with the mask by hand, everything
		// else goes through PDImage.getStencilImage. Both end as this one call;
		// the pixel work of the first is the Backend's. See the file comment.
		paint, paintErr := d.NonStrokingPaint()
		if paintErr != nil {
			return paintErr
		}
		err = d.backend.DrawStencil(pdImage, at, paint)
	} else {
		err = d.backend.DrawImage(pdImage, at, subsampling)
	}
	if err != nil {
		return err
	}

	if !pdImage.Interpolate() {
		// JDK 1.7 has a bug where rendering hints are reset by the above call to
		// the setRenderingHint method, so we re-set all hints, see PDFBOX-2302
		d.setRenderingHints()
	}
	return nil
}

// absRound is Java's `Math.abs(Math.round(x))`.
func absRound(value float32) int {
	return int(math.Abs(math.Round(float64(value))))
}

// Subsampling returns how far the given image may be subsampled and still fill
// the area it is drawn into, which is at least 1.
//
// Port of the protected getSubsampling.
func (d *PageDrawer) Subsampling(pdImage image.PDImage, at *geom.AffineTransform) int {
	// calculate subsampling according to the resulting image size
	scale := math.Abs(at.Determinant() * d.xform.Determinant())

	imageWidth := pdImage.Width()
	imageHeight := pdImage.Height()
	subsampling := int(math.Floor(math.Sqrt(float64(imageWidth*imageHeight) / scale)))
	if subsampling > 8 {
		subsampling = 8
	}
	if subsampling < 1 {
		subsampling = 1
	}
	if subsampling > imageWidth || subsampling > imageHeight {
		// For very small images it is possible that the subsampling would imply 0 size.
		// To avoid problems, the subsampling is set to no less than the smallest dimension.
		subsampling = min(imageWidth, imageHeight)
	}
	return subsampling
}

// ShadingFill paints the named shading over the clip.
func (d *PageDrawer) ShadingFill(shadingName *cos.Name) error {
	if !d.isContentRendered() {
		return nil
	}
	sh, err := d.Resources().GetShading(shadingName)
	if err != nil {
		return err
	}
	if sh == nil {
		slog.Error("rendering: shading does not exist in resources dictionary",
			"shading", shadingName)
		return nil
	}
	gs := d.GraphicsState()
	ctm := gs.CurrentTransformationMatrix()

	d.backend.SetComposite(gs.BlendMode(), gs.NonStrokeAlphaConstant())
	savedClip := d.backend.Clip()
	d.backend.SetClip(nil)
	d.lastClips, d.hasLastClips = nil, false

	// get the transformed BBox and intersect with current clipping path
	// need to do it here and not in shading getRaster() because it may have been rotated
	bbox := shadingBBox(sh)
	currentClippingPath := gs.CurrentClippingPath()
	var area *geom.Area
	if bbox != nil {
		area = geom.NewAreaOfShape(bbox.Transform(ctm))
		area.Intersect(currentClippingPath)
	} else {
		bounds, err := sh.Bounds(geom.NewAffineTransform(1, 0, 0, 1, 0, 0), ctm)
		if err != nil {
			return err
		}
		if bounds != nil {
			bounds.Add(math.Floor(bounds.MinX()-1), math.Floor(bounds.MinY()-1))
			bounds.Add(math.Ceil(bounds.MaxX()+1), math.Ceil(bounds.MaxY()+1))
			area = geom.NewAreaOfShape(bounds)
			area.Intersect(currentClippingPath)
		} else {
			area = currentClippingPath
		}
	}
	if !area.IsEmpty() {
		// creating Paint is sometimes a costly operation, so avoid if possible
		var paint Paint = ShadingPaint{Shading: sh, Matrix: ctm}
		paint = d.applySoftMaskToPaint(paint, gs.SoftMask())
		d.backend.SetPaint(paint)
		if err := d.backend.Fill(area); err != nil {
			return err
		}
	}
	d.backend.SetClip(savedClip)
	return nil
}

// shadingBBox is `shading.getBBox()`, which only the concrete shadings that
// carry the state have; the port's Shading interface does not name it.
func shadingBBox(sh shading.Shading) *common.PDRectangle {
	if boxed, hasBox := sh.(interface{ BBox() *common.PDRectangle }); hasBox {
		return boxed.BBox()
	}
	return nil
}

// ShowAnnotation shows one annotation of the page.
func (d *PageDrawer) ShowAnnotation(a annotation.PDAnnotation) error {
	d.lastClips, d.hasLastClips = nil, false

	if d.shouldSkipAnnotation(a) {
		return nil
	}

	// TODO support NoZoom, example can be found in p5 of PDFBOX-2348
	appearance := a.Appearance()
	if appearance == nil || appearance.NormalAppearance() == nil {
		if err := a.ConstructAppearancesInDocument(d.renderer.Document()); err != nil {
			return err
		}
	}
	if a.IsNoRotate() && d.CurrentPage().Rotation() != 0 {
		appearance = a.Appearance()
		if appearance != nil {
			appearanceEntry := appearance.NormalAppearance()
			if appearanceEntry != nil && appearanceEntry.IsStream() {
				hasTransparency, err := d.hasTransparency(
					&appearanceEntry.AppearanceStream().PDFormXObject)
				if err != nil {
					return err
				}
				if hasTransparency {
					// PDFBOX-4744: avoid appearances with transparency groups until we have fixed
					// the rendering. A real solution should probably be
					// in PDFStreamEngine.processAnnotation().
					if err := a.ConstructAppearances(); err != nil {
						return err
					}
				}
			}
		}
		rect := a.Rectangle()
		savedTransform := d.backend.Transform()
		// "The upper-left corner of the annotation remains at the same point in
		//  default user space; the annotation pivots around that point."
		rotated := savedTransform.Clone()
		rotated.Translate(float64(rect.LowerLeftX()), float64(rect.UpperRightY()))
		rotated.Rotate(float64(d.CurrentPage().Rotation()) * math.Pi / 180)
		rotated.Translate(float64(-rect.LowerLeftX()), float64(-rect.UpperRightY()))
		d.backend.SetTransform(rotated)
		err := d.PDFGraphicsStreamEngine.ShowAnnotation(a)
		d.backend.SetTransform(savedTransform)
		a.SetAppearance(appearance) // restore
		return err
	}
	return d.PDFGraphicsStreamEngine.ShowAnnotation(a)
}

// shouldSkipAnnotation reports whether the given annotation is not drawn at
// this destination.
func (d *PageDrawer) shouldSkipAnnotation(a annotation.PDAnnotation) bool {
	if d.destination == Print && !a.IsPrinted() {
		return true
	}
	if (d.destination == View || d.destination == Export) && a.IsNoView() {
		return true
	}
	if a.IsHidden() {
		return true
	}
	if _, isUnknown := a.(*annotation.PDAnnotationUnknown); a.IsInvisible() && isUnknown {
		// "If set, do not display the annotation if it does not belong to one
		// of the standard annotation types and no annotation handler is available."
		return true
	}
	return d.isHiddenOCG(a.OptionalContent())
}

// hasTransparency reports whether the given form, or any form inside it, holds
// a transparency group.
func (d *PageDrawer) hasTransparency(f *form.PDFormXObject) (bool, error) {
	if f == nil {
		return false, nil
	}
	resources, isResources := f.Resources().(*pdmodel.PDResources)
	if !isResources || resources == nil {
		return false, nil
	}
	for _, name := range resources.XObjectNames() {
		xObject, err := resources.GetXObject(name)
		if err != nil {
			return false, err
		}
		if _, isGroup := xObject.(*form.PDTransparencyGroup); isGroup {
			return true, nil
		}
		if nested, isForm := xObject.(*form.PDFormXObject); isForm {
			has, err := d.hasTransparency(nested)
			if err != nil {
				return false, err
			}
			if has {
				return true, nil
			}
		}
	}
	return false, nil
}

// ShowForm runs a form XObject's content stream.
func (d *PageDrawer) ShowForm(f *form.PDFormXObject) error {
	if d.isHiddenOCG(f.OptionalContent()) {
		return nil
	}
	if d.isContentRendered() {
		savedLinePath := d.linePath
		d.linePath = geom.NewPathFloat()
		err := d.PDFGraphicsStreamEngine.ShowForm(f)
		d.linePath = savedLinePath
		return err
	}
	return nil
}

// ShowTransparencyGroup runs a transparency group's content stream into a
// compositing layer of its own.
func (d *PageDrawer) ShowTransparencyGroup(group *form.PDTransparencyGroup) error {
	return d.ShowTransparencyGroupOnBackend(group, d.backend)
}

// ShowTransparencyGroupOnBackend runs a transparency group onto the given
// backend, which lets an embedder extract it into a separate device.
//
// Port of the protected showTransparencyGroupOnGraphics. Java builds the
// group's raster in the TransparencyGroup inner class and then draws it; the
// port works out the same box and hands the layer to the Backend.
func (d *PageDrawer) ShowTransparencyGroupOnBackend(group *form.PDTransparencyGroup,
	backend Backend) error {
	if d.isHiddenOCG(group.OptionalContent()) {
		return nil
	}
	if !d.isContentRendered() {
		return nil
	}
	gs := d.GraphicsState()
	bbox := d.transparencyGroupBox(group, gs.CurrentTransformationMatrix())
	if bbox == nil {
		// group is empty, don't bother
		return nil
	}

	needsBackdrop := !group.Group().IsIsolated() && d.hasBlendMode(group, map[cos.Base]bool{})

	backend.SetComposite(gs.BlendMode(), gs.NonStrokeAlphaConstant())
	d.setClip()

	if err := backend.PushGroup(bbox, false, needsBackdrop, nil); err != nil {
		return err
	}

	savedFlipTG := d.flipTG
	d.flipTG = false
	savedPageSize := d.pageSize
	d.pageSize = bbox
	savedClipWindingRule := d.clipWindingRule
	d.clipWindingRule = -1
	savedLinePath := d.linePath
	d.linePath = geom.NewPathFloat()
	savedBackend := d.backend
	d.backend = backend

	err := d.ProcessTransparencyGroup(group)

	d.backend = savedBackend
	d.linePath = savedLinePath
	d.clipWindingRule = savedClipWindingRule
	d.pageSize = savedPageSize
	d.flipTG = savedFlipTG
	if err != nil {
		return err
	}

	if softMask := gs.SoftMask(); softMask != nil {
		backend.SetPaint(d.applySoftMaskToPaint(nil, softMask))
	}
	return backend.PopGroup()
}

// transparencyGroupBox returns the box a transparency group is composited over,
// in user space, and nil where it is empty.
//
// Port of the part of the TransparencyGroup constructor that decides the box:
// the form's bbox through the CTM and the form matrix, clipped to the current
// clipping path. What Java does after that is make a BufferedImage of it.
func (d *PageDrawer) transparencyGroupBox(group *form.PDTransparencyGroup,
	ctm *util.Matrix) *common.PDRectangle {
	// get the CTM x Form Matrix transform
	transform := util.Concatenate(ctm, group.Matrix())

	// transform the bbox
	formBBox := group.BBox()
	if formBBox == nil {
		// PDFBOX-5471
		// check done here and not in caller to avoid getBBox() creating rectangle twice
		slog.Warn("rendering: transparency group ignored because BBox is null")
		formBBox = common.NewPDRectangle()
	}
	transformedBox := formBBox.Transform(transform)

	// clip the bbox to prevent giant bboxes from consuming all memory
	transformed := geom.NewAreaOfShape(transformedBox)
	transformed.Intersect(d.GraphicsState().CurrentClippingPath())
	clipRect := transformed.Bounds2D()
	if clipRect.IsEmpty() {
		return nil
	}
	return common.NewPDRectangleOf(float32(clipRect.X), float32(clipRect.Y),
		float32(clipRect.Width), float32(clipRect.Height))
}

// hasBlendMode reports whether the given group, or any group inside it, sets a
// blend mode other than Normal.
func (d *PageDrawer) hasBlendMode(group *form.PDTransparencyGroup,
	groupsDone map[cos.Base]bool) bool {
	groupCOSStream := cos.Base(group.Stream())
	if groupsDone[groupCOSStream] {
		// The group is being processed. Avoid endless recursion.
		return false
	}
	groupsDone[groupCOSStream] = true

	if val, known := d.blendModeMap[groupCOSStream]; known {
		return val
	}

	resources, isResources := group.Resources().(*pdmodel.PDResources)
	if !isResources || resources == nil {
		d.blendModeMap[groupCOSStream] = false
		return false
	}
	for _, name := range resources.ExtGStateNames() {
		extGState := resources.GetExtGState(name)
		if extGState == nil {
			continue
		}
		if extGState.BlendMode() != blend.Normal {
			d.blendModeMap[groupCOSStream] = true
			return true
		}
	}

	// Recursively process nested transparency groups
	for _, name := range resources.XObjectNames() {
		xObject, err := resources.GetXObject(name)
		if err != nil {
			continue
		}
		if nested, isGroup := xObject.(*form.PDTransparencyGroup); isGroup &&
			d.hasBlendMode(nested, groupsDone) {
			d.blendModeMap[groupCOSStream] = true
			return true
		}
	}

	d.blendModeMap[groupCOSStream] = false
	return false
}

// BeginMarkedContentSequence handles BMC and BDC, counting the nesting of a
// hidden optional content group.
func (d *PageDrawer) BeginMarkedContentSequence(tag *cos.Name, properties *cos.Dictionary) {
	if d.nestedHiddenOCGCount > 0 {
		d.nestedHiddenOCGCount++
		return
	}
	if properties == nil {
		return
	}
	if d.isHiddenOCG(markedcontent.CreatePropertyList(properties)) {
		d.nestedHiddenOCGCount = 1
	}
}

// EndMarkedContentSequence handles EMC.
func (d *PageDrawer) EndMarkedContentSequence() {
	if d.nestedHiddenOCGCount > 0 {
		d.nestedHiddenOCGCount--
	}
}

// isContentRendered reports whether what is being drawn is visible at all.
func (d *PageDrawer) isContentRendered() bool { return d.nestedHiddenOCGCount <= 0 }
