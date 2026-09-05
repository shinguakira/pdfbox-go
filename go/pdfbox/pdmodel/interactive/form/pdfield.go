package form

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/action"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
)

// The bits of the /Ff flags every field shares. Java declares them private.
const (
	flagReadOnly = 1
	flagRequired = 1 << 1
	flagNoExport = 1 << 2
)

// PDField is a field of an interactive form.
//
// Java's PDField is an abstract class; the port splits it into this interface
// for the contract and the embedded struct below for the state.
//
// importFDF and exportFDF are not here: they name FDFField, and pdmodel/fdf is
// not ported yet. They land with it. See migration/STATUS.md.
type PDField interface {
	common.COSObjectable

	// FieldDictionary returns the field dictionary, which getCOSObject narrows
	// to in Java.
	FieldDictionary() *cos.Dictionary

	// FieldType returns the /FT of the field.
	FieldType() string

	// ValueAsString returns the value of the field as a string.
	ValueAsString() string

	// SetValue sets the value of the field from a string.
	SetValue(value string) error

	// Widgets returns the widget annotations of the field.
	Widgets() []*annotation.PDAnnotationWidget

	// FieldFlags returns the /Ff flags of the field.
	FieldFlags() int

	// PartialName returns the /T of the field.
	PartialName() string

	// FullyQualifiedName returns the partial names of the field and its
	// parents, joined with full stops.
	FullyQualifiedName() string

	// Parent returns the field this one is a kid of, or nil.
	Parent() *PDNonTerminalField

	// AcroForm returns the form the field belongs to.
	AcroForm() *PDAcroForm

	// inheritableAttribute is the protected getInheritableAttribute, which the
	// fields read their inherited entries through.
	inheritableAttribute(key *cos.Name) cos.Base
}

// PDFieldBase carries the state and the concrete methods every field shares.
//
// Port of the non-abstract half of PDField.
type PDFieldBase struct {
	self       PDField
	acroForm   *PDAcroForm
	parent     *PDNonTerminalField
	dictionary *cos.Dictionary
}

// initField is the package-private PDField(PDAcroForm) constructor.
func (f *PDFieldBase) initField(self PDField, acroForm *PDAcroForm) {
	f.initFieldOf(self, acroForm, cos.NewDictionary(), nil)
}

// initFieldOf is the package-private PDField(PDAcroForm, COSDictionary,
// PDNonTerminalField) constructor.
func (f *PDFieldBase) initFieldOf(self PDField, acroForm *PDAcroForm, field *cos.Dictionary,
	parent *PDNonTerminalField) {
	f.self = self
	f.acroForm = acroForm
	f.dictionary = field
	f.parent = parent
}

// fieldFromDictionary builds the field the given dictionary describes.
//
// Port of the static PDField.fromDictionary.
func fieldFromDictionary(form *PDAcroForm, field *cos.Dictionary,
	parent *PDNonTerminalField) PDField {
	return CreateField(form, field, parent)
}

// inheritableAttribute returns the entry of the field, of its nearest parent
// that has it, or of the form. Java declares it protected.
func (f *PDFieldBase) inheritableAttribute(key *cos.Name) cos.Base {
	if f.dictionary.ContainsKey(key) {
		return f.dictionary.GetDictionaryObject(key)
	}
	if f.parent != nil {
		return f.parent.inheritableAttribute(key)
	}
	return f.acroForm.Dictionary().GetDictionaryObject(key)
}

// SetReadOnly sets whether the field may be changed.
func (f *PDFieldBase) SetReadOnly(readonly bool) {
	f.dictionary.SetFlag(cos.Ff, flagReadOnly, readonly)
}

// IsReadOnly reports whether the field may be changed.
func (f *PDFieldBase) IsReadOnly() bool {
	return f.dictionary.GetFlag(cos.Ff, flagReadOnly)
}

// SetRequired sets whether the field must be filled in.
func (f *PDFieldBase) SetRequired(required bool) {
	f.dictionary.SetFlag(cos.Ff, flagRequired, required)
}

// IsRequired reports whether the field must be filled in.
func (f *PDFieldBase) IsRequired() bool {
	return f.dictionary.GetFlag(cos.Ff, flagRequired)
}

// SetNoExport sets whether the field is left out when the form is submitted.
func (f *PDFieldBase) SetNoExport(noExport bool) {
	f.dictionary.SetFlag(cos.Ff, flagNoExport, noExport)
}

// IsNoExport reports whether the field is left out when the form is submitted.
func (f *PDFieldBase) IsNoExport() bool {
	return f.dictionary.GetFlag(cos.Ff, flagNoExport)
}

// SetFieldFlags sets the /Ff flags of the field.
func (f *PDFieldBase) SetFieldFlags(flags int) {
	f.dictionary.SetInt(cos.Ff, flags)
}

// Actions returns the /AA additional actions of the field, or nil.
func (f *PDFieldBase) Actions() *action.PDFormFieldAdditionalActions {
	if aa := f.dictionary.GetCOSDictionary(cos.AA); aa != nil {
		return action.NewPDFormFieldAdditionalActionsOf(aa)
	}
	return nil
}

// Parent returns the field this one is a kid of, or nil.
func (f *PDFieldBase) Parent() *PDNonTerminalField { return f.parent }

// findKid walks down the kids for the field the given path names. Java declares
// it package-private.
//
// Java casts each kid to COSDictionary without a check; the port asserts the
// same way.
func (f *PDFieldBase) findKid(name []string, nameIndex int) PDField {
	var retval PDField
	kids := f.dictionary.GetCOSArray(cos.Kids)
	if kids == nil {
		return nil
	}
	for i := 0; retval == nil && i < kids.Size(); i++ {
		kidDictionary := kids.GetObject(i).(*cos.Dictionary)
		if name[nameIndex] == kidDictionary.GetString(cos.T, "") {
			nonTerminal, _ := f.self.(*PDNonTerminalField)
			retval = fieldFromDictionary(f.acroForm, kidDictionary, nonTerminal)
			if retval != nil && len(name) > nameIndex+1 {
				retval = retval.(interface {
					findKid(name []string, nameIndex int) PDField
				}).findKid(name, nameIndex+1)
			}
		}
	}
	return retval
}

// AcroForm returns the form the field belongs to.
func (f *PDFieldBase) AcroForm() *PDAcroForm { return f.acroForm }

// COSObject returns the field dictionary.
func (f *PDFieldBase) COSObject() cos.Base { return f.dictionary }

// FieldDictionary returns the field dictionary, typed.
func (f *PDFieldBase) FieldDictionary() *cos.Dictionary { return f.dictionary }

// PartialName returns the /T of the field.
func (f *PDFieldBase) PartialName() string { return f.dictionary.GetString(cos.T, "") }

// SetPartialName sets the /T of the field.
//
// Java throws IllegalArgumentException for a name holding a full stop, which is
// unchecked, so the port panics.
func (f *PDFieldBase) SetPartialName(name string) {
	if strings.Contains(name, ".") {
		panic("A field partial name shall not contain a period character: " + name)
	}
	f.dictionary.SetString(cos.T, name)
}

// FullyQualifiedName returns the partial names of the field and its parents,
// joined with full stops.
func (f *PDFieldBase) FullyQualifiedName() string {
	finalName := f.PartialName()
	parentName := ""
	if f.parent != nil {
		parentName = f.parent.FullyQualifiedName()
	}
	if parentName != "" {
		if finalName != "" {
			finalName = parentName + "." + finalName
		} else {
			finalName = parentName
		}
	}
	return finalName
}

// AlternateFieldName returns the /TU of the field, which a reader shows the
// user.
func (f *PDFieldBase) AlternateFieldName() string { return f.dictionary.GetString(cos.TU, "") }

// SetAlternateFieldName sets the /TU of the field.
func (f *PDFieldBase) SetAlternateFieldName(alternateFieldName string) {
	f.dictionary.SetString(cos.TU, alternateFieldName)
}

// MappingName returns the /TM of the field, which an export uses.
func (f *PDFieldBase) MappingName() string { return f.dictionary.GetString(cos.TM, "") }

// SetMappingName sets the /TM of the field.
func (f *PDFieldBase) SetMappingName(mappingName string) {
	f.dictionary.SetString(cos.TM, mappingName)
}

// String renders the field the way Java writes it.
func (f *PDFieldBase) String() string {
	return fmt.Sprintf("%s{type: %T value: %v}", f.FullyQualifiedName(), f.self,
		f.inheritableAttribute(cos.V))
}

// Equals reports whether the two fields wrap the same dictionary, which is the
// equals Java declares.
func (f *PDFieldBase) Equals(o PDField) bool {
	if o == nil {
		return false
	}
	return o.FieldDictionary() == f.dictionary
}

// warnSameObject is the warning PDNonTerminalField logs for a kid that is its
// own parent.
func warnSameObject() {
	slog.Warn("form: child field is same object as parent")
}
