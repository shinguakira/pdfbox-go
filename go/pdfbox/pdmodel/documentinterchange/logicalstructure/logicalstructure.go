// Package logicalstructure holds the structure tree of a tagged PDF: the
// logical order and the meaning of the content, apart from the order it is
// painted in.
//
// Port of org.apache.pdfbox.pdmodel.documentinterchange.logicalstructure.
package logicalstructure

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// PageLike is the page a structure node points at with /Pg.
//
// Java names PDPage, which lives in pdmodel; pdmodel imports this package for
// PDDocumentCatalog.getStructureTreeRoot, so the dependency cannot run both
// ways. The port names what is used and takes the constructors below, which
// pdmodel sets from its init.
type PageLike interface {
	common.COSObjectable
}

// NewPageFromDictionary builds a page from its dictionary. pdmodel sets it.
var NewPageFromDictionary func(dic *cos.Dictionary) PageLike

// NewStructureElementNameTreeNode builds an /IDTree node. pdmodel sets it,
// because Java's PDStructureElementNameTreeNode lives there, even though it
// only needs this package and common.
var NewStructureElementNameTreeNode func(dic *cos.Dictionary) common.NameTreeNode[*PDStructureElement]

// CreateXObject builds an XObject from its stream with no resources, which is
// the PDXObject.createXObject(base, null) that PDObjectReference calls.
// pdmodel sets it: the factory needs both graphics/image and graphics/form.
var CreateXObject func(base cos.Base) (common.COSObjectable, error)
