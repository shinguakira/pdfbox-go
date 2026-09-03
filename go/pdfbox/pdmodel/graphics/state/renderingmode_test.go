package state

import "testing"

// Written from org.apache.pdfbox.pdmodel.graphics.state.RenderingMode. Only
// isFill is covered by the Java suite, in RenderingIntentTest.

func TestRenderingModeIntValue(t *testing.T) {
	modes := []RenderingMode{
		Fill, Stroke, FillStroke, Neither,
		FillClip, StrokeClip, FillStrokeClip, NeitherClip,
	}
	for want, mode := range modes {
		if got := mode.IntValue(); got != want {
			t.Errorf("%v.IntValue() = %d, want %d", mode, got, want)
		}
		if got := RenderingModeFromInt(want); got != mode {
			t.Errorf("RenderingModeFromInt(%d) = %v, want %v", want, got, mode)
		}
	}
}

func TestRenderingModePredicates(t *testing.T) {
	cases := []struct {
		mode                   RenderingMode
		fill, stroke, clipping bool
	}{
		{Fill, true, false, false},
		{Stroke, false, true, false},
		{FillStroke, true, true, false},
		{Neither, false, false, false},
		{FillClip, true, false, true},
		{StrokeClip, false, true, true},
		{FillStrokeClip, true, true, true},
		{NeitherClip, false, false, true},
	}
	for _, c := range cases {
		if got := c.mode.IsFill(); got != c.fill {
			t.Errorf("%v.IsFill() = %v, want %v", c.mode, got, c.fill)
		}
		if got := c.mode.IsStroke(); got != c.stroke {
			t.Errorf("%v.IsStroke() = %v, want %v", c.mode, got, c.stroke)
		}
		if got := c.mode.IsClip(); got != c.clipping {
			t.Errorf("%v.IsClip() = %v, want %v", c.mode, got, c.clipping)
		}
	}
}

// TestRenderingModeFromIntOutOfRange pins that an out-of-range value is not
// mapped to anything: Java indexes values() directly and throws
// ArrayIndexOutOfBoundsException, and every caller guards the range first.
func TestRenderingModeFromIntOutOfRange(t *testing.T) {
	for _, value := range []int{-1, 8} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("RenderingModeFromInt(%d) did not panic", value)
				}
			}()
			RenderingModeFromInt(value)
		}()
	}
}

func TestRenderingModeString(t *testing.T) {
	if got, want := FillStrokeClip.String(), "FILL_STROKE_CLIP"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestRenderingIntentString(t *testing.T) {
	if got, want := AbsoluteColorimetric.String(), "ABSOLUTE_COLORIMETRIC"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}
