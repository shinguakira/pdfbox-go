package logicalstructure

import (
	"fmt"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// PDParentTreeValue is a value of the /ParentTree number tree: either one
// structure element dictionary, or an array of them.
//
// Port of PDParentTreeValue.
type PDParentTreeValue struct {
	obj cos.Base
}

var _ common.COSObjectable = (*PDParentTreeValue)(nil)

// NewPDParentTreeValueOfArray wraps an array of structure elements.
func NewPDParentTreeValueOfArray(obj *cos.Array) *PDParentTreeValue {
	return &PDParentTreeValue{obj: obj}
}

// NewPDParentTreeValueOfDictionary wraps one structure element.
func NewPDParentTreeValueOfDictionary(obj *cos.Dictionary) *PDParentTreeValue {
	return &PDParentTreeValue{obj: obj}
}

// COSObject returns the array or dictionary below this value.
func (v *PDParentTreeValue) COSObject() cos.Base { return v.obj }

// String renders the wrapped object.
func (v *PDParentTreeValue) String() string { return fmt.Sprintf("%v", v.obj) }

// ParentTreeValueConverter builds the value of a /ParentTree entry.
//
// Java passes PDParentTreeValue.class to PDNumberTreeNode, which looks up a
// constructor taking exactly the runtime class of the entry, so an array or a
// dictionary works and anything else is an IOException. A stream is one of the
// anything else, even though COSStream extends COSDictionary in Java: the
// lookup is for the exact class. The port's *cos.Stream is not a *cos.Dictionary
// either, so the two agree here without being made to.
func ParentTreeValueConverter(base cos.Base) (common.COSObjectable, error) {
	switch value := base.(type) {
	case *cos.Array:
		return NewPDParentTreeValueOfArray(value), nil
	case *cos.Dictionary:
		return NewPDParentTreeValueOfDictionary(value), nil
	default:
		// Java's message carries the reflection failure; the port names the
		// type it was given instead.
		return nil, fmt.Errorf(
			"Error while trying to create value in number tree: no PDParentTreeValue for %T", base)
	}
}
