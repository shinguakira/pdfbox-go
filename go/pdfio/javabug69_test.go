package pdfio

// JAVA-BUGS 69: `RandomAccessReadBuffer.seek`'s "jump to the end of the
// buffer" branch sets the offset within the last chunk to `size % chunkSize`,
// which is 0 when the buffer holds an exact multiple of the chunk size -- the
// *start* of the last chunk rather than the end of it. Reading recovers
// because every read path seeks or checks the position first; writing does
// not.

import (
	"bytes"
	"testing"
)

// TestWriteAtAnExactChunkBoundaryAppends is the defect.
//
// The expected bytes are the ones that were written: a write at the end of a
// buffer appends. Java answers, out of the running JDK 17, an 8-byte chunk
// written full, seeked to 8, and written one more byte:
//
//	size=9 position=9
//	first 8 bytes: 99 2 3 4 5 6 7 8 (read 8)
//	readFully(9) threw java.io.IOException: No more chunks available, end of
//	    buffer reached
//
// -- the write landed on byte 0, and the length counts a ninth byte the
// chunks do not hold.
func TestWriteAtAnExactChunkBoundaryAppends(t *testing.T) {
	buffer := NewReadWriteBufferSize(8)
	writeAll(t, buffer, []byte{1, 2, 3, 4, 5, 6, 7, 8})

	noError(t, "Seek(8)", SeekTo(buffer, 8))
	writeAll(t, buffer, []byte{99})

	length, err := buffer.Length()
	noError(t, "Length", err)
	if length != 9 {
		t.Errorf("Length() = %d, want 9", length)
	}

	noError(t, "Seek(0)", SeekTo(buffer, 0))
	got := make([]byte, 9)
	if err := ReadFully(buffer, got); err != nil {
		t.Fatalf("ReadFully of the whole length: %v -- the length promises "+
			"bytes the chunks do not hold", err)
	}
	want := []byte{1, 2, 3, 4, 5, 6, 7, 8, 99}
	if !bytes.Equal(got, want) {
		t.Errorf("the buffer holds %v, want %v -- the write at position 8 "+
			"landed on byte 0", got, want)
	}
}

// TestSeekPastTheEndOfAPartChunkIsUnchanged keeps the case the fix must not
// disturb: a last chunk that is not full parks the cursor inside it.
func TestSeekPastTheEndOfAPartChunkIsUnchanged(t *testing.T) {
	buffer := NewReadWriteBufferSize(8)
	writeAll(t, buffer, []byte{1, 2, 3, 4, 5})

	noError(t, "Seek(20)", SeekTo(buffer, 20))
	writeAll(t, buffer, []byte{99})

	noError(t, "Seek(0)", SeekTo(buffer, 0))
	got := make([]byte, 6)
	if err := ReadFully(buffer, got); err != nil {
		t.Fatalf("ReadFully: %v", err)
	}
	want := []byte{1, 2, 3, 4, 5, 99}
	if !bytes.Equal(got, want) {
		t.Errorf("the buffer holds %v, want %v", got, want)
	}
}
