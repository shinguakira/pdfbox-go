// Package cff reads Compact Font Format fonts and their charstrings.
//
// Port of org.apache.fontbox.cff.
package cff

import (
	"errors"
	"fmt"
	"io"

	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// DataInput defines some functionality to read a CFF font.
//
// Port of the org.apache.fontbox.cff.DataInput interface. Its five default
// methods -- ReadShort, ReadUnsignedShort, ReadInt and ReadOffset -- are given
// to an implementation by embedding dataInputBase, which Go needs because an
// interface cannot carry an implementation.
type DataInput interface {
	// HasRemaining determines if there are any bytes left to read or not.
	HasRemaining() (bool, error)

	// Position returns the current position.
	Position() (int, error)

	// SetPosition sets the current position to the given value, reporting an
	// error if the new position is out of range.
	SetPosition(position int) error

	// ReadSignedByte reads one single byte from the buffer.
	//
	// Java calls this readByte and gives back a signed byte. Go reserves that
	// name for the io.ByteReader shape, so the pair here is ReadSignedByte and
	// ReadUnsignedByte.
	ReadSignedByte() (int8, error)

	// ReadUnsignedByte reads one single unsigned byte from the buffer.
	ReadUnsignedByte() (int, error)

	// PeekUnsignedByte peeks one single unsigned byte from the buffer, offset
	// being the offset to the byte to be peeked.
	PeekUnsignedByte(offset int) (int, error)

	// ReadShort reads one single short value from the buffer.
	ReadShort() (int16, error)

	// ReadUnsignedShort reads one single unsigned short (2 bytes) value from
	// the buffer.
	ReadUnsignedShort() (int, error)

	// ReadInt reads one single int (4 bytes) from the buffer.
	ReadInt() (int32, error)

	// ReadBytes reads a number of single byte values from the buffer.
	ReadBytes(length int) ([]byte, error)

	// Length returns the number of bytes in the source.
	Length() (int, error)

	// ReadOffset reads the offset from the buffer, offSize being the given
	// offsize.
	ReadOffset(offSize int) (int, error)
}

// dataInputBase carries the default methods of the DataInput interface, self
// being the implementation they read through.
type dataInputBase struct {
	self DataInput
}

// ReadShort reads one single short value from the buffer.
func (d dataInputBase) ReadShort() (int16, error) {
	value, err := d.self.ReadUnsignedShort()
	if err != nil {
		return 0, err
	}
	return int16(value), nil
}

// ReadUnsignedShort reads one single unsigned short (2 bytes) value from the
// buffer.
func (d dataInputBase) ReadUnsignedShort() (int, error) {
	b1, err := d.self.ReadUnsignedByte()
	if err != nil {
		return 0, err
	}
	b2, err := d.self.ReadUnsignedByte()
	if err != nil {
		return 0, err
	}
	return b1<<8 | b2, nil
}

// ReadInt reads one single int (4 bytes) from the buffer.
func (d dataInputBase) ReadInt() (int32, error) {
	var value int32
	for i := 0; i < 4; i++ {
		b, err := d.self.ReadUnsignedByte()
		if err != nil {
			return 0, err
		}
		// Java or-s the four bytes into an int, so the top byte carries the
		// sign; the arithmetic has to be done at the same width.
		value |= int32(b) << (24 - 8*i)
	}
	return value, nil
}

// ReadOffset reads the offset from the buffer, offSize being the given offsize.
func (d dataInputBase) ReadOffset(offSize int) (int, error) {
	value := 0
	for i := 0; i < offSize; i++ {
		b, err := d.self.ReadUnsignedByte()
		if err != nil {
			return 0, err
		}
		value = value<<8 | b
	}
	return value, nil
}

// DataInputByteArray implements the DataInput interface using a byte buffer as
// source.
//
// Port of org.apache.fontbox.cff.DataInputByteArray.
type DataInputByteArray struct {
	dataInputBase

	inputBuffer    []byte
	bufferPosition int
}

var _ DataInput = (*DataInputByteArray)(nil)

// NewDataInputByteArray returns a DataInput over the given buffer.
func NewDataInputByteArray(buffer []byte) *DataInputByteArray {
	d := &DataInputByteArray{inputBuffer: buffer}
	d.self = d
	return d
}

// HasRemaining determines if there are any bytes left to read or not.
func (d *DataInputByteArray) HasRemaining() (bool, error) {
	return d.bufferPosition < len(d.inputBuffer), nil
}

// Position returns the current position.
func (d *DataInputByteArray) Position() (int, error) { return d.bufferPosition, nil }

// SetPosition sets the current position to the given value.
func (d *DataInputByteArray) SetPosition(position int) error {
	if position < 0 {
		return errors.New("position is negative")
	}
	if position >= len(d.inputBuffer) {
		return fmt.Errorf("New position is out of range %d >= %d", position, len(d.inputBuffer))
	}
	d.bufferPosition = position
	return nil
}

// ReadSignedByte reads one single byte from the buffer, Java's readByte.
func (d *DataInputByteArray) ReadSignedByte() (int8, error) {
	if d.bufferPosition >= len(d.inputBuffer) {
		return 0, errors.New("End off buffer reached")
	}
	value := int8(d.inputBuffer[d.bufferPosition])
	d.bufferPosition++
	return value, nil
}

// ReadUnsignedByte reads one single unsigned byte from the buffer.
func (d *DataInputByteArray) ReadUnsignedByte() (int, error) {
	if d.bufferPosition >= len(d.inputBuffer) {
		return 0, errors.New("End off buffer reached")
	}
	value := int(d.inputBuffer[d.bufferPosition]) & 0xff
	d.bufferPosition++
	return value, nil
}

// PeekUnsignedByte peeks one single unsigned byte from the buffer.
func (d *DataInputByteArray) PeekUnsignedByte(offset int) (int, error) {
	if offset < 0 {
		return 0, errors.New("offset is negative")
	}
	if d.bufferPosition+offset >= len(d.inputBuffer) {
		return 0, fmt.Errorf("Offset position is out of range %d >= %d",
			d.bufferPosition+offset, len(d.inputBuffer))
	}
	return int(d.inputBuffer[d.bufferPosition+offset]) & 0xff, nil
}

// ReadBytes reads a number of single byte values from the buffer.
func (d *DataInputByteArray) ReadBytes(length int) ([]byte, error) {
	if length < 0 {
		return nil, errors.New("length is negative")
	}
	if len(d.inputBuffer)-d.bufferPosition < length {
		return nil, errors.New("Premature end of buffer reached")
	}
	bytes := make([]byte, length)
	copy(bytes, d.inputBuffer[d.bufferPosition:d.bufferPosition+length])
	d.bufferPosition += length
	return bytes, nil
}

// Length returns the number of bytes in the source.
func (d *DataInputByteArray) Length() (int, error) { return len(d.inputBuffer), nil }

// DataInputRandomAccessRead implements the DataInput interface using a
// RandomAccessRead as source.
//
// Note: things can get hairy when the underlying buffer is larger than the
// largest int32. Straight forward reading may work, but Position and
// SetPosition may have problems.
//
// Port of org.apache.fontbox.cff.DataInputRandomAccessRead.
type DataInputRandomAccessRead struct {
	dataInputBase

	randomAccessRead pdfio.RandomAccessRead
}

var _ DataInput = (*DataInputRandomAccessRead)(nil)

// NewDataInputRandomAccessRead returns a DataInput over the given source.
func NewDataInputRandomAccessRead(randomAccessRead pdfio.RandomAccessRead) *DataInputRandomAccessRead {
	d := &DataInputRandomAccessRead{randomAccessRead: randomAccessRead}
	d.self = d
	return d
}

// available is Java's RandomAccessRead.available, the number of bytes between
// the cursor and the end.
func (d *DataInputRandomAccessRead) available() (int64, error) {
	position, err := d.randomAccessRead.Position()
	if err != nil {
		return 0, err
	}
	length, err := d.randomAccessRead.Length()
	if err != nil {
		return 0, err
	}
	return length - position, nil
}

// HasRemaining determines if there are any bytes left to read or not.
func (d *DataInputRandomAccessRead) HasRemaining() (bool, error) {
	available, err := d.available()
	if err != nil {
		return false, err
	}
	return available > 0, nil
}

// Position returns the current position.
func (d *DataInputRandomAccessRead) Position() (int, error) {
	position, err := d.randomAccessRead.Position()
	if err != nil {
		return 0, err
	}
	return int(position), nil
}

// SetPosition sets the current absolute position to the given value. You
// cannot use SetPosition(-20) to move 20 bytes back.
func (d *DataInputRandomAccessRead) SetPosition(position int) error {
	if position < 0 {
		return errors.New("position is negative")
	}
	length, err := d.randomAccessRead.Length()
	if err != nil {
		return err
	}
	if int64(position) >= length {
		return fmt.Errorf("New position is out of range %d >= %d", position, length)
	}
	_, err = d.randomAccessRead.Seek(int64(position), io.SeekStart)
	return err
}

// ReadSignedByte reads one single byte from the buffer, Java's readByte.
func (d *DataInputRandomAccessRead) ReadSignedByte() (int8, error) {
	hasRemaining, err := d.HasRemaining()
	if err != nil {
		return 0, err
	}
	if !hasRemaining {
		return 0, errors.New("End of buffer reached!")
	}
	b, err := d.randomAccessRead.ReadByte()
	if err != nil {
		return 0, err
	}
	return int8(b), nil
}

// ReadUnsignedByte reads one single unsigned byte from the buffer.
func (d *DataInputRandomAccessRead) ReadUnsignedByte() (int, error) {
	hasRemaining, err := d.HasRemaining()
	if err != nil {
		return 0, err
	}
	if !hasRemaining {
		return 0, errors.New("End of buffer reached!")
	}
	b, err := d.randomAccessRead.ReadByte()
	if err != nil {
		return 0, err
	}
	return int(b), nil
}

// PeekUnsignedByte peeks one single unsigned byte from the buffer, offset being
// the offset to the byte to be peeked.
func (d *DataInputRandomAccessRead) PeekUnsignedByte(offset int) (int, error) {
	if offset < 0 {
		return 0, errors.New("offset is negative")
	}
	currentPosition, err := d.randomAccessRead.Position()
	if err != nil {
		return 0, err
	}
	if offset == 0 {
		// Java's RandomAccessRead.peek reads one byte and rewinds, and reports
		// -1 at the end of the source rather than failing.
		b, err := d.randomAccessRead.ReadByte()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return -1, nil
			}
			return 0, err
		}
		if _, err := d.randomAccessRead.Seek(currentPosition, io.SeekStart); err != nil {
			return 0, err
		}
		return int(b), nil
	}
	length, err := d.randomAccessRead.Length()
	if err != nil {
		return 0, err
	}
	if currentPosition+int64(offset) >= length {
		return 0, fmt.Errorf("Offset position is out of range %d >= %d",
			currentPosition+int64(offset), length)
	}
	if _, err := d.randomAccessRead.Seek(currentPosition+int64(offset), io.SeekStart); err != nil {
		return 0, err
	}
	peekValue, err := d.randomAccessRead.ReadByte()
	if err != nil {
		return 0, err
	}
	if _, err := d.randomAccessRead.Seek(currentPosition, io.SeekStart); err != nil {
		return 0, err
	}
	return int(peekValue), nil
}

// ReadBytes reads a number of single byte values from the buffer.
//
// Note: when ReadBytes(5) is called, but there are only 3 bytes available, the
// caller gets an error, not the 3 bytes.
func (d *DataInputRandomAccessRead) ReadBytes(length int) ([]byte, error) {
	if length < 0 {
		return nil, errors.New("length is negative")
	}
	bytes := make([]byte, length)
	if err := pdfio.ReadFully(d.randomAccessRead, bytes); err != nil {
		return nil, err
	}
	return bytes, nil
}

// Length returns the number of bytes in the source.
func (d *DataInputRandomAccessRead) Length() (int, error) {
	length, err := d.randomAccessRead.Length()
	if err != nil {
		return 0, err
	}
	return int(length), nil
}
