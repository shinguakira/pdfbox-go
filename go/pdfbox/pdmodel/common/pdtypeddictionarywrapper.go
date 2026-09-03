package common

import "github.com/shinguakira/pdfbox-go/go/pdfbox/cos"

// PDTypedDictionaryWrapper is a wrapper for a COS dictionary including Type
// information.
//
// Port of org.apache.pdfbox.pdmodel.common.PDTypedDictionaryWrapper.
type PDTypedDictionaryWrapper struct {
	*PDDictionaryWrapper
}

// NewPDTypedDictionaryWrapper returns a wrapper around a new dictionary of the
// given type.
func NewPDTypedDictionaryWrapper(typ string) *PDTypedDictionaryWrapper {
	w := &PDTypedDictionaryWrapper{PDDictionaryWrapper: NewPDDictionaryWrapper()}
	w.Dictionary().SetName(cos.Type, typ)
	return w
}

// NewPDTypedDictionaryWrapperOf returns a wrapper around the given dictionary.
func NewPDTypedDictionaryWrapperOf(dictionary *cos.Dictionary) *PDTypedDictionaryWrapper {
	return &PDTypedDictionaryWrapper{PDDictionaryWrapper: NewPDDictionaryWrapperOf(dictionary)}
}

// Type returns the type.
//
// There is no SetType method because changing the Type would most probably also
// change the type of PD object.
func (w *PDTypedDictionaryWrapper) Type() string {
	return w.Dictionary().GetNameAsString(cos.Type, "")
}
