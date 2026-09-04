package function

import (
	"bufio"
	"io"
)

// sampleBitReader reads a big endian bit stream.
//
// Port of the readBits half of javax.imageio.stream.MemoryCacheImageInputStream,
// which is what PDFunctionType0 reads its sample table through. The filter
// package has one of these for the LZW filter; the two do not share it, because
// that one is unexported there and this package must not reach into the filters.
type sampleBitReader struct {
	in        *bufio.Reader
	bitBuf    uint32
	bitOffset uint
}

func newSampleBitReader(r io.Reader) *sampleBitReader {
	return &sampleBitReader{in: bufio.NewReader(r)}
}

// readBits reads numBits bits, most significant first.
//
// Java's readBits throws EOFException where the stream runs out; the port
// returns io.EOF, which getSamples logs the way Java logs the IOException.
func (b *sampleBitReader) readBits(numBits int) (int64, error) {
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
