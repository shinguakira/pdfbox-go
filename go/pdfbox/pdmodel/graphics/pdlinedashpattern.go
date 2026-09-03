// Package graphics holds the pieces of the graphics machinery that do not
// belong to one of its subtrees.
//
// Port of org.apache.pdfbox.pdmodel.graphics.
package graphics

import (
	"fmt"
	"math"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/internal/javafmt"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// PDLineDashPattern is a line dash pattern for stroking paths.
//
// Port of org.apache.pdfbox.pdmodel.graphics.PDLineDashPattern. Instances are
// immutable.
type PDLineDashPattern struct {
	phase int
	array []float32
}

var _ common.COSObjectable = (*PDLineDashPattern)(nil)

// NewPDLineDashPattern returns a solid line: no dashes, and a phase of 0.
func NewPDLineDashPattern() *PDLineDashPattern {
	return &PDLineDashPattern{array: []float32{}}
}

// NewPDLineDashPatternOf returns the dash pattern given by a dash array and a
// phase.
func NewPDLineDashPatternOf(array *cos.Array, phase int) *PDLineDashPattern {
	p := &PDLineDashPattern{array: array.ToFloatArray()}

	// PDF 2.0 specification, 8.4.3.6 Line dash pattern:
	// "If the dash phase is negative, it shall be incremented by twice the sum
	// of all lengths in the dash array until it is positive"
	if phase < 0 {
		var sum2 float32
		for _, f := range p.array {
			sum2 += f
		}
		sum2 *= 2
		if sum2 > 0 {
			var increment float64
			if float32(-phase) < sum2 {
				increment = float64(sum2)
			} else {
				increment = (math.Floor(float64(float32(-phase)/sum2)) + 1) * float64(sum2)
			}
			// Java's compound assignment narrows the double back to an int.
			phase = int(int32(float64(phase) + increment))
		} else {
			phase = 0
		}
	}
	p.phase = phase
	return p
}

// COSObject returns the dash array and phase as a COS array.
func (p *PDLineDashPattern) COSObject() cos.Base {
	array := cos.NewArray()
	array.Add(cos.ArrayOfFloats(p.array))
	array.Add(cos.GetInteger(int64(p.phase)))
	return array
}

// Phase returns the dash phase, the distance into the dash pattern at which to
// start the dash.
func (p *PDLineDashPattern) Phase() int { return p.phase }

// DashArray returns the dash array, never nil.
func (p *PDLineDashPattern) DashArray() []float32 {
	return append([]float32(nil), p.array...)
}

// String returns the Java toString form.
func (p *PDLineDashPattern) String() string {
	parts := make([]string, len(p.array))
	for i, v := range p.array {
		parts[i] = javafmt.Float32(v)
	}
	return fmt.Sprintf("PDLineDashPattern{array=[%s], phase=%d}", strings.Join(parts, ", "), p.phase)
}
