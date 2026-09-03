package pdfio

import (
	"bytes"
	"errors"
	"io"
	"math"
	"testing"
)

// Ported from io/src/test/java/org/apache/pdfbox/io/RandomAccessInputStreamTest.java.

func TestReaderPositionSkip(t *testing.T) {
	source := NewReadBufferBytes(sampleBytes())
	defer source.Close()
	r := NewReader(source)

	if n, err := r.Skip(5); err != nil || n != 5 {
		t.Fatalf("Skip = %d, %v; want 5, nil", n, err)
	}
	b, err := r.ReadByte()
	if err != nil || b != 5 {
		t.Fatalf("ReadByte = %d, %v; want 5, nil", b, err)
	}
	if available, err := r.Available(); err != nil || available != 5 {
		t.Fatalf("Available = %d, %v; want 5, nil", available, err)
	}
}

func TestReaderRead(t *testing.T) {
	source := NewReadBufferBytes(sampleBytes())
	defer source.Close()
	r := NewReader(source)

	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !bytes.Equal(got, sampleBytes()) {
		t.Fatalf("ReadAll = %v, want %v", got, sampleBytes())
	}
	if _, err := r.ReadByte(); !errors.Is(err, io.EOF) {
		t.Fatalf("ReadByte at EOF error = %v, want io.EOF", err)
	}
}

// TestReaderKeepsOwnPosition is the point of the adapter: two readers over one
// source must not disturb each other.
func TestReaderKeepsOwnPosition(t *testing.T) {
	source := NewReadBufferBytes(sampleBytes())
	defer source.Close()

	first := NewReader(source)
	second := NewReader(source)

	if b, err := first.ReadByte(); err != nil || b != 0 {
		t.Fatalf("first.ReadByte = %d, %v; want 0, nil", b, err)
	}
	if b, err := second.ReadByte(); err != nil || b != 0 {
		t.Fatalf("second.ReadByte = %d, %v; want 0, nil", b, err)
	}
	if b, err := first.ReadByte(); err != nil || b != 1 {
		t.Fatalf("first.ReadByte = %d, %v; want 1, nil", b, err)
	}
	if _, err := second.Skip(4); err != nil {
		t.Fatalf("second.Skip: %v", err)
	}
	if b, err := second.ReadByte(); err != nil || b != 5 {
		t.Fatalf("second.ReadByte = %d, %v; want 5, nil", b, err)
	}
	if b, err := first.ReadByte(); err != nil || b != 2 {
		t.Fatalf("first.ReadByte = %d, %v; want 2, nil", b, err)
	}
}

func TestReaderEmpty(t *testing.T) {
	source := NewReadBufferBytes(nil)
	defer source.Close()
	r := NewReader(source)

	if available, err := r.Available(); err != nil || available != 0 {
		t.Fatalf("Available = %d, %v; want 0, nil", available, err)
	}
	if _, err := r.ReadByte(); !errors.Is(err, io.EOF) {
		t.Fatalf("ReadByte error = %v, want io.EOF", err)
	}
	if n, err := r.Read(make([]byte, 4)); n != 0 || !errors.Is(err, io.EOF) {
		t.Fatalf("Read = %d, %v; want 0, io.EOF", n, err)
	}
}

func TestWriterRoundTrip(t *testing.T) {
	buf := NewReadWriteBuffer()
	defer buf.Close()

	w := NewWriter(buf)
	want := []byte("the quick brown fox")
	if n, err := w.Write(want); err != nil || n != len(want) {
		t.Fatalf("Write = %d, %v; want %d, nil", n, err, len(want))
	}
	if err := w.WriteByte('!'); err != nil {
		t.Fatalf("WriteByte: %v", err)
	}

	if err := SeekTo(buf, 0); err != nil {
		t.Fatalf("SeekTo(0): %v", err)
	}
	got, err := io.ReadAll(NewReader(buf))
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(got) != string(want)+"!" {
		t.Fatalf("round trip = %q, want %q", got, string(want)+"!")
	}
}

func TestMemoryStreamCache(t *testing.T) {
	newCache := MemoryOnlyStreamCache()
	cache, err := newCache()
	if err != nil {
		t.Fatalf("cache func: %v", err)
	}
	defer cache.Close()

	buf, err := cache.CreateBuffer()
	if err != nil {
		t.Fatalf("CreateBuffer: %v", err)
	}
	defer buf.Close()

	if _, err := buf.Write([]byte{1, 2, 3}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	wantLength(t, buf, 3)
}

// hugeSource reports a length no int can hold, so that the clamp Java applies
// can be exercised without allocating anything.
type hugeSource struct{ RandomAccessRead }

func (hugeSource) Length() (int64, error)   { return 1 << 40, nil }
func (hugeSource) Position() (int64, error) { return 0, nil }

// TestReaderAvailableClamps pins that Available saturates rather than
// truncating. Java is Math.min(input.length() - position, Integer.MAX_VALUE),
// and the package-level Available already clamps; this one did not.
func TestReaderAvailableClamps(t *testing.T) {
	r := NewReader(hugeSource{NewReadBufferBytes(nil)})

	got, err := r.Available()
	if err != nil {
		t.Fatalf("Available: %v", err)
	}
	if got != math.MaxInt32 {
		t.Errorf("Available = %d, want MaxInt32", got)
	}
}
