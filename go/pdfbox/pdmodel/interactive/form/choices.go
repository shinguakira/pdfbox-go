package form

import (
	"slices"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// PDListBox is a choice field that shows its options in a list.
//
// Port of PDListBox, which Java declares final.
type PDListBox struct {
	PDChoice
}

var _ PDField = (*PDListBox)(nil)

// NewPDListBox creates a list box in the given form.
func NewPDListBox(acroForm *PDAcroForm) *PDListBox {
	f := &PDListBox{}
	f.initChoice(f, acroForm)
	return f
}

// NewPDListBoxOf creates one over the given dictionary. Java declares the
// constructor package-private.
func NewPDListBoxOf(acroForm *PDAcroForm, field *cos.Dictionary,
	parent *PDNonTerminalField) *PDListBox {
	f := &PDListBox{}
	f.initChoiceOf(f, acroForm, field, parent)
	return f
}

// TopIndex returns the /TI index of the first option shown.
func (f *PDListBox) TopIndex() int {
	return f.FieldDictionary().GetIntDefault(cos.TI, 0)
}

// SetTopIndex sets the /TI index of the first option shown, and removes it
// where the value is nil.
func (f *PDListBox) SetTopIndex(topIndex *int) {
	if topIndex != nil {
		f.FieldDictionary().SetInt(cos.TI, *topIndex)
		return
	}
	f.FieldDictionary().RemoveItem(cos.TI)
}

// constructAppearances draws the value of the field.
func (f *PDListBox) constructAppearances() error {
	apHelper, err := newAppearanceGeneratorHelper(&f.PDVariableText)
	if err != nil {
		return err
	}
	return apHelper.setAppearanceValue("")
}

// flagEdit is the bit of the /Ff flags that says a combo box may be typed into.
// Java declares it private.
const flagEdit = 1 << 18

// PDComboBox is a choice field that shows one option and drops down the rest.
//
// Port of PDComboBox, which Java declares final.
type PDComboBox struct {
	PDChoice
}

var _ PDField = (*PDComboBox)(nil)

// NewPDComboBox creates a combo box in the given form.
func NewPDComboBox(acroForm *PDAcroForm) *PDComboBox {
	f := &PDComboBox{}
	f.initChoice(f, acroForm)
	f.SetCombo(true)
	return f
}

// NewPDComboBoxOf creates one over the given dictionary. Java declares the
// constructor package-private.
func NewPDComboBoxOf(acroForm *PDAcroForm, field *cos.Dictionary,
	parent *PDNonTerminalField) *PDComboBox {
	f := &PDComboBox{}
	f.initChoiceOf(f, acroForm, field, parent)
	return f
}

// IsEdit reports whether the field may be typed into.
func (f *PDComboBox) IsEdit() bool { return f.FieldDictionary().GetFlag(cos.Ff, flagEdit) }

// SetEdit sets whether the field may be typed into.
func (f *PDComboBox) SetEdit(edit bool) { f.FieldDictionary().SetFlag(cos.Ff, flagEdit, edit) }

// constructAppearances draws the value of the field, through its display value
// where the options carry one.
func (f *PDComboBox) constructAppearances() error {
	apHelper, err := newAppearanceGeneratorHelper(&f.PDVariableText)
	if err != nil {
		return err
	}
	values := f.Value()
	if len(values) == 0 {
		return apHelper.setAppearanceValue("")
	}
	if f.HasSeparateExportAndDisplayValues() {
		displayValues := f.OptionsDisplayValues()
		index := slices.Index(f.Options(), values[0])
		if index != -1 && index < len(displayValues) {
			return apHelper.setAppearanceValue(displayValues[index])
		}
	}
	return apHelper.setAppearanceValue(values[0])
}
