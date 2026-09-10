//go:build ignore

// Merges javabug82-src.pdf into javabug82-dest.pdf with the port and writes
// javabug82-merged-go.pdf, the file to hold beside javabug82-merged-java.pdf.
//
//	go run pdfbox/multipdf/testdata/genjavabug82merged.go
//
// The two merged files are the point of the pair: open either and the two
// pages look the same, because /Threads is not something a page draws. The
// difference is in the catalog, which is why both are saved uncompressed.
package main

import (
	"log"
	"os"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/multipdf"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdfwriter/compress"
)

func main() {
	const dir = "pdfbox/multipdf/testdata/"

	destination, err := pdfbox.LoadPDF(dir + "javabug82-dest.pdf")
	check(err)
	defer destination.Close()
	source, err := pdfbox.LoadPDF(dir + "javabug82-src.pdf")
	check(err)
	defer source.Close()

	check(multipdf.NewPDFMergerUtility().AppendDocument(destination, source))

	out, err := os.Create(dir + "javabug82-merged-go.pdf")
	check(err)
	check(destination.SaveOfParameters(out, compress.NoCompression))
	check(out.Close())
	log.Printf("wrote %sjavabug82-merged-go.pdf", dir)
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
