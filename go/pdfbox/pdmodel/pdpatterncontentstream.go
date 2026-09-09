package pdmodel

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// TilingPatternLike is the tiling pattern a pattern content stream writes into.
//
// The interface is Go's, not Java's. graphics/pattern imports this package --
// a tiling pattern holds a PDResources and reads through a ResourceCache -- so
// PDTilingPattern cannot be named here. What is behind this is always one.
type TilingPatternLike interface {
	// ContentStream is the pattern's stream, which is nil for a pattern read
	// from a dictionary rather than built.
	ContentStream() *common.PDStream

	// Resources are the pattern's, which the PDF specification requires and
	// which NewPDTilingPattern therefore always sets.
	Resources() *PDResources
}

// PDPatternContentStream writes the content stream of a tiling pattern.
//
// Port of PDPatternContentStream, which Java declares final. It is the whole
// class: a constructor, and everything else inherited.
type PDPatternContentStream struct {
	pdAbstractContentStream
}

// NewPDPatternContentStream writes into the given tiling pattern.
func NewPDPatternContentStream(
	tilingPattern TilingPatternLike) (*PDPatternContentStream, error) {
	outputStream, err := tilingPattern.ContentStream().CreateOutputStream()
	if err != nil {
		return nil, err
	}
	c := &PDPatternContentStream{}
	c.initAbstractContentStream(nil, outputStream, tilingPattern.Resources())
	return c, nil
}
