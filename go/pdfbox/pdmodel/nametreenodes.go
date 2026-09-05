package pdmodel

// The three name tree nodes the document level offers, and the two dictionaries
// that hold them.
//
// Port of PDDestinationNameTreeNode, PDEmbeddedFilesNameTreeNode,
// PDJavascriptNameTreeNode, PDDocumentNameDictionary and
// PDDocumentNameDestinationDictionary. Java gives them a file each; each is a
// pair of overrides or a handful of accessors, and all five exist for the
// catalogue's /Names and /Dests, so the port keeps them together.

import (
	"fmt"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common/filespecification"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/action"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/documentnavigation/destination"
)

// PDDestinationNameTreeNode is a name tree whose values are page destinations.
//
// Port of PDDestinationNameTreeNode. Java's type argument is the abstract class
// PDPageDestination; the port's is destination.PageDestination, the interface
// that stands for it.
type PDDestinationNameTreeNode struct {
	common.PDNameTreeNode[destination.PageDestination]
}

var _ common.NameTreeNode[destination.PageDestination] = (*PDDestinationNameTreeNode)(nil)

// NewPDDestinationNameTreeNode builds an empty node.
func NewPDDestinationNameTreeNode() *PDDestinationNameTreeNode {
	n := &PDDestinationNameTreeNode{}
	n.InitNameTreeNode(n)
	return n
}

// NewPDDestinationNameTreeNodeOf builds one over the given dictionary.
func NewPDDestinationNameTreeNodeOf(dic *cos.Dictionary) *PDDestinationNameTreeNode {
	n := &PDDestinationNameTreeNode{}
	n.InitNameTreeNodeOf(n, dic)
	return n
}

// ConvertCOSToPD builds the page destination a value holds, and answers nil for
// an entry that is a destination of another kind.
func (n *PDDestinationNameTreeNode) ConvertCOSToPD(
	base cos.Base) (destination.PageDestination, error) {
	dest := base
	if dict, isDictionary := base.(*cos.Dictionary); isDictionary {
		// the destination is sometimes stored in the D dictionary
		// entry instead of being directly an array, so just dereference
		// it for now
		dest = dict.GetDictionaryObject(cos.D)
	}
	created, err := destination.Create(dest)
	if err != nil {
		return nil, err
	}
	if pageDestination, isPageDestination := created.(destination.PageDestination); isPageDestination {
		return pageDestination, nil
	}
	// PDFBOX-5975: invalid tree entry
	return nil, nil
}

// CreateChildNode returns a node over the given dictionary.
func (n *PDDestinationNameTreeNode) CreateChildNode(
	dic *cos.Dictionary) common.NameTreeNode[destination.PageDestination] {
	return NewPDDestinationNameTreeNodeOf(dic)
}

// PDEmbeddedFilesNameTreeNode is a name tree whose values are file
// specifications.
//
// Port of PDEmbeddedFilesNameTreeNode.
type PDEmbeddedFilesNameTreeNode struct {
	common.PDNameTreeNode[*filespecification.PDComplexFileSpecification]
}

var _ common.NameTreeNode[*filespecification.PDComplexFileSpecification] = (*PDEmbeddedFilesNameTreeNode)(nil)

// NewPDEmbeddedFilesNameTreeNode builds an empty node.
func NewPDEmbeddedFilesNameTreeNode() *PDEmbeddedFilesNameTreeNode {
	n := &PDEmbeddedFilesNameTreeNode{}
	n.InitNameTreeNode(n)
	return n
}

// NewPDEmbeddedFilesNameTreeNodeOf builds one over the given dictionary.
func NewPDEmbeddedFilesNameTreeNodeOf(dic *cos.Dictionary) *PDEmbeddedFilesNameTreeNode {
	n := &PDEmbeddedFilesNameTreeNode{}
	n.InitNameTreeNodeOf(n, dic)
	return n
}

// ConvertCOSToPD builds the file specification a value holds.
//
// Java lets a null through to the constructor, which keeps it, and rejects
// anything else that is not a dictionary.
func (n *PDEmbeddedFilesNameTreeNode) ConvertCOSToPD(
	base cos.Base) (*filespecification.PDComplexFileSpecification, error) {
	switch value := base.(type) {
	case nil:
		return filespecification.NewPDComplexFileSpecification(nil), nil
	case *cos.Stream:
		// COSStream is a COSDictionary in Java, so its instanceof lets one
		// through here.
		return filespecification.NewPDComplexFileSpecification(&value.Dictionary), nil
	case *cos.Dictionary:
		return filespecification.NewPDComplexFileSpecification(value), nil
	default:
		return nil, fmt.Errorf("dictionary expected here, but got %v", base)
	}
}

// CreateChildNode returns a node over the given dictionary.
func (n *PDEmbeddedFilesNameTreeNode) CreateChildNode(
	dic *cos.Dictionary) common.NameTreeNode[*filespecification.PDComplexFileSpecification] {
	return NewPDEmbeddedFilesNameTreeNodeOf(dic)
}

// PDJavascriptNameTreeNode is a name tree whose values are JavaScript actions.
//
// Port of PDJavascriptNameTreeNode.
type PDJavascriptNameTreeNode struct {
	common.PDNameTreeNode[*action.PDActionJavaScript]
}

var _ common.NameTreeNode[*action.PDActionJavaScript] = (*PDJavascriptNameTreeNode)(nil)

// NewPDJavascriptNameTreeNode builds an empty node.
func NewPDJavascriptNameTreeNode() *PDJavascriptNameTreeNode {
	n := &PDJavascriptNameTreeNode{}
	n.InitNameTreeNode(n)
	return n
}

// NewPDJavascriptNameTreeNodeOf builds one over the given dictionary.
func NewPDJavascriptNameTreeNodeOf(dic *cos.Dictionary) *PDJavascriptNameTreeNode {
	n := &PDJavascriptNameTreeNode{}
	n.InitNameTreeNodeOf(n, dic)
	return n
}

// ConvertCOSToPD builds the JavaScript action a value holds.
//
// Java casts what PDActionFactory answers to PDActionJavaScript without
// checking, so a /JavaScript name tree holding an action of another /S throws
// ClassCastException; the port's type assertion panics, which is that cast.
func (n *PDJavascriptNameTreeNode) ConvertCOSToPD(
	base cos.Base) (*action.PDActionJavaScript, error) {
	dict, isDictionary := base.(*cos.Dictionary)
	if !isDictionary {
		return nil, fmt.Errorf(
			"Error creating Javascript object, expected a COSDictionary and not %v", base)
	}
	return action.CreateAction(dict).(*action.PDActionJavaScript), nil
}

// CreateChildNode returns a node over the given dictionary.
func (n *PDJavascriptNameTreeNode) CreateChildNode(
	dic *cos.Dictionary) common.NameTreeNode[*action.PDActionJavaScript] {
	return NewPDJavascriptNameTreeNodeOf(dic)
}

// PDDocumentNameDictionary holds all of the name trees that are available at
// the document level.
//
// Port of PDDocumentNameDictionary.
type PDDocumentNameDictionary struct {
	nameDictionary *cos.Dictionary
	catalog        *PDDocumentCatalog
}

var _ common.COSObjectable = (*PDDocumentNameDictionary)(nil)

// NewPDDocumentNameDictionary returns the name dictionary of the given
// catalogue, adding an empty one where the catalogue has none.
func NewPDDocumentNameDictionary(cat *PDDocumentCatalog) *PDDocumentNameDictionary {
	names := cat.Dictionary().GetCOSDictionary(cos.Names)
	if names == nil {
		names = cos.NewDictionary()
		cat.Dictionary().SetItem(cos.Names, names)
	}
	return &PDDocumentNameDictionary{nameDictionary: names, catalog: cat}
}

// NewPDDocumentNameDictionaryOf returns a name dictionary over the given
// dictionary, belonging to the given catalogue.
func NewPDDocumentNameDictionaryOf(cat *PDDocumentCatalog,
	names *cos.Dictionary) *PDDocumentNameDictionary {
	return &PDDocumentNameDictionary{nameDictionary: names, catalog: cat}
}

// COSObject returns the dictionary.
func (d *PDDocumentNameDictionary) COSObject() cos.Base { return d.nameDictionary }

// Dictionary returns the dictionary, which is getCOSObject narrowed the way
// Java declares it.
func (d *PDDocumentNameDictionary) Dictionary() *cos.Dictionary { return d.nameDictionary }

// Dests returns the destination name tree node, whose values are page
// destinations, or nil where there is none.
func (d *PDDocumentNameDictionary) Dests() *PDDestinationNameTreeNode {
	dic := d.nameDictionary.GetCOSDictionary(cos.Dests)
	// The document catalog also contains the Dests entry sometimes
	// so check there as well.
	if dic == nil {
		dic = d.catalog.Dictionary().GetCOSDictionary(cos.Dests)
	}
	if dic == nil {
		return nil
	}
	return NewPDDestinationNameTreeNodeOf(dic)
}

// SetDests sets the named destinations that are associated with this document.
func (d *PDDocumentNameDictionary) SetDests(dests *PDDestinationNameTreeNode) {
	d.nameDictionary.SetItem(cos.Dests, common.COSObjectOrNil(dests))
	// The dests can either be in the document catalog or in the
	// names dictionary, PDFBox will just maintain the one in the
	// names dictionary for now unless there is a reason to do
	// something else.
	// clear the potentially out of date Dests reference.
	d.catalog.Dictionary().SetItem(cos.Dests, nil)
}

// EmbeddedFiles returns the embedded files name tree node, whose values are
// complex file specifications, or nil where there is none.
func (d *PDDocumentNameDictionary) EmbeddedFiles() *PDEmbeddedFilesNameTreeNode {
	dic := d.nameDictionary.GetCOSDictionary(cos.EmbeddedFiles)
	if dic == nil {
		return nil
	}
	return NewPDEmbeddedFilesNameTreeNodeOf(dic)
}

// SetEmbeddedFiles sets the named embedded files that are associated with this
// document.
func (d *PDDocumentNameDictionary) SetEmbeddedFiles(ef *PDEmbeddedFilesNameTreeNode) {
	d.nameDictionary.SetItem(cos.EmbeddedFiles, common.COSObjectOrNil(ef))
}

// JavaScript returns the document level JavaScript name tree. When the document
// is opened, all the JavaScript actions in it shall be executed, defining
// JavaScript functions for use by other scripts in the document.
func (d *PDDocumentNameDictionary) JavaScript() *PDJavascriptNameTreeNode {
	dic := d.nameDictionary.GetCOSDictionary(cos.JavaScript)
	if dic == nil {
		return nil
	}
	return NewPDJavascriptNameTreeNodeOf(dic)
}

// SetJavascript sets the named javascript entries that are associated with this
// document.
func (d *PDDocumentNameDictionary) SetJavascript(js *PDJavascriptNameTreeNode) {
	d.nameDictionary.SetItem(cos.JavaScript, common.COSObjectOrNil(js))
}

// PDDocumentNameDestinationDictionary encapsulates the "dictionary of names and
// corresponding destinations" for the /Dests entry in the document catalog.
//
// Port of PDDocumentNameDestinationDictionary.
type PDDocumentNameDestinationDictionary struct {
	nameDictionary *cos.Dictionary
}

var _ common.COSObjectable = (*PDDocumentNameDestinationDictionary)(nil)

// NewPDDocumentNameDestinationDictionary returns a dictionary of names and
// corresponding destinations over the given dictionary.
func NewPDDocumentNameDestinationDictionary(
	dict *cos.Dictionary) *PDDocumentNameDestinationDictionary {
	return &PDDocumentNameDestinationDictionary{nameDictionary: dict}
}

// COSObject returns the dictionary.
func (d *PDDocumentNameDestinationDictionary) COSObject() cos.Base { return d.nameDictionary }

// Dictionary returns the dictionary, which is getCOSObject narrowed the way
// Java declares it.
func (d *PDDocumentNameDestinationDictionary) Dictionary() *cos.Dictionary {
	return d.nameDictionary
}

// Destination returns the destination corresponding to the parameter, or nil
// where there is not any.
func (d *PDDocumentNameDestinationDictionary) Destination(
	name string) (destination.PDDestination, error) {
	item := d.nameDictionary.GetDictionaryObject(cos.GetPDFName(name))

	// "The value of this entry shall be a dictionary in which each key is a destination name
	// and the corresponding value is either an array defining the destination (...)
	// or a dictionary with a D entry whose value is such an array."
	switch value := item.(type) {
	case *cos.Array:
		return destination.Create(item)
	case *cos.Dictionary:
		if value.ContainsKey(cos.D) {
			return destination.Create(value.GetDictionaryObject(cos.D))
		}
	}
	return nil, nil
}
