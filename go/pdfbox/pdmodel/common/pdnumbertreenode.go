package common

import (
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// ValueConverter turns the COS form of a number tree value into its PD form.
//
// Java stores a Class and calls
// `valueType.getDeclaredConstructor(base.getClass()).newInstance(base)`, which
// is a reflective constructor lookup. Go has no such thing, and the port's
// convention for a reflection factory is a constructor passed in; see
// migration/conventions/java-to-go.md.
type ValueConverter func(base cos.Base) (COSObjectable, error)

// PDNumberTreeNode is a node of a number tree.
//
// Port of org.apache.pdfbox.pdmodel.common.PDNumberTreeNode. It is concrete in
// Java, so the port needs no self-dispatch: createChildNode is overridden by
// nothing in the main tree.
type PDNumberTreeNode struct {
	node *cos.Dictionary
	// valueType is Java's Class field, as the constructor it stands for.
	valueType ValueConverter
}

var _ COSObjectable = (*PDNumberTreeNode)(nil)

// NewPDNumberTreeNode creates a node whose values are built by the given
// converter.
func NewPDNumberTreeNode(valueClass ValueConverter) *PDNumberTreeNode {
	return &PDNumberTreeNode{node: cos.NewDictionary(), valueType: valueClass}
}

// NewPDNumberTreeNodeOf creates a node over the given dictionary.
func NewPDNumberTreeNodeOf(dict *cos.Dictionary, valueClass ValueConverter) *PDNumberTreeNode {
	return &PDNumberTreeNode{node: dict, valueType: valueClass}
}

// COSObject returns the node dictionary.
func (n *PDNumberTreeNode) COSObject() cos.Base { return n.node }

// Dictionary returns the node dictionary, typed.
func (n *PDNumberTreeNode) Dictionary() *cos.Dictionary { return n.node }

// Kids returns the child nodes, or nil where the node has no /Kids.
func (n *PDNumberTreeNode) Kids() *COSArrayList[*PDNumberTreeNode] {
	kids := n.node.GetCOSArray(cos.Kids)
	if kids == nil {
		return nil
	}
	pdObjects := make([]*PDNumberTreeNode, 0, kids.Size())
	for i := 0; i < kids.Size(); i++ {
		base := kids.GetObject(i)
		var childNode *PDNumberTreeNode
		if dictionary, ok := asNodeDictionary(base); ok {
			childNode = n.CreateChildNode(dictionary)
		} else {
			slog.Warn("pdmodel: Bad child node", "position", i)
			childNode = NewPDNumberTreeNode(n.valueType)
		}
		pdObjects = append(pdObjects, childNode)
	}
	return NewCOSArrayListOf(pdObjects, kids)
}

// SetKids sets the child nodes.
func (n *PDNumberTreeNode) SetKids(kids []*PDNumberTreeNode) {
	if len(kids) > 0 {
		firstKid := kids[0]
		lastKid := kids[len(kids)-1]
		lowerLimit := firstKid.LowerLimit()
		n.setLowerLimit(lowerLimit)
		upperLimit := lastKid.UpperLimit()
		n.setUpperLimit(upperLimit)
		array := cos.NewArray()
		for _, kid := range kids {
			array.Add(kid.COSObject())
		}
		n.node.SetItem(cos.Kids, array)
	} else if n.node.GetDictionaryObject(cos.Nums) == nil {
		// Remove limits if there are no kids and no numbers set.
		n.node.SetItem(cos.Limits, nil)
		n.node.SetItem(cos.Kids, nil)
	}
}

// Value looks an index up in this node and its children.
//
// Java declares the return as Object, and every value it can hold is a
// COSObjectable; the port says so.
func (n *PDNumberTreeNode) Value(index int) (COSObjectable, error) {
	numbers, err := n.Numbers()
	if err != nil {
		return nil, err
	}
	if numbers != nil {
		return numbers[index], nil
	}
	var retval COSObjectable
	kids := n.Kids()
	if kids == nil {
		slog.Warn("pdmodel: NumberTreeNode does not have \"nums\" nor \"kids\" objects.")
		return nil, nil
	}
	for i := 0; i < kids.Size() && retval == nil; i++ {
		childNode := kids.Get(i)
		lower := childNode.LowerLimit()
		upper := childNode.UpperLimit()
		// Java dereferences both without a null check, so a node whose /Limits
		// is missing throws NullPointerException here; the port panics.
		if lower == nil || upper == nil {
			panic("pdmodel: number tree kid has no /Limits")
		}
		if *lower <= index && *upper >= index {
			if retval, err = childNode.Value(index); err != nil {
				return nil, err
			}
		}
	}
	return retval, nil
}

// Numbers returns the index to value mapping of this node, or nil where the
// node has no /Nums.
func (n *PDNumberTreeNode) Numbers() (map[int]COSObjectable, error) {
	numbersArray := n.node.GetCOSArray(cos.Nums)
	if numbersArray == nil {
		return nil, nil
	}
	size := numbersArray.Size()
	indices := map[int]COSObjectable{}
	if size%2 != 0 {
		slog.Warn("pdmodel: Numbers array has odd size", "size", size)
	}
	for i := 0; i+1 < size; i += 2 {
		base := numbersArray.GetObject(i)
		key, ok := base.(*cos.Integer)
		if !ok {
			slog.Error("pdmodel: page labels ignored, index should be a number",
				"index", i, "value", base)
			return nil, nil
		}
		cosValue := numbersArray.GetObject(i + 1)
		if cosValue == nil {
			indices[int(key.IntValue())] = nil
			continue
		}
		value, err := n.ConvertCOSToPD(cosValue)
		if err != nil {
			return nil, err
		}
		indices[int(key.IntValue())] = value
	}
	return indices, nil
}

// ConvertCOSToPD builds the PD form of a value.
func (n *PDNumberTreeNode) ConvertCOSToPD(base cos.Base) (COSObjectable, error) {
	// valueType (passed in constructor here) must have a constructor of type of COSBase as parameter
	return n.valueType(base)
}

// CreateChildNode returns a node over the given dictionary, with this node's
// value converter.
func (n *PDNumberTreeNode) CreateChildNode(dic *cos.Dictionary) *PDNumberTreeNode {
	return NewPDNumberTreeNodeOf(dic, n.valueType)
}

// SetNumbers sets the index to value mapping of this node.
//
// A nil map is Java's null, which clears the entry.
func (n *PDNumberTreeNode) SetNumbers(numbers map[int]COSObjectable) {
	if numbers == nil {
		n.node.SetItem(cos.Nums, nil)
		n.node.SetItem(cos.Limits, nil)
		return
	}
	keys := sortedKeys(numbers)
	array := cos.NewArray()
	for _, key := range keys {
		array.Add(cos.GetInteger(int64(key)))
		obj := numbers[key]
		if obj == nil {
			array.Add(cos.NullObject)
		} else {
			array.Add(obj.COSObject())
		}
	}
	var lower, upper *int
	if len(keys) > 0 {
		lower = &keys[0]
		upper = &keys[len(keys)-1]
	}
	n.setUpperLimit(upper)
	n.setLowerLimit(lower)
	n.node.SetItem(cos.Nums, array)
}

// UpperLimit returns the highest index in this node's subtree, or nil where the
// node has no /Limits.
func (n *PDNumberTreeNode) UpperLimit() *int {
	arr := n.node.GetCOSArray(cos.Limits)
	if arr != nil && arr.Get(1) != nil {
		value := arr.GetInt(1)
		return &value
	}
	return nil
}

func (n *PDNumberTreeNode) setUpperLimit(upper *int) {
	arr := n.node.GetCOSArray(cos.Limits)
	if arr == nil {
		arr = cos.NewArray()
		arr.Add(nil)
		arr.Add(nil)
		n.node.SetItem(cos.Limits, arr)
	}
	if upper != nil {
		arr.SetInt(1, *upper)
	} else {
		arr.Set(1, nil)
	}
}

// LowerLimit returns the lowest index in this node's subtree, or nil.
func (n *PDNumberTreeNode) LowerLimit() *int {
	arr := n.node.GetCOSArray(cos.Limits)
	if arr != nil && arr.Get(0) != nil {
		value := arr.GetInt(0)
		return &value
	}
	return nil
}

func (n *PDNumberTreeNode) setLowerLimit(lower *int) {
	arr := n.node.GetCOSArray(cos.Limits)
	if arr == nil {
		arr = cos.NewArray()
		arr.Add(nil)
		arr.Add(nil)
		n.node.SetItem(cos.Limits, arr)
	}
	if lower != nil {
		arr.SetInt(0, *lower)
	} else {
		arr.Set(0, nil)
	}
}
