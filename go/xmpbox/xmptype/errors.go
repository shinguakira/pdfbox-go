package xmptype

// The exception the type system throws for a field it cannot make sense of.
//
// Port of org.apache.xmpbox.type.BadFieldValueException.

import "errors"

// ErrBadFieldValue reports a field whose value or type the XMP type system
// cannot accept.
//
// Port of BadFieldValueException, which Java declares as a checked Exception;
// the port makes it a sentinel every place that threw one wraps.
var ErrBadFieldValue = errors.New("xmptype: bad field value")

// errBothNamespacesNil is the IllegalArgumentException a structured type raises
// when it has neither a @StructuredType annotation nor a namespace argument.
var errBothNamespacesNil = errors.New(
	"Both StructuredType annotation and namespace parameter cannot be null")
