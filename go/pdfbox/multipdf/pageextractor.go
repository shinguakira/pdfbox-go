package multipdf

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
)

// PageExtractor extracts one or more sequential pages and creates a new
// document.
//
// Port of org.apache.pdfbox.multipdf.PageExtractor.
type PageExtractor struct {
	sourceDocument *pdmodel.PDDocument

	// first page to extract is page 1 (by default)
	startPage int

	endPage int
}

// NewPageExtractor creates a new instance of PageExtractor over the whole
// document.
func NewPageExtractor(sourceDocument *pdmodel.PDDocument) *PageExtractor {
	return &PageExtractor{
		sourceDocument: sourceDocument,
		startPage:      1,
		endPage:        sourceDocument.NumberOfPages(),
	}
}

// NewPageExtractorOfRange creates a new instance of PageExtractor over the
// given page range, both 1-based and inclusive.
func NewPageExtractorOfRange(sourceDocument *pdmodel.PDDocument, startPage, endPage int) *PageExtractor {
	return &PageExtractor{
		sourceDocument: sourceDocument,
		startPage:      startPage,
		endPage:        endPage,
	}
}

// Extract takes the document and extracts the desired pages into a new
// document. Both startPage and endPage are included in the extracted document.
// If the endPage is greater than the number of pages in the source document, it
// will go to the end of the document. If startPage is less than 1, it'll start
// with page 1. If startPage is greater than endPage or greater than the number
// of pages in the source document, a blank document will be returned.
func (e *PageExtractor) Extract() (*pdmodel.PDDocument, error) {
	if e.endPage-e.startPage+1 <= 0 {
		return pdmodel.NewPDDocument(), nil
	}
	splitter := NewSplitter()
	splitter.SetStartPage(max(e.startPage, 1))
	splitter.SetEndPage(min(e.endPage, e.sourceDocument.NumberOfPages()))
	splitter.SetSplitAtPage(e.EndPage() - e.StartPage() + 1)
	splitted, err := splitter.Split(e.sourceDocument)
	if err != nil {
		return nil, err
	}
	return splitted[0], nil
}

// StartPage gets the first page number to be extracted.
func (e *PageExtractor) StartPage() int { return e.startPage }

// SetStartPage sets the first page number to be extracted.
func (e *PageExtractor) SetStartPage(startPage int) { e.startPage = startPage }

// EndPage gets the last page number (inclusive) to be extracted.
func (e *PageExtractor) EndPage() int { return e.endPage }

// SetEndPage sets the last page number to be extracted.
func (e *PageExtractor) SetEndPage(endPage int) { e.endPage = endPage }
