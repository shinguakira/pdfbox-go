package graphics

import (
	"errors"
	"fmt"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// PDXObject is an external object, the base of an image XObject, a form XObject
// and a PostScript XObject.
//
// Port of org.apache.pdfbox.pdmodel.graphics.PDXObject.
type PDXObject struct {
	stream *common.PDStream
}

var _ common.COSObjectable = (*PDXObject)(nil)

// ErrFormXObjectNotPorted is what CreateXObject reports for a form XObject.
//
// pdmodel/graphics/form is slice 9's, together with the rendering that reads
// it; slice 6 ports the image half of PDXObject and the base they share.
var ErrFormXObjectNotPorted = errors.New(
	"graphics: form XObjects are not ported yet, they are slice 9")

// NewPDXObjectOfStream builds an XObject over the given stream, stamping the
// type and subtype into it.
//
// Port of the protected PDXObject(COSStream, COSName) constructor; the port
// exports it, because Go has no protected and PDImageXObject is in another
// package.
func NewPDXObjectOfStream(stream *cos.Stream, subtype *cos.Name) PDXObject {
	return NewPDXObjectOfPDStream(common.NewPDStream(stream), subtype)
}

// NewPDXObjectOfPDStream builds an XObject over the given stream.
//
// Port of the protected PDXObject(PDStream, COSName) constructor.
func NewPDXObjectOfPDStream(stream *common.PDStream, subtype *cos.Name) PDXObject {
	// could be used for writing:
	stream.Stream().SetName(cos.Type, cos.XObject.Name())
	stream.Stream().SetName(cos.Subtype, subtype.Name())
	return PDXObject{stream: stream}
}

// COSObject returns the stream below this XObject.
func (x *PDXObject) COSObject() cos.Base { return x.Stream() }

// Stream returns the stream below this XObject.
func (x *PDXObject) Stream() *cos.Stream { return x.stream.Stream() }

// PDStream returns the stream below this XObject.
func (x *PDXObject) PDStream() *common.PDStream { return x.stream }

// SubtypeOf returns the /Subtype of an XObject stream, which is what
// CreateXObject dispatches on.
//
// Java's createXObject is not ported here: it builds a PDImageXObject, which is
// in pdmodel/graphics/image and imports this package, so the factory lives
// there instead and this is the part of it that does not.
func SubtypeOf(base cos.Base) (*cos.Stream, string, error) {
	if base == nil {
		// TODO throw an exception?
		return nil, "", nil
	}
	stream, ok := base.(*cos.Stream)
	if !ok {
		return nil, "", fmt.Errorf("Unexpected object type: %T", base)
	}
	return stream, stream.GetNameAsString(cos.Subtype, ""), nil
}

// PDPostScriptXObject is a PostScript XObject.
//
// Port of org.apache.pdfbox.pdmodel.graphics.PDPostScriptXObject. A conforming
// reader ignores it, which is why there is nothing else here.
type PDPostScriptXObject struct {
	PDXObject
}

// NewPDPostScriptXObject builds a PostScript XObject over the given stream.
func NewPDPostScriptXObject(stream *cos.Stream) *PDPostScriptXObject {
	return &PDPostScriptXObject{PDXObject: NewPDXObjectOfStream(stream, cos.PS)}
}
