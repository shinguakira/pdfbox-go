package taggedpdf_test

// JAVA-BUGS 41: `PDFourColours`'s array constructor pads a short array from
// `size() - 1`, one index early, so it always adds one entry too many.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/documentinterchange/taggedpdf"
)

// TestFourColoursPadsToFour is the defect.
//
// The expected size is four, which the loop's own comment says -- "ensure that
// array has 4 items" -- and which PDF 32000-1:2008 table 344 requires of
// /BorderColor when it is not a single colour. Java pads from `size() - 1`, so
// every short array comes out five long, and the constructor mutates the array
// it was handed, so the extra entry is written back to the file.
func TestFourColoursPadsToFour(t *testing.T) {
	for _, size := range []int{0, 1, 2, 3, 4} {
		array := cos.NewArray()
		for i := 0; i < size; i++ {
			array.Add(cos.NullObject)
		}
		taggedpdf.NewPDFourColoursOfArray(array)
		want := 4
		if size > 4 {
			want = size
		}
		if got := array.Size(); got != want {
			t.Errorf("an array of %d entries came out %d long, want %d", size, got, want)
		}
	}
}
