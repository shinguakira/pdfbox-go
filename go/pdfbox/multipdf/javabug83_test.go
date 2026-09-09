package multipdf

// JAVA-BUGS 83: `PDFMergerUtility.mergeMarkInfo` reads `usesUserProperties`
// and writes `setSuspect` on the line that should write
// `setUserProperties`, so `/UserProperties` is never merged and `/Suspect` is
// decided twice, the second answer winning.
//
// In-package, because `mergeMarkInfo` is: reaching it through
// `AppendDocument` would need both documents to carry a structure tree with a
// non-empty parent tree, which is the condition the whole structure block sits
// under and is not what is being tested.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/documentinterchange/logicalstructure"
)

// markedCatalog is a catalog carrying a /MarkInfo with the two flags set as
// asked.
func markedCatalog(t *testing.T, suspect, userProperties bool) *pdmodel.PDDocumentCatalog {
	t.Helper()
	document := pdmodel.NewPDDocument()
	t.Cleanup(func() { document.Close() })
	markInfo := logicalstructure.NewPDMarkInfo()
	markInfo.SetMarked(true)
	markInfo.SetSuspect(suspect)
	markInfo.SetUserProperties(userProperties)
	document.DocumentCatalog().SetMarkInfo(markInfo)
	return document.DocumentCatalog()
}

// TestMergeMarkInfoKeepsBothFlags is the defect.
//
// The expected values are the two `or`s the block is written to compute: the
// merged document is suspect if either was, and carries user properties if
// either did. Both are ISO 32000-1 table 321 and both are read by assistive
// technology. Java writes the second answer to `/Suspect` as well, so a
// document whose structure is suspect but which has no user properties comes
// out not suspect, and `/UserProperties` is never written at all.
func TestMergeMarkInfoKeepsBothFlags(t *testing.T) {
	for _, c := range []struct {
		name                        string
		srcSuspect, srcProperties   bool
		destSuspect, destProperties bool
		wantSuspect, wantProperties bool
	}{
		// the case Java gets wrong twice: suspect, and no user properties
		{"suspectOnly", true, false, false, false, true, false},
		{"propertiesOnly", false, true, false, false, false, true},
		{"both", true, true, false, false, true, true},
		{"neither", false, false, false, false, false, false},
		// the destination's own flags survive as well
		{"destinationSuspect", false, false, true, false, true, false},
		{"destinationProperties", false, false, false, true, false, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			destCatalog := markedCatalog(t, c.destSuspect, c.destProperties)
			srcCatalog := markedCatalog(t, c.srcSuspect, c.srcProperties)

			mergeMarkInfo(destCatalog, srcCatalog)

			merged := destCatalog.MarkInfo()
			if merged == nil {
				t.Fatal("the merged catalog has no /MarkInfo")
			}
			if got := merged.IsSuspect(); got != c.wantSuspect {
				t.Errorf("/Suspects = %v, want %v", got, c.wantSuspect)
			}
			if got := merged.UsesUserProperties(); got != c.wantProperties {
				t.Errorf("/UserProperties = %v, want %v", got, c.wantProperties)
			}
			if !merged.IsMarked() {
				t.Error("/Marked = false, want the true the merge sets")
			}
		})
	}
}
