// Package filespecification names the file a PDF refers to or embeds.
//
// Port of org.apache.pdfbox.pdmodel.common.filespecification.
package filespecification

import (
	"fmt"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// PDFileSpecification is a file specification, in either of its two forms.
//
// Port of the abstract class
// org.apache.pdfbox.pdmodel.common.filespecification.PDFileSpecification; the
// port splits it into this interface for the contract and the two concrete
// types below, since the abstract class holds no state.
type PDFileSpecification interface {
	common.COSObjectable

	// File returns the file name, or "" where there is none.
	File() string

	// SetFile sets the file name.
	SetFile(file string)
}

// CreateFS returns the file specification the given object holds.
//
// Port of the static createFS(COSBase).
func CreateFS(base cos.Base) (PDFileSpecification, error) {
	switch value := base.(type) {
	case nil:
		// then simply return null
		return nil, nil
	case *cos.StringObj:
		return NewPDSimpleFileSpecificationOf(value), nil
	case *cos.Stream:
		// COSStream is a COSDictionary in Java, so it takes the same branch.
		return NewPDComplexFileSpecification(&value.Dictionary), nil
	case *cos.Dictionary:
		return NewPDComplexFileSpecification(value), nil
	}
	return nil, fmt.Errorf("Error: Unknown file specification %v", base)
}

// PDSimpleFileSpecification is a file specification that is just a string.
//
// Port of PDSimpleFileSpecification.
type PDSimpleFileSpecification struct {
	file *cos.StringObj
}

var _ PDFileSpecification = (*PDSimpleFileSpecification)(nil)

// NewPDSimpleFileSpecification creates a specification naming nothing.
func NewPDSimpleFileSpecification() *PDSimpleFileSpecification {
	return &PDSimpleFileSpecification{file: cos.NewStringObj("")}
}

// NewPDSimpleFileSpecificationOf creates a specification over the given string.
func NewPDSimpleFileSpecificationOf(fileName *cos.StringObj) *PDSimpleFileSpecification {
	return &PDSimpleFileSpecification{file: fileName}
}

// File returns the file name.
func (s *PDSimpleFileSpecification) File() string { return s.file.Value() }

// SetFile sets the file name.
func (s *PDSimpleFileSpecification) SetFile(fileName string) {
	s.file = cos.NewStringObj(fileName)
}

// COSObject returns the string.
func (s *PDSimpleFileSpecification) COSObject() cos.Base { return s.file }
