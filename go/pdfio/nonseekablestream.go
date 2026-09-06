package pdfio

// A random access read over a stream that cannot seek, which keeps three
// buffers so that a short rewind still works.
//
// Port of org.apache.pdfbox.io.NonSeekableRandomAccessReadInputStream.

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
)

// nonSeekableBufferSize is the size of each of the three buffers.
const nonSeekableBufferSize = 4096

// The three buffers the stream navigates with.
const (
	bufCurrent = 0
	bufLast    = 1
	bufNext    = 2
)

// ErrSeekNotSupported is returned by a source that cannot seek.
//
// Java raises IOException("<class>.seek isn't supported.").
var ErrSeekNotSupported = errors.New("pdfio: seek is not supported by this source")

// ErrNotEnoughToRewind is returned by a rewind that reaches further back than
// the buffers hold.
//
// Java raises IOException("not enough bytes available to perform the rewind
// operation").
var ErrNotEnoughToRewind = errors.New(
	"pdfio: not enough bytes available to perform the rewind operation")

// NonSeekableRead reads a stream that cannot seek, holding the current, the
// previous and the next buffer so that a rewind within them still works.
//
// Not safe for concurrent use, which is Java: the class synchronizes nothing,
// and the three buffers and the cursor into them are one shared position. Two
// readers would interleave into the same buffer.
type NonSeekableRead struct {
	// position is the current position within the stream.
	position int64

	// currentBufferPointer is the cursor within the current buffer.
	currentBufferPointer int

	// size is how much of the stream has been read so far.
	size int64

	// is is the source stream.
	is io.ReadCloser

	// buffers holds the current, the previous and the next buffer.
	buffers [3][]byte

	// bufferBytes holds how many bytes each buffer carries, -1 for none.
	bufferBytes [3]int

	isClosed bool
	isEOF    bool
}

var (
	_ RandomAccessRead = (*NonSeekableRead)(nil)
	_ rewinder         = (*NonSeekableRead)(nil)
	_ skipper          = (*NonSeekableRead)(nil)
	_ fullReader       = (*NonSeekableRead)(nil)
	_ availabler       = (*NonSeekableRead)(nil)
)

// NewNonSeekableRead returns a random access read over the given stream.
//
// Port of the NonSeekableRandomAccessReadInputStream(InputStream) constructor.
func NewNonSeekableRead(inputStream io.ReadCloser) *NonSeekableRead {
	return &NonSeekableRead{
		is:          inputStream,
		buffers:     [3][]byte{make([]byte, nonSeekableBufferSize), make([]byte, nonSeekableBufferSize), make([]byte, nonSeekableBufferSize)},
		bufferBytes: [3]int{-1, -1, -1},
	}
}

// Close closes the underlying stream.
//
// Java sets isClosed after is.close() returns, so a close that fails leaves
// the source open rather than half closed. The port does the same.
func (n *NonSeekableRead) Close() error {
	if err := n.is.Close(); err != nil {
		return err
	}
	n.isClosed = true
	return nil
}

// Seek is not supported: the source cannot go backwards past its buffers.
func (n *NonSeekableRead) Seek(offset int64, whence int) (int64, error) {
	return 0, fmt.Errorf("%w: NonSeekableRead.Seek isn't supported", ErrSeekNotSupported)
}

// Skip advances by the given number of bytes, by reading and discarding them.
//
// Port of the overridden skip(int); the source cannot seek, so the bytes have
// to go through the buffers.
func (n *NonSeekableRead) Skip(length int64) error {
	bufferLength := length
	if bufferLength > nonSeekableBufferSize {
		bufferLength = nonSeekableBufferSize
	}
	skipBuffer := make([]byte, bufferLength)
	remaining := length
	for remaining > 0 {
		want := remaining
		if want > int64(len(skipBuffer)) {
			want = int64(len(skipBuffer))
		}
		bytesRead, err := n.Read(skipBuffer[:want])
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		remaining -= int64(bytesRead)
	}
	return nil
}

// Position returns the current position within the stream.
func (n *NonSeekableRead) Position() (int64, error) {
	if err := n.checkClosed(); err != nil {
		return 0, err
	}
	return n.position, nil
}

// ReadByte reads one byte, reporting io.EOF at the end where Java's read()
// answers -1.
func (n *NonSeekableRead) ReadByte() (byte, error) {
	if err := n.checkClosed(); err != nil {
		return 0, err
	}
	eof, err := n.IsEOF()
	if err != nil {
		return 0, err
	}
	if eof {
		return 0, io.EOF
	}
	if n.currentBufferPointer >= n.bufferBytes[bufCurrent] {
		fetched, err := n.fetch()
		if err != nil {
			return 0, err
		}
		if !fetched {
			n.isEOF = true
			return 0, io.EOF
		}
	}
	n.position++
	c := n.buffers[bufCurrent][n.currentBufferPointer]
	n.currentBufferPointer++
	return c, nil
}

// Read fills p, reporting io.EOF at the end where Java's read(byte[], int, int)
// answers -1.
func (n *NonSeekableRead) Read(p []byte) (int, error) {
	if err := n.checkClosed(); err != nil {
		return 0, err
	}
	// Java validates the offset and the length against the array; Go carries
	// both in the slice, so there is nothing left to check but the empty read.
	if len(p) == 0 {
		return 0, nil
	}
	eof, err := n.IsEOF()
	if err != nil {
		return 0, err
	}
	if eof {
		return 0, io.EOF
	}

	numberOfBytesRead := 0
	for numberOfBytesRead < len(p) {
		available := n.bufferBytes[bufCurrent] - n.currentBufferPointer
		if available > 0 {
			bytes2Copy := len(p) - numberOfBytesRead
			if bytes2Copy > available {
				bytes2Copy = available
			}
			copy(p[numberOfBytesRead:], n.buffers[bufCurrent][n.currentBufferPointer:n.currentBufferPointer+bytes2Copy])
			n.currentBufferPointer += bytes2Copy
			n.position += int64(bytes2Copy)
			numberOfBytesRead += bytes2Copy
			continue
		}
		fetched, err := n.fetch()
		if err != nil {
			return numberOfBytesRead, err
		}
		if !fetched {
			n.isEOF = true
			break
		}
	}
	if numberOfBytesRead > 0 {
		return numberOfBytesRead, nil
	}
	return 0, io.EOF
}

// ReadFully reads len(p) bytes, looping until the buffer is full.
//
// Port of the overridden readFully, which the class declares because the value
// Length answers is not reliable for a stream.
func (n *NonSeekableRead) ReadFully(p []byte) error {
	if err := n.checkClosed(); err != nil {
		return err
	}
	bytesReadTotal := 0
	for bytesReadTotal < len(p) {
		bytesReadNow, err := n.Read(p[bytesReadTotal:])
		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}
		if bytesReadNow <= 0 {
			return fmt.Errorf("%w: EOF, should have been detected earlier",
				ErrPrematureEOF)
		}
		bytesReadTotal += bytesReadNow
	}
	return nil
}

// switchBuffers swaps two of the three buffers, and their byte counts with
// them.
func (n *NonSeekableRead) switchBuffers(firstBuffer, secondBuffer int) {
	n.buffers[firstBuffer], n.buffers[secondBuffer] =
		n.buffers[secondBuffer], n.buffers[firstBuffer]
	n.bufferBytes[firstBuffer], n.bufferBytes[secondBuffer] =
		n.bufferBytes[secondBuffer], n.bufferBytes[firstBuffer]
}

// fetch fills the current buffer, reporting false at the end of the stream.
func (n *NonSeekableRead) fetch() (bool, error) {
	if err := n.checkClosed(); err != nil {
		return false, err
	}
	n.currentBufferPointer = 0
	if n.bufferBytes[bufNext] > -1 {
		// there is a next buffer from a former rewind operation
		// switch to the next buffer and don't read any new data
		n.switchBuffers(bufCurrent, bufLast)
		n.switchBuffers(bufCurrent, bufNext)
		// reset next buffer
		n.bufferBytes[bufNext] = -1
		return true, nil
	}

	if n.bufferBytes[bufLast] == nonSeekableBufferSize && n.bufferBytes[bufCurrent] > 0 &&
		n.bufferBytes[bufCurrent] < nonSeekableBufferSize {
		// Likely EOF, we're risking losing the previous (full) buffer and get
		// an out of bounds. Fill LAST with as much as possible data of LAST and
		// CURRENT.
		copy(n.buffers[bufLast][0:], n.buffers[bufLast][n.bufferBytes[bufCurrent]:nonSeekableBufferSize])
		copy(n.buffers[bufLast][nonSeekableBufferSize-n.bufferBytes[bufCurrent]:],
			n.buffers[bufCurrent][:n.bufferBytes[bufCurrent]])
		n.bufferBytes[bufLast] = nonSeekableBufferSize
	} else {
		n.switchBuffers(bufCurrent, bufLast)
	}

	// java.io.InputStream.read(byte[]) blocks until it holds at least one
	// byte or the stream has ended, so it never answers 0. An io.Reader may,
	// and 0 is not the end -- taking it for one would truncate the stream
	// silently. Read on until the source says one of the two things Java can.
	var read int
	var err error
	for {
		read, err = n.is.Read(n.buffers[bufCurrent])
		if read > 0 || err != nil {
			break
		}
	}
	if err != nil && !errors.Is(err, io.EOF) {
		// Java logs this and rethrows, having marked the source at its end.
		slog.Warn("pdfio: premature end of stream, some data could be read",
			"err", err)
		n.isEOF = true
		return false, err
	}
	n.bufferBytes[bufCurrent] = read
	if n.bufferBytes[bufCurrent] <= 0 {
		n.bufferBytes[bufCurrent] = -1
		return false, nil
	}
	n.size += int64(n.bufferBytes[bufCurrent])
	return true, nil
}

// Available returns how many bytes can be read without blocking, which may be
// 0, and is 0 once the end of the stream is reached.
//
// Port of the overridden available().
func (n *NonSeekableRead) Available() (int, error) {
	if err := n.checkClosed(); err != nil {
		return 0, err
	}
	buffered := n.bufferBytes[bufCurrent] - n.currentBufferPointer
	if buffered < 0 {
		buffered = 0
	}
	sourceLeft, err := sourceAvailable(n.is)
	if err != nil {
		return 0, err
	}
	return buffered + sourceLeft, nil
}

// Length returns the bytes read so far plus what the source says it still
// holds.
//
// Port of the overridden length().
func (n *NonSeekableRead) Length() (int64, error) {
	if err := n.checkClosed(); err != nil {
		return 0, err
	}
	sourceLeft, err := sourceAvailable(n.is)
	if err != nil {
		return 0, err
	}
	return n.size + int64(sourceLeft), nil
}

// sourceAvailable is java.io.InputStream.available() for a Go reader, which has
// no such method.
//
// Java's contract is an estimate of what can be read without blocking, and both
// available() and length() are built on it -- so a port that always answered 0
// would fail the cases that ask for the length of a stream before reading it.
// The three shapes below cover every source PDFBox hands this class: a
// ByteArrayInputStream is a bytes.Reader, and Files.newInputStream is an
// os.File. Anything else answers 0, which is what InputStream itself answers by
// default.
func sourceAvailable(r io.Reader) (int, error) {
	switch source := r.(type) {
	case interface{ Available() (int, error) }:
		return source.Available()
	case interface{ Len() int }:
		return source.Len(), nil
	case *os.File:
		info, err := source.Stat()
		if err != nil {
			return 0, err
		}
		offset, err := source.Seek(0, io.SeekCurrent)
		if err != nil {
			return 0, err
		}
		left := info.Size() - offset
		if left < 0 {
			return 0, nil
		}
		return int(left), nil
	}
	return 0, nil
}

// Rewind goes back the given number of bytes, as far as the buffers reach.
//
// Port of the overridden rewind(int).
func (n *NonSeekableRead) Rewind(bytes int64) error {
	// check if the rewind operation is limited to the current buffer
	if int64(n.currentBufferPointer) >= bytes {
		n.currentBufferPointer -= int(bytes)
		n.position -= bytes
		n.isEOF = false
		return nil
	}
	if n.bufferBytes[bufLast] > 0 &&
		bytes-int64(n.currentBufferPointer) <= int64(n.bufferBytes[bufLast]) {
		// there is a former buffer
		remainingBytesToRewind := bytes - int64(n.currentBufferPointer)
		// save the current as next buffer
		n.switchBuffers(bufCurrent, bufNext)
		// make the former buffer the current one
		n.switchBuffers(bufCurrent, bufLast)
		// reset last buffer
		n.bufferBytes[bufLast] = -1
		n.currentBufferPointer = n.bufferBytes[bufCurrent] - int(remainingBytesToRewind)
		n.position -= bytes
		n.isEOF = false
		return nil
	}
	// there aren't enough bytes left in the buffers to perform the rewind
	// operation
	return ErrNotEnoughToRewind
}

// checkClosed reports whether the source has already been closed.
func (n *NonSeekableRead) checkClosed() error {
	if n.isClosed {
		return ErrClosed
	}
	return nil
}

// IsClosed reports whether the source has been closed.
func (n *NonSeekableRead) IsClosed() bool { return n.isClosed }

// IsEOF reports whether the end of the stream has been reached.
func (n *NonSeekableRead) IsEOF() (bool, error) {
	if err := n.checkClosed(); err != nil {
		return false, err
	}
	return n.isEOF, nil
}

// CreateView is not supported: a stream that cannot seek cannot make an
// independent cursor over itself.
func (n *NonSeekableRead) CreateView(start, length int64) (RandomAccessRead, error) {
	return nil, fmt.Errorf("%w: NonSeekableRead.CreateView isn't supported",
		ErrViewNotSupported)
}
