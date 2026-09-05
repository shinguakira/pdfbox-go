package common

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// PDStream is a stream of a PDF document.
//
// Port of org.apache.pdfbox.pdmodel.common.PDStream.
//
// getFile and setFile are not methods here: they name PDFileSpecification, and
// filespecification imports this package for PDEmbeddedFile. They are
// filespecification.FileOfStream and SetFileOfStream.
//
// createInputStream(DecodeOptions) is not ported: cos.Stream has no reader that
// takes decode options, because the Codec interface does not. See
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

// DecodeParms returns the list of decode parameters. Each entry in the list
// will refer to an entry in the filters list, and is a COSDictionaryMap of the
// basic types that entry holds. It answers nil where the stream has none, which
// is Java's null.
//
// Port of getDecodeParms.
func (s *PDStream) DecodeParms() (*COSArrayList[any], error) {
	// See PDF Ref 1.5 implementation note 7, /DP is sometimes used instead.
	return s.internalDecodeParams(cos.DecodeParms, cos.DP)
}

// FileDecodeParams returns the list of decode parameters of the external file.
//
// Port of getFileDecodeParams.
func (s *PDStream) FileDecodeParams() (*COSArrayList[any], error) {
	return s.internalDecodeParams(cos.FDecodeParms, nil)
}

// internalDecodeParams is the private internalGetDecodeParams.
func (s *PDStream) internalDecodeParams(name1, name2 *cos.Name) (*COSArrayList[any], error) {
	var dp cos.Base
	if name2 == nil {
		dp = s.stream.GetDictionaryObject(name1)
	} else {
		dp = s.stream.GetDictionaryObject2(name1, name2)
	}

	switch value := dp.(type) {
	case *cos.Dictionary:
		m, err := ConvertBasicTypesToMap(value)
		if err != nil {
			return nil, err
		}
		return NewCOSArrayListOfItem[any](m, dp, &s.stream.Dictionary, name1), nil
	case *cos.Array:
		actuals := make([]any, 0, value.Size())
		for i := 0; i < value.Size(); i++ {
			base := value.GetObject(i)
			dictionary, isDictionary := base.(*cos.Dictionary)
			if !isDictionary {
				slog.Warn("common: expected COSDictionary, ignored", "got", base)
				continue
			}
			m, err := ConvertBasicTypesToMap(dictionary)
			if err != nil {
				return nil, err
			}
			actuals = append(actuals, m)
		}
		return NewCOSArrayListOf(actuals, value), nil
	}

	return nil, nil
}

// SetDecodeParms sets the list of decode parameters.
func (s *PDStream) SetDecodeParms(decodeParams []any) {
	s.stream.SetItem(cos.DecodeParms, ConverterToCOSArray(decodeParams))
}

// SetFileDecodeParams sets the list of decode parameters of the external file.
func (s *PDStream) SetFileDecodeParams(decodeParams []any) {
	s.stream.SetItem(cos.FDecodeParms, ConverterToCOSArray(decodeParams))
}

// FileFilters returns the encoding filters of the external file, empty where
// there are none.
//
// Port of getFileFilters.
func (s *PDStream) FileFilters() []string {
	switch filters := s.stream.GetDictionaryObject(cos.FFilter).(type) {
	case *cos.Name:
		return []string{filters.Name()}
	case *cos.Array:
		return filters.ToNameStringList()
	}
	return []string{}
}

// SetFileFilters sets the encoding filters of the external file.
func (s *PDStream) SetFileFilters(filters []string) {
	s.stream.SetItem(cos.FFilter, cos.ArrayOfNames(filters))
}

// Metadata returns the metadata of this stream, or nil where it has none.
//
// Java throws IllegalStateException where the entry is neither a stream nor
// null, which is unchecked, so the port panics. A COSNull is authorized.
func (s *PDStream) Metadata() *PDMetadata {
	mdStream := s.stream.GetDictionaryObject(cos.Metadata)
	switch value := mdStream.(type) {
	case *cos.Stream:
		return NewPDMetadataOfStream(value)
	case nil, *cos.Null:
		// null is authorized
		return nil
	default:
		panic(fmt.Sprintf("Expected a COSStream but was a %T", value))
	}
}

// SetMetadata sets the metadata of this stream, which may be nil.
func (s *PDStream) SetMetadata(meta *PDMetadata) {
	s.stream.SetItem(cos.Metadata, COSObjectOrNil(meta))
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
// it with the given filter, and one that applies none for a nil filter.
//
// Port of createOutputStream(COSName). Java widens a null COSName to a null
// COSBase and COSStream.createOutputStream tests it for null; a nil *cos.Name
// widened to a cos.Base in Go is not nil, so the nil is answered here instead.
func (s *PDStream) CreateOutputStreamOfFilter(filter *cos.Name) (io.WriteCloser, error) {
	if filter == nil {
		return s.stream.CreateWriter()
	}
	return s.stream.CreateWriterWithFilters(filter)
}
