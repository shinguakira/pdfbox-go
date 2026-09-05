package common

import (
	"fmt"
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// NameTreeNode is what a concrete name tree node supplies: the two abstract
// methods of PDNameTreeNode, and the concrete ones, so that a kid can be used
// through the interface.
//
// Java's PDNameTreeNode is an abstract class; the port splits it into this
// interface for the contract and the embedded struct below for the state, per
// migration/conventions/java-to-go.md.
type NameTreeNode[T COSObjectable] interface {
	COSObjectable

	// Dictionary returns the node dictionary, which getCOSObject narrows to in
	// Java.
	Dictionary() *cos.Dictionary

	// ConvertCOSToPD is the abstract convertCOSToPD.
	ConvertCOSToPD(base cos.Base) (T, error)

	// CreateChildNode is the abstract createChildNode.
	CreateChildNode(dic *cos.Dictionary) NameTreeNode[T]

	// Parent returns the parent node, or nil for a root.
	Parent() NameTreeNode[T]

	// SetParent sets the parent node.
	SetParent(parentNode NameTreeNode[T])

	// IsRootNode reports whether this node has no parent.
	IsRootNode() bool

	// Kids returns the child nodes, or nil where there are none.
	Kids() *COSArrayList[NameTreeNode[T]]

	// SetKids sets the child nodes.
	SetKids(kids []NameTreeNode[T])

	// Value looks a name up in this node and its children.
	Value(name string) (T, error)

	// Names returns the name to value mapping of this node, or nil where the
	// node has none.
	Names() (map[string]T, error)

	// SetNames sets the name to value mapping of this node.
	SetNames(names map[string]T)

	// UpperLimit returns the highest name in this node's subtree.
	UpperLimit() string

	// LowerLimit returns the lowest name in this node's subtree.
	LowerLimit() string
}

// PDNameTreeNode carries the state and the concrete methods of a name tree
// node.
//
// Port of the non-abstract half of
// org.apache.pdfbox.pdmodel.common.PDNameTreeNode. The two abstract methods
// reach the concrete node through self, since Go embedding does not dispatch.
type PDNameTreeNode[T COSObjectable] struct {
	self   NameTreeNode[T]
	node   *cos.Dictionary
	parent NameTreeNode[T]
}

// InitNameTreeNode is the protected PDNameTreeNode() constructor. A concrete
// node calls it from its own constructor with itself as self.
func (n *PDNameTreeNode[T]) InitNameTreeNode(self NameTreeNode[T]) {
	n.self = self
	n.node = cos.NewDictionary()
}

// InitNameTreeNodeOf is the protected PDNameTreeNode(COSDictionary)
// constructor.
func (n *PDNameTreeNode[T]) InitNameTreeNodeOf(self NameTreeNode[T], dict *cos.Dictionary) {
	n.self = self
	n.node = dict
}

// COSObject returns the node dictionary.
func (n *PDNameTreeNode[T]) COSObject() cos.Base { return n.node }

// Dictionary returns the node dictionary, typed.
func (n *PDNameTreeNode[T]) Dictionary() *cos.Dictionary { return n.node }

// Parent returns the parent node, or nil for a root.
func (n *PDNameTreeNode[T]) Parent() NameTreeNode[T] { return n.parent }

// SetParent sets the parent node and recalculates the limits.
func (n *PDNameTreeNode[T]) SetParent(parentNode NameTreeNode[T]) {
	n.parent = parentNode
	n.calculateLimits()
}

// IsRootNode reports whether this node has no parent.
func (n *PDNameTreeNode[T]) IsRootNode() bool { return n.parent == nil }

// Kids returns the child nodes, or nil where the node has no /Kids.
func (n *PDNameTreeNode[T]) Kids() *COSArrayList[NameTreeNode[T]] {
	kids := n.node.GetCOSArray(cos.Kids)
	if kids == nil {
		return nil
	}
	pdObjects := make([]NameTreeNode[T], 0, kids.Size())
	for i := 0; i < kids.Size(); i++ {
		base := kids.GetObject(i)
		var childNode NameTreeNode[T]
		if dictionary, ok := asNodeDictionary(base); ok {
			childNode = n.self.CreateChildNode(dictionary)
		} else {
			slog.Warn("pdmodel: Bad child node", "position", i)
			childNode = n.self.CreateChildNode(cos.NewDictionary())
		}
		pdObjects = append(pdObjects, childNode)
	}
	return NewCOSArrayListOf(pdObjects, kids)
}

// asNodeDictionary is Java's `instanceof COSDictionary`, which a COSStream also
// satisfies.
func asNodeDictionary(base cos.Base) (*cos.Dictionary, bool) {
	switch value := base.(type) {
	case *cos.Stream:
		return &value.Dictionary, true
	case *cos.Dictionary:
		return value, true
	}
	return nil, false
}

// SetKids sets the child nodes.
func (n *PDNameTreeNode[T]) SetKids(kids []NameTreeNode[T]) {
	if len(kids) > 0 {
		for _, kidsNode := range kids {
			kidsNode.SetParent(n.self)
		}
		n.node.SetItem(cos.Kids, arrayOfNameTreeNodes(kids))
		// root nodes with kids don't have Names
		if n.self.IsRootNode() {
			n.node.SetItem(cos.Names, nil)
		}
	} else {
		// remove kids
		n.node.SetItem(cos.Kids, nil)
		// remove Limits
		n.node.SetItem(cos.Limits, nil)
	}
	n.calculateLimits()
}

// arrayOfNameTreeNodes is Java's `new COSArray(kids)`, the COSArray constructor
// that takes a list of COSObjectable. Slice 1 did not port that overload
// because it names a pdmodel type.
func arrayOfNameTreeNodes[T COSObjectable](kids []NameTreeNode[T]) *cos.Array {
	array := cos.NewArray()
	for _, kid := range kids {
		array.Add(kid.COSObject())
	}
	return array
}

func (n *PDNameTreeNode[T]) calculateLimits() {
	if n.self.IsRootNode() {
		n.node.SetItem(cos.Limits, nil)
		return
	}
	kids := n.self.Kids()
	if kids != nil && !kids.IsEmpty() {
		firstKid := kids.Get(0)
		lastKid := kids.Get(kids.Size() - 1)
		lowerLimit := firstKid.LowerLimit()
		n.setLowerLimit(lowerLimit)
		upperLimit := lastKid.UpperLimit()
		n.setUpperLimit(upperLimit)
		return
	}
	keys, _, err := n.namesInOrder()
	if err != nil {
		n.node.SetItem(cos.Limits, nil)
		slog.Error("pdmodel: Error while calculating the Limits of a PageNameTreeNode:", "err", err)
		return
	}
	if len(keys) > 0 {
		lowerLimit := keys[0]
		n.setLowerLimit(lowerLimit)
		upperLimit := keys[len(keys)-1]
		n.setUpperLimit(upperLimit)
	} else {
		n.node.SetItem(cos.Limits, nil)
	}
}

// Value looks a name up in this node and its children.
func (n *PDNameTreeNode[T]) Value(name string) (T, error) {
	var zero T
	names, err := n.self.Names()
	if err != nil {
		return zero, err
	}
	if names != nil {
		return names[name], nil
	}
	kids := n.self.Kids()
	if kids == nil {
		slog.Warn("pdmodel: NameTreeNode does not have \"names\" nor \"kids\" objects.")
		return zero, nil
	}
	for i := 0; i < kids.Size(); i++ {
		childNode := kids.Get(i)
		// Java compares against null, which getUpperLimit and getLowerLimit
		// answer only where the child has no /Limits array or its entry is not a
		// string. The empty name is a legal limit and is not null there, so it
		// must not be read as an absent one: a child whose range is ["" "a"]
		// would otherwise look unlimited and end the search.
		upperLimit, hasUpperLimit := limitOf(childNode, 1)
		lowerLimit, hasLowerLimit := limitOf(childNode, 0)
		if !hasUpperLimit || !hasLowerLimit || upperLimit < lowerLimit ||
			(lowerLimit <= name && upperLimit >= name) {
			return childNode.Value(name)
		}
	}
	return zero, nil
}

// Names returns the name to value mapping of this node, or nil where the node
// has no /Names.
//
// Java returns an unmodifiable LinkedHashMap, so the caller sees the array
// order; a Go map has no order, and calculateLimits needs the first and last
// key as the array holds them, so namesInOrder keeps that alongside.
func (n *PDNameTreeNode[T]) Names() (map[string]T, error) {
	_, names, err := n.namesInOrder()
	return names, err
}

// namesInOrder reads /Names, returning the keys in the order the array holds
// them and the mapping. Both are nil where there is no /Names array.
func (n *PDNameTreeNode[T]) namesInOrder() ([]string, map[string]T, error) {
	namesArray := n.node.GetCOSArray(cos.Names)
	if namesArray == nil {
		return nil, nil, nil
	}
	size := namesArray.Size()
	keys := make([]string, 0, size/2)
	names := make(map[string]T, size/2)
	if size%2 != 0 {
		slog.Warn("pdmodel: Names array has odd size", "size", size)
	}
	for i := 0; i+1 < size; i += 2 {
		base := namesArray.GetObject(i)
		key, ok := base.(*cos.StringObj)
		if !ok {
			return nil, nil, fmt.Errorf("Expected string, found %v in name tree at index %d", base, i)
		}
		cosValue := namesArray.GetObject(i + 1)
		value, err := n.self.ConvertCOSToPD(cosValue)
		if err != nil {
			return nil, nil, err
		}
		keys = append(keys, key.Value())
		names[key.Value()] = value
	}
	return keys, names, nil
}

// SetNames sets the name to value mapping of this node.
//
// A nil map is Java's null, which clears the entry.
func (n *PDNameTreeNode[T]) SetNames(names map[string]T) {
	if names == nil {
		n.node.SetItem(cos.Names, nil)
		n.node.SetItem(cos.Limits, nil)
		return
	}
	array := cos.NewArray()
	keys := sortedKeys(names)
	for _, key := range keys {
		array.Add(cos.NewStringObj(key))
		array.Add(names[key].COSObject())
	}
	n.node.SetItem(cos.Names, array)
	n.calculateLimits()
}

// UpperLimit returns the highest name in this node's subtree, or "" where the
// node has no /Limits.
//
// Java returns null, which every caller compares against null; the port returns
// the empty string, which is what COSArray.getString gives for a missing entry.
func (n *PDNameTreeNode[T]) UpperLimit() string {
	if arr := n.node.GetCOSArray(cos.Limits); arr != nil {
		return arr.GetString(1, "")
	}
	return ""
}

func (n *PDNameTreeNode[T]) setUpperLimit(upper string) {
	arr := n.node.GetCOSArray(cos.Limits)
	if arr == nil {
		arr = cos.NewArray()
		arr.Add(nil)
		arr.Add(nil)
		n.node.SetItem(cos.Limits, arr)
	}
	arr.SetString(1, upper)
}

// LowerLimit returns the lowest name in this node's subtree, or "".
func (n *PDNameTreeNode[T]) LowerLimit() string {
	if arr := n.node.GetCOSArray(cos.Limits); arr != nil {
		return arr.GetString(0, "")
	}
	return ""
}

func (n *PDNameTreeNode[T]) setLowerLimit(lower string) {
	arr := n.node.GetCOSArray(cos.Limits)
	if arr == nil {
		arr = cos.NewArray()
		arr.Add(nil)
		arr.Add(nil)
		n.node.SetItem(cos.Limits, arr)
	}
	arr.SetString(0, lower)
}

// limitOf returns one entry of a node's /Limits array, and reports whether it
// is there and is a string.
//
// Java's getUpperLimit and getLowerLimit answer null in either of those cases
// and the string otherwise; the port's accessors answer "" for both, which
// cannot tell an absent limit from the empty name. Value needs to, so it reads
// the array through this.
func limitOf[T COSObjectable](node NameTreeNode[T], index int) (string, bool) {
	arr := node.Dictionary().GetCOSArray(cos.Limits)
	if arr == nil || index >= arr.Size() {
		return "", false
	}
	str, isString := arr.GetObject(index).(*cos.StringObj)
	if !isString {
		return "", false
	}
	return str.Value(), true
}
