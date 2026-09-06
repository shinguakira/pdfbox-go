package form

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// PDTransparencyGroup is a transparency group XObject: a form XObject whose
// /Group says its contents are composited together before being painted.
//
// Port of PDTransparencyGroup, which adds no behaviour of its own; the renderer
// tells it apart by its type.
type PDTransparencyGroup struct {
	PDFormXObject
}

var _ common.COSObjectable = (*PDTransparencyGroup)(nil)

// NewPDTransparencyGroupOfPDStream creates a transparency group over the given
// stream.
func NewPDTransparencyGroupOfPDStream(stream *common.PDStream) *PDTransparencyGroup {
	return &PDTransparencyGroup{PDFormXObject: *NewPDFormXObjectOfPDStream(stream)}
}

// NewPDTransparencyGroupOfStreamCached creates one that reads its resources
// through the given cache.
func NewPDTransparencyGroupOfStreamCached(stream *cos.Stream, cache CacheLike) *PDTransparencyGroup {
	return &PDTransparencyGroup{PDFormXObject: *NewPDFormXObjectOfStreamCached(stream, cache)}
}

// NewPDTransparencyGroup creates a new empty transparency group in the given
// document.
func NewPDTransparencyGroup(document common.COSDocumentLike) *PDTransparencyGroup {
	// todo: set mandatory fields
	return &PDTransparencyGroup{PDFormXObject: *NewPDFormXObject(document)}
}
