package pdfio

import (
	"io"
	"sync"
)

// DefaultChunkSize4KB is the chunk size a ReadBuffer allocates by default.
const DefaultChunkSize4KB = 1 << 12

// ReadBuffer is a RandomAccessRead that keeps its data in memory, split into
// equally sized chunks.
//
// Port of org.apache.pdfbox.io.RandomAccessReadBuffer. Java stores the chunks
// in an ArrayList of ByteBuffer and leans on the position each ByteBuffer
// carries; the port keeps plain byte slices and tracks the cursor explicitly,
// which makes the chunk arithmetic the Java code performs implicitly visible.
type ReadBuffer struct {
	chunkSize int
	chunks    [][]byte

	// cursor state
	pointer       int64 // absolute offset into the whole buffer
	chunkIndex    int   // index of the chunk the cursor sits in
	chunkPointer  int   // offset of the cursor within that chunk
	maxChunkIndex int   // index of the last chunk holding data
	size          int64 // total number of bytes held

	closed bool

	// clones handed out by CreateView, closed when this buffer is closed
	mu     sync.Mutex
	clones []*ReadBuffer
}

var _ RandomAccessRead = (*ReadBuffer)(nil)

// NewReadBuffer returns an empty buffer using the default chunk size.
func NewReadBuffer() *ReadBuffer {
	return NewReadBufferSize(DefaultChunkSize4KB)
}

// NewReadBufferSize returns an empty buffer using the given chunk size.
func NewReadBufferSize(chunkSize int) *ReadBuffer {
	if chunkSize <= 0 {
		chunkSize = DefaultChunkSize4KB
	}
	return &ReadBuffer{
		chunkSize: chunkSize,
		chunks:    [][]byte{make([]byte, chunkSize)},
	}
}

// NewReadBufferBytes wraps an existing byte slice as a single chunk. The slice
// is not copied, matching ByteBuffer.wrap in the Java constructor.
func NewReadBufferBytes(input []byte) *ReadBuffer {
	if len(input) == 0 {
		return NewReadBuffer()
	}
	return &ReadBuffer{
		chunkSize: len(input),
		chunks:    [][]byte{input},
		size:      int64(len(input)),
	}
}

// NewReadBufferFromReader copies everything r produces into a new buffer and
// positions the cursor at zero.
//
// Port of the RandomAccessReadBuffer(InputStream) constructor. Java hand-rolls
// a fill loop with a one byte look-ahead to decide when to grow; filling each
// fresh chunk with io.ReadFull expresses the same thing without the look-ahead.
func NewReadBufferFromReader(r io.Reader) (*ReadBuffer, error) {
	b := NewReadBuffer()
	for {
		n, err := io.ReadFull(r, b.chunks[b.maxChunkIndex][b.chunkPointer:])
		b.chunkPointer += n
		b.size += int64(n)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if err := b.expandBuffer(); err != nil {
			return nil, err
		}
	}
	if err := SeekTo(b, 0); err != nil {
		return nil, err
	}
	return b, nil
}

// NewReadBufferFromReadCloser is NewReadBufferFromReader for a source that must
// be closed once copied, mirroring createBufferFromStream in Java.
func NewReadBufferFromReadCloser(r io.ReadCloser) (*ReadBuffer, error) {
	defer r.Close()
	return NewReadBufferFromReader(r)
}

// clone returns an independent cursor over the same chunks. The chunk slices
// are shared, exactly as ByteBuffer.duplicate shares its backing array.
func (b *ReadBuffer) clone() *ReadBuffer {
	chunks := make([][]byte, len(b.chunks))
	copy(chunks, b.chunks)
	return &ReadBuffer{
		chunkSize:     b.chunkSize,
		chunks:        chunks,
		size:          b.size,
		maxChunkIndex: b.maxChunkIndex,
	}
}

func (b *ReadBuffer) checkClosed() error {
	if b.closed {
		return ErrClosed
	}
	return nil
}

// Close releases the chunks and closes every view handed out by CreateView.
func (b *ReadBuffer) Close() error {
	if b.closed {
		return nil
	}
	b.mu.Lock()
	clones := b.clones
	b.clones = nil
	b.mu.Unlock()
	for _, c := range clones {
		c.Close()
	}
	b.closed = true
	b.chunks = nil
	return nil
}

// IsClosed reports whether Close has been called.
func (b *ReadBuffer) IsClosed() bool { return b.closed }

// Seek implements io.Seeker. Seeking past the end of the buffer is allowed and
// parks the cursor at the end, as it is in Java.
func (b *ReadBuffer) Seek(offset int64, whence int) (int64, error) {
	if err := b.checkClosed(); err != nil {
		return 0, err
	}
	abs := offset
	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		abs += b.pointer
	case io.SeekEnd:
		abs += b.size
	default:
		return 0, ErrInvalidPosition
	}
	if abs < 0 {
		return 0, ErrInvalidPosition
	}
	if abs < b.size {
		b.pointer = abs
		b.chunkIndex = int(b.pointer / int64(b.chunkSize))
		b.chunkPointer = int(b.pointer % int64(b.chunkSize))
	} else {
		// jumping beyond the end parks the cursor at the end
		b.pointer = b.size
		b.chunkIndex = b.maxChunkIndex
		b.chunkPointer = int(b.size % int64(b.chunkSize))
	}
	return b.pointer, nil
}

// Position returns the offset of the next byte to be read.
func (b *ReadBuffer) Position() (int64, error) {
	if err := b.checkClosed(); err != nil {
		return 0, err
	}
	return b.pointer, nil
}

// Length returns the number of bytes held by the buffer.
func (b *ReadBuffer) Length() (int64, error) {
	if err := b.checkClosed(); err != nil {
		return 0, err
	}
	return b.size, nil
}

// IsEOF reports whether the cursor is at or past the end of the data.
func (b *ReadBuffer) IsEOF() (bool, error) {
	if err := b.checkClosed(); err != nil {
		return false, err
	}
	return b.pointer >= b.size, nil
}

// ReadByte returns the next byte, or io.EOF at the end of the buffer.
func (b *ReadBuffer) ReadByte() (byte, error) {
	if err := b.checkClosed(); err != nil {
		return 0, err
	}
	if b.pointer >= b.size {
		return 0, io.EOF
	}
	if b.chunkPointer >= b.chunkSize {
		if b.chunkIndex >= b.maxChunkIndex {
			return 0, io.EOF
		}
		b.chunkIndex++
		b.chunkPointer = 0
	}
	v := b.chunks[b.chunkIndex][b.chunkPointer]
	b.chunkPointer++
	b.pointer++
	return v, nil
}

// Read fills p from the buffer, crossing chunk boundaries as needed.
func (b *ReadBuffer) Read(p []byte) (int, error) {
	if err := b.checkClosed(); err != nil {
		return 0, err
	}
	if len(p) == 0 {
		return 0, nil
	}
	read := b.readFromChunk(p)
	if read < 0 {
		if b.remaining() > 0 {
			read = 0
		} else {
			return 0, io.EOF
		}
	}
	for read < len(p) && b.remaining() > 0 {
		if b.chunkPointer == b.chunkSize {
			if err := b.nextChunk(); err != nil {
				return read, err
			}
		}
		n := b.readFromChunk(p[read:])
		// Java adds this result unconditionally, which would subtract on a -1;
		// stopping instead keeps the returned count honest.
		if n <= 0 {
			break
		}
		read += n
	}
	return read, nil
}

// readFromChunk copies as much of p as the current chunk can supply. It returns
// -1 when nothing is left, matching the Java helper it is ported from.
func (b *ReadBuffer) readFromChunk(p []byte) int {
	if b.pointer >= b.size {
		return -1
	}
	maxLength := len(p)
	if int64(maxLength) > b.size-b.pointer {
		maxLength = int(b.size - b.pointer)
	}
	remainingInChunk := b.chunkSize - b.chunkPointer
	if remainingInChunk == 0 {
		return -1
	}
	n := maxLength
	if n > remainingInChunk {
		n = remainingInChunk
	}
	copy(p[:n], b.chunks[b.chunkIndex][b.chunkPointer:b.chunkPointer+n])
	b.chunkPointer += n
	b.pointer += int64(n)
	return n
}

func (b *ReadBuffer) remaining() int64 {
	if b.pointer >= b.size {
		return 0
	}
	return b.size - b.pointer
}

// expandBuffer moves to the next chunk, allocating one if the cursor is already
// on the last chunk.
func (b *ReadBuffer) expandBuffer() error {
	if b.maxChunkIndex > b.chunkIndex {
		return b.nextChunk()
	}
	b.chunks = append(b.chunks, make([]byte, b.chunkSize))
	b.chunkPointer = 0
	b.maxChunkIndex++
	b.chunkIndex++
	return nil
}

// nextChunk advances to the already allocated next chunk.
func (b *ReadBuffer) nextChunk() error {
	if b.chunkIndex == b.maxChunkIndex {
		return ErrNoMoreChunks
	}
	b.chunkIndex++
	b.chunkPointer = 0
	return nil
}

// resetBuffers rewinds to position zero and drops every chunk but the first.
func (b *ReadBuffer) resetBuffers() {
	first := b.chunks[0]
	b.size = 0
	b.pointer = 0
	b.chunkPointer = 0
	b.chunkIndex = 0
	b.maxChunkIndex = 0
	b.chunks = [][]byte{first}
}

// CreateView returns a read-only window over an independent cursor.
//
// Java caches one clone per calling thread in a ConcurrentMap keyed by thread
// id. Go has no stable goroutine identity, so the port hands every call its own
// clone; the chunks stay shared, so a clone costs one slice header per chunk.
func (b *ReadBuffer) CreateView(start, length int64) (RandomAccessRead, error) {
	if err := b.checkClosed(); err != nil {
		return nil, err
	}
	c := b.clone()
	b.mu.Lock()
	b.clones = append(b.clones, c)
	b.mu.Unlock()
	return NewReadView(c, start, length), nil
}
