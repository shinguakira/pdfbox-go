package form

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// PDCheckBox is a check box.
//
// Port of PDCheckBox, which Java declares final.
type PDCheckBox struct {
	PDButton
}

var _ PDField = (*PDCheckBox)(nil)

// NewPDCheckBox creates a check box in the given form.
func NewPDCheckBox(acroForm *PDAcroForm) *PDCheckBox {
	f := &PDCheckBox{}
	f.initButton(f, acroForm)
	return f
}

// NewPDCheckBoxOf creates one over the given dictionary. Java declares the
// constructor package-private.
func NewPDCheckBoxOf(acroForm *PDAcroForm, field *cos.Dictionary,
	parent *PDNonTerminalField) *PDCheckBox {
	f := &PDCheckBox{}
	f.initButtonOf(f, acroForm, field, parent)
	return f
}

// IsChecked reports whether the box is ticked.
func (f *PDCheckBox) IsChecked() bool { return f.Value() == f.OnValue() }

// Check ticks the box.
func (f *PDCheckBox) Check() error { return f.SetValue(f.OnValue()) }

// UnCheck clears the box.
func (f *PDCheckBox) UnCheck() error { return f.SetValue(cos.Off.Name()) }

// OnValue returns the value that ticks the box.
//
// Java reads the first widget without checking that there is one, so a field
// with none throws; the port panics on the same index.
func (f *PDCheckBox) OnValue() string {
	widget := f.Widgets()[0]
	return onValueForWidget(widget)
}

// The bits of the /Ff flags a radio button carries beyond a button. Java
// declares it private.
const flagNoToggleToOff = 1 << 14

// PDRadioButton is one radio button of a group.
//
// Port of PDRadioButton, which Java declares final.
type PDRadioButton struct {
	PDButton
}

var _ PDField = (*PDRadioButton)(nil)

// NewPDRadioButton creates a radio button in the given form.
func NewPDRadioButton(acroForm *PDAcroForm) *PDRadioButton {
	f := &PDRadioButton{}
	f.initButton(f, acroForm)
	f.FieldDictionary().SetFlag(cos.Ff, flagRadio, true)
	return f
}

// NewPDRadioButtonOf creates one over the given dictionary. Java declares the
// constructor package-private.
func NewPDRadioButtonOf(acroForm *PDAcroForm, field *cos.Dictionary,
	parent *PDNonTerminalField) *PDRadioButton {
	f := &PDRadioButton{}
	f.initButtonOf(f, acroForm, field, parent)
	return f
}

// SetRadiosInUnison sets whether buttons with the same on value turn on
// together.
func (f *PDRadioButton) SetRadiosInUnison(radiosInUnison bool) {
	f.FieldDictionary().SetFlag(cos.Ff, flagRadiosInUnison, radiosInUnison)
}

// IsRadiosInUnison reports whether buttons with the same on value turn on
// together.
func (f *PDRadioButton) IsRadiosInUnison() bool {
	return f.FieldDictionary().GetFlag(cos.Ff, flagRadiosInUnison)
}

// SetNoToggleToOff sets whether clicking the button that is on turns it off.
func (f *PDRadioButton) SetNoToggleToOff(noToggleToOff bool) {
	f.FieldDictionary().SetFlag(cos.Ff, flagNoToggleToOff, noToggleToOff)
}

// IsNoToggleToOff reports whether clicking the button that is on turns it off.
func (f *PDRadioButton) IsNoToggleToOff() bool {
	return f.FieldDictionary().GetFlag(cos.Ff, flagNoToggleToOff)
}

// SelectedIndex returns the index of the widget that is on, or -1.
func (f *PDRadioButton) SelectedIndex() int {
	idx := 0
	for _, widget := range f.Widgets() {
		if widget.AppearanceState() != cos.Off {
			return idx
		}
		idx++
	}
	return -1
}

// SelectedExportValues returns the export values of the buttons that are on.
func (f *PDRadioButton) SelectedExportValues() []string {
	exportValues := f.ExportValues()
	selectedExportValues := []string{}
	if len(exportValues) == 0 {
		selectedExportValues = append(selectedExportValues, f.Value())
		return selectedExportValues
	}
	fieldValue := f.Value()
	for idx, onValue := range f.OnValues() {
		if onValue == fieldValue && idx < len(exportValues) {
			selectedExportValues = append(selectedExportValues, exportValues[idx])
		}
	}
	return selectedExportValues
}

// PDPushButton is a button that runs an action rather than holding a value.
//
// Port of PDPushButton.
type PDPushButton struct {
	PDButton
}

var _ PDField = (*PDPushButton)(nil)

// NewPDPushButton creates a push button in the given form.
func NewPDPushButton(acroForm *PDAcroForm) *PDPushButton {
	f := &PDPushButton{}
	f.initButton(f, acroForm)
	f.FieldDictionary().SetFlag(cos.Ff, flagPushButton, true)
	return f
}

// NewPDPushButtonOf creates one over the given dictionary. Java declares the
// constructor package-private.
func NewPDPushButtonOf(acroForm *PDAcroForm, field *cos.Dictionary,
	parent *PDNonTerminalField) *PDPushButton {
	f := &PDPushButton{}
	f.initButtonOf(f, acroForm, field, parent)
	return f
}

// ExportValues returns no export values: a push button has none.
func (f *PDPushButton) ExportValues() []string { return []string{} }

// SetExportValues panics unless the list is empty, which is the
// IllegalArgumentException Java throws.
func (f *PDPushButton) SetExportValues(values []string) {
	if len(values) != 0 {
		panic("A PDPushButton shall not use the Opt entry in the field dictionary")
	}
}

// Value returns the empty string: a push button holds no value.
func (f *PDPushButton) Value() string { return "" }

// DefaultValue returns the empty string: a push button holds no value.
func (f *PDPushButton) DefaultValue() string { return "" }

// ValueAsString returns the value of the button as a string.
func (f *PDPushButton) ValueAsString() string { return f.Value() }

// OnValues returns no values: a push button has none.
func (f *PDPushButton) OnValues() []string { return []string{} }

// constructAppearances does nothing.
func (f *PDPushButton) constructAppearances() error {
	// TODO: add appearance handler to generate/update appearance
	return nil
}
