package fdf_test

// JAVA-BUGS 45: `FDFDictionary.getPages` reads each entry of `/Pages` with
// `COSArray.get`, which hands back an indirect reference as it stands, where
// every other list accessor of the class uses `getObject` and dereferences it.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/fdf"
)

// TestPagesResolvesAnIndirectPage is the defect.
//
// The expected behaviour is `getFields`', three methods above: read the entry
// with `getObject`, so a page written as an indirect object is the dictionary
// it refers to. Nothing in the FDF specification says the entries of `/Pages`
// must be direct, and PDFBox's own writer makes them indirect, so Java throws
// ClassCastException on files it wrote itself.
func TestPagesResolvesAnIndirectPage(t *testing.T) {
	page := cos.NewDictionary()
	page.SetString(cos.T, "page one")

	dictionary := cos.NewDictionary()
	dictionary.SetItem(cos.Pages, cos.NewArrayOf([]cos.Base{cos.NewObject(page)}))

	pages := fdf.NewFDFDictionaryOf(dictionary).Pages()
	if pages == nil {
		t.Fatal("Pages() answered nothing for a /Pages of one entry")
	}
	if got := pages.Size(); got != 1 {
		t.Fatalf("Pages() has %d entries, want 1", got)
	}
}

// TestPagesStillReadsADirectPage keeps the entry that needs no dereferencing,
// which is the only shape the Java could read.
func TestPagesStillReadsADirectPage(t *testing.T) {
	page := cos.NewDictionary()

	dictionary := cos.NewDictionary()
	dictionary.SetItem(cos.Pages, cos.NewArrayOf([]cos.Base{page}))

	pages := fdf.NewFDFDictionaryOf(dictionary).Pages()
	if pages == nil || pages.Size() != 1 {
		t.Fatalf("Pages() = %v, want one page", pages)
	}
}
