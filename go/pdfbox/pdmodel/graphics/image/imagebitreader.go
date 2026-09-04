package image

import (
	"bufio"
	"io"
)

// imageBitReader reads a big endian bit stream.
//
// Port of the readBits half of javax.imageio.stream.MemoryCacheImageInputStream,
// which SampledImageReader reads sample data through. Java's readBits throws
// EOFException at the end of the stream; SampledImageReader never catches it,
// so a truncated image propagates the exception out of getRGBImage. The port
// returns zeroes instead and lets the caller have the rows that did read, which
// is a deliberate difference: it is the same tolerance the filters have for a
// damaged stream, and an image that stops half way is more useful than none.
// See migration/STATUS.md.
type imageBitReader struct {
	in        *bufio.Reader
	bitBuf    uint32
	bitOffset uint
}

func newImageBitReader(r io.Reader) *imageBitReader {
	return &imageBitReader{in: bufio.NewReader(r)}
}

// readBits reads numBits bits, most significant first.
func (b *imageBitReader) readBits(numBits int) int64 {
	var value int64
	for i := 0; i < numBits; i++ {
		if b.bitOffset == 0 {
			c, err := b.in.ReadByte()
			if err != nil {
				return value << uint(numBits-i)
			}
			b.bitBuf = uint32(c)
			b.bitOffset = 8
		}
		b.bitOffset--
		value = value<<1 | int64((b.bitBuf>>b.bitOffset)&1)
	}
	return value
}
