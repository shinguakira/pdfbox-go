package pdfio_test

// The `pdfio` half of `track/java-bug-fixes`: what the Go now does that the
// Java does not, with the entry of `migration/JAVA-BUGS.md` each one closes.
//
// Every expected value here comes from the arithmetic or from what the method
// promises, never from the Java — the point of these is that the Java is wrong.

import (
	"bytes"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// TestSequenceReadCannotAccumulateMinusOne is the evidence for JAVA-BUGS 2 and
// 3 being **kept** rather than fixed: the defect is unobservable.
//
// Both entries are the same shape. `readFromChunk` and Java's `read()` answer
// -1 at the end of what they have, and the loop above them adds that answer to
// a running byte count. A read that ended that way would report one byte fewer
// than it produced, and in `SequenceRead` would move the cursor backwards too.
//
// Neither loop can reach it. `ReadBuffer.Read` calls `readFromChunk` only after
// checking `remaining() > 0` and moving to the next chunk, and those are the
// only two ways `readFromChunk` answers -1. `SequenceRead.Read` caps its work
// at `totalLength - position` and advances past an exhausted source, so while
// there are bytes left to read there is a source holding them. The two entries
// say as much themselves -- "the effect is bounded", "bytesRead can never fall
// below -1" -- and A0 marked them **fix** before that was checked.
//
// This test is what makes the claim checkable rather than argued: a sequence
// read into a buffer larger than it answers what it produced.
func TestSequenceReadCannotAccumulateMinusOne(t *testing.T) {
	source := pdfio.NewReadBufferBytes([]byte{1, 2, 3, 4})
	sequence, err := pdfio.NewSequenceRead([]pdfio.RandomAccessRead{source})
	if err != nil {
		t.Fatalf("NewSequenceRead: %v", err)
	}

	buffer := make([]byte, 8)
	n, err := sequence.Read(buffer)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if n != 4 {
		t.Errorf("Read answered %d for a four-byte sequence; the -1 that ends a "+
			"source reached the count", n)
	}
	position, err := sequence.Position()
	if err != nil {
		t.Fatal(err)
	}
	if position != 4 {
		t.Errorf("the cursor is at %d after reading 4 bytes, want 4", position)
	}
	if !bytes.Equal(buffer[:4], []byte{1, 2, 3, 4}) {
		t.Errorf("the bytes read are %v, want [1 2 3 4]", buffer[:4])
	}
}

// TestSequenceReadOverALyingView is the case entry 3's Confidence line names:
// a view that says it is 100 bytes over a source that holds 10.
func TestSequenceReadOverALyingView(t *testing.T) {
	source := pdfio.NewReadBufferBytes([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
	view := pdfio.NewReadView(source, 0, 100)
	sequence, err := pdfio.NewSequenceRead([]pdfio.RandomAccessRead{view})
	if err != nil {
		t.Fatalf("NewSequenceRead: %v", err)
	}
	p := make([]byte, 20)
	n, err := sequence.Read(p)
	t.Logf("Read answered %d, err %v", n, err)
	if n != 10 {
		t.Errorf("Read answered %d bytes from a ten-byte source, want 10", n)
	}
}
