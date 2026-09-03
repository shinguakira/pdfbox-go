// Package contentstream walks the content stream of a page or form and hands
// its operators to whatever is interested in them.
//
// Port of org.apache.pdfbox.contentstream.
package contentstream

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// PDContentStream is a content stream.
//
// Port of org.apache.pdfbox.contentstream.PDContentStream. Java's getContents
// returns an InputStream; the port leaves it out, since every caller in PDFBox
// reads through one of the random access forms and Go code that wants a reader
// wraps one with pdfio.NewReader.
type PDContentStream interface {
	// ContentsForRandomAccess returns this stream's content, or nil.
	ContentsForRandomAccess() (pdfio.RandomAccessRead, error)

	// Resources returns this stream's resources, or nil.
	Resources() *pdmodel.PDResources

	// BBox returns the bounding box of the contents, or nil.
	BBox() *common.PDRectangle

	// Matrix returns the matrix which transforms from the stream's space to
	// user space, or the identity matrix if there isn't any.
	Matrix() *util.Matrix
}

// StreamParsingContent is implemented by a content stream that can offer its
// content in a form cheaper to parse than random access, where peeking and
// rewinding are limited to a small range and seeking is not supported at all.
//
// Port of the getContentsForStreamParsing default method on PDContentStream.
// Go interfaces carry no default, so the method moves to an interface of its
// own and the default becomes ContentsForStreamParsing below.
type StreamParsingContent interface {
	ContentsForStreamParsing() (pdfio.RandomAccessRead, error)
}

// ContentsForStreamParsing returns the content of cs for a parser that only
// reads forwards, falling back to its random access content when it has nothing
// cheaper to offer.
func ContentsForStreamParsing(cs PDContentStream) (pdfio.RandomAccessRead, error) {
	if content, ok := cs.(StreamParsingContent); ok {
		return content.ContentsForStreamParsing()
	}
	return cs.ContentsForRandomAccess()
}

// A page is the content stream everything else is built on, so it is checked
// here rather than left to the first caller.
var (
	_ PDContentStream      = (*pdmodel.PDPage)(nil)
	_ StreamParsingContent = (*pdmodel.PDPage)(nil)
)
