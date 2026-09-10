//go:build ignore

// Writes the two documents that show what JAVA-BUGS.md 82 does to a merge.
//
// Regenerate them with:
//
//	go run pdfbox/multipdf/testdata/genjavabug82.go
//
// `appendDocument` reads the destination catalog twice where it means to read
// the source once:
//
//	COSArray destThreads = destCatalog.getCOSObject().getCOSArray(COSName.THREADS);
//	COSArray srcThreads = cloner.cloneForNewDocument(destCatalog.getCOSObject().getCOSArray(
//	        COSName.THREADS));
//
// so a merge appends the destination's own article threads to itself and never
// takes the source's. Merging src into dest must leave three threads named
//
//	Destination: quarterly report
//	Source: appendix A
//	Source: appendix B
//
// and leaves two named "Destination: quarterly report" instead.
//
// The two files are as small as a legitimate article thread gets: one page,
// one bead per thread, the bead's /N and /V pointing at itself, which is what
// a one-bead chain is. ISO 32000-1 table 161 for the thread, 162 for the bead.
// Nothing else is in either file, so nothing else can be what moved.
package main

import (
	"log"
	"os"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdfwriter/compress"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/pagenavigation"
)

func main() {
	write("pdfbox/multipdf/testdata/javabug82-dest.pdf",
		"destination", "Destination: quarterly report")
	write("pdfbox/multipdf/testdata/javabug82-src.pdf",
		"source", "Source: appendix A", "Source: appendix B")
}

// write builds a one-page document carrying one article thread per title.
func write(path, role string, titles ...string) {
	document := pdmodel.NewPDDocument()
	defer document.Close()

	page := pdmodel.NewPDPageOfSize(common.A4)
	document.AddPage(page)
	page.SetResources(helvetica())
	page.Dictionary().SetItem(cos.Contents, label(role, titles))

	threads := cos.NewArray()
	for i, title := range titles {
		threads.Add(thread(page, title, i).COSObject())
	}
	document.DocumentCatalog().Dictionary().SetItem(cos.Threads, threads)

	out, err := os.Create(path)
	check(err)
	// Uncompressed, so that /Threads can be read with a text editor: a
	// reproducing file nobody can read is half a report.
	check(document.SaveOfParameters(out, compress.NoCompression))
	check(out.Close())
	log.Printf("wrote %s: %d thread(s)", path, len(titles))
}

// thread is one article thread with a single bead, on the given page, in the
// nth horizontal band of it.
func thread(page *pdmodel.PDPage, title string, n int) *pagenavigation.PDThread {
	info := cos.NewDictionary()
	info.SetString(cos.Title, title)

	t := pagenavigation.NewPDThread()
	t.Dictionary().SetItem(cos.I, info)

	bead := pagenavigation.NewPDThreadBead()
	bead.SetPage(page)
	top := float32(700 - 120*n)
	bead.SetRectangle(common.NewPDRectangleOf(56, top-100, 483, 100))
	t.SetFirstBead(bead)
	return t
}

// helvetica is the one resource either page needs, so that a reader opening
// the file sees which of the two it is.
func helvetica() *pdmodel.PDResources {
	font := cos.NewDictionary()
	font.SetItem(cos.Type, cos.Font)
	font.SetItem(cos.Subtype, cos.GetPDFName("Type1"))
	font.SetItem(cos.BaseFont, cos.GetPDFName("Helvetica"))

	fonts := cos.NewDictionary()
	fonts.SetItem(cos.GetPDFName("F1"), font)

	resources := pdmodel.NewPDResources()
	resources.Dictionary().SetItem(cos.Font, fonts)
	return resources
}

// label writes the page's text: which file it is and what it carries.
func label(role string, titles []string) *cos.Stream {
	content := "BT /F1 16 Tf 56 760 Td (JAVA-BUGS 82 -- the " + role + " document) Tj ET\n"
	content += "BT /F1 11 Tf 56 735 Td (Its catalog carries these article threads:) Tj ET\n"
	for i, title := range titles {
		y := 712 - 16*i
		content += "BT /F1 11 Tf 70 " + itoa(y) + " Td (" + title + ") Tj ET\n"
	}
	stream := cos.NewStream(nil)
	writer, err := stream.CreateWriter()
	check(err)
	_, err = writer.Write([]byte(content))
	check(err)
	check(writer.Close())
	return stream
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
