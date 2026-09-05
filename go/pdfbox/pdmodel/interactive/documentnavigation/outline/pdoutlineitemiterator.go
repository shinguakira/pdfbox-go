package outline

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// PDOutlineItemIterator walks the siblings of an outline item, stopping at one
// it has already seen so that a loop in the file does not loop here.
//
// Port of PDOutlineItemIterator, which Java declares package-private.
type PDOutlineItemIterator struct {
	currentItem  *PDOutlineItem
	startingItem *PDOutlineItem
	visited      map[*cos.Dictionary]bool
}

// NewPDOutlineItemIterator builds an iterator over the given item and the
// siblings after it.
func NewPDOutlineItemIterator(startingItem *PDOutlineItem) *PDOutlineItemIterator {
	return &PDOutlineItemIterator{
		startingItem: startingItem,
		visited:      map[*cos.Dictionary]bool{},
	}
}

// HasNext reports whether there is another item.
func (i *PDOutlineItemIterator) HasNext() bool {
	if i.startingItem == nil {
		return false
	}
	if i.currentItem == nil {
		return true
	}
	sibling := i.currentItem.NextSibling()
	return sibling != nil && !i.visited[sibling.Dictionary()]
}

// Next returns the next item, and panics where there is none, which is the
// NoSuchElementException Java throws.
func (i *PDOutlineItemIterator) Next() *PDOutlineItem {
	if !i.HasNext() {
		panic("outline: no such element")
	}
	if i.currentItem == nil {
		i.currentItem = i.startingItem
	} else {
		i.currentItem = i.currentItem.NextSibling()
	}
	i.visited[i.currentItem.Dictionary()] = true
	return i.currentItem
}

// Remove panics, which is the UnsupportedOperationException Java throws.
func (i *PDOutlineItemIterator) Remove() {
	panic("outline: remove is not supported")
}
