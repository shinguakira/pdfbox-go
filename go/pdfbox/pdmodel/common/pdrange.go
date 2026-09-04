package common

import (
	"fmt"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// PDRange is a one dimensional interval, a minimum and a maximum, held as two
// adjacent entries of a COS array.
//
// Port of org.apache.pdfbox.pdmodel.common.PDRange. The array may hold several
// ranges end to end, which is what the starting index picks between.
type PDRange struct {
	rangeArray    *cos.Array
	startingIndex int
}

var _ COSObjectable = (*PDRange)(nil)

// NewPDRange returns the range 0 to 1.
func NewPDRange() *PDRange {
	rangeArray := cos.NewArray()
	rangeArray.Add(cos.FloatZero)
	rangeArray.Add(cos.FloatOne)
	return &PDRange{rangeArray: rangeArray}
}

// NewPDRangeOf returns the first range of the given array.
func NewPDRangeOf(rangeArray *cos.Array) *PDRange {
	return &PDRange{rangeArray: rangeArray}
}

// NewPDRangeOfIndex returns the range at the given index of the given array.
func NewPDRangeOfIndex(rangeArray *cos.Array, index int) *PDRange {
	return &PDRange{rangeArray: rangeArray, startingIndex: index}
}

// COSObject returns the array below this range.
func (r *PDRange) COSObject() cos.Base { return r.rangeArray }

// COSArray returns the array below this range.
func (r *PDRange) COSArray() *cos.Array { return r.rangeArray }

// Min returns the lower bound.
//
// Java casts the entry to COSNumber without a check, so a range whose array
// holds something else throws ClassCastException; the port asserts without the
// comma-ok and panics for the same.
func (r *PDRange) Min() float32 {
	min := r.rangeArray.GetObject(r.startingIndex * 2).(cos.Number)
	return min.FloatValue()
}

// SetMin sets the lower bound.
func (r *PDRange) SetMin(min float32) {
	r.rangeArray.Set(r.startingIndex*2, cos.NewFloat(min))
}

// Max returns the upper bound.
func (r *PDRange) Max() float32 {
	max := r.rangeArray.GetObject(r.startingIndex*2 + 1).(cos.Number)
	return max.FloatValue()
}

// SetMax sets the upper bound.
func (r *PDRange) SetMax(max float32) {
	r.rangeArray.Set(r.startingIndex*2+1, cos.NewFloat(max))
}

// String is Java's toString.
func (r *PDRange) String() string {
	return fmt.Sprintf("PDRange{%v, %v}", r.Min(), r.Max())
}
