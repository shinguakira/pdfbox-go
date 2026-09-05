package form

import (
	"fmt"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
)

// PDNonTerminalField is a field that gathers other fields.
//
// Port of PDNonTerminalField.
type PDNonTerminalField struct {
	PDFieldBase
}

var _ PDField = (*PDNonTerminalField)(nil)

// NewPDNonTerminalField creates a field in the given form.
func NewPDNonTerminalField(acroForm *PDAcroForm) *PDNonTerminalField {
	f := &PDNonTerminalField{}
	f.initField(f, acroForm)
	return f
}

// NewPDNonTerminalFieldOf creates one over the given dictionary. Java declares
// the constructor package-private.
func NewPDNonTerminalFieldOf(acroForm *PDAcroForm, field *cos.Dictionary,
	parent *PDNonTerminalField) *PDNonTerminalField {
	f := &PDNonTerminalField{}
	f.initFieldOf(f, acroForm, field, parent)
	return f
}

// FieldFlags returns the /Ff flags of the field.
func (f *PDNonTerminalField) FieldFlags() int {
	// There is no need to look up the parent hierarchy within a non terminal field
	return f.FieldDictionary().GetIntDefault(cos.Ff, 0)
}

// Children returns the fields below this one.
func (f *PDNonTerminalField) Children() []PDField {
	// TODO: why not return a COSArrayList like in PDPage.getAnnotations() ?
	children := []PDField{}
	kids := f.FieldDictionary().GetCOSArray(cos.Kids)
	if kids == nil {
		return children
	}
	for i := 0; i < kids.Size(); i++ {
		kid, isDictionary := kids.GetObject(i).(*cos.Dictionary)
		if !isDictionary {
			continue
		}
		if kid == f.FieldDictionary() {
			warnSameObject()
			continue
		}
		if field := fieldFromDictionary(f.AcroForm(), kid, f); field != nil {
			children = append(children, field)
		}
	}
	return children
}

// SetChildren sets the fields below this one.
func (f *PDNonTerminalField) SetChildren(children []PDField) {
	kidsArray := cos.NewArray()
	for _, child := range children {
		kidsArray.Add(child.COSObject())
	}
	f.FieldDictionary().SetItem(cos.Kids, kidsArray)
}

// FieldType returns the /FT of the field.
func (f *PDNonTerminalField) FieldType() string {
	return f.FieldDictionary().GetNameAsString(cos.FT, "")
}

// Value returns the /V of the field, whatever type it holds.
func (f *PDNonTerminalField) Value() cos.Base {
	return f.FieldDictionary().GetDictionaryObject(cos.V)
}

// ValueAsString returns the /V of the field as a string.
func (f *PDNonTerminalField) ValueAsString() string {
	if fieldValue := f.FieldDictionary().GetDictionaryObject(cos.V); fieldValue != nil {
		return fmt.Sprintf("%v", fieldValue)
	}
	return ""
}

// SetValueObject sets the /V of the field to the given object.
//
// Java names this setValue(COSBase), overloading the setValue(String) the
// interface asks for.
func (f *PDNonTerminalField) SetValueObject(object cos.Base) {
	f.FieldDictionary().SetItem(cos.V, object)
	// todo: propagate change event to children?
	// todo: construct appearances of children?
}

// SetValue sets the /V of the field from a string.
func (f *PDNonTerminalField) SetValue(value string) error {
	f.FieldDictionary().SetString(cos.V, value)
	// todo: propagate change event to children?
	// todo: construct appearances of children?
	return nil
}

// DefaultValue returns the /DV of the field, whatever type it holds.
func (f *PDNonTerminalField) DefaultValue() cos.Base {
	return f.FieldDictionary().GetDictionaryObject(cos.DV)
}

// SetDefaultValue sets the /DV of the field.
func (f *PDNonTerminalField) SetDefaultValue(value cos.Base) {
	f.FieldDictionary().SetItem(cos.DV, value)
}

// Widgets returns no widgets: a non terminal field has none of its own.
func (f *PDNonTerminalField) Widgets() []*annotation.PDAnnotationWidget {
	return []*annotation.PDAnnotationWidget{}
}
