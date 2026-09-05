package pdmodel

import (
	"fmt"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/documentinterchange/logicalstructure"
)

// PDStructureElementNameTreeNode is a name tree whose values are structure
// elements, which is what the structure tree root's /IDTree is.
//
// Port of PDStructureElementNameTreeNode.
type PDStructureElementNameTreeNode struct {
	common.PDNameTreeNode[*logicalstructure.PDStructureElement]
}

var _ common.NameTreeNode[*logicalstructure.PDStructureElement] = (*PDStructureElementNameTreeNode)(nil)

// NewPDStructureElementNameTreeNode builds an empty node.
func NewPDStructureElementNameTreeNode() *PDStructureElementNameTreeNode {
	n := &PDStructureElementNameTreeNode{}
	n.InitNameTreeNode(n)
	return n
}

// NewPDStructureElementNameTreeNodeOf builds one over the given dictionary.
func NewPDStructureElementNameTreeNodeOf(dic *cos.Dictionary) *PDStructureElementNameTreeNode {
	n := &PDStructureElementNameTreeNode{}
	n.InitNameTreeNodeOf(n, dic)
	return n
}

// ConvertCOSToPD builds the structure element a value holds.
//
// Java builds one over a null dictionary where the value is null, rather than
// rejecting it, and the port keeps that.
func (n *PDStructureElementNameTreeNode) ConvertCOSToPD(
	base cos.Base) (*logicalstructure.PDStructureElement, error) {
	switch value := base.(type) {
	case nil:
		return logicalstructure.NewPDStructureElementOf(nil), nil
	case *cos.Stream:
		// COSStream is a COSDictionary in Java, so its instanceof lets one
		// through here.
		return logicalstructure.NewPDStructureElementOf(&value.Dictionary), nil
	case *cos.Dictionary:
		return logicalstructure.NewPDStructureElementOf(value), nil
	default:
		return nil, fmt.Errorf("dictionary expected here, but got %v", base)
	}
}

// CreateChildNode returns a node over the given dictionary.
func (n *PDStructureElementNameTreeNode) CreateChildNode(
	dic *cos.Dictionary) common.NameTreeNode[*logicalstructure.PDStructureElement] {
	return NewPDStructureElementNameTreeNodeOf(dic)
}
