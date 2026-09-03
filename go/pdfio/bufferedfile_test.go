package pdfio

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// Ported from io/src/test/java/org/apache/pdfbox/io/RandomAccessReadBufferedFileTest.java.
// The Java tests read fixture files checked into the source tree; the port
// writes the equivalent data to a temp file instead.

// writeTempFile writes data to a file in the test temp dir and returns its path.
func writeTempFile(t *testing.T, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

func openSampleFile(t *testing.T) *BufferedFile {
	t.Helper()
	path := writeTempFile(t, "sample.bin", sampleBytes())
	f, err := OpenBufferedFile(path)
	if err != nil {
		t.Fatalf("OpenBufferedFile: %v", err)
	}
	return f
}

func TestBufferedFilePositionSkip(t *testing.T) {
	f := openSampleFile(t)
	defer f.Close()

	wantPosition(t, f, 0)
	if err := Skip(f, 5); err != nil {
		t.Fatalf("Skip: %v", err)
	}
	wantByte(t, f, 5)
	wantPosition(t, f, 6)
}

func TestBufferedFilePositionRead(t *testing.T) {
	f := openSampleFile(t)

	wantPosition(t, f, 0)
	wantByte(t, f, 0)
	wantByte(t, f, 1)
	wantByte(t, f, 2)
	wantPosition(t, f, 3)

	if f.IsClosed() {
		t.Fatal("file reported closed before Close")
	}
	if err := f.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !f.IsClosed() {
		t.Fatal("file did not report closed after Close")
	}
}

func TestBufferedFileSeekEOF(t *testing.T) {
	f := openSampleFile(t)

	if err := SeekTo(f, 3); err != nil {
		t.Fatalf("SeekTo(3): %v", err)
	}
	wantPosition(t, f, 3)

	if err := SeekTo(f, -1); !errors.Is(err, ErrInvalidPosition) {
		t.Fatalf("SeekTo(-1) error = %v, want ErrInvalidPosition", err)
	}
	if eof, err := f.IsEOF(); err != nil || eof {
		t.Fatalf("IsEOF = %v, %v; want false, nil", eof, err)
	}
	if err := SeekTo(f, 20); err != nil {
		t.Fatalf("SeekTo(20): %v", err)
	}
	if eof, err := f.IsEOF(); err != nil || !eof {
		t.Fatalf("IsEOF = %v, %v; want true, nil", eof, err)
	}
	if _, err := f.ReadByte(); !errors.Is(err, io.EOF) {
		t.Fatalf("ReadByte at EOF error = %v, want io.EOF", err)
	}
	if _, err := f.Read(make([]byte, 1)); !errors.Is(err, io.EOF) {
		t.Fatalf("Read at EOF error = %v, want io.EOF", err)
	}

	f.Close()
	if _, err := f.ReadByte(); !errors.Is(err, ErrClosed) {
		t.Fatalf("ReadByte after Close error = %v, want ErrClosed", err)
	}
}

func TestBufferedFilePositionReadBytes(t *testing.T) {
	f := openSampleFile(t)
	defer f.Close()

	wantPosition(t, f, 0)
	buf := make([]byte, 4)
	if _, err := f.Read(buf); err != nil {
		t.Fatalf("Read: %v", err)
	}
	if buf[0] != 0 || buf[3] != 3 {
		t.Fatalf("buf = %v, want first 0 and last 3", buf)
	}
	wantPosition(t, f, 4)

	if _, err := f.Read(buf[1:3]); err != nil {
		t.Fatalf("Read into slice: %v", err)
	}
	if want := []byte{0, 4, 5, 3}; !bytes.Equal(buf, want) {
		t.Fatalf("buf = %v, want %v", buf, want)
	}
	wantPosition(t, f, 6)
}

func TestBufferedFilePositionPeek(t *testing.T) {
	f := openSampleFile(t)
	defer f.Close()

	if err := Skip(f, 6); err != nil {
		t.Fatalf("Skip: %v", err)
	}
	wantPosition(t, f, 6)
	if v, err := Peek(f); err != nil || v != 6 {
		t.Fatalf("Peek = %d, %v; want 6, nil", v, err)
	}
	wantPosition(t, f, 6)
}

func TestBufferedFileRewind(t *testing.T) {
	f := openSampleFile(t)
	defer f.Close()

	f.ReadByte()
	f.ReadByte()
	readBytes := make([]byte, 6)
	if n, err := f.Read(readBytes); err != nil || n != len(readBytes) {
		t.Fatalf("Read = %d, %v; want %d, nil", n, err, len(readBytes))
	}
	wantPosition(t, f, 8)
	if err := Rewind(f, int64(len(readBytes))); err != nil {
		t.Fatalf("Rewind: %v", err)
	}
	wantPosition(t, f, 2)
	wantByte(t, f, 2)
}

func TestBufferedFileEmpty(t *testing.T) {
	path := writeTempFile(t, "empty.bin", nil)
	f, err := OpenBufferedFile(path)
	if err != nil {
		t.Fatalf("OpenBufferedFile: %v", err)
	}
	defer f.Close()

	if _, err := f.ReadByte(); !errors.Is(err, io.EOF) {
		t.Fatalf("ReadByte error = %v, want io.EOF", err)
	}
	if _, err := Peek(f); !errors.Is(err, io.EOF) {
		t.Fatalf("Peek error = %v, want io.EOF", err)
	}
	if _, err := f.Read(make([]byte, 6)); !errors.Is(err, io.EOF) {
		t.Fatalf("Read error = %v, want io.EOF", err)
	}
	if err := SeekTo(f, 6); err != nil {
		t.Fatalf("SeekTo(6): %v", err)
	}
	wantPosition(t, f, 0)
	if eof, err := f.IsEOF(); err != nil || !eof {
		t.Fatalf("IsEOF = %v, %v; want true, nil", eof, err)
	}
}

func TestBufferedFileView(t *testing.T) {
	f := openSampleFile(t)
	defer f.Close()

	view, err := f.CreateView(3, 5)
	if err != nil {
		t.Fatalf("CreateView: %v", err)
	}
	defer view.Close()

	wantPosition(t, view, 0)
	wantByte(t, view, 3)
	wantByte(t, view, 4)
	wantByte(t, view, 5)
	wantPosition(t, view, 3)

	// the view must not disturb the cursor of the file it was created from
	wantPosition(t, f, 0)
}

func TestBufferedFileReadFully(t *testing.T) {
	f := openSampleFile(t)
	defer f.Close()

	got := make([]byte, len(sampleBytes()))
	if err := ReadFully(f, got); err != nil {
		t.Fatalf("ReadFully: %v", err)
	}
	if !bytes.Equal(got, sampleBytes()) {
		t.Fatalf("ReadFully = %v, want %v", got, sampleBytes())
	}
	if eof, err := f.IsEOF(); err != nil || !eof {
		t.Fatalf("IsEOF = %v, %v; want true, nil", eof, err)
	}
}

func TestBufferedFileReadFullyEOF(t *testing.T) {
	f := openSampleFile(t)
	defer f.Close()

	if err := ReadFully(f, make([]byte, len(sampleBytes())+1)); !errors.Is(err, ErrPrematureEOF) {
		t.Fatalf("ReadFully error = %v, want ErrPrematureEOF", err)
	}
}

func TestBufferedFileReadFullyNothing(t *testing.T) {
	f := openSampleFile(t)
	defer f.Close()

	wantPosition(t, f, 0)
	if err := ReadFully(f, nil); err != nil {
		t.Fatalf("ReadFully(nil): %v", err)
	}
	wantPosition(t, f, 0)
}

// TestBufferedFileAcrossPages reads a file several pages long, which is what
// exercises the page cache and the page boundary handling in Read.
func TestBufferedFileAcrossPages(t *testing.T) {
	data := make([]byte, pageSize*3+123)
	for i := range data {
		data[i] = byte(i * 31)
	}
	path := writeTempFile(t, "pages.bin", data)
	f, err := OpenBufferedFile(path)
	if err != nil {
		t.Fatalf("OpenBufferedFile: %v", err)
	}
	defer f.Close()

	wantLength(t, f, int64(len(data)))

	got := make([]byte, len(data))
	if err := ReadFully(f, got); err != nil {
		t.Fatalf("ReadFully: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatal("data read back does not match the file")
	}

	// a read starting mid-page and crossing into the next one
	const start = pageSize - 10
	if err := SeekTo(f, start); err != nil {
		t.Fatalf("SeekTo: %v", err)
	}
	span := make([]byte, 40)
	if err := ReadFully(f, span); err != nil {
		t.Fatalf("ReadFully across page boundary: %v", err)
	}
	if !bytes.Equal(span, data[start:start+40]) {
		t.Fatal("bytes read across the page boundary do not match the file")
	}
}

// TestBufferedFileConcurrentViews checks that views handed out by CreateView
// carry independent cursors, which is what the per-thread copies in the Java
// version exist to guarantee.
func TestBufferedFileConcurrentViews(t *testing.T) {
	data := make([]byte, 256)
	for i := range data {
		data[i] = byte(i)
	}
	path := writeTempFile(t, "views.bin", data)
	f, err := OpenBufferedFile(path)
	if err != nil {
		t.Fatalf("OpenBufferedFile: %v", err)
	}
	defer f.Close()

	first, err := f.CreateView(0, 128)
	if err != nil {
		t.Fatalf("CreateView: %v", err)
	}
	second, err := f.CreateView(128, 128)
	if err != nil {
		t.Fatalf("CreateView: %v", err)
	}

	wantByte(t, first, 0)
	wantByte(t, second, 128)
	wantByte(t, first, 1)
	wantByte(t, second, 129)
	wantPosition(t, first, 2)
	wantPosition(t, second, 2)
}
