package pdfio

// JAVA-BUGS 68: the default `available()` is `(int) Math.min(length() -
// getPosition(), Integer.MAX_VALUE)`, which bounds the count above and not
// below, so a source whose position is past its length answers a negative
// number of bytes still to read.

import (
	"bytes"
	"testing"
)

// TestAvailableIsNeverNegative is the defect.
//
// The expected value is the lower bound
// `RandomAccessInputStream.available()`, twelve files away in the same
// package, already has: `Math.max(0, Math.min(...))`. A count of bytes still
// to be read cannot be negative, and `available` exists to size a buffer --
// a negative one is a NegativeArraySizeException at the caller. Java answers
// -16 here, out of the running JDK 17:
//
//	ReadView seek(20) ok, position=20 length=4 available=-16 isEOF=true
func TestAvailableIsNeverNegative(t *testing.T) {
	data := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	source, err := NewReadBufferFromReader(bytes.NewReader(data))
	noError(t, "NewReadBufferFromReader", err)
	view := NewReadView(source, 2, 4)

	// a view records a seek past its end verbatim, which is how the position
	// gets beyond the length in the first place
	noError(t, "Seek(20)", SeekTo(view, 20))

	available, err := Available(view)
	noError(t, "Available", err)
	if available != 0 {
		t.Errorf("Available() = %d past the end of the view, want 0", available)
	}

	eof, err := view.IsEOF()
	noError(t, "IsEOF", err)
	if !eof {
		t.Error("IsEOF() = false past the end of the view, want true")
	}
}

// TestAvailableStillCountsWhatIsLeft keeps the count the bound must not
// disturb.
func TestAvailableStillCountsWhatIsLeft(t *testing.T) {
	data := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	source, err := NewReadBufferFromReader(bytes.NewReader(data))
	noError(t, "NewReadBufferFromReader", err)
	view := NewReadView(source, 2, 4)

	noError(t, "Seek(1)", SeekTo(view, 1))
	available, err := Available(view)
	noError(t, "Available", err)
	if available != 3 {
		t.Errorf("Available() = %d one byte into a four byte view, want 3", available)
	}
}
