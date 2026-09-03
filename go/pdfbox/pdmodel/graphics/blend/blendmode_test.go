package blend

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// Written from org.apache.pdfbox.pdmodel.graphics.blend.BlendMode; the Java
// suite covers the blend functions through rendered images, which this port has
// not reached.

func TestGetInstance(t *testing.T) {
	cases := []struct {
		name *cos.Name
		want *BlendMode
	}{
		{cos.Multiply, Multiply},
		{cos.Screen, Screen},
		{cos.Luminosity, Luminosity},
		// Compatible is not a mode of its own; it means Normal.
		{cos.Compatible, Normal},
		// Anything unrecognised falls back to Normal rather than failing.
		{cos.GetPDFName("NoSuchMode"), Normal},
	}
	for _, c := range cases {
		if got := GetInstance(c.name); got != c.want {
			t.Errorf("GetInstance(%v) = %v, want %v", c.name, got, c.want)
		}
	}

	if got := GetInstance(nil); got != Normal {
		t.Errorf("GetInstance(nil) = %v, want Normal", got)
	}
}

// TestGetInstanceArray pins that an array of names is read left to right and
// the first recognised one wins, which is how a file offers a fallback.
func TestGetInstanceArray(t *testing.T) {
	array := cos.NewArray()
	array.Add(cos.GetPDFName("NoSuchMode"))
	array.Add(cos.Darken)
	array.Add(cos.Lighten)

	if got := GetInstance(array); got != Darken {
		t.Errorf("GetInstance = %v, want Darken", got)
	}

	empty := cos.NewArray()
	empty.Add(cos.NewDictionary())
	if got := GetInstance(empty); got != Normal {
		t.Errorf("GetInstance of an array naming nothing = %v, want Normal", got)
	}
}

func TestSeparability(t *testing.T) {
	if !Multiply.IsSeparableBlendMode() {
		t.Error("Multiply is not separable")
	}
	if Multiply.BlendChannelFunction() == nil {
		t.Error("a separable mode has no channel function")
	}
	if Multiply.BlendFunction() != nil {
		t.Error("a separable mode has a non-separable function")
	}

	if Luminosity.IsSeparableBlendMode() {
		t.Error("Luminosity is separable")
	}
	if Luminosity.BlendFunction() == nil {
		t.Error("a non-separable mode has no blend function")
	}
	if Luminosity.BlendChannelFunction() != nil {
		t.Error("a non-separable mode has a channel function")
	}
}

func TestSeparableFunctions(t *testing.T) {
	cases := []struct {
		mode      *BlendMode
		src, dest float32
		want      float32
	}{
		{Normal, 0.25, 0.75, 0.25},
		{Multiply, 0.5, 0.5, 0.25},
		{Screen, 0.5, 0.5, 0.75},
		{Darken, 0.25, 0.75, 0.25},
		{Lighten, 0.25, 0.75, 0.75},
		{Difference, 0.25, 0.75, 0.5},
		{Exclusion, 0.5, 0.5, 0.5},
		// Overlay branches on the backdrop, HardLight on the source.
		{Overlay, 0.5, 0.25, 0.25},
		{HardLight, 0.25, 0.5, 0.25},
		// The dodge and burn edge cases from the PDF 2.0 specification.
		{ColorDodge, 0.5, 0, 0},
		{ColorDodge, 0.5, 0.5, 1},
		{ColorDodge, 0.75, 0.125, 0.5},
		{ColorBurn, 0.5, 1, 1},
		{ColorBurn, 0.25, 0.5, 0},
		{ColorBurn, 0.5, 0.75, 0.5},
	}
	for _, c := range cases {
		got := c.mode.BlendChannelFunction()(c.src, c.dest)
		if got != c.want {
			t.Errorf("%v(%v, %v) = %v, want %v", c.mode.COSName(), c.src, c.dest, got, c.want)
		}
	}
}

// TestSaturationOfGreyBackdrop pins the divide-by-zero guard: a backdrop with
// no saturation of its own comes back as itself.
func TestSaturationOfGreyBackdrop(t *testing.T) {
	result := make([]float32, 3)
	Saturation.BlendFunction()([]float32{1, 0, 0}, []float32{0.5, 0.5, 0.5}, result)

	if result[0] != result[1] || result[1] != result[2] {
		t.Errorf("a grey backdrop came back as %v, want a grey", result)
	}
}

func TestLuminosityKeepsHue(t *testing.T) {
	result := make([]float32, 3)
	// A white source over a red backdrop lifts it to white.
	Luminosity.BlendFunction()([]float32{1, 1, 1}, []float32{1, 0, 0}, result)

	for i, v := range result {
		if v < 0.99 {
			t.Errorf("component %d = %v, want about 1", i, v)
		}
	}
}

func TestBlendModeString(t *testing.T) {
	if got, want := Multiply.String(), "BlendMode{name=Multiply, isSeparable=true}"; got != want {
		t.Errorf("String = %q, want %q", got, want)
	}
	if got, want := Hue.String(), "BlendMode{name=Hue, isSeparable=false}"; got != want {
		t.Errorf("String = %q, want %q", got, want)
	}
}
