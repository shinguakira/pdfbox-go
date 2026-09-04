package ttf

import "time"

// reader reads a sequence of values from a DataStream, holding the first error
// rather than returning one from every call.
//
// A Java table reader is a straight run of assignments, because every read
// throws. Writing that in Go with an error check between each line buries the
// shape of the table in error handling; this keeps the shape and checks once at
// the end. After an error every subsequent read is a no-op returning zero, so
// nothing runs on a half-read table.
type reader struct {
	data DataStream
	err  error
}

func newReader(data DataStream) *reader {
	return &reader{data: data}
}

func (r *reader) fixed() float32 {
	if r.err != nil {
		return 0
	}
	value, err := readFixed(r.data)
	r.err = err
	return value
}

func (r *reader) unsignedInt() int64 {
	if r.err != nil {
		return 0
	}
	value, err := readUnsignedInt(r.data)
	r.err = err
	return value
}

func (r *reader) unsignedShort() int {
	if r.err != nil {
		return 0
	}
	value, err := readUnsignedShort(r.data)
	r.err = err
	return value
}

func (r *reader) signedShort() int16 {
	if r.err != nil {
		return 0
	}
	value, err := readSignedShort(r.data)
	r.err = err
	return value
}

func (r *reader) unsignedByte() int {
	if r.err != nil {
		return 0
	}
	value, err := readUnsignedByte(r.data)
	r.err = err
	return value
}

func (r *reader) signedByte() int {
	if r.err != nil {
		return 0
	}
	value, err := readSignedByte(r.data)
	r.err = err
	return value
}

func (r *reader) long() int64 {
	if r.err != nil {
		return 0
	}
	value, err := r.data.ReadLong()
	r.err = err
	return value
}

func (r *reader) date() time.Time {
	if r.err != nil {
		return time.Time{}
	}
	value, err := readInternationalDate(r.data)
	r.err = err
	return value
}

func (r *reader) str(length int) string {
	if r.err != nil {
		return ""
	}
	value, err := readString(r.data, length)
	r.err = err
	return value
}

func (r *reader) bytes(length int) []byte {
	if r.err != nil {
		return nil
	}
	value, err := readBytes(r.data, length)
	r.err = err
	return value
}

func (r *reader) unsignedShortArray(length int) []int {
	if r.err != nil {
		return nil
	}
	value, err := readUnsignedShortArray(r.data, length)
	r.err = err
	return value
}

func (r *reader) seek(pos int64) {
	if r.err != nil {
		return
	}
	r.err = r.data.SeekTo(pos)
}

// position returns where the stream is, which several tables need mid-read.
func (r *reader) position() int64 {
	return r.data.CurrentPosition()
}
