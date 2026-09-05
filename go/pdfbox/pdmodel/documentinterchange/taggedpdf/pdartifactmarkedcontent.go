package taggedpdf

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/documentinterchange/markedcontent"
)

// PDArtifactMarkedContent is a marked content sequence tagged as an artifact:
// content that is there for the page rather than for the document, such as a
// running header or a rule.
//
// Port of PDArtifactMarkedContent.
//
// markedcontent.Create still answers the base type for an artifact, not this
// one: its result is the element type of PDFMarkedContentExtractor's list, and
// Go has no downcast from the base back to here, so widening it would change
// that public list type. Nothing in the port reads the accessors below through
// the extractor; a caller that has the properties builds this itself.
type PDArtifactMarkedContent struct {
	markedcontent.PDMarkedContent
}

// NewPDArtifactMarkedContent builds one over the given properties.
func NewPDArtifactMarkedContent(properties *cos.Dictionary) *PDArtifactMarkedContent {
	return &PDArtifactMarkedContent{
		PDMarkedContent: *markedcontent.NewPDMarkedContent(cos.Artifact, properties),
	}
}

// Type returns the /Type of the artifact.
func (m *PDArtifactMarkedContent) Type() string {
	return m.Properties().GetNameAsString(cos.Type, "")
}

// BBox returns the /BBox of the artifact, or nil.
func (m *PDArtifactMarkedContent) BBox() *common.PDRectangle {
	if a := m.Properties().GetCOSArray(cos.BBox); a != nil {
		return common.NewPDRectangleOfCOSArray(a)
	}
	return nil
}

// IsTopAttached reports whether the artifact is attached to the top edge.
func (m *PDArtifactMarkedContent) IsTopAttached() bool {
	return m.isAttached("Top")
}

// IsBottomAttached reports whether the artifact is attached to the bottom edge.
func (m *PDArtifactMarkedContent) IsBottomAttached() bool {
	return m.isAttached("Bottom")
}

// IsLeftAttached reports whether the artifact is attached to the left edge.
func (m *PDArtifactMarkedContent) IsLeftAttached() bool {
	return m.isAttached("Left")
}

// IsRightAttached reports whether the artifact is attached to the right edge.
func (m *PDArtifactMarkedContent) IsRightAttached() bool {
	return m.isAttached("Right")
}

// Subtype returns the /Subtype of the artifact.
func (m *PDArtifactMarkedContent) Subtype() string {
	return m.Properties().GetNameAsString(cos.Subtype, "")
}

// isAttached reports whether /Attached names the given edge.
func (m *PDArtifactMarkedContent) isAttached(edge string) bool {
	a := m.Properties().GetCOSArray(cos.Attached)
	if a == nil {
		return false
	}
	for i := 0; i < a.Size(); i++ {
		if edge == a.GetName(i, "") {
			return true
		}
	}
	return false
}
