package logicalstructure

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// StructureNode is a node of the structure tree: the tree root, or an element
// of it.
//
// Java's PDStructureNode is an abstract class; the port splits it into this
// interface for the contract and the embedded struct below for the state, per
// migration/conventions/java-to-go.md.
type StructureNode interface {
	common.COSObjectable

	// NodeDictionary returns the dictionary, which getCOSObject narrows to in
	// Java.
	NodeDictionary() *cos.Dictionary

	// Type returns the /Type of this node.
	Type() string
}

// PDStructureNode carries the state and the concrete methods of a structure
// node.
//
// Port of PDStructureNode.
type PDStructureNode struct {
	self       StructureNode
	dictionary *cos.Dictionary
}

var _ common.COSObjectable = (*PDStructureNode)(nil)

// InitStructureNode is the protected PDStructureNode(String) constructor. A
// concrete node calls it from its own constructor with itself as self, since Go
// embedding does not dispatch.
func (n *PDStructureNode) InitStructureNode(self StructureNode, nodeType string) {
	n.self = self
	n.dictionary = cos.NewDictionary()
	n.dictionary.SetName(cos.Type, nodeType)
}

// InitStructureNodeOf is the protected PDStructureNode(COSDictionary)
// constructor.
func (n *PDStructureNode) InitStructureNodeOf(self StructureNode, dictionary *cos.Dictionary) {
	n.self = self
	n.dictionary = dictionary
}

// Create builds the structure node the given dictionary describes.
//
// Java throws IllegalArgumentException for a /Type that is neither of the two,
// which is unchecked, so the port panics.
func Create(node *cos.Dictionary) StructureNode {
	nodeType := node.GetNameAsString(cos.Type, "")
	if nodeType == typeStructTreeRoot {
		return NewPDStructureTreeRootOf(node)
	}
	if nodeType == "" || nodeType == TypeStructElem {
		return NewPDStructureElementOf(node)
	}
	panic("Dictionary must not include a Type entry with a value that is neither StructTreeRoot nor StructElem.")
}

// COSObject returns the dictionary of this node.
func (n *PDStructureNode) COSObject() cos.Base { return n.dictionary }

// NodeDictionary returns the dictionary of this node, typed.
func (n *PDStructureNode) NodeDictionary() *cos.Dictionary { return n.dictionary }

// Type returns the /Type of this node.
func (n *PDStructureNode) Type() string {
	return n.dictionary.GetNameAsString(cos.Type, "")
}

// Kids returns the children of this node.
//
// Java's element type is Object: a structure element, an object reference, a
// marked content reference, or an int marked-content identifier. The port uses
// any for the same reason.
func (n *PDStructureNode) Kids() []any {
	kidObjects := []any{}
	k := n.dictionary.GetDictionaryObject(cos.K)
	if array, isArray := k.(*cos.Array); isArray {
		for i := 0; i < array.Size(); i++ {
			if kidObject := n.CreateObject(array.Get(i)); kidObject != nil {
				kidObjects = append(kidObjects, kidObject)
			}
		}
		return kidObjects
	}
	if kidObject := n.CreateObject(k); kidObject != nil {
		kidObjects = append(kidObjects, kidObject)
	}
	return kidObjects
}

// SetKids sets the children of this node.
func (n *PDStructureNode) SetKids(kids []any) {
	n.dictionary.SetItem(cos.K, common.ConverterToCOSArray(kids))
}

// AppendKid appends a structure element as a child, and makes this node its
// parent.
func (n *PDStructureNode) AppendKid(structureElement *PDStructureElement) {
	n.AppendObjectableKid(structureElement)
	structureElement.SetParent(n.self)
}

// AppendObjectableKid appends an objectable child. Java declares it protected.
func (n *PDStructureNode) AppendObjectableKid(objectable common.COSObjectable) {
	if objectable == nil {
		return
	}
	n.AppendKidBase(objectable.COSObject())
}

// AppendKidBase appends a COS object as a child, which is Java's protected
// appendKid(COSBase).
func (n *PDStructureNode) AppendKidBase(object cos.Base) {
	if object == nil {
		return
	}
	k := n.dictionary.GetDictionaryObject(cos.K)
	switch current := k.(type) {
	case nil:
		// currently no kid: set new kid as kids
		n.dictionary.SetItem(cos.K, object)
	case *cos.Array:
		// currently more than one kid: add new kid to existing array
		current.Add(object)
	default:
		// currently one kid: put current and new kid into array and set array as kids
		array := cos.NewArrayOf([]cos.Base{k, object})
		n.dictionary.SetItem(cos.K, array)
	}
}

// InsertBefore inserts a structure element before the given child.
func (n *PDStructureNode) InsertBefore(newKid *PDStructureElement, refKid any) {
	n.InsertObjectableBefore(newKid, refKid)
}

// InsertObjectableBefore inserts an objectable child before the given one.
// Java declares it protected.
func (n *PDStructureNode) InsertObjectableBefore(newKid common.COSObjectable, refKid any) {
	if newKid == nil {
		return
	}
	n.InsertBeforeBase(newKid.COSObject(), refKid)
}

// InsertBeforeBase inserts a COS object before the given child, which is Java's
// protected insertBefore(COSBase, Object).
//
// JAVA BUG: a refKid the /K array does not hold, or a marked-content identifier
// as getKids returns it, gives an index of -1, and inserting at -1 throws
// instead of doing nothing. See migration/JAVA-BUGS.md entry 39. The port
// keeps it: AddAt at -1 panics.
func (n *PDStructureNode) InsertBeforeBase(newKid cos.Base, refKid any) {
	if newKid == nil || refKid == nil {
		return
	}
	k := n.dictionary.GetDictionaryObject(cos.K)
	if k == nil {
		return
	}
	var refKidBase cos.Base
	if objectable, isObjectable := refKid.(common.COSObjectable); isObjectable {
		refKidBase = objectable.COSObject()
	}
	if array, isArray := k.(*cos.Array); isArray {
		refIndex := array.IndexOfObject(refKidBase)
		array.AddAt(refIndex, newKid.COSObject())
		return
	}
	onlyKid := cos.Equal(k, refKidBase)
	if !onlyKid {
		if reference, isReference := k.(*cos.Object); isReference {
			onlyKid = cos.Equal(reference.Object(), refKidBase)
		}
	}
	if onlyKid {
		array := cos.NewArrayOf([]cos.Base{newKid, refKidBase})
		n.dictionary.SetItem(cos.K, array)
	}
}

// RemoveKid removes a structure element child, reporting whether it was there,
// and clears its parent when it was.
func (n *PDStructureNode) RemoveKid(structureElement *PDStructureElement) bool {
	removed := n.RemoveObjectableKid(structureElement)
	if removed {
		structureElement.SetParent(nil)
	}
	return removed
}

// RemoveObjectableKid removes an objectable child. Java declares it protected.
func (n *PDStructureNode) RemoveObjectableKid(objectable common.COSObjectable) bool {
	if objectable == nil {
		return false
	}
	return n.RemoveKidBase(objectable.COSObject())
}

// RemoveKidBase removes a COS object child, which is Java's protected
// removeKid(COSBase).
func (n *PDStructureNode) RemoveKidBase(object cos.Base) bool {
	if object == nil {
		return false
	}
	k := n.dictionary.GetDictionaryObject(cos.K)
	switch current := k.(type) {
	case nil:
		// no kids: objectable is not a kid
		return false
	case *cos.Array:
		// currently more than one kid: remove kid from existing array
		removed := current.RemoveObject(object)
		// if now only one kid: set remaining kid as kids
		if current.Size() == 1 {
			n.dictionary.SetItem(cos.K, current.GetObject(0))
		}
		return removed
	default:
		// currently one kid: if current kid equals given object, remove kids entry
		onlyKid := cos.Equal(k, object)
		if !onlyKid {
			if reference, isReference := k.(*cos.Object); isReference {
				onlyKid = cos.Equal(reference.Object(), object)
			}
		}
		if onlyKid {
			n.dictionary.SetItem(cos.K, nil)
			return true
		}
		return false
	}
}

// CreateObject builds the child a /K entry describes: a structure element, an
// object reference, a marked content reference, an int marked-content
// identifier, or nil where it is none of them. Java declares it protected.
func (n *PDStructureNode) CreateObject(kid cos.Base) any {
	var kidDic *cos.Dictionary
	switch value := kid.(type) {
	case *cos.Dictionary:
		kidDic = value
	case *cos.Object:
		if base, isDictionary := value.Object().(*cos.Dictionary); isDictionary {
			kidDic = base
		}
	}
	if kidDic != nil {
		return createObjectFromDic(kidDic)
	}
	if mcid, isInteger := kid.(*cos.Integer); isInteger {
		// An integer marked-content identifier denoting a marked-content sequence
		return mcid.IntValue()
	}
	return nil
}

// createObjectFromDic builds the child a /K dictionary describes.
func createObjectFromDic(kidDic *cos.Dictionary) common.COSObjectable {
	switch kidDic.GetNameAsString(cos.Type, "") {
	case "", TypeStructElem:
		// A structure element dictionary denoting another structure element
		return NewPDStructureElementOf(kidDic)
	case TypeObjectReference:
		// An object reference dictionary denoting a PDF object
		return NewPDObjectReferenceOf(kidDic)
	case TypeMarkedContentReference:
		// A marked-content reference dictionary denoting a marked-content sequence
		return NewPDMarkedContentReferenceOf(kidDic)
	default:
		return nil
	}
}
