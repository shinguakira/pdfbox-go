package pdfio

import "io"

// ReadView exposes a section of another RandomAccessRead as a source in its own
// right, clipping it to a start offset and a length.
//
// Port of org.apache.pdfbox.io.RandomAccessReadView.
type ReadView struct {
	source     RandomAccessRead
	start      int64
	length     int64
	closeInput bool
	position   int64
	closed     bool
}

var _ RandomAccessRead = (*ReadView)(nil)

// NewReadView clips source to the given section. Closing the view leaves source
// open; use NewReadViewOwned when the view should own its source.
func NewReadView(source RandomAccessRead, start, length int64) *ReadView {
	return &ReadView{source: source, start: start, length: length}
}

// NewReadViewOwned clips source to the given section and closes source when the
// view is closed.
func NewReadViewOwned(source RandomAccessRead, start, length int64) *ReadView {
	return &ReadView{source: source, start: start, length: length, closeInput: true}
}

func (v *ReadView) checkClosed() error {
	if v.IsClosed() {
		return ErrClosed
	}
	return nil
}

// restorePosition points the underlying source at the byte this view is on.
// Each read does this because the source may be shared with other views.
func (v *ReadView) restorePosition() error {
	return SeekTo(v.source, v.start+v.position)
}

// Position returns the offset within the view.
func (v *ReadView) Position() (int64, error) {
	if err := v.checkClosed(); err != nil {
		return 0, err
	}
	return v.position, nil
}

// Seek moves the view cursor. Offsets past the end of the view are clamped when
// seeking the underlying source but still recorded, as they are in Java.
func (v *ReadView) Seek(offset int64, whence int) (int64, error) {
	if err := v.checkClosed(); err != nil {
		return 0, err
	}
	abs := offset
	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		abs += v.position
	case io.SeekEnd:
		abs += v.length
	default:
		return 0, ErrInvalidPosition
	}
	if abs < 0 {
		return 0, ErrInvalidPosition
	}
	clamped := abs
	if clamped > v.length {
		clamped = v.length
	}
	if err := SeekTo(v.source, v.start+clamped); err != nil {
		return 0, err
	}
	v.position = abs
	return v.position, nil
}

// ReadByte returns the next byte of the view, or io.EOF at its end.
func (v *ReadView) ReadByte() (byte, error) {
	eof, err := v.IsEOF()
	if err != nil {
		return 0, err
	}
	if eof {
		return 0, io.EOF
	}
	if err := v.restorePosition(); err != nil {
		return 0, err
	}
	b, err := v.source.ReadByte()
	if err != nil {
		return 0, err
	}
	v.position++
	return b, nil
}

// Read fills p from the view, never reading past the end of the section.
func (v *ReadView) Read(p []byte) (int, error) {
	eof, err := v.IsEOF()
	if err != nil {
		return 0, err
	}
	if eof {
		return 0, io.EOF
	}
	if err := v.restorePosition(); err != nil {
		return 0, err
	}
	limit := int64(len(p))
	if available := v.length - v.position; limit > available {
		limit = available
	}
	n, err := v.source.Read(p[:limit])
	if n > 0 {
		v.position += int64(n)
	}
	if err == io.EOF && n > 0 {
		err = nil
	}
	return n, err
}

// Length returns the length of the section this view covers.
func (v *ReadView) Length() (int64, error) {
	if err := v.checkClosed(); err != nil {
		return 0, err
	}
	return v.length, nil
}

// Close releases the view, closing the underlying source only if the view owns
// it.
func (v *ReadView) Close() error {
	if v.closed {
		return nil
	}
	v.closed = true
	if v.closeInput && v.source != nil {
		return v.source.Close()
	}
	v.source = nil
	return nil
}

// IsClosed reports whether the view or the source behind it has been closed.
func (v *ReadView) IsClosed() bool {
	return v.closed || v.source == nil || v.source.IsClosed()
}

// IsEOF reports whether the cursor has reached the end of the section.
func (v *ReadView) IsEOF() (bool, error) {
	if err := v.checkClosed(); err != nil {
		return false, err
	}
	return v.position >= v.length, nil
}

// CreateView is not supported: a view cannot subdivide itself, because the
// cursor it would hand out would alias the one this view already owns.
func (v *ReadView) CreateView(start, length int64) (RandomAccessRead, error) {
	return nil, ErrViewNotSupported
}
