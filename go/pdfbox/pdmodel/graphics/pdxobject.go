package graphics

import (
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

// Java's static createXObject is not here: it builds a PDImageXObject and a
// PDFormXObject, whose packages import this one, so the factory lives in
// pdmodel, which reaches both. See pdmodel.CreateXObject.

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
