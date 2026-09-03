// Package ttf reads TrueType and OpenType fonts.
//
// Port of org.apache.fontbox.ttf.
package ttf

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"time"

	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// DataStream reads the primitives a TrueType file is built from.
//
// Port of the abstract org.apache.fontbox.ttf.TTFDataStream. Java uses
// inheritance to give the concrete streams the composite reads; the port keeps
// them as an interface for what varies and package functions for what does not.
//
// A read past the end reports -1 from Read, the way Java's does. The composite
// reads turn that into io.ErrUnexpectedEOF, which is what Java's EOFException
// becomes.
type DataStream interface {
	io.Closer

	// Read returns the next byte, or -1 at the end of the data.
	Read() (int, error)

	// ReadLong returns the next eight bytes as a signed 64-bit integer.
	ReadLong() (int64, error)

	// SeekTo moves to an absolute position. Java names this seek; the port
	// follows pdfio.SeekTo, since a Seek with no whence shadows io.Seeker.
	SeekTo(pos int64) error

	// ReadInto fills len bytes of b from off, returning how many it read, or
	// -1 at the end of the data.
	ReadInto(b []byte, off, length int) (int, error)

	// CurrentPosition returns the offset of the next byte to be read.
	CurrentPosition() int64

	// OriginalData returns a reader over the whole of the data.
	OriginalData() (io.Reader, error)

	// OriginalDataSize returns how much data there is.
	OriginalDataSize() int64

	// CreateSubView returns a read over the next length bytes, or nil where the
	// stream cannot give one.
	CreateSubView(length int64) pdfio.RandomAccessRead
}

// RandomAccessReadDataStream reads a TrueType file held in memory.
//
// Port of org.apache.fontbox.ttf.RandomAccessReadDataStream. As in Java the
// whole file is read into a byte slice up front: a font is read all over, and
// the tables point at each other by absolute offset.
type RandomAccessReadDataStream struct {
	length          int64
	data            []byte
	currentPosition int
}

var _ DataStream = (*RandomAccessReadDataStream)(nil)

// maxDataStreamLength is the largest font this can hold.
//
// Java refuses a stream longer than Integer.MAX_VALUE - 8, the largest array
// the JVM will allocate; see PDFBOX-5991. Go has no such limit, but the port
// keeps the check so that the same file is refused.
const maxDataStreamLength = math.MaxInt32 - 8

// NewRandomAccessReadDataStream reads the whole of source into memory.
func NewRandomAccessReadDataStream(source pdfio.RandomAccessRead) (*RandomAccessReadDataStream, error) {
	length, err := source.Length()
	if err != nil {
		return nil, err
	}
	if length > maxDataStreamLength {
		// PDFBOX-5991
		return nil, fmt.Errorf("ttf: stream is too long, size: %d", length)
	}

	data := make([]byte, length)
	remainingBytes := len(data)
	for remainingBytes > 0 {
		amountRead, err := source.Read(data[len(data)-remainingBytes:])
		if amountRead > 0 {
			remainingBytes -= amountRead
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		if amountRead <= 0 {
			break
		}
	}
	return &RandomAccessReadDataStream{length: length, data: data}, nil
}

// NewDataStreamFromReader reads everything the reader gives.
//
// Port of the RandomAccessReadDataStream(InputStream) constructor.
func NewDataStreamFromReader(r io.Reader) (*RandomAccessReadDataStream, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return &RandomAccessReadDataStream{data: data, length: int64(len(data))}, nil
}

// CurrentPosition returns the offset of the next byte to be read.
func (s *RandomAccessReadDataStream) CurrentPosition() int64 { return int64(s.currentPosition) }

// Close releases the stream. There is nothing to release, and closing twice is
// allowed; see PDFBOX-4242.
func (s *RandomAccessReadDataStream) Close() error { return nil }

// Read returns the next byte, or -1 at the end of the data.
func (s *RandomAccessReadDataStream) Read() (int, error) {
	if int64(s.currentPosition) >= s.length {
		return -1, nil
	}
	b := s.data[s.currentPosition]
	s.currentPosition++
	return int(b), nil
}

// ReadLong returns the next eight bytes as a signed 64-bit integer.
func (s *RandomAccessReadDataStream) ReadLong() (int64, error) {
	high, err := s.readInt()
	if err != nil {
		return 0, err
	}
	low, err := s.readInt()
	if err != nil {
		return 0, err
	}
	return int64(high)<<32 + int64(uint32(low)), nil
}

// readInt returns the next four bytes as a signed 32-bit integer. A read past
// the end contributes -1 to the arithmetic, as it does in Java.
func (s *RandomAccessReadDataStream) readInt() (int32, error) {
	var value int32
	for i := 0; i < 4; i++ {
		b, err := s.Read()
		if err != nil {
			return 0, err
		}
		value = value<<8 + int32(b)
	}
	return value, nil
}

// SeekTo moves to an absolute position. A position past the end lands at the
// end.
func (s *RandomAccessReadDataStream) SeekTo(pos int64) error {
	if pos < 0 {
		return fmt.Errorf("ttf: invalid position %d", pos)
	}
	if pos < s.length {
		s.currentPosition = int(pos)
	} else {
		s.currentPosition = int(s.length)
	}
	return nil
}

// ReadInto fills length bytes of b from off, returning how many it read, or -1
// at the end of the data.
func (s *RandomAccessReadDataStream) ReadInto(b []byte, off, length int) (int, error) {
	if int64(s.currentPosition) >= s.length {
		return -1, nil
	}
	remainingBytes := int(s.length - int64(s.currentPosition))
	bytesToRead := min(remainingBytes, length)
	copy(b[off:off+bytesToRead], s.data[s.currentPosition:s.currentPosition+bytesToRead])
	s.currentPosition += bytesToRead
	return bytesToRead, nil
}

// CreateSubView returns a read over the next length bytes.
//
// The buffer behind it is deliberately not closed: it has to stay alive for as
// long as the view does.
func (s *RandomAccessReadDataStream) CreateSubView(length int64) pdfio.RandomAccessRead {
	buffer := pdfio.NewReadBufferBytes(s.data)
	view, err := buffer.CreateView(int64(s.currentPosition), length)
	if err != nil {
		slog.Warn("ttf: could not create a SubView", "err", err)
		return nil
	}
	return view
}

// OriginalData returns a reader over the whole of the data.
func (s *RandomAccessReadDataStream) OriginalData() (io.Reader, error) {
	return pdfio.NewReader(pdfio.NewReadBufferBytes(s.data)), nil
}

// OriginalDataSize returns how much data there is.
func (s *RandomAccessReadDataStream) OriginalDataSize() int64 { return s.length }

// --- the composite reads, which Java puts on the abstract class ---

// Read32Fixed reads a 16.16 fixed point number.
func (s *RandomAccessReadDataStream) Read32Fixed() (float32, error) {
	whole, err := s.ReadSignedShort()
	if err != nil {
		return 0, err
	}
	fraction, err := s.ReadUnsignedShort()
	if err != nil {
		return 0, err
	}
	return float32(whole) + float32(fraction)/65536, nil
}

// ReadString reads length bytes as a string, decoded as ISO-8859-1, which maps
// each byte to the code point of the same value.
func (s *RandomAccessReadDataStream) ReadString(length int) (string, error) {
	data, err := s.ReadBytes(length)
	if err != nil {
		return "", err
	}
	runes := make([]rune, len(data))
	for i, b := range data {
		runes[i] = rune(b)
	}
	return string(runes), nil
}

// ReadSignedByte reads one byte as a signed value.
func (s *RandomAccessReadDataStream) ReadSignedByte() (int, error) {
	signedByte, err := s.Read()
	if err != nil {
		return 0, err
	}
	if signedByte <= 127 {
		return signedByte, nil
	}
	return signedByte - 256, nil
}

// ReadUnsignedByte reads one byte, failing at the end of the data.
func (s *RandomAccessReadDataStream) ReadUnsignedByte() (int, error) {
	unsignedByte, err := s.Read()
	if err != nil {
		return 0, err
	}
	if unsignedByte == -1 {
		return 0, fmt.Errorf("ttf: premature EOF: %w", io.ErrUnexpectedEOF)
	}
	return unsignedByte, nil
}

// ReadUnsignedInt reads four bytes as an unsigned 32-bit value.
func (s *RandomAccessReadDataStream) ReadUnsignedInt() (int64, error) {
	bytes := make([]int, 4)
	for i := range bytes {
		b, err := s.Read()
		if err != nil {
			return 0, err
		}
		bytes[i] = b
	}
	// Java tests only the last byte, so a stream that ends between the first
	// and the fourth is caught here and nowhere earlier.
	if bytes[3] < 0 {
		return 0, fmt.Errorf("ttf: EOF at %d, b1: %d, b2: %d, b3: %d, b4: %d: %w",
			s.CurrentPosition(), bytes[0], bytes[1], bytes[2], bytes[3], io.ErrUnexpectedEOF)
	}
	return int64(bytes[0])<<24 + int64(bytes[1])<<16 + int64(bytes[2])<<8 + int64(bytes[3]), nil
}

// ReadUnsignedShort reads two bytes as an unsigned 16-bit value.
func (s *RandomAccessReadDataStream) ReadUnsignedShort() (int, error) {
	b1, err := s.Read()
	if err != nil {
		return 0, err
	}
	b2, err := s.Read()
	if err != nil {
		return 0, err
	}
	if (b1 | b2) < 0 {
		return 0, fmt.Errorf("ttf: EOF at %d, b1: %d, b2: %d: %w",
			s.CurrentPosition(), b1, b2, io.ErrUnexpectedEOF)
	}
	return b1<<8 + b2, nil
}

// ReadUnsignedByteArray reads length bytes, each of which may be -1 at the end
// of the data, as Java's does.
func (s *RandomAccessReadDataStream) ReadUnsignedByteArray(length int) ([]int, error) {
	array := make([]int, length)
	for i := range array {
		value, err := s.Read()
		if err != nil {
			return nil, err
		}
		array[i] = value
	}
	return array, nil
}

// ReadUnsignedShortArray reads length unsigned 16-bit values.
func (s *RandomAccessReadDataStream) ReadUnsignedShortArray(length int) ([]int, error) {
	array := make([]int, length)
	for i := range array {
		value, err := s.ReadUnsignedShort()
		if err != nil {
			return nil, err
		}
		array[i] = value
	}
	return array, nil
}

// ReadSignedShort reads two bytes as a signed 16-bit value.
func (s *RandomAccessReadDataStream) ReadSignedShort() (int16, error) {
	value, err := s.ReadUnsignedShort()
	if err != nil {
		return 0, err
	}
	return int16(value), nil
}

// ttfEpoch is the instant TrueType dates count from: midnight, 1 January 1904,
// UTC.
var ttfEpoch = time.Date(1904, time.January, 1, 0, 0, 0, 0, time.UTC)

// ReadInternationalDate reads a date, held as seconds since the start of 1904.
func (s *RandomAccessReadDataStream) ReadInternationalDate() (time.Time, error) {
	secondsSince1904, err := s.ReadLong()
	if err != nil {
		return time.Time{}, err
	}
	// Java builds this through a Calendar in UTC and adds the seconds as
	// milliseconds, which is the same instant.
	return ttfEpoch.Add(time.Duration(secondsSince1904) * time.Second), nil
}

// ReadTag reads the four-character tag that names a table.
func (s *RandomAccessReadDataStream) ReadTag() (string, error) {
	data, err := s.ReadBytes(4)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ReadBytes reads exactly numberOfBytes bytes, failing if the data runs out
// first.
func (s *RandomAccessReadDataStream) ReadBytes(numberOfBytes int) ([]byte, error) {
	data := make([]byte, numberOfBytes)
	totalAmountRead := 0
	for totalAmountRead < numberOfBytes {
		amountRead, err := s.ReadInto(data, totalAmountRead, numberOfBytes-totalAmountRead)
		if err != nil {
			return nil, err
		}
		if amountRead == -1 {
			break
		}
		totalAmountRead += amountRead
	}
	if totalAmountRead != numberOfBytes {
		return nil, fmt.Errorf("ttf: unexpected end of TTF stream reached: %w", io.ErrUnexpectedEOF)
	}
	return data, nil
}

// --- the composite reads as package functions, so that they work against any
// DataStream rather than only the in-memory one. Java puts them on the abstract
// class, which Go embedding cannot reproduce for an interface.

func readFixed(s DataStream) (float32, error) {
	whole, err := readSignedShort(s)
	if err != nil {
		return 0, err
	}
	fraction, err := readUnsignedShort(s)
	if err != nil {
		return 0, err
	}
	return float32(whole) + float32(fraction)/65536, nil
}

func readSignedByte(s DataStream) (int, error) {
	signedByte, err := s.Read()
	if err != nil {
		return 0, err
	}
	if signedByte <= 127 {
		return signedByte, nil
	}
	return signedByte - 256, nil
}

func readUnsignedByte(s DataStream) (int, error) {
	unsignedByte, err := s.Read()
	if err != nil {
		return 0, err
	}
	if unsignedByte == -1 {
		return 0, fmt.Errorf("ttf: premature EOF: %w", io.ErrUnexpectedEOF)
	}
	return unsignedByte, nil
}

func readUnsignedInt(s DataStream) (int64, error) {
	values := make([]int, 4)
	for i := range values {
		b, err := s.Read()
		if err != nil {
			return 0, err
		}
		values[i] = b
	}
	if values[3] < 0 {
		return 0, fmt.Errorf("ttf: EOF at %d, b1: %d, b2: %d, b3: %d, b4: %d: %w",
			s.CurrentPosition(), values[0], values[1], values[2], values[3], io.ErrUnexpectedEOF)
	}
	return int64(values[0])<<24 + int64(values[1])<<16 + int64(values[2])<<8 + int64(values[3]), nil
}

func readUnsignedShort(s DataStream) (int, error) {
	b1, err := s.Read()
	if err != nil {
		return 0, err
	}
	b2, err := s.Read()
	if err != nil {
		return 0, err
	}
	if (b1 | b2) < 0 {
		return 0, fmt.Errorf("ttf: EOF at %d, b1: %d, b2: %d: %w",
			s.CurrentPosition(), b1, b2, io.ErrUnexpectedEOF)
	}
	return b1<<8 + b2, nil
}

func readSignedShort(s DataStream) (int16, error) {
	value, err := readUnsignedShort(s)
	if err != nil {
		return 0, err
	}
	return int16(value), nil
}

func readUnsignedShortArray(s DataStream, length int) ([]int, error) {
	array := make([]int, length)
	for i := range array {
		value, err := readUnsignedShort(s)
		if err != nil {
			return nil, err
		}
		array[i] = value
	}
	return array, nil
}

func readInternationalDate(s DataStream) (time.Time, error) {
	secondsSince1904, err := s.ReadLong()
	if err != nil {
		return time.Time{}, err
	}
	return ttfEpoch.Add(time.Duration(secondsSince1904) * time.Second), nil
}

func readBytes(s DataStream, numberOfBytes int) ([]byte, error) {
	data := make([]byte, numberOfBytes)
	totalAmountRead := 0
	for totalAmountRead < numberOfBytes {
		amountRead, err := s.ReadInto(data, totalAmountRead, numberOfBytes-totalAmountRead)
		if err != nil {
			return nil, err
		}
		if amountRead == -1 {
			break
		}
		totalAmountRead += amountRead
	}
	if totalAmountRead != numberOfBytes {
		return nil, fmt.Errorf("ttf: unexpected end of TTF stream reached: %w", io.ErrUnexpectedEOF)
	}
	return data, nil
}

func readString(s DataStream, length int) (string, error) {
	data, err := readBytes(s, length)
	if err != nil {
		return "", err
	}
	runes := make([]rune, len(data))
	for i, b := range data {
		runes[i] = rune(b)
	}
	return string(runes), nil
}

func readTag(s DataStream) (string, error) {
	data, err := readBytes(s, 4)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func readUnsignedByteArray(s DataStream, length int) ([]int, error) {
	array := make([]int, length)
	for i := range array {
		value, err := readUnsignedByte(s)
		if err != nil {
			return nil, err
		}
		array[i] = value
	}
	return array, nil
}
