package pdfio

// JAVA-BUGS 67: `RandomAccessReadView.rewind` checks nothing. It points the
// source at the view's position and rewinds the source, so a rewind longer
// than the view has read lands the source before the view's own start, leaves
// the view's position negative, and every later read comes out of the bytes
// the view exists to exclude.

import (
	"bytes"
	"errors"
	"testing"
)

// TestReadViewRefusesToRewindPastItsOwnStart is the defect.
//
// The expected answer is `Seek`'s, five methods above: a position before the
// start of the view is refused with the same error, `Invalid position`. That
// is also what the interface's own default `rewind` reaches, since it is
// `seek(getPosition() - bytes)` -- this override exists only to avoid the
// seek, not to drop the check.
//
// Java answers, out of the running JDK 17: a view of data[4..7] over the bytes
// 0..9, seeked to 2 and read once so its position is 3, then
//
//	rewind(5) ok: position=-2 read=2 (source now at 0)
//
// where 2 is data[2], two bytes before the view begins.
func TestReadViewRefusesToRewindPastItsOwnStart(t *testing.T) {
	data := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	source, err := NewReadBufferFromReader(bytes.NewReader(data))
	noError(t, "NewReadBufferFromReader", err)
	view := NewReadView(source, 4, 4)

	noError(t, "Seek(2)", SeekTo(view, 2))
	wantByteOrEOF(t, view, 6)

	if err := Rewind(view, 5); !errors.Is(err, ErrInvalidPosition) {
		t.Errorf("Rewind(5) from position 3 = %v, want ErrInvalidPosition", err)
	}
	position, err := view.Position()
	noError(t, "Position", err)
	if position != 3 {
		t.Errorf("Position() = %d after the refused rewind, want the 3 it was at",
			position)
	}
	// and the view still reads its own bytes
	wantByteOrEOF(t, view, 7)
}

// TestReadViewStillRewindsInsideItself keeps the rewind the check must not
// turn away.
func TestReadViewStillRewindsInsideItself(t *testing.T) {
	data := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	source, err := NewReadBufferFromReader(bytes.NewReader(data))
	noError(t, "NewReadBufferFromReader", err)
	view := NewReadView(source, 4, 4)

	noError(t, "Seek(2)", SeekTo(view, 2))
	wantByteOrEOF(t, view, 6)
	noError(t, "Rewind(3)", Rewind(view, 3))

	position, err := view.Position()
	noError(t, "Position", err)
	if position != 0 {
		t.Errorf("Position() = %d after rewinding to the start, want 0", position)
	}
	wantByteOrEOF(t, view, 4)
}
