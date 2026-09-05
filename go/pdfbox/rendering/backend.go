package rendering

// The raster boundary.
//
// Java draws through java.awt.Graphics2D, and PageDrawer is written against it:
// it sets a Paint, a Stroke, a Composite and a clip, then fills or draws a
// Shape, draws a BufferedImage, or renders a transparency group into one.
//
// Slice 9's B0 decision is to port everything that computes and to put only
// that last step behind an interface. Backend is that interface. Everything
// above it -- which paint applies, what the stroke is made of, what the clip
// intersects to, which shading is evaluated where, whether an optional content
// group is visible -- is ported and runs; what a Backend does with the calls is
// the drawing, and no Backend ships in this slice. See migration/STATUS.md.

import (
	"errors"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/blend"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/image"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/pattern"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/shading"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/state"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// ErrNoBackend is what every entry point that would have to produce pixels
// answers when no Backend is installed.
//
// Java's renderImage makes a BufferedImage and hands its Graphics2D to the page
// drawer. There is nothing to make one with here, so the port says so rather
// than answering a blank image, which would look like a rendered page.
var ErrNoBackend = errors.New("rendering: no raster backend is installed")

// Paint says how an area is to be coloured.
//
// Port of the java.awt.Paint values PageDrawer.getPaint answers. Java's Paint
// is a factory for a PaintContext that produces pixels; the port's is a
// description of what was decided, which a Backend turns into pixels. The four
// implementations below are the four arms of getPaint, plus the soft mask
// wrapper applySoftMaskToPaint adds.
type Paint interface {
	isPaint()
}

// ColorPaint is a solid colour, in sRGB, with an alpha.
//
// Port of the java.awt.Color arms of getPaint: a colour space converted through
// toRGB, and the two fully transparent colours that stand for "paint nothing".
type ColorPaint struct {
	Red, Green, Blue, Alpha float32
}

func (ColorPaint) isPaint() {}

// transparentPaint is `new Color(0, 0, 0, 0)`, which getPaint answers where
// there is nothing to paint with.
var transparentPaint = ColorPaint{}

// ShadingPaint is the colour a shading gives at each point, through the given
// matrix.
//
// Port of PDShading.toPaint(Matrix), which builds a java.awt.Paint over the
// shading. The shading itself evaluates colours, and that half is ported.
type ShadingPaint struct {
	Shading shading.Shading
	Matrix  *util.Matrix
}

func (ShadingPaint) isPaint() {}

// TilingPaint is a tiling pattern's content stream, repeated.
//
// Port of what TilingPaintFactory.create names: the pattern, the transform in
// force, and -- for an uncoloured pattern -- the colour and colour space it is
// painted in, both nil for a coloured one.
type TilingPaint struct {
	Pattern    *pattern.PDTilingPattern
	ColorSpace color.PDColorSpace
	Color      *color.PDColor
	Transform  *geom.AffineTransform
}

func (TilingPaint) isPaint() {}

// SoftMaskedPaint is another paint seen through a soft mask.
//
// Port of the rendering.SoftMask paint applySoftMaskToPaint wraps around.
// Java builds the mask's raster there and then; the port names the mask and the
// matrix it was installed under, which is what a backend needs to build the
// same raster.
type SoftMaskedPaint struct {
	Paint Paint
	Mask  *state.PDSoftMask
}

func (SoftMaskedPaint) isPaint() {}

// Stroke is what a line is drawn with.
//
// Port of the java.awt.BasicStroke PageDrawer.getStroke builds, field for
// field, plus Invisible for the all-zero dash array, which Java expresses by
// returning a Stroke whose createStrokedShape answers an empty Area.
type Stroke struct {
	LineWidth  float32
	LineCap    int
	LineJoin   int
	MiterLimit float32

	// DashArray is nil where the line is solid.
	DashArray []float32
	DashPhase float32

	// Invisible says the dash array was all zeros, which Adobe draws as
	// nothing at all. See PDFBOX-5168.
	Invisible bool
}

// Backend is the raster half of the renderer.
//
// It is java.awt.Graphics2D as PageDrawer uses it, and nothing more. The state
// setters are separate from the drawing calls because Java's are: PageDrawer
// sets a composite, a paint, a stroke and a clip, and only then fills or draws.
type Backend interface {
	// Transform returns the transform from user space to device space.
	Transform() *geom.AffineTransform

	// SetTransform replaces it.
	SetTransform(at *geom.AffineTransform)

	// Clip returns the clipping area in force, or nil where there is none.
	Clip() *geom.Area

	// SetClip installs the clipping area, nil meaning none.
	SetClip(clip *geom.Area)

	// SetPaint installs the paint the next fill or draw uses.
	SetPaint(paint Paint)

	// SetStroke installs the stroke the next draw uses.
	SetStroke(stroke *Stroke)

	// SetComposite installs the blend mode and alpha constant the next fill or
	// draw is composited with, which is Java's setComposite of a
	// BlendComposite.
	SetComposite(blendMode *blend.BlendMode, alphaConstant float64)

	// SetAntiAliasing turns anti-aliasing on or off, which fillPath does for a
	// rectangular path and drawImage for one scaled up.
	SetAntiAliasing(on bool)

	// SetInterpolation chooses how a scaled image is sampled.
	SetInterpolation(interpolation Interpolation)

	// Fill fills the given shape with the current paint.
	Fill(shape geom.Shape) error

	// Draw strokes the outline of the given shape with the current paint and
	// stroke.
	Draw(shape geom.Shape) error

	// DrawImage draws the given image through the given transform, which maps
	// the unit square onto where the image goes. subsampling is 1 where the
	// whole image is wanted.
	DrawImage(pdImage image.PDImage, at *geom.AffineTransform, subsampling int) error

	// DrawStencil draws the given stencil mask filled with the given paint,
	// through the given transform.
	DrawStencil(pdImage image.PDImage, at *geom.AffineTransform, paint Paint) error

	// PushGroup begins a transparency group over the given box in user space,
	// which every following call paints into until PopGroup.
	//
	// needsBackdrop says the group is neither isolated nor a soft mask and uses
	// a blend mode, so the backdrop it is composited over has to be visible to
	// it. backdropColor is the /BC of a luminosity soft mask, and nil
	// otherwise.
	PushGroup(bbox *common.PDRectangle, isSoftMask, needsBackdrop bool,
		backdropColor *color.PDColor) error

	// PopGroup composites the group PushGroup began and ends it.
	PopGroup() error
}

// Interpolation is how a scaled image is sampled.
//
// Port of the RenderingHints.KEY_INTERPOLATION values PDFRenderer and
// PageDrawer set.
type Interpolation int

const (
	// NearestNeighbor takes the nearest sample, which is what a scaled up image
	// with /Interpolate false gets.
	NearestNeighbor Interpolation = iota

	// Bicubic interpolates, which is the default for everything else.
	Bicubic
)

// String returns the name of the rendering hint value.
func (i Interpolation) String() string {
	if i == NearestNeighbor {
		return "VALUE_INTERPOLATION_NEAREST_NEIGHBOR"
	}
	return "VALUE_INTERPOLATION_BICUBIC"
}

// RenderingHints say how carefully to draw.
//
// Port of the java.awt.RenderingHints PDFRenderer builds and PageDrawer applies
// -- the three keys PDFBox actually sets. Java's is an open map; the port names
// the three, because every reader of it reads one of them by name.
type RenderingHints struct {
	// Interpolation is KEY_INTERPOLATION.
	Interpolation Interpolation

	// AntiAliasing is KEY_ANTIALIASING.
	AntiAliasing bool

	// Quality is KEY_RENDERING, true for VALUE_RENDER_QUALITY.
	Quality bool
}

// DefaultRenderingHints returns the hints PDFBox chooses at runtime.
//
// Port of PDFRenderer.createDefaultRenderingHints. isBitonal comes from the
// display the Graphics2D is on, which the port cannot ask for; a Backend that
// draws to a one-bit device says so itself.
func DefaultRenderingHints(isBitonal bool) RenderingHints {
	return RenderingHints{
		Interpolation: map[bool]Interpolation{true: NearestNeighbor, false: Bicubic}[isBitonal],
		AntiAliasing:  !isBitonal,
		Quality:       true,
	}
}
