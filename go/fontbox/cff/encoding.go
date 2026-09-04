package cff

import (
	"github.com/shinguakira/pdfbox-go/go/fontbox/encoding"
)

// CFFEncoding is a CFF Type 1-equivalent Encoding. An encoding is an array of
// codes associated with some or all glyphs in a font.
//
// Port of the abstract org.apache.fontbox.cff.CFFEncoding, which extends
// fontbox's Encoding; the port embeds that one, as it does elsewhere.
type CFFEncoding struct {
	*encoding.Encoding
}

// NewCFFEncoding returns an encoding with nothing mapped.
func NewCFFEncoding() *CFFEncoding {
	return &CFFEncoding{Encoding: encoding.NewEncoding()}
}

// Add adds a new code/SID combination to the encoding, name being the glyph
// name.
//
// Java takes the SID and ignores it, which the port keeps so that the call
// sites read the same.
func (e *CFFEncoding) Add(code, sid int, name string) {
	e.AddCharacterEncoding(code, name)
}

// addSID adds a new code/SID combination, taking the glyph name from the
// standard strings. It is Java's protected add(int, int), for use by the two
// static encodings alone.
func (e *CFFEncoding) addSID(code, sid int) {
	e.AddCharacterEncoding(code, StandardStringName(sid))
}

// newStaticEncoding builds one of the two encodings that come from a table of
// code and SID pairs.
func newStaticEncoding(table [][2]int) *CFFEncoding {
	e := NewCFFEncoding()
	for _, entry := range table {
		e.addSID(entry[0], entry[1])
	}
	return e
}

// CFFStandardEncoding is the specialized CFFEncoding used if the EncodingId of
// a font is set to 0.
//
// Port of org.apache.fontbox.cff.CFFStandardEncoding. Java exposes a singleton
// through getInstance; the port has the instance alone.
var CFFStandardEncoding = newStaticEncoding(cffStandardEncodingTable)

// CFFExpertEncoding is the specialized CFFEncoding used if the EncodingId of a
// font is set to 1.
//
// Port of org.apache.fontbox.cff.CFFExpertEncoding.
var CFFExpertEncoding = newStaticEncoding(cffExpertEncodingTable)

// StandardStringName returns the string mapped to the given SID.
//
// Port of org.apache.fontbox.cff.CFFStandardString.getName. Java indexes the
// array directly and throws ArrayIndexOutOfBoundsException past its end, which
// is unchecked; Go's own bounds check is the same failure.
func StandardStringName(sid int) string {
	return sid2str[sid]
}

// NumberOfStandardStrings is how many standard strings there are.
func NumberOfStandardStrings() int { return len(sid2str) }
