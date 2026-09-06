// Package outline holds the document outline: the bookmarks a reader shows
// beside the page.
//
// Port of org.apache.pdfbox.pdmodel.interactive.documentnavigation.outline.
package outline

import (
	"iter"
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// OutlineNode is a node of the outline tree: the document outline itself, or
// one item of it.
//
// Java's PDOutlineNode is an abstract class; the port splits it into this
// interface for the contract and the embedded struct below for the state. The
// interface is what updateParentOpenCount needs, since PDDocumentOutline
// overrides isNodeOpen and Go embedding does not dispatch.
type OutlineNode interface {
	common.COSObjectable

	// Dictionary returns the node dictionary, which PDDictionaryWrapper holds.
	Dictionary() *cos.Dictionary

	// IsNodeOpen reports whether this node shows its children.
	IsNodeOpen() bool

	// OpenNode makes this node show its children.
	OpenNode()

	// CloseNode makes this node hide its children.
	CloseNode()

	// OpenCount returns the /Count of this node.
	OpenCount() int

	// SetOpenCount sets the /Count of this node. Java declares it
	// package-private.
	SetOpenCount(openCount int)

	// UpdateParentOpenCount carries a change in the open count up the tree.
	// Java declares it package-private.
	UpdateParentOpenCount(delta int)

	// Parent returns the parent node, or nil. Java declares it
	// package-private.
	Parent() OutlineNode

	// SetFirstChild sets the /First child. Java declares it package-private.
	SetFirstChild(outlineNode OutlineNode)

	// SetLastChild sets the /Last child. Java declares it package-private.
	SetLastChild(outlineNode OutlineNode)
}

// PDOutlineNode carries the state and the concrete methods of an outline node.
//
// Port of the non-abstract half of PDOutlineNode.
type PDOutlineNode struct {
	common.PDDictionaryWrapper
	self OutlineNode
}

var _ common.COSObjectable = (*PDOutlineNode)(nil)

// InitOutlineNode is the protected PDOutlineNode() constructor. A concrete node
// calls it from its own constructor with itself as self.
func (n *PDOutlineNode) InitOutlineNode(self OutlineNode) {
	n.self = self
	n.PDDictionaryWrapper = *common.NewPDDictionaryWrapper()
}

// InitOutlineNodeOf is the protected PDOutlineNode(COSDictionary) constructor.
func (n *PDOutlineNode) InitOutlineNodeOf(self OutlineNode, dict *cos.Dictionary) {
	n.self = self
	n.PDDictionaryWrapper = *common.NewPDDictionaryWrapperOf(dict)
}

// Parent returns the parent node, or nil. Java declares it package-private.
func (n *PDOutlineNode) Parent() OutlineNode {
	parent := n.Dictionary().GetCOSDictionary(cos.Parent)
	if parent == nil {
		return nil
	}
	if cos.Outlines == parent.GetCOSName(cos.Type) {
		return NewPDDocumentOutlineOf(parent)
	}
	return NewPDOutlineItemOf(parent)
}

// SetParent sets the parent node. Java declares it package-private.
func (n *PDOutlineNode) SetParent(parent OutlineNode) {
	if parent == nil {
		n.Dictionary().SetItem(cos.Parent, nil)
		return
	}
	n.Dictionary().SetItem(cos.Parent, parent.COSObject())
}

// AddLast appends the given item as the last child of this node.
func (n *PDOutlineNode) AddLast(newChild *PDOutlineItem) {
	n.RequireSingleNode(newChild)
	n.append(newChild)
	n.UpdateParentOpenCountForAddedChild(newChild)
}

// AddFirst prepends the given item as the first child of this node.
func (n *PDOutlineNode) AddFirst(newChild *PDOutlineItem) {
	n.RequireSingleNode(newChild)
	n.prepend(newChild)
	n.UpdateParentOpenCountForAddedChild(newChild)
}

// RequireSingleNode panics unless the given item has no siblings. Java declares
// it package-private and throws IllegalArgumentException, which is unchecked.
func (n *PDOutlineNode) RequireSingleNode(node *PDOutlineItem) {
	if node.NextSibling() != nil || node.PreviousSibling() != nil {
		panic("A single node with no siblings is required")
	}
}

// append adds the given item after the last child.
func (n *PDOutlineNode) append(newChild *PDOutlineItem) {
	newChild.SetParent(n.self)
	if !n.HasChildren() {
		n.self.SetFirstChild(newChild)
	} else {
		previousLastChild := n.LastChild()
		previousLastChild.SetNextSibling(newChild)
		newChild.SetPreviousSibling(previousLastChild)
	}
	n.self.SetLastChild(newChild)
}

// prepend adds the given item before the first child.
func (n *PDOutlineNode) prepend(newChild *PDOutlineItem) {
	newChild.SetParent(n.self)
	if !n.HasChildren() {
		n.self.SetLastChild(newChild)
	} else {
		previousFirstChild := n.FirstChild()
		newChild.SetNextSibling(previousFirstChild)
		previousFirstChild.SetPreviousSibling(newChild)
	}
	n.self.SetFirstChild(newChild)
}

// UpdateParentOpenCountForAddedChild carries the arrival of a child up the
// tree. Java declares it package-private.
func (n *PDOutlineNode) UpdateParentOpenCountForAddedChild(newChild *PDOutlineItem) {
	delta := 1
	if newChild.IsNodeOpen() {
		delta += newChild.OpenCount()
	}
	newChild.UpdateParentOpenCount(delta)
}

// HasChildren reports whether this node has a /First child.
func (n *PDOutlineNode) HasChildren() bool {
	return n.Dictionary().GetCOSDictionary(cos.First) != nil
}

// OutlineItem returns the item the given key names, or nil. Java declares it
// package-private.
func (n *PDOutlineNode) OutlineItem(name *cos.Name) *PDOutlineItem {
	if outline := n.Dictionary().GetCOSDictionary(name); outline != nil {
		return NewPDOutlineItemOf(outline)
	}
	return nil
}

// FirstChild returns the /First child, or nil.
func (n *PDOutlineNode) FirstChild() *PDOutlineItem {
	return n.OutlineItem(cos.First)
}

// SetFirstChild sets the /First child. Java declares it package-private.
func (n *PDOutlineNode) SetFirstChild(outlineNode OutlineNode) {
	if outlineNode == nil {
		n.Dictionary().SetItem(cos.First, nil)
		return
	}
	n.Dictionary().SetItem(cos.First, outlineNode.COSObject())
}

// LastChild returns the /Last child, or nil.
func (n *PDOutlineNode) LastChild() *PDOutlineItem {
	return n.OutlineItem(cos.Last)
}

// SetLastChild sets the /Last child. Java declares it package-private.
func (n *PDOutlineNode) SetLastChild(outlineNode OutlineNode) {
	if outlineNode == nil {
		n.Dictionary().SetItem(cos.Last, nil)
		return
	}
	n.Dictionary().SetItem(cos.Last, outlineNode.COSObject())
}

// OpenCount returns the /Count of this node.
func (n *PDOutlineNode) OpenCount() int {
	return n.Dictionary().GetIntDefault(cos.Count, 0)
}

// SetOpenCount sets the /Count of this node. Java declares it package-private.
func (n *PDOutlineNode) SetOpenCount(openCount int) {
	n.Dictionary().SetInt(cos.Count, openCount)
}

// OpenNode makes this node show its children.
func (n *PDOutlineNode) OpenNode() {
	// if the node is already open then do nothing.
	if !n.self.IsNodeOpen() {
		n.switchNodeCount()
	}
}

// CloseNode makes this node hide its children.
func (n *PDOutlineNode) CloseNode() {
	if n.self.IsNodeOpen() {
		n.switchNodeCount()
	}
}

// switchNodeCount flips the sign of the open count.
func (n *PDOutlineNode) switchNodeCount() {
	openCount := n.self.OpenCount()
	n.self.SetOpenCount(-openCount)
	n.self.UpdateParentOpenCount(-openCount)
}

// IsNodeOpen reports whether this node shows its children.
func (n *PDOutlineNode) IsNodeOpen() bool {
	return n.self.OpenCount() > 0
}

// UpdateParentOpenCount carries a change in the open count up the tree. Java
// declares it package-private.
func (n *PDOutlineNode) UpdateParentOpenCount(delta int) {
	parent := n.self.Parent()
	if parent == nil {
		return
	}
	if n.Dictionary() == parent.Dictionary() {
		// PDFBOX-5939
		slog.Warn("outline: outline parent points to itself")
		return
	}
	if parent.IsNodeOpen() {
		parent.SetOpenCount(parent.OpenCount() + delta)
		parent.UpdateParentOpenCount(delta)
		return
	}
	parent.SetOpenCount(parent.OpenCount() - delta)
}

// Children walks the children of this node.
//
// Java answers an Iterable over a PDOutlineItemIterator; the port answers the
// range-over-function sequence Go reads the same way.
func (n *PDOutlineNode) Children() iter.Seq[*PDOutlineItem] {
	return func(yield func(*PDOutlineItem) bool) {
		iterator := NewPDOutlineItemIterator(n.FirstChild())
		for iterator.HasNext() {
			if !yield(iterator.Next()) {
				return
			}
		}
	}
}
