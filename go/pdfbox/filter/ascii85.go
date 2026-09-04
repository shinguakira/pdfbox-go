package filter

import (
	"bufio"
	"errors"
	"io"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// ASCII85 decodes data encoded in an ASCII base-85 representation, reproducing
// the original binary data.
//
// Port of org.apache.pdfbox.filter.ASCII85Filter together with
// ASCII85InputStream and ASCII85OutputStream.
type ASCII85 struct{}

var _ Filter = ASCII85{}

// Decode expands the base-85 groups.
func (ASCII85) Decode(w io.Writer, r io.Reader, parameters *cos.Dictionary,
	index int) (DecodeResult, error) {
	result := DecodeResult{Parameters: parameters}
	if _, err := io.Copy(w, newASCII85Reader(r)); err != nil {
		return result, err
	}
	return result, nil
}

// Encode writes the base-85 groups.
func (ASCII85) Encode(w io.Writer, r io.Reader, parameters *cos.Dictionary) error {
	out := newASCII85Writer(w)
	if _, err := io.Copy(out, r); err != nil {
		return err
	}
	// Java closes the stream, which flushes the tail and the terminator.
	return out.Close()
}

// The characters ASCII85InputStream and ASCII85OutputStream single out.
const (
	ascii85Terminator = '~'
	ascii85Offset     = '!'
	ascii85Newline    = '\n'
	ascii85Return     = '\r'
	ascii85Space      = ' '
	ascii85PaddingU   = 'u'
	ascii85Z          = 'z'
)

// ascii85Reader is an ASCII85 stream.
//
// Port of org.apache.pdfbox.filter.ASCII85InputStream. Java's read() works a
// byte at a time out of a four byte group, and the port keeps that shape rather
// than decoding in bulk, because the group boundaries are where the terminator
// and the padding rules live.
//
// The bytes it reads are held as int8, not byte, because every comparison the
// Java makes is on a signed byte and one of them shows: `int zz = (byte)
// in.read(); if (zz == -1)` narrows before it tests for the end of the stream,
// so a data byte 0xFF ends the stream instead of being rejected. See
// migration/JAVA-BUGS.md.
type ascii85Reader struct {
	in    *bufio.Reader
	index int
	n     int
	eof   bool
	ascii [5]int8
	b     [4]byte
}

var _ io.Reader = (*ascii85Reader)(nil)

func newASCII85Reader(r io.Reader) *ascii85Reader {
	return &ascii85Reader{in: bufio.NewReader(r)}
}

// errInvalidASCII85 is Java's IOException("Invalid data in Ascii85 stream").
var errInvalidASCII85 = errors.New("Invalid data in Ascii85 stream")

// readSignificant reads the next byte that is not a line break or a space.
//
// It reports ok false where Java's `int zz = (byte) in.read(); if (zz == -1)`
// takes the end of the stream, which is either a real end or a 0xFF byte.
func (a *ascii85Reader) readSignificant() (z int8, ok bool) {
	for {
		c, err := a.in.ReadByte()
		if err != nil {
			return 0, false
		}
		zz := int8(c)
		if zz == -1 {
			return 0, false
		}
		z = zz
		if z != ascii85Newline && z != ascii85Return && z != ascii85Space {
			return z, true
		}
	}
}

// readByte reads the next decoded byte.
//
// Port of ASCII85InputStream.read(). It returns io.EOF where Java returns -1.
func (a *ascii85Reader) readByte() (byte, error) {
	if a.index >= a.n {
		if a.eof {
			return 0, io.EOF
		}
		a.index = 0

		z, ok := a.readSignificant()
		if !ok {
			a.eof = true
			return 0, io.EOF
		}

		switch {
		case z == ascii85Terminator:
			a.eof = true
			a.n = 0
			return 0, io.EOF

		case z == ascii85Z:
			a.b[0], a.b[1], a.b[2], a.b[3] = 0, 0, 0, 0
			a.n = 4

		default:
			a.ascii[0] = z // may be EOF here....
			k := 1
			for ; k < 5; k++ {
				z, ok = a.readSignificant()
				if !ok {
					a.eof = true
					return 0, io.EOF
				}
				a.ascii[k] = z
				if z == ascii85Terminator {
					// don't include ~ as padding byte
					a.ascii[k] = ascii85PaddingU
					break
				}
			}
			a.n = k - 1
			if a.n == 0 {
				a.eof = true
				return 0, io.EOF
			}
			if k < 5 {
				for k++; k < 5; k++ {
					// use 'u' for padding
					a.ascii[k] = ascii85PaddingU
				}
				a.eof = true
			}

			// decode stream
			var t int64
			for k = 0; k < 5; k++ {
				z = a.ascii[k] - ascii85Offset
				if z < 0 || z > 93 {
					a.n = 0
					a.eof = true
					return 0, errInvalidASCII85
				}
				t = t*85 + int64(z)
			}
			for k = 3; k >= 0; k-- {
				a.b[k] = byte(t & 0xFF)
				t = int64(uint64(t) >> 8)
			}
		}
	}
	b := a.b[a.index]
	a.index++
	return b, nil
}

// Read fills data with decoded bytes.
//
// Port of ASCII85InputStream.read(byte[], int, int), which stops at the first
// end of stream and returns what it had.
func (a *ascii85Reader) Read(data []byte) (int, error) {
	if a.eof && a.index >= a.n {
		return 0, io.EOF
	}
	for i := range data {
		if a.index < a.n {
			data[i] = a.b[a.index]
			a.index++
			continue
		}
		t, err := a.readByte()
		if err == io.EOF {
			return i, nil
		}
		if err != nil {
			return i, err
		}
		data[i] = t
	}
	return len(data), nil
}

// ascii85Writer writes an ASCII85 stream.
//
// Port of org.apache.pdfbox.filter.ASCII85OutputStream.
type ascii85Writer struct {
	out       *bufio.Writer
	lineBreak int
	count     int
	indata    [4]byte
	outdata   [5]byte
	maxline   int
	flushed   bool

	// terminator is Java's field of the same name. setTerminator and
	// setLineLength have no caller in PDFBox, so the port keeps the values the
	// constructor sets and leaves the two setters out.
	terminator byte
}

var _ io.Writer = (*ascii85Writer)(nil)

func newASCII85Writer(w io.Writer) *ascii85Writer {
	return &ascii85Writer{
		out:        bufio.NewWriter(w),
		lineBreak:  36 * 2,
		maxline:    36 * 2,
		flushed:    true,
		terminator: ascii85Terminator,
	}
}

// transformASCII85 turns the four bytes of indata into the five of outdata.
//
// Java builds the word with int arithmetic over signed bytes and only then
// widens it to a long: `((((indata[0] << 8) | (indata[1] & 0xFF)) << 16) |
// ((indata[2] & 0xFF) << 8) | (indata[3] & 0xFF)) & 0xFFFFFFFFL`. The shifts
// overflow a 32 bit int and the mask takes the low 32 bits back, which is the
// four bytes big endian; the port writes that out rather than assuming it.
func (a *ascii85Writer) transformASCII85() {
	word32 := ((int32(int8(a.indata[0]))<<8 | int32(a.indata[1])) << 16) |
		int32(a.indata[2])<<8 | int32(a.indata[3])
	word := int64(uint32(word32))
	if word == 0 {
		a.outdata[0] = ascii85Z
		a.outdata[1] = 0
		return
	}
	var x int64
	x = word / (85 * 85 * 85 * 85)
	a.outdata[0] = byte(x + ascii85Offset)
	word -= x * 85 * 85 * 85 * 85
	x = word / (85 * 85 * 85)
	a.outdata[1] = byte(x + ascii85Offset)
	word -= x * 85 * 85 * 85
	x = word / (85 * 85)
	a.outdata[2] = byte(x + ascii85Offset)
	word -= x * 85 * 85
	x = word / 85
	a.outdata[3] = byte(x + ascii85Offset)
	a.outdata[4] = byte(word%85 + ascii85Offset)
}

// writeByte is Java's write(int b).
func (a *ascii85Writer) writeByte(b byte) error {
	a.flushed = false
	a.indata[a.count] = b
	a.count++
	if a.count < 4 {
		return nil
	}
	a.transformASCII85()
	for i := 0; i < 5; i++ {
		if a.outdata[i] == 0 {
			break
		}
		if err := a.out.WriteByte(a.outdata[i]); err != nil {
			return err
		}
		a.lineBreak--
		if a.lineBreak == 0 {
			if err := a.out.WriteByte(ascii85Newline); err != nil {
				return err
			}
			a.lineBreak = a.maxline
		}
	}
	a.count = 0
	return nil
}

func (a *ascii85Writer) Write(p []byte) (int, error) {
	for i, b := range p {
		if err := a.writeByte(b); err != nil {
			return i, err
		}
	}
	return len(p), nil
}

// Flush is Java's flush(), which writes the partial group and the terminator.
//
// Java's flushed flag starts true and only a write clears it, so a stream that
// took no bytes at all writes nothing -- terminator included. An empty input
// therefore encodes to an empty stream.
func (a *ascii85Writer) Flush() error {
	if a.flushed {
		return a.out.Flush()
	}
	if a.count > 0 {
		for i := a.count; i < 4; i++ {
			a.indata[i] = 0
		}
		a.transformASCII85()
		if a.outdata[0] == ascii85Z {
			for i := 0; i < 5; i++ { // expand 'z',
				a.outdata[i] = ascii85Offset
			}
		}
		for i := 0; i < a.count+1; i++ {
			if err := a.out.WriteByte(a.outdata[i]); err != nil {
				return err
			}
			a.lineBreak--
			if a.lineBreak == 0 {
				if err := a.out.WriteByte(ascii85Newline); err != nil {
					return err
				}
				a.lineBreak = a.maxline
			}
		}
	}
	// Java decrements the counter once more here without resetting it first,
	// and writes a newline where that happens to reach zero.
	a.lineBreak--
	if a.lineBreak == 0 {
		if err := a.out.WriteByte(ascii85Newline); err != nil {
			return err
		}
	}
	if err := a.out.WriteByte(a.terminator); err != nil {
		return err
	}
	if err := a.out.WriteByte('>'); err != nil {
		return err
	}
	if err := a.out.WriteByte(ascii85Newline); err != nil {
		return err
	}
	a.count = 0
	a.lineBreak = a.maxline
	a.flushed = true
	return a.out.Flush()
}

// Close flushes, which is what Java's close does before it releases the
// buffers.
func (a *ascii85Writer) Close() error { return a.Flush() }
