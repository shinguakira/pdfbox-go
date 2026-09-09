package multipdf

// JAVA-BUGS 84: `PDFMergerUtility.appendDocument`'s /PageMode block asks
// whether `destCatalog.getPageMode()` is null, and `getPageMode` answers
// USE_NONE for a document with no /PageMode and never null -- so the block is
// dead and a merge into an empty destination loses the source's page mode.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
)

// catalogWithPageMode is a catalog with the given /PageMode, or with none at
// all where the mode is empty.
func catalogWithPageMode(t *testing.T, mode pdmodel.PageMode) *pdmodel.PDDocumentCatalog {
	t.Helper()
	document := pdmodel.NewPDDocument()
	t.Cleanup(func() { document.Close() })
	catalog := document.DocumentCatalog()
	if mode != "" {
		catalog.SetPageMode(mode)
	}
	return catalog
}

// TestMergePageModeTakesTheSourcesWhereThereIsNone is the defect.
//
// The expected behaviour is what the dead block was written to do: a
// destination with no /PageMode of its own takes the source's. `pdfbox merge`
// starts with an empty document, so this is the case the command line tool
// takes every time, and a source that asks to open with its bookmarks showing
// is merged into a document that asks for nothing.
func TestMergePageModeTakesTheSourcesWhereThereIsNone(t *testing.T) {
	destCatalog := catalogWithPageMode(t, "")
	srcCatalog := catalogWithPageMode(t, pdmodel.PageModeUseOutlines)

	mergePageMode(destCatalog, srcCatalog)

	if got := destCatalog.PageMode(); got != pdmodel.PageModeUseOutlines {
		t.Errorf("PageMode() = %q, want the source's %q",
			got, pdmodel.PageModeUseOutlines)
	}
}

// TestMergePageModeKeepsTheDestinationsOwn is the other arm: a destination
// that has said what it wants keeps it.
func TestMergePageModeKeepsTheDestinationsOwn(t *testing.T) {
	destCatalog := catalogWithPageMode(t, pdmodel.PageModeUseThumbs)
	srcCatalog := catalogWithPageMode(t, pdmodel.PageModeUseOutlines)

	mergePageMode(destCatalog, srcCatalog)

	if got := destCatalog.PageMode(); got != pdmodel.PageModeUseThumbs {
		t.Errorf("PageMode() = %q, want the destination's own %q",
			got, pdmodel.PageModeUseThumbs)
	}
}

// TestMergePageModeWritesNothingForASourceWithNone keeps the merge from
// writing a /PageMode that says nothing: neither document has one, so neither
// does the result.
func TestMergePageModeWritesNothingForASourceWithNone(t *testing.T) {
	destCatalog := catalogWithPageMode(t, "")
	srcCatalog := catalogWithPageMode(t, "")

	mergePageMode(destCatalog, srcCatalog)

	if destCatalog.COSObject().(*cos.Dictionary).ContainsKey(cos.PageMode) {
		t.Error("the merge wrote a /PageMode neither document had")
	}
}
