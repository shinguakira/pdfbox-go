package multipdf

import (
	"log/slog"
	"math"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
)

// Splitter splits a PDF document into many.
//
// Port of org.apache.pdfbox.multipdf.Splitter.
//
// Five of its private methods are not here, and split therefore does less than
// Java's: fixDestinations, cloneStructureTree, cloneIDTree, cloneRoleMap,
// cloneTreeElement, processResources and processAnnotations. Each of them
// works on pdmodel/interactive and pdmodel/documentinterchange/logicalstructure
// -- page destinations, link and widget annotations, the structure tree -- and
// slice 8 brings that whole subtree. The pages themselves, their content and
// their resources are split as Java splits them; what is left behind is the
// structure tree, the outline destinations and the annotations. The three maps
// those methods fill are kept, empty, so that the shape of the port matches.
// See migration/STATUS.md.
type Splitter struct {
	sourceDocument             *pdmodel.PDDocument
	currentDestinationDocument *pdmodel.PDDocument

	splitLength          int
	startPage            int
	endPage              int
	destinationDocuments []*pdmodel.PDDocument
	// pageDictMap maps old page => new page for the current destination document
	pageDictMap map[*cos.Dictionary]*cos.Dictionary
	// pageDictMaps is the list of these maps for all destination documents
	pageDictMaps []map[*cos.Dictionary]*cos.Dictionary
	// annotDictMap maps old annotation => new annotation for the current destination document
	annotDictMap map[*cos.Dictionary]*cos.Dictionary
	// annotDictMaps is the list of these maps for all destination documents
	annotDictMaps []map[*cos.Dictionary]*cos.Dictionary

	currentPageNumber int
}

// NewSplitter returns a splitter that splits at every page, which is Java's
// field initialisers.
func NewSplitter() *Splitter {
	return &Splitter{
		splitLength: 1,
		startPage:   math.MinInt32,
		endPage:     math.MaxInt32,
	}
}

// Split splits the given document into many documents.
func (s *Splitter) Split(document *pdmodel.PDDocument) ([]*pdmodel.PDDocument, error) {
	// reset the currentPageNumber for a case if the split method will be used several times
	s.currentPageNumber = 0
	s.destinationDocuments = nil
	s.sourceDocument = document
	s.pageDictMaps = nil
	s.annotDictMaps = nil

	if err := s.processPages(); err != nil {
		return nil, err
	}

	// Java walks the destination documents here and calls cloneStructureTree and
	// fixDestinations on each. Both need slice 8; see the type comment.

	return s.destinationDocuments, nil
}

// processPages walks the source pages and hands the ones in range to
// processPage.
func (s *Splitter) processPages() error {
	for i := 0; i < s.sourceDocument.NumberOfPages(); i++ {
		page := s.sourceDocument.Page(i)
		if s.currentPageNumber+1 >= s.startPage && s.currentPageNumber+1 <= s.endPage {
			if err := s.processPage(page); err != nil {
				return err
			}
			s.currentPageNumber++
		} else {
			if s.currentPageNumber > s.endPage {
				break
			}
			s.currentPageNumber++
		}
	}
	return nil
}

// createNewDocumentIfNecessary is a helper method for creating new documents at
// the appropriate pages.
func (s *Splitter) createNewDocumentIfNecessary() error {
	if s.SplitAtPage(s.currentPageNumber) || s.currentDestinationDocument == nil {
		document, err := s.CreateNewDocument()
		if err != nil {
			return err
		}
		s.currentDestinationDocument = document
		s.destinationDocuments = append(s.destinationDocuments, s.currentDestinationDocument)
		s.pageDictMap = map[*cos.Dictionary]*cos.Dictionary{}
		s.pageDictMaps = append(s.pageDictMaps, s.pageDictMap)
		s.annotDictMap = map[*cos.Dictionary]*cos.Dictionary{}
		s.annotDictMaps = append(s.annotDictMaps, s.annotDictMap)
	}
	return nil
}

// SplitAtPage checks if it is necessary to create a new document. By default a
// split occurs at every page.
//
// Java declares it protected so that a subclass can override it with more
// complex logic; Go has no subclassing, and the port exports it so that the
// same decision can be seen and, in a wrapper, replaced.
func (s *Splitter) SplitAtPage(pageNumber int) bool {
	return (pageNumber+1-max(1, s.startPage))%s.splitLength == 0
}

// CreateNewDocument creates a new document to write the split contents to.
func (s *Splitter) CreateNewDocument() (*pdmodel.PDDocument, error) {
	document := pdmodel.NewPDDocument()
	document.Document().SetVersion(s.SourceDocument().Version())
	sourceDocumentInformation := s.SourceDocument().DocumentInformation()
	if sourceDocumentInformation != nil {
		// PDFBOX-5317: Image Capture Plus files where /Root and /Info share the same dictionary
		// Only copy simple elements to avoid huge files
		sourceDocumentInformationDictionary := sourceDocumentInformation.Dictionary()
		destDocumentInformationDictionary := cos.NewDictionary()
		for _, key := range sourceDocumentInformationDictionary.KeySet() {
			value := sourceDocumentInformationDictionary.GetDictionaryObject(key)
			if _, isDictionary := asDictionary(value); isDictionary {
				slog.Warn("multipdf: Nested entry skipped in document information dictionary",
					"key", key.Name())
				if s.sourceDocument.DocumentCatalog().Dictionary() ==
					s.sourceDocument.DocumentInformation().Dictionary() {
					slog.Warn("multipdf: /Root and /Info share the same dictionary")
				}
				continue
			}
			if key == cos.Type {
				continue // there is no /Type in the document information dictionary
			}
			destDocumentInformationDictionary.SetItem(key, value)
		}
		document.SetDocumentInformation(
			pdmodel.NewPDDocumentInformation(destDocumentInformationDictionary))
	}
	destCatalog := document.DocumentCatalog()
	// Java copies the viewer preferences, the language, the mark info and the
	// metadata from the source catalog here. All four need types slice 8 brings;
	// see the type comment.
	// reset reused object keys to avoid gaps in the xref table
	destCatalog.Dictionary().ResetImportedObjectKeys()
	return document, nil
}

// processPage starts processing a new page.
func (s *Splitter) processPage(page *pdmodel.PDPage) error {
	if err := s.createNewDocumentIfNecessary(); err != nil {
		return err
	}

	imported, err := s.DestinationDocument().ImportPage(page)
	if err != nil {
		return err
	}
	if page.Resources() != nil && !page.Dictionary().ContainsKey(cos.Resources) {
		imported.SetResources(page.Resources())
		slog.Info("multipdf: Resources imported in Splitter") // follow-up to warning in importPage
	}
	if imported.Dictionary().ContainsKey(cos.B) {
		imported.Dictionary().RemoveItem(cos.B)
		slog.Warn("multipdf: /B entry (beads) removed by splitter")
	}
	// Java removes the page links here, with processAnnotations; that needs
	// slice 8, so the annotations of the imported page are left as they are.

	s.pageDictMap[page.Dictionary()] = imported.Dictionary()
	return nil
}

// SetSplitAtPage tells the splitting algorithm where to split the pages. The
// default is 1, so every page will become a new document. If it was two then
// each document would contain 2 pages. If the source document had 5 pages it
// would split into 3 new documents, 2 documents containing 2 pages and 1
// document containing one page.
//
// Java throws IllegalArgumentException for a page smaller than one, which is
// unchecked, so the port panics.
func (s *Splitter) SetSplitAtPage(split int) {
	if split <= 0 {
		panic("Number of pages is smaller than one")
	}
	s.splitLength = split
}

// SetStartPage sets the 1-based start page.
//
// Java throws IllegalArgumentException for a start page smaller than one.
func (s *Splitter) SetStartPage(start int) {
	if start <= 0 {
		panic("Start page is smaller than one")
	}
	s.startPage = start
}

// SetEndPage sets the 1-based end page.
//
// Java throws IllegalArgumentException for an end page smaller than one or than
// the start page.
func (s *Splitter) SetEndPage(end int) {
	if end <= 0 {
		panic("End page is smaller than one")
	}
	if end < s.startPage {
		panic("End page is smaller than startPage")
	}
	s.endPage = end
}

// SourceDocument returns the pdf to be split.
func (s *Splitter) SourceDocument() *pdmodel.PDDocument { return s.sourceDocument }

// DestinationDocument returns the current destination pdf.
func (s *Splitter) DestinationDocument() *pdmodel.PDDocument {
	return s.currentDestinationDocument
}
