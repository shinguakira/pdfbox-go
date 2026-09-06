package common

import (
	"io"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// PDDestinationOrAction is implemented by anything that can be the target of a
// link: a destination or an action.
//
// Port of the marker interface
// org.apache.pdfbox.pdmodel.common.PDDestinationOrAction.
type PDDestinationOrAction interface {
	COSObjectable
}

// PDMetadata is the XMP metadata stream of a document or of one of its parts.
//
// Port of org.apache.pdfbox.pdmodel.common.PDMetadata, which extends PDStream.
type PDMetadata struct {
	PDStream
}

var _ COSObjectable = (*PDMetadata)(nil)

// NewPDMetadata creates a new empty metadata stream in the given document.
func NewPDMetadata(document COSDocumentLike) *PDMetadata {
	m := &PDMetadata{PDStream: *NewPDStreamOfDocument(document)}
	m.Stream().SetName(cos.Type, "Metadata")
	m.Stream().SetName(cos.Subtype, "XML")
	return m
}

// NewPDMetadataOfInput creates a metadata stream holding what the reader gives.
func NewPDMetadataOfInput(doc COSDocumentLike, str io.Reader) (*PDMetadata, error) {
	stream, err := NewPDStreamOfInput(doc, str, nil)
	if err != nil {
		return nil, err
	}
	m := &PDMetadata{PDStream: *stream}
	m.Stream().SetName(cos.Type, "Metadata")
	m.Stream().SetName(cos.Subtype, "XML")
	return m, nil
}

// NewPDMetadataOfStream wraps an existing stream.
func NewPDMetadataOfStream(str *cos.Stream) *PDMetadata {
	return &PDMetadata{PDStream: *NewPDStream(str)}
}

// ExportXMPMetadata returns a reader over the XMP.
func (m *PDMetadata) ExportXMPMetadata() (io.Reader, error) {
	return m.CreateInputStream()
}

// ImportXMPMetadata replaces the XMP with the given bytes.
func (m *PDMetadata) ImportXMPMetadata(xmp []byte) error {
	os, err := m.CreateOutputStream()
	if err != nil {
		return err
	}
	if _, err := os.Write(xmp); err != nil {
		os.Close()
		return err
	}
	return os.Close()
}

// PDObjectStream is an object stream of a document.
//
// Port of org.apache.pdfbox.pdmodel.common.PDObjectStream.
type PDObjectStream struct {
	PDStream
}

var _ COSObjectable = (*PDObjectStream)(nil)

// NewPDObjectStream wraps an existing stream.
func NewPDObjectStream(str *cos.Stream) *PDObjectStream {
	return &PDObjectStream{PDStream: *NewPDStream(str)}
}

// CreateObjectStream creates a new object stream in the given document.
func CreateObjectStream(document COSDocumentLike) *PDObjectStream {
	cosStream := document.CreateStream()
	strm := NewPDObjectStream(cosStream)
	strm.Stream().SetItem(cos.Type, cos.ObjStm)
	return strm
}

// Type returns the /Type of the stream.
func (s *PDObjectStream) Type() string {
	return s.Stream().GetNameAsString(cos.Type, "")
}

// NumberOfObjects returns the /N entry.
func (s *PDObjectStream) NumberOfObjects() int {
	return s.Stream().GetIntDefault(cos.N, 0)
}

// SetNumberOfObjects sets the /N entry.
func (s *PDObjectStream) SetNumberOfObjects(n int) {
	s.Stream().SetInt(cos.N, n)
}

// FirstByteOffset returns the /First entry.
func (s *PDObjectStream) FirstByteOffset() int {
	return s.Stream().GetIntDefault(cos.First, 0)
}

// SetFirstByteOffset sets the /First entry.
func (s *PDObjectStream) SetFirstByteOffset(n int) {
	s.Stream().SetInt(cos.First, n)
}

// Extends returns the stream this one extends, or nil.
func (s *PDObjectStream) Extends() *PDObjectStream {
	stream, _ := s.Stream().GetDictionaryObject(cos.Extends).(*cos.Stream)
	if stream != nil {
		return NewPDObjectStream(stream)
	}
	return nil
}

// SetExtends sets the stream this one extends.
func (s *PDObjectStream) SetExtends(stream *PDObjectStream) {
	if stream == nil {
		s.Stream().SetItem(cos.Extends, nil)
		return
	}
	s.Stream().SetItem(cos.Extends, stream.COSObject())
}
