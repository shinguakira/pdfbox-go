package pdmodel

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/pagenavigation"
)

// PDThreadBead.getPage and PDThread.getThreadInfo build a PDPage and a
// PDDocumentInformation, which live here. pagenavigation cannot import this
// package, because PDPage.getThreadBeads goes the other way, so it declares
// what it needs and takes the two constructors; these are them.
func init() {
	pagenavigation.NewPageFromDictionary = func(dic *cos.Dictionary) pagenavigation.PageLike {
		return NewPDPageOf(dic)
	}
	pagenavigation.NewDocumentInformationFromDictionary =
		func(dic *cos.Dictionary) pagenavigation.DocumentInformationLike {
			return NewPDDocumentInformation(dic)
		}
}
