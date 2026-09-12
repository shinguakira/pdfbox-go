package multipdf_test

// JAVA-BUGS 82: `PDFMergerUtility.appendDocument`'s /Threads block reads the
// destination catalog twice, so the source's article threads are never merged
// and the destination's are cloned into themselves.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/multipdf"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// documentWithThreads is a one-page document whose catalog carries a /Threads
// array of the given titles, each an article thread with a /Title in its
// /Info.
func documentWithThreads(t *testing.T, titles ...string) *pdmodel.PDDocument {
	t.Helper()
	document := pdmodel.NewPDDocument()
	document.AddPage(pdmodel.NewPDPageOfSize(common.A4))
	threads := cos.NewArray()
	for _, title := range titles {
		info := cos.NewDictionary()
		info.SetString(cos.Title, title)
		thread := cos.NewDictionary()
		thread.SetItem(cos.GetPDFName("I"), info)
		threads.Add(thread)
	}
	document.DocumentCatalog().COSObject().(*cos.Dictionary).SetItem(cos.Threads, threads)
	return document
}

// threadTitles reads the titles back out of a document's /Threads.
func threadTitles(t *testing.T, document *pdmodel.PDDocument) []string {
	t.Helper()
	catalog := document.DocumentCatalog().COSObject().(*cos.Dictionary)
	threads := catalog.GetCOSArray(cos.Threads)
	if threads == nil {
		return nil
	}
	var titles []string
	for i := 0; i < threads.Size(); i++ {
		thread, isDictionary := threads.GetObject(i).(*cos.Dictionary)
		if !isDictionary {
			t.Fatalf("thread %d is %T, want a dictionary", i, threads.GetObject(i))
		}
		info := thread.GetCOSDictionary(cos.GetPDFName("I"))
		if info == nil {
			t.Fatalf("thread %d has no /I", i)
		}
		titles = append(titles, info.GetString(cos.Title, ""))
	}
	return titles
}

// TestMergeTakesTheSourcesThreads is the defect.
//
// The expected value is the destination's threads followed by the source's,
// which is what every other block of `appendDocument` does with the two
// documents. Java clones the destination's array and appends it to itself, so
// the source's threads are lost and the destination's are doubled -- a
// document merged with N others ends up with 2^N copies of its own.
func TestMergeTakesTheSourcesThreads(t *testing.T) {
	destination := documentWithThreads(t, "dest one")
	defer destination.Close()
	source := documentWithThreads(t, "src one", "src two")
	defer source.Close()

	if err := multipdf.NewPDFMergerUtility().AppendDocument(destination, source); err != nil {
		t.Fatalf("AppendDocument: %v", err)
	}

	got := threadTitles(t, destination)
	want := []string{"dest one", "src one", "src two"}
	if len(got) != len(want) {
		t.Fatalf("the merged /Threads holds %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("thread %d is %q, want %q", i, got[i], want[i])
		}
	}
}

// TestMergeIntoADocumentWithNoThreadsTakesTheSources is the other arm of the
// same block: the destination has none, so the source's array is put in whole.
func TestMergeIntoADocumentWithNoThreadsTakesTheSources(t *testing.T) {
	destination := pdmodel.NewPDDocument()
	defer destination.Close()
	destination.AddPage(pdmodel.NewPDPageOfSize(common.A4))
	source := documentWithThreads(t, "src one")
	defer source.Close()

	if err := multipdf.NewPDFMergerUtility().AppendDocument(destination, source); err != nil {
		t.Fatalf("AppendDocument: %v", err)
	}

	got := threadTitles(t, destination)
	if len(got) != 1 || got[0] != "src one" {
		t.Errorf("the merged /Threads holds %v, want the source's one thread", got)
	}
}

// TestMergeTheCheckedInPairTakesTheSourcesThreads is the same defect over two
// files on disk rather than two documents built in memory.
//
// `testdata/javabug82-dest.pdf` and `testdata/javabug82-src.pdf` are written by
// `testdata/genjavabug82.go` and are as small as a legitimate article thread
// gets: one page, one thread per title, one bead per thread with its /N and /V
// pointing at itself, which is what a one-bead chain is. ISO 32000-1 table 161
// for the thread, 162 for the bead. They are saved uncompressed so that
// /Threads can be read out of them with a text editor.
//
// They exist because a claim about the Java is worth more measured than
// argued. `testdata/Merge82Drv.java` runs the same merge through the Java, and
// its output is in JAVA-BUGS.md 82.
func TestMergeTheCheckedInPairTakesTheSourcesThreads(t *testing.T) {
	destination, err := pdfbox.LoadPDF("testdata/javabug82-dest.pdf")
	if err != nil {
		t.Fatalf("loading the destination: %v", err)
	}
	defer destination.Close()
	source, err := pdfbox.LoadPDF("testdata/javabug82-src.pdf")
	if err != nil {
		t.Fatalf("loading the source: %v", err)
	}
	defer source.Close()

	if got := threadTitles(t, destination); len(got) != 1 {
		t.Fatalf("the destination carries %v, want one thread", got)
	}
	if got := threadTitles(t, source); len(got) != 2 {
		t.Fatalf("the source carries %v, want two threads", got)
	}

	if err := multipdf.NewPDFMergerUtility().AppendDocument(destination, source); err != nil {
		t.Fatalf("AppendDocument: %v", err)
	}

	got := threadTitles(t, destination)
	want := []string{
		"Destination: quarterly report",
		"Source: appendix A",
		"Source: appendix B",
	}
	if len(got) != len(want) {
		t.Fatalf("the merged /Threads holds %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("thread %d is %q, want %q", i, got[i], want[i])
		}
	}
}
