package pdfparser

import (
	"io"
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/filter"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// Written from parseCOSStream, readUntilEndStream and validateStreamLength in
// COSParser.java, and from EndstreamFilterStream.java. The Java suite exercises
// these only through whole documents.

// TestEndstreamFilterBinary covers the trailing end-of-line handling for binary
// data: a stream's data ends before the EOL that precedes "endstream", so that
// EOL is not part of the stream.
func TestEndstreamFilterBinary(t *testing.T) {
	cases := []struct {
		name  string
		input []byte
		want  int64
	}{
		{"no trailing EOL", []byte{0x00, 0x01, 0x02}, 3},
		{"trailing LF is dropped", []byte{0x00, 0x01, '\n'}, 2},
		{"trailing CRLF is dropped", []byte{0x00, 0x01, '\r', '\n'}, 2},
		// calculateLength puts a lone CR back — the Java comment on it says
		// "if there is only a CR and no LF, write it".
		{"a trailing lone CR is kept", []byte{0x00, 0x01, '\r'}, 3},
		{"a CR by itself is kept", []byte{'\r'}, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := &endstreamFilter{}
			f.filter(c.input, 0, len(c.input))
			if got := f.calculateLength(); got != c.want {
				t.Errorf("length = %d, want %d", got, c.want)
			}
		})
	}
}

// TestEndstreamFilterPDFBOX2120 covers PDFBOX-2120: for data that looks like
// ASCII, the trailing end-of-line is kept rather than trimmed. The heuristic
// looks at the first ten bytes and comes from PDFBOX-1164.
func TestEndstreamFilterPDFBOX2120(t *testing.T) {
	ascii := []byte("hello world, this is text\n")
	f := &endstreamFilter{}
	f.filter(ascii, 0, len(ascii))
	if got := f.calculateLength(); got != int64(len(ascii)) {
		t.Errorf("length = %d, want %d — ASCII data keeps its trailing newline",
			got, len(ascii))
	}

	// the same data with a control byte in the first ten is treated as binary
	binary := []byte("hel\x01lo world, this is text\n")
	f = &endstreamFilter{}
	f.filter(binary, 0, len(binary))
	if got := f.calculateLength(); got != int64(len(binary))-1 {
		t.Errorf("length = %d, want %d — binary data drops its trailing newline",
			got, len(binary)-1)
	}
}

// TestEndstreamFilterAcrossBuffers covers a CR that lands at the end of one
// buffer and its LF at the start of the next, which is why the filter carries
// state between calls.
func TestEndstreamFilterAcrossBuffers(t *testing.T) {
	f := &endstreamFilter{}
	first := []byte{0x00, 0x01, 0x02, '\r'}
	f.filter(first, 0, len(first))
	second := []byte{'\n'}
	f.filter(second, 0, len(second))

	if got := f.calculateLength(); got != 3 {
		t.Errorf("length = %d, want 3 — the split CRLF must be dropped whole", got)
	}
}

func newStreamParser(t *testing.T, input string) (*StreamParser, *cos.Document) {
	t.Helper()
	source := pdfio.NewReadBufferBytes([]byte(input))
	p, err := NewStreamParser(source, nil, filter.Provider{})
	if err != nil {
		t.Fatalf("NewStreamParser: %v", err)
	}
	return p, p.Document()
}

func TestParseCOSStreamWithLength(t *testing.T) {
	const data = "Hello stream"
	input := "stream\n" + data + "\nendstream"

	p, _ := newStreamParser(t, input)
	dict := cos.NewDictionary()
	dict.SetInt(cos.Length, len(data))

	stream, err := p.ParseCOSStream(dict)
	if err != nil {
		t.Fatalf("ParseCOSStream: %v", err)
	}
	defer stream.Close()

	r, err := stream.CreateRawReader()
	if err != nil {
		t.Fatalf("CreateRawReader: %v", err)
	}
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(got) != data {
		t.Errorf("stream data = %q, want %q", got, data)
	}
}

// TestParseCOSStreamMissingLength covers the fallback: with no /Length the
// parser scans for the endstream keyword and records the length it found.
func TestParseCOSStreamMissingLength(t *testing.T) {
	const data = "Hello stream"
	input := "stream\n" + data + "\nendstream"

	p, _ := newStreamParser(t, input)
	dict := cos.NewDictionary()

	stream, err := p.ParseCOSStream(dict)
	if err != nil {
		t.Fatalf("ParseCOSStream: %v", err)
	}
	defer stream.Close()

	// The data is ASCII, so PDFBOX-2120 keeps the trailing newline as part of
	// the stream rather than trimming it.
	if got, _ := stream.Length(); got != int64(len(data)+1) {
		t.Errorf("recorded /Length = %d, want %d", got, len(data)+1)
	}
}

// TestParseCOSStreamWrongLength covers a /Length that does not point at the
// endstream keyword: the value is not trusted and the scan is used instead.
func TestParseCOSStreamWrongLength(t *testing.T) {
	const data = "Hello stream"
	input := "stream\n" + data + "\nendstream"

	p, _ := newStreamParser(t, input)
	dict := cos.NewDictionary()
	dict.SetInt(cos.Length, 3) // wrong

	stream, err := p.ParseCOSStream(dict)
	if err != nil {
		t.Fatalf("ParseCOSStream: %v", err)
	}
	defer stream.Close()

	// ASCII data keeps its trailing newline; see PDFBOX-2120.
	if got, _ := stream.Length(); got != int64(len(data)+1) {
		t.Errorf("recorded /Length = %d, want %d — the wrong value must be replaced",
			got, len(data)+1)
	}
}

// TestParseCOSStreamEndsWithEndobj covers the lenient path for a stream that
// ends with "endobj" instead of "endstream".
func TestParseCOSStreamEndsWithEndobj(t *testing.T) {
	const data = "Hello stream"
	input := "stream\n" + data + "\nendobj"

	p, _ := newStreamParser(t, input)
	dict := cos.NewDictionary()

	stream, err := p.ParseCOSStream(dict)
	if err != nil {
		t.Fatalf("ParseCOSStream: %v", err)
	}
	defer stream.Close()

	// The data is ASCII, so PDFBOX-2120 keeps the trailing newline as part of
	// the stream rather than trimming it.
	if got, _ := stream.Length(); got != int64(len(data)+1) {
		t.Errorf("recorded /Length = %d, want %d", got, len(data)+1)
	}
}

// TestParseCOSStreamNotLenient covers the strict path: without lenient parsing
// a stream that does not end with endstream is an error.
func TestParseCOSStreamNotLenient(t *testing.T) {
	input := "stream\ndata\nsomethingelse"

	p, _ := newStreamParser(t, input)
	p.SetLenient(false)

	if _, err := p.ParseCOSStream(cos.NewDictionary()); err == nil {
		t.Error("a stream with no endstream keyword parsed without error in strict mode")
	}
}

// TestReadUntilEndStreamLargeData exercises the buffered scan across more than
// one buffer, where a keyword can straddle the boundary.
func TestReadUntilEndStreamLargeData(t *testing.T) {
	data := strings.Repeat("0123456789", 1000) // 10000 bytes, several buffers
	input := "stream\n" + data + "\nendstream"

	p, _ := newStreamParser(t, input)
	stream, err := p.ParseCOSStream(cos.NewDictionary())
	if err != nil {
		t.Fatalf("ParseCOSStream: %v", err)
	}
	defer stream.Close()

	// The data is ASCII, so PDFBOX-2120 keeps the trailing newline as part of
	// the stream rather than trimming it.
	if got, _ := stream.Length(); got != int64(len(data)+1) {
		t.Errorf("recorded /Length = %d, want %d", got, len(data)+1)
	}

	r, _ := stream.CreateRawReader()
	got, _ := io.ReadAll(r)
	if string(got) != data+"\n" {
		t.Errorf("stream data does not round-trip: got %d bytes, want %d", len(got), len(data)+1)
	}
}

func TestGetStreamLength(t *testing.T) {
	p, doc := newStreamParser(t, "")

	// a direct number
	if got, err := p.streamLength(cos.GetInteger(42)); err != nil || got == nil || got.LongValue() != 42 {
		t.Errorf("direct length = %v, %v; want 42", got, err)
	}
	// absent
	if got, err := p.streamLength(nil); err != nil || got != nil {
		t.Errorf("absent length = %v, %v; want nil, nil", got, err)
	}
	// through an indirect reference
	key, _ := cos.NewObjectKey(7, 0)
	ref := doc.ObjectFromPool(key)
	ref.SetToNull()
	if got, err := p.streamLength(ref); err != nil || got != nil {
		t.Errorf("length through a null reference = %v, %v; want nil, nil", got, err)
	}
	// a wrong type is an error
	if _, err := p.streamLength(cos.GetPDFName("NotANumber")); err == nil {
		t.Error("a name used as a stream length was accepted")
	}
}
