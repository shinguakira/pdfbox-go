package form

import (
	"bytes"
	"io"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// PDXFAResource is the XFA form of a document, which is either one stream or an
// array of named packets.
//
// Port of PDXFAResource, which Java declares final.
//
// getDocument is not here: it parses the bytes into an org.w3c.dom.Document
// through org.apache.pdfbox.util.XMLUtil, and the port has no DOM yet. See
// migration/STATUS.md.
type PDXFAResource struct {
	xfa cos.Base
}

var _ common.COSObjectable = (*PDXFAResource)(nil)

// NewPDXFAResource wraps the given object.
func NewPDXFAResource(xfaBase cos.Base) *PDXFAResource {
	return &PDXFAResource{xfa: xfaBase}
}

// COSObject returns the object below this resource.
func (r *PDXFAResource) COSObject() cos.Base { return r.xfa }

// Bytes returns the XFA as one run of bytes, joining the packets where it is
// split into them.
func (r *PDXFAResource) Bytes() ([]byte, error) {
	// handle the case if the XFA is split into individual parts
	switch value := r.COSObject().(type) {
	case *cos.Array:
		return bytesFromPacket(value)
	case *cos.Stream:
		return bytesFromStream(value)
	}
	return []byte{}, nil
}

// bytesFromPacket joins the streams of a packet array, which holds a name
// before each of them. Java declares it private.
func bytesFromPacket(cosArray *cos.Array) ([]byte, error) {
	baos := bytes.Buffer{}
	for i := 1; i < cosArray.Size(); i += 2 {
		if stream, isStream := cosArray.GetObject(i).(*cos.Stream); isStream {
			part, err := bytesFromStream(stream)
			if err != nil {
				return nil, err
			}
			baos.Write(part)
		}
	}
	return baos.Bytes(), nil
}

// bytesFromStream reads a stream out. Java declares it private.
func bytesFromStream(stream *cos.Stream) ([]byte, error) {
	is, err := stream.CreateReader()
	if err != nil {
		return nil, err
	}
	return io.ReadAll(is)
}
