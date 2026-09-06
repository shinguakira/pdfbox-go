package pagenavigation

// Ported from
// pdfbox/src/test/java/org/apache/pdfbox/pdmodel/interactive/pagenavigation/PDTransitionTest.java
// and PDTransitionDirectionTest.java.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// TestTransitionDirectionCOSBase is PDTransitionDirectionTest.getCOSBase.
func TestTransitionDirectionCOSBase(t *testing.T) {
	if got := TransitionDirectionNone.COSBase(); got != cos.Base(cos.None) {
		t.Errorf("NONE.COSBase() = %v, want %v", got, cos.None)
	}
	for _, c := range []struct {
		direction PDTransitionDirection
		want      int64
	}{
		{TransitionDirectionLeftToRight, 0},
		{TransitionDirectionBottomToTop, 90},
		{TransitionDirectionRightToLeft, 180},
		{TransitionDirectionTopToBottom, 270},
		{TransitionDirectionTopLeftToBottomRight, 315},
	} {
		integer, isInteger := c.direction.COSBase().(*cos.Integer)
		if !isInteger {
			t.Fatalf("COSBase() = %T, want *cos.Integer", c.direction.COSBase())
		}
		if got := int64(integer.IntValue()); got != c.want {
			t.Errorf("COSBase().intValue() = %d, want %d", got, c.want)
		}
	}
}

// TestTransitionDefaultStyle is PDTransitionTest.defaultStyle.
func TestTransitionDefaultStyle(t *testing.T) {
	transition := NewPDTransition()
	if got := transition.Dictionary().GetCOSName(cos.Type); got != cos.Trans {
		t.Errorf("/Type = %v, want %v", got, cos.Trans)
	}
	if got, want := transition.Style(), string(TransitionStyleR); got != want {
		t.Errorf("Style() = %q, want %q", got, want)
	}
}

// TestTransitionGetStyle is PDTransitionTest.getStyle.
func TestTransitionGetStyle(t *testing.T) {
	transition := NewPDTransitionOfStyle(TransitionStyleFade)
	if got := transition.Dictionary().GetCOSName(cos.Type); got != cos.Trans {
		t.Errorf("/Type = %v, want %v", got, cos.Trans)
	}
	if got, want := transition.Style(), string(TransitionStyleFade); got != want {
		t.Errorf("Style() = %q, want %q", got, want)
	}
}

// TestTransitionDefaultValues is PDTransitionTest.defaultValues.
func TestTransitionDefaultValues(t *testing.T) {
	transition := NewPDTransitionOf(cos.NewDictionary())
	if got, want := transition.Style(), string(TransitionStyleR); got != want {
		t.Errorf("Style() = %q, want %q", got, want)
	}
	if got, want := transition.Dimension(), string(TransitionDimensionH); got != want {
		t.Errorf("Dimension() = %q, want %q", got, want)
	}
	if got, want := transition.Motion(), string(TransitionMotionI); got != want {
		t.Errorf("Motion() = %q, want %q", got, want)
	}
	if got := transition.Direction(); got != cos.Base(cos.GetInteger(0)) {
		t.Errorf("Direction() = %v, want %v", got, cos.GetInteger(0))
	}
	if got := transition.Duration(); got != 1 {
		t.Errorf("Duration() = %v, want 1", got)
	}
	if got := transition.FlyScale(); got != 1 {
		t.Errorf("FlyScale() = %v, want 1", got)
	}
	if transition.IsFlyAreaOpaque() {
		t.Error("IsFlyAreaOpaque() = true, want false")
	}
}

// TestTransitionDimension is PDTransitionTest.dimension.
func TestTransitionDimension(t *testing.T) {
	transition := NewPDTransition()
	transition.SetDimension(TransitionDimensionH)
	if got, want := transition.Dimension(), string(TransitionDimensionH); got != want {
		t.Errorf("Dimension() = %q, want %q", got, want)
	}
}

// TestTransitionDirectionNone is PDTransitionTest.directionNone.
func TestTransitionDirectionNone(t *testing.T) {
	transition := NewPDTransition()
	transition.SetDirection(TransitionDirectionNone)
	if _, isName := transition.Direction().(*cos.Name); !isName {
		t.Errorf("Direction() = %T, want *cos.Name", transition.Direction())
	}
	if got := transition.Direction(); got != cos.Base(cos.None) {
		t.Errorf("Direction() = %v, want %v", got, cos.None)
	}
}

// TestTransitionDirectionNumber is PDTransitionTest.directionNumber.
func TestTransitionDirectionNumber(t *testing.T) {
	transition := NewPDTransition()
	transition.SetDirection(TransitionDirectionLeftToRight)
	if _, isInteger := transition.Direction().(*cos.Integer); !isInteger {
		t.Errorf("Direction() = %T, want *cos.Integer", transition.Direction())
	}
	if got := transition.Direction(); got != cos.Base(cos.GetInteger(0)) {
		t.Errorf("Direction() = %v, want %v", got, cos.GetInteger(0))
	}
}

// TestTransitionMotion is PDTransitionTest.motion.
func TestTransitionMotion(t *testing.T) {
	transition := NewPDTransition()
	transition.SetMotion(TransitionMotionO)
	if got, want := transition.Motion(), string(TransitionMotionO); got != want {
		t.Errorf("Motion() = %q, want %q", got, want)
	}
}

// TestTransitionDuration is PDTransitionTest.duration.
func TestTransitionDuration(t *testing.T) {
	transition := NewPDTransition()
	transition.SetDuration(4)
	if got := transition.Duration(); got != 4 {
		t.Errorf("Duration() = %v, want 4", got)
	}
}

// TestTransitionFlyScale is PDTransitionTest.flyScale.
func TestTransitionFlyScale(t *testing.T) {
	transition := NewPDTransition()
	transition.SetFlyScale(4)
	if got := transition.FlyScale(); got != 4 {
		t.Errorf("FlyScale() = %v, want 4", got)
	}
}

// TestTransitionFlyArea is PDTransitionTest.flyArea.
func TestTransitionFlyArea(t *testing.T) {
	transition := NewPDTransition()
	transition.SetFlyAreaOpaque(true)
	if !transition.IsFlyAreaOpaque() {
		t.Error("IsFlyAreaOpaque() = false, want true")
	}
}
