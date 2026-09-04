// Package pdfwriter writes a PDF document out.
//
// Port of org.apache.pdfbox.pdfwriter.
package pdfwriter

import "io"

// The line endings the writer emits.
//
// Port of the COSStandardOutputStream constants.
var (
	crlf = []byte{'\r', '\n'}
	lf   = []byte{'\n'}
	eol  = []byte{'\n'}
)

// standardOutputStream counts the bytes written and remembers whether the last
// thing written was a line ending, so that the writer never emits two in a row.
//
// Port of org.apache.pdfbox.pdfwriter.COSStandardOutputStream. The position it
// keeps is what every cross-reference offset is taken from, so it is not
// bookkeeping: an off-by-one here writes an xref table that points at the wrong
// byte.
type standardOutputStream struct {
	out io.Writer
	// current byte position in the output stream
	position int64
	// flag to prevent generating two newlines in sequence
	onNewLine bool
}

var _ io.Writer = (*standardOutputStream)(nil)

// newStandardOutputStream returns a stream that starts counting from zero.
func newStandardOutputStream(out io.Writer) *standardOutputStream {
	return &standardOutputStream{out: out}
}

// newStandardOutputStreamAt returns a stream that starts counting from the
// given position, which an incremental save needs: what it appends carries on
// from the end of the file it is updating.
func newStandardOutputStreamAt(out io.Writer, position int64) *standardOutputStream {
	return &standardOutputStream{out: out, position: position}
}

// Pos returns the current byte position.
func (s *standardOutputStream) Pos() int64 { return s.position }

// IsOnNewLine reports whether the last thing written was a line ending.
func (s *standardOutputStream) IsOnNewLine() bool { return s.onNewLine }

// SetOnNewLine records whether the last thing written was a line ending.
func (s *standardOutputStream) SetOnNewLine(newOnNewLine bool) { s.onNewLine = newOnNewLine }

// Write writes the bytes and advances the position.
func (s *standardOutputStream) Write(b []byte) (int, error) {
	s.SetOnNewLine(false)
	n, err := s.out.Write(b)
	s.position += int64(n)
	return n, err
}

// writeByte writes one byte, which is Java's write(int).
func (s *standardOutputStream) writeByte(b byte) error {
	_, err := s.Write([]byte{b})
	return err
}

// writeCRLF writes a carriage return and a line feed.
func (s *standardOutputStream) writeCRLF() error {
	_, err := s.Write(crlf)
	return err
}

// writeEOL writes a line ending, unless the last thing written was one.
func (s *standardOutputStream) writeEOL() error {
	if s.IsOnNewLine() {
		return nil
	}
	if _, err := s.Write(eol); err != nil {
		return err
	}
	s.SetOnNewLine(true)
	return nil
}

// writeLF writes a line feed.
func (s *standardOutputStream) writeLF() error {
	_, err := s.Write(lf)
	return err
}
