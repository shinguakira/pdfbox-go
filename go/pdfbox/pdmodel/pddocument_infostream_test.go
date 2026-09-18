package pdmodel

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// TestDocumentInformationInAStream checks a trailer whose /Info is a stream.
//
// Java's getDocumentInformation asks the trailer for getCOSDictionary(INFO),
// and a COSStream is a COSDictionary, so the stream's dictionary is the
// information; only where there is no dictionary at all does Java put an empty
// one in the trailer. The Go version asked for a *cos.Dictionary, did not find
// one in a *cos.Stream, and replaced the stream with an empty dictionary. Found
// by the corpus comparison of document information: pdfjs/issue18765.pdf's
// /Info is its XMP stream, whose /Length, /Subtype and /Type PDFBox reads as the
// information.
//
// The expected values are PDFBox's: getTitle answers the stream's /Title, and
// the trailer's /Info is still the stream.
func TestDocumentInformationInAStream(t *testing.T) {
	doc := NewPDDocument()
	stream := cos.NewStream(nil)
	stream.SetString(cos.Title, "a title in a stream")
	doc.Document().Trailer().SetItem(cos.Info, stream)

	info := doc.DocumentInformation()
	if got, want := info.Title(), "a title in a stream"; got != want {
		t.Errorf("Title() = %q, want %q", got, want)
	}
	if got := doc.Document().Trailer().GetDictionaryObject(cos.Info); got != cos.Base(stream) {
		t.Errorf("the trailer's /Info is %T, want the stream it was", got)
	}
	if got := info.COSObject(); got != cos.Base(&stream.Dictionary) {
		t.Errorf("the information is %T, want the stream's dictionary", got)
	}
}
