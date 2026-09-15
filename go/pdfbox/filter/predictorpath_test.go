package filter

import (
	"bytes"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// The predictor the way a PDF reaches it: through FlateDecode or LZWDecode,
// with /DecodeParms read out of the stream dictionary.
//
// Java has no test at this level -- PredictorTest covers getBitSeq and
// calcSetBitSeq and nothing else -- so every expected value below is PDFBox's
// own output. Each was taken by running FlateFilter.decode or LZWFilter.decode,
// compiled from this repository's Java tree the way
// migration/scripts/run-oracle.ps1 compiles it, over the same raw bytes and the
// same parameters. None was worked out from the Go.
//
// The cases are where a reading of the port against Predictor.java found the
// two answering differently:
//
//   - wrapPredictor clamps /Colors to 32 before anything else sees it.
//   - Every product is Java's 32-bit int, bytes per pixel as well as row length.
//   - The last row may be short. PredictorOutputStream.flush fills it with zeros
//     and decodes it whole.
//   - A PNG row's algorithm byte is a signed Java byte plus 10, so 0xF8 is
//     predictor 2.
//   - Only a predictor above 1 is applied; 0 and below pass the data through.
//   - A row length of zero is not an error, and parameters that are each
//     nonsense can still multiply to a row PDFBox decodes.
//   - A negative bytes per pixel throws ArrayIndexOutOfBoundsException, which
//     the port answers as errPredictorIndex.
//   - For a bit depth that does not divide 8, Java reads the bits of a component
//     that crosses a byte boundary through the sign extension of that byte.
//
// The one case PDFBox has no answer for, a TIFF predictor with a zero row
// length, is JAVA-BUGS 88 and has its own test.
func TestPredictorAnswersWhatPDFBoxAnswers(t *testing.T) {
	type params = map[*cos.Name]int
	cases := []struct {
		name    string
		filter  Filter
		params  params
		raw     string // hex
		want    string // hex, PDFBox's output
		wantErr error  // where PDFBox throws
	}{
		{
			// qpdf/issue-1688a.pdf declares these. /Colors becomes 32, so rows
			// are 32 bytes, and the eight bytes past the first row are the start
			// of a second one that flush completes with zeros.
			name:   "the qpdf issue-1688a parameters",
			filter: Flate{},
			params: params{cos.Predictor: 2, cos.Colors: 536870913, cos.BitsPerComponent: 8, cos.Columns: 1},
			raw:    "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728",
			want: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20" +
				"2122232425262728000000000000000000000000000000000000000000000000",
		},
		{
			name:   "clamped colors are the pixel TIFF Sub subtracts",
			filter: Flate{},
			params: params{cos.Predictor: 2, cos.Colors: 536870913, cos.BitsPerComponent: 8, cos.Columns: 2},
			raw: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20" +
				"2122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f40",
			want: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20" +
				"22242628 2a2c2e30 32343638 3a3c3e40 42444648 4a4c4e50 52545658 5a5c5e60",
		},
		{
			name:   "33 colors are 32",
			filter: Flate{},
			params: params{cos.Predictor: 2, cos.Colors: 33, cos.BitsPerComponent: 8, cos.Columns: 2},
			raw: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20" +
				"2122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f40" +
				"4142",
			want: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20" +
				"22242628 2a2c2e30 32343638 3a3c3e40 42444648 4a4c4e50 52545658 5a5c5e60" +
				"4142000000000000000000000000000000000000000000000000000000000000" +
				"4142000000000000000000000000000000000000000000000000000000000000",
		},
		{
			name:   "clamped colors are the pixel PNG Sub subtracts",
			filter: Flate{},
			params: params{cos.Predictor: 11, cos.Colors: 536870913, cos.BitsPerComponent: 8, cos.Columns: 2},
			raw: "01" +
				"0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20" +
				"2122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f40",
			want: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20" +
				"22242628 2a2c2e30 32343638 3a3c3e40 42444648 4a4c4e50 52545658 5a5c5e60",
		},
		{
			// 4 * 1073741828 is 16 in 32 bits, so a pixel is two bytes; and
			// 3 * 4 * 1073741828 is 48, so a row is six.
			name:   "bytes per pixel wraps as the row length does",
			filter: Flate{},
			params: params{cos.Predictor: 11, cos.Colors: 4, cos.BitsPerComponent: 1073741828, cos.Columns: 3},
			raw:    "01010203040506" + "010a0b0c0d0e0f",
			want:   "01020406090c" + "0a0b16182427",
		},
		{
			name:   "a short last row is completed with zeros",
			filter: Flate{},
			params: params{cos.Predictor: 12, cos.Columns: 4},
			raw:    "0201020304" + "0201010101" + "020506",
			want:   "01020304" + "02030405" + "07090405",
		},
		{
			name:   "a short last row is completed with zeros through LZW as well",
			filter: LZW{},
			params: params{cos.Predictor: 12, cos.Columns: 4},
			raw:    "0201020304" + "020506",
			want:   "01020304" + "06080304",
		},
		{
			name:   "an algorithm byte with no row after it adds nothing",
			filter: Flate{},
			params: params{cos.Predictor: 12, cos.Columns: 2},
			raw:    "020102" + "02",
			want:   "0102",
		},
		{
			name:   "predictor 0 passes the data through",
			filter: Flate{},
			params: params{cos.Predictor: 0, cos.Columns: 0},
			raw:    "010203",
			want:   "010203",
		},
		{
			name:   "a negative predictor passes the data through",
			filter: Flate{},
			params: params{cos.Predictor: -3, cos.Columns: -5},
			raw:    "010203",
			want:   "010203",
		},
		{
			name:   "algorithm byte 0xF8 is TIFF Sub",
			filter: Flate{},
			params: params{cos.Predictor: 10, cos.Colors: 1, cos.BitsPerComponent: 8, cos.Columns: 3},
			raw:    "f80a0505",
			want:   "0a0f14",
		},
		{
			name:   "3 bits per component, a byte with its top bit set",
			filter: Flate{},
			params: params{cos.Predictor: 2, cos.Colors: 2, cos.BitsPerComponent: 3, cos.Columns: 3},
			raw:    "800000",
			want:   "800200",
		},
		{
			name:   "3 bits per component, every bit set",
			filter: Flate{},
			params: params{cos.Predictor: 2, cos.Colors: 2, cos.BitsPerComponent: 3, cos.Columns: 3},
			raw:    "ffffff",
			want:   "ffe1ff",
		},
		{
			name:   "3 bits per component, mixed",
			filter: Flate{},
			params: params{cos.Predictor: 2, cos.Colors: 2, cos.BitsPerComponent: 3, cos.Columns: 3},
			raw:    "a5c35a",
			want:   "a5d55a",
		},
		{
			name:   "an empty stream with a zero TIFF row length",
			filter: Flate{},
			params: params{cos.Predictor: 2, cos.Columns: 0},
			raw:    "",
			want:   "",
		},
		{
			name:   "a zero PNG row length reads every byte as an algorithm",
			filter: Flate{},
			params: params{cos.Predictor: 12, cos.Columns: 0},
			raw:    "02010203",
			want:   "",
		},
		{
			// -1 * -2 * 8 is a row of 2 and (-16 + 7) / 8 is a pixel of -1
			name:    "a negative pixel is outside the row for Sub",
			filter:  Flate{},
			params:  params{cos.Predictor: 11, cos.Colors: -2, cos.BitsPerComponent: 8, cos.Columns: -1},
			raw:     "010102",
			wantErr: errPredictorIndex,
		},
		{
			name:    "a negative pixel is outside the row for Average",
			filter:  Flate{},
			params:  params{cos.Predictor: 13, cos.Colors: -2, cos.BitsPerComponent: 8, cos.Columns: -1},
			raw:     "030102",
			wantErr: errPredictorIndex,
		},
		{
			// PDFBox: IOException "Calculated row length is negative: -1"
			name:    "a negative row length",
			filter:  Flate{},
			params:  params{cos.Predictor: 12, cos.Columns: -2},
			raw:     "0001",
			wantErr: errNegativeRowLength,
		},
		{
			// -1 * -8 is a pixel of one byte, so Up decodes normally
			name:   "two negative parameters that make a row",
			filter: Flate{},
			params: params{cos.Predictor: 12, cos.Colors: -1, cos.BitsPerComponent: -8, cos.Columns: 2},
			raw:    "020102" + "020303",
			want:   "0102" + "0405",
		},
		{
			name:   "the ordinary cross-reference shape",
			filter: Flate{},
			params: params{cos.Predictor: 12, cos.Columns: 3},
			raw:    "02010203" + "02010101",
			want:   "010203" + "020304",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := decodeWithParams(t, c.filter, c.params, mustHex(t, c.raw))
			if c.wantErr != nil {
				if !errors.Is(err, c.wantErr) {
					t.Fatalf("err = %v, want %v", err, c.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}
			if want := mustHex(t, c.want); !bytes.Equal(got, want) {
				t.Errorf("decoded %x\n                    want %x", got, want)
			}
		})
	}
}

// decodeWithParams encodes raw with the filter and decodes it again with the
// given entries as the stream's /DecodeParms.
func decodeWithParams(t *testing.T, f Filter, entries map[*cos.Name]int, raw []byte) ([]byte, error) {
	t.Helper()

	var encoded bytes.Buffer
	if err := f.Encode(&encoded, bytes.NewReader(raw), cos.NewDictionary()); err != nil {
		t.Fatalf("Encode: %v", err)
	}

	params := cos.NewDictionary()
	for key, value := range entries {
		params.SetInt(key, value)
	}
	stream := cos.NewDictionary()
	if _, isLZW := f.(LZW); isLZW {
		stream.SetItem(cos.Filter, cos.LZWDecode)
	} else {
		stream.SetItem(cos.Filter, cos.FlateDecode)
	}
	stream.SetItem(cos.DecodeParms, params)

	var decoded bytes.Buffer
	_, err := f.Decode(&decoded, bytes.NewReader(encoded.Bytes()), stream, 0)
	return decoded.Bytes(), err
}

// mustHex reads hex written with spaces for readability.
func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(string(bytes.ReplaceAll([]byte(s), []byte(" "), nil)))
	if err != nil {
		t.Fatalf("bad hex %q: %v", s, err)
	}
	return b
}
