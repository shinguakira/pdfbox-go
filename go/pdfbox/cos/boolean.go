package cos

import "io"

// Boolean is a PDF boolean value.
//
// Port of org.apache.pdfbox.cos.COSBoolean. As in Java there are exactly two
// instances, True and False, and equality is identity — compare with == against
// those values rather than comparing the wrapped bool.
type Boolean struct {
	object
	value bool
}

var _ Base = (*Boolean)(nil)

// The two PDF boolean values. Construction is limited to these, as it is in
// Java, where the constructor is private.
var (
	True  = &Boolean{value: true}
	False = &Boolean{value: false}
)

// Token bytes for the two values, as written to a PDF.
var (
	trueBytes  = []byte{116, 114, 117, 101}     // "true" in ISO-8859-1
	falseBytes = []byte{102, 97, 108, 115, 101} // "false" in ISO-8859-1
)

// GetBoolean returns the shared instance for the given value.
//
// Port of the two COSBoolean.getBoolean overloads; Java needs a second one for
// the boxed Boolean, which Go does not have.
func GetBoolean(value bool) *Boolean {
	if value {
		return True
	}
	return False
}

// Value returns the wrapped boolean.
func (b *Boolean) Value() bool { return b.value }

// COSObject returns the receiver.
func (b *Boolean) COSObject() Base { return b }

// Accept dispatches to the visitor.
func (b *Boolean) Accept(v Visitor) error { return v.VisitBoolean(b) }

// String returns "true" or "false".
func (b *Boolean) String() string {
	if b.value {
		return "true"
	}
	return "false"
}

// WritePDF writes the token for this value.
func (b *Boolean) WritePDF(w io.Writer) error {
	var err error
	if b.value {
		_, err = w.Write(trueBytes)
	} else {
		_, err = w.Write(falseBytes)
	}
	return err
}
