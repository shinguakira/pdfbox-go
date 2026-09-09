package pdmodel

// JAVA-BUGS 63: `PDPage.getContentsForStreamParsing` takes a fast path for a
// content stream whose only filter is FlateDecode, and that path is an
// inflater with no predictor -- so a stream that declares one is decoded
// wrongly here and correctly by the general path.

import (
	"bytes"
	"compress/zlib"
	"io"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/filter"
)

// predictoredContentPage is a page whose content stream is
// `/Filter /FlateDecode /DecodeParms <</Predictor 12 /Columns 4>>`, which is
// legal and which no PDF in the corpus happens to have.
//
// The raw bytes are one PNG row of four columns whose filter type is 0
// (None), so the predictor's own arithmetic is the identity and the decoded
// content is exactly the four operator bytes -- the point being which bytes
// come back, not how clever the predictor is.
func predictoredContentPage(t *testing.T) (*PDPage, []byte) {
	t.Helper()
	const content = "q Q\n"
	row := append([]byte{0}, content...)

	var deflated bytes.Buffer
	writer := zlib.NewWriter(&deflated)
	if _, err := writer.Write(row); err != nil {
		t.Fatalf("deflating the row: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("closing the deflater: %v", err)
	}

	stream := cos.NewStream(filter.Provider{})
	stream.SetFilters(cos.FlateDecode)
	parms := cos.NewDictionary()
	parms.SetInt(cos.Predictor, 12)
	parms.SetInt(cos.Columns, 4)
	stream.SetItem(cos.DecodeParms, parms)

	raw, err := stream.CreateRawWriter()
	if err != nil {
		t.Fatalf("CreateRawWriter: %v", err)
	}
	if _, err := raw.Write(deflated.Bytes()); err != nil {
		t.Fatalf("writing the raw stream: %v", err)
	}
	if err := raw.Close(); err != nil {
		t.Fatalf("closing the raw stream: %v", err)
	}

	page := NewPDPage()
	page.Dictionary().SetItem(cos.Contents, stream)
	return page, []byte(content)
}

// TestStreamParsingHonoursThePredictor is the defect.
//
// The expected value is what the general path answers, which is the whole
// filter chain: the same page must parse the same whichever accessor the
// caller reaches for. Java's fast path hands back the PNG-filtered bytes,
// which begin with the row's filter-type byte and are not content stream
// operators.
func TestStreamParsingHonoursThePredictor(t *testing.T) {
	page, want := predictoredContentPage(t)

	source, err := page.ContentsForStreamParsing()
	if err != nil {
		t.Fatalf("ContentsForStreamParsing: %v", err)
	}
	got, err := io.ReadAll(source)
	if err != nil {
		t.Fatalf("reading the content: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("ContentsForStreamParsing gave %q, want %q", got, want)
	}

	// and it is what the general path gives, which is the point
	general, err := page.ContentsForRandomAccess()
	if err != nil {
		t.Fatalf("ContentsForRandomAccess: %v", err)
	}
	alsoGot, err := io.ReadAll(general)
	if err != nil {
		t.Fatalf("reading the content: %v", err)
	}
	if !bytes.Equal(got, alsoGot) {
		t.Errorf("the two accessors disagree: %q and %q", got, alsoGot)
	}
}

// TestStreamParsingStillTakesTheFastPath keeps the stream the guard must not
// turn away: flate with no decode parameters at all.
func TestStreamParsingStillTakesTheFastPath(t *testing.T) {
	const content = "q Q\n"
	stream := cos.NewStream(filter.Provider{})
	writer, err := stream.CreateWriterWithFilters(cos.FlateDecode)
	if err != nil {
		t.Fatalf("CreateWriterWithFilters: %v", err)
	}
	if _, err := writer.Write([]byte(content)); err != nil {
		t.Fatalf("writing the stream: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("closing the stream: %v", err)
	}

	page := NewPDPage()
	page.Dictionary().SetItem(cos.Contents, stream)

	source, err := page.ContentsForStreamParsing()
	if err != nil {
		t.Fatalf("ContentsForStreamParsing: %v", err)
	}
	got, err := io.ReadAll(source)
	if err != nil {
		t.Fatalf("reading the content: %v", err)
	}
	if string(got) != content {
		t.Errorf("ContentsForStreamParsing gave %q, want %q", got, content)
	}
}
