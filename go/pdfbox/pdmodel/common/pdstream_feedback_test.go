package common_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/filter"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// What the slice 6 feedback asked, written down as a test.
//
// PDStream.createInputStream(List<String>) builds the filters up to the stop
// name and hands them to the static Filter.decode, which reduces a repeated
// filter to one before it applies any -- a stream whose /Filter array names the
// same filter twice is a malformed one PDFBox repairs rather than refuses. The
// port decoded every entry, so such a stream came back over-decoded.

func TestCreateInputStreamStoppingReducesDuplicates(t *testing.T) {
	original := []byte("the quick brown fox jumps over the lazy dog")

	// a stream whose data is deflated once but whose /Filter names Fl twice,
	// which is the shape Filter.decode repairs
	var deflated bytes.Buffer
	if err := (filter.Flate{}).Encode(&deflated, bytes.NewReader(original),
		cos.NewDictionary()); err != nil {
		t.Fatalf("Encode: %v", err)
	}

	stream := cos.NewStream(filter.Provider{})
	filters := cos.NewArray()
	filters.Add(cos.FlateDecode)
	filters.Add(cos.FlateDecode)
	stream.SetItem(cos.Filter, filters)
	raw, err := stream.CreateRawWriter()
	if err != nil {
		t.Fatalf("CreateRawWriter: %v", err)
	}
	if _, err := raw.Write(deflated.Bytes()); err != nil {
		t.Fatalf("writing: %v", err)
	}
	if err := raw.Close(); err != nil {
		t.Fatalf("closing: %v", err)
	}

	// stopping at a filter that is not there decodes the whole chain, which
	// after the reduction is one Flate
	reader, err := common.NewPDStream(stream).CreateInputStreamStopping(
		[]string{"DCTDecode"})
	if err != nil {
		t.Fatalf("CreateInputStreamStopping: %v", err)
	}
	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("reading: %v", err)
	}
	if !bytes.Equal(got, original) {
		t.Errorf("the duplicated filter gave %q, want the data decoded once", got)
	}
}

// TestCreateInputStreamStoppingStops checks the thing the method is for: the
// data comes back still encoded through the named filter.
func TestCreateInputStreamStoppingStops(t *testing.T) {
	original := []byte("image data")

	var deflated bytes.Buffer
	if err := (filter.Flate{}).Encode(&deflated, bytes.NewReader(original),
		cos.NewDictionary()); err != nil {
		t.Fatalf("Encode: %v", err)
	}

	stream := cos.NewStream(filter.Provider{})
	stream.SetItem(cos.Filter, cos.FlateDecode)
	raw, err := stream.CreateRawWriter()
	if err != nil {
		t.Fatalf("CreateRawWriter: %v", err)
	}
	if _, err := raw.Write(deflated.Bytes()); err != nil {
		t.Fatalf("writing: %v", err)
	}
	if err := raw.Close(); err != nil {
		t.Fatalf("closing: %v", err)
	}

	reader, err := common.NewPDStream(stream).CreateInputStreamStopping(
		[]string{"FlateDecode"})
	if err != nil {
		t.Fatalf("CreateInputStreamStopping: %v", err)
	}
	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("reading: %v", err)
	}
	if !bytes.Equal(got, deflated.Bytes()) {
		t.Error("stopping at the only filter should give the data as stored")
	}
}
