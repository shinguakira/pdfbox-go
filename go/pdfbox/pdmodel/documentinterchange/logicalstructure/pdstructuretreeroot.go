package logicalstructure

import (
	"log/slog"
	"maps"
	"slices"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// typeStructTreeRoot is the /Type of the structure tree root. Java declares it
// private.
const typeStructTreeRoot = "StructTreeRoot"

// PDStructureTreeRoot is the root of the structure tree.
//
// Port of PDStructureTreeRoot.
type PDStructureTreeRoot struct {
	PDStructureNode
}

var _ StructureNode = (*PDStructureTreeRoot)(nil)

// NewPDStructureTreeRoot builds an empty structure tree root.
func NewPDStructureTreeRoot() *PDStructureTreeRoot {
	r := &PDStructureTreeRoot{}
	r.InitStructureNode(r, typeStructTreeRoot)
	return r
}

// NewPDStructureTreeRootOf builds one over the given dictionary.
func NewPDStructureTreeRootOf(dic *cos.Dictionary) *PDStructureTreeRoot {
	r := &PDStructureTreeRoot{}
	r.InitStructureNodeOf(r, dic)
	return r
}

// K returns the /K entry, which is one child or an array of them.
func (r *PDStructureTreeRoot) K() cos.Base {
	return r.NodeDictionary().GetDictionaryObject(cos.K)
}

// SetK sets the /K entry.
func (r *PDStructureTreeRoot) SetK(k cos.Base) {
	r.NodeDictionary().SetItem(cos.K, k)
}

// IDTree returns the /IDTree name tree, or nil.
func (r *PDStructureTreeRoot) IDTree() common.NameTreeNode[*PDStructureElement] {
	if idTree := r.NodeDictionary().GetCOSDictionary(cos.IDTree); idTree != nil {
		return NewStructureElementNameTreeNode(idTree)
	}
	return nil
}

// SetIDTree sets the /IDTree name tree.
func (r *PDStructureTreeRoot) SetIDTree(idTree common.NameTreeNode[*PDStructureElement]) {
	if idTree == nil {
		r.NodeDictionary().SetItem(cos.IDTree, nil)
		return
	}
	r.NodeDictionary().SetItem(cos.IDTree, idTree.COSObject())
}

// ParentTree returns the /ParentTree number tree, or nil.
func (r *PDStructureTreeRoot) ParentTree() *common.PDNumberTreeNode {
	if parentTree := r.NodeDictionary().GetCOSDictionary(cos.ParentTree); parentTree != nil {
		return common.NewPDNumberTreeNodeOf(parentTree, ParentTreeValueConverter)
	}
	return nil
}

// SetParentTree sets the /ParentTree number tree.
func (r *PDStructureTreeRoot) SetParentTree(parentTree *common.PDNumberTreeNode) {
	if parentTree == nil {
		r.NodeDictionary().SetItem(cos.ParentTree, nil)
		return
	}
	r.NodeDictionary().SetItem(cos.ParentTree, parentTree.COSObject())
}

// ParentTreeNextKey returns the /ParentTreeNextKey.
func (r *PDStructureTreeRoot) ParentTreeNextKey() int {
	return r.NodeDictionary().GetInt(cos.ParentTreeNextKey)
}

// SetParentTreeNextKey sets the /ParentTreeNextKey.
func (r *PDStructureTreeRoot) SetParentTreeNextKey(parentTreeNextkey int) {
	r.NodeDictionary().SetInt(cos.ParentTreeNextKey, parentTreeNextkey)
}

// RoleMap returns the /RoleMap, which maps a structure type onto a standard
// one. It is empty where there is no role map, and where reading it failed.
func (r *PDStructureTreeRoot) RoleMap() *common.COSDictionaryMap[any] {
	if rm := r.NodeDictionary().GetCOSDictionary(cos.RoleMap); rm != nil {
		roleMap, err := common.ConvertBasicTypesToMap(rm)
		if err != nil {
			slog.Error("logicalstructure: could not read the role map", slog.Any("error", err))
		} else {
			return roleMap
		}
	}
	return common.NewCOSDictionaryMap(map[string]any{}, cos.NewDictionary())
}

// SetRoleMap sets the /RoleMap.
//
// Java walks a HashMap, whose order is arbitrary but fixed; a Go map has no
// order at all, so the port writes the names sorted, to keep one input giving
// one file.
func (r *PDStructureTreeRoot) SetRoleMap(roleMap map[string]string) {
	rmDic := cos.NewDictionary()
	for _, name := range slices.Sorted(maps.Keys(roleMap)) {
		rmDic.SetName(cos.GetPDFName(name), roleMap[name])
	}
	r.NodeDictionary().SetItem(cos.RoleMap, rmDic)
}

// ClassMap returns the /ClassMap, which maps a class name onto one attribute
// object or a list of them. It is empty where there is no class map.
func (r *PDStructureTreeRoot) ClassMap() map[string]any {
	classMap := map[string]any{}
	classMapDictionary := r.NodeDictionary().GetCOSDictionary(cos.ClassMap)
	if classMapDictionary == nil {
		return classMap
	}
	for _, name := range classMapDictionary.KeySet() {
		base := classMapDictionary.GetItem(name)
		if reference, isReference := base.(*cos.Object); isReference {
			base = reference.Object()
		}
		if dictionary, isDictionary := asDictionary(base); isDictionary {
			classMap[name.Name()] = CreateAttributeObject(dictionary)
			continue
		}
		if array, isArray := base.(*cos.Array); isArray {
			list := []PDAttributeObject{}
			for i := 0; i < array.Size(); i++ {
				if dictionary, isDictionary := asDictionary(array.GetObject(i)); isDictionary {
					list = append(list, CreateAttributeObject(dictionary))
				}
			}
			classMap[name.Name()] = list
		}
	}
	return classMap
}

// SetClassMap sets the /ClassMap, and removes it where the map is empty.
//
// The values are one attribute object or a list of them; anything else is
// dropped, which is what Java's two instanceof tests do.
//
// Java walks a HashMap, whose order is arbitrary but fixed; the port writes the
// names sorted, to keep one input giving one file.
func (r *PDStructureTreeRoot) SetClassMap(classMap map[string]any) {
	if len(classMap) == 0 {
		r.NodeDictionary().RemoveItem(cos.ClassMap)
		return
	}
	classMapDictionary := cos.NewDictionary()
	for _, name := range slices.Sorted(maps.Keys(classMap)) {
		switch object := classMap[name].(type) {
		case PDAttributeObject:
			classMapDictionary.SetItem(cos.GetPDFName(name), object.COSObject())
		case []PDAttributeObject:
			array := cos.NewArray()
			for _, attributeObject := range object {
				array.Add(attributeObject.COSObject())
			}
			classMapDictionary.SetItem(cos.GetPDFName(name), array)
		}
	}
	r.NodeDictionary().SetItem(cos.ClassMap, classMapDictionary)
}
