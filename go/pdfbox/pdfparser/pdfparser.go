package pdfparser

import (
	"fmt"
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// PDFParser opens a PDF file: the header, the trailer and the page tree.
//
// Port of org.apache.pdfbox.pdfparser.PDFParser. Java extends COSParser; the
// port embeds FileParser, which is that class's file half.
//
// Building the PDDocument the parse returns is the caller's job here rather
// than the parser's: pdmodel imports this package for nothing today, but the
// document model sits above the parser and having the parser name it would put
// a cycle where Java has none. Loader does the wrapping.
type PDFParser struct {
	*FileParser
}

// NewPDFParser returns a parser over the given file.
func NewPDFParser(source pdfio.RandomAccessRead, cache pdfio.StreamCache, codecs cos.CodecProvider) (*PDFParser, error) {
	fileParser, err := NewFileParser(source, cache, codecs)
	if err != nil {
		return nil, err
	}
	return &PDFParser{FileParser: fileParser}, nil
}

// initialParse reads the trailer and checks the page tree.
func (p *PDFParser) initialParse() error {
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
	// in some pdfs the type value "Catalog" is missing in the root object
	if p.IsLenient() && !root.ContainsKey(cos.Type) {
		root.SetItem(cos.Type, cos.Catalog)
	}
	// check pages dictionaries
	if err := p.CheckPages(root); err != nil {
		return err
	}
	p.Document().SetDecrypted()
	p.initialParseDone = true
	return nil
}

// Parse reads the file and returns the COS document it holds.
//
// Java returns a PDDocument; the port stops one level lower and lets Loader
// wrap it, for the reason the type comment gives.
func (p *PDFParser) Parse(lenient bool) (*cos.Document, error) {
	p.SetLenient(lenient)

	// PDFBOX-1922 read the version header and rewind
	isPDF, err := p.ParsePDFHeader()
	if err != nil {
		return nil, p.failed(err)
	}
	if !isPDF {
		isFDF, err := p.ParseFDFHeader()
		if err != nil {
			return nil, p.failed(err)
		}
		if !isFDF {
			if !lenient {
				return nil, p.failed(fmt.Errorf("pdfparser: Error: Header doesn't contain versioninfo"))
			}
			slog.Warn("Error: Header doesn't contain versioninfo")
		}
	}

	if !p.initialParseDone {
		if err := p.initialParse(); err != nil {
			return nil, p.failed(err)
		}
	}
	return p.Document(), nil
}

// failed closes the document a failed parse half-built, as Java's finally does.
func (p *PDFParser) failed(err error) error {
	if document := p.Document(); document != nil {
		document.Close()
	}
	return err
}
