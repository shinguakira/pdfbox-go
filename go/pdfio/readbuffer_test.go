package pdfio

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// Ported from io/src/test/java/org/apache/pdfbox/io/RandomAccessReadBufferTest.java.
//
// testPDFBOX5111 is not ported: it downloads a PDF from the Apache issue
// tracker, which a unit test should not depend on.

func sampleBytes() []byte {
	return []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
}

func newSampleBuffer(t *testing.T) *ReadBuffer {
	t.Helper()
	b, err := NewReadBufferFromReader(bytes.NewReader(sampleBytes()))
	if err != nil {
		t.Fatalf("NewReadBufferFromReader: %v", err)
	}
	return b
}

func wantPosition(t *testing.T, r RandomAccessRead, want int64) {
	t.Helper()
	got, err := r.Position()
	if err != nil {
		t.Fatalf("Position: %v", err)
	}
	if got != want {
		t.Fatalf("Position = %d, want %d", got, want)
	}
}

func wantByte(t *testing.T, r RandomAccessRead, want byte) {
	t.Helper()
	got, err := r.ReadByte()
	if err != nil {
		t.Fatalf("ReadByte: %v", err)
	}
	if got != want {
		t.Fatalf("ReadByte = %d, want %d", got, want)
	}
}

func TestReadBufferPositionSkip(t *testing.T) {
	b := newSampleBuffer(t)
	defer b.Close()

	wantPosition(t, b, 0)
	if err := Skip(b, 5); err != nil {
		t.Fatalf("Skip: %v", err)
	}
	wantByte(t, b, 5)
	wantPosition(t, b, 6)
}

func TestReadBufferPositionRead(t *testing.T) {
	b := newSampleBuffer(t)

	wantPosition(t, b, 0)
	wantByte(t, b, 0)
	wantByte(t, b, 1)
	wantByte(t, b, 2)
	wantPosition(t, b, 3)

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

func TestReadBufferSeekEOF(t *testing.T) {
	b := newSampleBuffer(t)

	if err := SeekTo(b, 3); err != nil {
		t.Fatalf("SeekTo(3): %v", err)
	}
	wantPosition(t, b, 3)

	if err := SeekTo(b, -1); !errors.Is(err, ErrInvalidPosition) {
		t.Fatalf("SeekTo(-1) error = %v, want ErrInvalidPosition", err)
	}

	if eof, err := b.IsEOF(); err != nil || eof {
		t.Fatalf("IsEOF = %v, %v; want false, nil", eof, err)
	}
	if err := SeekTo(b, 20); err != nil {
		t.Fatalf("SeekTo(20): %v", err)
	}
	if eof, err := b.IsEOF(); err != nil || !eof {
		t.Fatalf("IsEOF = %v, %v; want true, nil", eof, err)
	}
	if _, err := b.ReadByte(); !errors.Is(err, io.EOF) {
		t.Fatalf("ReadByte at EOF error = %v, want io.EOF", err)
	}
	if _, err := b.Read(make([]byte, 1)); !errors.Is(err, io.EOF) {
		t.Fatalf("Read at EOF error = %v, want io.EOF", err)
	}

	b.Close()
	if _, err := b.ReadByte(); !errors.Is(err, ErrClosed) {
		t.Fatalf("ReadByte after Close error = %v, want ErrClosed", err)
	}
}

func TestReadBufferPositionReadBytes(t *testing.T) {
	b := newSampleBuffer(t)
	defer b.Close()

	wantPosition(t, b, 0)
	buf := make([]byte, 4)
	if _, err := b.Read(buf); err != nil {
		t.Fatalf("Read: %v", err)
	}
	if buf[0] != 0 || buf[3] != 3 {
		t.Fatalf("buf = %v, want first 0 and last 3", buf)
	}
	wantPosition(t, b, 4)

	// Java calls read(buffer, 1, 2); the Go equivalent is a slice of the target.
	if _, err := b.Read(buf[1:3]); err != nil {
		t.Fatalf("Read into slice: %v", err)
	}
	if want := []byte{0, 4, 5, 3}; !bytes.Equal(buf, want) {
		t.Fatalf("buf = %v, want %v", buf, want)
	}
	wantPosition(t, b, 6)
}

func TestReadBufferPositionPeek(t *testing.T) {
	b := newSampleBuffer(t)
	defer b.Close()

	wantPosition(t, b, 0)
	if err := Skip(b, 6); err != nil {
		t.Fatalf("Skip: %v", err)
	}
	wantPosition(t, b, 6)

	v, err := Peek(b)
	if err != nil {
		t.Fatalf("Peek: %v", err)
	}
	if v != 6 {
		t.Fatalf("Peek = %d, want 6", v)
	}
	wantPosition(t, b, 6)
}

func TestReadBufferRewind(t *testing.T) {
	b := newSampleBuffer(t)
	defer b.Close()

	wantPosition(t, b, 0)
	b.ReadByte()
	b.ReadByte()

	readBytes := make([]byte, 6)
	n, err := b.Read(readBytes)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if n != len(readBytes) {
		t.Fatalf("Read = %d, want %d", n, len(readBytes))
	}
	wantPosition(t, b, 8)

	if err := Rewind(b, int64(len(readBytes))); err != nil {
		t.Fatalf("Rewind: %v", err)
	}
	wantPosition(t, b, 2)
	wantByte(t, b, 2)
	wantPosition(t, b, 3)

	if _, err := b.Read(readBytes[2:6]); err != nil {
		t.Fatalf("Read into slice: %v", err)
	}
	wantPosition(t, b, 7)
	if err := Rewind(b, 4); err != nil {
		t.Fatalf("Rewind: %v", err)
	}
	wantPosition(t, b, 3)
}

func TestReadBufferEmpty(t *testing.T) {
	b := NewReadBufferBytes(nil)
	defer b.Close()

	if _, err := b.ReadByte(); !errors.Is(err, io.EOF) {
		t.Fatalf("ReadByte error = %v, want io.EOF", err)
	}
	if _, err := Peek(b); !errors.Is(err, io.EOF) {
		t.Fatalf("Peek error = %v, want io.EOF", err)
	}
	if _, err := b.Read(make([]byte, 6)); !errors.Is(err, io.EOF) {
		t.Fatalf("Read error = %v, want io.EOF", err)
	}
	if err := SeekTo(b, 0); err != nil {
		t.Fatalf("SeekTo(0): %v", err)
	}
	wantPosition(t, b, 0)
	if err := SeekTo(b, 6); err != nil {
		t.Fatalf("SeekTo(6): %v", err)
	}
	wantPosition(t, b, 0)
	if eof, err := b.IsEOF(); err != nil || !eof {
		t.Fatalf("IsEOF = %v, %v; want true, nil", eof, err)
	}
	if err := Rewind(b, 3); !errors.Is(err, ErrInvalidPosition) {
		t.Fatalf("Rewind(3) error = %v, want ErrInvalidPosition", err)
	}
}

func TestReadBufferView(t *testing.T) {
	b := newSampleBuffer(t)
	defer b.Close()

	view, err := b.CreateView(3, 5)
	if err != nil {
		t.Fatalf("CreateView: %v", err)
	}
	defer view.Close()

	wantPosition(t, view, 0)
	wantByte(t, view, 3)
	wantByte(t, view, 4)
	wantByte(t, view, 5)
	wantPosition(t, view, 3)
}

// TestReadBufferPDFBOX5158 covers PDFBOX-5158: an endless loop when reading a
// stream whose length is an exact multiple of the chunk size.
func TestReadBufferPDFBOX5158(t *testing.T) {
	path := filepath.Join(t.TempDir(), "len4096.pdf")
	if err := os.WriteFile(path, make([]byte, 4096), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	b, err := NewReadBufferFromReadCloser(f)
	if err != nil {
		t.Fatalf("NewReadBufferFromReadCloser: %v", err)
	}
	defer b.Close()

	if length, err := b.Length(); err != nil || length != 4096 {
		t.Fatalf("Length = %d, %v; want 4096, nil", length, err)
	}
	wantByte(t, b, 0)
}

// TestReadBufferPDFBOX5161 covers PDFBOX-5161: reads failing after exactly one
// chunk has been consumed.
func TestReadBufferPDFBOX5161(t *testing.T) {
	b, err := NewReadBufferFromReader(bytes.NewReader(make([]byte, 4099)))
	if err != nil {
		t.Fatalf("NewReadBufferFromReader: %v", err)
	}
	defer b.Close()

	buf := make([]byte, 4096)
	if n, err := b.Read(buf); err != nil || n != 4096 {
		t.Fatalf("Read = %d, %v; want 4096, nil", n, err)
	}
	if n, err := b.Read(buf[:3]); err != nil || n != 3 {
		t.Fatalf("Read = %d, %v; want 3, nil", n, err)
	}
}

// TestReadBufferPDFBOX5764 covers PDFBOX-5764: the wrapping constructor must
// use the length of the given buffer, not its capacity.
func TestReadBufferPDFBOX5764(t *testing.T) {
	const capacity, length = 4096, 2048
	b := NewReadBufferBytes(make([]byte, capacity)[:length])
	defer b.Close()

	buf := make([]byte, capacity)
	if n, err := b.Read(buf); err != nil || n != length {
		t.Fatalf("Read = %d, %v; want %d, nil", n, err, length)
	}
}

// TestReadBufferChunkBoundary reads across several chunks in one call, which
// the sample-sized tests above never exercise.
func TestReadBufferChunkBoundary(t *testing.T) {
	data := make([]byte, DefaultChunkSize4KB*3+17)
	for i := range data {
		data[i] = byte(i)
	}
	b, err := NewReadBufferFromReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("NewReadBufferFromReader: %v", err)
	}
	defer b.Close()

	if length, err := b.Length(); err != nil || length != int64(len(data)) {
		t.Fatalf("Length = %d, %v; want %d, nil", length, err, len(data))
	}
	got := make([]byte, len(data))
	if err := ReadFully(b, got); err != nil {
		t.Fatalf("ReadFully: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatal("data read back does not match data written")
	}
	if eof, err := b.IsEOF(); err != nil || !eof {
		t.Fatalf("IsEOF = %v, %v; want true, nil", eof, err)
	}
}

func TestReadFullyPrematureEOF(t *testing.T) {
	b := NewReadBufferBytes([]byte{1, 2, 3})
	defer b.Close()

	if err := ReadFully(b, make([]byte, 4)); !errors.Is(err, ErrPrematureEOF) {
		t.Fatalf("ReadFully error = %v, want ErrPrematureEOF", err)
	}
}
