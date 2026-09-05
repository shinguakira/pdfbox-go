package common

import (
	"reflect"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// NewCOSArrayOfObjectables returns an array holding the COS object of each of
// the given objects, and a nil entry for each object that is nil.
//
// Port of the COSArray(List<? extends COSObjectable>) constructor. Go cannot
// write that constructor in the cos package, because COSObjectable lives here.
func NewCOSArrayOfObjectables[E COSObjectable](objectables []E) *cos.Array {
	objects := make([]cos.Base, 0, len(objectables))
	for _, objectable := range objectables {
		if isNilObjectable(objectable) {
			objects = append(objects, nil)
			continue
		}
		objects = append(objects, objectable.COSObject())
	}
	return cos.NewArrayOf(objects)
}

// COSObjectOrNil returns the COS object of the given object, and nil where it
// is nil.
//
// Port of the null check the setItem(COSName, COSObjectable) of Java does
// before it calls setItem(COSName, COSBase).
func COSObjectOrNil[E COSObjectable](objectable E) cos.Base {
	if isNilObjectable(objectable) {
		return nil
	}
	return objectable.COSObject()
}

// isNilObjectable reports whether the given object is the Java null a
// COSObjectable parameter can be. A Go nil pointer in an interface is not the
// nil interface, so the check reaches through it.
func isNilObjectable(objectable any) bool {
	if objectable == nil {
		return true
	}
	value := reflect.ValueOf(objectable)
	switch value.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan:
		return value.IsNil()
	}
	return false
}
