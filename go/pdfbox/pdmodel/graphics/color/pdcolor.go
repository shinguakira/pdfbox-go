package color

import (
	"fmt"
	"log/slog"
	"math"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/internal/javafmt"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// PDColor is a colour value, consisting of one or more colour components, or
// for pattern colour spaces, a name and optional colour components. Colour
// values are not associated with any given colour space.
//
// Port of org.apache.pdfbox.pdmodel.graphics.color.PDColor. Instances are
// immutable.
type PDColor struct {
	components  []float32
	patternName *cos.Name
	colorSpace  PDColorSpace
}

// NewPDColorOfCOSArray returns the colour value held by a COS array, read in
// the given colour space.
func NewPDColorOfCOSArray(array *cos.Array, colorSpace PDColorSpace) *PDColor {
	c := &PDColor{colorSpace: colorSpace}
	if !array.IsEmpty() {
		if _, ok := array.Get(array.Size() - 1).(*cos.Name); ok {
			// color components (optional), for the color of an uncoloured tiling pattern
			c.components = make([]float32, array.Size()-1)
			c.initComponents(array)

			// pattern name (required)
			base := array.Get(array.Size() - 1)
			if name, ok := base.(*cos.Name); ok {
				c.patternName = name
			} else {
				// Java re-tests the very value it just matched on, so this
				// cannot be reached; ported as written.
				slog.Warn("pattern name isn't a name, ignored", "array", array)
				c.patternName = cos.GetPDFName("Unknown")
			}
			return c
		}
	}
	// color components only
	c.components = make([]float32, array.Size())
	c.initComponents(array)
	return c
}

func (c *PDColor) initComponents(array *cos.Array) {
	for i := range c.components {
		base := array.Get(i)
		if number, ok := base.(cos.Number); ok {
			c.components[i] = number.FloatValue()
		} else {
			slog.Warn("color component isn't a number, ignored", "index", i, "array", array)
		}
	}
}

// NewPDColorOfComponents returns the colour value with the given components,
// read in the given colour space.
func NewPDColorOfComponents(components []float32, colorSpace PDColorSpace) *PDColor {
	c := &PDColor{
		components: append([]float32(nil), components...),
		colorSpace: colorSpace,
	}
	if colorSpace != nil && colorSpace.NumberOfComponents() != len(components) {
		// PDFBOX-5882
		slog.Warn("colorspace component count doesn't match components length",
			"colorspace", colorSpace.NumberOfComponents(), "components", len(components))
	}
	return c
}

// NewPDColorOfPattern returns the colour value naming a pattern in a pattern
// dictionary.
func NewPDColorOfPattern(patternName *cos.Name, colorSpace PDColorSpace) *PDColor {
	return &PDColor{
		components:  []float32{},
		patternName: patternName,
		colorSpace:  colorSpace,
	}
}

// NewPDColorOfPatternComponents returns the colour value naming a pattern and
// carrying the components it is to be painted with.
func NewPDColorOfPatternComponents(components []float32, patternName *cos.Name, colorSpace PDColorSpace) *PDColor {
	c := &PDColor{
		components:  append([]float32(nil), components...),
		patternName: patternName,
		colorSpace:  colorSpace,
	}
	if pattern, ok := colorSpace.(PatternColorSpace); ok {
		ucs := pattern.UnderlyingColorSpace()
		if ucs != nil && ucs.NumberOfComponents() != len(components) {
			// PDFBOX-5882
			slog.Warn("pattern colorspace component count doesn't match components length",
				"colorspace", ucs.NumberOfComponents(), "components", len(components))
		}
	}
	return c
}

// Components returns the components of this colour value, never nil.
func (c *PDColor) Components() []float32 {
	if _, isPattern := c.colorSpace.(PatternColorSpace); isPattern || c.colorSpace == nil {
		// colorspace of the pattern color isn't known, so just clone
		// null colorspace can happen with empty annotation color
		// see PDFBOX-3351-538928-p4.pdf
		out := make([]float32, len(c.components))
		copy(out, c.components)
		return out
	}
	// PDFBOX-4279: a copy of the colour space's length, in case the array is
	// too small — the extra components come back as zero rather than missing.
	out := make([]float32, c.colorSpace.NumberOfComponents())
	copy(out, c.components)
	return out
}

// PatternName returns the pattern name from this colour value.
func (c *PDColor) PatternName() *cos.Name { return c.patternName }

// IsPattern reports whether this colour value is a pattern.
func (c *PDColor) IsPattern() bool { return c.patternName != nil }

// ToRGB returns the packed RGB value for this colour.
//
// It is not defined for a pattern, where Java throws
// UnsupportedOperationException from the colour space it reaches.
func (c *PDColor) ToRGB() (int, error) {
	floats, err := c.colorSpace.ToRGB(c.components)
	if err != nil {
		return 0, err
	}
	r := javaRound(floats[0] * 255)
	g := javaRound(floats[1] * 255)
	b := javaRound(floats[2] * 255)
	rgb := r
	rgb = (rgb << 8) + g
	rgb = (rgb << 8) + b
	return rgb, nil
}

// javaRound rounds half up, as Java's Math.round(float) does, rather than to
// even as Go's math.Round does at a tie.
func javaRound(v float32) int {
	return int(math.Floor(float64(v) + 0.5))
}

// ToCOSArray returns the colour component values as a COS array.
func (c *PDColor) ToCOSArray() *cos.Array {
	array := cos.ArrayOfFloats(c.components)
	if c.patternName != nil {
		array.Add(c.patternName)
	}
	return array
}

// ColorSpace returns the colour space in which this colour value is defined.
func (c *PDColor) ColorSpace() PDColorSpace { return c.colorSpace }

// String returns the Java toString form.
func (c *PDColor) String() string {
	parts := make([]string, len(c.components))
	for i, component := range c.components {
		parts[i] = javafmt.Float32(component)
	}
	return fmt.Sprintf("PDColor{components=[%s], patternName=%v, colorSpace=%v}",
		strings.Join(parts, ", "), c.patternName, c.colorSpace)
}
