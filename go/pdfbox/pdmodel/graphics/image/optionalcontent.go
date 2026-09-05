package image

// The optional content an image belongs to.
//
// Port of PDImageXObject.getOptionalContent and setOptionalContent, kept in a
// file of its own because they name a type the rest of this package does not.

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/documentinterchange/markedcontent"
)

// OptionalContent returns the optional content group or optional content
// membership dictionary this image belongs to, or nil.
func (i *PDImageXObject) OptionalContent() markedcontent.PropertyList {
	optionalContent := i.Stream().GetCOSDictionary(cos.OC)
	if optionalContent == nil {
		return nil
	}
	return markedcontent.CreatePropertyList(optionalContent)
}

// SetOptionalContent sets the optional content group or optional content
// membership dictionary.
func (i *PDImageXObject) SetOptionalContent(oc markedcontent.PropertyList) {
	i.Stream().SetItem(cos.OC, oc.COSObject())
}
