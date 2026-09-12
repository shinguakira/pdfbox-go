package filter

import (
	"bytes"
	"compress/flate"
	"compress/zlib"
	"errors"
	"io"
	"math/rand"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// Ported from pdfbox/src/test/java/org/apache/pdfbox/filter/TestFilters.java.
//
// testFilters there generates random data with a mix of pseudo-random runs and
// very predictable runs, then round-trips it through every filter that supports
// it. The port keeps that generator and applies it to the filters implemented
// so far. Its other tests need a document loader (testPDFBOX4517), the LZW
// filter (testPDFBOX1977) or the RunLength filter (testRLE), none of which are
// ported yet.

// randomFilterData reproduces the generator in the Java testFilters loop: runs
// of pseudo-random bytes interleaved with runs of a single repeated value, so
// that the data is neither incompressible nor trivially compressible.
func randomFilterData(r *rand.Rand) []byte {
	numBytes := 10000 + r.Intn(20000)
	data := make([]byte, numBytes)

	upto := 0
	for upto < numBytes {
		left := numBytes - upto
		if r.Intn(2) == 0 || left < 2 {
			// pseudo-random bytes
			end := upto + min(left, 10+r.Intn(100))
			for upto < end {
				data[upto] = byte(r.Int())
				upto++
			}
		} else {
			// very predictable bytes
			end := upto + min(left, 2+r.Intn(10))
			value := byte(r.Intn(4))
			for upto < end {
				data[upto] = value
				upto++
			}
		}
	}
	return data
}

func checkEncodeDecode(t *testing.T, f Filter, original []byte) {
	t.Helper()

	var encoded bytes.Buffer
	if err := f.Encode(&encoded, bytes.NewReader(original), cos.NewDictionary()); err != nil {
		t.Fatalf("Encode: %v", err)
	}

	var decoded bytes.Buffer
	if _, err := f.Decode(&decoded, bytes.NewReader(encoded.Bytes()), cos.NewDictionary(), 0); err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if !bytes.Equal(original, decoded.Bytes()) {
		t.Fatalf("round trip changed the data: %d bytes in, %d bytes out",
			len(original), len(decoded.Bytes()))
	}
}

func TestFlateRoundTrip(t *testing.T) {
	// Java seeds a Random with 123456 and draws ten seeds from it, then runs
	// ten more with non-deterministic seeds. A fixed seed here keeps failures
	// reproducible; the shape of the data is what matters.
	rd := rand.New(rand.NewSource(123456))
	for i := 0; i < 10; i++ {
		original := randomFilterData(rd)
		checkEncodeDecode(t, Flate{}, original)
	}
}

func TestFlateRoundTripEmpty(t *testing.T) {
	checkEncodeDecode(t, Flate{}, nil)
}

func TestIdentityRoundTrip(t *testing.T) {
	checkEncodeDecode(t, Identity{}, []byte("unchanged"))
}

// TestFlateDecodeTruncated pins the damage tolerance PDFBox relies on: it
// inflates with nowrap after skipping the two header bytes, so a stream whose
// checksum is missing or wrong still yields the bytes that did decode. Real
// PDFs are frequently truncated this way.
func TestFlateDecodeTruncated(t *testing.T) {
	original := bytes.Repeat([]byte("the quick brown fox. "), 200)

	var encoded bytes.Buffer
	if err := (Flate{}).Encode(&encoded, bytes.NewReader(original), cos.NewDictionary()); err != nil {
		t.Fatalf("Encode: %v", err)
	}

	// drop the trailing checksum and a little of the data with it
	truncated := encoded.Bytes()[:encoded.Len()-8]

	var decoded bytes.Buffer
	// An error here is acceptable — the point is that whatever decoded before
	// the truncation is still handed back rather than discarded.
	_, _ = (Flate{}).Decode(&decoded, bytes.NewReader(truncated), cos.NewDictionary(), 0)

	if decoded.Len() == 0 {
		t.Fatal("a truncated stream yielded nothing; partial data must survive")
	}
	if !bytes.HasPrefix(original, decoded.Bytes()) {
		t.Fatal("the partial output is not a prefix of the original data")
	}
}

// TestFlateDecodeWithPredictor covers the combination a cross-reference stream
// uses: Flate with PNG Up prediction.
func TestFlateDecodeWithPredictor(t *testing.T) {
	const columns = 4
	rows := [][]byte{
		{10, 20, 30, 40},
		{11, 22, 33, 44},
		{12, 24, 36, 48},
	}

	// Build the predicted form: each row prefixed with the PNG algorithm byte
	// (2 for Up) and holding the difference from the row above.
	var predicted bytes.Buffer
	prev := make([]byte, columns)
	for _, row := range rows {
		predicted.WriteByte(2) // PNG Up
		for i := range row {
			predicted.WriteByte(row[i] - prev[i])
		}
		prev = row
	}

	var encoded bytes.Buffer
	if err := (Flate{}).Encode(&encoded, bytes.NewReader(predicted.Bytes()), cos.NewDictionary()); err != nil {
		t.Fatalf("Encode: %v", err)
	}

	params := cos.NewDictionary()
	params.SetInt(cos.Predictor, 12)
	params.SetInt(cos.Colors, 1)
	params.SetInt(cos.BitsPerComponent, 8)
	params.SetInt(cos.Columns, columns)

	stream := cos.NewDictionary()
	stream.SetItem(cos.Filter, cos.FlateDecode)
	stream.SetItem(cos.DecodeParms, params)

	var decoded bytes.Buffer
	if _, err := (Flate{}).Decode(&decoded, bytes.NewReader(encoded.Bytes()), stream, 0); err != nil {
		t.Fatalf("Decode: %v", err)
	}

	var want bytes.Buffer
	for _, row := range rows {
		want.Write(row)
	}
	if !bytes.Equal(want.Bytes(), decoded.Bytes()) {
		t.Fatalf("decoded % d, want % d", decoded.Bytes(), want.Bytes())
	}
}

func TestByName(t *testing.T) {
	for _, name := range []*cos.Name{cos.FlateDecode, cos.Fl} {
		f, err := ByName(name)
		if err != nil {
			t.Fatalf("ByName(%s): %v", name.Name(), err)
		}
		if _, ok := f.(Flate); !ok {
			t.Errorf("ByName(%s) = %T, want Flate", name.Name(), f)
		}
	}

	if _, err := ByName(cos.GetPDFName("NoSuchFilter")); err == nil {
		t.Error("ByName of an unknown filter succeeded, want an error")
	}
}

// TestDecodeParamsFor covers the shapes /DecodeParms takes: a bare dictionary
// when there is one filter, and an array parallel to /Filter otherwise.
func TestDecodeParamsFor(t *testing.T) {
	t.Run("single filter with a dictionary", func(t *testing.T) {
		params := cos.NewDictionary()
		params.SetInt(cos.Predictor, 12)

		d := cos.NewDictionary()
		d.SetItem(cos.Filter, cos.FlateDecode)
		d.SetItem(cos.DecodeParms, params)

		if got := decodeParamsFor(d, 0); got != params {
			t.Errorf("decodeParamsFor = %v, want the parameter dictionary", got)
		}
	})

	t.Run("filter array with a parallel parameter array", func(t *testing.T) {
		first := cos.NewDictionary()
		first.SetInt(cos.Predictor, 1)
		second := cos.NewDictionary()
		second.SetInt(cos.Predictor, 12)

		d := cos.NewDictionary()
		d.SetItem(cos.Filter, cos.NewArrayOf([]cos.Base{cos.FlateDecode, cos.FlateDecode}))
		d.SetItem(cos.DecodeParms, cos.NewArrayOf([]cos.Base{first, second}))

		if got := decodeParamsFor(d, 1); got != second {
			t.Errorf("decodeParamsFor(1) = %v, want the second parameter dictionary", got)
		}
		if got := decodeParamsFor(d, 5); got != nil {
			t.Errorf("decodeParamsFor(5) = %v, want nil for an out-of-range index", got)
		}
	})

	t.Run("absent", func(t *testing.T) {
		if got := decodeParamsFor(cos.NewDictionary(), 0); got != nil {
			t.Errorf("decodeParamsFor = %v, want nil", got)
		}
		if got := decodeParamsFor(nil, 0); got != nil {
			t.Errorf("decodeParamsFor(nil) = %v, want nil", got)
		}
	})

	t.Run("abbreviated inline image keys", func(t *testing.T) {
		params := cos.NewDictionary()
		params.SetInt(cos.Predictor, 12)

		d := cos.NewDictionary()
		d.SetItem(cos.F, cos.Fl)
		d.SetItem(cos.DP, params)

		if got := decodeParamsFor(d, 0); got != params {
			t.Errorf("decodeParamsFor = %v, want the parameter dictionary", got)
		}
	})
}

// TestFlateDecodeCorrupt pins the damage tolerance in
// FlateFilterDecoderStream.fetch: a DataFormatException is caught and logged,
// not thrown, and whatever inflated before it is handed back. Without that a
// damaged PDF loses everything after the first bad bit rather than everything
// after the damage.
func TestFlateDecodeCorrupt(t *testing.T) {
	original := bytes.Repeat([]byte("the quick brown fox. "), 200)

	var encoded bytes.Buffer
	if err := (Flate{}).Encode(&encoded, bytes.NewReader(original), cos.NewDictionary()); err != nil {
		t.Fatalf("Encode: %v", err)
	}

	// Corrupt the middle of the compressed data, so inflate fails with a data
	// error rather than running out of input.
	corrupt := encoded.Bytes()
	corrupt[len(corrupt)/2] ^= 0xFF

	var decoded bytes.Buffer
	if _, err := (Flate{}).Decode(&decoded, bytes.NewReader(corrupt), cos.NewDictionary(), 0); err != nil {
		t.Fatalf("Decode returned %v; Java logs the corruption and returns what decoded", err)
	}
	if decoded.Len() == 0 {
		t.Fatal("a corrupt stream yielded nothing; the bytes decoded before the damage must survive")
	}
	if !bytes.HasPrefix(original, decoded.Bytes()) {
		t.Fatal("the partial output is not a prefix of the original data")
	}
}

// TestFlateDecoderReaderEndsAtDamageInsteadOfFailing pins the damage tolerance
// of FlateFilterDecoderStream on the streaming path.
//
// Java catches the DataFormatException in fetch, logs it, keeps whatever
// inflated and reports the end of the stream. compress/flate reports an error
// instead, so a port that handed its reader straight to the caller would fail
// where Java carries on -- and PDFBOX-1232, the reason PDFBox inflates raw
// rather than through a zlib reader, is exactly that real PDFs end without a
// Z_STREAM_END.
//
// A truncated stream is the damage that is worth testing. Corrupting bytes in
// the middle is not: deflate carries no check within the stream, PDFBox skips
// the Adler-32 on purpose, and both inflaters simply produce different bytes
// without noticing -- measured, over eight corruption offsets, before this test
// was written.
//
// The expected shape was read out of the running Java, FlateFilterDecoderStream
// compiled against JDK 17 over the same two inputs:
//
//	whole      decoded=408 prefixOfPlain=true lastRead=-1
//	truncated  decoded=310 prefixOfPlain=true lastRead=-1
//
// The exact truncated count belongs to Java's own buffering and its recovery of
// partly inflated bytes out of a 4096-byte array, so it is not asserted. What
// is asserted is what those numbers say: no failure, and what comes out is a
// non-empty prefix of what went in.
func TestFlateDecoderReaderEndsAtDamageInsteadOfFailing(t *testing.T) {
	plain := bytes.Repeat([]byte("BT /F1 12 Tf 100 700 Td (Hello scratch file) Tj ET\n"), 8)

	var deflated bytes.Buffer
	// zlib, so that the stream carries the two header bytes a stream out of a
	// PDF has and NewFlateDecoderReader skips
	zlibWriter := zlib.NewWriter(&deflated)
	if _, err := zlibWriter.Write(plain); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := zlibWriter.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	whole := deflated.Bytes()

	for _, row := range []struct {
		name    string
		encoded []byte
		whole   bool
	}{
		{"whole", whole, true},
		{"truncated", whole[:len(whole)-8], false},
	} {
		t.Run(row.name, func(t *testing.T) {
			decoder, err := NewFlateDecoderReader(bytes.NewReader(row.encoded))
			if err != nil {
				t.Fatalf("NewFlateDecoderReader: %v", err)
			}
			defer decoder.Close()

			got, err := io.ReadAll(decoder)
			if err != nil {
				t.Fatalf("ReadAll = %v, want no error: Java ends the stream at "+
					"the damage rather than failing", err)
			}
			if !bytes.HasPrefix(plain, got) {
				t.Errorf("decoded %d bytes that are not a prefix of the input",
					len(got))
			}
			switch {
			case row.whole && len(got) != len(plain):
				t.Errorf("decoded %d bytes of an undamaged stream, want %d",
					len(got), len(plain))
			case !row.whole && len(got) == 0:
				t.Error("decoded nothing at all; Java keeps what inflated " +
					"before the damage")
			}
		})
	}
}

// erroringReader hands back some bytes and then a real I/O failure, the way a
// disk or a network source does.
type erroringReader struct {
	head []byte
	err  error
}

func (e *erroringReader) Read(p []byte) (int, error) {
	if len(e.head) > 0 {
		n := copy(p, e.head)
		e.head = e.head[n:]
		return n, nil
	}
	return 0, e.err
}

// TestFlateDecoderReaderPassesSourceErrorsOn checks that only damaged deflate
// data ends the stream quietly.
//
// FlateFilterDecoderStream.fetch reads the source *outside* its try block and
// catches only DataFormatException, so an IOException out of the wrapped stream
// propagates and only corrupt compressed data is swallowed. compress/flate
// reports both through one error, so the port has to tell them apart: a failing
// disk must not look like the end of the page's content.
func TestFlateDecoderReaderPassesSourceErrorsOn(t *testing.T) {
	plain := bytes.Repeat([]byte("BT /F1 12 Tf 100 700 Td (Hello) Tj ET\n"), 8)
	var deflated bytes.Buffer
	zlibWriter := zlib.NewWriter(&deflated)
	if _, err := zlibWriter.Write(plain); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := zlibWriter.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	sourceErr := errors.New("the disk went away")
	// enough of the stream to get past the header and start inflating
	source := &erroringReader{head: deflated.Bytes()[:12], err: sourceErr}

	decoder, err := NewFlateDecoderReader(source)
	if err != nil {
		t.Fatalf("NewFlateDecoderReader: %v", err)
	}
	defer decoder.Close()

	if _, err := io.ReadAll(decoder); !errors.Is(err, sourceErr) {
		t.Errorf("ReadAll = %v, want the source's own error: Java lets an "+
			"IOException out of the wrapped stream propagate and swallows "+
			"only DataFormatException", err)
	}
}

// TestFlateDecoderReaderCloseSwallowsDamage pins the fourth defect the corpus
// found: qpdf/shared-images-errors.pdf and shared-images-errors-2-out.pdf, whose
// page content streams are deliberately damaged Flate. PDFBox extracts 65 and 1
// characters from them; the port failed the whole extraction with
// "flate: corrupt input before offset 5".
//
// The Read side was already right, and says so at length: Java's
// FlateFilterDecoderStream.fetch catches the DataFormatException, keeps what
// inflated and reports end of data, and the port does the same. What leaked was
// Close. compress/flate's decompressor stores the error it stopped on and hands
// it back from Close; Java's close is inflater.end() plus FilterInputStream.close
// on a RandomAccessInputStream, and neither of those can report a format
// problem. So the stream said "no more data", the caller believed it, and then
// closing raised the damage the Read had already dealt with.
func TestFlateDecoderReaderCloseSwallowsDamage(t *testing.T) {
	// two zlib header bytes, then bytes that are not a deflate stream
	damaged := []byte{0x78, 0x9c, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}

	reader, err := NewFlateDecoderReader(bytes.NewReader(damaged))
	if err != nil {
		t.Fatalf("NewFlateDecoderReader: %v", err)
	}

	// Reading is expected to end quietly: Java keeps what inflated, which here
	// is nothing, and reports end of stream rather than failing.
	if _, err := io.ReadAll(reader); err != nil {
		t.Errorf("ReadAll = %v, want nil -- damaged data ends the stream, it does not fail it", err)
	}

	if err := reader.Close(); err != nil {
		t.Errorf("Close = %v, want nil -- Java's close is inflater.end(), which cannot report damage", err)
	}
}

// failingReader yields some valid deflate data and then a real I/O failure,
// which is the case Close must not confuse with damage.
type failingReader struct {
	data []byte
	at   int
	err  error
}

func (r *failingReader) Read(p []byte) (int, error) {
	if r.at >= len(r.data) {
		return 0, r.err
	}
	n := copy(p, r.data[r.at:])
	r.at += n
	return n, nil
}

// TestFlateDecoderReaderCloseKeepsASourceFailure is the other half of
// TestFlateDecoderReaderCloseSwallowsDamage: Close stays quiet about damage a
// Read already absorbed, and about nothing else.
//
// Unlike the tests around it, this one does not fail without its change -- it
// passes against the Close that classified the error on its own too, and it is
// here because that Close was one error value away from being wrong.
// io.ErrUnexpectedEOF is both what a truncated deflate stream looks like and
// what a source that stopped early returns, so a Close that decides by looking
// at the error alone cannot tell a damaged document from a failing disk. Java
// never has to: it reads the source outside the try and catches
// DataFormatException alone. The port's equivalent is for Close to follow the
// decision Read already took rather than take its own, which is what `absorbed`
// records, and this pins the contract that arrangement exists for.
func TestFlateDecoderReaderCloseKeepsASourceFailure(t *testing.T) {
	var compressed bytes.Buffer
	compressed.Write([]byte{0x78, 0x9c}) // the two header bytes the reader skips
	writer, err := flate.NewWriter(&compressed, flate.DefaultCompression)
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	if _, err := writer.Write([]byte("a readable page")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// the source gives up part way through, the way a failing disk would
	sourceErr := errors.New("the disk went away")
	source := &failingReader{data: compressed.Bytes()[:len(compressed.Bytes())/2], err: sourceErr}

	reader, err := NewFlateDecoderReader(source)
	if err != nil {
		t.Fatalf("NewFlateDecoderReader: %v", err)
	}
	if _, err := io.ReadAll(reader); !errors.Is(err, sourceErr) {
		t.Errorf("ReadAll = %v, want the source's own error", err)
	}
	if err := reader.Close(); err == nil {
		t.Error("Close = nil, want the failure reported -- no Read absorbed anything here")
	}
}
