package ttf

import (
	"errors"
	"fmt"
	"io"

	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// unbufferedDataStream works with a RandomAccessRead directly, where
// RandomAccessReadDataStream pre-loads the whole of it into a byte slice.
//
// Performance: it is much faster if most of the buffer is skipped, and slower
// if the whole buffer is read.
//
// Port of org.apache.fontbox.ttf.RandomAccessReadUnbufferedDataStream.
type unbufferedDataStream struct {
	length           int64
	randomAccessRead pdfio.RandomAccessRead
}

var _ DataStream = (*unbufferedDataStream)(nil)

func newUnbufferedDataStream(randomAccessRead pdfio.RandomAccessRead) (*unbufferedDataStream, error) {
	length, err := randomAccessRead.Length()
	if err != nil {
		return nil, err
	}
	return &unbufferedDataStream{length: length, randomAccessRead: randomAccessRead}, nil
}

// CurrentPosition returns the offset of the next byte to be read.
func (s *unbufferedDataStream) CurrentPosition() int64 {
	position, err := s.randomAccessRead.Position()
	if err != nil {
		return 0
	}
	return position
}

// Close closes the underlying resources.
func (s *unbufferedDataStream) Close() error { return s.randomAccessRead.Close() }

// Read returns the next byte, or -1 at the end of the data.
func (s *unbufferedDataStream) Read() (int, error) {
	b, err := s.randomAccessRead.ReadByte()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return -1, nil
		}
		return -1, err
	}
	return int(b), nil
}

// ReadLong returns the next eight bytes as a signed 64-bit integer.
func (s *unbufferedDataStream) ReadLong() (int64, error) {
	high, err := s.readInt()
	if err != nil {
		return 0, err
	}
	low, err := s.readInt()
	if err != nil {
		return 0, err
	}
	return int64(high)<<32 | int64(uint32(low)), nil
}

func (s *unbufferedDataStream) readInt() (int32, error) {
	var value int32
	for i := 0; i < 4; i++ {
		b, err := s.Read()
		if err != nil {
			return 0, err
		}
		value |= int32(b) << (24 - 8*i)
	}
	return value, nil
}

// SeekTo moves to an absolute position.
func (s *unbufferedDataStream) SeekTo(pos int64) error {
	_, err := s.randomAccessRead.Seek(pos, io.SeekStart)
	return err
}

// ReadInto fills length bytes of b from off.
func (s *unbufferedDataStream) ReadInto(b []byte, off, length int) (int, error) {
	n, err := s.randomAccessRead.Read(b[off : off+length])
	if err != nil && errors.Is(err, io.EOF) && n == 0 {
		return -1, nil
	}
	return n, err
}

// OriginalData returns a reader over the whole of the data. Its lifetime is
// bound by this stream's, and it does not close the underlying source.
func (s *unbufferedDataStream) OriginalData() (io.Reader, error) {
	view, err := s.randomAccessRead.CreateView(0, s.length)
	if err != nil {
		return nil, err
	}
	return pdfio.NewReader(view), nil
}

// OriginalDataSize returns how much data there is.
func (s *unbufferedDataStream) OriginalDataSize() int64 { return s.length }

// CreateSubView returns a read over the next length bytes.
func (s *unbufferedDataStream) CreateSubView(length int64) pdfio.RandomAccessRead {
	position, err := s.randomAccessRead.Position()
	if err != nil {
		return nil
	}
	view, err := s.randomAccessRead.CreateView(position, length)
	if err != nil {
		// Java asserts "Please implement createView()" and returns null.
		return nil
	}
	return view
}

// ttcDataStream is a wrapper for a TTF stream inside a TTC file. It does not
// close the underlying shared stream.
//
// Port of org.apache.fontbox.ttf.TTCDataStream.
type ttcDataStream struct {
	stream DataStream
}

var _ DataStream = (*ttcDataStream)(nil)

func newTTCDataStream(stream DataStream) *ttcDataStream {
	return &ttcDataStream{stream: stream}
}

func (s *ttcDataStream) Read() (int, error) { return s.stream.Read() }

func (s *ttcDataStream) ReadLong() (int64, error) { return s.stream.ReadLong() }

// Close does not close the underlying stream, as it is shared by all fonts from
// the same TTC; TrueTypeCollection.Close must be called instead.
func (s *ttcDataStream) Close() error { return nil }

func (s *ttcDataStream) SeekTo(pos int64) error { return s.stream.SeekTo(pos) }

func (s *ttcDataStream) ReadInto(b []byte, off, length int) (int, error) {
	return s.stream.ReadInto(b, off, length)
}

func (s *ttcDataStream) CurrentPosition() int64 { return s.stream.CurrentPosition() }

func (s *ttcDataStream) OriginalData() (io.Reader, error) { return s.stream.OriginalData() }

func (s *ttcDataStream) OriginalDataSize() int64 { return s.stream.OriginalDataSize() }

func (s *ttcDataStream) CreateSubView(length int64) pdfio.RandomAccessRead {
	return s.stream.CreateSubView(length)
}

// TrueTypeCollection is a TrueType Collection, now more properly known as a
// "Font Collection" as it may contain either TrueType or OpenType fonts.
//
// Port of org.apache.fontbox.ttf.TrueTypeCollection.
type TrueTypeCollection struct {
	stream      DataStream
	numFonts    int
	fontOffsets []int64
}

// NewTrueTypeCollectionFile creates a new TrueTypeCollection from a .ttc file.
func NewTrueTypeCollectionFile(path string) (*TrueTypeCollection, error) {
	file, err := pdfio.OpenBufferedFile(path)
	if err != nil {
		return nil, err
	}
	stream, err := createBufferedDataStream(file, true)
	if err != nil {
		return nil, err
	}
	return newTrueTypeCollection(stream)
}

// NewTrueTypeCollection creates a new TrueTypeCollection from a .ttc input
// stream.
func NewTrueTypeCollection(source io.Reader) (*TrueTypeCollection, error) {
	buffer, err := pdfio.NewReadBufferFromReader(source)
	if err != nil {
		return nil, err
	}
	stream, err := createBufferedDataStream(buffer, false)
	if err != nil {
		return nil, err
	}
	return newTrueTypeCollection(stream)
}

// newTrueTypeCollection creates a new TrueTypeCollection from a data stream.
func newTrueTypeCollection(stream DataStream) (*TrueTypeCollection, error) {
	c := &TrueTypeCollection{stream: stream}

	// TTC header
	r := newReader(stream)
	tag := r.str(4)
	if r.err != nil {
		return nil, r.err
	}
	if tag != "ttcf" {
		return nil, errors.New("Missing TTC header")
	}
	version := r.fixed()
	c.numFonts = int(r.unsignedInt())
	if r.err != nil {
		return nil, r.err
	}
	if c.numFonts <= 0 || c.numFonts > 1024 {
		return nil, fmt.Errorf("Invalid number of fonts %d", c.numFonts)
	}
	c.fontOffsets = make([]int64, c.numFonts)
	for i := 0; i < c.numFonts; i++ {
		c.fontOffsets[i] = r.unsignedInt()
	}
	if r.err != nil {
		return nil, r.err
	}
	if version >= 2 {
		// not used at this time
		_ = r.unsignedShort() // ulDsigTag
		_ = r.unsignedShort() // ulDsigLength
		_ = r.unsignedShort() // ulDsigOffset
		if r.err != nil {
			return nil, r.err
		}
	}
	return c, nil
}

func createBufferedDataStream(randomAccessRead pdfio.RandomAccessRead,
	closeAfterReading bool) (DataStream, error) {
	stream, err := NewRandomAccessReadDataStream(randomAccessRead)
	if closeAfterReading {
		pdfio.CloseQuietly(randomAccessRead)
	}
	if err != nil {
		return nil, err
	}
	return stream, nil
}

// ProcessAllFonts runs the callback for each TT font in the collection.
func (c *TrueTypeCollection) ProcessAllFonts(process func(ttf *TrueTypeFont) error) error {
	for i := 0; i < c.numFonts; i++ {
		font, err := c.fontAtIndex(i)
		if err != nil {
			return err
		}
		if err := process(font); err != nil {
			return err
		}
	}
	return nil
}

// ProcessAllFontHeaders runs the callback for the headers of each TT font in
// the named collection file.
func ProcessAllFontHeaders(ttcFile string, process func(fontHeaders *FontHeaders)) error {
	read, err := pdfio.OpenBufferedFile(ttcFile)
	if err != nil {
		return err
	}
	defer pdfio.CloseQuietly(read)
	stream, err := newUnbufferedDataStream(read)
	if err != nil {
		return err
	}
	defer stream.Close()
	ttc, err := newTrueTypeCollection(stream)
	if err != nil {
		return err
	}
	for i := 0; i < ttc.numFonts; i++ {
		parser, err := ttc.createFontParserAtIndexAndSeek(i)
		if err != nil {
			return err
		}
		headers, err := parser.parseTableHeaders(newTTCDataStream(ttc.stream))
		if err != nil {
			return err
		}
		process(headers)
	}
	return nil
}

func (c *TrueTypeCollection) fontAtIndex(idx int) (*TrueTypeFont, error) {
	parser, err := c.createFontParserAtIndexAndSeek(idx)
	if err != nil {
		return nil, err
	}
	return parser.ParseStream(newTTCDataStream(c.stream))
}

func (c *TrueTypeCollection) createFontParserAtIndexAndSeek(idx int) (*Parser, error) {
	if err := c.stream.SeekTo(c.fontOffsets[idx]); err != nil {
		return nil, err
	}
	r := newReader(c.stream)
	tag := r.str(4)
	if r.err != nil {
		return nil, r.err
	}
	var parser *Parser
	if tag == "OTTO" {
		parser = NewOTFParserEmbedded(false).Parser
	} else {
		parser = NewParserEmbedded(false)
	}
	if err := c.stream.SeekTo(c.fontOffsets[idx]); err != nil {
		return nil, err
	}
	return parser, nil
}

// FontByName gets a TT font from a collection by its postscript name, or nil
// where none is found.
func (c *TrueTypeCollection) FontByName(name string) (*TrueTypeFont, error) {
	for i := 0; i < c.numFonts; i++ {
		font, err := c.fontAtIndex(i)
		if err != nil {
			return nil, err
		}
		fontName, err := font.Name()
		if err != nil {
			return nil, err
		}
		if fontName == name {
			return font, nil
		}
	}
	return nil, nil
}

// Close closes the shared stream.
func (c *TrueTypeCollection) Close() error { return c.stream.Close() }
