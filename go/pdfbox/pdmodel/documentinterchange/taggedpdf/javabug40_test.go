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

// TestHeadersKeepAHoleForAnEntryThatIsNotAString is the malformed array.
//
// The fix reads strings where Java read names. What happens to an entry that is
// neither moved with PDFBOX-6261 (Apache 691e7d100, in the sync of
// 2026-11-21): the Java reads every entry with COSArray.getString, which
// answers null for one that is not a string, and leaves that null in the array
// it hands back, so the list is the array's length with holes in it. Go has no
// nil inside a []string, so the hole is the empty string.
//
// The port used to leave the entry out, which made the list shorter than the
// array. The Java is the specification, so the length follows it.
func TestHeadersKeepAHoleForAnEntryThatIsNotAString(t *testing.T) {
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
	want := []string{"h1", "", "h2"}
	if len(got) != len(want) {
		t.Fatalf("Headers() = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("header %d is %q, want %q", i, got[i], want[i])
		}
	}
}
