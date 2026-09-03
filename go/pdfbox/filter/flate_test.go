package filter

import (
	"bytes"
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
