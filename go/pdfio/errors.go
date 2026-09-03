package pdfio

import "errors"

// Sentinel errors returned by the random access implementations. The Java code
// raises IOException with a message for each of these conditions; callers there
// can only match on the message, so the port promotes them to values that can
// be compared with errors.Is.
var (
	// ErrClosed is returned by every operation on a source that has already
	// been closed.
	ErrClosed = errors.New("pdfio: random access already closed")

	// ErrInvalidPosition is returned when a seek target is negative.
	ErrInvalidPosition = errors.New("pdfio: invalid position")

	// ErrPrematureEOF is returned by ReadFully when the source holds fewer
	// bytes than requested.
	ErrPrematureEOF = errors.New("pdfio: premature end of buffer reached")

	// ErrNoMoreChunks is returned when a buffer walks past its last chunk.
	ErrNoMoreChunks = errors.New("pdfio: no more chunks available, end of buffer reached")

	// ErrViewNotSupported is returned by CreateView on sources that cannot
	// produce an independent cursor over themselves.
	ErrViewNotSupported = errors.New("pdfio: createView is not supported by this source")
)
