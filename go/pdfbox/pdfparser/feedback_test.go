package pdfparser

import (
	"errors"
	"io"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// The tests below pin two defects the slice 3 review feedback found. Each fails
// without its fix.

// errRead is the failure a source reports that is not the end of the data.
var errRead = errors.New("the disk fell over")

// failingRead is a source whose Read reports errRead rather than reaching the
// end of the data.
//
// Java's RandomAccessRead.read throws IOException for this and returns -1 only
// at the end; every caller distinguishes the two. The port must too.
type failingRead struct {
	pdfio.RandomAccessRead
}

func (f failingRead) Read(p []byte) (int, error) { return 0, errRead }

func newFailingRead() failingRead {
	return failingRead{RandomAccessRead: pdfio.NewReadBufferBytes([]byte("12345"))}
}

// TestIsDigitAtReportsAReadFailure pins the first: isDigitAt swallowed the
// error, so a source that failed mid-file read as "not a digit" and the parse
// carried on over corrupt state.
func TestIsDigitAtReportsAReadFailure(t *testing.T) {
	_, err := isDigitAt(newFailingRead())
	if !errors.Is(err, errRead) {
		t.Errorf("isDigitAt on a failing source gave err = %v, want %v", err, errRead)
	}
}

// TestIsDigitAtIsQuietAtTheEnd checks the other half: the end of the data is
// not a failure, and reads as "not a digit", which is what Java's peek
// returning -1 does.
func TestIsDigitAtIsQuietAtTheEnd(t *testing.T) {
	source := pdfio.NewReadBufferBytes([]byte("7"))
	if _, err := source.Seek(1, io.SeekStart); err != nil {
		t.Fatalf("seek: %v", err)
	}
	digit, err := isDigitAt(source)
	if err != nil {
		t.Errorf("isDigitAt at the end gave err = %v, want none", err)
	}
	if digit {
		t.Error("isDigitAt at the end reported a digit")
	}

	// and a digit still reads as one
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("seek: %v", err)
	}
	digit, err = isDigitAt(source)
	if err != nil || !digit {
		t.Errorf("isDigitAt on '7' = %v, %v, want true, nil", digit, err)
	}
}

// TestXrefStreamReadNextValueReportsAReadFailure pins the second:
// readNextValue returned nil for every short read, so a source that failed
// mid-entry left the buffer holding the previous entry and the cross-reference
// table took those bytes as real.
func TestXrefStreamReadNextValueReportsAReadFailure(t *testing.T) {
	parser := &XrefStreamParser{source: newFailingRead()}
	err := parser.readNextValue(make([]byte, 4))
	if !errors.Is(err, errRead) {
		t.Errorf("readNextValue on a failing source gave err = %v, want %v", err, errRead)
	}
}

// TestXrefStreamReadNextValueIsQuietAtTheEnd checks the other half: running out
// of data is where Java's loop ends, leaving the rest of the buffer as it was.
func TestXrefStreamReadNextValueIsQuietAtTheEnd(t *testing.T) {
	parser := &XrefStreamParser{source: pdfio.NewReadBufferBytes([]byte{1, 2})}
	value := []byte{9, 9, 9, 9}
	if err := parser.readNextValue(value); err != nil {
		t.Fatalf("readNextValue gave err = %v, want none", err)
	}
	if value[0] != 1 || value[1] != 2 {
		t.Errorf("value = %v, want it to start 1 2", value)
	}
	if value[2] != 9 || value[3] != 9 {
		t.Errorf("value = %v, want the tail left as it was", value)
	}
}
