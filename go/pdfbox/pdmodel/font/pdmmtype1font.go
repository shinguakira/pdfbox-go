package font

import "github.com/shinguakira/pdfbox-go/go/pdfbox/cos"

// NewPDMMType1Font creates a Type 1 Multiple Master font from a font dictionary
// in a PDF.
//
// Port of org.apache.pdfbox.pdmodel.font.PDMMType1Font, whose whole body is a
// call to the PDType1Font constructor with no resource cache. Java makes it a
// subclass so that the font's own class names it; nothing else in the library
// tests for that class, so the port is the constructor alone.
func NewPDMMType1Font(fontDictionary *cos.Dictionary) (*PDType1Font, error) {
	return NewPDType1FontFromDictionary(fontDictionary, nil)
}
