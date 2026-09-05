package rendering

// A Backend that records what it was asked to do.
//
// Slice 9's A5 decision: rendered output is compared by numbers and call
// sequences, never by pixels. Java's three rendering tests all compare against
// reference images, which needs a rasteriser to produce one; with no rasteriser
// there is no image, and this is what stands in its place. It says what the
// drawer decided to draw, in order, with what state -- which is what a
// rendering test is for.

import (
	"fmt"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/blend"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/image"
)

// recordingBackend records every call rather than drawing anything.
type recordingBackend struct {
	calls []string

	transform     *geom.AffineTransform
	clip          *geom.Area
	paint         Paint
	stroke        *Stroke
	blendMode     *blend.BlendMode
	alphaConstant float64
	antiAliasing  bool
	interpolation Interpolation
}

var _ Backend = (*recordingBackend)(nil)

// newRecordingBackend returns a backend at the identity transform with no clip.
func newRecordingBackend() *recordingBackend {
	return &recordingBackend{transform: geom.NewAffineTransform(1, 0, 0, 1, 0, 0)}
}

// record appends one line to the call log.
func (b *recordingBackend) record(format string, args ...any) {
	b.calls = append(b.calls, fmt.Sprintf(format, args...))
}

// Log returns the recorded calls, one per line.
func (b *recordingBackend) Log() string { return strings.Join(b.calls, "\n") }

// Drawn returns only the calls that put something on the page, which is what a
// content stream's operators are being judged by.
func (b *recordingBackend) Drawn() []string {
	var drawn []string
	for _, call := range b.calls {
		switch {
		case strings.HasPrefix(call, "fill"),
			strings.HasPrefix(call, "draw"),
			strings.HasPrefix(call, "pushGroup"),
			strings.HasPrefix(call, "popGroup"):
			drawn = append(drawn, call)
		}
	}
	return drawn
}

func (b *recordingBackend) Transform() *geom.AffineTransform { return b.transform }

func (b *recordingBackend) SetTransform(at *geom.AffineTransform) { b.transform = at }

func (b *recordingBackend) Clip() *geom.Area { return b.clip }

func (b *recordingBackend) SetClip(clip *geom.Area) { b.clip = clip }

func (b *recordingBackend) SetPaint(paint Paint) { b.paint = paint }

func (b *recordingBackend) SetStroke(stroke *Stroke) { b.stroke = stroke }

func (b *recordingBackend) SetComposite(blendMode *blend.BlendMode, alphaConstant float64) {
	b.blendMode = blendMode
	b.alphaConstant = alphaConstant
}

func (b *recordingBackend) SetAntiAliasing(on bool) { b.antiAliasing = on }

func (b *recordingBackend) SetInterpolation(interpolation Interpolation) {
	b.interpolation = interpolation
}

func (b *recordingBackend) Fill(shape geom.Shape) error {
	b.record("fill %s paint=%s", boundsOf(shape), describePaint(b.paint))
	return nil
}

func (b *recordingBackend) Draw(shape geom.Shape) error {
	b.record("draw %s paint=%s stroke=%s", boundsOf(shape), describePaint(b.paint),
		describeStroke(b.stroke))
	return nil
}

func (b *recordingBackend) DrawImage(pdImage image.PDImage, at *geom.AffineTransform,
	subsampling int) error {
	b.record("drawImage %dx%d subsampling=%d", pdImage.Width(), pdImage.Height(), subsampling)
	return nil
}

func (b *recordingBackend) DrawStencil(pdImage image.PDImage, at *geom.AffineTransform,
	paint Paint) error {
	b.record("drawStencil %dx%d paint=%s", pdImage.Width(), pdImage.Height(), describePaint(paint))
	return nil
}

func (b *recordingBackend) PushGroup(bbox *common.PDRectangle, isSoftMask, needsBackdrop bool,
	backdropColor *color.PDColor) error {
	b.record("pushGroup [%.2f %.2f %.2f %.2f] softMask=%t backdrop=%t",
		bbox.LowerLeftX(), bbox.LowerLeftY(), bbox.Width(), bbox.Height(),
		isSoftMask, needsBackdrop)
	return nil
}

func (b *recordingBackend) PopGroup() error {
	b.record("popGroup")
	return nil
}

// boundsOf describes a shape by its bounding box, which is enough to say where
// it was drawn without depending on how the path was built.
func boundsOf(shape geom.Shape) string {
	bounds := shape.Bounds2D()
	return fmt.Sprintf("[%.2f %.2f %.2f %.2f]",
		bounds.X, bounds.Y, bounds.Width, bounds.Height)
}

// describePaint names the paint and the values that decide what it puts down.
func describePaint(paint Paint) string {
	switch p := paint.(type) {
	case nil:
		return "none"
	case ColorPaint:
		return fmt.Sprintf("color(%.3f %.3f %.3f %.3f)", p.Red, p.Green, p.Blue, p.Alpha)
	case ShadingPaint:
		return fmt.Sprintf("shading(type=%d)", p.Shading.ShadingType())
	case TilingPaint:
		if p.ColorSpace == nil {
			return "tiling(colored)"
		}
		return fmt.Sprintf("tiling(uncolored, %s)", p.ColorSpace.Name())
	case SoftMaskedPaint:
		return fmt.Sprintf("softMask(%s, %s)", p.Mask.SubType().Name(), describePaint(p.Paint))
	}
	return "unknown"
}

// describeStroke names the stroke parameters a fill would not have.
func describeStroke(stroke *Stroke) string {
	if stroke == nil {
		return "none"
	}
	if stroke.Invisible {
		return "invisible"
	}
	return fmt.Sprintf("w=%.3f cap=%d join=%d miter=%.1f dash=%v phase=%.3f",
		stroke.LineWidth, stroke.LineCap, stroke.LineJoin, stroke.MiterLimit,
		stroke.DashArray, stroke.DashPhase)
}
