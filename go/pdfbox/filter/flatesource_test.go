package filter

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// TestFlateLetsAFailingSourceOut pins what FlateFilter.decode and
// FlateFilterDecoderStream do when the stream they read from fails, as against
// when it ends or its data is damaged.
//
// FlateFilterDecoderStream reads its two header bytes in its constructor and
// the compressed data in fetch, both outside the try block, which catches
// DataFormatException alone. So an IOException from the source comes out --
// of the constructor, of read, and through transferTo out of decode -- while a
// source that simply ends, or data that will not inflate, ends the output
// quietly. The port's Decode ended the output at every error, and both it and
// NewFlateDecoderReader treated a header they could not read, for whatever
// reason, as a stream with nothing in it.
//
// What the running PDFBox answered, over 10,690 bytes deflated to 945, from a
// source that hands out a number of bytes and then fails or ends:
//
//	fails before the header       decode threw boom; constructor threw boom
//	fails after one header byte   decode threw boom; constructor threw boom
//	fails part way                decode threw boom; read threw boom
//	ends part way                 decode returned;   read ended
//	ends after one header byte    decode returned;   read ended
//	whole                         decode returned 10,690 bytes
//
// How many bytes a truncated stream yields is each inflater's own business and
// is not asserted; see TestFlateDecoderReaderEndsAtDamageInsteadOfFailing.
func TestFlateLetsAFailingSourceOut(t *testing.T) {
	var text strings.Builder
	for i := 0; i < 400; i++ {
		fmt.Fprintf(&text, "line %d of a flate stream\n", i)
	}
	plain := []byte(text.String())
	deflated := zlibCompress(t, plain)
	boom := errors.New("boom")

	for _, c := range []struct {
		name  string
		given int
		fails bool
	}{
		{"fails before the header", 0, true},
		{"fails after one header byte", 1, true},
		{"fails part way", 200, true},
		{"ends part way", 200, false},
		{"ends after one header byte", 1, false},
		{"whole", len(deflated), false},
	} {
		t.Run(c.name, func(t *testing.T) {
			source := func() io.Reader {
				if c.fails {
					return &erroringReader{head: deflated[:c.given], err: boom}
				}
				return bytes.NewReader(deflated[:c.given])
			}

			var out bytes.Buffer
			_, err := (Flate{}).Decode(&out, source(), cos.NewDictionary(), 0)
			switch {
			case c.fails && !errors.Is(err, boom):
				t.Errorf("Decode = %v, want the source's error", err)
			case !c.fails && err != nil:
				t.Errorf("Decode = %v, want the output ended quietly", err)
			case !c.fails && !bytes.HasPrefix(plain, out.Bytes()):
				t.Errorf("Decode wrote %d bytes that are not a prefix of the data", out.Len())
			case c.given == len(deflated) && !bytes.Equal(out.Bytes(), plain):
				t.Errorf("Decode of the whole stream wrote %d bytes, want %d", out.Len(), len(plain))
			}

			decoder, err := NewFlateDecoderReader(source())
			if err != nil {
				if !c.fails || !errors.Is(err, boom) {
					t.Errorf("NewFlateDecoderReader = %v", err)
				}
				if c.given >= 2 {
					t.Errorf("NewFlateDecoderReader failed with the header read: %v", err)
				}
				return
			}
			if c.fails && c.given < 2 {
				t.Errorf("NewFlateDecoderReader read no header and did not fail; Java's constructor throws")
			}
			read, err := io.ReadAll(decoder)
			switch {
			case c.fails && !errors.Is(err, boom):
				t.Errorf("reading = %v, want the source's error", err)
			case !c.fails && err != nil:
				t.Errorf("reading = %v, want the stream ended quietly", err)
			case !c.fails && !bytes.HasPrefix(plain, read):
				t.Errorf("read %d bytes that are not a prefix of the data", len(read))
			}
			decoder.Close()
		})
	}
}
