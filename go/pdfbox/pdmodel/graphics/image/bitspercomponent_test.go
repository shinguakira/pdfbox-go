package image

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// TestBitsPerComponentWhereTheDictionaryHasNone checks what BitsPerComponent
// answers by what the image dictionary holds.
//
// Java reads getInt(BITS_PER_COMPONENT, BPC), whose default is -1, and
// PDInlineImage reads its parameters the same way; the Go version of the image
// XObject answered 0. Found by the corpus comparison of images: every JPX image,
// whose /BitsPerComponent is left out for the decoder to supply and which
// neither side decodes here, read as -1 in PDFBox and 0 in the Go version.
//
// The expected values are PDFBox's, printed by getBitsPerComponent for these
// dictionaries.
func TestBitsPerComponentWhereTheDictionaryHasNone(t *testing.T) {
	image := func(set func(*cos.Stream)) *PDImageXObject {
		stream := cos.NewStream(nil)
		set(stream)
		return NewPDImageXObject(common.NewPDStream(stream), nil)
	}
	for _, c := range []struct {
		what string
		set  func(*cos.Stream)
		want int
	}{
		{"no entry", func(*cos.Stream) {}, -1},
		{"/BPC 4", func(s *cos.Stream) { s.SetInt(cos.BPC, 4) }, 4},
		{"/BitsPerComponent 2", func(s *cos.Stream) { s.SetInt(cos.BitsPerComponent, 2) }, 2},
		{"a stencil mask", func(s *cos.Stream) { s.SetBoolean(cos.ImageMask, true) }, 1},
		{"a JPX image with no data", func(s *cos.Stream) { s.SetItem(cos.Filter, cos.JPXDecode) }, -1},
	} {
		if got := image(c.set).BitsPerComponent(); got != c.want {
			t.Errorf("%s: BitsPerComponent() = %d, want PDFBox's %d", c.what, got, c.want)
		}
	}
}
