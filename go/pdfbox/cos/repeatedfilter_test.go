package cos_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// TestRepeatedFilterIsDecodedOnceOnEveryPath pins what the static Filter.decode
// does with a /Filter array that names a filter more than once: it keeps the
// first of each and drops the rest, logging "Removed duplicated filter entries",
// before it applies any.
//
// createInputStream, createView and PDStream.createInputStream(List<String>)
// all go through Filter.decode, so all three decode such a stream the same way.
// The port reduced the list on the last of the three only, and the reader and
// the view decoded every entry.
//
// "The same filter" is the same FilterFactory instance, which a name and its
// abbreviation share, and the repeat need not be next to the first. The
// reduction also moves the index each filter is decoded with, and /DecodeParms
// is read at that index: a Flate that is third in the array and second after
// the reduction takes the second parameters. Every wanted value was printed by
// the running PDFBox, from all three paths, for the same stream.
func TestRepeatedFilterIsDecodedOnceOnEveryPath(t *testing.T) {
	hi := []byte("Hi")
	hexOfHi := encodeData(t, hi, cos.ASCIIHexDecode)
	rows := []byte{1, 2, 3, 4}
	predictor := cos.NewDictionary()
	predictor.SetInt(cos.Predictor, 2)
	predictor.SetInt(cos.Columns, 2)

	for _, c := range []struct {
		name    string
		raw     []byte
		filters []cos.Base
		parms   []cos.Base
		want    []byte
	}{
		{
			name:    "the same name twice",
			raw:     encodeData(t, hexOfHi, cos.ASCIIHexDecode),
			filters: []cos.Base{cos.ASCIIHexDecode, cos.ASCIIHexDecode},
			want:    []byte("4869"),
		},
		{
			name:    "a name and its abbreviation",
			raw:     encodeData(t, hexOfHi, cos.ASCIIHexDecode),
			filters: []cos.Base{cos.ASCIIHexDecode, cos.AHx},
			want:    []byte("4869"),
		},
		{
			name:    "not next to each other",
			raw:     encodeData(t, encodeData(t, hexOfHi, cos.FlateDecode), cos.ASCIIHexDecode),
			filters: []cos.Base{cos.AHx, cos.Fl, cos.AHx},
			want:    []byte("4869"),
		},
		{
			name:    "parameters at the index the array gives are not read",
			raw:     encodeData(t, encodeData(t, rows, cos.FlateDecode), cos.ASCIIHexDecode),
			filters: []cos.Base{cos.AHx, cos.AHx, cos.Fl},
			parms:   []cos.Base{cos.NullObject, cos.NullObject, predictor},
			want:    []byte{1, 2, 3, 4},
		},
		{
			name:    "parameters at the index the reduction gives are",
			raw:     encodeData(t, encodeData(t, rows, cos.FlateDecode), cos.ASCIIHexDecode),
			filters: []cos.Base{cos.AHx, cos.AHx, cos.Fl},
			parms:   []cos.Base{cos.NullObject, predictor},
			want:    []byte{1, 3, 3, 7},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			newStream := func() *cos.Stream {
				s := newTestStream()
				s.SetItem(cos.Filter, cos.NewArrayOf(c.filters))
				if c.parms != nil {
					s.SetItem(cos.DecodeParms, cos.NewArrayOf(c.parms))
				}
				w, err := s.CreateRawWriter()
				if err != nil {
					t.Fatalf("CreateRawWriter: %v", err)
				}
				if _, err := w.Write(c.raw); err != nil {
					t.Fatalf("Write: %v", err)
				}
				if err := w.Close(); err != nil {
					t.Fatalf("Close: %v", err)
				}
				return s
			}

			reader, err := newStream().CreateReader()
			if err != nil {
				t.Fatalf("CreateReader: %v", err)
			}
			if got, err := io.ReadAll(reader); err != nil || !bytes.Equal(got, c.want) {
				t.Errorf("CreateReader = %q, %v; want %q", got, err, c.want)
			}

			view, err := newStream().CreateView()
			if err != nil {
				t.Fatalf("CreateView: %v", err)
			}
			if got, err := io.ReadAll(pdfio.NewReader(view)); err != nil || !bytes.Equal(got, c.want) {
				t.Errorf("CreateView = %q, %v; want %q", got, err, c.want)
			}

			stopping, err := newStream().CreateReaderStopping(len(c.filters))
			if err != nil {
				t.Fatalf("CreateReaderStopping: %v", err)
			}
			if got, err := io.ReadAll(stopping); err != nil || !bytes.Equal(got, c.want) {
				t.Errorf("CreateReaderStopping = %q, %v; want %q", got, err, c.want)
			}
		})
	}
}

// TestRepeatedFilterIsWrittenEveryTime pins the other half: the writer does not
// reduce. COSOutputStream encodes through every entry of the array, so a stream
// written through [/FlateDecode /FlateDecode] holds its data deflated twice,
// and reading it back answers the data deflated once. Both were printed by the
// running PDFBox.
func TestRepeatedFilterIsWrittenEveryTime(t *testing.T) {
	input := []byte(streamTestInput)
	filters := cos.NewArrayOf([]cos.Base{cos.FlateDecode, cos.FlateDecode})

	validateEncoded(t, createStream(t, input, filters),
		encodeData(t, encodeData(t, input, cos.FlateDecode), cos.FlateDecode))
	validateDecoded(t, createStream(t, input, filters), encodeData(t, input, cos.FlateDecode))
}
