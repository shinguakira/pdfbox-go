package cos

import (
	"errors"
	"fmt"
	"io"

	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// Errors returned by Stream.
var (
	// ErrStreamClosed is returned by a read on a stream whose backing storage
	// has been closed, usually because the document was closed.
	ErrStreamClosed = errors.New("cos: stream has been closed and cannot be read")

	// ErrStreamWriting is returned when a read or a second write is attempted
	// while a writer is open. Java throws IllegalStateException.
	ErrStreamWriting = errors.New("cos: cannot read or open a second writer while a stream writer is open")

	// ErrStreamNoData is returned when a stream is read before anything has
	// been written to it.
	ErrStreamNoData = errors.New("cos: stream read before any data was written")

	// ErrNoCodecProvider is returned when a filtered stream is decoded but the
	// stream was created without a CodecProvider.
	ErrNoCodecProvider = errors.New("cos: stream has filters but no codec provider")
)

// StreamCodec encodes and decodes stream data for one PDF filter.
//
// This interface exists to break a package cycle. In Java,
// org.apache.pdfbox.cos.COSStream imports org.apache.pdfbox.filter.Filter while
// Filter imports COSDictionary — a package cycle, which Java permits and Go
// does not. So cos declares the shape it needs and pdfbox/filter satisfies it.
type StreamCodec interface {
	// Decode reads encoded data from r and writes the decoded data to w.
	// parameters is the stream dictionary and index says which entry of its
	// filter array this call is for.
	Decode(w io.Writer, r io.Reader, parameters *Dictionary, index int) error

	// Encode reads raw data from r and writes the encoded data to w.
	Encode(w io.Writer, r io.Reader, parameters *Dictionary) error
}

// CodecProvider resolves a PDF filter name to its codec.
//
// Port of the role FilterFactory plays in Java. It is passed to a Stream rather
// than reached through a package-level singleton, so the dependency is explicit
// and a test can supply its own.
type CodecProvider interface {
	CodecForName(name *Name) (StreamCodec, error)
}

// Stream is a PDF stream: a dictionary followed by a run of bytes.
//
// Port of org.apache.pdfbox.cos.COSStream, which extends COSDictionary; the
// port embeds Dictionary for the same reason.
//
// A stream holds its data in a buffer from a StreamCache, or reads it through a
// view onto the file it was parsed from. Closing it releases that storage,
// after which reads fail with ErrStreamClosed.
type Stream struct {
	Dictionary

	// randomAccess holds data written to this stream.
	randomAccess pdfio.RandomAccess
	// readView reads data belonging to a stream parsed from a file.
	readView pdfio.RandomAccessRead

	streamCache      pdfio.StreamCache
	closeStreamCache bool
	codecs           CodecProvider
	isWriting        bool
}

var _ Base = (*Stream)(nil)

// NewStream returns an empty stream.
//
// Port of the COSStream() constructor. codecs may be nil for a stream that will
// never carry filters; decoding one that does then fails with
// ErrNoCodecProvider rather than silently returning encoded bytes.
func NewStream(codecs CodecProvider) *Stream {
	return NewStreamWithCache(nil, codecs)
}

// NewStreamWithCache returns an empty stream that allocates its buffers from
// the given cache.
//
// Port of COSStream(RandomAccessStreamCache).
func NewStreamWithCache(cache pdfio.StreamCache, codecs CodecProvider) *Stream {
	s := &Stream{
		Dictionary:  *NewDictionary(),
		streamCache: cache,
		codecs:      codecs,
	}
	s.SetInt(Length, 0)
	return s
}

// NewStreamFromView returns a stream whose data is read through a view onto the
// file it was parsed from.
//
// Port of COSStream(RandomAccessStreamCache, RandomAccessReadView).
func NewStreamFromView(cache pdfio.StreamCache, view pdfio.RandomAccessRead, codecs CodecProvider) (*Stream, error) {
	s := NewStreamWithCache(cache, codecs)
	s.readView = view
	length, err := view.Length()
	if err != nil {
		return nil, err
	}
	s.SetLong(Length, length)
	return s, nil
}

func (s *Stream) checkClosed() error {
	if s.randomAccess != nil && s.randomAccess.IsClosed() {
		return ErrStreamClosed
	}
	return nil
}

// cache returns the stream cache, creating a memory-backed one on first use.
func (s *Stream) cache() (pdfio.StreamCache, error) {
	if s.streamCache == nil {
		created, err := pdfio.MemoryOnlyStreamCache()()
		if err != nil {
			return nil, err
		}
		s.streamCache = created
		s.closeStreamCache = true
	}
	return s.streamCache, nil
}

// CreateRawReader returns a reader over the stream data as stored, without
// decoding it.
//
// Port of createRawInputStream.
func (s *Stream) CreateRawReader() (io.Reader, error) {
	if err := s.checkClosed(); err != nil {
		return nil, err
	}
	if s.isWriting {
		return nil, ErrStreamWriting
	}
	if s.randomAccess != nil {
		return pdfio.NewReader(s.randomAccess), nil
	}
	if s.readView != nil {
		if err := pdfio.SeekTo(s.readView, 0); err != nil {
			return nil, err
		}
		return pdfio.NewReader(s.readView), nil
	}
	return nil, ErrStreamNoData
}

// CreateReader returns a reader over the decoded stream data, applying every
// filter in the stream's filter array in order.
//
// Port of createInputStream. Java returns a COSInputStream, which is an
// InputStream carrying the DecodeResult; the port returns a plain io.Reader,
// because the only thing on DecodeResult that callers use is the JPX colour
// space, which arrives with that filter.
func (s *Stream) CreateReader() (io.Reader, error) {
	raw, err := s.CreateRawReader()
	if err != nil {
		return nil, err
	}

	codecs, err := s.codecList()
	if err != nil {
		return nil, err
	}
	if len(codecs) == 0 {
		return raw, nil
	}

	// Java chains decoder streams; the port decodes into a buffer per filter,
	// which is simpler and matches how the filters are written.
	current := raw
	for i, codec := range codecs {
		decoded := pdfio.NewReadWriteBuffer()
		if err := codec.Decode(decoded, current, &s.Dictionary, i); err != nil {
			return nil, fmt.Errorf("cos: decoding filter %d: %w", i, err)
		}
		if err := pdfio.SeekTo(decoded, 0); err != nil {
			return nil, err
		}
		current = pdfio.NewReader(decoded)
	}
	return current, nil
}

// CreateRawWriter returns a writer that stores bytes as given, without encoding
// them. The returned writer must be closed, which is what records the length.
//
// Port of createRawOutputStream.
func (s *Stream) CreateRawWriter() (io.WriteCloser, error) {
	if err := s.checkClosed(); err != nil {
		return nil, err
	}
	if s.isWriting {
		return nil, ErrStreamWriting
	}
	if err := s.prepareBuffer(); err != nil {
		return nil, err
	}
	s.isWriting = true
	return &streamWriter{stream: s, dst: pdfio.NewWriter(s.randomAccess)}, nil
}

// CreateWriter returns a writer that applies the stream's filters as it stores
// bytes. The returned writer must be closed.
//
// Port of createOutputStream.
func (s *Stream) CreateWriter() (io.WriteCloser, error) {
	if err := s.checkClosed(); err != nil {
		return nil, err
	}
	if s.isWriting {
		return nil, ErrStreamWriting
	}
	codecs, err := s.codecList()
	if err != nil {
		return nil, err
	}
	if err := s.prepareBuffer(); err != nil {
		return nil, err
	}
	s.isWriting = true
	return &streamWriter{
		stream: s,
		dst:    pdfio.NewWriter(s.randomAccess),
		codecs: codecs,
		buffer: pdfio.NewReadWriteBuffer(),
	}, nil
}

// SetFilters records the filter or filter array to apply. A nil argument leaves
// any existing /Filter entry alone, as Java's createOutputStream(null) does.
func (s *Stream) SetFilters(filters Base) {
	if filters != nil {
		s.SetItem(Filter, filters)
	}
}

// CreateWriterWithFilters records the given filters and returns a writer that
// applies them.
//
// Port of createOutputStream(COSBase filters), which does both in one call.
func (s *Stream) CreateWriterWithFilters(filters Base) (io.WriteCloser, error) {
	s.SetFilters(filters)
	return s.CreateWriter()
}

func (s *Stream) prepareBuffer() error {
	if s.randomAccess != nil {
		return s.randomAccess.Clear()
	}
	cache, err := s.cache()
	if err != nil {
		return err
	}
	buffer, err := cache.CreateBuffer()
	if err != nil {
		return err
	}
	s.randomAccess = buffer
	return nil
}

// Filters returns the /Filter entry, a name or an array of names.
func (s *Stream) Filters() Base {
	return s.GetDictionaryObject(Filter)
}

// codecList resolves the stream's filter array to codecs, in order.
//
// Port of the private getFilterList.
func (s *Stream) codecList() ([]StreamCodec, error) {
	switch filters := s.Filters().(type) {
	case nil:
		return nil, nil

	case *Name:
		codec, err := s.codecFor(filters)
		if err != nil {
			return nil, err
		}
		return []StreamCodec{codec}, nil

	case *Array:
		out := make([]StreamCodec, 0, filters.Size())
		for i := 0; i < filters.Size(); i++ {
			name, ok := filters.Get(i).(*Name)
			if !ok {
				return nil, fmt.Errorf("cos: forbidden type in filter array at %d: %T",
					i, filters.Get(i))
			}
			codec, err := s.codecFor(name)
			if err != nil {
				return nil, err
			}
			out = append(out, codec)
		}
		return out, nil

	default:
		return nil, nil
	}
}

func (s *Stream) codecFor(name *Name) (StreamCodec, error) {
	if s.codecs == nil {
		return nil, fmt.Errorf("%w: %s", ErrNoCodecProvider, name.Name())
	}
	return s.codecs.CodecForName(name)
}

// Length returns the /Length entry, the number of stored bytes.
func (s *Stream) Length() int64 {
	return s.GetLongDefault(Length, 0)
}

// HasData reports whether the stream holds any data.
func (s *Stream) HasData() bool {
	return s.randomAccess != nil || s.readView != nil
}

// TextString returns the decoded stream data as a PDF text string.
//
// Port of toTextString. Java logs and returns "" when decoding fails; the port
// returns the error, since a caller here can act on it.
func (s *Stream) TextString() (string, error) {
	r, err := s.CreateReader()
	if err != nil {
		return "", err
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	return NewStringObjBytes(data).Value(), nil
}

// COSObject returns the receiver.
func (s *Stream) COSObject() Base { return s }

// Accept dispatches to the visitor.
func (s *Stream) Accept(v Visitor) error { return v.VisitStream(s) }

// Close releases the stream's storage. Reads afterwards fail.
func (s *Stream) Close() error {
	var firstErr error
	if s.randomAccess != nil {
		if err := s.randomAccess.Close(); err != nil {
			firstErr = err
		}
	}
	if s.readView != nil {
		if err := s.readView.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if s.closeStreamCache && s.streamCache != nil {
		if err := s.streamCache.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// streamWriter records the stream length when it is closed, and applies the
// stream's filters if it has any.
//
// Port of the anonymous FilterOutputStream subclasses in createOutputStream and
// createRawOutputStream, both of which exist only to set /Length on close and
// clear the writing flag.
type streamWriter struct {
	stream *Stream
	dst    io.Writer
	codecs []StreamCodec
	// buffer collects raw bytes when filters have to be applied on close.
	buffer *pdfio.ReadWriteBuffer
	closed bool
}

func (w *streamWriter) Write(p []byte) (int, error) {
	if w.closed {
		return 0, ErrStreamClosed
	}
	if w.buffer != nil {
		return w.buffer.Write(p)
	}
	return w.dst.Write(p)
}

func (w *streamWriter) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true
	defer func() { w.stream.isWriting = false }()

	if w.buffer != nil {
		if err := w.encodeBuffered(); err != nil {
			return err
		}
	}

	length, err := w.stream.randomAccess.Length()
	if err != nil {
		return err
	}
	w.stream.SetLong(Length, length)
	return nil
}

// encodeBuffered runs the collected bytes through the stream's filters, in
// reverse order, and writes the result to the backing buffer. Filters are
// listed in the order a reader applies them, so a writer applies them backwards.
func (w *streamWriter) encodeBuffered() error {
	if err := pdfio.SeekTo(w.buffer, 0); err != nil {
		return err
	}
	var current io.Reader = pdfio.NewReader(w.buffer)

	for i := len(w.codecs) - 1; i >= 0; i-- {
		encoded := pdfio.NewReadWriteBuffer()
		if err := w.codecs[i].Encode(encoded, current, &w.stream.Dictionary); err != nil {
			return fmt.Errorf("cos: encoding filter %d: %w", i, err)
		}
		if err := pdfio.SeekTo(encoded, 0); err != nil {
			return err
		}
		current = pdfio.NewReader(encoded)
	}

	_, err := io.Copy(w.dst, current)
	return err
}
