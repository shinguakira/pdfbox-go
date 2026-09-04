package filter

import (
	"bufio"
	"io"
)

// The bit-level streams the LZW filter reads and writes through.
//
// Java uses javax.imageio.stream.MemoryCacheImageInputStream and
// MemoryCacheImageOutputStream, which are general purpose image IO streams; all
// the LZW filter asks of them is readBits and writeBits, most significant bit
// first, so the port supplies those two rather than the whole class. The names
// match the Java methods so that the filter reads the same.

// bitReader reads a big endian bit stream.
//
// Port of the readBits half of ImageInputStreamImpl, which keeps a bit offset
// into the current byte and fills the accumulator most significant bit first.
type bitReader struct {
	in        *bufio.Reader
	bitBuf    uint32
	bitOffset uint
}

func newBitReader(r io.Reader) *bitReader {
	return &bitReader{in: bufio.NewReader(r)}
}

// readBits reads numBits bits, most significant first.
//
// Java's readBits throws EOFException where the stream runs out mid-value, and
// LZWFilter catches that to end the stream; the port returns io.EOF for it.
func (b *bitReader) readBits(numBits int) (int64, error) {
	var value int64
	for i := 0; i < numBits; i++ {
		if b.bitOffset == 0 {
			c, err := b.in.ReadByte()
			if err != nil {
				return 0, io.EOF
			}
			b.bitBuf = uint32(c)
			b.bitOffset = 8
		}
		b.bitOffset--
		value = value<<1 | int64((b.bitBuf>>b.bitOffset)&1)
	}
	return value, nil
}

// bitWriter writes a big endian bit stream.
//
// Port of the writeBits half of ImageOutputStreamImpl.
type bitWriter struct {
	out        *bufio.Writer
	bitBuf     byte
	bitsFilled uint
}

func newBitWriter(w io.Writer) *bitWriter {
	return &bitWriter{out: bufio.NewWriter(w)}
}

// writeBits writes the low numBits bits of value, most significant first.
func (b *bitWriter) writeBits(value int64, numBits int) error {
	for i := numBits - 1; i >= 0; i-- {
		bit := byte((value >> uint(i)) & 1)
		b.bitBuf = b.bitBuf<<1 | bit
		b.bitsFilled++
		if b.bitsFilled == 8 {
			if err := b.out.WriteByte(b.bitBuf); err != nil {
				return err
			}
			b.bitBuf = 0
			b.bitsFilled = 0
		}
	}
	return nil
}

// flush writes any partial byte, padding it with zeroes on the right, and
// flushes the writer below.
//
// Java's MemoryCacheImageOutputStream flushes the same way when it is closed:
// the bits already written stay, and the rest of the byte is zero. LZWFilter
// pads with seven zero bits itself before it closes, so at most one byte is at
// stake.
func (b *bitWriter) flush() error {
	if b.bitsFilled > 0 {
		if err := b.out.WriteByte(b.bitBuf << (8 - b.bitsFilled)); err != nil {
			return err
		}
		b.bitBuf = 0
		b.bitsFilled = 0
	}
	return b.out.Flush()
}
