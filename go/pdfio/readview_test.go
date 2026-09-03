package pdfio

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

// Ported from io/src/test/java/org/apache/pdfbox/io/RandomAccessReadViewTest.java.

func viewSampleBytes() []byte {
	values := make([]byte, 21)
	for i := range values {
		values[i] = byte(i)
	}
	return values
}

func newSampleView(t *testing.T) (*ReadBuffer, *ReadView) {
	t.Helper()
	source, err := NewReadBufferFromReader(bytes.NewReader(viewSampleBytes()))
	if err != nil {
		t.Fatalf("NewReadBufferFromReader: %v", err)
	}
	return source, NewReadView(source, 10, 20)
}

func TestReadViewPositionSkip(t *testing.T) {
	source, view := newSampleView(t)
	defer source.Close()
	defer view.Close()

	wantPosition(t, view, 0)
	if v, err := Peek(view); err != nil || v != 10 {
		t.Fatalf("Peek = %d, %v; want 10, nil", v, err)
	}
	if err := Skip(view, 5); err != nil {
		t.Fatalf("Skip: %v", err)
	}
	wantPosition(t, view, 5)
	if v, err := Peek(view); err != nil || v != 15 {
		t.Fatalf("Peek = %d, %v; want 15, nil", v, err)
	}
}

func TestReadViewPositionRead(t *testing.T) {
	source, view := newSampleView(t)
	defer source.Close()

	wantPosition(t, view, 0)
	wantByte(t, view, 10)
	wantByte(t, view, 11)
	wantByte(t, view, 12)
	wantPosition(t, view, 3)

	if view.IsClosed() {
		t.Fatal("view reported closed before Close")
	}
	if err := view.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !view.IsClosed() {
		t.Fatal("view did not report closed after Close")
	}
	// closing twice must not be a problem
	if err := view.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestReadViewSeekEOF(t *testing.T) {
	source, view := newSampleView(t)
	defer source.Close()

	if err := SeekTo(view, 3); err != nil {
		t.Fatalf("SeekTo(3): %v", err)
	}
	wantPosition(t, view, 3)

	if err := SeekTo(view, -1); !errors.Is(err, ErrInvalidPosition) {
		t.Fatalf("SeekTo(-1) error = %v, want ErrInvalidPosition", err)
	}
	if eof, err := view.IsEOF(); err != nil || eof {
		t.Fatalf("IsEOF = %v, %v; want false, nil", eof, err)
	}
	if err := SeekTo(view, 20); err != nil {
		t.Fatalf("SeekTo(20): %v", err)
	}
	if eof, err := view.IsEOF(); err != nil || !eof {
		t.Fatalf("IsEOF = %v, %v; want true, nil", eof, err)
	}
	if _, err := view.ReadByte(); !errors.Is(err, io.EOF) {
		t.Fatalf("ReadByte at EOF error = %v, want io.EOF", err)
	}
	if _, err := view.Read(make([]byte, 1)); !errors.Is(err, io.EOF) {
		t.Fatalf("Read at EOF error = %v, want io.EOF", err)
	}

	view.Close()
	if _, err := view.ReadByte(); !errors.Is(err, ErrClosed) {
		t.Fatalf("ReadByte after Close error = %v, want ErrClosed", err)
	}
}

func TestReadViewPositionReadBytes(t *testing.T) {
	source, view := newSampleView(t)
	defer source.Close()
	defer view.Close()

	wantPosition(t, view, 0)
	buf := make([]byte, 4)
	if _, err := view.Read(buf); err != nil {
		t.Fatalf("Read: %v", err)
	}
	if buf[0] != 10 || buf[3] != 13 {
		t.Fatalf("buf = %v, want first 10 and last 13", buf)
	}
	wantPosition(t, view, 4)

	if _, err := view.Read(buf[1:3]); err != nil {
		t.Fatalf("Read into slice: %v", err)
	}
	if want := []byte{10, 14, 15, 13}; !bytes.Equal(buf, want) {
		t.Fatalf("buf = %v, want %v", buf, want)
	}
	wantPosition(t, view, 6)
}

func TestReadViewPositionPeek(t *testing.T) {
	source, view := newSampleView(t)
	defer source.Close()
	defer view.Close()

	wantPosition(t, view, 0)
	if err := Skip(view, 6); err != nil {
		t.Fatalf("Skip: %v", err)
	}
	wantPosition(t, view, 6)
	if v, err := Peek(view); err != nil || v != 16 {
		t.Fatalf("Peek = %d, %v; want 16, nil", v, err)
	}
	wantPosition(t, view, 6)
}

func TestReadViewRewind(t *testing.T) {
	source, view := newSampleView(t)
	defer source.Close()
	defer view.Close()

	wantPosition(t, view, 0)
	view.ReadByte()
	view.ReadByte()

	readBytes := make([]byte, 6)
	if n, err := view.Read(readBytes); err != nil || n != len(readBytes) {
		t.Fatalf("Read = %d, %v; want %d, nil", n, err, len(readBytes))
	}
	wantPosition(t, view, 8)

	if err := Rewind(view, int64(len(readBytes))); err != nil {
		t.Fatalf("Rewind: %v", err)
	}
	wantPosition(t, view, 2)
	wantByte(t, view, 12)
	wantPosition(t, view, 3)

	if _, err := view.Read(readBytes[2:6]); err != nil {
		t.Fatalf("Read into slice: %v", err)
	}
	if readBytes[0] != 12 || readBytes[2] != 13 || readBytes[5] != 16 {
		t.Fatalf("readBytes = %v, want [12 _ 13 _ _ 16]", readBytes)
	}
	wantPosition(t, view, 7)
	if err := Rewind(view, 4); err != nil {
		t.Fatalf("Rewind: %v", err)
	}
	wantPosition(t, view, 3)
}

func TestReadViewCreateView(t *testing.T) {
	source, view := newSampleView(t)
	defer source.Close()
	defer view.Close()

	if _, err := view.CreateView(0, 20); !errors.Is(err, ErrViewNotSupported) {
		t.Fatalf("CreateView error = %v, want ErrViewNotSupported", err)
	}
}
