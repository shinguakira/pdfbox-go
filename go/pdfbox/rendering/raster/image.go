// Package raster implements rendering.Backend over an in-memory image.
//
// Slice 9 ported everything in the renderer that computes and put only the
// drawing behind rendering.Backend; this is the interface's other side, and
// `track/raster` is the branch that writes it.
//
// **This is a substitution, not a transliteration.** There is no Java to port
// here: Java draws onto a java.awt.Graphics2D it takes from a BufferedImage,
// and Graphics2D has no Go equivalent. What is written here is what
// Graphics2D would have done, against the same interface the rest of the
// renderer already calls. The filling and stroking come from
// github.com/srwiley/rasterx, chosen in this branch's A0 -- see
// migration/STATUS.md -- and the compositing, the clip and the transparency
// groups are written here, over the blend.BlendMode the port already has.
package raster

import (
	"errors"
	goimage "image"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/blend"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/rendering"
)

// ErrNotDrawn is what an operation answers while it is still a stub.
//
// Phase A of this branch declares the whole backend so that the comparisons it
// exists to make possible can be written and can fail for the right reason;
// phase B fills them in. Nothing outside this package should ever see it.
var ErrNotDrawn = errors.New("raster: not implemented yet")

// Image is a rendering.Backend that draws onto an in-memory image.
type Image struct {
	dst *goimage.RGBA

	transform *geom.AffineTransform

	clip *geom.Area
	// clipTransform is the transform the clip was installed under, and
	// clipMask the coverage it rasterises to, built once and dropped when the
	// clip changes.
	clipTransform *geom.AffineTransform
	clipMask      *goimage.Alpha

	paint         rendering.Paint
	stroke        *rendering.Stroke
	blendMode     *blend.BlendMode
	alphaConstant float64
	antiAliasing  bool
	interpolation rendering.Interpolation

	// groups is the stack of open transparency groups, and secondary the
	// alpha-only surface the innermost one is drawn onto in parallel.
	groups    []groupFrame
	secondary *goimage.RGBA
}

var _ rendering.Backend = (*Image)(nil)

// NewImage returns a backend drawing onto an image of the given size.
//
// This is the BufferedImage Java's PDFRenderer.renderImage makes for itself.
// imageType says what Java would have asked BufferedImage for; the pixels are
// held as RGBA whatever it says, and the type decides what is written into
// them -- an opaque white ground for RGB and Gray, nothing for ARGB.
func NewImage(width, height int, imageType rendering.ImageType) *Image {
	dst := goimage.NewRGBA(goimage.Rect(0, 0, width, height))
	i := &Image{
		dst:           dst,
		transform:     geom.NewAffineTransform(1, 0, 0, 1, 0, 0),
		paint:         rendering.ColorPaint{Alpha: 1},
		alphaConstant: 1,
		antiAliasing:  true,
	}
	if imageType != rendering.ARGB {
		fillOpaqueWhite(dst)
	}
	return i
}

// fillOpaqueWhite is the white ground Java's renderImage paints before it
// draws, for every image type but ARGB.
func fillOpaqueWhite(dst *goimage.RGBA) {
	for i := range dst.Pix {
		dst.Pix[i] = 0xFF
	}
}

// Image returns the pixels drawn so far.
//
// Java's renderImage answers a BufferedImage; the port draws through a backend
// the caller installs and keeps, so the caller asks the backend for it.
func (i *Image) Image() goimage.Image { return i.dst }

// Create returns a backend drawing to the same image with a copy of the state.
func (i *Image) Create() rendering.Backend {
	copied := *i
	if i.transform != nil {
		copied.transform = i.transform.Clone()
	}
	return &copied
}

// Dispose releases what Create took, which for an in-memory image is nothing.
func (i *Image) Dispose() {}

// Transform returns the transform from user space to device space.
func (i *Image) Transform() *geom.AffineTransform { return i.transform }

// SetTransform replaces it.
func (i *Image) SetTransform(at *geom.AffineTransform) { i.transform = at }

// Clip returns the clipping area in force, or nil.
func (i *Image) Clip() *geom.Area { return i.clip }

// SetClip installs the clipping area.
//
// The transform in force is kept with it. java.awt.Graphics2D.setClip stores
// the clip in device space, transformed by the transform of the moment, so a
// later transform does not move it; the port keeps the shape and the transform
// that goes with it and rasterises when it is first needed.
func (i *Image) SetClip(clip *geom.Area) {
	i.clip = clip
	i.clipTransform = i.transform
	if i.transform != nil {
		i.clipTransform = i.transform.Clone()
	}
	i.clipMask = nil
}

// clipCoverage is the clip as an alpha mask, rasterised once.
func (i *Image) clipCoverage() *goimage.Alpha {
	if i.clip == nil {
		return nil
	}
	if i.clipMask == nil {
		bounds := i.dst.Bounds()
		i.clipMask = coverageOf(i.clip, i.clipTransform, bounds.Dx(), bounds.Dy(), true)
	}
	return i.clipMask
}

// SetPaint installs the paint the next fill or draw uses.
func (i *Image) SetPaint(paint rendering.Paint) { i.paint = paint }

// SetStroke installs the stroke the next draw uses.
func (i *Image) SetStroke(stroke *rendering.Stroke) { i.stroke = stroke }

// SetComposite installs the blend mode and alpha constant.
func (i *Image) SetComposite(blendMode *blend.BlendMode, alphaConstant float64) {
	i.blendMode = blendMode
	i.alphaConstant = alphaConstant
}

// SetAntiAliasing turns anti-aliasing on or off.
func (i *Image) SetAntiAliasing(on bool) { i.antiAliasing = on }

// SetInterpolation chooses how a scaled image is sampled.
func (i *Image) SetInterpolation(interpolation rendering.Interpolation) {
	i.interpolation = interpolation
}

// Fill fills the given shape with the current paint.
func (i *Image) Fill(shape geom.Shape) error {
	bounds := i.dst.Bounds()
	return i.compose(coverageOf(shape, i.transform, bounds.Dx(), bounds.Dy(), i.antiAliasing))
}

// Draw strokes the outline of the given shape with the current paint and
// stroke.
func (i *Image) Draw(shape geom.Shape) error {
	if i.stroke == nil {
		// Java would have a BasicStroke here whatever happened; PageDrawer
		// always sets one before it draws.
		return nil
	}
	if i.stroke.Invisible {
		// an all-zero dash array, which Adobe draws as nothing: PDFBOX-5168
		return nil
	}
	bounds := i.dst.Bounds()
	return i.compose(strokeCoverage(shape, i.transform, i.stroke,
		bounds.Dx(), bounds.Dy(), i.antiAliasing))
}

// compose puts the current paint onto the destination through a coverage mask
// and the clip.
func (i *Image) compose(mask *goimage.Alpha) error {
	source, paintAlpha, err := i.sourceOf(i.paint)
	if err != nil {
		return err
	}
	clip := i.clipCoverage()
	constant := paintAlpha * i.alphaConstant
	if constant <= 0 {
		return nil
	}

	bounds := i.dst.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			coverage := float64(mask.AlphaAt(x, y).A) / 255
			if coverage == 0 {
				continue
			}
			if clip != nil {
				coverage *= float64(clip.AlphaAt(x, y).A) / 255
				if coverage == 0 {
					continue
				}
			}
			c, painted := source.colorAt(x, y)
			if !painted {
				continue
			}
			i.blendPixel(x, y, c, coverage*constant*float64(c.A)/255)
		}
	}
	return nil
}

// errNoGroup is a PopGroup with no PushGroup, which is a programming error in
// the caller rather than anything a document can cause.
var errNoGroup = errors.New("raster: PopGroup without PushGroup")
