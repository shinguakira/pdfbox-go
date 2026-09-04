package color

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// PDTristimulus is a colour as three CIE values.
//
// Port of org.apache.pdfbox.pdmodel.graphics.color.PDTristimulus.
type PDTristimulus struct {
	values *cos.Array
}

var _ common.COSObjectable = (*PDTristimulus)(nil)

// NewPDTristimulus returns 0 0 0.
func NewPDTristimulus() *PDTristimulus {
	values := cos.NewArray()
	values.Add(cos.FloatZero)
	values.Add(cos.FloatZero)
	values.Add(cos.FloatZero)
	return &PDTristimulus{values: values}
}

// NewPDTristimulusOfArray wraps the given array.
func NewPDTristimulusOfArray(array *cos.Array) *PDTristimulus {
	return &PDTristimulus{values: array}
}

// NewPDTristimulusOfFloats takes the first three of the given values.
func NewPDTristimulusOfFloats(array []float32) *PDTristimulus {
	values := cos.NewArray()
	for i := 0; i < len(array) && i < 3; i++ {
		values.Add(cos.NewFloat(array[i]))
	}
	return &PDTristimulus{values: values}
}

// COSObject returns the array below this value.
func (t *PDTristimulus) COSObject() cos.Base { return t.values }

// X returns the first value.
func (t *PDTristimulus) X() float32 { return t.values.Get(0).(cos.Number).FloatValue() }

// SetX sets the first value.
func (t *PDTristimulus) SetX(x float32) { t.values.Set(0, cos.NewFloat(x)) }

// Y returns the second value.
func (t *PDTristimulus) Y() float32 { return t.values.Get(1).(cos.Number).FloatValue() }

// SetY sets the second value.
func (t *PDTristimulus) SetY(y float32) { t.values.Set(1, cos.NewFloat(y)) }

// Z returns the third value.
func (t *PDTristimulus) Z() float32 { return t.values.Get(2).(cos.Number).FloatValue() }

// SetZ sets the third value.
func (t *PDTristimulus) SetZ(z float32) { t.values.Set(2, cos.NewFloat(z)) }
