package pdmodel

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/documentinterchange/logicalstructure"
)

// The structure tree names PDPage, PDStructureElementNameTreeNode and the
// XObject factory, which live here, and this package imports it back for the
// catalog's structure tree root. Go forbids that, so it declares what it needs
// and takes the constructors; these are them.
func init() {
	logicalstructure.NewPageFromDictionary = func(dic *cos.Dictionary) logicalstructure.PageLike {
		return NewPDPageOf(dic)
	}
	logicalstructure.NewStructureElementNameTreeNode = func(
		dic *cos.Dictionary) common.NameTreeNode[*logicalstructure.PDStructureElement] {
		return NewPDStructureElementNameTreeNodeOf(dic)
	}
	logicalstructure.CreateXObject = func(base cos.Base) (common.COSObjectable, error) {
		return CreateXObject(base, nil)
	}
}
