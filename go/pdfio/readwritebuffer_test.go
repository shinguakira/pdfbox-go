package pdfio

import (
	"bytes"
	"testing"
)

// Ported from io/src/test/java/org/apache/pdfbox/io/RandomAccessReadWriteBufferTest.java.

const numIterations = 3

func wantLength(t *testing.T, r RandomAccessRead, want int64) {
	t.Helper()
	got, err := r.Length()
	if err != nil {
		t.Fatalf("Length: %v", err)
	}
	if got != want {
		t.Fatalf("Length = %d, want %d", got, want)
	}
}

func TestReadWriteBufferClose(t *testing.T) {
	b := NewReadWriteBuffer()
	if _, err := b.Write([]byte{1, 2, 3, 4}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if b.IsClosed() {
		t.Fatal("buffer reported closed before Close")
	}
	if err := b.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !b.IsClosed() {
		t.Fatal("buffer did not report closed after Close")
	}
}

func TestReadWriteBufferClear(t *testing.T) {
	b := NewReadWriteBufferSize(4)
	defer b.Close()

	if _, err := b.Write([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	wantLength(t, b, 10)
	wantPosition(t, b, 10)

	if err := b.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if b.IsClosed() {
		t.Fatal("Clear must not close the buffer")
	}
	wantLength(t, b, 0)
	wantPosition(t, b, 0)
}

func TestReadWriteBufferLengthWriteByte(t *testing.T) {
	b := NewReadWriteBuffer()
	defer b.Close()

	wantLength(t, b, 0)
	for _, v := range []byte{1, 2, 3} {
		if err := b.WriteByte(v); err != nil {
			t.Fatalf("WriteByte: %v", err)
		}
	}
	wantLength(t, b, 3)
}

func TestReadWriteBufferLengthWriteBytes(t *testing.T) {
	b := NewReadWriteBuffer()
	defer b.Close()

	wantLength(t, b, 0)
	if _, err := b.Write([]byte{1, 2, 3, 4, 5, 6, 7}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	wantLength(t, b, 7)
	if _, err := b.Write([]byte{8, 9, 10, 11}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	wantLength(t, b, 11)
}

// TestReadWriteBufferPaging writes past the end of a chunk so that a new chunk
// has to be allocated mid-write.
func TestReadWriteBufferPaging(t *testing.T) {
	b := NewReadWriteBufferSize(5)
	defer b.Close()

	wantLength(t, b, 0)
	if _, err := b.Write([]byte{1, 2, 3, 4, 5, 6, 7}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	wantLength(t, b, 7)
	if _, err := b.Write([]byte{8, 9, 10, 11}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	wantLength(t, b, 11)
}

func TestReadWriteBufferRandomAccessRead(t *testing.T) {
	b := NewReadWriteBuffer()
	defer b.Close()

	want := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}
	if _, err := b.Write(want); err != nil {
		t.Fatalf("Write: %v", err)
	}
	wantLength(t, b, 11)
	if err := SeekTo(b, 0); err != nil {
		t.Fatalf("SeekTo(0): %v", err)
	}
	wantLength(t, b, 11)

	got := make([]byte, 11)
	if n, err := b.Read(got); err != nil || n != 11 {
		t.Fatalf("Read = %d, %v; want 11, nil", n, err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("read back %v, want %v", got, want)
	}
}

// TestReadWriteBufferEOFBugInSeek covers PDFBOX-4756: positions must stay
// correct when seeking past the last chunk.
func TestReadWriteBufferEOFBugInSeek(t *testing.T) {
	b := NewReadWriteBuffer()
	defer b.Close()

	chunk := make([]byte, DefaultChunkSize4KB)
	for i := 0; i < numIterations; i++ {
		p0, err := b.Position()
		if err != nil {
			t.Fatalf("Position: %v", err)
		}
		if _, err := b.Write(chunk); err != nil {
			t.Fatalf("Write: %v", err)
		}
		p1, _ := b.Position()
		if p1-p0 != DefaultChunkSize4KB {
			t.Fatalf("iteration %d: first write advanced %d, want %d", i, p1-p0, DefaultChunkSize4KB)
		}
		if _, err := b.Write(chunk); err != nil {
			t.Fatalf("Write: %v", err)
		}
		p2, _ := b.Position()
		if p2-p1 != DefaultChunkSize4KB {
			t.Fatalf("iteration %d: second write advanced %d, want %d", i, p2-p1, DefaultChunkSize4KB)
		}
		if err := SeekTo(b, 0); err != nil {
			t.Fatalf("SeekTo(0): %v", err)
		}
		if err := SeekTo(b, int64(i)*2*DefaultChunkSize4KB); err != nil {
			t.Fatalf("SeekTo: %v", err)
		}
	}
}

func TestReadWriteBufferBufferLength(t *testing.T) {
	b := NewReadWriteBuffer()
	defer b.Close()

	if _, err := b.Write(make([]byte, DefaultChunkSize4KB)); err != nil {
		t.Fatalf("Write: %v", err)
	}
	wantLength(t, b, DefaultChunkSize4KB)
}

// TestReadWriteBufferRoundTrip writes more than three chunks and reads the
// whole thing back, which the Java tests only do at small sizes.
func TestReadWriteBufferRoundTrip(t *testing.T) {
	b := NewReadWriteBufferSize(64)
	defer b.Close()

	want := make([]byte, 64*5+7)
	for i := range want {
		want[i] = byte(i * 7)
	}
	if n, err := b.Write(want); err != nil || n != len(want) {
		t.Fatalf("Write = %d, %v; want %d, nil", n, err, len(want))
	}
	wantLength(t, b, int64(len(want)))

	if err := SeekTo(b, 0); err != nil {
		t.Fatalf("SeekTo(0): %v", err)
	}
	got := make([]byte, len(want))
	if err := ReadFully(b, got); err != nil {
		t.Fatalf("ReadFully: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("data read back does not match data written")
	}
}
