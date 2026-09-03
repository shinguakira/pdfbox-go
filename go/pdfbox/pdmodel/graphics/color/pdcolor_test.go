package color

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// Written from org.apache.pdfbox.pdmodel.graphics.color.PDColor and
// PDDeviceGray. The Java suite covers colour through rendered images, which
// this port has not reached.

func TestPDColorOfComponents(t *testing.T) {
	c := NewPDColorOfComponents([]float32{0.5}, DeviceGray)

	if got := c.Components(); len(got) != 1 || got[0] != 0.5 {
		t.Errorf("Components = %v, want [0.5]", got)
	}
	if c.IsPattern() {
		t.Error("a colour with components is a pattern")
	}
	if c.ColorSpace() != PDColorSpace(DeviceGray) {
		t.Error("the colour did not keep its colour space")
	}
}

// TestPDColorComponentsAreCopied pins that the components cannot be changed
// through the slice handed in or the one handed back: PDColor is immutable.
func TestPDColorComponentsAreCopied(t *testing.T) {
	given := []float32{0.5}
	c := NewPDColorOfComponents(given, DeviceGray)

	given[0] = 1
	if got := c.Components()[0]; got != 0.5 {
		t.Errorf("changing the slice that was passed in changed the colour to %v", got)
	}

	c.Components()[0] = 1
	if got := c.Components()[0]; got != 0.5 {
		t.Errorf("changing the slice that came out changed the colour to %v", got)
	}
}

// TestPDColorComponentsPadded covers PDFBOX-4279: a colour with fewer
// components than its colour space takes comes back padded rather than short,
// so a caller reading the third of three does not run off the end.
func TestPDColorComponentsPadded(t *testing.T) {
	c := NewPDColorOfComponents(nil, DeviceGray)
	if got := c.Components(); len(got) != 1 || got[0] != 0 {
		t.Errorf("Components = %v, want [0]", got)
	}
}

func TestPDColorOfCOSArray(t *testing.T) {
	c := NewPDColorOfCOSArray(cos.ArrayOfFloats([]float32{0.25}), DeviceGray)
	if got := c.Components(); len(got) != 1 || got[0] != 0.25 {
		t.Errorf("Components = %v, want [0.25]", got)
	}
	if c.IsPattern() {
		t.Error("a colour with components is a pattern")
	}
}

// TestPDColorOfCOSArrayPattern pins that a trailing name makes the value a
// pattern, with the components before it being the colour to paint it in.
func TestPDColorOfCOSArrayPattern(t *testing.T) {
	array := cos.ArrayOfFloats([]float32{0.1, 0.2})
	array.Add(cos.GetPDFName("P0"))

	c := NewPDColorOfCOSArray(array, nil)
	if !c.IsPattern() {
		t.Error("a colour ending in a name is not a pattern")
	}
	if got := c.PatternName().Name(); got != "P0" {
		t.Errorf("PatternName = %q, want P0", got)
	}
	if got := c.Components(); len(got) != 2 || got[0] != 0.1 || got[1] != 0.2 {
		t.Errorf("Components = %v, want [0.1 0.2]", got)
	}
}

// TestPDColorOfCOSArrayIgnoresNonNumbers pins that a component that is not a
// number is left at zero rather than ending the read.
func TestPDColorOfCOSArrayIgnoresNonNumbers(t *testing.T) {
	array := cos.NewArray()
	array.Add(cos.NewFloat(0.5))
	array.Add(cos.NewDictionary())
	array.Add(cos.NewFloat(0.75))

	c := NewPDColorOfCOSArray(array, nil)
	got := c.Components()
	if len(got) != 3 || got[0] != 0.5 || got[1] != 0 || got[2] != 0.75 {
		t.Errorf("Components = %v, want [0.5 0 0.75]", got)
	}
}

func TestPDColorOfPattern(t *testing.T) {
	c := NewPDColorOfPattern(cos.GetPDFName("P0"), nil)
	if !c.IsPattern() {
		t.Error("a colour built from a pattern name is not a pattern")
	}
	if got := c.Components(); len(got) != 0 {
		t.Errorf("Components = %v, want none", got)
	}
}

func TestPDColorToCOSArray(t *testing.T) {
	array := NewPDColorOfComponents([]float32{0.5, 0.25}, nil).ToCOSArray()
	if array.Size() != 2 {
		t.Fatalf("array = %v, want two components", array)
	}

	withPattern := NewPDColorOfPatternComponents(
		[]float32{0.5}, cos.GetPDFName("P0"), nil).ToCOSArray()
	if withPattern.Size() != 2 {
		t.Fatalf("array = %v, want a component and a name", withPattern)
	}
	if got, ok := withPattern.Get(1).(*cos.Name); !ok || got.Name() != "P0" {
		t.Errorf("the last entry is %v, want the pattern name", withPattern.Get(1))
	}
}

func TestPDColorToRGB(t *testing.T) {
	black, err := NewPDColorOfComponents([]float32{0}, DeviceGray).ToRGB()
	if err != nil {
		t.Fatalf("ToRGB: %v", err)
	}
	if black != 0x000000 {
		t.Errorf("black = %#06x, want 0x000000", black)
	}

	white, err := NewPDColorOfComponents([]float32{1}, DeviceGray).ToRGB()
	if err != nil {
		t.Fatalf("ToRGB: %v", err)
	}
	if white != 0xffffff {
		t.Errorf("white = %#06x, want 0xffffff", white)
	}

	// A half-tone rounds half up, as Java's Math.round does: 127.5 becomes 128.
	half, err := NewPDColorOfComponents([]float32{0.5}, DeviceGray).ToRGB()
	if err != nil {
		t.Fatalf("ToRGB: %v", err)
	}
	if half != 0x808080 {
		t.Errorf("half grey = %#06x, want 0x808080", half)
	}
}

func TestPDDeviceGray(t *testing.T) {
	if got := DeviceGray.Name(); got != "DeviceGray" {
		t.Errorf("Name = %q, want DeviceGray", got)
	}
	if got := DeviceGray.NumberOfComponents(); got != 1 {
		t.Errorf("NumberOfComponents = %d, want 1", got)
	}
	if got := DeviceGray.DefaultDecode(8); len(got) != 2 || got[0] != 0 || got[1] != 1 {
		t.Errorf("DefaultDecode = %v, want [0 1]", got)
	}
	if got := DeviceGray.COSObject(); got != cos.Base(cos.DeviceGray) {
		t.Errorf("COSObject = %v, want /DeviceGray", got)
	}
	if got := DeviceGray.String(); got != "DeviceGray" {
		t.Errorf("String = %q, want DeviceGray", got)
	}
}

// TestPDDeviceGrayInitialColor pins that a content stream starts in black, and
// that the initial colour knows the space it belongs to.
func TestPDDeviceGrayInitialColor(t *testing.T) {
	initial := DeviceGray.InitialColor()
	if got := initial.Components(); len(got) != 1 || got[0] != 0 {
		t.Errorf("initial colour = %v, want [0]", got)
	}
	if initial.ColorSpace() != PDColorSpace(DeviceGray) {
		t.Error("the initial colour does not know its colour space")
	}
	if DeviceGray.InitialColor() != initial {
		t.Error("the initial colour is rebuilt on each call")
	}
}

func TestPDDeviceGrayToRGB(t *testing.T) {
	got, err := DeviceGray.ToRGB([]float32{0.25})
	if err != nil {
		t.Fatalf("ToRGB: %v", err)
	}
	if len(got) != 3 || got[0] != 0.25 || got[1] != 0.25 || got[2] != 0.25 {
		t.Errorf("ToRGB = %v, want [0.25 0.25 0.25]", got)
	}
}

// TestPDColorComponentsNeverNil pins the doc contract. Java returns
// components.clone(), and cloning an empty array gives a non-null empty array;
// appending to a nil slice with nothing to append gives nil.
func TestPDColorComponentsNeverNil(t *testing.T) {
	pattern := NewPDColorOfPattern(cos.GetPDFName("P0"), nil)
	if got := pattern.Components(); got == nil {
		t.Error("Components() of a pattern colour is nil, want an empty slice")
	}
	if got := NewPDColorOfComponents(nil, nil).Components(); got == nil {
		t.Error("Components() with no colour space is nil, want an empty slice")
	}
}
