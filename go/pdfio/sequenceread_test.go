package pdfio

import (
	"errors"
	"io"
	"testing"
)

// Ported from io/src/test/java/org/apache/pdfbox/io/SequenceRandomAccessReadTest.java.

func newSequence(t *testing.T, parts ...string) *SequenceRead {
	t.Helper()
	readers := make([]RandomAccessRead, 0, len(parts))
	for _, p := range parts {
		readers = append(readers, NewReadBufferBytes([]byte(p)))
	}
	s, err := NewSequenceRead(readers)
	if err != nil {
		t.Fatalf("NewSequenceRead: %v", err)
	}
	return s
}

func TestSequenceCreateAndRead(t *testing.T) {
	input1 := "This is a test string number 1"
	input2 := "This is a test string number 2"

	readers := []RandomAccessRead{
		NewReadBufferBytes([]byte(input1)),
		NewReadBufferBytes([]byte(input2)),
	}
	s, err := NewSequenceRead(readers)
	if err != nil {
		t.Fatalf("NewSequenceRead: %v", err)
	}

	if _, err := s.CreateView(0, 10); !errors.Is(err, ErrViewNotSupported) {
		t.Fatalf("CreateView error = %v, want ErrViewNotSupported", err)
	}

	overall := int64(len(input1) + len(input2))
	wantLength(t, s, overall)

	got := make([]byte, overall)
	if n, err := s.Read(got); err != nil || int64(n) != overall {
		t.Fatalf("Read = %d, %v; want %d, nil", n, err, overall)
	}
	if string(got) != input1+input2 {
		t.Fatalf("Read = %q, want %q", got, input1+input2)
	}
	s.Close()

	// nil and empty inputs are rejected
	if _, err := NewSequenceRead(nil); err == nil {
		t.Fatal("NewSequenceRead(nil) succeeded, want error")
	}
	if _, err := NewSequenceRead([]RandomAccessRead{}); err == nil {
		t.Fatal("NewSequenceRead(empty) succeeded, want error")
	}
	// the sources were closed with the sequence above, so reusing them fails
	if _, err := NewSequenceRead(readers); err == nil {
		t.Fatal("NewSequenceRead over closed sources succeeded, want error")
	}
}

func TestSequenceSeekPeekAndRewind(t *testing.T) {
	s := newSequence(t, "01234567890123456789", "abcdefghijklmnopqrst")
	defer s.Close()

	// first part of the sequence
	if err := SeekTo(s, 4); err != nil {
		t.Fatalf("SeekTo(4): %v", err)
	}
	wantPosition(t, s, 4)
	wantByte(t, s, '4')
	wantPosition(t, s, 5)
	if err := Rewind(s, 1); err != nil {
		t.Fatalf("Rewind: %v", err)
	}
	wantPosition(t, s, 4)
	wantByte(t, s, '4')
	if v, err := Peek(s); err != nil || v != '5' {
		t.Fatalf("Peek = %q, %v; want '5', nil", v, err)
	}
	wantPosition(t, s, 5)
	wantByte(t, s, '5')
	wantPosition(t, s, 6)

	// second part of the sequence
	if err := SeekTo(s, 24); err != nil {
		t.Fatalf("SeekTo(24): %v", err)
	}
	wantPosition(t, s, 24)
	wantByte(t, s, 'e')
	if err := Rewind(s, 1); err != nil {
		t.Fatalf("Rewind: %v", err)
	}
	wantByte(t, s, 'e')
	if v, err := Peek(s); err != nil || v != 'f' {
		t.Fatalf("Peek = %q, %v; want 'f', nil", v, err)
	}
	wantByte(t, s, 'f')

	if err := SeekTo(s, -1); !errors.Is(err, ErrInvalidPosition) {
		t.Fatalf("SeekTo(-1) error = %v, want ErrInvalidPosition", err)
	}
}

func TestSequenceBorderCases(t *testing.T) {
	s := newSequence(t, "01234567890123456789", "abcdefghijklmnopqrst")
	defer s.Close()

	// last byte of the first source, then across the boundary
	if err := SeekTo(s, 19); err != nil {
		t.Fatalf("SeekTo(19): %v", err)
	}
	wantByte(t, s, '9')
	if err := Rewind(s, 1); err != nil {
		t.Fatalf("Rewind: %v", err)
	}
	wantByte(t, s, '9')
	if v, err := Peek(s); err != nil || v != 'a' {
		t.Fatalf("Peek = %q, %v; want 'a', nil", v, err)
	}
	wantByte(t, s, 'a')

	// a single read that spans the boundary
	if err := SeekTo(s, 17); err != nil {
		t.Fatalf("SeekTo(17): %v", err)
	}
	got := make([]byte, 6)
	if n, err := s.Read(got); err != nil || n != 6 {
		t.Fatalf("Read = %d, %v; want 6, nil", n, err)
	}
	if string(got) != "789abc" {
		t.Fatalf("Read = %q, want %q", got, "789abc")
	}
	wantPosition(t, s, 23)

	// rewind back over the boundary and read the same bytes again
	if err := Rewind(s, 6); err != nil {
		t.Fatalf("Rewind: %v", err)
	}
	wantPosition(t, s, 17)
	if n, err := s.Read(got); err != nil || n != 6 {
		t.Fatalf("Read = %d, %v; want 6, nil", n, err)
	}
	if string(got) != "789abc" {
		t.Fatalf("Read = %q, want %q", got, "789abc")
	}

	// back to the start
	if err := SeekTo(s, 0); err != nil {
		t.Fatalf("SeekTo(0): %v", err)
	}
	if n, err := s.Read(got); err != nil || n != 6 {
		t.Fatalf("Read = %d, %v; want 6, nil", n, err)
	}
	if string(got) != "012345" {
		t.Fatalf("Read = %q, want %q", got, "012345")
	}
}

func TestSequenceEOF(t *testing.T) {
	s := newSequence(t, "01234567890123456789", "abcdefghijklmnopqrst")
	const overall = 40

	if err := SeekTo(s, overall-1); err != nil {
		t.Fatalf("SeekTo: %v", err)
	}
	if eof, err := s.IsEOF(); err != nil || eof {
		t.Fatalf("IsEOF = %v, %v; want false, nil", eof, err)
	}
	if v, err := Peek(s); err != nil || v != 't' {
		t.Fatalf("Peek = %q, %v; want 't', nil", v, err)
	}
	if eof, err := s.IsEOF(); err != nil || eof {
		t.Fatalf("IsEOF after Peek = %v, %v; want false, nil", eof, err)
	}
	wantByte(t, s, 't')
	if eof, err := s.IsEOF(); err != nil || !eof {
		t.Fatalf("IsEOF = %v, %v; want true, nil", eof, err)
	}
	if _, err := s.ReadByte(); !errors.Is(err, io.EOF) {
		t.Fatalf("ReadByte at EOF error = %v, want io.EOF", err)
	}
	if _, err := s.Read(make([]byte, 1)); !errors.Is(err, io.EOF) {
		t.Fatalf("Read at EOF error = %v, want io.EOF", err)
	}

	if err := Rewind(s, 5); err != nil {
		t.Fatalf("Rewind: %v", err)
	}
	if eof, err := s.IsEOF(); err != nil || eof {
		t.Fatalf("IsEOF after Rewind = %v, %v; want false, nil", eof, err)
	}
	got := make([]byte, 5)
	if n, err := s.Read(got); err != nil || n != 5 {
		t.Fatalf("Read = %d, %v; want 5, nil", n, err)
	}
	if string(got) != "pqrst" {
		t.Fatalf("Read = %q, want %q", got, "pqrst")
	}
	if eof, err := s.IsEOF(); err != nil || !eof {
		t.Fatalf("IsEOF = %v, %v; want true, nil", eof, err)
	}

	// seeking beyond the end parks the cursor at the end
	if err := SeekTo(s, overall+10); err != nil {
		t.Fatalf("SeekTo beyond end: %v", err)
	}
	if eof, err := s.IsEOF(); err != nil || !eof {
		t.Fatalf("IsEOF = %v, %v; want true, nil", eof, err)
	}
	wantPosition(t, s, overall)

	if s.IsClosed() {
		t.Fatal("sequence reported closed before Close")
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !s.IsClosed() {
		t.Fatal("sequence did not report closed after Close")
	}
	// closing twice must not be a problem
	if err := s.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
	if _, err := s.ReadByte(); !errors.Is(err, ErrClosed) {
		t.Fatalf("ReadByte after Close error = %v, want ErrClosed", err)
	}
}

// TestSequenceSkipsEmptySources checks that zero length sources are dropped, as
// the Java constructor does when it filters the list.
func TestSequenceSkipsEmptySources(t *testing.T) {
	readers := []RandomAccessRead{
		NewReadBufferBytes([]byte("ab")),
		NewReadBufferBytes(nil),
		NewReadBufferBytes([]byte("cd")),
	}
	s, err := NewSequenceRead(readers)
	if err != nil {
		t.Fatalf("NewSequenceRead: %v", err)
	}
	defer s.Close()

	wantLength(t, s, 4)
	got := make([]byte, 4)
	if err := ReadFully(s, got); err != nil {
		t.Fatalf("ReadFully: %v", err)
	}
	if string(got) != "abcd" {
		t.Fatalf("Read = %q, want %q", got, "abcd")
	}
}
