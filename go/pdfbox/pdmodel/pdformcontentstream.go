package pdmodel

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/form"
)

// PDFormContentStream writes the content stream of a form XObject.
//
// Port of PDFormContentStream, which Java declares final.
type PDFormContentStream struct {
	pdAbstractContentStream
}

// NewPDFormContentStream writes into the given form XObject.
func NewPDFormContentStream(formXObject *form.PDFormXObject) (*PDFormContentStream, error) {
	outputStream, err := formXObject.ContentStream().CreateOutputStream()
	if err != nil {
		return nil, err
	}
	c := &PDFormContentStream{}
	resources, _ := formXObject.Resources().(*PDResources)
	c.initAbstractContentStream(nil, outputStream, resources)
	return c, nil
}
