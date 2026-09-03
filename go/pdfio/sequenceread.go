package pdfio

import (
	"errors"
	"io"
)

// SequenceRead concatenates several RandomAccessRead sources so that they can
// be addressed as one contiguous source.
//
// Port of org.apache.pdfbox.io.SequenceRandomAccessRead.
type SequenceRead struct {
	readers        []RandomAccessRead
	startPositions []int64
	endPositions   []int64
	currentIndex   int
	position       int64
	totalLength    int64
	closed         bool
}

var _ RandomAccessRead = (*SequenceRead)(nil)

// NewSequenceRead concatenates the given sources in order. Empty sources are
// dropped. It fails if the list is empty or if a source cannot report its
// length; the Java constructor throws IllegalArgumentException in both cases.
func NewSequenceRead(readers []RandomAccessRead) (*SequenceRead, error) {
	if len(readers) == 0 {
		return nil, errors.New("pdfio: missing input parameter")
	}
	s := &SequenceRead{}
	for _, r := range readers {
		length, err := r.Length()
		if err != nil {
			return nil, err
		}
		if length <= 0 {
			continue
		}
		s.readers = append(s.readers, r)
		s.startPositions = append(s.startPositions, s.totalLength)
		s.totalLength += length
		s.endPositions = append(s.endPositions, s.totalLength-1)
	}
	if len(s.readers) == 0 {
		return nil, errors.New("pdfio: empty list")
	}
	return s, nil
}

func (s *SequenceRead) checkClosed() error {
	if s.closed {
		return ErrClosed
	}
	return nil
}

// Close closes every source in the sequence.
func (s *SequenceRead) Close() error {
	if s.closed {
		return nil
	}
	var firstErr error
	for _, r := range s.readers {
		if err := r.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	s.readers = nil
	s.closed = true
	return firstErr
}

// IsClosed reports whether Close has been called.
func (s *SequenceRead) IsClosed() bool { return s.closed }

// currentReader returns the source the cursor sits in, advancing to the next
// one when the current source is exhausted.
func (s *SequenceRead) currentReader() (RandomAccessRead, error) {
	r := s.readers[s.currentIndex]
	eof, err := r.IsEOF()
	if err != nil {
		return nil, err
	}
	if eof && s.currentIndex < len(s.readers)-1 {
		s.currentIndex++
		r = s.readers[s.currentIndex]
		if err := SeekTo(r, 0); err != nil {
			return nil, err
		}
	}
	return r, nil
}

// ReadByte returns the next byte of the concatenation, or io.EOF at its end.
func (s *SequenceRead) ReadByte() (byte, error) {
	if err := s.checkClosed(); err != nil {
		return 0, err
	}
	r, err := s.currentReader()
	if err != nil {
		return 0, err
	}
	b, err := r.ReadByte()
	if err != nil {
		return 0, err
	}
	s.position++
	return b, nil
}

// Read fills p, crossing source boundaries as needed.
func (s *SequenceRead) Read(p []byte) (int, error) {
	if err := s.checkClosed(); err != nil {
		return 0, err
	}
	if len(p) == 0 {
		return 0, nil
	}
	limit := int64(len(p))
	if available := s.totalLength - s.position; limit > available {
		limit = available
	}
	if limit <= 0 {
		return 0, io.EOF
	}
	read := 0
	for int64(read) < limit {
		r, err := s.currentReader()
		if err != nil {
			return read, err
		}
		n, err := r.Read(p[read:limit])
		if n > 0 {
			read += n
			s.position += int64(n)
			continue
		}
		if err != nil && !errors.Is(err, io.EOF) {
			return read, err
		}
		// The current source is exhausted; stop if it was also the last one.
		if s.currentIndex >= len(s.readers)-1 {
			break
		}
		s.currentIndex++
		if err := SeekTo(s.readers[s.currentIndex], 0); err != nil {
			return read, err
		}
	}
	if read == 0 {
		return 0, io.EOF
	}
	return read, nil
}

// Position returns the offset within the concatenation.
func (s *SequenceRead) Position() (int64, error) {
	if err := s.checkClosed(); err != nil {
		return 0, err
	}
	return s.position, nil
}

// Seek moves the cursor, selecting the source that holds the target offset.
// Seeking past the end is allowed and parks the cursor at the end.
func (s *SequenceRead) Seek(offset int64, whence int) (int64, error) {
	if err := s.checkClosed(); err != nil {
		return 0, err
	}
	abs := offset
	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		abs += s.position
	case io.SeekEnd:
		abs += s.totalLength
	default:
		return 0, ErrInvalidPosition
	}
	if abs < 0 {
		return 0, ErrInvalidPosition
	}
	if abs >= s.totalLength {
		s.currentIndex = len(s.readers) - 1
		s.position = s.totalLength
	} else {
		for i := range s.readers {
			if abs >= s.startPositions[i] && abs <= s.endPositions[i] {
				s.currentIndex = i
				break
			}
		}
		s.position = abs
	}
	target := s.position - s.startPositions[s.currentIndex]
	if err := SeekTo(s.readers[s.currentIndex], target); err != nil {
		return 0, err
	}
	return s.position, nil
}

// Length returns the summed length of every source in the sequence.
func (s *SequenceRead) Length() (int64, error) {
	if err := s.checkClosed(); err != nil {
		return 0, err
	}
	return s.totalLength, nil
}

// IsEOF reports whether the cursor has reached the end of the concatenation.
func (s *SequenceRead) IsEOF() (bool, error) {
	if err := s.checkClosed(); err != nil {
		return false, err
	}
	return s.position >= s.totalLength, nil
}

// CreateView is not supported: the sources are shared, so a second cursor over
// them would alias the one this sequence already owns.
func (s *SequenceRead) CreateView(start, length int64) (RandomAccessRead, error) {
	return nil, ErrViewNotSupported
}
