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
	goimagecolor "image/color"
	"math"

	xdraw "golang.org/x/image/draw"

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
	dst *goimage.NRGBA

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

	// imageType is what the surface can hold: Java's BufferedImage type, which
	// quantizes what is written into it. See quantize.go.
	imageType rendering.ImageType

	// strokeNormalization is KEY_STROKE_CONTROL, and it is on because the JDK's
	// default is and PDFBox never sets the hint. See normalize.go.
	strokeNormalization bool

	// tiles is the tiling patterns rendered so far, which is Java's
	// TilingPaintFactory. It is shared with every copy Create makes, the way
	// Java's is a field of the one PageDrawer.
	tiles map[tilingKey]*tilingSource

	// groups is the stack of open transparency groups, and secondary the
	// alpha-only surface the innermost one is drawn onto in parallel.
	groups    []groupFrame
	secondary *goimage.NRGBA

	// blendScratch is the source, the destination and the result a
	// nonseparable blend mode is handed; see blendNonSeparable.
	blendScratch [3][3]float32
}

var _ rendering.Backend = (*Image)(nil)

// white is the ground Graphics2D.clearRect puts down under a blit.
var white = goimagecolor.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}

// NewImage returns a backend drawing onto an image of the given size.
//
// This is the BufferedImage Java's PDFRenderer.renderImage makes for itself.
// imageType says what Java would have asked BufferedImage for. The pixels are
// held straight, in an image.NRGBA, whatever it says; the type decides what is
// written into them -- an opaque white ground for everything but ARGB -- and
// what the finished surface is quantized to, which is quantizeTo.
func NewImage(width, height int, imageType rendering.ImageType) *Image {
	dst := goimage.NewNRGBA(goimage.Rect(0, 0, width, height))
	i := &Image{
		dst:                 dst,
		imageType:           imageType,
		transform:           geom.NewAffineTransform(1, 0, 0, 1, 0, 0),
		paint:               rendering.ColorPaint{Alpha: 1},
		alphaConstant:       1,
		antiAliasing:        true,
		strokeNormalization: true,
	}
	if imageType != rendering.ARGB {
		fillOpaqueWhite(dst)
	}
	return i
}

// fillOpaqueWhite is the white ground Java's renderImage paints before it
// draws, for every image type but ARGB.
func fillOpaqueWhite(dst *goimage.NRGBA) {
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

// SetStrokeNormalization turns stroke normalization on or off, which is
// RenderingHints.KEY_STROKE_CONTROL: on is VALUE_STROKE_NORMALIZE and off is
// VALUE_STROKE_PURE.
//
// It is **not** on Backend, because PageDrawer never sets it -- the hint is
// Graphics2D's and PDFBox leaves it alone, so it belongs to whoever makes the
// surface. It is on by default, because the JDK's default is, and that is what
// a page rendered by PDFBox goes through.
func (i *Image) SetStrokeNormalization(on bool) { i.strokeNormalization = on }

// SetInterpolation chooses how a scaled image is sampled.
func (i *Image) SetInterpolation(interpolation rendering.Interpolation) {
	i.interpolation = interpolation
}

// Fill fills the given shape with the current paint.
func (i *Image) Fill(shape geom.Shape) error {
	bounds := i.dst.Bounds()
	return i.composeWithin(coverageOf(shape, i.transform, bounds.Dx(), bounds.Dy(), i.antiAliasing),
		i.deviceBounds(shape, 0))
}

// deviceBounds answers where on the surface a shape can put anything down: its
// bounding box through the current transform, grown by pad and then by one
// pixel each way for the antialiasing at its edge.
//
// The coverage mask is the size of the whole surface, and a page is mostly made
// of fills that cover a little of it, so this is what keeps compose from walking
// every pixel of the page for every one of them. It is only a bound: what it
// answers is never smaller than where the mask is not zero.
func (i *Image) deviceBounds(shape geom.Shape, pad float64) goimage.Rectangle {
	box := shape.Bounds2D()
	if box == nil {
		return i.dst.Bounds()
	}
	if i.transform != nil {
		box = transformedRect(i.transform, box)
	}
	return paddedRect(geom.NewRectangle2D(box.X-pad, box.Y-pad,
		box.Width+2*pad, box.Height+2*pad))
}

// transformedRect answers the bounding box of a rectangle through a transform.
func transformedRect(at *geom.AffineTransform, r *geom.Rectangle2D) *geom.Rectangle2D {
	corners := []float64{
		r.X, r.Y,
		r.X + r.Width, r.Y,
		r.X + r.Width, r.Y + r.Height,
		r.X, r.Y + r.Height,
	}
	at.TransformDoubles(corners, 0, corners, 0, 4)
	minX, minY := corners[0], corners[1]
	maxX, maxY := minX, minY
	for at := 2; at < len(corners); at += 2 {
		minX = math.Min(minX, corners[at])
		maxX = math.Max(maxX, corners[at])
		minY = math.Min(minY, corners[at+1])
		maxY = math.Max(maxY, corners[at+1])
	}
	return geom.NewRectangle2D(minX, minY, maxX-minX, maxY-minY)
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
	mask := strokeCoverage(shape, i.transform, i.stroke,
		bounds.Dx(), bounds.Dy(), i.antiAliasing, i.strokeNormalization)
	if mask == nil {
		// a transform that flattens everything, through which Marlin strokes
		// nothing
		return nil
	}
	// A stroke puts paint half its width either side of the path, and a square
	// cap's corner or a miter join reaches further, so the pad is the whole
	// width rather than half of it, times the miter limit. The width is the one
	// the stroker drew in device space: penThrough multiplies it by the
	// transform, and never by more than the most the transform lengthens
	// anything.
	width := float64(i.stroke.LineWidth) * maximumScale(i.transform)
	if miter := float64(i.stroke.MiterLimit); miter > 1 {
		width *= miter
	}
	return i.composeWithin(mask, i.deviceBounds(shape, width))
}

// compose puts the current paint onto the destination through a coverage mask
// and the clip.
// The coverage of a fill is a mask the size of the whole surface, and a page
// holds many fills that cover a little of it, so the loop reads the mask's rows
// straight out of its Pix slice rather than asking AlphaAt for a pixel at a
// time: AlphaAt tests the bounds and works out the offset on every call, and on
// a page of tiling patterns that was 95% of the time a render took. What is
// composed does not change.
func (i *Image) compose(mask *goimage.Alpha) error {
	return i.composeWithin(mask, i.dst.Bounds())
}

// composeWithin is compose over the part of the surface a shape can reach.
func (i *Image) composeWithin(mask *goimage.Alpha, within goimage.Rectangle) error {
	source, paintAlpha, err := i.sourceOf(i.paint)
	if err != nil {
		return err
	}
	clip := i.clipCoverage()
	constant := paintAlpha * i.alphaConstant
	if constant <= 0 {
		return nil
	}

	bounds := i.dst.Bounds().Intersect(mask.Bounds()).Intersect(within)
	if clip != nil {
		bounds = bounds.Intersect(clip.Bounds())
	}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		row := mask.Pix[mask.PixOffset(bounds.Min.X, y):][:bounds.Dx()]
		var clipRow []uint8
		if clip != nil {
			clipRow = clip.Pix[clip.PixOffset(bounds.Min.X, y):][:bounds.Dx()]
		}
		for offset, alpha := range row {
			if alpha == 0 {
				continue
			}
			coverage := float64(alpha) / 255
			if clipRow != nil {
				if clipRow[offset] == 0 {
					continue
				}
				coverage *= float64(clipRow[offset]) / 255
			}
			x := bounds.Min.X + offset
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

// ErrNoPatternBBox is a tiling pattern with no /BBox, which Java's
// TilingPaint.getAnchorRect throws an IOException for.
var ErrNoPatternBBox = errors.New("raster: pattern /BBox is missing")

// NewOffscreen returns a transparent surface of the given size.
//
// Java's `new BufferedImage(w, h, TYPE_INT_ARGB)`, which is what PDFPrintable
// rasterizes a page into. The state is not carried over: Java's image comes
// with a fresh Graphics2D, and so does this.
func (i *Image) NewOffscreen(width, height int) rendering.Backend {
	return NewImage(width, height, rendering.ARGB)
}

// DrawSurface draws another backend's pixels onto this one, through the
// transform in force.
//
// Port of the three lines PDFPrintable ends its rasterizing arm with:
//
//	printerGraphics.setBackground(Color.WHITE);
//	printerGraphics.clearRect(0, 0, image.getWidth(), image.getHeight());
//	printerGraphics.drawImage(image, 0, 0, null);
//
// **All three go through the transform**, which is the thing to know about
// this method. `PDFPrintable.Print` puts the imageable-area translation, the
// centring and the `scale / dpiScale` rescale on the surface before it calls
// this, so a blit that copied pixel to pixel would put the page in the paper's
// corner at the wrong size. The image occupies `[0,w] x [0,h]` in user space,
// one pixel to the unit, which is what `drawImage(image, 0, 0, null)` means.
//
// The clearRect is a white ground under the blit and not over it, and it
// covers the same rectangle. The clip in force applies to both, which is what
// Graphics2D does.
func (i *Image) DrawSurface(surface rendering.Backend) error {
	source, isImage := surface.(*Image)
	if !isImage {
		return ErrNotDrawn
	}
	sourceBounds := source.dst.Bounds()
	if sourceBounds.Empty() {
		return nil
	}
	bounds := i.dst.Bounds()

	// Where the image lands, which is what clearRect covers.
	ground := coverageOf(geom.NewRectangle2D(0, 0,
		float64(sourceBounds.Dx()), float64(sourceBounds.Dy())),
		i.transform, bounds.Dx(), bounds.Dy(), i.antiAliasing)

	// And the pixels, sampled through the same transform with the
	// interpolation the hints ask for. Premultiplied, because that is what
	// x/image/draw writes.
	sampled := goimage.NewRGBA(bounds)
	i.interpolator().Transform(sampled, aff3Of(i.transform),
		source.dst, sourceBounds, xdraw.Src, nil)

	clip := i.clipCoverage()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			coverage := float64(ground.AlphaAt(x, y).A) / 255
			if coverage == 0 {
				continue
			}
			if clip != nil {
				coverage *= float64(clip.AlphaAt(x, y).A) / 255
				if coverage == 0 {
					continue
				}
			}
			// the white ground clearRect puts down
			i.blendPixel(x, y, white, coverage)

			c := sampled.RGBAAt(x, y)
			if c.A == 0 {
				continue
			}
			i.blendPixel(x, y, unpremultiply(c), coverage*float64(c.A)/255)
		}
	}
	return nil
}

// paddedRect answers the integer rectangle that holds a real one, with a pixel
// each way for what antialiasing puts at its edge.
func paddedRect(r *geom.Rectangle2D) goimage.Rectangle {
	return goimage.Rect(
		int(math.Floor(r.X))-1,
		int(math.Floor(r.Y))-1,
		int(math.Ceil(r.X+r.Width))+2,
		int(math.Ceil(r.Y+r.Height))+2,
	)
}
