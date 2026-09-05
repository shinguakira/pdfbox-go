package form

import (
	"log/slog"
	"strconv"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// PDSignatureField is a field that holds a digital signature.
//
// Port of PDSignatureField.
//
// The value accessors -- getSignature, getValue, setValue(PDSignature),
// getDefaultValue, setDefaultValue and the seed value pair -- are not here:
// they name PDSignature and PDSeedValue, and
// pdmodel/interactive/digitalsignature is not ported yet. They land with it.
// See migration/STATUS.md.
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

// SetValue panics: a signature field holds a signature dictionary rather than a
// string, which is the UnsupportedOperationException Java throws.
func (f *PDSignatureField) SetValue(value string) error {
	panic("Signature fields don't support setting the value as String " +
		"- use setValue(PDSignature value) instead")
}

// ValueAsString returns the value of the field as a string, which is empty
// until the signature accessors land.
func (f *PDSignatureField) ValueAsString() string {
	if value := f.FieldDictionary().GetCOSDictionary(cos.V); value != nil {
		return value.String()
	}
	return ""
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
