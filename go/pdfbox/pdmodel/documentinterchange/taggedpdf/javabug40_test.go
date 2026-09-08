package taggedpdf_test

// JAVA-BUGS 40: `PDStandardAttributeObject.setArrayOfString` writes COSString
// entries and `getArrayOfString` reads each entry as a COSName, so a round
// trip through the pair throws.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/documentinterchange/taggedpdf"
)

// TestHeadersRoundTrip is the defect.
//
// The expected value is what was set: the two halves of one accessor pair have
// to agree, and PDF 32000-1:2008 table 337 gives /Headers as an array of byte
// strings, so the setter is the half that is right. Java's getter casts each
// entry to COSName and throws ClassCastException.
func TestHeadersRoundTrip(t *testing.T) {
	for _, o := range []interface {
		Headers() []string
		SetHeaders([]string)
	}{
		taggedpdf.NewPDTableAttributeObject(),
		taggedpdf.NewPDExportFormatAttributeObject(taggedpdf.OwnerHTML401),
	} {
		want := []string{"h1", "h2"}
		o.SetHeaders(want)
		got := o.Headers()
		if len(got) != len(want) {
			t.Errorf("%T gave %d headers, want %d", o, len(got), len(want))
			continue
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("%T header %d is %q, want %q", o, i, got[i], want[i])
			}
		}
	}
}

// TestHeadersInAStringOfTheirOwn covers toString, which the entry names: it
// calls the getter, so printing an object filled in through its own API threw
// as well.
func TestHeadersInAStringOfTheirOwn(t *testing.T) {
	o := taggedpdf.NewPDTableAttributeObject()
	o.SetHeaders([]string{"h1"})
	if got := o.String(); got == "" {
		t.Error("the attribute object printed as nothing")
	}
}
