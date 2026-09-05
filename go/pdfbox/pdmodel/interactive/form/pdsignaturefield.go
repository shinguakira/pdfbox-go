package form

import (
	"fmt"
	"log/slog"
	"reflect"
	"strconv"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/digitalsignature"
)

// PDSignatureField is a field that holds a digital signature.
//
// Port of PDSignatureField.
type PDSignatureField struct {
	PDTerminalField
}

var _ PDField = (*PDSignatureField)(nil)

// NewPDSignatureField creates a signature field in the given form, with a name
// no other field has.
func NewPDSignatureField(acroForm *PDAcroForm) *PDSignatureField {
	f := &PDSignatureField{}
	f.initTerminalField(f, acroForm)
	f.FieldDictionary().SetItem(cos.FT, cos.Sig)
	firstWidget := f.Widgets()[0]
	firstWidget.SetLocked(true)
	firstWidget.SetPrinted(true)
	f.SetPartialName(f.generatePartialName())
	return f
}

// NewPDSignatureFieldOf creates one over the given dictionary. Java declares
// the constructor package-private.
func NewPDSignatureFieldOf(acroForm *PDAcroForm, field *cos.Dictionary,
	parent *PDNonTerminalField) *PDSignatureField {
	f := &PDSignatureField{}
	f.initTerminalFieldOf(f, acroForm, field, parent)
	return f
}

// generatePartialName returns a name no field of the form has. Java declares it
// private.
func (f *PDSignatureField) generatePartialName() string {
	const fieldName = "Signature"
	nameSet := map[string]bool{}
	for field := range f.AcroForm().FieldTree().All() {
		nameSet[field.PartialName()] = true
	}
	i := 1
	for nameSet[fieldName+strconv.Itoa(i)] {
		i++
	}
	return fieldName + strconv.Itoa(i)
}

// Signature returns the signature the field holds, or nil where it holds none.
func (f *PDSignatureField) Signature() *digitalsignature.PDSignature { return f.Value() }

// SetSignatureValue sets the signature the field holds.
//
// Java names it setValue(PDSignature), overloading setValue(String); Go has no
// overloading, and PDField already asks for the second.
func (f *PDSignatureField) SetSignatureValue(value *digitalsignature.PDSignature) error {
	f.FieldDictionary().SetItem(cos.V, common.COSObjectOrNil(value))
	return f.applyChange()
}

// SetValue panics: a signature field holds a signature dictionary rather than a
// string, which is the UnsupportedOperationException Java throws.
func (f *PDSignatureField) SetValue(value string) error {
	panic("Signature fields don.t support setting the value as String " +
		"- use setValue(PDSignature value) instead")
}

// SetDefaultValue sets the /DV of the field.
func (f *PDSignatureField) SetDefaultValue(value *digitalsignature.PDSignature) {
	f.FieldDictionary().SetItem(cos.DV, common.COSObjectOrNil(value))
}

// Value returns the signature the field holds, or nil where it holds none.
func (f *PDSignatureField) Value() *digitalsignature.PDSignature {
	if value := f.FieldDictionary().GetCOSDictionary(cos.V); value != nil {
		return digitalsignature.NewPDSignatureOf(value)
	}
	return nil
}

// DefaultValue returns the /DV of the field, or nil where it has none.
func (f *PDSignatureField) DefaultValue() *digitalsignature.PDSignature {
	if value := f.FieldDictionary().GetCOSDictionary(cos.DV); value != nil {
		return digitalsignature.NewPDSignatureOf(value)
	}
	return nil
}

// ValueAsString returns the value of the field as a string.
//
// Java answers signature.toString(), and PDSignature overrides no toString, so
// what comes back is Object.toString: the class name and the identity hash,
// which says nothing about the signature. Go has no such default, so the port
// writes the same shape out of the type and the address. Neither string is
// useful; that is the Java.
func (f *PDSignatureField) ValueAsString() string {
	signature := f.Value()
	if signature != nil {
		return fmt.Sprintf("%T@%x", signature, reflect.ValueOf(signature).Pointer())
	}
	return ""
}

// SeedValue returns the seed value dictionary of the field, or nil where it has
// none.
func (f *PDSignatureField) SeedValue() *digitalsignature.PDSeedValue {
	if dict := f.FieldDictionary().GetCOSDictionary(cos.SV); dict != nil {
		return digitalsignature.NewPDSeedValueOf(dict)
	}
	return nil
}

// SetSeedValue sets the seed value dictionary of the field, and does nothing
// for a nil one.
func (f *PDSignatureField) SetSeedValue(sv *digitalsignature.PDSeedValue) {
	if sv != nil {
		f.FieldDictionary().SetItem(cos.SV, sv.COSObject())
	}
}

// constructAppearances warns rather than drawing: Java has no appearance
// generation for a signature.
func (f *PDSignatureField) constructAppearances() error {
	widgets := f.Widgets()
	if len(widgets) == 0 {
		return nil
	}
	widget := widgets[0]
	rectangle := widget.Rectangle()

	// check if the signature is visible
	if rectangle == nil ||
		rectangle.Height() == 0 && rectangle.Width() == 0 ||
		widget.IsNoView() || widget.IsHidden() {
		return nil
	}

	// TODO: implement appearance generation for signatures (PDFBOX-3524)
	slog.Warn("form: appearance generation for signature fields not implemented here. " +
		"You need to generate/update that manually, see the " +
		"CreateVisibleSignature*.java files in the examples subproject " +
		"of the source code download")
	return nil
}
