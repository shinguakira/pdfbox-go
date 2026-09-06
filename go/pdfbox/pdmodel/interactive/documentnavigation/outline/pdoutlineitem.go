package outline

import (
	"fmt"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/documentinterchange/logicalstructure"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/action"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/documentnavigation/destination"
)

// The bits of the /F flags of an outline item. Java declares them private.
const (
	italicFlag = 1
	boldFlag   = 2
)

// DocumentLike is the document an outline item looks a destination up in.
//
// Java names PDDocument, which lives in pdmodel; pdmodel imports this package
// for PDDocumentCatalog.getDocumentOutline, so the dependency cannot run both
// ways. The port names what is used, which a *pdmodel.PDDocument satisfies.
type DocumentLike interface {
	// FindNamedDestinationPage is
	// getDocumentCatalog().findNamedDestinationPage(namedDest).
	FindNamedDestinationPage(namedDest *destination.PDNamedDestination) (destination.PageDestination, error)

	// PageAt is getPage(pageIndex).
	PageAt(pageIndex int) destination.PageLike
}

// PDOutlineItem is one bookmark of the outline tree.
//
// Port of PDOutlineItem, which Java declares final.
type PDOutlineItem struct {
	PDOutlineNode
}

var _ OutlineNode = (*PDOutlineItem)(nil)

// NewPDOutlineItem builds an empty outline item.
func NewPDOutlineItem() *PDOutlineItem {
	i := &PDOutlineItem{}
	i.InitOutlineNode(i)
	return i
}

// NewPDOutlineItemOf builds one over the given dictionary.
func NewPDOutlineItemOf(dic *cos.Dictionary) *PDOutlineItem {
	i := &PDOutlineItem{}
	i.InitOutlineNodeOf(i, dic)
	return i
}

// InsertSiblingAfter puts the given item after this one.
func (i *PDOutlineItem) InsertSiblingAfter(newSibling *PDOutlineItem) {
	i.RequireSingleNode(newSibling)
	parent := i.Parent()
	newSibling.SetParent(parent)
	next := i.NextSibling()
	i.SetNextSibling(newSibling)
	newSibling.SetPreviousSibling(i)
	if next != nil {
		newSibling.SetNextSibling(next)
		next.SetPreviousSibling(newSibling)
	} else if parent != nil {
		i.Parent().SetLastChild(newSibling)
	}
	i.UpdateParentOpenCountForAddedChild(newSibling)
}

// InsertSiblingBefore puts the given item before this one.
func (i *PDOutlineItem) InsertSiblingBefore(newSibling *PDOutlineItem) {
	i.RequireSingleNode(newSibling)
	parent := i.Parent()
	newSibling.SetParent(parent)
	previous := i.PreviousSibling()
	i.SetPreviousSibling(newSibling)
	newSibling.SetNextSibling(i)
	if previous != nil {
		previous.SetNextSibling(newSibling)
		newSibling.SetPreviousSibling(previous)
	} else if parent != nil {
		i.Parent().SetFirstChild(newSibling)
	}
	i.UpdateParentOpenCountForAddedChild(newSibling)
}

// PreviousSibling returns the /Prev item, or nil.
func (i *PDOutlineItem) PreviousSibling() *PDOutlineItem {
	return i.OutlineItem(cos.Prev)
}

// SetPreviousSibling sets the /Prev item. Java declares it package-private.
func (i *PDOutlineItem) SetPreviousSibling(outlineNode OutlineNode) {
	if outlineNode == nil {
		i.Dictionary().SetItem(cos.Prev, nil)
		return
	}
	i.Dictionary().SetItem(cos.Prev, outlineNode.COSObject())
}

// NextSibling returns the /Next item, or nil.
func (i *PDOutlineItem) NextSibling() *PDOutlineItem {
	return i.OutlineItem(cos.Next)
}

// SetNextSibling sets the /Next item. Java declares it package-private.
func (i *PDOutlineItem) SetNextSibling(outlineNode OutlineNode) {
	if outlineNode == nil {
		i.Dictionary().SetItem(cos.Next, nil)
		return
	}
	i.Dictionary().SetItem(cos.Next, outlineNode.COSObject())
}

// Title returns the /Title of this item.
func (i *PDOutlineItem) Title() string {
	return i.Dictionary().GetString(cos.Title, "")
}

// SetTitle sets the /Title of this item.
func (i *PDOutlineItem) SetTitle(title string) {
	i.Dictionary().SetString(cos.Title, title)
}

// Destination returns the /Dest of this item, or nil.
func (i *PDOutlineItem) Destination() (destination.PDDestination, error) {
	return destination.Create(i.Dictionary().GetDictionaryObject(cos.Dest))
}

// SetDestination sets the /Dest of this item.
func (i *PDOutlineItem) SetDestination(dest destination.PDDestination) {
	if dest == nil {
		i.Dictionary().SetItem(cos.Dest, nil)
		return
	}
	i.Dictionary().SetItem(cos.Dest, dest.COSObject())
}

// SetDestinationPage points this item at the top left of the given page, and
// clears the destination where the page is nil.
func (i *PDOutlineItem) SetDestinationPage(page destination.PageLike) {
	var dest *destination.PDPageXYZDestination
	if page != nil {
		dest = destination.NewPDPageXYZDestination()
		dest.SetPage(page)
	}
	if dest == nil {
		i.SetDestination(nil)
		return
	}
	i.SetDestination(dest)
}

// FindDestinationPage returns the page this item points at, through its
// destination or through a go to action, or nil where there is none.
func (i *PDOutlineItem) FindDestinationPage(doc DocumentLike) (destination.PageLike, error) {
	dest, err := i.Destination()
	if err != nil {
		return nil, err
	}
	if dest == nil {
		if outlineAction, isGoTo := i.Action().(*action.PDActionGoTo); isGoTo {
			dest, err = outlineAction.Destination()
			if err != nil {
				return nil, err
			}
		}
	}
	if dest == nil {
		return nil, nil
	}
	var pageDestination destination.PageDestination
	switch value := dest.(type) {
	case *destination.PDNamedDestination:
		pageDestination, err = doc.FindNamedDestinationPage(value)
		if err != nil {
			return nil, err
		}
		if pageDestination == nil {
			return nil, nil
		}
	case destination.PageDestination:
		pageDestination = value
	default:
		return nil, fmt.Errorf("Error: Unknown destination type %v", dest)
	}
	page := pageDestination.Page()
	if page == nil {
		// Malformed PDF: local destinations must have a page object,
		// not a page number, these are meant for remote destinations.
		pageNumber := pageDestination.PageNumber()
		if pageNumber != -1 {
			page = doc.PageAt(pageNumber)
		}
	}
	return page, nil
}

// Action returns the /A action of this item, or nil.
func (i *PDOutlineItem) Action() action.Action {
	return action.CreateAction(i.Dictionary().GetCOSDictionary(cos.A))
}

// SetAction sets the /A action of this item.
func (i *PDOutlineItem) SetAction(outlineAction action.Action) {
	if outlineAction == nil {
		i.Dictionary().SetItem(cos.A, nil)
		return
	}
	i.Dictionary().SetItem(cos.A, outlineAction.COSObject())
}

// StructureElement returns the /SE structure element of this item, or nil.
func (i *PDOutlineItem) StructureElement() *logicalstructure.PDStructureElement {
	if dic := i.Dictionary().GetCOSDictionary(cos.SE); dic != nil {
		return logicalstructure.NewPDStructureElementOf(dic)
	}
	return nil
}

// SetStructureElement sets the /SE structure element of this item.
func (i *PDOutlineItem) SetStructureElement(
	structureElement *logicalstructure.PDStructureElement) {
	if structureElement == nil {
		i.Dictionary().SetItem(cos.SE, nil)
		return
	}
	i.Dictionary().SetItem(cos.SE, structureElement.COSObject())
}

// TextColor returns the /C colour of the title, writing the default of black
// into the dictionary where there is none.
func (i *PDOutlineItem) TextColor() *color.PDColor {
	csValues := i.Dictionary().GetCOSArray(cos.C)
	if csValues == nil {
		csValues = cos.NewArray()
		csValues.GrowToSizeWith(3, cos.NewFloat(0))
		i.Dictionary().SetItem(cos.C, csValues)
	}
	return color.NewPDColorOfCOSArray(csValues, color.DeviceRGB)
}

// SetTextColor sets the /C colour of the title.
func (i *PDOutlineItem) SetTextColor(textColor *color.PDColor) {
	i.Dictionary().SetItem(cos.C, textColor.ToCOSArray())
}

// SetTextColorRGB sets the /C colour of the title from three components of
// zero to 255.
//
// Java takes a java.awt.Color and reads its red, green and blue the same way;
// Go has no such type, so the port takes the three components themselves.
func (i *PDOutlineItem) SetTextColorRGB(red, green, blue int) {
	array := cos.NewArray()
	array.Add(cos.NewFloat(float32(red) / 255))
	array.Add(cos.NewFloat(float32(green) / 255))
	array.Add(cos.NewFloat(float32(blue) / 255))
	i.Dictionary().SetItem(cos.C, array)
}

// IsItalic reports whether the title is drawn in italic.
func (i *PDOutlineItem) IsItalic() bool {
	return i.Dictionary().GetFlag(cos.F, italicFlag)
}

// SetItalic sets whether the title is drawn in italic.
func (i *PDOutlineItem) SetItalic(italic bool) {
	i.Dictionary().SetFlag(cos.F, italicFlag, italic)
}

// IsBold reports whether the title is drawn in bold.
func (i *PDOutlineItem) IsBold() bool {
	return i.Dictionary().GetFlag(cos.F, boldFlag)
}

// SetBold sets whether the title is drawn in bold.
func (i *PDOutlineItem) SetBold(bold bool) {
	i.Dictionary().SetFlag(cos.F, boldFlag, bold)
}
