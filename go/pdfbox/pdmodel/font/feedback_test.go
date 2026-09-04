package font

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// Tests for the slice 4 pull request feedback. Each fails without its fix.

// TestMalformedW2Fails pins what PDCIDFont.readVerticalDisplacements does with
// a /W2 array that does not hold what it should.
//
// Java casts every entry with (COSNumber), which throws ClassCastException on
// anything else and takes the font construction with it. The port had softened
// that to a warning and a partial read, so a malformed font came back with some
// of its vertical metrics filled in and the rest defaulted.
func TestMalformedW2Fails(t *testing.T) {
	dict := cos.NewDictionary()
	dict.SetItem(cos.Type, cos.Font)
	dict.SetItem(cos.Subtype, cos.CIDFontType2)
	dict.SetName(cos.BaseFont, "Test")

	// a first range that reads cleanly, then a name where a number belongs
	w2 := cos.NewArray()
	w2.Add(cos.GetInteger(1))
	w2.Add(cos.GetInteger(2))
	w2.Add(cos.GetInteger(-500))
	w2.Add(cos.GetInteger(0))
	w2.Add(cos.GetInteger(880))
	w2.Add(cos.GetInteger(3))
	w2.Add(cos.GetPDFName("Bogus"))
	dict.SetItem(cos.W2, w2)

	defer func() {
		if recovered := recover(); recovered == nil {
			t.Error("reading a /W2 array with a name where a number belongs should fail, " +
				"the way Java's (COSNumber) cast does")
		}
	}()
	//nolint:errcheck // the panic is the assertion
	_, _ = NewPDCIDFontType2(dict, nil)
}
