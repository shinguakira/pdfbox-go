package pdfio

// Port of org.apache.pdfbox.io.RandomAccessReadMemoryMappedFileTest.

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// ioFixture is where the Java test resources for this package are, which the
// port reads rather than copying.
const ioFixture = "../../io/src/test/resources/org/apache/pdfbox/io/"

// openMappedFixture maps one of the Java test resources.
func openMappedFixture(t *testing.T, name string) *MappedFile {
	t.Helper()
	source, err := NewMappedFile(ioFixture + name)
	if err != nil {
		t.Fatalf("NewMappedFile(%s): %v", name, err)
	}
	return source
}

func TestMappedFilePositionSkip(t *testing.T) {
	source := openMappedFixture(t, "RandomAccessReadFile1.txt")
	defer source.Close()

	wantPosition(t, source, 0)
	noError(t, "Skip", Skip(source, 5))
	wantByteOrEOF(t, source, '5')
	wantPosition(t, source, 6)
}

func TestMappedFilePathConstructor(t *testing.T) {
	source := openMappedFixture(t, "RandomAccessReadFile1.txt")
	defer source.Close()

	wantLength(t, source, 130)
}

func TestMappedFilePositionRead(t *testing.T) {
	source := openMappedFixture(t, "RandomAccessReadFile1.txt")

	wantPosition(t, source, 0)
	wantByteOrEOF(t, source, '0')
	wantByteOrEOF(t, source, '1')
	wantByteOrEOF(t, source, '2')
	wantPosition(t, source, 3)
	if source.IsClosed() {
		t.Error("IsClosed() = true before Close, want false")
	}
	noError(t, "Close", source.Close())
	if !source.IsClosed() {
		t.Error("IsClosed() = false after Close, want true")
	}
}

func TestMappedFileSeekEOF(t *testing.T) {
	source := openMappedFixture(t, "RandomAccessReadFile1.txt")

	noError(t, "Seek", SeekTo(source, 3))
	wantPosition(t, source, 3)
	if err := SeekTo(source, -1); !errors.Is(err, ErrInvalidPosition) {
		t.Errorf("Seek(-1) = %v, want ErrInvalidPosition", err)
	}
	eof, err := source.IsEOF()
	noError(t, "IsEOF", err)
	if eof {
		t.Error("IsEOF() = true in the middle of the file, want false")
	}

	length, err := source.Length()
	noError(t, "Length", err)
	noError(t, "Seek", SeekTo(source, length))
	eof, err = source.IsEOF()
	noError(t, "IsEOF", err)
	if !eof {
		t.Error("IsEOF() = false at the end, want true")
	}
	wantByteOrEOF(t, source, -1)
	if _, err := source.Read(make([]byte, 1)); !errors.Is(err, io.EOF) {
		t.Errorf("Read at the end = %v, want io.EOF", err)
	}

	noError(t, "Close", source.Close())
	if _, err := source.ReadByte(); !errors.Is(err, ErrClosed) {
		t.Errorf("ReadByte on a closed source = %v, want ErrClosed", err)
	}
}

func TestMappedFilePositionReadBytes(t *testing.T) {
	source := openMappedFixture(t, "RandomAccessReadFile1.txt")
	defer source.Close()

	wantPosition(t, source, 0)
	buffer := make([]byte, 4)
	if _, err := source.Read(buffer); err != nil {
		t.Fatalf("Read: %v", err)
	}
	if buffer[0] != '0' || buffer[3] != '3' {
		t.Errorf("buffer = %q, want it to start '0' and end '3'", buffer)
	}
	wantPosition(t, source, 4)

	// Java's read(buffer, 1, 2); Go carries the offset and the length in the
	// slice it is handed.
	if _, err := source.Read(buffer[1:3]); err != nil {
		t.Fatalf("Read: %v", err)
	}
	for i, want := range []byte{'0', '4', '5', '3'} {
		if buffer[i] != want {
			t.Errorf("buffer[%d] = %q, want %q", i, buffer[i], want)
		}
	}
	wantPosition(t, source, 6)
}

func TestMappedFilePositionPeek(t *testing.T) {
	source := openMappedFixture(t, "RandomAccessReadFile1.txt")
	defer source.Close()

	wantPosition(t, source, 0)
	noError(t, "Skip", Skip(source, 6))
	wantPosition(t, source, 6)

	peeked, err := Peek(source)
	noError(t, "Peek", err)
	if peeked != '6' {
		t.Errorf("Peek() = %q, want '6'", peeked)
	}
	wantPosition(t, source, 6)
}

func TestMappedFilePositionUnreadBytes(t *testing.T) {
	source := openMappedFixture(t, "RandomAccessReadFile1.txt")
	defer source.Close()

	wantPosition(t, source, 0)
	wantByteOrEOF(t, source, '0')
	wantByteOrEOF(t, source, '1')

	readBytes := make([]byte, 6)
	n, err := source.Read(readBytes)
	noError(t, "Read", err)
	if n != len(readBytes) {
		t.Fatalf("Read read %d bytes, want %d", n, len(readBytes))
	}
	wantPosition(t, source, 8)
	noError(t, "Rewind", Rewind(source, int64(len(readBytes))))
	wantPosition(t, source, 2)
	wantByteOrEOF(t, source, '2')
	wantPosition(t, source, 3)

	if _, err := source.Read(readBytes[2:6]); err != nil {
		t.Fatalf("Read: %v", err)
	}
	wantPosition(t, source, 7)
	noError(t, "Rewind", Rewind(source, 4))
	wantPosition(t, source, 3)
}

func TestMappedFileEmptyBuffer(t *testing.T) {
	source := openMappedFixture(t, "RandomAccessReadEmptyFile.txt")
	defer source.Close()

	wantByteOrEOF(t, source, -1)
	if _, err := Peek(source); !errors.Is(err, io.EOF) {
		t.Errorf("Peek = %v, want io.EOF", err)
	}
	if _, err := source.Read(make([]byte, 6)); !errors.Is(err, io.EOF) {
		t.Errorf("Read = %v, want io.EOF", err)
	}
	noError(t, "Seek(0)", SeekTo(source, 0))
	wantPosition(t, source, 0)
	// jumping beyond the end parks the cursor at the end, which for an empty
	// file is still zero
	noError(t, "Seek(6)", SeekTo(source, 6))
	wantPosition(t, source, 0)
	eof, err := source.IsEOF()
	noError(t, "IsEOF", err)
	if !eof {
		t.Error("IsEOF() = false on an empty file, want true")
	}
	if err := Rewind(source, 3); !errors.Is(err, ErrInvalidPosition) {
		t.Errorf("Rewind = %v, want ErrInvalidPosition", err)
	}
}

// TestMappedFileUnmapping is the case for the unmapping trouble Windows has,
// JDK-4724038: the file must be deletable once the source is closed.
func TestMappedFileUnmapping(t *testing.T) {
	tempFile := filepath.Join(t.TempDir(), "PDFBOX.txt")
	noError(t, "WriteFile", os.WriteFile(tempFile, []byte("Apache PDFBox test"), 0o600))

	source, err := NewMappedFile(tempFile)
	noError(t, "NewMappedFile", err)
	wantByteOrEOF(t, source, 65)
	noError(t, "Close", source.Close())

	noError(t, "Remove", os.Remove(tempFile))
}

func TestMappedFileView(t *testing.T) {
	source := openMappedFixture(t, "RandomAccessReadFile1.txt")
	defer source.Close()

	view, err := source.CreateView(3, 10)
	noError(t, "CreateView", err)
	defer view.Close()

	wantPosition(t, view, 0)
	wantByteOrEOF(t, view, '3')
	wantByteOrEOF(t, view, '4')
	wantByteOrEOF(t, view, '5')
	wantPosition(t, view, 3)
}

// TestMappedFileViewOfAClosedSource pins the one deliberate difference from
// Java in this file.
//
// RandomAccessReadMemoryMappedFile.createView calls no checkClosed and does not
// declare throws IOException, so on a closed source it reaches through the
// released buffer. Read out of the running Java, JDK 17:
//
//	mapped createView on closed:       java.lang.NullPointerException: Cannot
//	    invoke "java.nio.ByteBuffer.duplicate()" because
//	    "<parameter1>.mappedByteBuffer" is null
//	bufferedfile createView on closed: java.io.IOException:
//	    org.apache.pdfbox.io.RandomAccessReadBufferedFile already closed
//
// The port answers ErrClosed, which is what the sibling source answers and what
// every other method on a closed MappedFile answers. See
// migration/JAVA-BUGS.md entry 65.
func TestMappedFileViewOfAClosedSource(t *testing.T) {
	source := openMappedFixture(t, "RandomAccessReadFile1.txt")
	noError(t, "Close", source.Close())

	view, err := source.CreateView(0, 10)
	if !errors.Is(err, ErrClosed) {
		t.Errorf("CreateView on a closed source = %v, want ErrClosed", err)
	}
	if view != nil {
		t.Error("CreateView on a closed source returned a view, want none")
	}
}
