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
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/blend"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
	pdimage "github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/image"
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

	transform     *geom.AffineTransform
	clip          *geom.Area
	paint         rendering.Paint
	stroke        *rendering.Stroke
	blendMode     *blend.BlendMode
	alphaConstant float64
	antiAliasing  bool
	interpolation rendering.Interpolation
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
func (i *Image) SetClip(clip *geom.Area) { i.clip = clip }

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
func (i *Image) Fill(shape geom.Shape) error { return ErrNotDrawn }

// Draw strokes the outline of the given shape.
func (i *Image) Draw(shape geom.Shape) error { return ErrNotDrawn }

// DrawImage draws the given image through the given transform.
func (i *Image) DrawImage(pdImage pdimage.PDImage, at *geom.AffineTransform,
	subsampling int) error {
	return ErrNotDrawn
}

// DrawStencil draws the given stencil mask filled with the given paint.
func (i *Image) DrawStencil(pdImage pdimage.PDImage, at *geom.AffineTransform,
	paint rendering.Paint) error {
	return ErrNotDrawn
}

// PushGroup begins a transparency group.
func (i *Image) PushGroup(bbox *common.PDRectangle, isSoftMask, needsBackdrop bool,
	backdropColor *color.PDColor) error {
	return ErrNotDrawn
}

// PopGroup composites the group PushGroup began.
func (i *Image) PopGroup() error { return ErrNotDrawn }
