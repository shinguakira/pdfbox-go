package pdfio

// Port of org.apache.pdfbox.io.NonSeekableRandomAccessReadInputStreamTest.

import (
	"bytes"
	"errors"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

// byteArrayInputStream is what the Java test hands the source every time:
// a stream over a byte array that closes and can say how much is left.
//
// io.NopCloser will not do, because it hides the Len of the reader it wraps and
// so cannot answer available().
type byteArrayInputStream struct{ *bytes.Reader }

// Close does nothing, as ByteArrayInputStream.close() does nothing.
func (byteArrayInputStream) Close() error { return nil }

// nonSeekableOver returns a source over the given bytes, which is Java's
// `new NonSeekableRandomAccessReadInputStream(new ByteArrayInputStream(b))`.
func nonSeekableOver(b []byte) *NonSeekableRead {
	return NewNonSeekableRead(byteArrayInputStream{bytes.NewReader(b)})
}

// wantByteOrEOF is assertEquals(n, rar.read()) for the single-byte read, where
// a negative want is the -1 Java answers at the end of the stream and the port
// answers as io.EOF.
func wantByteOrEOF(t *testing.T, r RandomAccessRead, want int) {
	t.Helper()
	got, err := r.ReadByte()
	if want < 0 {
		if !errors.Is(err, io.EOF) {
			t.Errorf("ReadByte() = %d, %v, want io.EOF", got, err)
		}
		return
	}
	if err != nil {
		t.Fatalf("ReadByte: %v", err)
	}
	if int(got) != want {
		t.Errorf("ReadByte() = %d, want %d", got, want)
	}
}

func TestNonSeekablePositionSkip(t *testing.T) {
	source := nonSeekableOver([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
	defer source.Close()

	wantPosition(t, source, 0)
	noError(t, "Skip", Skip(source, 5))
	wantByteOrEOF(t, source, 5)
	wantPosition(t, source, 6)
}

func TestNonSeekablePositionRead(t *testing.T) {
	source := nonSeekableOver([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10})

	wantPosition(t, source, 0)
	wantByteOrEOF(t, source, 0)
	wantByteOrEOF(t, source, 1)
	wantByteOrEOF(t, source, 2)
	wantPosition(t, source, 3)
	if source.IsClosed() {
		t.Error("IsClosed() = true before Close, want false")
	}
	noError(t, "Close", source.Close())
	if !source.IsClosed() {
		t.Error("IsClosed() = false after Close, want true")
	}
}

func TestNonSeekableSeekEOF(t *testing.T) {
	source := nonSeekableOver([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
	defer source.Close()

	if err := SeekTo(source, 3); !errors.Is(err, ErrSeekNotSupported) {
		t.Errorf("Seek = %v, want ErrSeekNotSupported", err)
	}
}

func TestNonSeekablePositionReadBytes(t *testing.T) {
	source := nonSeekableOver([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
	defer source.Close()

	wantPosition(t, source, 0)
	buffer := make([]byte, 4)
	if _, err := source.Read(buffer); err != nil {
		t.Fatalf("Read: %v", err)
	}
	if buffer[0] != 0 || buffer[3] != 3 {
		t.Errorf("buffer = %v, want it to start 0 and end 3", buffer)
	}
	wantPosition(t, source, 4)

	// Java's read(buffer, 1, 2) fills two bytes from index one; Go carries the
	// offset and the length in the slice it is handed.
	if _, err := source.Read(buffer[1:3]); err != nil {
		t.Fatalf("Read: %v", err)
	}
	for i, want := range []byte{0, 4, 5, 3} {
		if buffer[i] != want {
			t.Errorf("buffer[%d] = %d, want %d", i, buffer[i], want)
		}
	}
	wantPosition(t, source, 6)
}

func TestNonSeekablePositionPeek(t *testing.T) {
	source := nonSeekableOver([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
	defer source.Close()

	wantPosition(t, source, 0)
	noError(t, "Skip", Skip(source, 6))
	wantPosition(t, source, 6)

	peeked, err := Peek(source)
	noError(t, "Peek", err)
	if peeked != 6 {
		t.Errorf("Peek() = %d, want 6", peeked)
	}
	wantPosition(t, source, 6)
}

func TestNonSeekablePositionUnreadBytes(t *testing.T) {
	source := nonSeekableOver([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
	defer source.Close()

	wantPosition(t, source, 0)
	wantByteOrEOF(t, source, 0)
	wantByteOrEOF(t, source, 1)

	readBytes := make([]byte, 6)
	n, err := source.Read(readBytes)
	noError(t, "Read", err)
	if n != len(readBytes) {
		t.Fatalf("Read read %d bytes, want %d", n, len(readBytes))
	}
	wantPosition(t, source, 8)
	noError(t, "Rewind", Rewind(source, int64(len(readBytes))))
	wantPosition(t, source, 2)
	wantByteOrEOF(t, source, 2)
	wantPosition(t, source, 3)

	if _, err := source.Read(readBytes[2:6]); err != nil {
		t.Fatalf("Read: %v", err)
	}
	wantPosition(t, source, 7)
	noError(t, "Rewind", Rewind(source, 4))
	wantPosition(t, source, 3)

	// PDFBOX-5965: check that it also works near EOF
	for _, want := range []int{3, 4, 5, 6, 7, 8, 9, 10, -1} {
		wantByteOrEOF(t, source, want)
	}
	eof, err := source.IsEOF()
	noError(t, "IsEOF", err)
	if !eof {
		t.Error("IsEOF() = false at the end, want true")
	}
	noError(t, "Rewind", Rewind(source, 4))
	eof, err = source.IsEOF()
	noError(t, "IsEOF", err)
	if eof {
		t.Error("IsEOF() = true after a rewind, want false")
	}
	for _, want := range []int{7, 8, 9, 10, -1} {
		wantByteOrEOF(t, source, want)
	}
}

func TestNonSeekableView(t *testing.T) {
	source := nonSeekableOver([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
	defer source.Close()

	if _, err := source.CreateView(3, 5); !errors.Is(err, ErrViewNotSupported) {
		t.Errorf("CreateView = %v, want ErrViewNotSupported", err)
	}
}

// createRandomData is the Java test's helper: a run of pseudo-random bytes
// mixed with runs of one repeated value.
func createRandomData() []byte {
	random := rand.New(rand.NewSource(rand.Int63()))
	numBytes := 10000 + random.Intn(20000)
	original := make([]byte, numBytes)
	upto := 0
	for upto < numBytes {
		left := numBytes - upto
		if random.Intn(2) == 0 || left < 2 {
			// Fill w/ pseudo-random bytes
			end := upto + min(left, 10+random.Intn(100))
			for upto < end {
				original[upto] = byte(random.Int())
				upto++
			}
			continue
		}
		// Fill w/ very predictable bytes
		end := upto + min(left, 2+random.Intn(10))
		value := byte(random.Intn(4))
		for upto < end {
			original[upto] = value
			upto++
		}
	}
	return original
}

func TestNonSeekableBufferSwitch(t *testing.T) {
	original := createRandomData()
	source := nonSeekableOver(original)
	defer source.Close()

	noError(t, "Skip", Skip(source, 4098))
	wantPosition(t, source, 4098)
	noError(t, "Rewind", Rewind(source, 4))
	wantPosition(t, source, 4094)
	wantByteOrEOF(t, source, int(original[4094]))
}

func TestNonSeekableRewindException(t *testing.T) {
	source := nonSeekableOver(createRandomData())
	defer source.Close()

	noError(t, "Skip", Skip(source, 10000))
	wantPosition(t, source, 10000)
	noError(t, "Rewind", Rewind(source, 4096))
	wantPosition(t, source, 5904)
	if err := Rewind(source, 4096); !errors.Is(err, ErrNotEnoughToRewind) {
		t.Errorf("Rewind = %v, want ErrNotEnoughToRewind", err)
	}
}

func TestNonSeekableRewindAcrossBuffers(t *testing.T) {
	const rewSize = 7
	const testVal = 123
	ba := make([]byte, 4096+5)
	ba[len(ba)-rewSize] = testVal
	source := nonSeekableOver(ba)
	defer source.Close()

	n, err := source.Read(make([]byte, len(ba)-rewSize))
	noError(t, "Read", err)
	if n != len(ba)-rewSize {
		t.Fatalf("Read read %d bytes, want %d", n, len(ba)-rewSize)
	}
	n, err = source.Read(make([]byte, rewSize))
	noError(t, "Read", err)
	if n != rewSize {
		t.Fatalf("Read read %d bytes, want %d", n, rewSize)
	}
	wantByteOrEOF(t, source, -1)
	eof, err := source.IsEOF()
	noError(t, "IsEOF", err)
	if !eof {
		t.Error("IsEOF() = false at the end, want true")
	}
	noError(t, "Rewind", Rewind(source, int64(n)))
	// went ArrayIndexOutOfBoundsException here
	wantByteOrEOF(t, source, testVal)
}

func TestNonSeekableRewindAcrossBuffers2(t *testing.T) {
	ba := make([]byte, 4096*2)
	ba[4095] = 1
	ba[4096] = 2
	ba[4097] = 3
	ba[4096*2-1] = 4
	source := nonSeekableOver(ba)
	defer source.Close()

	wantLength(t, source, 4096*2)
	n, err := source.Read(make([]byte, 4096+1))
	noError(t, "Read", err)
	wantLength(t, source, 4096*2)
	if n != 4096+1 {
		t.Fatalf("Read read %d bytes, want %d", n, 4096+1)
	}
	noError(t, "Rewind", Rewind(source, 2))
	wantByteOrEOF(t, source, 1)
	wantByteOrEOF(t, source, 2)
	wantByteOrEOF(t, source, 3)
	wantLength(t, source, 4096*2)

	buf := make([]byte, 4096)
	n, err = source.Read(buf)
	noError(t, "Read", err)
	if n != 4096-2 {
		t.Fatalf("Read read %d bytes, want %d", n, 4096-2)
	}
	if buf[n-1] != 4 {
		t.Errorf("buf[%d] = %d, want 4", n-1, buf[n-1])
	}
	wantByteOrEOF(t, source, -1)
	if _, err := source.Read(make([]byte, 1)); !errors.Is(err, io.EOF) {
		t.Errorf("Read at the end = %v, want io.EOF", err)
	}
}

func TestNonSeekableAccessClosed(t *testing.T) {
	source := nonSeekableOver([]byte{1})

	wantByteOrEOF(t, source, 1)
	wantByteOrEOF(t, source, -1)
	noError(t, "Close", source.Close())
	if _, err := source.ReadByte(); !errors.Is(err, ErrClosed) {
		t.Errorf("ReadByte on a closed source = %v, want ErrClosed", err)
	}
}

// TestNonSeekableClosedStreamMethods checks that every method needing an open
// stream reports the source is closed.
func TestNonSeekableClosedStreamMethods(t *testing.T) {
	source := nonSeekableOver([]byte{1, 2, 3})
	noError(t, "Close", source.Close())

	for _, c := range []struct {
		what string
		call func() error
	}{
		{"ReadByte", func() error { _, err := source.ReadByte(); return err }},
		{"Read", func() error { _, err := source.Read(make([]byte, 1)); return err }},
		{"ReadFully", func() error { return source.ReadFully(make([]byte, 1)) }},
		{"Position", func() error { _, err := source.Position(); return err }},
		{"Available", func() error { _, err := source.Available(); return err }},
		{"Length", func() error { _, err := source.Length(); return err }},
		{"IsEOF", func() error { _, err := source.IsEOF(); return err }},
	} {
		if err := c.call(); !errors.Is(err, ErrClosed) {
			t.Errorf("%s on a closed source = %v, want ErrClosed", c.what, err)
		}
	}
}

// TestNonSeekableReadBytesParameterValidation covers the one assertion of
// Java's parameter validation case that Go can express.
//
// The other three -- a null buffer, a negative offset and a negative length --
// cannot happen here: a Go slice carries its own bounds, so read(b, off, len)
// is Read(b[off:off+len]) and the slice expression is what would fail. Java
// raises NullPointerException and IndexOutOfBoundsException for them, which the
// port has no equivalent of and no way to reach.
func TestNonSeekableReadBytesParameterValidation(t *testing.T) {
	source := nonSeekableOver([]byte{0, 1, 2, 3, 4})
	defer source.Close()

	// length == 0 must return 0 immediately without advancing position
	n, err := source.Read(nil)
	noError(t, "Read", err)
	if n != 0 {
		t.Errorf("Read(nil) = %d, want 0", n)
	}
	wantPosition(t, source, 0)
}

// TestNonSeekableReadFully checks that ReadFully reads exactly what was asked
// for across a buffer boundary.
func TestNonSeekableReadFully(t *testing.T) {
	inputValues := make([]byte, 10)
	for i := range inputValues {
		inputValues[i] = byte(i)
	}
	source := nonSeekableOver(inputValues)
	defer source.Close()

	buf := make([]byte, 10)
	noError(t, "ReadFully", ReadFully(source, buf))
	for i := 0; i < 10; i++ {
		if buf[i] != byte(i) {
			t.Errorf("buf[%d] = %d, want %d", i, buf[i], i)
		}
	}
	wantPosition(t, source, 10)
}

// TestNonSeekableReadFullyEOF checks that ReadFully reports the stream ending
// before it had what was asked for.
func TestNonSeekableReadFullyEOF(t *testing.T) {
	source := nonSeekableOver([]byte{0, 1, 2})
	defer source.Close()

	if err := ReadFully(source, make([]byte, 10)); !errors.Is(err, ErrPrematureEOF) {
		t.Errorf("ReadFully = %v, want ErrPrematureEOF", err)
	}
}

// TestNonSeekableSkipPastEOF checks that Skip stops at the end without
// reporting a problem.
func TestNonSeekableSkipPastEOF(t *testing.T) {
	source := nonSeekableOver([]byte{0, 1, 2, 3, 4})
	defer source.Close()

	// skipping far beyond the end of the stream should not fail
	noError(t, "Skip", Skip(source, 100))
	wantPosition(t, source, 5)
	eof, err := source.IsEOF()
	noError(t, "IsEOF", err)
	if !eof {
		t.Error("IsEOF() = false past the end, want true")
	}
}

// TestNonSeekableAvailable checks that Available counts the bytes held in the
// buffers as well as those left in the source, and answers 0 at the end.
func TestNonSeekableAvailable(t *testing.T) {
	source := nonSeekableOver(make([]byte, 10))
	defer source.Close()

	// before any read, Available reflects the source since nothing is buffered
	// yet
	wantAvailable(t, source, 10)
	// read one byte: the fetch pulls all 10 bytes into the internal buffer, so
	// available = 9 buffered + 0 remaining in the underlying stream
	wantByteOrEOF(t, source, 0)
	wantAvailable(t, source, 9)
	// consume all remaining bytes
	for {
		if _, err := source.ReadByte(); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			t.Fatalf("ReadByte: %v", err)
		}
	}
	wantAvailable(t, source, 0)
}

// wantAvailable is assertEquals(n, rar.available()).
func wantAvailable(t *testing.T, r RandomAccessRead, want int) {
	t.Helper()
	got, err := Available(r)
	if err != nil {
		t.Fatalf("Available: %v", err)
	}
	if got != want {
		t.Errorf("Available() = %d, want %d", got, want)
	}
}

// TestNonSeekableLengthAfterFullConsumption checks that Length answers the
// exact total once the stream is fully read.
func TestNonSeekableLengthAfterFullConsumption(t *testing.T) {
	source := nonSeekableOver(make([]byte, 100))
	defer source.Close()

	for {
		if _, err := source.ReadByte(); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			t.Fatalf("ReadByte: %v", err)
		}
	}
	eof, err := source.IsEOF()
	noError(t, "IsEOF", err)
	if !eof {
		t.Error("IsEOF() = false at the end, want true")
	}
	wantLength(t, source, 100)
}

// TestNonSeekablePDFBOX5158 is the endless loop reading a stream whose length
// is a multiple of 4096 from a file. It does not fail over a byte slice, so the
// source has to be a real file.
func TestNonSeekablePDFBOX5158(t *testing.T) {
	path := filepath.Join(t.TempDir(), "len4096.pdf")
	noError(t, "WriteFile", os.WriteFile(path, make([]byte, 4096), 0o600))
	info, err := os.Stat(path)
	noError(t, "Stat", err)
	if info.Size() != 4096 {
		t.Fatalf("the file is %d bytes, want 4096", info.Size())
	}

	file, err := os.Open(path)
	noError(t, "Open", err)
	source := NewNonSeekableRead(file)
	defer source.Close()

	wantByteOrEOF(t, source, 0)
}

// TestNonSeekablePDFBOX5161 is the failure to read bytes after reading a
// multiple of 4096.
func TestNonSeekablePDFBOX5161(t *testing.T) {
	source := nonSeekableOver(make([]byte, 4099))
	defer source.Close()

	buf := make([]byte, 4096)
	bytesRead, err := source.Read(buf)
	noError(t, "Read", err)
	if bytesRead != 4096 {
		t.Errorf("Read read %d bytes, want 4096", bytesRead)
	}
	bytesRead, err = source.Read(buf[:3])
	noError(t, "Read", err)
	if bytesRead != 3 {
		t.Errorf("Read read %d bytes, want 3", bytesRead)
	}
}

// stutteringReader answers (0, nil) before every real read, which io.Reader
// explicitly permits and java.io.InputStream.read(byte[]) cannot do -- it
// blocks until it holds a byte or the stream has ended.
type stutteringReader struct {
	remaining []byte
	stutter   bool
}

func (s *stutteringReader) Read(p []byte) (int, error) {
	s.stutter = !s.stutter
	if s.stutter {
		return 0, nil
	}
	if len(s.remaining) == 0 {
		return 0, io.EOF
	}
	n := copy(p, s.remaining)
	s.remaining = s.remaining[n:]
	return n, nil
}

func (s *stutteringReader) Close() error { return nil }

// TestNonSeekableReadsPastAZeroByteRead checks a source that answers (0, nil)
// is not taken for an ended one.
//
// Java's fetch stores is.read(buffer) and treats <= 0 as the end, which is
// right for an InputStream: it blocks until it has a byte or the stream ends,
// so 0 never comes back for a non-empty array. Go's io.Reader may answer
// (0, nil) at any time, and a port that copied the <= 0 test literally would
// end the stream at the first such answer -- silently, and mid-content.
func TestNonSeekableReadsPastAZeroByteRead(t *testing.T) {
	// two buffers' worth, so fetch runs more than once
	content := createRandomData()
	source := NewNonSeekableRead(&stutteringReader{remaining: content})

	got := make([]byte, len(content))
	if err := ReadFully(source, got); err != nil {
		t.Fatalf("ReadFully: %v", err)
	}
	for i := range content {
		if got[i] != content[i] {
			t.Fatalf("byte %d = %d, want %d", i, got[i], content[i])
		}
	}

	length, err := source.Length()
	noError(t, "Length", err)
	if length != int64(len(content)) {
		t.Errorf("Length() = %d, want %d", length, len(content))
	}
}

// failingCloser is a source whose Close reports a problem, which is Java's
// is.close() throwing.
type failingCloser struct {
	*bytes.Reader
	err error
}

func (f failingCloser) Close() error { return f.err }

// TestNonSeekableCloseFailureLeavesTheSourceOpen pins the order of Java's
// close: is.close() first, isClosed = true after. An exception from the first
// leaves isClosed false, so the source still reads.
func TestNonSeekableCloseFailureLeavesTheSourceOpen(t *testing.T) {
	closeErr := errors.New("cannot close")
	source := NewNonSeekableRead(failingCloser{
		Reader: bytes.NewReader([]byte("0123456789")),
		err:    closeErr,
	})

	wantByteOrEOF(t, source, '0')

	if err := source.Close(); !errors.Is(err, closeErr) {
		t.Errorf("Close() = %v, want %v", err, closeErr)
	}
	if source.IsClosed() {
		t.Error("IsClosed() = true after a close that failed, want false")
	}
	// and it still reads, which is what "not closed" has to mean
	wantByteOrEOF(t, source, '1')
}

// bytesThenError hands back its content and the failure in one call, which
// io.Reader permits and java.io.InputStream cannot express.
type bytesThenError struct {
	content []byte
	err     error
}

func (b *bytesThenError) Read(p []byte) (int, error) {
	if len(b.content) == 0 {
		return 0, b.err
	}
	n := copy(p, b.content)
	b.content = b.content[n:]
	return n, b.err
}

func (b *bytesThenError) Close() error { return nil }

// TestNonSeekableKeepsBytesReturnedWithAnError checks that data handed back
// together with a failure still reaches the reader.
//
// Java's fetch assigns `bufferBytes[CURRENT] = is.read(...)` and only then can
// an exception happen, because InputStream.read either returns bytes or throws
// -- never both. Go's io.Reader may do both in one call, so a port that reports
// the error and drops the count loses bytes the source did deliver, and marks
// itself at the end so they can never be asked for again.
//
// What Java would have seen is two calls: one returning the bytes, and the next
// throwing. So the bytes come out first and the error follows.
func TestNonSeekableKeepsBytesReturnedWithAnError(t *testing.T) {
	sourceErr := errors.New("the disk went away")
	source := NewNonSeekableRead(&bytesThenError{
		content: []byte("0123456789"),
		err:     sourceErr,
	})

	got := make([]byte, 10)
	n, err := source.Read(got)
	if err != nil {
		t.Fatalf("Read = %v, want the ten bytes the source handed back", err)
	}
	if n != 10 || string(got) != "0123456789" {
		t.Fatalf("Read gave %d bytes %q, want 10 bytes \"0123456789\"", n, got)
	}

	position, err := source.Position()
	noError(t, "Position", err)
	if position != 10 {
		t.Errorf("Position() = %d, want 10", position)
	}

	// and the failure is reported next, not swallowed
	if _, err := source.Read(make([]byte, 4)); !errors.Is(err, sourceErr) {
		t.Errorf("the second Read = %v, want the source's own error", err)
	}
}
