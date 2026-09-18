package common_test

// JAVA-BUGS.md 90: what `PDStream.Filters` answers for a filter name written as
// an indirect reference.
//
// Java's `getFilters` hands the array's entries out as they were parsed, cast to
// `List<COSName>` without checking, so a `COSObject` reaches the caller as a
// name and the caller's first `getName()` raises `ClassCastException`. A Go
// slice of `*cos.Name` cannot hold a `*cos.Object`, and the port answers the
// name the reference points at.
//
// The expected value here comes from the specification — `/Filter` may be an
// indirect reference — not from the Java.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

func TestFiltersResolveAnIndirectName(t *testing.T) {
	stream := cos.NewStream(nil)
	filters := cos.NewArray()
	filters.Add(cos.NewObject(cos.DCTDecode))
	stream.SetItem(cos.Filter, filters)

	names := common.NewPDStream(stream).Filters()
	if len(names) != 1 {
		t.Fatalf("Filters() answered %d names, want one", len(names))
	}
	if names[0] == nil {
		t.Fatal("Filters() answered no name for an indirect /DCTDecode")
	}
	if got := names[0].Name(); got != "DCTDecode" {
		t.Errorf("Filters() answered %q, want DCTDecode", got)
	}

	// Decoding still refuses it, as PDFBox's getFilterList does: it reads the
	// entry without resolving and calls a reference a forbidden type.
	if _, err := stream.CreateReader(); err == nil {
		t.Error("a stream whose /Filter array holds a reference should not decode")
	}
}
