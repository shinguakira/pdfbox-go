package pdfparser_test

// Defects the corpus found. See migration/TESTDATA.md for what the corpus is
// and how it is scored; each test here names the file that reached the fault,
// and each one fails without its fix.

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
)

// swappedXrefPDF writes a document whose cross-reference entries for two
// objects point at each other's offsets, which is the shape qpdf's issue-202.pdf
// arrives in after its trailer is damaged.
//
// Object 2 is the page tree, so the swap is on the path from /Root to the pages
// and a reader that does not repair it cannot find them. Object 4 is otherwise
// unused and exists only to be the object 2's entry points at.
func swappedXrefPDF() []byte {
	objects := []string{
		"1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n",
		"2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n",
		"3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] >>\nendobj\n",
		"4 0 obj\n<< /Unused true >>\nendobj\n",
	}

	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n")

	offsets := make([]int, len(objects))
	for i, object := range objects {
		offsets[i] = out.Len()
		out.WriteString(object)
	}

	// The swap: the entry for object 2 carries object 4's offset and the entry
	// for object 4 carries object 2's. Everything else is right.
	entries := []int{offsets[0], offsets[3], offsets[2], offsets[1]}

	startxref := out.Len()
	out.WriteString("xref\n0 5\n0000000000 65535 f \n")
	for _, offset := range entries {
		fmt.Fprintf(&out, "%010d 00000 n \n", offset)
	}
	out.WriteString("trailer\n<< /Size 5 /Root 1 0 R >>\n")
	fmt.Fprintf(&out, "startxref\n%d\n%%%%EOF\n", startxref)
	return out.Bytes()
}

// TestSwappedXrefEntriesAreRepaired pins the third defect the corpus found:
// qpdf/issue-202.pdf, which PDFBox opens at ten pages and the port refused with
// "Page tree root must be a dictionary".
//
// Both sides notice the problem and both log it -- "found wrong object number
// expected 212 found 213", and the same pair the other way round. Java's
// XrefParser.validateXrefOffsets then rewrites the two entries so each points at
// the object that is really there, and it can do that because
// XrefTrailerResolver.getXrefTable() hands out the resolver's own map and the
// method edits it in place.
//
// The port's XrefTable() answers a copy, which is the right shape for Go and
// was the whole of the bug: validateXrefOffsets corrected the copy, checkXrefOffsets
// returned, and the corrections went out of scope. The xref still pointed object
// 2 at object 4, dereferencing /Pages failed, and CheckPages threw.
//
// What the document holds is not the point -- one page, no content. The
// assertion is that it opens at all and that the page tree is reachable.
func TestSwappedXrefEntriesAreRepaired(t *testing.T) {
	document, err := pdfbox.LoadPDFBytes(swappedXrefPDF())
	if err != nil {
		t.Fatalf("LoadPDFBytes: %v -- the swapped entries were not repaired", err)
	}
	defer document.Close()

	if got := document.NumberOfPages(); got != 1 {
		t.Errorf("NumberOfPages = %d, want 1", got)
	}
}
