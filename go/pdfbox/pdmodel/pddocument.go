package pdmodel

import (
	"errors"
	"math"
	"strconv"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// PDDocument is a PDF document.
//
// Port of org.apache.pdfbox.pdmodel.PDDocument.
//
// The reading path is here. Signatures, form fields, importing a page,
// subsetting a font and saving are not: each needs a package this port has not
// reached. See migration/STATUS.md.
type PDDocument struct {
	document *cos.Document

	documentInformation *PDDocumentInformation
	documentCatalog     *PDDocumentCatalog

	pdfSource pdfio.RandomAccessRead

	resourceCache ResourceCache
}

// NewPDDocument returns an empty document.
func NewPDDocument() *PDDocument {
	doc := cos.NewDocument(nil)
	trailer := cos.NewDictionary()
	doc.SetTrailer(trailer)
	d := &PDDocument{document: doc, resourceCache: NewDefaultResourceCache()}

	// initialise the document catalogue, which builds the page tree
	root := cos.NewDictionary()
	root.SetItem(cos.Type, cos.Catalog)
	trailer.SetItem(cos.Root, root)

	pages := cos.NewDictionary()
	pages.SetItem(cos.Type, cos.Pages)
	pages.SetItem(cos.Kids, cos.NewArray())
	pages.SetInt(cos.Count, 0)
	root.SetItem(cos.Pages, pages)
	return d
}

// NewPDDocumentOf returns the document the given COS document holds, read
// through the given source.
func NewPDDocumentOf(doc *cos.Document, source pdfio.RandomAccessRead) *PDDocument {
	return &PDDocument{
		document:      doc,
		pdfSource:     source,
		resourceCache: NewDefaultResourceCache(),
	}
}

// Document returns the COS document behind this one.
func (d *PDDocument) Document() *cos.Document { return d.document }

// ResourceCache returns the cache the document's resources are read through.
func (d *PDDocument) ResourceCache() ResourceCache { return d.resourceCache }

// SetResourceCache sets the cache the document's resources are read through.
func (d *PDDocument) SetResourceCache(cache ResourceCache) { d.resourceCache = cache }

// DocumentInformation returns the /Info dictionary of the document, creating
// one where the trailer has none.
func (d *PDDocument) DocumentInformation() *PDDocumentInformation {
	if d.documentInformation == nil {
		trailer := d.document.Trailer()
		infoDic := trailer.GetCOSDictionary(cos.Info)
		if infoDic == nil {
			infoDic = cos.NewDictionary()
			trailer.SetItem(cos.Info, infoDic)
		}
		d.documentInformation = NewPDDocumentInformation(infoDic)
	}
	return d.documentInformation
}

// SetDocumentInformation sets the /Info dictionary of the document.
func (d *PDDocument) SetDocumentInformation(info *PDDocumentInformation) {
	d.documentInformation = info
	d.document.Trailer().SetItem(cos.Info, info.COSObject())
}

// DocumentCatalog returns the catalogue of the document.
func (d *PDDocument) DocumentCatalog() *PDDocumentCatalog {
	if d.documentCatalog == nil {
		trailer := d.document.Trailer()
		dictionary := trailer.GetCOSDictionary(cos.Root)
		if dictionary != nil {
			d.documentCatalog = NewPDDocumentCatalogOf(d, dictionary)
		} else {
			d.documentCatalog = NewPDDocumentCatalog(d)
		}
	}
	return d.documentCatalog
}

// Page returns one page of the document, counting from zero.
func (d *PDDocument) Page(pageIndex int) *PDPage {
	return d.DocumentCatalog().Pages().Get(pageIndex)
}

// Pages returns the page tree of the document.
func (d *PDDocument) Pages() *PDPageTree {
	return d.DocumentCatalog().Pages()
}

// NumberOfPages returns how many pages the document has.
func (d *PDDocument) NumberOfPages() int {
	return d.DocumentCatalog().Pages().Count()
}

// AddPage adds a page to the end of the document.
func (d *PDDocument) AddPage(page *PDPage) {
	d.Pages().Add(page)
}

// IsEncrypted reports whether the document is encrypted.
func (d *PDDocument) IsEncrypted() bool { return d.document.IsEncrypted() }

// Version returns which version of the PDF specification the document claims,
// preferring the one the catalogue gives where it is the more recent.
func (d *PDDocument) Version() float32 {
	headerVersionFloat := d.Document().Version()
	// there may be a second version information in the document catalog
	// starting with 1.4
	if headerVersionFloat < 1.4 {
		return headerVersionFloat
	}
	catalogVersion := d.DocumentCatalog().Version()
	catalogVersionFloat := float32(-1)
	if catalogVersion != "" {
		if v, err := strconv.ParseFloat(catalogVersion, 32); err == nil {
			catalogVersionFloat = float32(v)
		}
	}
	// the most recent version is the correct one
	return float32(math.Max(float64(catalogVersionFloat), float64(headerVersionFloat)))
}

// Close releases the document and the file it was read from.
func (d *PDDocument) Close() error {
	if d.document.IsClosed() {
		return nil
	}
	// Make sure that:
	// - first Exception is kept
	// - all IO resources are closed
	// - there's a way to see which errors occurred
	var firstException error
	// close all intermediate I/O streams
	if err := d.document.Close(); err != nil {
		firstException = err
	}
	// close the source PDF stream, if we read from one
	if d.pdfSource != nil {
		if err := d.pdfSource.Close(); err != nil && firstException == nil {
			firstException = err
		}
	}
	// Java also closes the fonts it opened for subsetting; that set is only
	// filled by the writing path, which is not ported.
	return firstException
}

// PDDocumentCatalog is the root of the object graph of a document.
//
// Port of org.apache.pdfbox.pdmodel.PDDocumentCatalog. The form, outline,
// names, threads, metadata, actions and viewer preference accessors are not
// here: each needs a type this port has not reached. See migration/STATUS.md.
type PDDocumentCatalog struct {
	root     *cos.Dictionary
	document *PDDocument
}

var _ common_COSObjectable = (*PDDocumentCatalog)(nil)

// common_COSObjectable stands for pdmodel/common.COSObjectable, which this file
// cannot name without importing the package for one assertion.
type common_COSObjectable interface{ COSObject() cos.Base }

// NewPDDocumentCatalog returns a fresh catalogue for the given document.
func NewPDDocumentCatalog(doc *PDDocument) *PDDocumentCatalog {
	root := cos.NewDictionary()
	root.SetItem(cos.Type, cos.Catalog)
	doc.Document().Trailer().SetItem(cos.Root, root)
	return &PDDocumentCatalog{root: root, document: doc}
}

// NewPDDocumentCatalogOf returns the catalogue the given dictionary holds.
func NewPDDocumentCatalogOf(doc *PDDocument, rootDictionary *cos.Dictionary) *PDDocumentCatalog {
	return &PDDocumentCatalog{root: rootDictionary, document: doc}
}

// COSObject returns the dictionary behind the catalogue.
func (c *PDDocumentCatalog) COSObject() cos.Base { return c.root }

// Dictionary returns the dictionary behind the catalogue, typed.
func (c *PDDocumentCatalog) Dictionary() *cos.Dictionary { return c.root }

// Pages returns the page tree of the document.
func (c *PDDocumentCatalog) Pages() *PDPageTree {
	// todo: cache me?
	pages := c.root.GetCOSDictionary(cos.Pages)
	if pages == nil {
		return NewPDPageTree()
	}
	return NewPDPageTreeOfCache(pages, c.document.ResourceCache())
}

// Version returns the version of the PDF specification the catalogue claims.
func (c *PDDocumentCatalog) Version() string {
	return c.root.GetNameAsString(cos.Version, "")
}

// SetVersion sets the version of the PDF specification the catalogue claims.
func (c *PDDocumentCatalog) SetVersion(version string) {
	c.root.SetName(cos.Version, version)
}

// PDDocumentInformation is the /Info dictionary: what the document says about
// itself.
//
// Port of org.apache.pdfbox.pdmodel.PDDocumentInformation. The date accessors
// are not here: they need the COS date parsing, which slice 1 left out. See
// migration/STATUS.md.
type PDDocumentInformation struct {
	info *cos.Dictionary
}

// NewPDDocumentInformationEmpty returns an empty information dictionary.
func NewPDDocumentInformationEmpty() *PDDocumentInformation {
	return &PDDocumentInformation{info: cos.NewDictionary()}
}

// NewPDDocumentInformation returns the information the given dictionary holds.
func NewPDDocumentInformation(dic *cos.Dictionary) *PDDocumentInformation {
	return &PDDocumentInformation{info: dic}
}

// COSObject returns the dictionary behind the information.
func (i *PDDocumentInformation) COSObject() cos.Base { return i.info }

// Dictionary returns the dictionary behind the information, typed.
func (i *PDDocumentInformation) Dictionary() *cos.Dictionary { return i.info }

// Title returns the title of the document.
func (i *PDDocumentInformation) Title() string { return i.info.GetString(cos.Title, "") }

// SetTitle sets the title of the document.
func (i *PDDocumentInformation) SetTitle(title string) { i.setString(cos.Title, title) }

// Author returns who wrote the document.
func (i *PDDocumentInformation) Author() string { return i.info.GetString(cos.Author, "") }

// SetAuthor sets who wrote the document.
func (i *PDDocumentInformation) SetAuthor(author string) { i.setString(cos.Author, author) }

// Subject returns what the document is about.
func (i *PDDocumentInformation) Subject() string { return i.info.GetString(cos.Subject, "") }

// SetSubject sets what the document is about.
func (i *PDDocumentInformation) SetSubject(subject string) { i.setString(cos.Subject, subject) }

// Keywords returns the keywords of the document.
func (i *PDDocumentInformation) Keywords() string { return i.info.GetString(cos.Keywords, "") }

// SetKeywords sets the keywords of the document.
func (i *PDDocumentInformation) SetKeywords(keywords string) { i.setString(cos.Keywords, keywords) }

// Creator returns what made the original document.
func (i *PDDocumentInformation) Creator() string { return i.info.GetString(cos.Creator, "") }

// SetCreator sets what made the original document.
func (i *PDDocumentInformation) SetCreator(creator string) { i.setString(cos.Creator, creator) }

// Producer returns what turned the original into a PDF.
func (i *PDDocumentInformation) Producer() string { return i.info.GetString(cos.Producer, "") }

// SetProducer sets what turned the original into a PDF.
func (i *PDDocumentInformation) SetProducer(producer string) { i.setString(cos.Producer, producer) }

// CustomMetadataValue returns one of the entries a producer added of its own.
func (i *PDDocumentInformation) CustomMetadataValue(fieldName string) string {
	return i.info.GetString(cos.GetPDFName(fieldName), "")
}

// SetCustomMetadataValue sets one of the entries a producer adds of its own.
func (i *PDDocumentInformation) SetCustomMetadataValue(fieldName, fieldValue string) {
	i.setString(cos.GetPDFName(fieldName), fieldValue)
}

// setString writes a string entry, removing it for the empty string, which is
// what Java does for null.
func (i *PDDocumentInformation) setString(key *cos.Name, value string) {
	if value == "" {
		i.info.RemoveItem(key)
		return
	}
	i.info.SetItem(key, cos.NewStringObj(value))
}

// ErrMissingRoot is what a document with no catalogue is reported with.
var ErrMissingRoot = errors.New("pdmodel: Missing root object specification in trailer.")
