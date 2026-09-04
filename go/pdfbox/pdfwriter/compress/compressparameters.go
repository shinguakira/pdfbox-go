// Package compress holds the object stream compression a PDF save can apply.
//
// Port of org.apache.pdfbox.pdfwriter.compress.
package compress

// DefaultObjectStreamSize is how many objects one object stream holds by
// default.
//
// Port of CompressParameters.DEFAULT_OBJECT_STREAM_SIZE.
const DefaultObjectStreamSize = 200

// Parameters says whether a save compresses its objects into object streams,
// and how many go in each.
//
// Port of org.apache.pdfbox.pdfwriter.compress.CompressParameters.
type Parameters struct {
	objectStreamSize int
}

// DefaultCompression compresses with the default object stream size.
//
// Port of CompressParameters.DEFAULT_COMPRESSION.
var DefaultCompression = NewParameters()

// NoCompression writes every object on its own.
//
// Port of CompressParameters.NO_COMPRESSION.
var NoCompression = NewParametersOfSize(0)

// NewParameters returns the default parameters.
func NewParameters() *Parameters {
	return NewParametersOfSize(DefaultObjectStreamSize)
}

// NewParametersOfSize returns parameters with the given object stream size.
//
// Java throws IllegalArgumentException for a negative size, which is unchecked,
// so the port panics.
func NewParametersOfSize(objectStreamSize int) *Parameters {
	if objectStreamSize < 0 {
		panic("Object stream size can't be a negative value")
	}
	return &Parameters{objectStreamSize: objectStreamSize}
}

// ObjectStreamSize returns how many objects one object stream holds.
func (p *Parameters) ObjectStreamSize() int { return p.objectStreamSize }

// IsCompress reports whether objects are to be compressed at all.
func (p *Parameters) IsCompress() bool { return p.objectStreamSize > 0 }
