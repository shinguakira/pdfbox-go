// Package util holds the helpers PDFBox keeps outside its object model.
//
// Port of org.apache.pdfbox.util.
package util

import "github.com/shinguakira/pdfbox-go/go/internal/javafmt"

// Vector is a 2D vector.
//
// Port of org.apache.pdfbox.util.Vector. The Java class is final with final
// fields, so a Go value type carries it exactly.
type Vector struct {
	x, y float32
}

// NewVector returns the vector (x, y).
func NewVector(x, y float32) Vector { return Vector{x: x, y: y} }

// X returns the x magnitude.
func (v Vector) X() float32 { return v.x }

// Y returns the y magnitude.
func (v Vector) Y() float32 { return v.y }

// Scale returns a new vector scaled by both x and y.
func (v Vector) Scale(sxy float32) Vector {
	return Vector{x: v.x * sxy, y: v.y * sxy}
}

// String returns the Java toString form.
func (v Vector) String() string {
	return "(" + javafmt.Float32(v.x) + ", " + javafmt.Float32(v.y) + ")"
}
