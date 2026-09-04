package common

import (
	"io"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// PDStream is a stream of a PDF document.
//
// Port of org.apache.pdfbox.pdmodel.common.PDStream. The decode parameters, the
// file specification and the metadata need pieces a later slice brings. See
// migration/STATUS.md.
type PDStream struct {
	stream *cos.Stream
}

var _ COSObjectable = (*PDStream)(nil)

// NewPDStream returns a wrapper round an existing stream.
func NewPDStream(str *cos.Stream) *PDStream {
	return &PDStream{stream: str}
}

// COSObject returns the stream itself.
func (s *PDStream) COSObject() cos.Base { return s.stream }

// Stream returns the stream itself, typed.
func (s *PDStream) Stream() *cos.Stream { return s.stream }

// CreateInputStream returns a reader over the decoded contents of the stream.
func (s *PDStream) CreateInputStream() (io.Reader, error) {
	return s.stream.CreateReader()
}

// ToByteArray returns the decoded contents of the stream.
func (s *PDStream) ToByteArray() ([]byte, error) {
	is, err := s.CreateInputStream()
	if err != nil {
		return nil, err
	}
	return io.ReadAll(is)
}

// Length returns the /Length entry of the stream, which is how many bytes it
// takes up as written.
func (s *PDStream) Length() int {
	return s.stream.GetIntDefault(cos.Length, 0)
}

// Filters returns the filters the stream is encoded with.
func (s *PDStream) Filters() []*cos.Name {
	filters := s.stream.Filters()
	switch value := filters.(type) {
	case *cos.Name:
		return []*cos.Name{value}
	case *cos.Array:
		names := make([]*cos.Name, 0, value.Size())
		for i := 0; i < value.Size(); i++ {
			// Java casts the whole list to List<COSName>, so a non-name entry
			// only fails when it is read.
			name, _ := value.GetObject(i).(*cos.Name)
			names = append(names, name)
		}
		return names
	}
	return nil
}

// SetFilters sets the filters the stream is encoded with.
func (s *PDStream) SetFilters(filters []*cos.Name) {
	array := cos.NewArray()
	for _, filter := range filters {
		array.Add(filter)
	}
	s.stream.SetItem(cos.Filter, array)
}

// DecodedStreamLength returns the /DL entry, which is how many bytes the
// stream takes up decoded.
func (s *PDStream) DecodedStreamLength() int {
	return s.stream.GetInt(cos.DL)
}

// SetDecodedStreamLength sets the /DL entry.
func (s *PDStream) SetDecodedStreamLength(decodedStreamLength int) {
	s.stream.SetInt(cos.DL, decodedStreamLength)
}

// CreateInputStreamStopping returns the content of this stream, decoded
// through every filter up to but not including the first one named in
// stopFilters.
//
// Port of createInputStream(List<String>). PDImageXObject uses it to hand the
// still-encoded image data to a caller that wants the JPEG or the fax data
// rather than the samples.
func (s *PDStream) CreateInputStreamStopping(stopFilters []string) (io.Reader, error) {
	raw, err := s.stream.CreateRawReader()
	if err != nil {
		return nil, err
	}
	var someFilters []*cos.Name
	for _, nextFilter := range s.Filters() {
		if containsString(stopFilters, nextFilter.Name()) {
			break
		}
		someFilters = append(someFilters, nextFilter)
	}
	if len(someFilters) == 0 {
		return raw, nil
	}
	// The port decodes through the stream's own codec provider rather than
	// through filter.Decode: pdmodel/common must not import pdfbox/filter,
	// which imports this package's sibling cos, and the provider is what the
	// stream was built with.
	return s.stream.CreateReaderStopping(len(someFilters))
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// COSDocumentLike is the half of a COS document PDStream creates streams
// through.
//
// Java's constructors take a PDDocument or a COSDocument; the port names what
// is used, so that this package does not import pdmodel back.
type COSDocumentLike interface {
	// CreateStream returns a new empty stream belonging to the document.
	CreateStream() *cos.Stream
}

// NewPDStreamOfDocument creates a new empty PDStream object.
//
// Port of PDStream(PDDocument) and PDStream(COSDocument), which are the same
// method once the document is taken through what it is used for.
func NewPDStreamOfDocument(document COSDocumentLike) *PDStream {
	return &PDStream{stream: document.CreateStream()}
}

// NewPDStreamOfInput reads all data from the input stream and embeds it into
// the document with the given filters applied, if any.
//
// Port of the private PDStream(PDDocument, InputStream, COSBase), which the
// three public overloads delegate to. Java closes the InputStream; a Go
// io.Reader has nothing to close, so the caller keeps that duty.
func NewPDStreamOfInput(doc COSDocumentLike, input io.Reader, filters cos.Base) (*PDStream, error) {
	stream := doc.CreateStream()
	output, err := stream.CreateWriterWithFilters(filters)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(output, input); err != nil {
		output.Close()
		return nil, err
	}
	if err := output.Close(); err != nil {
		return nil, err
	}
	return &PDStream{stream: stream}, nil
}

// CreateOutputStream returns a writer that stores what is written to it in the
// stream, without a filter.
//
// Port of createOutputStream().
func (s *PDStream) CreateOutputStream() (io.WriteCloser, error) {
	return s.stream.CreateWriter()
}

// CreateOutputStreamOfFilter returns a writer that encodes what is written to
// it with the given filter.
//
// Port of createOutputStream(COSName).
func (s *PDStream) CreateOutputStreamOfFilter(filter *cos.Name) (io.WriteCloser, error) {
	return s.stream.CreateWriterWithFilters(filter)
}
