package state

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/blend"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// Written from org.apache.pdfbox.pdmodel.graphics.state.PDGraphicsState; the
// Java suite has no test for it.

// TestPDGraphicsStateDefaults pins the values a content stream starts with,
// which the PDF specification fixes and a file may then change.
func TestPDGraphicsStateDefaults(t *testing.T) {
	s := NewPDGraphicsState(common.NewPDRectangleOf(0, 0, 100, 200))

	if got := s.LineWidth(); got != 1 {
		t.Errorf("LineWidth = %v, want 1", got)
	}
	if got := s.LineCap(); got != CapButt {
		t.Errorf("LineCap = %v, want CapButt", got)
	}
	if got := s.LineJoin(); got != JoinMiter {
		t.Errorf("LineJoin = %v, want JoinMiter", got)
	}
	if got := s.MiterLimit(); got != 10 {
		t.Errorf("MiterLimit = %v, want 10", got)
	}
	if got := s.AlphaConstant(); got != 1 {
		t.Errorf("AlphaConstant = %v, want 1", got)
	}
	if got := s.NonStrokeAlphaConstant(); got != 1 {
		t.Errorf("NonStrokeAlphaConstant = %v, want 1", got)
	}
	if got := s.Flatness(); got != 1 {
		t.Errorf("Flatness = %v, want 1", got)
	}
	if got := s.Smoothness(); got != 0 {
		t.Errorf("Smoothness = %v, want 0", got)
	}
	if s.BlendMode() != blend.Normal {
		t.Errorf("BlendMode = %v, want Normal", s.BlendMode())
	}
	if s.StrokingColorSpace() != color.PDColorSpace(color.DeviceGray) {
		t.Error("the stroking colour space is not DeviceGray")
	}
	if s.NonStrokingColorSpace() != color.PDColorSpace(color.DeviceGray) {
		t.Error("the non-stroking colour space is not DeviceGray")
	}
	if got := s.StrokingColor().Components(); len(got) != 1 || got[0] != 0 {
		t.Errorf("the stroking colour is %v, want black", got)
	}
	if s.TextState() == nil {
		t.Error("there is no text state")
	}
	if s.LineDashPattern() == nil {
		t.Error("there is no line dash pattern")
	}
	if !s.CurrentTransformationMatrix().Equals(util.NewMatrix()) {
		t.Errorf("the CTM is %v, want the identity", s.CurrentTransformationMatrix())
	}
	// The text matrices are set by BT, not before it.
	if s.TextMatrix() != nil || s.TextLineMatrix() != nil {
		t.Error("a new state already has text matrices")
	}
}

// TestPDGraphicsStateInitialClippingPath pins that the clipping path starts as
// the whole page.
func TestPDGraphicsStateInitialClippingPath(t *testing.T) {
	s := NewPDGraphicsState(common.NewPDRectangleOf(10, 20, 100, 200))

	paths := s.CurrentClippingPaths()
	if len(paths) != 1 {
		t.Fatalf("the state starts with %d clipping paths, want 1", len(paths))
	}
	bounds := paths[0].Bounds2D()
	if bounds.X != 10 || bounds.Y != 20 || bounds.Width != 100 || bounds.Height != 200 {
		t.Errorf("the clipping path is %v, want the page", bounds)
	}
}

func TestPDGraphicsStateIntersectClippingPath(t *testing.T) {
	s := NewPDGraphicsState(common.NewPDRectangleOf(0, 0, 100, 100))

	path := geom.NewPathFloat()
	path.MoveTo(10, 10)
	path.LineTo(20, 20)
	s.IntersectClippingPath(path)

	if got := len(s.CurrentClippingPaths()); got != 2 {
		t.Errorf("the state holds %d clipping paths, want 2", got)
	}

	// The path was copied, so going on to use it does not reach the state.
	path.LineTo(90, 90)
	if got := s.CurrentClippingPaths()[1].Bounds2D().Width; got != 10 {
		t.Errorf("changing the path afterwards changed the state: width %v, want 10", got)
	}
}

// TestPDGraphicsStateCloneSharesClippingPaths pins the lazy copy: a clone
// shares the list until one of the two adds to it, and neither then sees the
// other's additions.
func TestPDGraphicsStateCloneSharesClippingPaths(t *testing.T) {
	s := NewPDGraphicsState(common.NewPDRectangleOf(0, 0, 100, 100))
	clone := s.Clone()

	path := geom.NewPathFloat()
	path.MoveTo(0, 0)
	path.LineTo(1, 1)
	clone.IntersectClippingPath(path)

	if got := len(clone.CurrentClippingPaths()); got != 2 {
		t.Errorf("the clone holds %d clipping paths, want 2", got)
	}
	if got := len(s.CurrentClippingPaths()); got != 1 {
		t.Errorf("the original holds %d clipping paths, want 1", got)
	}
}

func TestPDGraphicsStateCloneIsIndependent(t *testing.T) {
	s := NewPDGraphicsState(common.NewPDRectangleOf(0, 0, 100, 100))
	s.SetTextMatrix(util.NewMatrix())
	s.SetTextLineMatrix(util.NewMatrix())

	clone := s.Clone()
	clone.SetLineWidth(7)
	clone.TextState().SetFontSize(24)
	clone.CurrentTransformationMatrix().SetValue(0, 0, 5)
	clone.TextMatrix().SetValue(0, 0, 5)
	clone.TextLineMatrix().SetValue(0, 0, 5)

	if got := s.LineWidth(); got != 1 {
		t.Errorf("the original line width changed to %v", got)
	}
	if got := s.TextState().FontSize(); got != 0 {
		t.Errorf("the original font size changed to %v", got)
	}
	if got := s.CurrentTransformationMatrix().Value(0, 0); got != 1 {
		t.Errorf("the original CTM changed to %v", got)
	}
	if got := s.TextMatrix().Value(0, 0); got != 1 {
		t.Errorf("the original text matrix changed to %v", got)
	}
	if got := s.TextLineMatrix().Value(0, 0); got != 1 {
		t.Errorf("the original text line matrix changed to %v", got)
	}
}

// TestPDGraphicsStateCloneKeepsNilTextMatrices pins that a state cloned before
// BT does not gain matrices it did not have.
func TestPDGraphicsStateCloneKeepsNilTextMatrices(t *testing.T) {
	clone := NewPDGraphicsState(common.NewPDRectangleOf(0, 0, 1, 1)).Clone()
	if clone.TextMatrix() != nil || clone.TextLineMatrix() != nil {
		t.Error("the clone gained text matrices")
	}
}

func TestPDGraphicsStateSetters(t *testing.T) {
	s := NewPDGraphicsState(common.NewPDRectangleOf(0, 0, 1, 1))

	s.SetLineWidth(2)
	s.SetLineCap(1)
	s.SetLineJoin(2)
	s.SetMiterLimit(4)
	s.SetStrokeAdjustment(true)
	s.SetAlphaConstant(0.5)
	s.SetNonStrokeAlphaConstant(0.25)
	s.SetAlphaSource(true)
	s.SetBlendMode(blend.Multiply)
	s.SetOverprint(true)
	s.SetNonStrokingOverprint(true)
	s.SetOverprintMode(1)
	s.SetFlatness(0.5)
	s.SetSmoothness(0.75)
	s.SetRenderingIntent(Perceptual)

	if s.LineWidth() != 2 || s.LineCap() != 1 || s.LineJoin() != 2 || s.MiterLimit() != 4 ||
		!s.IsStrokeAdjustment() || s.AlphaConstant() != 0.5 ||
		s.NonStrokeAlphaConstant() != 0.25 || !s.IsAlphaSource() ||
		s.BlendMode() != blend.Multiply || !s.IsOverprint() || !s.IsNonStrokingOverprint() ||
		s.OverprintMode() != 1 || s.Flatness() != 0.5 || s.Smoothness() != 0.75 ||
		s.RenderingIntent() == nil || *s.RenderingIntent() != Perceptual {
		t.Error("a setter did not take")
	}
}

func TestPDGraphicsStateColors(t *testing.T) {
	s := NewPDGraphicsState(common.NewPDRectangleOf(0, 0, 1, 1))
	white := color.NewPDColorOfComponents([]float32{1}, color.DeviceGray)

	s.SetStrokingColor(white)
	s.SetNonStrokingColor(white)
	if s.StrokingColor() != white || s.NonStrokingColor() != white {
		t.Error("the colours did not take")
	}
}

// TestPDGraphicsStateRenderingIntentUnspecified pins that a state which has not
// seen ri reports no intent at all. The Java field is null until an operator or
// an extended graphics state sets it, so a consumer can tell "not specified"
// from "specified as absolute"; a bare Go enum reads as its zero value, which
// is AbsoluteColorimetric.
func TestPDGraphicsStateRenderingIntentUnspecified(t *testing.T) {
	s := NewPDGraphicsState(common.NewPDRectangleOf(0, 0, 1, 1))
	if got := s.RenderingIntent(); got != nil {
		t.Errorf("RenderingIntent = %v, want nil on a state that has not seen ri", *got)
	}

	s.SetRenderingIntent(Perceptual)
	if got := s.RenderingIntent(); got == nil || *got != Perceptual {
		t.Errorf("RenderingIntent = %v, want PERCEPTUAL", got)
	}

	// A clone keeps it, and does not share the pointer.
	clone := s.Clone()
	if got := clone.RenderingIntent(); got == nil || *got != Perceptual {
		t.Errorf("the clone lost the rendering intent: %v", got)
	}
	clone.SetRenderingIntent(Saturation)
	if got := s.RenderingIntent(); *got != Perceptual {
		t.Errorf("the original changed to %v", *got)
	}
}
