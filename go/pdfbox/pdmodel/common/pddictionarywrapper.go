package common

import "github.com/shinguakira/pdfbox-go/go/pdfbox/cos"

// PDDictionaryWrapper is a wrapper for a COS dictionary.
//
// Port of org.apache.pdfbox.pdmodel.common.PDDictionaryWrapper. Java's
// getCOSObject narrows its return type to COSDictionary as the class hierarchy
// goes down; Go has no covariant returns, so COSObject keeps the interface type
// and Dictionary hands back the concrete one.
type PDDictionaryWrapper struct {
	dictionary *cos.Dictionary
}

var _ COSObjectable = (*PDDictionaryWrapper)(nil)

// NewPDDictionaryWrapper returns a wrapper around a new empty dictionary.
func NewPDDictionaryWrapper() *PDDictionaryWrapper {
	return &PDDictionaryWrapper{dictionary: cos.NewDictionary()}
}

// NewPDDictionaryWrapperOf returns a wrapper around the given dictionary.
func NewPDDictionaryWrapperOf(dictionary *cos.Dictionary) *PDDictionaryWrapper {
	return &PDDictionaryWrapper{dictionary: dictionary}
}

// COSObject returns the dictionary behind this wrapper.
func (w *PDDictionaryWrapper) COSObject() cos.Base { return w.dictionary }

// Dictionary returns the dictionary behind this wrapper.
func (w *PDDictionaryWrapper) Dictionary() *cos.Dictionary { return w.dictionary }

// Equals reports whether both wrappers stand for the same dictionary.
//
// COSDictionary does not define equality, so Java compares the two by identity
// and this compares the pointers.
func (w *PDDictionaryWrapper) Equals(other *PDDictionaryWrapper) bool {
	if w == other {
		return true
	}
	if other == nil {
		return false
	}
	return w.dictionary == other.dictionary
}
