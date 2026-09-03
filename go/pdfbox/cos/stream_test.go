package cos_test

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/filter"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// Ported from pdfbox/src/test/java/org/apache/pdfbox/cos/TestCOSStream.java.
//
// This is an external test package because it needs both cos and filter, and
// filter imports cos. That is the same package cycle Stream itself has to work
// around; see the StreamCodec doc in stream.go.
//
// testCompressedStream2Encode and testCompressedStream2Decode chain ASCII85
// with Flate. The ASCII85 filter is not ported, so those two are not ported
// either; the two-filter chaining they exercise is covered by
// TestStreamTwoFilterChain below using Flate twice.

const streamTestInput = "This is a test string to be used as input for TestCOSStream"

func newTestStream() *cos.Stream {
	return cos.NewStream(filter.Provider{})
}

// encodeData produces the encoded form of data for one filter, the way the Java
// helper of the same name does.
func encodeData(t *testing.T, original []byte, name *cos.Name) []byte {
	t.Helper()
	f, err := filter.ByName(name)
	if err != nil {
		t.Fatalf("ByName(%s): %v", name.Name(), err)
	}
	var encoded bytes.Buffer
	if err := f.Encode(&encoded, bytes.NewReader(original), cos.NewDictionary()); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	return encoded.Bytes()
}

// createStream writes data into a new stream through the filtering writer.
func createStream(t *testing.T, data []byte, filters cos.Base) *cos.Stream {
	t.Helper()
	s := newTestStream()
	w, err := s.CreateWriterWithFilters(filters)
	if err != nil {
		t.Fatalf("CreateWriterWithFilters: %v", err)
	}
	if _, err := w.Write(data); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	return s
}

func validateEncoded(t *testing.T, s *cos.Stream, want []byte) {
	t.Helper()
	defer s.Close()
	r, err := s.CreateRawReader()
	if err != nil {
		t.Fatalf("CreateRawReader: %v", err)
	}
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !bytes.Equal(want, got) {
		t.Fatalf("encoded data does not match: %d bytes, want %d", len(got), len(want))
	}
}

func validateDecoded(t *testing.T, s *cos.Stream, want []byte) {
	t.Helper()
	defer s.Close()
	r, err := s.CreateReader()
	if err != nil {
		t.Fatalf("CreateReader: %v", err)
	}
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !bytes.Equal(want, got) {
		t.Fatalf("decoded data does not match: %q, want %q", got, want)
	}
}

func TestStreamUncompressedEncode(t *testing.T) {
	input := []byte(streamTestInput)
	validateEncoded(t, createStream(t, input, nil), input)
}

func TestStreamUncompressedDecode(t *testing.T) {
	input := []byte(streamTestInput)
	validateDecoded(t, createStream(t, input, nil), input)
}

func TestStreamCompressedEncode(t *testing.T) {
	input := []byte(streamTestInput)
	want := encodeData(t, input, cos.FlateDecode)
	validateEncoded(t, createStream(t, input, cos.FlateDecode), want)
}

func TestStreamCompressedDecode(t *testing.T) {
	input := []byte(streamTestInput)
	encoded := encodeData(t, input, cos.FlateDecode)

	s := newTestStream()
	w, err := s.CreateRawWriter()
	if err != nil {
		t.Fatalf("CreateRawWriter: %v", err)
	}
	if _, err := w.Write(encoded); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	s.SetItem(cos.Filter, cos.FlateDecode)
	validateDecoded(t, s, input)
}

// TestStreamTwoFilterChain covers a stream whose /Filter is an array, which the
// Java testCompressedStream2 tests do with ASCII85 over Flate. Flate twice
// exercises the same chaining and ordering.
func TestStreamTwoFilterChain(t *testing.T) {
	input := []byte(streamTestInput)

	filters := cos.NewArrayOf([]cos.Base{cos.FlateDecode, cos.FlateDecode})
	s := createStream(t, input, filters)

	// A reader applies the filters in array order, so a writer applies them in
	// reverse; the stored bytes are the input encoded twice.
	want := encodeData(t, encodeData(t, input, cos.FlateDecode), cos.FlateDecode)
	validateEncoded(t, s, want)

	// and it decodes back to the original
	validateDecoded(t, createStream(t, input, filters), input)
}

func TestStreamDoubleClose(t *testing.T) {
	input := []byte(streamTestInput)
	want := encodeData(t, input, cos.FlateDecode)

	s := newTestStream()
	w, err := s.CreateWriterWithFilters(cos.FlateDecode)
	if err != nil {
		t.Fatalf("CreateWriterWithFilters: %v", err)
	}
	if _, err := w.Write(input); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	// closing twice must not be a problem, and must not corrupt the data
	if err := w.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
	validateEncoded(t, s, want)
}

func TestStreamHasData(t *testing.T) {
	s := newTestStream()
	defer s.Close()

	if s.HasData() {
		t.Error("HasData() = true on an empty stream")
	}
	if _, err := s.CreateReader(); err == nil {
		t.Error("CreateReader on an empty stream succeeded, want an error")
	} else if !errors.Is(err, cos.ErrStreamNoData) {
		t.Errorf("error = %v, want ErrStreamNoData", err)
	}

	w, err := s.CreateWriter()
	if err != nil {
		t.Fatalf("CreateWriter: %v", err)
	}
	if _, err := w.Write([]byte(streamTestInput)); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if !s.HasData() {
		t.Error("HasData() = false after writing")
	}
}

func TestStreamLengthIsRecordedOnClose(t *testing.T) {
	input := []byte(streamTestInput)
	s := createStream(t, input, nil)
	defer s.Close()

	if got, _ := s.Length(); got != int64(len(input)) {
		t.Errorf("Length() = %d, want %d", got, len(input))
	}
}

func TestStreamRejectsSecondWriter(t *testing.T) {
	s := newTestStream()
	defer s.Close()

	w, err := s.CreateWriter()
	if err != nil {
		t.Fatalf("CreateWriter: %v", err)
	}
	defer w.Close()

	if _, err := s.CreateWriter(); !errors.Is(err, cos.ErrStreamWriting) {
		t.Errorf("second CreateWriter error = %v, want ErrStreamWriting", err)
	}
	if _, err := s.CreateRawReader(); !errors.Is(err, cos.ErrStreamWriting) {
		t.Errorf("read while writing error = %v, want ErrStreamWriting", err)
	}
}

// TestStreamWithoutCodecProvider pins that a filtered stream built without a
// provider fails loudly rather than handing back encoded bytes as if they were
// decoded.
func TestStreamWithoutCodecProvider(t *testing.T) {
	s := cos.NewStream(nil)
	defer s.Close()

	w, err := s.CreateRawWriter()
	if err != nil {
		t.Fatalf("CreateRawWriter: %v", err)
	}
	w.Write([]byte("data"))
	w.Close()

	s.SetItem(cos.Filter, cos.FlateDecode)
	if _, err := s.CreateReader(); !errors.Is(err, cos.ErrNoCodecProvider) {
		t.Errorf("error = %v, want ErrNoCodecProvider", err)
	}
}

func TestStreamTextString(t *testing.T) {
	s := createStream(t, []byte("hello"), cos.FlateDecode)
	defer s.Close()

	got, err := s.TextString()
	if err != nil {
		t.Fatalf("TextString: %v", err)
	}
	if got != "hello" {
		t.Errorf("TextString() = %q, want %q", got, "hello")
	}
}

func TestStreamAcceptDispatches(t *testing.T) {
	// Stream embeds Dictionary, so this checks that Accept was overridden
	// rather than inherited — embedding promotes methods but gives no virtual
	// dispatch, which is exactly the trap the conventions warn about.
	s := newTestStream()
	defer s.Close()

	v := &dispatchRecorder{}
	if err := s.Accept(v); err != nil {
		t.Fatalf("Accept: %v", err)
	}
	if v.visited != "stream" {
		t.Errorf("Accept dispatched to %q, want %q", v.visited, "stream")
	}
}

type dispatchRecorder struct{ visited string }

func (v *dispatchRecorder) rec(n string) error              { v.visited = n; return nil }
func (v *dispatchRecorder) VisitArray(*cos.Array) error     { return v.rec("array") }
func (v *dispatchRecorder) VisitBoolean(*cos.Boolean) error { return v.rec("boolean") }
func (v *dispatchRecorder) VisitDictionary(*cos.Dictionary) error {
	return v.rec("dictionary")
}
func (v *dispatchRecorder) VisitDocument(*cos.Document) error   { return v.rec("document") }
func (v *dispatchRecorder) VisitFloat(*cos.Float) error         { return v.rec("float") }
func (v *dispatchRecorder) VisitInteger(*cos.Integer) error     { return v.rec("integer") }
func (v *dispatchRecorder) VisitName(*cos.Name) error           { return v.rec("name") }
func (v *dispatchRecorder) VisitNull(*cos.Null) error           { return v.rec("null") }
func (v *dispatchRecorder) VisitObject(*cos.Object) error       { return v.rec("object") }
func (v *dispatchRecorder) VisitStream(*cos.Stream) error       { return v.rec("stream") }
func (v *dispatchRecorder) VisitStringObj(*cos.StringObj) error { return v.rec("string") }

// TestStreamLengthRejectedWhileWriting pins COSStream.getLength, which throws
// IllegalStateException while a writer is open:
//
//	"There is an open OutputStream associated with this COSStream. It must be
//	 closed before querying the length of this COSStream."
//
// The length is only recorded on close, so answering before then would hand
// back a stale value.
func TestStreamLengthRejectedWhileWriting(t *testing.T) {
	s := newTestStream()
	defer s.Close()

	// before any writer, the length is readable
	if _, err := s.Length(); err != nil {
		t.Fatalf("Length on a fresh stream: %v", err)
	}

	w, err := s.CreateWriter()
	if err != nil {
		t.Fatalf("CreateWriter: %v", err)
	}
	if _, err := w.Write([]byte(streamTestInput)); err != nil {
		t.Fatalf("Write: %v", err)
	}

	if _, err := s.Length(); !errors.Is(err, cos.ErrStreamWriting) {
		t.Errorf("Length while writing returned %v, want ErrStreamWriting", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	got, err := s.Length()
	if err != nil {
		t.Fatalf("Length after Close: %v", err)
	}
	if got != int64(len(streamTestInput)) {
		t.Errorf("Length() = %d, want %d", got, len(streamTestInput))
	}
}

// TestStreamCreateView covers createView, which the page content stream is read
// through. There is no Java test for it.
func TestStreamCreateView(t *testing.T) {
	input := []byte(streamTestInput)

	for _, c := range []struct {
		name    string
		filters cos.Base
	}{
		{"unfiltered", nil},
		{"flate", cos.FlateDecode},
	} {
		t.Run(c.name, func(t *testing.T) {
			s := createStream(t, input, c.filters)
			defer s.Close()

			view, err := s.CreateView()
			if err != nil {
				t.Fatalf("CreateView: %v", err)
			}
			defer view.Close()

			got, err := io.ReadAll(pdfio.NewReader(view))
			if err != nil {
				t.Fatalf("ReadAll: %v", err)
			}
			if !bytes.Equal(input, got) {
				t.Errorf("view holds %q, want %q", got, input)
			}

			// The view is a random access read, so it can be rewound and read
			// again — that is what it is for.
			if err := pdfio.SeekTo(view, 0); err != nil {
				t.Fatalf("SeekTo: %v", err)
			}
			again, err := io.ReadAll(pdfio.NewReader(view))
			if err != nil {
				t.Fatalf("ReadAll: %v", err)
			}
			if !bytes.Equal(input, again) {
				t.Errorf("second read holds %q, want %q", again, input)
			}
		})
	}
}

// TestStreamCreateViewOfParsedStream pins that an unfiltered stream read from a
// file is given a second view onto that file rather than copied into memory.
func TestStreamCreateViewOfParsedStream(t *testing.T) {
	input := []byte(streamTestInput)
	source := pdfio.NewReadBufferBytes(input)

	s, err := cos.NewStreamFromView(nil, source, filter.Provider{})
	if err != nil {
		t.Fatalf("NewStreamFromView: %v", err)
	}
	defer s.Close()

	view, err := s.CreateView()
	if err != nil {
		t.Fatalf("CreateView: %v", err)
	}
	got, err := io.ReadAll(pdfio.NewReader(view))
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !bytes.Equal(input, got) {
		t.Errorf("view holds %q, want %q", got, input)
	}

	// Closing the view leaves the stream readable, which is what makes it a
	// view rather than the stream's own cursor.
	if err := view.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := s.CreateView(); err != nil {
		t.Errorf("a second CreateView failed: %v", err)
	}
}
