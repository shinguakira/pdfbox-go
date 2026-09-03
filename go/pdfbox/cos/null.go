package cos

import "io"

// Null is the PDF null object.
//
// Port of org.apache.pdfbox.cos.COSNull. As in Java there is exactly one
// instance, NullObject; the type is exported only so the Visitor can name it.
type Null struct {
	object
}

var _ Base = (*Null)(nil)

// NullObject is the one null object in the system.
//
// Java calls the singleton COSNull.NULL. Here the type takes the name Null, so
// the value is NullObject — callers compare against cos.NullObject.
var NullObject = &Null{}

// NullBytes is the null token as written to a PDF.
var NullBytes = []byte{110, 117, 108, 108} // "null" in ISO-8859-1

// COSObject returns the receiver.
func (n *Null) COSObject() Base { return n }

// Accept dispatches to the visitor.
func (n *Null) Accept(v Visitor) error { return v.VisitNull(n) }

// String returns the Java toString form, which the Java tests and log output
// depend on.
func (n *Null) String() string { return "COSNull{}" }

// WritePDF writes the null token.
func (n *Null) WritePDF(w io.Writer) error {
	_, err := w.Write(NullBytes)
	return err
}
