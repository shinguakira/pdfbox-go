package compress

import (
	"bytes"
	"io"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// The tokens a PDF file is built out of, and the writer for a COS string.
//
// These are COSWriter's public byte array constants and its static writeString
// in Java, where COSWriterObjectStream reaches back into COSWriter for them.
// Go forbids that: pdfwriter imports this package for CompressParameters and
// the compression pool, so this package cannot import pdfwriter. The
// definitions therefore live here, at the bottom of the dependency, and
// pdfwriter re-exports them under the Java names. There is one implementation,
// not two.
var (
	// DictOpen is the opening token of a dictionary.
	DictOpen = []byte("<<")
	// DictClose is the closing token of a dictionary.
	DictClose = []byte(">>")
	// Space is a space.
	Space = []byte{' '}
	// Comment is the comment token.
	Comment = []byte{'%'}
	// Version is the version of the PDF this writer produces.
	Version = []byte("PDF-1.4")
	// Garbage is the four high bytes of the second header line, which tell a
	// transfer program the file is binary.
	Garbage = []byte{0xf6, 0xe4, 0xfc, 0xdf}
	// EOF is the end of file token.
	EOF = []byte("%%EOF")
	// Reference is the token of an indirect reference.
	Reference = []byte("R")
	// XRef is the cross-reference table token.
	XRef = []byte("xref")
	// XRefFree marks a free cross-reference entry.
	XRefFree = []byte("f")
	// XRefUsed marks a used cross-reference entry.
	XRefUsed = []byte("n")
	// Trailer is the trailer token.
	Trailer = []byte("trailer")
	// StartXRef is the token before the cross-reference offset.
	StartXRef = []byte("startxref")
	// Obj opens an indirect object.
	Obj = []byte("obj")
	// EndObj closes one.
	EndObj = []byte("endobj")
	// ArrayOpen is the opening token of an array.
	ArrayOpen = []byte("[")
	// ArrayClose is the closing token of an array.
	ArrayClose = []byte("]")
	// Stream opens stream data.
	Stream = []byte("stream")
	// EndStream closes it.
	EndStream = []byte("endstream")
)

// WriteString writes a COS string, choosing the literal or the hexadecimal
// form.
//
// Port of the static COSWriter.writeString(COSString, OutputStream).
func WriteString(str *cos.StringObj, output io.Writer) error {
	return writeString(str.Bytes(), str.ForceHexForm(), output)
}

// WriteStringBytes writes the given bytes as a literal PDF string.
//
// Port of the static COSWriter.writeString(byte[], OutputStream).
func WriteStringBytes(b []byte, output io.Writer) error {
	return writeString(b, false, output)
}

// writeString is the private static COSWriter.writeString(byte[], boolean,
// OutputStream).
func writeString(b []byte, forceHex bool, output io.Writer) error {
	// check for non-ASCII characters
	isASCII := true
	if !forceHex {
		for _, value := range b {
			// if the byte is negative then it is an eight bit byte and is
			// outside the ASCII range. Java's byte is signed and Go's is not,
			// so the sign is taken by hand.
			if int8(value) < 0 {
				isASCII = false
				break
			}
			// PDFBOX-3107 EOL markers within a string are troublesome
			if value == 0x0d || value == 0x0a {
				isASCII = false
				break
			}
		}
	}

	var out bytes.Buffer
	if isASCII && !forceHex {
		// write ASCII string
		out.WriteByte('(')
		for _, value := range b {
			switch value {
			case '(', ')', '\\':
				out.WriteByte('\\')
				out.WriteByte(value)
			default:
				out.WriteByte(value)
			}
		}
		out.WriteByte(')')
	} else {
		// write hex string
		out.WriteByte('<')
		writeHexBytes(&out, b)
		out.WriteByte('>')
	}
	_, err := output.Write(out.Bytes())
	return err
}

// writeHexBytes is org.apache.pdfbox.util.Hex.writeHexBytes: each byte as two
// upper-case hexadecimal digits.
func writeHexBytes(out *bytes.Buffer, b []byte) {
	const digits = "0123456789ABCDEF"
	for _, value := range b {
		out.WriteByte(digits[value>>4])
		out.WriteByte(digits[value&0x0F])
	}
}
