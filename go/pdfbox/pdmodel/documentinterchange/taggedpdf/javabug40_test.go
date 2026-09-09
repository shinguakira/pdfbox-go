package taggedpdf_test

// JAVA-BUGS 40: `PDStandardAttributeObject.setArrayOfString` writes COSString
// entries and `getArrayOfString` reads each entry as a COSName, so a round
// trip through the pair throws.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
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

// TestHeadersLeaveOutAnEntryThatIsNotAString is the malformed array.
//
// The fix reads strings where Java read names; an entry that is neither is
// left out. Kept in as the empty string it would be a header identifier no
// cell carries, and indistinguishable from an entry that really is empty, so
// the list is shorter than the array instead.
func TestHeadersLeaveOutAnEntryThatIsNotAString(t *testing.T) {
	o := taggedpdf.NewPDTableAttributeObject()
	o.SetHeaders([]string{"h1", "h2"})

	// put a number between the two, which no writer produces and a damaged
	// file can hold
	headers := o.Dictionary().GetCOSArray(cos.GetPDFName("Headers"))
	if headers == nil {
		t.Fatal("/Headers is not an array")
	}
	headers.AddAt(1, cos.GetInteger(3))

	got := o.Headers()
	want := []string{"h1", "h2"}
	if len(got) != len(want) {
		t.Fatalf("Headers() = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("header %d is %q, want %q", i, got[i], want[i])
		}
	}
}
