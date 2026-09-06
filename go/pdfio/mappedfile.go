package pdfio

// A random access read over a memory mapped file.
//
// Port of org.apache.pdfbox.io.RandomAccessReadMemoryMappedFile.
//
// Java opens a channel, asks its size, refuses a file over 2 GB -- leaking the
// channel on the way out, JAVA-BUGS entry 73 -- and maps the rest through the
// ByteBuffer that comes back. Go has no mapping in its standard library, so the
// mapping is golang.org/x/exp/mmap -- the decision migration/STATUS.md recorded
// as open and the track's B0 settles. It carries the per-platform work the port
// would otherwise have to write twice, once against syscall.Mmap and once
// against CreateFileMapping, and it is confined to this file, so replacing it
// later touches nothing else. Its cost is that x/exp carries no compatibility
// promise.

import (
	"fmt"
	"io"
	"math"
	"os"

	"golang.org/x/exp/mmap"
)

// MappedFile reads a file that has been mapped into memory.
//
// Not safe for concurrent use, which is Java: the position is Java's position
// in the ByteBuffer, and one buffer has one. CreateView is how a second cursor
// over the same mapping is had, and each view carries its own -- which is also
// what the port does for RandomAccessReadBuffer, see the slice 0 note in
// STATUS.md.
type MappedFile struct {
	// mapped is the mapping, and nil once this source is closed.
	mapped *mmap.ReaderAt

	// size is the size of the whole file.
	size int64

	// position is the cursor, which Java keeps in the ByteBuffer.
	position int64

	// isDuplicate says this source shares another's mapping and must not
	// unmap it, which is Java's null unmapper on a duplicate.
	isDuplicate bool
}

var _ RandomAccessRead = (*MappedFile)(nil)

// NewMappedFile returns a random access read over the file of the given name,
// mapped into memory.
//
// Port of the three constructors, which Java overloads on String, File and
// Path.
func NewMappedFile(filename string) (*MappedFile, error) {
	// The size is checked before the mapping is made, the order Java opens the
	// channel, asks its size and refuses in: mapping first would reserve the
	// address range for a file the type has already decided not to support.
	info, err := os.Stat(filename)
	if err != nil {
		return nil, err
	}
	// TODO only ints are allowed -> implement paging
	if info.Size() > math.MaxInt32 {
		return nil, fmt.Errorf(
			"pdfio.MappedFile doesn't yet support files bigger than %d", math.MaxInt32)
	}

	mapped, err := mmap.Open(filename)
	if err != nil {
		return nil, err
	}
	size := int64(mapped.Len())
	if size > math.MaxInt32 {
		// the file grew between the two calls
		mapped.Close()
		return nil, fmt.Errorf(
			"pdfio.MappedFile doesn't yet support files bigger than %d", math.MaxInt32)
	}
	return &MappedFile{mapped: mapped, size: size}, nil
}

// duplicate returns a second cursor over the same mapping, which is what a view
// reads through.
//
// Port of the private RandomAccessReadMemoryMappedFile(RandomAccessReadMemoryMappedFile)
// constructor, whose buffer is a duplicate() of its parent's and whose unmapper
// is null because unmapping a duplicate does not work.
func (m *MappedFile) duplicate() *MappedFile {
	return &MappedFile{mapped: m.mapped, size: m.size, isDuplicate: true}
}

// Close releases the mapping.
func (m *MappedFile) Close() error {
	if m.IsClosed() {
		return nil
	}
	mapped := m.mapped
	m.mapped = nil
	if m.isDuplicate {
		// unmap doesn't work on a duplicate; the parent owns the mapping.
		return nil
	}
	return mapped.Close()
}

// Seek moves the cursor. Jumping beyond the end of the file is allowed, and
// parks the cursor at the end.
func (m *MappedFile) Seek(offset int64, whence int) (int64, error) {
	if err := m.checkClosed(); err != nil {
		return 0, err
	}
	position := offset
	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		position += m.position
	case io.SeekEnd:
		position += m.size
	default:
		return 0, ErrInvalidPosition
	}
	if position < 0 {
		return 0, fmt.Errorf("%w: %d", ErrInvalidPosition, position)
	}
	// it is allowed to jump beyond the end of the file: jump to the end of the
	// reader
	if position > m.size {
		position = m.size
	}
	m.position = position
	return m.position, nil
}

// Position returns the offset of the next byte to be read.
func (m *MappedFile) Position() (int64, error) {
	if err := m.checkClosed(); err != nil {
		return 0, err
	}
	return m.position, nil
}

// ReadByte reads one byte, reporting io.EOF at the end where Java's read()
// answers -1.
func (m *MappedFile) ReadByte() (byte, error) {
	eof, err := m.IsEOF()
	if err != nil {
		return 0, err
	}
	if eof {
		return 0, io.EOF
	}
	c := m.mapped.At(int(m.position))
	m.position++
	return c, nil
}

// Read fills p, reporting io.EOF at the end where Java's read(byte[], int, int)
// answers -1.
func (m *MappedFile) Read(p []byte) (int, error) {
	eof, err := m.IsEOF()
	if err != nil {
		return 0, err
	}
	if eof {
		return 0, io.EOF
	}
	remainingBytes := int(m.size - m.position)
	if remainingBytes > len(p) {
		remainingBytes = len(p)
	}
	n, err := m.mapped.ReadAt(p[:remainingBytes], m.position)
	m.position += int64(n)
	if err != nil && err != io.EOF {
		return n, err
	}
	return n, nil
}

// Length returns the size of the whole file.
func (m *MappedFile) Length() (int64, error) {
	if err := m.checkClosed(); err != nil {
		return 0, err
	}
	return m.size, nil
}

// checkClosed reports whether the source has already been closed.
func (m *MappedFile) checkClosed() error {
	if m.IsClosed() {
		return ErrClosed
	}
	return nil
}

// IsClosed reports whether the mapping has been released.
func (m *MappedFile) IsClosed() bool { return m.mapped == nil }

// IsEOF reports whether the cursor sits at or past the end of the file.
func (m *MappedFile) IsEOF() (bool, error) {
	if err := m.checkClosed(); err != nil {
		return false, err
	}
	return m.position >= m.size, nil
}

// CreateView returns an independent cursor clipped to the given section, which
// reads through a second cursor over the same mapping.
//
// Java does not check that the source is still open here, so createView on a
// closed one dereferences the released buffer and raises NullPointerException
// -- where its sibling RandomAccessReadBufferedFile.createView calls
// checkClosed and raises IOException. The port checks, so this answers
// ErrClosed like every other operation on a closed source. The difference is
// recorded in migration/JAVA-BUGS.md entry 65 and in STATUS.md, and pinned by
// TestMappedFileViewOfAClosedSource.
func (m *MappedFile) CreateView(start, length int64) (RandomAccessRead, error) {
	if err := m.checkClosed(); err != nil {
		return nil, err
	}
	return NewReadViewOwned(m.duplicate(), start, length), nil
}
