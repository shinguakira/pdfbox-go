package pdfio

import (
	"io"
	"math"
)

// Reader adapts a RandomAccessRead to a plain io.Reader that keeps its own
// position, so several readers can walk one source independently.
//
// Port of org.apache.pdfbox.io.RandomAccessInputStream.
type Reader struct {
	source   RandomAccessRead
	position int64
}

var _ io.Reader = (*Reader)(nil)

// NewReader returns a reader positioned at the start of source.
func NewReader(source RandomAccessRead) *Reader {
	return &Reader{source: source}
}

// restorePosition points the source at the byte this reader is on.
func (r *Reader) restorePosition() error {
	return SeekTo(r.source, r.position)
}

// Available returns the number of bytes that can still be read.
func (r *Reader) Available() (int, error) {
	length, err := r.source.Length()
	if err != nil {
		return 0, err
	}
	remaining := length - r.position
	if remaining <= 0 {
		return 0, nil
	}
	// Java is Math.min(input.length() - position, Integer.MAX_VALUE), so a
	// source larger than an int saturates rather than wrapping.
	if remaining > math.MaxInt32 {
		return math.MaxInt32, nil
	}
	return int(remaining), nil
}

// Read implements io.Reader.
func (r *Reader) Read(p []byte) (int, error) {
	if err := r.restorePosition(); err != nil {
		return 0, err
	}
	eof, err := r.source.IsEOF()
	if err != nil {
		return 0, err
	}
	if eof {
		return 0, io.EOF
	}
	n, err := r.source.Read(p)
	if n > 0 {
		r.position += int64(n)
	}
	return n, err
}

// ReadByte implements io.ByteReader.
func (r *Reader) ReadByte() (byte, error) {
	if err := r.restorePosition(); err != nil {
		return 0, err
	}
	b, err := r.source.ReadByte()
	if err != nil {
		return 0, err
	}
	r.position++
	return b, nil
}

// Skip advances the reader by n bytes and reports how many it skipped.
func (r *Reader) Skip(n int64) (int64, error) {
	if n <= 0 {
		return 0, nil
	}
	if err := r.restorePosition(); err != nil {
		return 0, err
	}
	if err := SeekTo(r.source, r.position+n); err != nil {
		return 0, err
	}
	r.position += n
	return n, nil
}

// Writer adapts a RandomAccessWrite to a plain io.Writer.
//
// Port of org.apache.pdfbox.io.RandomAccessOutputStream. No position is tracked
// because each stream has a single writer.
type Writer struct {
	dst RandomAccessWrite
}

var _ io.Writer = (*Writer)(nil)

// NewWriter returns a writer that appends to dst.
func NewWriter(dst RandomAccessWrite) *Writer {
	return &Writer{dst: dst}
}

// Write implements io.Writer.
func (w *Writer) Write(p []byte) (int, error) { return w.dst.Write(p) }

// WriteByte implements io.ByteWriter.
func (w *Writer) WriteByte(b byte) error { return w.dst.WriteByte(b) }
