// Package destination names the place in a document a link jumps to.
//
// Port of org.apache.pdfbox.pdmodel.interactive.documentnavigation.destination.
package destination

import (
	"fmt"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// The /D array type names, which are the TYPE and TYPE_BOUNDED constants of the
// five concrete destinations. Java declares each on its own class as protected
// static final; Go has no such level and one package, so they sit together.
const (
	typeFit         = "Fit"
	typeFitBounded  = "FitB"
	typeFitV        = "FitV"
	typeFitVBounded = "FitBV"
	typeFitH        = "FitH"
	typeFitHBounded = "FitBH"
	typeFitR        = "FitR"
	typeXYZ         = "XYZ"
)

// PageLike is the page a destination points at.
//
// Java names PDPage and PDPageTree, which live in pdmodel; pdmodel imports this
// package for the outline and the annotations, so the dependency cannot run
// both ways. The port names what is used and takes the constructors below,
// which pdmodel sets from its init.
type PageLike interface {
	common.COSObjectable
}

// NewPageFromDictionary builds a page from its dictionary. pdmodel sets it.
var NewPageFromDictionary func(dic *cos.Dictionary) PageLike

// IndexOfPageInTree returns the index of the page the given dictionary holds
// within the page tree rooted at the given one, or -1. It is
// `new PDPageTree(parent).indexOf(new PDPage(pageDict))`. pdmodel sets it.
var IndexOfPageInTree func(root, pageDict *cos.Dictionary) int

// PDDestination is a place in a document.
//
// Port of the abstract class PDDestination. It holds no state, so the port is
// an interface for the contract and the concrete types below.
type PDDestination interface {
	common.PDDestinationOrAction
}

// Create returns the destination the given object holds.
//
// Port of the static create(COSBase).
func Create(base cos.Base) (PDDestination, error) {
	switch value := base.(type) {
	case nil:
		// this is ok, just return null.
		return nil, nil
	case *cos.Array:
		if value.Size() > 1 {
			if typeName, ok := value.GetObject(1).(*cos.Name); ok {
				return createOfArray(value, typeName)
			}
		}
		return nil, fmt.Errorf("Error: can't convert to Destination %v", base)
	case *cos.StringObj:
		return NewPDNamedDestinationOfString(value), nil
	case *cos.Name:
		return NewPDNamedDestinationOfName(value), nil
	}
	return nil, fmt.Errorf("Error: can't convert to Destination %v", base)
}

func createOfArray(array *cos.Array, typeName *cos.Name) (PDDestination, error) {
	switch typeName.Name() {
	case typeFit, typeFitBounded:
		return NewPDPageFitDestinationOf(array), nil
	case typeFitV, typeFitVBounded:
		return NewPDPageFitHeightDestinationOf(array), nil
	case typeFitR:
		return NewPDPageFitRectangleDestinationOf(array), nil
	case typeFitH, typeFitHBounded:
		return NewPDPageFitWidthDestinationOf(array), nil
	case typeXYZ:
		return NewPDPageXYZDestinationOf(array), nil
	}
	return nil, fmt.Errorf("Unknown destination type: %s", typeName.Name())
}

// PDNamedDestination is a destination named rather than spelled out.
//
// Port of PDNamedDestination.
type PDNamedDestination struct {
	namedDestination cos.Base
}

var _ PDDestination = (*PDNamedDestination)(nil)

// NewPDNamedDestinationOfString creates a destination named by a string.
func NewPDNamedDestinationOfString(dest *cos.StringObj) *PDNamedDestination {
	return &PDNamedDestination{namedDestination: dest}
}

// NewPDNamedDestinationOfName creates a destination named by a name.
func NewPDNamedDestinationOfName(dest *cos.Name) *PDNamedDestination {
	return &PDNamedDestination{namedDestination: dest}
}

// NewPDNamedDestination creates a destination naming nothing.
func NewPDNamedDestination() *PDNamedDestination {
	// default, so do nothing
	return &PDNamedDestination{}
}

// NewPDNamedDestinationOf creates a destination named by the given string.
func NewPDNamedDestinationOf(dest string) *PDNamedDestination {
	return &PDNamedDestination{namedDestination: cos.NewStringObj(dest)}
}

// COSObject returns the name or string.
func (d *PDNamedDestination) COSObject() cos.Base { return d.namedDestination }

// NamedDestination returns the name, or "" where there is none.
func (d *PDNamedDestination) NamedDestination() string {
	switch value := d.namedDestination.(type) {
	case *cos.StringObj:
		return value.Value()
	case *cos.Name:
		return value.Name()
	}
	return ""
}

// SetNamedDestination sets the name. The empty string is Java's null, which
// clears it.
func (d *PDNamedDestination) SetNamedDestination(dest string) {
	if dest == "" {
		d.namedDestination = nil
		return
	}
	d.namedDestination = cos.NewStringObj(dest)
}

// PDPageDestination is a destination that names a page.
//
// Port of the abstract class PDPageDestination; the port keeps the state and
// the concrete methods here and each concrete destination embeds it.
type PDPageDestination struct {
	// Array is the protected `array` field the five subclasses write through.
	Array *cos.Array
}

// InitPageDestination is the protected PDPageDestination() constructor.
func (d *PDPageDestination) InitPageDestination() {
	d.Array = cos.NewArray()
}

// InitPageDestinationOf is the protected PDPageDestination(COSArray)
// constructor.
func (d *PDPageDestination) InitPageDestinationOf(arr *cos.Array) {
	d.Array = arr
}

// Page returns the page this destination names, or nil where it names a page
// number instead.
func (d *PDPageDestination) Page() PageLike {
	if d.Array.IsEmpty() {
		return nil
	}
	if page, ok := asDictionary(d.Array.GetObject(0)); ok {
		return NewPageFromDictionary(page)
	}
	return nil
}

// asDictionary is Java's `instanceof COSDictionary`, which a COSStream also
// satisfies.
func asDictionary(base cos.Base) (*cos.Dictionary, bool) {
	switch value := base.(type) {
	case *cos.Stream:
		return &value.Dictionary, true
	case *cos.Dictionary:
		return value, true
	}
	return nil, false
}

// SetPage sets the page this destination names.
func (d *PDPageDestination) SetPage(page PageLike) {
	if page == nil {
		d.Array.Set(0, nil)
		return
	}
	d.Array.Set(0, page.COSObject())
}

// PageNumber returns the page number this destination names, or -1 where it
// names a page dictionary instead.
func (d *PDPageDestination) PageNumber() int {
	if d.Array.IsEmpty() {
		return -1
	}
	if page, ok := d.Array.GetObject(0).(cos.Number); ok {
		return int(page.IntValue())
	}
	return -1
}

// RetrievePageNumber returns the page number this destination names, working it
// out from the page tree where it names a page dictionary.
func (d *PDPageDestination) RetrievePageNumber() int {
	if d.Array.IsEmpty() {
		return -1
	}
	page := d.Array.GetObject(0)
	if number, ok := page.(cos.Number); ok {
		return int(number.IntValue())
	}
	if dictionary, ok := asDictionary(page); ok {
		return indexOfPageTree(dictionary)
	}
	return -1
}

// indexOfPageTree climbs up the page tree up to the top to be able to call
// PageTree.indexOf for a page dictionary.
func indexOfPageTree(pageDict *cos.Dictionary) int {
	parent := pageDict
	for {
		prevParent := parent.GetCOSDictionary2(cos.Parent, cos.P)
		if prevParent == nil {
			break
		}
		parent = prevParent
	}
	if parent.ContainsKey(cos.Kids) && parent.GetCOSName(cos.Type) == cos.Pages {
		// now parent is the highest pages node
		return IndexOfPageInTree(parent, pageDict)
	}
	return -1
}

// SetPageNumber sets the page number this destination names.
func (d *PDPageDestination) SetPageNumber(pageNumber int) {
	d.Array.SetInt(0, pageNumber)
}

// COSObject returns the array.
func (d *PDPageDestination) COSObject() cos.Base { return d.Array }

// PageDestination is what the five page destinations share, and what an
// instanceof PDPageDestination asks in Java.
//
// PDPageDestination is an abstract class there; the port keeps its state in the
// struct above, which each concrete destination embeds, so a caller that needs
// to know whether a destination is one of them asserts to this.
type PageDestination interface {
	PDDestination

	// Page returns the page this destination names, or nil.
	Page() PageLike

	// SetPage sets the page this destination names.
	SetPage(page PageLike)

	// PageNumber returns the page number this destination names, or -1.
	PageNumber() int

	// RetrievePageNumber returns the page number this destination names,
	// whether it holds one or a page dictionary, or -1.
	RetrievePageNumber() int

	// SetPageNumber sets the page number this destination names.
	SetPageNumber(pageNumber int)
}

var (
	_ PageDestination = (*PDPageFitDestination)(nil)
	_ PageDestination = (*PDPageFitHeightDestination)(nil)
	_ PageDestination = (*PDPageFitWidthDestination)(nil)
	_ PageDestination = (*PDPageFitRectangleDestination)(nil)
	_ PageDestination = (*PDPageXYZDestination)(nil)
)
