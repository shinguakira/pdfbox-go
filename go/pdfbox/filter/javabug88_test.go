package filter

// JAVA-BUGS 88: PredictorOutputStream.write never returns when a TIFF
// predictor's row length is zero. toRead is always 0, so the offset never
// moves, and the empty row counts as full on every pass round the loop.

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// TestZeroTIFFRowLengthIsRefused is the defect.
//
// PDFBox gives no answer for /DecodeParms << /Predictor 2 /Columns 0 >> over
// three bytes -- it was still going round the loop after five seconds -- so
// there is no Java value to copy. The port answers errZeroRowLength instead.
// The same parameters over an empty stream PDFBox does answer, with nothing,
// and so does the port; that case is in TestPredictorAnswersWhatPDFBoxAnswers.
func TestZeroTIFFRowLengthIsRefused(t *testing.T) {
	var encoded bytes.Buffer
	if err := (Flate{}).Encode(&encoded, bytes.NewReader([]byte{1, 2, 3}), cos.NewDictionary()); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	params := cos.NewDictionary()
	params.SetInt(cos.Predictor, 2)
	params.SetInt(cos.Columns, 0)
	stream := cos.NewDictionary()
	stream.SetItem(cos.Filter, cos.FlateDecode)
	stream.SetItem(cos.DecodeParms, params)

	done := make(chan error, 1)
	go func() {
		var decoded bytes.Buffer
		_, err := (Flate{}).Decode(&decoded, bytes.NewReader(encoded.Bytes()), stream, 0)
		done <- err
	}()

	select {
	case err := <-done:
		if !errors.Is(err, errZeroRowLength) {
			t.Fatalf("err = %v, want %v", err, errZeroRowLength)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("decoding did not finish: the port goes round the loop PDFBox does")
	}
}
