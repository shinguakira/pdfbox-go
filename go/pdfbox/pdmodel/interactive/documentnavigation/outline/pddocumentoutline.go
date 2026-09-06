package outline

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// PDDocumentOutline is the root of the outline tree.
//
// Port of PDDocumentOutline, which Java declares final.
type PDDocumentOutline struct {
	PDOutlineNode
}

var _ OutlineNode = (*PDDocumentOutline)(nil)

// NewPDDocumentOutline builds an empty document outline.
func NewPDDocumentOutline() *PDDocumentOutline {
	o := &PDDocumentOutline{}
	o.InitOutlineNode(o)
	o.Dictionary().SetName(cos.Type, cos.Outlines.Name())
	return o
}

// NewPDDocumentOutlineOf builds one over the given dictionary, stamping the
// type into it.
func NewPDDocumentOutlineOf(dic *cos.Dictionary) *PDDocumentOutline {
	o := &PDDocumentOutline{}
	o.InitOutlineNodeOf(o, dic)
	o.Dictionary().SetName(cos.Type, cos.Outlines.Name())
	return o
}

// IsNodeOpen reports true: the root of the outline hierarchy is always open.
func (o *PDDocumentOutline) IsNodeOpen() bool { return true }

// OpenNode does nothing: the root of the outline hierarchy is not an
// OutlineItem and cannot be opened or closed.
func (o *PDDocumentOutline) OpenNode() {}

// CloseNode does nothing: the root of the outline hierarchy is not an
// OutlineItem and cannot be opened or closed.
func (o *PDDocumentOutline) CloseNode() {}
