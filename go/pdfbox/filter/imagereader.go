package filter

import (
	"errors"
	"fmt"
	"io"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// The two filters PDFBox declares but cannot decode on its own, because each
// needs an ImageIO plugin that is not part of the Java runtime: JBIG2 needs
// jbig2-imageio and JPX needs the JAI Image I/O Tools. Where the plugin is
// absent, Filter.findImageReader throws MissingImageReaderException and neither
// filter decodes anything.
//
// Go has no such plugin either -- there is no JBIG2 or JPEG 2000 decoder in the
// standard library, and neither format has PDFBox code to port, since PDFBox
// hands both to the plugin. So the port declares them and reports the same
// thing the Java reports on a build without the jars. A document that uses
// either still opens; only that image is missing, which is what happens in Java
// too. See migration/STATUS.md.

// ErrMissingImageReader is returned when a required image reader is missing.
//
// Port of org.apache.pdfbox.filter.MissingImageReaderException, which extends
// IOException, so callers that catch IOException catch it. Callers that single
// it out use errors.Is.
var ErrMissingImageReader = errors.New("filter: missing image reader")

// missingImageReader builds the message Filter.findImageReader raises.
func missingImageReader(formatName, errorCause string) error {
	return fmt.Errorf("%w: Cannot read %s image: %s", ErrMissingImageReader,
		formatName, errorCause)
}

// JBIG2 is the JBIG2Decode filter.
//
// Port of org.apache.pdfbox.filter.JBIG2Filter, whose decode starts by asking
// for a "JBIG2" ImageReader.
type JBIG2 struct{}

var _ Filter = JBIG2{}

// Decode reports that no JBIG2 reader is installed, which is what Java's
// findImageReader does first thing on a build without jbig2-imageio.
func (JBIG2) Decode(w io.Writer, r io.Reader, parameters *cos.Dictionary,
	index int) (DecodeResult, error) {
	return DecodeResult{Parameters: parameters},
		missingImageReader("JBIG2", "jbig2-imageio is not installed")
}

// Encode panics, where Java throws UnsupportedOperationException.
func (JBIG2) Encode(w io.Writer, r io.Reader, parameters *cos.Dictionary) error {
	panic("JBIG2 encoding not implemented")
}

// JPX is the JPXDecode filter, JPEG 2000.
//
// Port of org.apache.pdfbox.filter.JPXFilter, whose decode starts by asking for
// a "JPEG2000" ImageReader.
type JPX struct{}

var _ Filter = JPX{}

// Decode reports that no JPEG 2000 reader is installed, which is what Java's
// findImageReader does first thing on a build without the JAI tools.
func (JPX) Decode(w io.Writer, r io.Reader, parameters *cos.Dictionary,
	index int) (DecodeResult, error) {
	return DecodeResult{Parameters: parameters}, missingImageReader("JPEG2000",
		"Java Advanced Imaging (JAI) Image I/O Tools are not installed")
}

// Encode panics, where Java throws UnsupportedOperationException.
func (JPX) Encode(w io.Writer, r io.Reader, parameters *cos.Dictionary) error {
	panic("JPX encoding not implemented")
}
