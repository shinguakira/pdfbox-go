package blend

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// Written from org.apache.pdfbox.pdmodel.graphics.blend.BlendMode.
//
// The header here used to say the Java suite covered the blend functions only
// through rendered images. It does not: BlendModeTest calls blendChannel with
// exact values. It is ported at the end of this file.

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

// --- Port of org.apache.pdfbox.pdmodel.graphics.blend.BlendModeTest -------
//
// The header above says the Java suite covers the blend functions through
// rendered images. It does not: BlendModeTest calls blendChannel directly with
// exact values, seventeen cases of it. Ported below; see
// migration/tasks/track-test-backfill.md.

// TestInstances is testInstances: every name maps to its mode, /Compatible maps
// to Normal, an array is read through to its first element, and an array of
// something that is not a name falls back to Normal.
func TestInstances(t *testing.T) {
	for _, row := range []struct {
		name *cos.Name
		want *BlendMode
	}{
		{cos.Normal, Normal},
		{cos.Compatible, Normal},
		{cos.Multiply, Multiply},
		{cos.Screen, Screen},
		{cos.Overlay, Overlay},
		{cos.Darken, Darken},
		{cos.Lighten, Lighten},
		{cos.ColorDodge, ColorDodge},
		{cos.ColorBurn, ColorBurn},
		{cos.HardLight, HardLight},
		{cos.SoftLight, SoftLight},
		{cos.Difference, Difference},
		{cos.Exclusion, Exclusion},
		{cos.Hue, Hue},
		{cos.Saturation, Saturation},
		{cos.Luminosity, Luminosity},
		{cos.Color, Color},
	} {
		t.Run(row.name.Name(), func(t *testing.T) {
			if got := GetInstance(row.name); got != row.want {
				t.Errorf("GetInstance(%s) = %v, want %v", row.name.Name(), got, row.want)
			}
		})
	}

	t.Run("an array of one name", func(t *testing.T) {
		array := cos.NewArray()
		array.Add(cos.Overlay)
		if got := GetInstance(array); got != Overlay {
			t.Errorf("GetInstance([/Overlay]) = %v, want Overlay", got)
		}
	})

	t.Run("an array of something else", func(t *testing.T) {
		array := cos.NewArray()
		array.Add(cos.GetInteger(0))
		if got := GetInstance(array); got != Normal {
			t.Errorf("GetInstance([0]) = %v, want Normal", got)
		}
	})
}

// TestSeparableBlendModes is the twelve separable testBlendMode* cases: each
// has a channel function and no whole-colour one, and answers a fixed value.
//
// The inputs are outside 0..1 in several cases -- blendChannel(3, 5) -- which
// is deliberate in the Java: the functions are pure arithmetic and are not
// asked to clamp.
func TestSeparableBlendModes(t *testing.T) {
	for _, row := range []struct {
		name string
		mode *BlendMode
		want *cos.Name
		// each pair is src, dest, expected
		values [][3]float32
	}{
		{"Normal", Normal, cos.Normal, [][3]float32{{3, 5, 3}}},
		{"Multiply", Multiply, cos.Multiply, [][3]float32{{3, 5, 15}}},
		{"Screen", Screen, cos.Screen, [][3]float32{{3, 5, -7}}},
		{"Overlay", Overlay, cos.Overlay, [][3]float32{{1, 0, 0}, {0.5, 0.3, 0.3}}},
		{"Darken", Darken, cos.Darken, [][3]float32{{3, 5, 3}}},
		{"Lighten", Lighten, cos.Lighten, [][3]float32{{3, 5, 5}}},
		{"ColorDodge", ColorDodge, cos.ColorDodge, [][3]float32{{1, 0, 0}, {0.3, 0.7, 1}}},
		{"ColorBurn", ColorBurn, cos.ColorBurn, [][3]float32{{0, 1, 1}, {0.7, 0.3, 0}}},
		{"HardLight", HardLight, cos.HardLight,
			[][3]float32{{0, 0.5, 0}, {0.2, 0.5, 0.2}, {0.6, 0.4, 0.52}}},
		{"SoftLight", SoftLight, cos.SoftLight,
			[][3]float32{{0, 0.5, 0.25}, {0.2, 0.5, 0.35}, {0.5, 0.2, 0.2}}},
		{"Difference", Difference, cos.Difference, [][3]float32{{3, 5, 2}}},
		{"Exclusion", Exclusion, cos.Exclusion, nil},
	} {
		t.Run(row.name, func(t *testing.T) {
			if !row.mode.IsSeparableBlendMode() {
				t.Error("IsSeparableBlendMode() = false, want true")
			}
			if row.mode.BlendFunction() != nil {
				t.Error("BlendFunction() is set, want nil for a separable mode")
			}
			blend := row.mode.BlendChannelFunction()
			if blend == nil {
				t.Fatal("BlendChannelFunction() = nil, want the channel function")
			}
			if got := row.mode.COSName(); got != row.want {
				t.Errorf("COSName() = %v, want %v", got, row.want)
			}
			for _, v := range row.values {
				if got := blend(v[0], v[1]); got != v[2] {
					t.Errorf("blendChannel(%v, %v) = %v, want %v",
						v[0], v[1], got, v[2])
				}
			}
		})
	}

	// Compatible is Normal but keeps its own name in Java's getCOSName, which
	// testBlendModeNormal checks at the end.
	if got := GetInstance(cos.Compatible).COSName(); got != cos.Normal {
		t.Errorf("Compatible's COSName() = %v, want /Normal", got)
	}
}

// TestNonSeparableBlendModes is the four non-separable cases: they have a
// whole-colour function and no channel one.
func TestNonSeparableBlendModes(t *testing.T) {
	for _, row := range []struct {
		name string
		mode *BlendMode
		want *cos.Name
	}{
		{"Hue", Hue, cos.Hue},
		{"Saturation", Saturation, cos.Saturation},
		{"Luminosity", Luminosity, cos.Luminosity},
		{"Color", Color, cos.Color},
	} {
		t.Run(row.name, func(t *testing.T) {
			if row.mode.IsSeparableBlendMode() {
				t.Error("IsSeparableBlendMode() = true, want false")
			}
			if row.mode.BlendFunction() == nil {
				t.Error("BlendFunction() = nil, want the whole-colour function")
			}
			if row.mode.BlendChannelFunction() != nil {
				t.Error("BlendChannelFunction() is set, want nil for a " +
					"non-separable mode")
			}
			if got := row.mode.COSName(); got != row.want {
				t.Errorf("COSName() = %v, want %v", got, row.want)
			}
		})
	}
}
