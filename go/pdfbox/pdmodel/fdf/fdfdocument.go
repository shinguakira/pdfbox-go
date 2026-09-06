package fdf

import (
	"errors"
	"io"
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdfwriter"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/digitalsignature"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
	"github.com/shinguakira/pdfbox-go/go/w3c/dom"
)

// FDFCatalog is the root dictionary of an FDF document.
//
// Port of FDFCatalog.
type FDFCatalog struct {
	catalog *cos.Dictionary
}

var _ common.COSObjectable = (*FDFCatalog)(nil)

// NewFDFCatalog returns an empty catalogue.
func NewFDFCatalog() *FDFCatalog { return &FDFCatalog{catalog: cos.NewDictionary()} }

// NewFDFCatalogOf returns the catalogue the given dictionary holds.
func NewFDFCatalogOf(cat *cos.Dictionary) *FDFCatalog { return &FDFCatalog{catalog: cat} }

// NewFDFCatalogOfXML returns the catalogue the given XFDF element describes.
func NewFDFCatalogOfXML(element *dom.Element) *FDFCatalog {
	c := NewFDFCatalog()
	fdfDict := NewFDFDictionaryOfXML(element)
	c.SetFDF(fdfDict)
	return c
}

// WriteXML writes this catalogue out as XFDF.
func (c *FDFCatalog) WriteXML(output io.Writer) error {
	return c.FDF().WriteXML(output)
}

// COSObject returns the dictionary.
func (c *FDFCatalog) COSObject() cos.Base { return c.catalog }

// Dictionary returns the dictionary, typed.
func (c *FDFCatalog) Dictionary() *cos.Dictionary { return c.catalog }

// Version returns the /Version of the catalogue, or the empty string where it
// has none.
func (c *FDFCatalog) Version() string { return c.catalog.GetNameAsString(cos.Version, "") }

// SetVersion sets the /Version of the catalogue.
func (c *FDFCatalog) SetVersion(version string) { c.catalog.SetName(cos.Version, version) }

// FDF returns the /FDF dictionary of the catalogue, and writes an empty one into
// it where it has none.
func (c *FDFCatalog) FDF() *FDFDictionary {
	fdf := c.catalog.GetCOSDictionary(cos.FDF)
	var retval *FDFDictionary
	if fdf != nil {
		retval = NewFDFDictionaryOf(fdf)
	} else {
		retval = NewFDFDictionary()
		c.SetFDF(retval)
	}
	return retval
}

// SetFDF sets the /FDF dictionary of the catalogue.
func (c *FDFCatalog) SetFDF(fdf *FDFDictionary) {
	c.catalog.SetItem(cos.FDF, common.COSObjectOrNil(fdf))
}

// Signature returns the /Sig of the catalogue, or nil where it has none.
func (c *FDFCatalog) Signature() *digitalsignature.PDSignature {
	sig := c.catalog.GetCOSDictionary(cos.Sig)
	if sig != nil {
		return digitalsignature.NewPDSignatureOf(sig)
	}
	return nil
}

// SetSignature sets the /Sig of the catalogue.
func (c *FDFCatalog) SetSignature(sig *digitalsignature.PDSignature) {
	c.catalog.SetItem(cos.Sig, common.COSObjectOrNil(sig))
}

// FDFDocument is a Forms Data Format document.
//
// Port of FDFDocument, which implements Closeable.
type FDFDocument struct {
	document  *cos.Document
	fdfSource pdfio.RandomAccessRead
}

// NewFDFDocument returns an empty FDF document.
func NewFDFDocument() *FDFDocument {
	d := &FDFDocument{document: cos.NewDocument(nil)}
	d.document.DocumentState().SetParsing(false)
	d.document.SetVersion(1.2)
	// First we need a trailer
	d.document.SetTrailer(cos.NewDictionary())
	// Next we need the root dictionary.
	catalog := NewFDFCatalog()
	d.SetCatalog(catalog)
	return d
}

// NewFDFDocumentOf returns the FDF document the given COS document holds, read
// from the given source.
func NewFDFDocumentOf(doc *cos.Document, source pdfio.RandomAccessRead) *FDFDocument {
	d := &FDFDocument{document: doc, fdfSource: source}
	d.document.DocumentState().SetParsing(false)
	return d
}

// NewFDFDocumentOfXML returns the FDF document the given XFDF tree describes.
func NewFDFDocumentOfXML(doc *dom.Document) (*FDFDocument, error) {
	d := NewFDFDocument()
	xfdf := doc.DocumentElement()
	if xfdf == nil || xfdf.NodeName() != "xfdf" {
		name := ""
		if xfdf != nil {
			name = xfdf.NodeName()
		}
		return nil, errors.New(
			"Error while importing xfdf document, root should be 'xfdf' and not '" + name + "'")
	}
	cat := NewFDFCatalogOfXML(xfdf)
	d.SetCatalog(cat)
	return d, nil
}

// WriteXML writes this document out as XFDF.
func (d *FDFDocument) WriteXML(output io.Writer) error {
	w := &xmlWriter{out: output}
	w.write("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	w.write("<xfdf xmlns=\"http://ns.adobe.com/xfdf/\" xml:space=\"preserve\">\n")
	if w.err != nil {
		return w.err
	}
	if err := d.Catalog().WriteXML(output); err != nil {
		return err
	}
	w.write("</xfdf>\n")
	return w.err
}

// Document returns the COS document behind this one.
func (d *FDFDocument) Document() *cos.Document { return d.document }

// Catalog returns the catalogue of the document, and writes an empty one into
// the trailer where it has none.
func (d *FDFDocument) Catalog() *FDFCatalog {
	var retval *FDFCatalog
	trailer := d.document.Trailer()
	root := trailer.GetCOSDictionary(cos.Root)
	if root == nil {
		retval = NewFDFCatalog()
		d.SetCatalog(retval)
	} else {
		retval = NewFDFCatalogOf(root)
	}
	return retval
}

// SetCatalog sets the catalogue of the document.
func (d *FDFDocument) SetCatalog(cat *FDFCatalog) {
	trailer := d.document.Trailer()
	trailer.SetItem(cos.Root, common.COSObjectOrNil(cat))
}

// Save writes this document out as FDF.
//
// Java has three overloads: File, String and OutputStream. The first two open
// the file and call the third; a Go caller opens the file, so only the writer
// form is here.
func (d *FDFDocument) Save(output io.Writer) error {
	writer := pdfwriter.NewCOSWriter(output)
	return writer.WriteFDF(d.document)
}

// SaveXFDF writes this document out as XFDF.
//
// Java has three overloads here too, and closes the writer it is given; a Go
// caller closes what it opened, so this one does not.
func (d *FDFDocument) SaveXFDF(output io.Writer) error {
	return d.WriteXML(output)
}

// Close closes the COS document and the source it was read from.
func (d *FDFDocument) Close() error {
	if d.document.IsClosed() {
		return nil
	}
	var firstException error
	// close all intermediate I/O streams
	if err := d.document.Close(); err != nil {
		slog.Error("fdf: error closing COSDocument", slog.Any("err", err))
		firstException = err
	}
	// close the source PDF stream, if we read from one
	if d.fdfSource != nil {
		if err := d.fdfSource.Close(); err != nil {
			slog.Error("fdf: error closing RandomAccessRead pdfSource", slog.Any("err", err))
			if firstException == nil {
				firstException = err
			}
		}
	}
	// rethrow first exception to keep method contract
	return firstException
}
