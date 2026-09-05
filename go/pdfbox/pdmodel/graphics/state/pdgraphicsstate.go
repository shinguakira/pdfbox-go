package state

import (
	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/blend"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// Line cap and join styles, which a PDF gives as the same small integers
// java.awt.BasicStroke uses.
const (
	// CapButt ends a line square at its endpoint.
	CapButt = 0

	// JoinMiter joins two segments by extending their outer edges to a point.
	JoinMiter = 0
)

// PDGraphicsState is the current state of the graphics parameters when
// executing a content stream.
//
// Port of org.apache.pdfbox.pdmodel.graphics.state.PDGraphicsState.
//
// The two Java composites are not here: getStrokingJavaComposite and
// getNonStrokingJavaComposite each wrap the blend mode and an alpha constant in
// a java.awt.Composite, which is the rasteriser's half of the work. The blend
// mode and both constants are here, which is what a backend needs to build one.
// See migration/STATUS.md.
type PDGraphicsState struct {
	// softMask is the mask paint reaches the page through, or nil.
	softMask *PDSoftMask

	isClippingPathDirty         bool
	clippingPaths               []*geom.Path2D
	clippingPathCache           *geom.Area
	currentTransformationMatrix *util.Matrix
	strokingColor               *color.PDColor
	nonStrokingColor            *color.PDColor
	strokingColorSpace          color.PDColorSpace
	nonStrokingColorSpace       color.PDColorSpace
	textState                   *PDTextState
	lineWidth                   float32
	lineCap                     int
	lineJoin                    int
	miterLimit                  float32
	lineDashPattern             *graphics.PDLineDashPattern
	renderingIntent             *RenderingIntent
	strokeAdjustment            bool
	blendMode                   *blend.BlendMode
	alphaConstant               float64
	nonStrokingAlphaConstant    float64
	alphaSource                 bool
	textMatrix                  *util.Matrix
	textLineMatrix              *util.Matrix

	// DEVICE-DEPENDENT parameters
	overprint            bool
	nonStrokingOverprint bool
	overprintMode        int
	// black generation
	// undercolor removal
	transfer cos.Base
	// halftone
	flatness   float64
	smoothness float64
}

// NewPDGraphicsState returns the state a content stream starts in, with the
// clipping path set to the whole page.
func NewPDGraphicsState(page *common.PDRectangle) *PDGraphicsState {
	s := &PDGraphicsState{
		currentTransformationMatrix: util.NewMatrix(),
		strokingColor:               color.DeviceGray.InitialColor(),
		nonStrokingColor:            color.DeviceGray.InitialColor(),
		strokingColorSpace:          color.DeviceGray,
		nonStrokingColorSpace:       color.DeviceGray,
		textState:                   NewPDTextState(),
		lineWidth:                   1,
		lineCap:                     CapButt,
		lineJoin:                    JoinMiter,
		miterLimit:                  10,
		lineDashPattern:             graphics.NewPDLineDashPattern(),
		blendMode:                   blend.Normal,
		alphaConstant:               1.0,
		nonStrokingAlphaConstant:    1.0,
		flatness:                    1.0,
	}
	s.clippingPaths = []*geom.Path2D{geom.NewPathDoubleShape(page.ToGeneralPath())}
	return s
}

// CurrentTransformationMatrix returns the value of the CTM.
func (s *PDGraphicsState) CurrentTransformationMatrix() *util.Matrix {
	return s.currentTransformationMatrix
}

// SetCurrentTransformationMatrix sets the value of the CTM.
func (s *PDGraphicsState) SetCurrentTransformationMatrix(value *util.Matrix) {
	s.currentTransformationMatrix = value
}

// LineWidth returns the current line width.
func (s *PDGraphicsState) LineWidth() float32 { return s.lineWidth }

// SetLineWidth sets the line width.
func (s *PDGraphicsState) SetLineWidth(value float32) { s.lineWidth = value }

// LineCap returns the current line cap.
func (s *PDGraphicsState) LineCap() int { return s.lineCap }

// SetLineCap sets the line cap.
func (s *PDGraphicsState) SetLineCap(value int) { s.lineCap = value }

// LineJoin returns the current line join.
func (s *PDGraphicsState) LineJoin() int { return s.lineJoin }

// SetLineJoin sets the line join.
func (s *PDGraphicsState) SetLineJoin(value int) { s.lineJoin = value }

// MiterLimit returns the current miter limit.
func (s *PDGraphicsState) MiterLimit() float32 { return s.miterLimit }

// SetMiterLimit sets the miter limit.
func (s *PDGraphicsState) SetMiterLimit(value float32) { s.miterLimit = value }

// IsStrokeAdjustment returns the current stroke adjustment.
func (s *PDGraphicsState) IsStrokeAdjustment() bool { return s.strokeAdjustment }

// SetStrokeAdjustment sets the stroke adjustment.
func (s *PDGraphicsState) SetStrokeAdjustment(value bool) { s.strokeAdjustment = value }

// AlphaConstant returns the value of the stroke alpha constant parameter.
func (s *PDGraphicsState) AlphaConstant() float64 { return s.alphaConstant }

// SetAlphaConstant sets the stroke alpha constant parameter.
func (s *PDGraphicsState) SetAlphaConstant(value float64) { s.alphaConstant = value }

// NonStrokeAlphaConstant returns the value of the non-stroke alpha constant
// parameter.
func (s *PDGraphicsState) NonStrokeAlphaConstant() float64 { return s.nonStrokingAlphaConstant }

// SetNonStrokeAlphaConstant sets the non-stroke alpha constant parameter.
func (s *PDGraphicsState) SetNonStrokeAlphaConstant(value float64) {
	s.nonStrokingAlphaConstant = value
}

// IsAlphaSource returns the value of the stroke alpha source parameter.
func (s *PDGraphicsState) IsAlphaSource() bool { return s.alphaSource }

// SetAlphaSource sets the stroke alpha source parameter.
func (s *PDGraphicsState) SetAlphaSource(value bool) { s.alphaSource = value }

// BlendMode returns the current blend mode.
func (s *PDGraphicsState) BlendMode() *blend.BlendMode { return s.blendMode }

// SetBlendMode sets the blend mode.
func (s *PDGraphicsState) SetBlendMode(blendMode *blend.BlendMode) { s.blendMode = blendMode }

// IsOverprint returns the overprint setting.
func (s *PDGraphicsState) IsOverprint() bool { return s.overprint }

// SetOverprint sets the overprint.
func (s *PDGraphicsState) SetOverprint(value bool) { s.overprint = value }

// IsNonStrokingOverprint returns the non-stroking overprint setting.
func (s *PDGraphicsState) IsNonStrokingOverprint() bool { return s.nonStrokingOverprint }

// SetNonStrokingOverprint sets the non-stroking overprint.
func (s *PDGraphicsState) SetNonStrokingOverprint(value bool) { s.nonStrokingOverprint = value }

// OverprintMode returns the overprint mode.
func (s *PDGraphicsState) OverprintMode() int { return s.overprintMode }

// SetOverprintMode sets the overprint mode.
func (s *PDGraphicsState) SetOverprintMode(value int) { s.overprintMode = value }

// Flatness returns the flatness tolerance.
func (s *PDGraphicsState) Flatness() float64 { return s.flatness }

// SetFlatness sets the flatness tolerance.
func (s *PDGraphicsState) SetFlatness(value float64) { s.flatness = value }

// Smoothness returns the smoothness tolerance.
func (s *PDGraphicsState) Smoothness() float64 { return s.smoothness }

// SetSmoothness sets the smoothness tolerance.
func (s *PDGraphicsState) SetSmoothness(value float64) { s.smoothness = value }

// TextState returns the text state.
func (s *PDGraphicsState) TextState() *PDTextState { return s.textState }

// SetTextState sets the text state.
func (s *PDGraphicsState) SetTextState(value *PDTextState) { s.textState = value }

// LineDashPattern returns the line dash pattern.
func (s *PDGraphicsState) LineDashPattern() *graphics.PDLineDashPattern { return s.lineDashPattern }

// SetLineDashPattern sets the line dash pattern.
func (s *PDGraphicsState) SetLineDashPattern(value *graphics.PDLineDashPattern) {
	s.lineDashPattern = value
}

// RenderingIntent returns the rendering intent, or nil where the stream has not
// set one.
//
// The Java field is null until an operator or an extended graphics state fills
// it in, and a consumer applies its own default for that. A bare Go enum would
// read as its zero value instead, which is AbsoluteColorimetric — an intent the
// file never asked for.
func (s *PDGraphicsState) RenderingIntent() *RenderingIntent { return s.renderingIntent }

// SetRenderingIntent sets the rendering intent.
func (s *PDGraphicsState) SetRenderingIntent(value RenderingIntent) {
	s.renderingIntent = &value
}

// StrokingColor returns the stroking colour.
func (s *PDGraphicsState) StrokingColor() *color.PDColor { return s.strokingColor }

// SetStrokingColor sets the stroking colour.
func (s *PDGraphicsState) SetStrokingColor(c *color.PDColor) { s.strokingColor = c }

// NonStrokingColor returns the non-stroking colour.
func (s *PDGraphicsState) NonStrokingColor() *color.PDColor { return s.nonStrokingColor }

// SetNonStrokingColor sets the non-stroking colour.
func (s *PDGraphicsState) SetNonStrokingColor(c *color.PDColor) { s.nonStrokingColor = c }

// StrokingColorSpace returns the stroking colour space.
func (s *PDGraphicsState) StrokingColorSpace() color.PDColorSpace { return s.strokingColorSpace }

// SetStrokingColorSpace sets the stroking colour space.
func (s *PDGraphicsState) SetStrokingColorSpace(colorSpace color.PDColorSpace) {
	s.strokingColorSpace = colorSpace
}

// NonStrokingColorSpace returns the non-stroking colour space.
func (s *PDGraphicsState) NonStrokingColorSpace() color.PDColorSpace {
	return s.nonStrokingColorSpace
}

// SetNonStrokingColorSpace sets the non-stroking colour space.
func (s *PDGraphicsState) SetNonStrokingColorSpace(colorSpace color.PDColorSpace) {
	s.nonStrokingColorSpace = colorSpace
}

// IntersectClippingPath modifies the current clipping path by intersecting it
// with the given path. The path is copied, so the caller may go on using it.
func (s *PDGraphicsState) IntersectClippingPath(path *geom.Path2D) {
	s.intersectClippingPath(geom.NewPathDoubleShape(path), true)
}

func (s *PDGraphicsState) intersectClippingPath(path *geom.Path2D, clonePath bool) {
	// lazy cloning of clipping path for performance
	if !s.isClippingPathDirty {
		// shallow copy
		s.clippingPaths = append([]*geom.Path2D(nil), s.clippingPaths...)
		s.isClippingPathDirty = true
	}
	// add path to current clipping paths, combined later (see CurrentClippingPath)
	if clonePath {
		path = path.Clone()
	}
	s.clippingPaths = append(s.clippingPaths, path)
	// clear cache
	s.clippingPathCache = nil
}

// CurrentClippingPaths returns the paths the clipping region is the
// intersection of. Do not modify them.
func (s *PDGraphicsState) CurrentClippingPaths() []*geom.Path2D { return s.clippingPaths }

// IntersectClippingArea modifies the current clipping path by intersecting it
// with the given area. The area is not copied, so the caller must not go on
// using it.
//
// Port of intersectClippingPath(Area), which Java overloads on the argument.
func (s *PDGraphicsState) IntersectClippingArea(area *geom.Area) {
	s.intersectClippingPath(geom.NewPathDoubleShape(area), false)
}

// CurrentClippingPath returns the area the clipping paths intersect to, and
// replaces the list with that one area so the work is done once.
//
// Port of getCurrentClippingPath.
func (s *PDGraphicsState) CurrentClippingPath() *geom.Area {
	// If there is just a single clipping path, no intersections are needed.
	if len(s.clippingPaths) == 1 {
		if s.clippingPathCache == nil {
			s.clippingPathCache = geom.NewAreaOfShape(s.clippingPaths[0])
		}
		return s.clippingPathCache
	}
	// calculate the intersected overall bounding box for all clipping paths
	boundingBox := s.clippingPaths[0].Bounds2D()
	for i := 1; i < len(s.clippingPaths); i++ {
		geom.Intersect(boundingBox, s.clippingPaths[i].Bounds2D(), boundingBox)
	}
	// use the overall bounding box as starting area
	clippingArea := geom.NewAreaOfShape(boundingBox)
	// combine all clipping paths to a single area
	for i := 0; i < len(s.clippingPaths); i++ {
		nextArea := geom.NewAreaOfShape(s.clippingPaths[i])
		clippingArea.Intersect(nextArea)
		nextArea.Reset()
	}
	s.clippingPathCache = clippingArea
	// Replace the list of individual clipping paths with the intersection
	s.clippingPaths = []*geom.Path2D{geom.NewPathDoubleShape(clippingArea)}
	return clippingArea
}

// Transfer returns the transfer function.
func (s *PDGraphicsState) Transfer() cos.Base { return s.transfer }

// SetTransfer sets the transfer function.
func (s *PDGraphicsState) SetTransfer(transfer cos.Base) { s.transfer = transfer }

// TextLineMatrix returns the text line matrix.
func (s *PDGraphicsState) TextLineMatrix() *util.Matrix { return s.textLineMatrix }

// SetTextLineMatrix sets the text line matrix.
func (s *PDGraphicsState) SetTextLineMatrix(value *util.Matrix) { s.textLineMatrix = value }

// TextMatrix returns the text matrix.
func (s *PDGraphicsState) TextMatrix() *util.Matrix { return s.textMatrix }

// SetTextMatrix sets the text matrix.
func (s *PDGraphicsState) SetTextMatrix(value *util.Matrix) { s.textMatrix = value }

// Clone returns a copy of this state, as q pushes onto the stack.
//
// The clipping paths are shared rather than copied; the copy takes its own list
// the first time something is added to it, which is what isClippingPathDirty is
// for. The colours and the dash pattern are immutable and shared outright.
func (s *PDGraphicsState) Clone() *PDGraphicsState {
	clone := *s
	clone.textState = s.textState.Clone()
	clone.currentTransformationMatrix = s.currentTransformationMatrix.Clone()
	clone.strokingColor = s.strokingColor       // immutable
	clone.nonStrokingColor = s.nonStrokingColor // immutable
	clone.lineDashPattern = s.lineDashPattern   // immutable
	clone.clippingPaths = s.clippingPaths       // not cloned, see intersectClippingPath
	clone.clippingPathCache = s.clippingPathCache
	clone.isClippingPathDirty = false
	if s.textLineMatrix != nil {
		clone.textLineMatrix = s.textLineMatrix.Clone()
	}
	if s.textMatrix != nil {
		clone.textMatrix = s.textMatrix.Clone()
	}
	return &clone
}

// SetRenderingIntentOrNil sets the rendering intent, or clears it where the
// value is nil, which is what Java's setRenderingIntent does when it is handed
// the null an extended graphics state can carry.
func (s *PDGraphicsState) SetRenderingIntentOrNil(value *RenderingIntent) {
	s.renderingIntent = value
}

// SoftMask returns the soft mask paint is reaching the page through, or nil
// where there is none.
func (s *PDGraphicsState) SoftMask() *PDSoftMask { return s.softMask }

// SetSoftMask sets the soft mask paint reaches the page through.
func (s *PDGraphicsState) SetSoftMask(softMask *PDSoftMask) { s.softMask = softMask }
