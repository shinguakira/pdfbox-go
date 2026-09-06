package xml

// What parsing and serialising report when they fail.
//
// Port of XmpParsingException and XmpSerializationException, which Java
// declares as checked Exceptions.

import (
	"errors"
	"fmt"
)

// ErrorType says what kind of problem parsing ran into.
//
// Port of XmpParsingException.ErrorType.
type ErrorType int

// The kinds of problem parsing reports.
const (
	// Undefined is anything the other kinds do not name.
	Undefined ErrorType = iota
	// Configuration is a parser that could not be built.
	Configuration
	// XpacketBadStart is a packet that does not open as it should.
	XpacketBadStart
	// XpacketBadEnd is a packet that does not close as it should.
	XpacketBadEnd
	// NoRootElement is a packet with no root element.
	NoRootElement
	// NoSchema is an undefined schema.
	NoSchema
	// InvalidPdfaSchema is a PDF/A extension schema that is not well formed.
	InvalidPdfaSchema
	// NoType is an undefined type.
	NoType
	// InvalidType is a type that cannot hold what was written.
	InvalidType
	// Format is something weird in the serialized document.
	Format
	// NoValueType is a value type an extension schema does not define.
	NoValueType
	// RequiredProperty is a property an extension schema leaves out.
	RequiredProperty
	// InvalidPrefix is an unexpected namespace prefix.
	InvalidPrefix
)

// errorTypeNames is what ErrorType.name() answers.
var errorTypeNames = [...]string{
	Undefined:         "Undefined",
	Configuration:     "Configuration",
	XpacketBadStart:   "XpacketBadStart",
	XpacketBadEnd:     "XpacketBadEnd",
	NoRootElement:     "NoRootElement",
	NoSchema:          "NoSchema",
	InvalidPdfaSchema: "InvalidPdfaSchema",
	NoType:            "NoType",
	InvalidType:       "InvalidType",
	Format:            "Format",
	NoValueType:       "NoValueType",
	RequiredProperty:  "RequiredProperty",
	InvalidPrefix:     "InvalidPrefix",
}

// String returns the name of the kind.
func (e ErrorType) String() string {
	if int(e) < 0 || int(e) >= len(errorTypeNames) {
		return "ErrorType(" + fmt.Sprint(int(e)) + ")"
	}
	return errorTypeNames[e]
}

// ErrXmpParsing reports a packet that could not be parsed.
//
// Port of XmpParsingException. A caller that wants the kind of the problem
// reaches it with errors.As and XmpParsingError.ErrorType.
var ErrXmpParsing = errors.New("xml: xmp parsing")

// XmpParsingError is what parsing returns when it fails.
type XmpParsingError struct {
	// ErrorType says what kind of problem this is.
	ErrorType ErrorType

	// Message describes the problem.
	Message string

	// Cause is what this problem came of, and nil where it came of nothing.
	Cause error
}

// NewXmpParsingError returns a parsing error of the given kind.
//
// Port of XmpParsingException(ErrorType, String).
func NewXmpParsingError(errorType ErrorType, message string) *XmpParsingError {
	return &XmpParsingError{ErrorType: errorType, Message: message}
}

// NewXmpParsingErrorCause returns a parsing error of the given kind that came
// of another.
//
// Port of XmpParsingException(ErrorType, String, Throwable).
func NewXmpParsingErrorCause(errorType ErrorType, message string,
	cause error) *XmpParsingError {
	return &XmpParsingError{ErrorType: errorType, Message: message, Cause: cause}
}

// Error returns the message.
func (e *XmpParsingError) Error() string { return e.Message }

// Unwrap returns what this problem came of.
func (e *XmpParsingError) Unwrap() error { return e.Cause }

// Is reports whether this is a parsing error, so that errors.Is finds
// ErrXmpParsing through it.
func (e *XmpParsingError) Is(target error) bool { return target == ErrXmpParsing }

// ErrXmpSerialization reports a packet that could not be written.
//
// Port of XmpSerializationException.
var ErrXmpSerialization = errors.New("xml: xmp serialization")
