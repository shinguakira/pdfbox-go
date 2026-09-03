package pdfio

import (
	"errors"
	"io"
	"math"
)

// RandomAccessRead allows random access read operations over a source of bytes.
//
// Port of org.apache.pdfbox.io.RandomAccessRead. The Java interface signals the
// end of the source by returning -1 from read(); the Go port follows the
// standard library instead and reports io.EOF, so implementations compose with
// bufio, io.Copy and the rest of the stdlib.
type RandomAccessRead interface {
	io.Reader
	io.ByteReader
	io.Seeker
	io.Closer

	// Position returns the offset of the next byte to be read.
	Position() (int64, error)

	// Length returns the total number of bytes available in the source.
	Length() (int64, error)

	// IsClosed reports whether the source has been closed.
	IsClosed() bool

	// IsEOF reports whether the cursor sits at or past the end of the source.
	IsEOF() (bool, error)

	// CreateView returns an independent read cursor clipped to the section
	// starting at start and running for length bytes. Closing the view does
	// not close the source it was created from.
	CreateView(start, length int64) (RandomAccessRead, error)
}

// RandomAccessWrite allows random access write operations.
//
// Port of org.apache.pdfbox.io.RandomAccessWrite. Java declares three write
// overloads; Go needs only io.Writer plus the single-byte form.
type RandomAccessWrite interface {
	io.Writer
	io.ByteWriter
	io.Closer

	// Clear discards all data held by the destination.
	Clear() error
}

// RandomAccess is a source that can be both read and written, allowing data to
// be held in memory or spilled to a scratch file on disk.
//
// Port of org.apache.pdfbox.io.RandomAccess.
type RandomAccess interface {
	RandomAccessRead
	RandomAccessWrite
}

// SeekTo moves the cursor of r to an absolute position. It is the direct
// equivalent of the Java seek(long) method, which has no whence parameter.
func SeekTo(r io.Seeker, position int64) error {
	_, err := r.Seek(position, io.SeekStart)
	return err
}

// Available returns an estimate of the number of bytes that can still be read,
// clamped to the range of an int as the Java version is.
func Available(r RandomAccessRead) (int, error) {
	length, err := r.Length()
	if err != nil {
		return 0, err
	}
	position, err := r.Position()
	if err != nil {
		return 0, err
	}
	remaining := length - position
	if remaining <= 0 {
		return 0, nil
	}
	if remaining > math.MaxInt32 {
		return math.MaxInt32, nil
	}
	return int(remaining), nil
}

// Peek returns the next byte without advancing the cursor. It reports io.EOF at
// the end of the source, where the Java version returns -1.
func Peek(r RandomAccessRead) (byte, error) {
	b, err := r.ReadByte()
	if err != nil {
		return 0, err
	}
	if err := Rewind(r, 1); err != nil {
		return 0, err
	}
	return b, nil
}

// Rewind seeks backwards by the given number of bytes.
func Rewind(r RandomAccessRead, bytes int64) error {
	position, err := r.Position()
	if err != nil {
		return err
	}
	return SeekTo(r, position-bytes)
}

// Skip advances the cursor by the given number of bytes. As in Java, seeking
// past the end of the source is allowed.
func Skip(r RandomAccessRead, length int64) error {
	position, err := r.Position()
	if err != nil {
		return err
	}
	return SeekTo(r, position+length)
}

// ReadFully reads len(p) bytes into p, looping until the buffer is full. It
// returns ErrPrematureEOF if the source holds fewer bytes than requested.
func ReadFully(r RandomAccessRead, p []byte) error {
	length, err := r.Length()
	if err != nil {
		return err
	}
	position, err := r.Position()
	if err != nil {
		return err
	}
	if length-position < int64(len(p)) {
		return ErrPrematureEOF
	}
	total := 0
	for total < len(p) {
		n, err := r.Read(p[total:])
		if n > 0 {
			total += n
			continue
		}
		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}
		return ErrPrematureEOF
	}
	return nil
}
