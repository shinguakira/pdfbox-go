package destination

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// PDPageFitDestination fits the whole page in the window.
//
// Port of PDPageFitDestination.
type PDPageFitDestination struct {
	PDPageDestination
}

var _ PDDestination = (*PDPageFitDestination)(nil)

// NewPDPageFitDestination creates a new /Fit destination.
func NewPDPageFitDestination() *PDPageFitDestination {
	d := &PDPageFitDestination{}
	d.InitPageDestination()
	d.Array.GrowToSize(2)
	d.Array.SetName(1, typeFit)
	return d
}

// NewPDPageFitDestinationOf creates a destination over the given array.
func NewPDPageFitDestinationOf(arr *cos.Array) *PDPageFitDestination {
	d := &PDPageFitDestination{}
	d.InitPageDestinationOf(arr)
	return d
}

// FitBoundingBox reports whether the bounding box is fitted rather than the
// whole page.
func (d *PDPageFitDestination) FitBoundingBox() bool {
	return d.Array.GetName(1, "") == typeFitBounded
}

// SetFitBoundingBox sets whether the bounding box is fitted.
func (d *PDPageFitDestination) SetFitBoundingBox(fitBoundingBox bool) {
	d.Array.GrowToSize(2)
	if fitBoundingBox {
		d.Array.SetName(1, typeFitBounded)
	} else {
		d.Array.SetName(1, typeFit)
	}
}

// PDPageFitHeightDestination fits the height of the page in the window.
//
// Port of PDPageFitHeightDestination, whose type name is /FitV.
type PDPageFitHeightDestination struct {
	PDPageDestination
}

var _ PDDestination = (*PDPageFitHeightDestination)(nil)

// NewPDPageFitHeightDestination creates a new /FitV destination.
func NewPDPageFitHeightDestination() *PDPageFitHeightDestination {
	d := &PDPageFitHeightDestination{}
	d.InitPageDestination()
	d.Array.GrowToSize(3)
	d.Array.SetName(1, typeFitV)
	return d
}

// NewPDPageFitHeightDestinationOf creates a destination over the given array.
func NewPDPageFitHeightDestinationOf(arr *cos.Array) *PDPageFitHeightDestination {
	d := &PDPageFitHeightDestination{}
	d.InitPageDestinationOf(arr)
	return d
}

// Left returns the left edge, or -1 where the array does not say.
func (d *PDPageFitHeightDestination) Left() int { return d.Array.GetInt(2) }

// SetLeft sets the left edge; -1 clears it.
func (d *PDPageFitHeightDestination) SetLeft(x int) {
	d.Array.GrowToSize(3)
	if x == -1 {
		d.Array.Set(2, nil)
	} else {
		d.Array.SetInt(2, x)
	}
}

// FitBoundingBox reports whether the bounding box is fitted.
func (d *PDPageFitHeightDestination) FitBoundingBox() bool {
	return d.Array.GetName(1, "") == typeFitVBounded
}

// SetFitBoundingBox sets whether the bounding box is fitted.
func (d *PDPageFitHeightDestination) SetFitBoundingBox(fitBoundingBox bool) {
	d.Array.GrowToSize(3)
	if fitBoundingBox {
		d.Array.SetName(1, typeFitVBounded)
	} else {
		d.Array.SetName(1, typeFitV)
	}
}

// PDPageFitWidthDestination fits the width of the page in the window.
//
// Port of PDPageFitWidthDestination, whose type name is /FitH.
type PDPageFitWidthDestination struct {
	PDPageDestination
}

var _ PDDestination = (*PDPageFitWidthDestination)(nil)

// NewPDPageFitWidthDestination creates a new /FitH destination.
func NewPDPageFitWidthDestination() *PDPageFitWidthDestination {
	d := &PDPageFitWidthDestination{}
	d.InitPageDestination()
	d.Array.GrowToSize(3)
	d.Array.SetName(1, typeFitH)
	return d
}

// NewPDPageFitWidthDestinationOf creates a destination over the given array.
func NewPDPageFitWidthDestinationOf(arr *cos.Array) *PDPageFitWidthDestination {
	d := &PDPageFitWidthDestination{}
	d.InitPageDestinationOf(arr)
	return d
}

// Top returns the top edge, or -1 where the array does not say.
func (d *PDPageFitWidthDestination) Top() int { return d.Array.GetInt(2) }

// SetTop sets the top edge; -1 clears it.
func (d *PDPageFitWidthDestination) SetTop(y int) {
	d.Array.GrowToSize(3)
	if y == -1 {
		d.Array.Set(2, nil)
	} else {
		d.Array.SetInt(2, y)
	}
}

// FitBoundingBox reports whether the bounding box is fitted.
func (d *PDPageFitWidthDestination) FitBoundingBox() bool {
	return d.Array.GetName(1, "") == typeFitHBounded
}

// SetFitBoundingBox sets whether the bounding box is fitted.
func (d *PDPageFitWidthDestination) SetFitBoundingBox(fitBoundingBox bool) {
	d.Array.GrowToSize(3)
	if fitBoundingBox {
		d.Array.SetName(1, typeFitHBounded)
	} else {
		d.Array.SetName(1, typeFitH)
	}
}

// PDPageFitRectangleDestination fits a rectangle of the page in the window.
//
// Port of PDPageFitRectangleDestination.
type PDPageFitRectangleDestination struct {
	PDPageDestination
}

var _ PDDestination = (*PDPageFitRectangleDestination)(nil)

// NewPDPageFitRectangleDestination creates a new /FitR destination.
func NewPDPageFitRectangleDestination() *PDPageFitRectangleDestination {
	d := &PDPageFitRectangleDestination{}
	d.InitPageDestination()
	d.Array.GrowToSize(6)
	d.Array.SetName(1, typeFitR)
	return d
}

// NewPDPageFitRectangleDestinationOf creates a destination over the given
// array.
func NewPDPageFitRectangleDestinationOf(arr *cos.Array) *PDPageFitRectangleDestination {
	d := &PDPageFitRectangleDestination{}
	d.InitPageDestinationOf(arr)
	return d
}

// Left returns the left edge, or -1.
func (d *PDPageFitRectangleDestination) Left() int { return d.Array.GetInt(2) }

// SetLeft sets the left edge; -1 clears it.
func (d *PDPageFitRectangleDestination) SetLeft(x int) { d.setBound(2, x) }

// Bottom returns the bottom edge, or -1.
func (d *PDPageFitRectangleDestination) Bottom() int { return d.Array.GetInt(3) }

// SetBottom sets the bottom edge; -1 clears it.
func (d *PDPageFitRectangleDestination) SetBottom(y int) { d.setBound(3, y) }

// Right returns the right edge, or -1.
func (d *PDPageFitRectangleDestination) Right() int { return d.Array.GetInt(4) }

// SetRight sets the right edge; -1 clears it.
func (d *PDPageFitRectangleDestination) SetRight(x int) { d.setBound(4, x) }

// Top returns the top edge, or -1.
func (d *PDPageFitRectangleDestination) Top() int { return d.Array.GetInt(5) }

// SetTop sets the top edge; -1 clears it.
func (d *PDPageFitRectangleDestination) SetTop(y int) { d.setBound(5, y) }

// setBound is the body the four setters share.
func (d *PDPageFitRectangleDestination) setBound(index, value int) {
	d.Array.GrowToSize(6)
	if value == -1 {
		d.Array.Set(index, nil)
	} else {
		d.Array.SetInt(index, value)
	}
}

// PDPageXYZDestination positions a corner of the page and sets the zoom.
//
// Port of PDPageXYZDestination.
type PDPageXYZDestination struct {
	PDPageDestination
}

var _ PDDestination = (*PDPageXYZDestination)(nil)

// NewPDPageXYZDestination creates a new /XYZ destination.
func NewPDPageXYZDestination() *PDPageXYZDestination {
	d := &PDPageXYZDestination{}
	d.InitPageDestination()
	d.Array.GrowToSize(5)
	d.Array.SetName(1, typeXYZ)
	return d
}

// NewPDPageXYZDestinationOf creates a destination over the given array.
func NewPDPageXYZDestinationOf(arr *cos.Array) *PDPageXYZDestination {
	d := &PDPageXYZDestination{}
	d.InitPageDestinationOf(arr)
	return d
}

// Left returns the left edge, or -1.
func (d *PDPageXYZDestination) Left() int { return d.Array.GetInt(2) }

// SetLeft sets the left edge; -1 clears it.
func (d *PDPageXYZDestination) SetLeft(x int) {
	d.Array.GrowToSize(5)
	if x == -1 {
		d.Array.Set(2, nil)
	} else {
		d.Array.SetInt(2, x)
	}
}

// Top returns the top edge, or -1.
func (d *PDPageXYZDestination) Top() int { return d.Array.GetInt(3) }

// SetTop sets the top edge; -1 clears it.
func (d *PDPageXYZDestination) SetTop(y int) {
	d.Array.GrowToSize(5)
	if y == -1 {
		d.Array.Set(3, nil)
	} else {
		d.Array.SetInt(3, y)
	}
}

// Zoom returns the zoom factor, or -1 where the array does not say.
func (d *PDPageXYZDestination) Zoom() float32 {
	if obj, ok := d.Array.GetObject(4).(cos.Number); ok {
		return obj.FloatValue()
	}
	return -1
}

// SetZoom sets the zoom factor; -1 clears it.
func (d *PDPageXYZDestination) SetZoom(zoom float32) {
	d.Array.GrowToSize(5)
	if zoom == -1 {
		d.Array.Set(4, nil)
	} else {
		d.Array.Set(4, cos.NewFloat(zoom))
	}
}
