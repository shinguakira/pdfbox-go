// Package pdfbox opens a PDF file.
//
// Port of the root of org.apache.pdfbox, which in Java is the single class
// Loader.
package pdfbox

import (
	"io"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/filter"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdfparser"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// LoadPDF opens the PDF file at the given path.
//
// Port of org.apache.pdfbox.Loader.loadPDF(File).
func LoadPDF(path string) (*pdmodel.PDDocument, error) {
	source, err := pdfio.OpenBufferedFile(path)
	if err != nil {
		return nil, err
	}
	document, err := LoadPDFFrom(source)
	if err != nil {
		source.Close()
		return nil, err
	}
	return document, nil
}

// LoadPDFBytes opens the PDF the given bytes hold.
//
// Port of loadPDF(byte[]).
func LoadPDFBytes(input []byte) (*pdmodel.PDDocument, error) {
	return LoadPDFFrom(pdfio.NewReadBufferBytes(input))
}

// LoadPDFReader opens the PDF the given reader yields, reading all of it first.
//
// Java has no such overload -- its callers wrap the stream in a
// RandomAccessReadBuffer themselves -- but every Go caller would write the same
// two lines.
func LoadPDFReader(input io.Reader) (*pdmodel.PDDocument, error) {
	source, err := pdfio.NewReadBufferFromReader(input)
	if err != nil {
		return nil, err
	}
	return LoadPDFFrom(source)
}

// LoadPDFFrom opens the PDF the given source holds, which it takes ownership
// of: closing the document closes the source.
//
// Port of loadPDF(RandomAccessRead).
func LoadPDFFrom(source pdfio.RandomAccessRead) (*pdmodel.PDDocument, error) {
	parser, err := pdfparser.NewPDFParser(source, nil, filter.Provider{})
	if err != nil {
		return nil, err
	}
	document, err := parser.Parse(true)
	if err != nil {
		return nil, err
	}
	pdDocument := pdmodel.NewPDDocumentOf(document, source)
	pdDocument.SetEncryptionDictionary(parser.Encryption())
	return pdDocument, nil
}

// LoadPDFStrict opens the PDF the given source holds, rejecting a file that
// does not follow the specification rather than repairing it.
//
// Java sets this through PDFParser.parse(boolean); the port names it.
func LoadPDFStrict(source pdfio.RandomAccessRead) (*pdmodel.PDDocument, error) {
	parser, err := pdfparser.NewPDFParser(source, nil, filter.Provider{})
	if err != nil {
		return nil, err
	}
	document, err := parser.Parse(false)
	if err != nil {
		return nil, err
	}
	pdDocument := pdmodel.NewPDDocumentOf(document, source)
	pdDocument.SetEncryptionDictionary(parser.Encryption())
	return pdDocument, nil
}
