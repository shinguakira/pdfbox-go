package common

import (
	"io"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// PDStream is a stream of a PDF document.
//
// Port of org.apache.pdfbox.pdmodel.common.PDStream. The reading path is
// ported; the parts that write a stream into a document, and the decode
// parameters, the file specification and the metadata, need pieces a later
// slice brings. See migration/STATUS.md.
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
