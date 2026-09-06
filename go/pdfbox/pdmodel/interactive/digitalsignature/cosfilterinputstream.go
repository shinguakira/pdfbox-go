package digitalsignature

import (
	"bytes"
	"fmt"
	"io"
)

// COSFilterInputStream is a filtered stream that includes the bytes that are in
// the (begin,length) intervals passed to the constructor.
//
// Port of COSFilterInputStream, which extends FilterInputStream. Go has no such
// class, so the port wraps a reader and implements io.Reader.
type COSFilterInputStream struct {
	in       io.Reader
	ranges   [][2]int
	rangeIdx int
	position int64
}

var _ io.Reader = (*COSFilterInputStream)(nil)

// NewCOSFilterInputStream returns a stream over the given ranges of the given
// reader.
//
// Java declares no exception here; the port returns one because the skip that
// Java does lazily has no place to fail from in Go's Read.
func NewCOSFilterInputStream(in io.Reader, byteRange []int) (*COSFilterInputStream, error) {
	s := &COSFilterInputStream{in: in}
	s.calculateRanges(byteRange)
	return s, nil
}

// NewCOSFilterInputStreamOfBytes returns a stream over the given ranges of the
// given bytes.
func NewCOSFilterInputStreamOfBytes(in []byte, byteRange []int) (*COSFilterInputStream, error) {
	return NewCOSFilterInputStream(bytes.NewReader(in), byteRange)
}

// Read fills b with the next bytes inside the ranges, and answers io.EOF once
// they are exhausted, which is the -1 Java answers.
func (s *COSFilterInputStream) Read(b []byte) (int, error) {
	if s.rangeIdx == -1 || s.remaining() <= 0 {
		advanced, err := s.nextRange()
		if err != nil {
			return 0, err
		}
		if !advanced {
			return 0, io.EOF // EOF
		}
	}
	length := int64(len(b))
	if remaining := s.remaining(); remaining < length {
		length = remaining
	}
	bytesRead, err := s.in.Read(b[:length])
	s.position += int64(bytesRead)
	if err == io.EOF && bytesRead > 0 {
		// Java's InputStream.read answers the count and leaves the -1 for the
		// next call; Go's may answer both at once, so the EOF is held back.
		return bytesRead, nil
	}
	return bytesRead, err
}

// ToByteArray reads everything inside the ranges.
func (s *COSFilterInputStream) ToByteArray() ([]byte, error) {
	return io.ReadAll(s)
}

// Close closes the underlying reader where it can be closed, which is what
// FilterInputStream.close does.
func (s *COSFilterInputStream) Close() error {
	if closer, isCloser := s.in.(io.Closer); isCloser {
		return closer.Close()
	}
	return nil
}

// calculateRanges turns the flat byte range into the pairs the stream walks.
// Java declares it private.
func (s *COSFilterInputStream) calculateRanges(byteRange []int) {
	s.ranges = make([][2]int, len(byteRange)/2)
	for i := 0; i < len(byteRange)/2; i++ {
		s.ranges[i] = [2]int{byteRange[i*2], byteRange[i*2] + byteRange[i*2+1]}
	}
	s.rangeIdx = -1
}

// remaining returns how much of the current range is left. Java declares it
// private.
func (s *COSFilterInputStream) remaining() int64 {
	return int64(s.ranges[s.rangeIdx][1]) - s.position
}

// nextRange moves to the next range, skipping the gap before it, and reports
// whether there was one. Java declares it private.
func (s *COSFilterInputStream) nextRange() (bool, error) {
	if s.rangeIdx+1 >= len(s.ranges) {
		return false, nil
	}
	s.rangeIdx++
	for s.position < int64(s.ranges[s.rangeIdx][0]) {
		skipped, err := skip(s.in, int64(s.ranges[s.rangeIdx][0])-s.position)
		if err != nil {
			return false, err
		}
		if skipped == 0 {
			return false, fmt.Errorf(
				"digitalsignature: FilterInputStream.skip() returns 0, range: [%d, %d]",
				s.ranges[s.rangeIdx][0], s.ranges[s.rangeIdx][1])
		}
		s.position += skipped
	}
	return true, nil
}

// skip discards the next n bytes of the reader and returns how many it
// discarded, which is what InputStream.skip does.
func skip(in io.Reader, n int64) (int64, error) {
	if seeker, isSeeker := in.(io.Seeker); isSeeker {
		before, err := seeker.Seek(0, io.SeekCurrent)
		if err == nil {
			after, err := seeker.Seek(n, io.SeekCurrent)
			if err != nil {
				return 0, err
			}
			return after - before, nil
		}
	}
	skipped, err := io.CopyN(io.Discard, in, n)
	if err == io.EOF {
		return skipped, nil
	}
	return skipped, err
}
