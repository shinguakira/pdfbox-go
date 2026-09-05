package pdfparser

import (
	"fmt"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// FDFParser opens an FDF file: the header and the trailer.
//
// Port of org.apache.pdfbox.pdfparser.FDFParser. Java extends COSParser; the
// port embeds FileParser, which is that class's file half.
//
// Building the FDFDocument the parse returns is the caller's job here rather
// than the parser's, exactly as it is for PDFParser: pdmodel/fdf sits above
// this package -- it imports pdfwriter, which imports this one -- so having the
// parser name FDFDocument would put a cycle where Java has none. Loader does
// the wrapping.
type FDFParser struct {
	*FileParser
}

// NewFDFParser returns a parser over the given file.
func NewFDFParser(source pdfio.RandomAccessRead, cache pdfio.StreamCache,
	codecs cos.CodecProvider) (*FDFParser, error) {
	fileParser, err := NewFileParser(source, cache, codecs)
	if err != nil {
		return nil, err
	}
	return &FDFParser{FileParser: fileParser}, nil
}

// initialParse reads the trailer and checks that it names a root. Java declares
// it private.
func (p *FDFParser) initialParse() error {
	trailer, err := p.RetrieveTrailer()
	if err != nil {
		return err
	}
	if trailer == nil {
		return fmt.Errorf("pdfparser: Missing root object specification in trailer.")
	}
	root := trailer.GetCOSDictionary(cos.Root)
	if root == nil {
		return fmt.Errorf("pdfparser: Missing root object specification in trailer.")
	}
	p.initialParseDone = true
	return nil
}

// Parse reads the file and returns the COS document it holds.
//
// Java returns an FDFDocument; the port stops one level lower and lets Loader
// wrap it, for the reason the type comment gives.
func (p *FDFParser) Parse() (*cos.Document, error) {
	// set to false if all is processed
	exceptionOccurred := true
	defer func() {
		if exceptionOccurred && p.Document() != nil {
			p.Document().Close()
		}
	}()

	isFDF, err := p.ParseFDFHeader()
	if err != nil {
		return nil, err
	}
	if !isFDF {
		return nil, fmt.Errorf("pdfparser: Error: Header doesn't contain versioninfo")
	}
	if err := p.initialParse(); err != nil {
		return nil, err
	}
	exceptionOccurred = false
	return p.Document(), nil
}
