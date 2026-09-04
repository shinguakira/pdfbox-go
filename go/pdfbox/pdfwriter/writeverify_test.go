package pdfwriter_test

// These are not ported Java tests. They are the checks slice 7's phase D asks
// for, kept because they are the only ones that can fail when the writer is
// subtly wrong:
//
//   - D7 says round-tripping proves nothing on its own, because writing and
//     reading with the same broken port passes. TestEmittedBytesMatchPDFBox
//     compares the bytes this writer emits against a PDF that PDFBox itself
//     wrote, which is in the Java test resources.
//   - D8 says an incremental save must leave the original bytes untouched and
//     append the update. TestIncrementalSaveAppends checks that byte for byte.
//
// The round trip is here too, but only as the cheap first signal; the two
// checks above are what make it mean something.

import (
	"bytes"
	"os"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdfwriter/compress"
)

// pdfBoxWritten is a file PDFBox wrote, from the Java test resources: the
// result of merging a document into a fresh one and saving it uncompressed.
const pdfBoxWritten = "../../../pdfbox/src/test/resources/org/apache/pdfbox/multipdf/" +
	"PDFBoxLegacyMerge-SameMerged.pdf"

// TestEmittedBytesMatchPDFBox compares the byte-level shape of what this writer
// emits against what PDFBox emitted.
//
// The two files cannot be identical: PDFBox wrote its one from a document built
// in memory, whose highest object number was 0, and this one is written from
// that file, whose highest object number is 227 --- and COSWriter starts
// numbering at getHighestXRefObjectNumber(), so the objects come out as 228
// upwards. What must match is every byte sequence the writer chooses: the two
// header lines, the object header and endobj, the dictionary layout, the
// endstream marker, the xref entry format, and the trailer down to %%EOF.
func TestEmittedBytesMatchPDFBox(t *testing.T) {
	ref, err := os.ReadFile(pdfBoxWritten)
	if err != nil {
		t.Fatalf("reading the reference: %v", err)
	}
	src, err := pdfbox.LoadPDF(pdfBoxWritten)
	if err != nil {
		t.Fatalf("LoadPDF: %v", err)
	}
	defer src.Close()
	var out bytes.Buffer
	if err := src.SaveOfParameters(&out, compress.NoCompression); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got := out.Bytes()

	// the two header lines
	header := []byte("%PDF-1.4\n%\xf6\xe4\xfc\xdf\n")
	if !bytes.HasPrefix(ref, header) {
		t.Fatalf("the reference does not start with the header this test assumes: %q", ref[:16])
	}
	if !bytes.HasPrefix(got, header) {
		t.Errorf("header = %q, want %q", got[:min(len(got), 16)], header)
	}

	// the first object: its header, the dictionary layout, and endobj
	assertShape(t, "object header", ref, got, []byte(" 0 obj\n<<\n/Type /Catalog\n/Version /1.6\n/Pages "))
	assertShape(t, "endobj", ref, got, []byte("\n>>\nendobj\n"))
	assertShape(t, "endstream", ref, got, []byte("\r\nendstream\nendobj\n"))

	// the cross-reference entries: ten digits, five digits, a marker, CRLF
	assertShape(t, "free xref entry", ref, got, []byte(" 65535 f\r\n"))
	assertShape(t, "used xref entry", ref, got, []byte(" 00000 n\r\n"))
	assertShape(t, "xref header", ref, got, []byte("\nxref\n0 "))

	// the trailer and the end of the file
	assertShape(t, "trailer", ref, got, []byte("\n>>\nstartxref\n"))
	if !bytes.HasSuffix(ref, []byte("\n%%EOF\n")) {
		t.Fatalf("the reference does not end the way this test assumes")
	}
	if !bytes.HasSuffix(got, []byte("\n%%EOF\n")) {
		t.Errorf("the file does not end with %q", "\n%%EOF\n")
	}
}

// assertShape checks that a byte sequence PDFBox emits is one this writer emits
// too, and fails the test if the reference does not contain it --- a reference
// that has moved on is not evidence of anything.
func assertShape(t *testing.T, what string, ref, got, shape []byte) {
	t.Helper()
	if !bytes.Contains(ref, shape) {
		t.Fatalf("%s: the PDFBox-written reference does not contain %q, so this check is stale",
			what, shape)
	}
	if !bytes.Contains(got, shape) {
		t.Errorf("%s: the emitted bytes do not contain %q, which PDFBox emits", what, shape)
	}
}

// TestIncrementalSaveAppends checks that an incremental save leaves every byte
// of the original where it was and appends the update after it.
func TestIncrementalSaveAppends(t *testing.T) {
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

	// change something reachable from the catalog, and mark the path to it
	marker := cos.GetPDFName("GoPortIncrementalMarker")
	catalog := doc.DocumentCatalog().Dictionary()
	catalog.SetItem(marker, cos.NewStringObj("touched"))
	catalog.SetNeedToBeUpdated(true)
	doc.Document().Trailer().SetNeedToBeUpdated(true)

	var out bytes.Buffer
	if err := doc.SaveIncremental(&out); err != nil {
		t.Fatalf("SaveIncremental: %v", err)
	}
	got := out.Bytes()

	if len(got) <= len(original) {
		t.Fatalf("the incremental save wrote %d bytes, no more than the original %d",
			len(got), len(original))
	}
	if !bytes.Equal(got[:len(original)], original) {
		for i := range original {
			if got[i] != original[i] {
				t.Fatalf("the original bytes were rewritten at offset %d: %#x, was %#x",
					i, got[i], original[i])
			}
		}
	}

	back, err := pdfbox.LoadPDFBytes(got)
	if err != nil {
		t.Fatalf("reloading the updated file: %v", err)
	}
	defer back.Close()
	if pages := back.NumberOfPages(); pages != 2 {
		t.Errorf("NumberOfPages() = %d after the update, want 2", pages)
	}
	if value := back.DocumentCatalog().Dictionary().GetDictionaryObject(marker); value == nil {
		t.Error("the appended update is not visible after reloading")
	}
}

// TestSaveRoundTrip checks that a saved document reads back, compressed and
// not. On its own this proves nothing --- see the file comment --- but it is
// what catches a writer that emits nothing at all.
func TestSaveRoundTrip(t *testing.T) {
	for _, name := range []string{"cweb.pdf", "PDFBOX-3110-poems-beads.pdf", "hello3.pdf"} {
		src, err := pdfbox.LoadPDF(inputFixture + name)
		if err != nil {
			t.Errorf("%s: LoadPDF: %v", name, err)
			continue
		}
		pages := src.NumberOfPages()
		for _, mode := range []struct {
			label string
			p     *compress.Parameters
		}{
			{"compressed", compress.DefaultCompression},
			{"uncompressed", compress.NoCompression},
		} {
			var out bytes.Buffer
			if err := src.SaveOfParameters(&out, mode.p); err != nil {
				t.Errorf("%s %s: Save: %v", name, mode.label, err)
				continue
			}
			back, err := pdfbox.LoadPDFBytes(out.Bytes())
			if err != nil {
				t.Errorf("%s %s: reloading %d bytes: %v", name, mode.label, out.Len(), err)
				continue
			}
			if got := back.NumberOfPages(); got != pages {
				t.Errorf("%s %s: NumberOfPages() = %d after the round trip, want %d",
					name, mode.label, got, pages)
			}
			back.Close()
		}
		src.Close()
	}
}
