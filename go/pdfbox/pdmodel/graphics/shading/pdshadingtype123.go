package shading

// The three shading types that are not mesh based: function based, axial and
// radial.
//
// Port of PDShadingType1, PDShadingType2 and PDShadingType3. Java gives them a
// file each; each is a handful of cached dictionary accessors, and type 3
// extends type 2, so the port keeps them together.

import (
	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// PDShadingType1 is function based shading.
//
// Port of PDShadingType1.
type PDShadingType1 struct {
	PDShading
	domain *cos.Array
}

var _ Shading = (*PDShadingType1)(nil)

// NewPDShadingType1 creates a function based shading over the given dictionary.
func NewPDShadingType1(shadingDictionary *cos.Dictionary) *PDShadingType1 {
	s := &PDShadingType1{}
	s.InitShadingOf(shadingDictionary)
	return s
}

// ShadingType returns ShadingType1.
func (s *PDShadingType1) ShadingType() int { return ShadingType1 }

// Matrix returns the /Matrix entry, which maps the shading's own coordinate
// space into the space it is used in.
func (s *PDShadingType1) Matrix() *util.Matrix {
	return util.CreateMatrix(s.Dictionary().GetDictionaryObject(cos.Matrix))
}

// SetMatrix sets the /Matrix entry from an affine transform.
func (s *PDShadingType1) SetMatrix(transform *geom.AffineTransform) {
	matrix := cos.NewArray()
	values := make([]float64, 6)
	transform.GetMatrix(values)
	for _, v := range values {
		matrix.Add(cos.NewFloat(float32(v)))
	}
	s.Dictionary().SetItem(cos.Matrix, matrix)
}

// Domain returns the /Domain entry, or nil where there is none.
func (s *PDShadingType1) Domain() *cos.Array {
	if s.domain == nil {
		s.domain = s.Dictionary().GetCOSArray(cos.Domain)
	}
	return s.domain
}

// SetDomain sets the /Domain entry.
func (s *PDShadingType1) SetDomain(newDomain *cos.Array) {
	s.domain = newDomain
	s.Dictionary().SetItem(cos.Domain, newDomain)
}

// PDShadingType2 is axial shading.
//
// Port of PDShadingType2.
type PDShadingType2 struct {
	PDShading
	coords *cos.Array
	domain *cos.Array
	extend *cos.Array
}

var _ Shading = (*PDShadingType2)(nil)

// NewPDShadingType2 creates an axial shading over the given dictionary.
func NewPDShadingType2(shadingDictionary *cos.Dictionary) *PDShadingType2 {
	s := &PDShadingType2{}
	s.InitShadingOf(shadingDictionary)
	return s
}

// ShadingType returns ShadingType2.
func (s *PDShadingType2) ShadingType() int { return ShadingType2 }

// Extend returns the /Extend entry, which says whether the shading runs on
// past each end, or nil where there is none.
func (s *PDShadingType2) Extend() *cos.Array {
	if s.extend == nil {
		s.extend = s.Dictionary().GetCOSArray(cos.Extend)
	}
	return s.extend
}

// SetExtend sets the /Extend entry.
func (s *PDShadingType2) SetExtend(newExtend *cos.Array) {
	s.extend = newExtend
	s.Dictionary().SetItem(cos.Extend, newExtend)
}

// Domain returns the /Domain entry, or nil where there is none.
func (s *PDShadingType2) Domain() *cos.Array {
	if s.domain == nil {
		s.domain = s.Dictionary().GetCOSArray(cos.Domain)
	}
	return s.domain
}

// SetDomain sets the /Domain entry.
func (s *PDShadingType2) SetDomain(newDomain *cos.Array) {
	s.domain = newDomain
	s.Dictionary().SetItem(cos.Domain, newDomain)
}

// Coords returns the /Coords entry, which is the axis for a type 2 shading and
// the two circles for a type 3, or nil where there is none.
func (s *PDShadingType2) Coords() *cos.Array {
	if s.coords == nil {
		s.coords = s.Dictionary().GetCOSArray(cos.Coords)
	}
	return s.coords
}

// SetCoords sets the /Coords entry.
func (s *PDShadingType2) SetCoords(newCoords *cos.Array) {
	s.coords = newCoords
	s.Dictionary().SetItem(cos.Coords, newCoords)
}

// PDShadingType3 is radial shading.
//
// Port of PDShadingType3, which extends PDShadingType2: the two read the same
// entries and differ in what /Coords means and in how they are drawn.
type PDShadingType3 struct {
	PDShadingType2
}

var _ Shading = (*PDShadingType3)(nil)

// NewPDShadingType3 creates a radial shading over the given dictionary.
func NewPDShadingType3(shadingDictionary *cos.Dictionary) *PDShadingType3 {
	s := &PDShadingType3{}
	s.InitShadingOf(shadingDictionary)
	return s
}

// ShadingType returns ShadingType3.
func (s *PDShadingType3) ShadingType() int { return ShadingType3 }
