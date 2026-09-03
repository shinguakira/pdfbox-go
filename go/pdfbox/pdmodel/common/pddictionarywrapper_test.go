package common

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// Written from PDDictionaryWrapper and PDTypedDictionaryWrapper; the Java suite
// has no test for either.

func TestPDDictionaryWrapper(t *testing.T) {
	w := NewPDDictionaryWrapper()
	if w.Dictionary() == nil {
		t.Fatal("a new wrapper has no dictionary")
	}
	if w.COSObject() != cos.Base(w.Dictionary()) {
		t.Error("COSObject and Dictionary disagree")
	}

	d := cos.NewDictionary()
	of := NewPDDictionaryWrapperOf(d)
	if of.Dictionary() != d {
		t.Error("the wrapper did not keep the dictionary it was given")
	}
}

// TestPDDictionaryWrapperEquals pins that equality is the identity of the
// dictionary behind the wrapper: COSDictionary does not define equality of its
// own, so two wrappers around equal-looking dictionaries are not equal.
func TestPDDictionaryWrapperEquals(t *testing.T) {
	d := cos.NewDictionary()
	if !NewPDDictionaryWrapperOf(d).Equals(NewPDDictionaryWrapperOf(d)) {
		t.Error("two wrappers around the same dictionary are unequal")
	}
	if NewPDDictionaryWrapper().Equals(NewPDDictionaryWrapper()) {
		t.Error("two wrappers around different dictionaries are equal")
	}
	if NewPDDictionaryWrapper().Equals(nil) {
		t.Error("Equals = true against nil")
	}
}

func TestPDTypedDictionaryWrapper(t *testing.T) {
	w := NewPDTypedDictionaryWrapper("Page")
	if got := w.Type(); got != "Page" {
		t.Errorf("Type = %q, want %q", got, "Page")
	}
	if got := w.Dictionary().GetCOSName(cos.Type); got != cos.Page {
		t.Errorf("/Type = %v, want /Page", got)
	}

	// A wrapper around a dictionary with no /Type reports the empty string,
	// which is what Java's null becomes.
	if got := NewPDTypedDictionaryWrapperOf(cos.NewDictionary()).Type(); got != "" {
		t.Errorf("Type of an untyped dictionary = %q, want the empty string", got)
	}
}
