package logicalstructure

import (
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/documentinterchange/markedcontent"
)

// TypeStructElem is the /Type of a structure element.
//
// Port of PDStructureElement.TYPE.
const TypeStructElem = "StructElem"

// PDStructureElement is one element of the structure tree.
//
// Port of PDStructureElement.
type PDStructureElement struct {
	PDStructureNode
}

var _ StructureNode = (*PDStructureElement)(nil)

// NewPDStructureElement builds an element of the given structure type below the
// given parent.
func NewPDStructureElement(structureType string, parent StructureNode) *PDStructureElement {
	e := &PDStructureElement{}
	e.InitStructureNode(e, TypeStructElem)
	e.SetStructureType(structureType)
	e.SetParent(parent)
	return e
}

// NewPDStructureElementOf builds one over the given dictionary.
func NewPDStructureElementOf(dic *cos.Dictionary) *PDStructureElement {
	e := &PDStructureElement{}
	e.InitStructureNodeOf(e, dic)
	return e
}

// StructureType returns the /S structure type.
func (e *PDStructureElement) StructureType() string {
	return e.NodeDictionary().GetNameAsString(cos.S, "")
}

// SetStructureType sets the /S structure type. Java declares it final, because
// the constructor calls it.
func (e *PDStructureElement) SetStructureType(structureType string) {
	e.NodeDictionary().SetName(cos.S, structureType)
}

// Parent returns the /P parent node, or nil.
func (e *PDStructureElement) Parent() StructureNode {
	if parent := e.NodeDictionary().GetCOSDictionary(cos.P); parent != nil {
		return Create(parent)
	}
	return nil
}

// SetParent sets the /P parent node. Java declares it final, because the
// constructor calls it.
func (e *PDStructureElement) SetParent(structureNode StructureNode) {
	if structureNode == nil {
		e.NodeDictionary().SetItem(cos.P, nil)
		return
	}
	e.NodeDictionary().SetItem(cos.P, structureNode.COSObject())
}

// ElementIdentifier returns the /ID of this element.
func (e *PDStructureElement) ElementIdentifier() string {
	return e.NodeDictionary().GetString(cos.ID, "")
}

// SetElementIdentifier sets the /ID of this element.
func (e *PDStructureElement) SetElementIdentifier(id string) {
	e.NodeDictionary().SetString(cos.ID, id)
}

// Page returns the /Pg page this element is on, or nil.
func (e *PDStructureElement) Page() PageLike {
	if page := e.NodeDictionary().GetCOSDictionary(cos.Pg); page != nil {
		return NewPageFromDictionary(page)
	}
	return nil
}

// SetPage sets the /Pg page this element is on.
func (e *PDStructureElement) SetPage(page PageLike) {
	if page == nil {
		e.NodeDictionary().SetItem(cos.Pg, nil)
		return
	}
	e.NodeDictionary().SetItem(cos.Pg, page.COSObject())
}

// Attributes returns the /A attribute objects with their revision numbers.
func (e *PDStructureElement) Attributes() *Revisions[PDAttributeObject] {
	attributes := NewRevisions[PDAttributeObject]()
	a := e.NodeDictionary().GetDictionaryObject(cos.A)
	if array, isArray := a.(*cos.Array); isArray {
		var ao PDAttributeObject
		for i := 0; i < array.Size(); i++ {
			item := array.GetObject(i)
			if dictionary, isDictionary := asDictionary(item); isDictionary {
				ao = CreateAttributeObject(dictionary)
				ao.SetStructureElement(e)
				attributes.AddObject(ao, 0)
			} else if revision, isInteger := item.(*cos.Integer); isInteger {
				// Read "14.7.5.3 Attribute Revision Numbers"
				// This is additional to the /R entry
				attributes.SetRevisionNumber(ao, revision.IntValue())
			}
		}
	}
	if dictionary, isDictionary := asDictionary(a); isDictionary {
		ao := CreateAttributeObject(dictionary)
		ao.SetStructureElement(e)
		attributes.AddObject(ao, 0)
	}
	return attributes
}

// SetAttributes sets the /A attribute objects with their revision numbers.
//
// Java throws IllegalArgumentException for a negative revision number, which is
// unchecked, so the port panics.
func (e *PDStructureElement) SetAttributes(attributes *Revisions[PDAttributeObject]) {
	key := cos.A
	if attributes.Size() == 1 && attributes.RevisionNumber(0) == 0 {
		attributeObject := attributes.Object(0)
		attributeObject.SetStructureElement(e)
		e.NodeDictionary().SetItem(key, attributeObject.COSObject())
		return
	}
	array := cos.NewArray()
	for i := 0; i < attributes.Size(); i++ {
		attributeObject := attributes.Object(i)
		attributeObject.SetStructureElement(e)
		revisionNumber := attributes.RevisionNumber(i)
		if revisionNumber < 0 {
			panic("The revision number shall be > -1")
		}
		array.Add(attributeObject.COSObject())
		array.Add(cos.GetInteger(int64(revisionNumber)))
	}
	e.NodeDictionary().SetItem(key, array)
}

// AddAttribute adds one attribute object at this element's revision number.
func (e *PDStructureElement) AddAttribute(attributeObject PDAttributeObject) {
	key := cos.A
	attributeObject.SetStructureElement(e)
	a := e.NodeDictionary().GetDictionaryObject(key)
	array, isArray := a.(*cos.Array)
	if !isArray {
		array = cos.NewArray()
		if a != nil {
			array.Add(a)
			array.Add(cos.GetInteger(0))
		}
	}
	e.NodeDictionary().SetItem(key, array)
	array.Add(attributeObject.COSObject())
	array.Add(cos.GetInteger(int64(e.RevisionNumber())))
}

// RemoveAttribute removes one attribute object.
func (e *PDStructureElement) RemoveAttribute(attributeObject PDAttributeObject) {
	key := cos.A
	a := e.NodeDictionary().GetDictionaryObject(key)
	if array, isArray := a.(*cos.Array); isArray {
		array.Remove(attributeObject.COSObject())
		if array.Size() == 2 && array.GetInt(1) == 0 {
			e.NodeDictionary().SetItem(key, array.GetObject(0))
		}
	} else {
		directA := a
		if reference, isReference := a.(*cos.Object); isReference {
			directA = reference.Object()
		}
		if cos.Equal(attributeObject.COSObject(), directA) {
			e.NodeDictionary().SetItem(key, nil)
		}
	}
	attributeObject.SetStructureElement(nil)
}

// AttributeChanged updates the revision number of an attribute object that has
// changed.
func (e *PDStructureElement) AttributeChanged(attributeObject PDAttributeObject) {
	key := cos.A
	a := e.NodeDictionary().GetDictionaryObject(key)
	if array, isArray := a.(*cos.Array); isArray {
		for i := 0; i < array.Size(); i++ {
			entry := array.GetObject(i)
			if cos.Equal(entry, attributeObject.COSObject()) {
				next := array.Get(i + 1)
				if _, isInteger := next.(*cos.Integer); isInteger {
					array.Set(i+1, cos.GetInteger(int64(e.RevisionNumber())))
				}
			}
		}
		return
	}
	array := cos.NewArrayOf([]cos.Base{a, cos.GetInteger(int64(e.RevisionNumber()))})
	e.NodeDictionary().SetItem(key, array)
}

// ClassNames returns the /C class names with their revision numbers.
func (e *PDStructureElement) ClassNames() *Revisions[string] {
	key := cos.C
	classNames := NewRevisions[string]()
	c := e.NodeDictionary().GetDictionaryObject(key)
	if name, isName := c.(*cos.Name); isName {
		classNames.AddObject(name.Name(), 0)
	}
	if array, isArray := c.(*cos.Array); isArray {
		// Java starts the class name at null, so a revision number before the
		// first name has nothing to attach to; the port starts it empty, which
		// only differs for the empty name.
		className := ""
		for i := 0; i < array.Size(); i++ {
			item := array.GetObject(i)
			if name, isName := item.(*cos.Name); isName {
				className = name.Name()
				classNames.AddObject(className, 0)
			} else if revision, isInteger := item.(*cos.Integer); isInteger {
				// Read "14.7.5.3 Attribute Revision Numbers"
				// This is additional to the /R entry
				classNames.SetRevisionNumber(className, revision.IntValue())
			}
		}
	}
	return classNames
}

// SetClassNames sets the /C class names with their revision numbers.
//
// Java throws IllegalArgumentException for a negative revision number, which is
// unchecked, so the port panics.
func (e *PDStructureElement) SetClassNames(classNames *Revisions[string]) {
	if classNames == nil {
		return
	}
	key := cos.C
	if classNames.Size() == 1 && classNames.RevisionNumber(0) == 0 {
		e.NodeDictionary().SetName(key, classNames.Object(0))
		return
	}
	array := cos.NewArray()
	for i := 0; i < classNames.Size(); i++ {
		className := classNames.Object(i)
		revisionNumber := classNames.RevisionNumber(i)
		if revisionNumber < 0 {
			panic("The revision number shall be > -1")
		}
		array.Add(cos.GetPDFName(className))
		array.Add(cos.GetInteger(int64(revisionNumber)))
	}
	e.NodeDictionary().SetItem(key, array)
}

// AddClassName adds one class name at this element's revision number.
//
// An empty name is Java's null, which it ignores.
func (e *PDStructureElement) AddClassName(className string) {
	if className == "" {
		return
	}
	key := cos.C
	c := e.NodeDictionary().GetDictionaryObject(key)
	array, isArray := c.(*cos.Array)
	if !isArray {
		array = cos.NewArray()
		if c != nil {
			array.Add(c)
			array.Add(cos.GetInteger(0))
		}
	}
	e.NodeDictionary().SetItem(key, array)
	array.Add(cos.GetPDFName(className))
	array.Add(cos.GetInteger(int64(e.RevisionNumber())))
}

// RemoveClassName removes one class name.
//
// An empty name is Java's null, which it ignores.
func (e *PDStructureElement) RemoveClassName(className string) {
	if className == "" {
		return
	}
	key := cos.C
	c := e.NodeDictionary().GetDictionaryObject(key)
	name := cos.GetPDFName(className)
	if array, isArray := c.(*cos.Array); isArray {
		array.Remove(name)
		if array.Size() == 2 && array.GetInt(1) == 0 {
			e.NodeDictionary().SetItem(key, array.GetObject(0))
		}
		return
	}
	directC := c
	if reference, isReference := c.(*cos.Object); isReference {
		directC = reference.Object()
	}
	if cos.Equal(name, directC) {
		e.NodeDictionary().SetItem(key, nil)
	}
}

// RevisionNumber returns the /R revision number of this element.
func (e *PDStructureElement) RevisionNumber() int {
	return e.NodeDictionary().GetIntDefault(cos.R, 0)
}

// SetRevisionNumber sets the /R revision number of this element.
//
// Java throws IllegalArgumentException for a negative one, which is unchecked,
// so the port panics.
func (e *PDStructureElement) SetRevisionNumber(revisionNumber int) {
	if revisionNumber < 0 {
		panic("The revision number shall be > -1")
	}
	e.NodeDictionary().SetInt(cos.R, revisionNumber)
}

// IncrementRevisionNumber raises the revision number by one.
func (e *PDStructureElement) IncrementRevisionNumber() {
	e.SetRevisionNumber(e.RevisionNumber() + 1)
}

// Title returns the /T title of this element.
func (e *PDStructureElement) Title() string {
	return e.NodeDictionary().GetString(cos.T, "")
}

// SetTitle sets the /T title of this element.
func (e *PDStructureElement) SetTitle(title string) {
	e.NodeDictionary().SetString(cos.T, title)
}

// Language returns the /Lang language of this element.
func (e *PDStructureElement) Language() string {
	return e.NodeDictionary().GetString(cos.Lang, "")
}

// SetLanguage sets the /Lang language of this element.
func (e *PDStructureElement) SetLanguage(language string) {
	e.NodeDictionary().SetString(cos.Lang, language)
}

// AlternateDescription returns the /Alt description of this element.
func (e *PDStructureElement) AlternateDescription() string {
	return e.NodeDictionary().GetString(cos.Alt, "")
}

// SetAlternateDescription sets the /Alt description of this element.
func (e *PDStructureElement) SetAlternateDescription(alternateDescription string) {
	e.NodeDictionary().SetString(cos.Alt, alternateDescription)
}

// ExpandedForm returns the /E expanded form of this element.
func (e *PDStructureElement) ExpandedForm() string {
	return e.NodeDictionary().GetString(cos.E, "")
}

// SetExpandedForm sets the /E expanded form of this element.
func (e *PDStructureElement) SetExpandedForm(expandedForm string) {
	e.NodeDictionary().SetString(cos.E, expandedForm)
}

// ActualText returns the /ActualText of this element.
func (e *PDStructureElement) ActualText() string {
	return e.NodeDictionary().GetString(cos.ActualText, "")
}

// SetActualText sets the /ActualText of this element.
func (e *PDStructureElement) SetActualText(actualText string) {
	e.NodeDictionary().SetString(cos.ActualText, actualText)
}

// StandardStructureType returns the standard structure type this element's
// structure type maps to through the role map, or the structure type itself.
func (e *PDStructureElement) StandardStructureType() string {
	structureType := e.StructureType()
	roleMap := e.roleMap()
	if roleMap == nil {
		return structureType
	}
	if mappedValue, found := roleMap.Get(structureType); found {
		if mapped, isString := mappedValue.(string); isString {
			structureType = mapped
		}
	}
	return structureType
}

// AppendKidMCID appends a marked-content identifier as a child.
//
// Java throws IllegalArgumentException for a negative one, which is unchecked,
// so the port panics.
func (e *PDStructureElement) AppendKidMCID(mcid int) {
	if mcid < 0 {
		panic("MCID should not be negative")
	}
	e.AppendKidBase(cos.GetInteger(int64(mcid)))
}

// AppendKidMarkedContent appends the marked-content identifier of the given
// marked content as a child.
//
// Java throws IllegalArgumentException where there is no identifier, which is
// unchecked, so the port panics.
func (e *PDStructureElement) AppendKidMarkedContent(markedContent *markedcontent.PDMarkedContent) {
	if markedContent == nil {
		return
	}
	mcid := markedContent.MCID()
	if mcid < 0 {
		panic("MCID is negative or doesn't exist")
	}
	e.AppendKidBase(cos.GetInteger(int64(mcid)))
}

// AppendKidMarkedContentReference appends a marked content reference as a
// child.
func (e *PDStructureElement) AppendKidMarkedContentReference(
	markedContentReference *PDMarkedContentReference) {
	e.AppendObjectableKid(markedContentReference)
}

// AppendKidObjectReference appends an object reference as a child.
func (e *PDStructureElement) AppendKidObjectReference(objectReference *PDObjectReference) {
	e.AppendObjectableKid(objectReference)
}

// InsertBeforeMCID inserts a marked-content identifier before the given child.
func (e *PDStructureElement) InsertBeforeMCID(markedContentIdentifier *cos.Integer, refKid any) {
	e.InsertBeforeBase(markedContentIdentifier, refKid)
}

// InsertBeforeMarkedContentReference inserts a marked content reference before
// the given child.
func (e *PDStructureElement) InsertBeforeMarkedContentReference(
	markedContentReference *PDMarkedContentReference, refKid any) {
	e.InsertObjectableBefore(markedContentReference, refKid)
}

// InsertBeforeObjectReference inserts an object reference before the given
// child.
func (e *PDStructureElement) InsertBeforeObjectReference(
	objectReference *PDObjectReference, refKid any) {
	e.InsertObjectableBefore(objectReference, refKid)
}

// RemoveKidMCID removes a marked-content identifier child.
func (e *PDStructureElement) RemoveKidMCID(markedContentIdentifier *cos.Integer) {
	e.RemoveKidBase(markedContentIdentifier)
}

// RemoveKidMarkedContentReference removes a marked content reference child.
func (e *PDStructureElement) RemoveKidMarkedContentReference(
	markedContentReference *PDMarkedContentReference) {
	e.RemoveObjectableKid(markedContentReference)
}

// RemoveKidObjectReference removes an object reference child.
func (e *PDStructureElement) RemoveKidObjectReference(objectReference *PDObjectReference) {
	e.RemoveObjectableKid(objectReference)
}

// structureTreeRoot walks up the parents to the structure tree root, or nil
// where the chain does not reach one.
func (e *PDStructureElement) structureTreeRoot() *PDStructureTreeRoot {
	visited := map[*cos.Dictionary]bool{}
	parent := e.Parent()
	for {
		element, isElement := parent.(*PDStructureElement)
		if !isElement {
			break
		}
		if visited[element.NodeDictionary()] {
			slog.Warn("logicalstructure: element ignored",
				slog.Any("element", element.NodeDictionary()))
			return nil // Cycle detected
		}
		visited[element.NodeDictionary()] = true
		parent = element.Parent()
	}
	if root, isRoot := parent.(*PDStructureTreeRoot); isRoot {
		return root
	}
	return nil
}

// roleMap returns the role map of the structure tree root, or nil where there
// is none. Java returns an empty map there.
func (e *PDStructureElement) roleMap() *common.COSDictionaryMap[any] {
	if root := e.structureTreeRoot(); root != nil {
		return root.RoleMap()
	}
	return nil
}
