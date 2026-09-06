package pdfbox

import (
	"io"
	"os"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/filter"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdfparser"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/fdf"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// LoadFDF opens the FDF file at the given path.
//
// Port of org.apache.pdfbox.Loader.loadFDF(File).
func LoadFDF(path string) (*fdf.FDFDocument, error) {
	// PDFBOX-5894: RandomAccessRead is not closed here
	source, err := pdfio.OpenBufferedFile(path)
	if err != nil {
		return nil, err
	}
	document, err := LoadFDFFrom(source)
	if err != nil {
		source.Close()
		return nil, err
	}
	return document, nil
}

// LoadFDFReader opens the FDF the given reader yields, reading all of it first.
//
// Port of loadFDF(InputStream), which wraps the stream in a
// RandomAccessReadBuffer and closes it when the parse is done.
func LoadFDFReader(input io.Reader) (*fdf.FDFDocument, error) {
	source, err := pdfio.NewReadBufferFromReader(input)
	if err != nil {
		return nil, err
	}
	defer source.Close()
	return LoadFDFFrom(source)
}

// LoadFDFFrom opens the FDF the given source holds, which it takes ownership
// of: closing the document closes the source.
//
// Java has no such overload; both of its loadFDF methods end here, and the port
// names the shared body so that the wrapping the parser cannot do -- see
// pdfparser.FDFParser -- happens in one place.
func LoadFDFFrom(source pdfio.RandomAccessRead) (*fdf.FDFDocument, error) {
	parser, err := pdfparser.NewFDFParser(source, nil, filter.Provider{})
	if err != nil {
		return nil, err
	}
	document, err := parser.Parse()
	if err != nil {
		return nil, err
	}
	return fdf.NewFDFDocumentOf(document, source), nil
}

// openFile opens the file at the given path for reading, which is the
// BufferedInputStream over a FileInputStream of Java.
func openFile(path string) (io.ReadCloser, error) { return os.Open(path) }

// LoadXFDF opens the XFDF file at the given path.
//
// Port of loadXFDF(File).
func LoadXFDF(path string) (*fdf.FDFDocument, error) {
	file, err := openFile(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return LoadXFDFReader(file)
}

// LoadXFDFReader opens the XFDF the given reader yields.
//
// Port of loadXFDF(InputStream).
func LoadXFDFReader(input io.Reader) (*fdf.FDFDocument, error) {
	document, err := util.XMLParse(input)
	if err != nil {
		return nil, err
	}
	return fdf.NewFDFDocumentOfXML(document)
}
