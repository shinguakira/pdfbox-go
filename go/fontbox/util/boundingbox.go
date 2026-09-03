// Package util holds the helpers fontbox keeps outside its font models.
//
// Port of org.apache.fontbox.util.
package util

import "github.com/shinguakira/pdfbox-go/go/internal/javafmt"

// BoundingBox is a bounding box. This was originally written for the AMF
// parser.
//
// Port of org.apache.fontbox.util.BoundingBox.
type BoundingBox struct {
	lowerLeftX  float32
	lowerLeftY  float32
	upperRightX float32
	upperRightY float32
}

// NewBoundingBox returns the box with every corner at the origin.
func NewBoundingBox() *BoundingBox { return &BoundingBox{} }

// NewBoundingBoxOf returns the box with the given corners.
func NewBoundingBoxOf(minX, minY, maxX, maxY float32) *BoundingBox {
	return &BoundingBox{
		lowerLeftX:  minX,
		lowerLeftY:  minY,
		upperRightX: maxX,
		upperRightY: maxY,
	}
}

// NewBoundingBoxOfNumbers returns the box given by a list of four numbers.
func NewBoundingBoxOfNumbers(numbers []float32) *BoundingBox {
	return NewBoundingBoxOf(numbers[0], numbers[1], numbers[2], numbers[3])
}

// LowerLeftX returns the lower left x value.
func (b *BoundingBox) LowerLeftX() float32 { return b.lowerLeftX }

// SetLowerLeftX sets the lower left x value.
func (b *BoundingBox) SetLowerLeftX(value float32) { b.lowerLeftX = value }

// LowerLeftY returns the lower left y value.
func (b *BoundingBox) LowerLeftY() float32 { return b.lowerLeftY }

// SetLowerLeftY sets the lower left y value.
func (b *BoundingBox) SetLowerLeftY(value float32) { b.lowerLeftY = value }

// UpperRightX returns the upper right x value.
func (b *BoundingBox) UpperRightX() float32 { return b.upperRightX }

// SetUpperRightX sets the upper right x value.
func (b *BoundingBox) SetUpperRightX(value float32) { b.upperRightX = value }

// UpperRightY returns the upper right y value.
func (b *BoundingBox) UpperRightY() float32 { return b.upperRightY }

// SetUpperRightY sets the upper right y value.
func (b *BoundingBox) SetUpperRightY(value float32) { b.upperRightY = value }

// Width returns the width of this rectangle as calculated by
// upperRightX - lowerLeftX.
func (b *BoundingBox) Width() float32 { return b.UpperRightX() - b.LowerLeftX() }

// Height returns the height of this rectangle as calculated by
// upperRightY - lowerLeftY.
func (b *BoundingBox) Height() float32 { return b.UpperRightY() - b.LowerLeftY() }

// Contains reports whether the point is on the edge or inside the rectangle
// bounds.
func (b *BoundingBox) Contains(x, y float32) bool {
	return x >= b.lowerLeftX && x <= b.upperRightX &&
		y >= b.lowerLeftY && y <= b.upperRightY
}

// String returns the Java toString form.
func (b *BoundingBox) String() string {
	return "[" + javafmt.Float32(b.LowerLeftX()) + "," + javafmt.Float32(b.LowerLeftY()) + "," +
		javafmt.Float32(b.UpperRightX()) + "," + javafmt.Float32(b.UpperRightY()) + "]"
}
