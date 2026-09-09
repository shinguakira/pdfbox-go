package pagenavigation

// JAVA-BUGS 35: `COSName.BEAD` is `getPDFName("BEAD")`, and PDF
// 32000-1:2008 table 30 gives the thread bead dictionary a `/Type` of `Bead`.
// PDF names are case-sensitive, so a bead PDFBox creates carries a `/Type` no
// conforming reader looks for.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// TestNewThreadBeadIsTypeBead is the defect.
//
// The expected value is the specification's: table 30 names the entry `Bead`.
// Nothing in either tree reads the entry back, so PDFBox round-trips its own
// output and only a conforming reader notices.
func TestNewThreadBeadIsTypeBead(t *testing.T) {
	bead := NewPDThreadBead()
	if got := bead.COSObject().(*cos.Dictionary).GetNameAsString(cos.Type, ""); got != "Bead" {
		t.Errorf("/Type is %q, want %q", got, "Bead")
	}
}
