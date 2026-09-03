package pdfio

// ReadWriteBuffer is an in-memory RandomAccess: data written to it can be read
// back through the embedded ReadBuffer.
//
// Port of org.apache.pdfbox.io.RandomAccessReadWriteBuffer, which extends
// RandomAccessReadBuffer. Go has no class inheritance, so the port embeds the
// read buffer instead and the write methods reach into its chunk state; see
// migration/conventions/java-to-go.md on porting extends.
type ReadWriteBuffer struct {
	ReadBuffer
}

var _ RandomAccess = (*ReadWriteBuffer)(nil)

// NewReadWriteBuffer returns an empty buffer using the default chunk size.
func NewReadWriteBuffer() *ReadWriteBuffer {
	return NewReadWriteBufferSize(DefaultChunkSize4KB)
}

// NewReadWriteBufferSize returns an empty buffer using the given chunk size.
func NewReadWriteBufferSize(chunkSize int) *ReadWriteBuffer {
	return &ReadWriteBuffer{ReadBuffer: *NewReadBufferSize(chunkSize)}
}

// Clear discards everything written so far and rewinds to position zero.
func (b *ReadWriteBuffer) Clear() error {
	if err := b.checkClosed(); err != nil {
		return err
	}
	b.resetBuffers()
	return nil
}

// WriteByte appends a single byte at the current position.
func (b *ReadWriteBuffer) WriteByte(v byte) error {
	if err := b.checkClosed(); err != nil {
		return err
	}
	if b.chunkSize-b.chunkPointer <= 0 {
		if err := b.expandBuffer(); err != nil {
			return err
		}
	}
	b.chunks[b.chunkIndex][b.chunkPointer] = v
	b.chunkPointer++
	b.pointer++
	if b.pointer > b.size {
		b.size = b.pointer
	}
	return nil
}

// Write appends p at the current position, allocating chunks as needed.
func (b *ReadWriteBuffer) Write(p []byte) (int, error) {
	if err := b.checkClosed(); err != nil {
		return 0, err
	}
	offset := 0
	for offset < len(p) {
		n := len(p) - offset
		if space := b.chunkSize - b.chunkPointer; n > space {
			n = space
		}
		if n <= 0 {
			if err := b.expandBuffer(); err != nil {
				return offset, err
			}
			n = len(p) - offset
			if space := b.chunkSize - b.chunkPointer; n > space {
				n = space
			}
		}
		if n > 0 {
			copy(b.chunks[b.chunkIndex][b.chunkPointer:b.chunkPointer+n], p[offset:offset+n])
			b.chunkPointer += n
			b.pointer += int64(n)
			offset += n
		}
	}
	if b.pointer > b.size {
		b.size = b.pointer
	}
	return offset, nil
}
