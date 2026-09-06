package pdfio

// Port of org.apache.pdfbox.io.ScratchFileBufferTest.
//
// In package pdfio, as the Java test is in org.apache.pdfbox.io: ScratchFileBuffer
// is package-private there and unexported here.

import (
	"errors"
	"io"
	"testing"
)

// The two constants the Java test declares.
const (
	scratchPageSize   = 4096
	scratchIterations = 3
)

// noError fails the test where a step reported a problem.
func noError(t *testing.T, what string, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", what, err)
	}
}

// writeAll is Java's write(byte[]), which writes the whole slice or throws.
func writeAll(t *testing.T, w io.Writer, b []byte) {
	t.Helper()
	n, err := w.Write(b)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if n != len(b) {
		t.Fatalf("Write wrote %d bytes, want %d", n, len(b))
	}
}

// newMainMemoryScratchFile is the `new ScratchFile(MemoryUsageSetting
// .setupMainMemoryOnly())` every case opens with, closed at the end of the
// case the way the try-with-resources does.
func newMainMemoryScratchFile(t *testing.T) *ScratchFile {
	t.Helper()
	scratchFile, err := NewScratchFile(SetupMainMemoryOnly())
	if err != nil {
		t.Fatalf("NewScratchFile: %v", err)
	}
	t.Cleanup(func() {
		if err := scratchFile.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	return scratchFile
}

// createBuffer is scratchFile.createBuffer().
func createBuffer(t *testing.T, scratchFile *ScratchFile) RandomAccess {
	t.Helper()
	buffer, err := scratchFile.CreateBuffer()
	if err != nil {
		t.Fatalf("CreateBuffer: %v", err)
	}
	return buffer
}

// TestEOFBugInSeek is PDFBOX-4756: positions are correct when seeking, and no
// end-of-file error is raised beyond the last page.
func TestEOFBugInSeek(t *testing.T) {
	scratchFile := newMainMemoryScratchFile(t)
	scratchFileBuffer := createBuffer(t, scratchFile)
	bytes := make([]byte, scratchPageSize)

	for i := 0; i < scratchIterations; i++ {
		p0, err := scratchFileBuffer.Position()
		noError(t, "Position", err)
		writeAll(t, scratchFileBuffer, bytes)
		p1, err := scratchFileBuffer.Position()
		noError(t, "Position", err)
		if p1-p0 != scratchPageSize {
			t.Errorf("position advanced by %d, want %d", p1-p0, scratchPageSize)
		}
		writeAll(t, scratchFileBuffer, bytes)
		p2, err := scratchFileBuffer.Position()
		noError(t, "Position", err)
		if p2-p1 != scratchPageSize {
			t.Errorf("position advanced by %d, want %d", p2-p1, scratchPageSize)
		}
		noError(t, "Seek(0)", SeekTo(scratchFileBuffer, 0))
		noError(t, "Seek", SeekTo(scratchFileBuffer,
			int64(i)*2*scratchPageSize))
	}
}

func TestBufferLength(t *testing.T) {
	scratchFile := newMainMemoryScratchFile(t)
	buffer := createBuffer(t, scratchFile)
	writeAll(t, buffer, make([]byte, scratchPageSize))

	length, err := buffer.Length()
	noError(t, "Length", err)
	if length != scratchPageSize {
		t.Errorf("Length() = %d, want %d", length, scratchPageSize)
	}
}

func TestBufferSeek(t *testing.T) {
	scratchFile := newMainMemoryScratchFile(t)
	buffer := createBuffer(t, scratchFile)
	writeAll(t, buffer, make([]byte, scratchPageSize))

	// Java: assertThrows(IOException.class, () -> seek(-1))
	if err := SeekTo(buffer, -1); err == nil {
		t.Error("Seek(-1) reported nothing, want an error")
	}
	// Java: assertThrows(EOFException.class, () -> seek(PAGE_SIZE + 1))
	err := SeekTo(buffer, scratchPageSize+1)
	if !errors.Is(err, io.EOF) {
		t.Errorf("Seek past the end = %v, want io.EOF", err)
	}
}

func TestBufferEOF(t *testing.T) {
	scratchFile := newMainMemoryScratchFile(t)
	buffer := createBuffer(t, scratchFile)
	writeAll(t, buffer, make([]byte, scratchPageSize))

	noError(t, "Seek(0)", SeekTo(buffer, 0))
	eof, err := buffer.IsEOF()
	noError(t, "IsEOF", err)
	if eof {
		t.Error("IsEOF() = true at the start of the buffer, want false")
	}

	noError(t, "Seek", SeekTo(buffer, scratchPageSize))
	eof, err = buffer.IsEOF()
	noError(t, "IsEOF", err)
	if !eof {
		t.Error("IsEOF() = false at the end of the buffer, want true")
	}
}

func TestAlreadyClose(t *testing.T) {
	scratchFile := newMainMemoryScratchFile(t)
	buffer := createBuffer(t, scratchFile)
	writeAll(t, buffer, make([]byte, scratchPageSize))
	noError(t, "Close", buffer.Close())

	if err := SeekTo(buffer, 0); err == nil {
		t.Error("Seek on a closed buffer reported nothing, want an error")
	}
}

func TestBuffersClosed(t *testing.T) {
	scratchFile, err := NewScratchFile(SetupMainMemoryOnly())
	noError(t, "NewScratchFile", err)

	bytes := make([]byte, scratchPageSize)
	buffers := make([]RandomAccess, 4)
	for i := range buffers {
		buffers[i] = createBuffer(t, scratchFile)
		writeAll(t, buffers[i], bytes)
	}

	// close two of the buffers explicitly
	noError(t, "Close", buffers[0].Close())
	noError(t, "Close", buffers[2].Close())

	// check status
	for i, want := range []bool{true, false, true, false} {
		if got := buffers[i].IsClosed(); got != want {
			t.Errorf("buffers[%d].IsClosed() = %v, want %v", i, got, want)
		}
	}

	// closing ScratchFile shall close all remaining buffers which aren't
	// closed yet
	noError(t, "Close", scratchFile.Close())
	for _, i := range []int{1, 3} {
		if !buffers[i].IsClosed() {
			t.Errorf("buffers[%d].IsClosed() = false after the scratch file "+
				"was closed, want true", i)
		}
	}
}

func TestView(t *testing.T) {
	scratchFile := newMainMemoryScratchFile(t)
	buffer := createBuffer(t, scratchFile)
	writeAll(t, buffer, []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10})

	// Java raises UnsupportedOperationException, which the port answers as
	// ErrViewNotSupported -- the same value every other source that cannot
	// make a view answers.
	if _, err := buffer.CreateView(0, 10); !errors.Is(err, ErrViewNotSupported) {
		t.Errorf("CreateView = %v, want ErrViewNotSupported", err)
	}
}

// TestClearLeaksTheLastPage pins the page leak of ScratchFile.markPagesAsFree.
//
// Java walks `for (aIdx = off; aIdx < count; aIdx++)` rather than to
// off + count, and Clear passes off 1 with a count of pageCount - 1, so the
// last page of the buffer is never returned to the free pool. A buffer that
// filled its allowance cannot be refilled after Clear.
//
// Reproduced against JDK 17 before it was written: three pages of main memory,
// three pages written, cleared, and the second write fails with "Maximum
// allowed scratch file memory exceeded." See migration/JAVA-BUGS.md entry 62.
func TestClearLeaksTheLastPage(t *testing.T) {
	// Three pages of main memory and no more.
	scratchFile, err := NewScratchFile(SetupMainMemoryOnlyMax(3 * scratchPageSize))
	noError(t, "NewScratchFile", err)
	defer scratchFile.Close()

	buffer := createBuffer(t, scratchFile)
	page := make([]byte, scratchPageSize)
	for i := 0; i < 3; i++ {
		writeAll(t, buffer, page)
	}
	length, err := buffer.Length()
	noError(t, "Length", err)
	if length != 3*scratchPageSize {
		t.Fatalf("Length() = %d, want %d", length, 3*scratchPageSize)
	}

	noError(t, "Clear", buffer.Clear())

	// The third page was not freed, so only two of the three can be had again.
	var writeErr error
	for i := 0; i < 3 && writeErr == nil; i++ {
		_, writeErr = buffer.Write(page)
	}
	if writeErr == nil {
		t.Error("the buffer refilled after Clear, so every page was freed; " +
			"Java leaks the last one and the port is supposed to as well")
	}
}
