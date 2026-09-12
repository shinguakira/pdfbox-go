package pdfwriter_test

// PDFBOX-6236, Apache 21661b79f, in the sync of 2026-09-07.
//
// COSWriter numbers the objects an incremental update adds from
// getHighestXRefObjectNumber(), which is the highest number the parser saw in
// the cross-reference data it could read. The trailer's /Size is the file's own
// statement of the same thing, and the two disagree whenever the parser did not
// index everything: an xref section it could not follow, an object the file
// declares and does not list. Where /Size is the larger, numbering from the
// other one hands a new object a number the file has already used, and the
// update silently replaces an object instead of adding one.
//
//	long trailerSize = trailer.getLong(COSName.SIZE);
//	number = Math.max(trailerSize - 1, number);
//
// getLong answers -1 for an absent or non-numeric /Size, so -2 is what the max
// sees there and the old behaviour stands. The change only ever raises.

import (
	"bytes"
	"os"
	"regexp"
	"strconv"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// objectHeader matches the "N 0 obj" line each written object opens with.
var objectHeader = regexp.MustCompile(`(?m)^(\d+) 0 obj`)

// incrementWithSize saves an incremental update over the poems fixture with the
// trailer's /Size set to size, having added one new indirect object, and
// answers the numbers the update wrote.
//
// It also answers the highest number the parser found, which is what the writer
// used to number from on its own.
func incrementWithSize(t *testing.T, size int64) (written []int64, highest int64) {
	t.Helper()
	path := inputFixture + "PDFBOX-3110-poems-beads.pdf"
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the original: %v", err)
	}
	doc, err := pdfbox.LoadPDF(path)
	if err != nil {
		t.Fatalf("LoadPDF: %v", err)
	}
	defer doc.Close()

	highest = doc.Document().HighestXRefObjectNumber()
	trailer := doc.Document().Trailer()
	// The file itself is consistent -- 46 objects and /Size 47 -- so the
	// disagreement this is about has to be written in. A file that carries it
	// is a file whose xref the parser could only partly read, and there is no
	// way to build one of those that stays a fixture of this repository.
	if size < 0 {
		trailer.RemoveItem(cos.Size)
	} else {
		trailer.SetLong(cos.Size, size)
	}

	// A brand new indirect object, reachable from the catalog: this is the one
	// that gets the next free number.
	added := cos.NewDictionary()
	added.SetItem(cos.GetPDFName("Marker"), cos.NewStringObj("new"))
	reference := cos.NewObject(added)
	reference.SetNeedToBeUpdated(true)
	catalog := doc.DocumentCatalog().Dictionary()
	catalog.SetItem(cos.GetPDFName("GoPortNewObject"), reference)
	catalog.SetNeedToBeUpdated(true)
	trailer.SetNeedToBeUpdated(true)

	var out bytes.Buffer
	if err := doc.SaveIncremental(&out); err != nil {
		t.Fatalf("SaveIncremental: %v", err)
	}
	if !bytes.HasPrefix(out.Bytes(), original) {
		t.Fatalf("the incremental save did not append to the original")
	}
	for _, match := range objectHeader.FindAllStringSubmatch(
		string(out.Bytes()[len(original):]), -1) {
		number, err := strconv.ParseInt(match[1], 10, 64)
		if err != nil {
			t.Fatalf("object header %q: %v", match[1], err)
		}
		written = append(written, number)
	}
	return written, highest
}

// TestIncrementalUpdateNumbersFromTheTrailerSize is the site. A /Size well
// above what the parser indexed must be what the new object is numbered from,
// because every number below it may already be in the file.
func TestIncrementalUpdateNumbersFromTheTrailerSize(t *testing.T) {
	const size = 96
	written, highest := incrementWithSize(t, size)
	if highest >= size-1 {
		t.Fatalf("the fixture's highest object number is %d, which is not below "+
			"/Size-1 = %d, so this case tests nothing", highest, size-1)
	}
	// The catalog is rewritten under the number it already had; the new object
	// is the one that had none.
	var newObject int64 = -1
	for _, number := range written {
		if number > highest {
			newObject = number
		}
	}
	if newObject < 0 {
		t.Fatalf("the update wrote %v, none of it a new object above %d",
			written, highest)
	}
	if newObject != size {
		t.Errorf("the new object was numbered %d, want %d: numbering from "+
			"/Size-1 = %d means the first free number is /Size itself",
			newObject, size, size-1)
	}
	for _, number := range written {
		if number > highest && number < size {
			t.Errorf("the update wrote object %d, which is below /Size %d and so "+
				"may already be in the file", number, size)
		}
	}
}

// TestIncrementalUpdateKeepsTheXRefNumberWhereItIsHigher is the guard: the max
// only ever raises, so a file whose /Size is the consistent one numbers exactly
// where it did before.
func TestIncrementalUpdateKeepsTheXRefNumberWhereItIsHigher(t *testing.T) {
	written, highest := incrementWithSize(t, -1) // no /Size at all
	for _, number := range written {
		if number > highest+1 {
			t.Errorf("the update wrote object %d with no /Size to read; the "+
				"highest the parser found is %d, so %d is the first free number",
				number, highest, highest+1)
		}
	}
}
